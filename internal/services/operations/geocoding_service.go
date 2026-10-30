package operations

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	geocodeCacheKeyPrefix    = "baidu_map:geocode:"
	geocodingDefaultCacheTTL = 30 * time.Minute
	geocodingMaxConcurrency  = 5
	geocodingAPITimeout      = 10 * time.Second
	baiduGeocodingAPIURL     = "https://api.map.baidu.com/geocoding/v3/"
)

// BaiduGeocodingResponse 百度地图地理编码响应
type BaiduGeocodingResponse struct {
	Status  int              `json:"status"`
	Message string           `json:"message,omitempty"`
	Result  *GeocodingResult `json:"result,omitempty"`
}

// GeocodingResult 地理编码结果
type GeocodingResult struct {
	Location struct {
		Lng float64 `json:"lng"`
		Lat float64 `json:"lat"`
	} `json:"location"`
	FormattedAddress string `json:"formatted_address"`
	AddressComponent struct {
		Province string `json:"province"`
		City     string `json:"city"`
		District string `json:"district"`
		Street   string `json:"street"`
	} `json:"addressComponent"`
	Precise int    `json:"precise"`
	Level   string `json:"level"`
}

// Coordinate 坐标
type Coordinate struct {
	Lng float64
	Lat float64
}

type cacheEntry struct {
	result    *GeocodingResult
	expiresAt time.Time
}

// GeocodingService 地理编码服务
type GeocodingService struct {
	apiKey      string
	httpClient  *http.Client
	cache       sync.Map
	redisCache  cache.Cache
	stats       *CacheStats
	rateLimiter *RateLimiter
}

// NewGeocodingService 创建地理编码服务
func NewGeocodingService(apiKey string) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: geocodingAPITimeout,
		},
		stats:       &CacheStats{},
		rateLimiter: BaiduAPIRateLimiter,
	}
}

// NewGeocodingServiceWithCache 创建带Redis缓存的地理编码服务
func NewGeocodingServiceWithCache(apiKey string, redisCache cache.Cache) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: geocodingAPITimeout,
		},
		redisCache:  redisCache,
		stats:       &CacheStats{},
		rateLimiter: BaiduAPIRateLimiter,
	}
}

// Geocode 地址解析：将详细地址转换为经纬度坐标
func (s *GeocodingService) Geocode(ctx context.Context, address string) (lng, lat float64, err error) {
	if address == "" {
		return 0, 0, fmt.Errorf("地址不能为空")
	}

	cacheKey := buildCacheKey(address)

	// 检查缓存（内存 -> Redis）
	if result, found := s.getFromCache(ctx, cacheKey); found {
		return result.Location.Lng, result.Location.Lat, nil
	}

	// 检查限流器
	if !s.rateLimiter.Allow() {
		s.stats.RecordMiss()
		return 0, 0, fmt.Errorf("API调用频率超限，请稍后重试")
	}

	// 调用百度API
	result, err := s.callGeocodingAPI(ctx, address)
	if err != nil {
		s.stats.RecordMiss()
		return 0, 0, err
	}

	s.stats.RecordAPICall()

	// 缓存结果
	s.setToCache(ctx, cacheKey, result)
	logger.Debugf("地址解析成功: %s -> (%f, %f)", address, result.Location.Lng, result.Location.Lat)

	return result.Location.Lng, result.Location.Lat, nil
}

// getFromCache 从缓存获取结果（内存 -> Redis）
func (s *GeocodingService) getFromCache(ctx context.Context, cacheKey string) (*GeocodingResult, bool) {
	// 检查内存缓存
	if result, ok := s.getFromMemoryCache(cacheKey); ok {
		s.stats.RecordMemoryHit()
		return result, true
	}

	// 检查Redis缓存
	if s.redisCache != nil {
		if result, err := s.getFromRedisCache(ctx, cacheKey); err == nil && result != nil {
			s.stats.RecordRedisHit()
			s.setToMemoryCache(cacheKey, result)
			return result, true
		}
	}

	return nil, false
}

// setToCache 设置缓存（内存 + Redis）
func (s *GeocodingService) setToCache(ctx context.Context, cacheKey string, result *GeocodingResult) {
	s.setToMemoryCache(cacheKey, result)
	if s.redisCache != nil {
		s.setToRedisCache(ctx, cacheKey, result)
	}
}

// BatchGeocode 批量地址解析（带并发控制）
func (s *GeocodingService) BatchGeocode(ctx context.Context, addresses []string) (map[string]Coordinate, error) {
	results := make(map[string]Coordinate)
	var mu sync.Mutex
	sem := make(chan struct{}, geocodingMaxConcurrency)
	var wg sync.WaitGroup

	for _, addr := range addresses {
		if addr == "" {
			continue
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(address string) {
			defer wg.Done()
			defer func() { <-sem }()

			lng, lat, err := s.Geocode(ctx, address)
			if err != nil {
				logger.Warnf("地址解析失败: %s, 错误: %v", address, err)
				return
			}

			mu.Lock()
			results[address] = Coordinate{Lng: lng, Lat: lat}
			mu.Unlock()
		}(addr)
	}

	wg.Wait()

	successCount := len(results)
	failCount := len(addresses) - successCount
	logger.Infof("批量地址解析完成: 成功 %d, 失败 %d", successCount, failCount)

	if successCount == 0 {
		return nil, fmt.Errorf("批量地址解析失败")
	}

	return results, nil
}

// buildCacheKey 构建缓存键（使用MD5哈希）
func buildCacheKey(address string) string {
	hash := md5.Sum([]byte(address))
	return geocodeCacheKeyPrefix + hex.EncodeToString(hash[:])
}

// getFromMemoryCache 从内存缓存获取
func (s *GeocodingService) getFromMemoryCache(cacheKey string) (*GeocodingResult, bool) {
	val, ok := s.cache.Load(cacheKey)
	if !ok {
		return nil, false
	}

	entry, ok := val.(*cacheEntry)
	if !ok || time.Now().After(entry.expiresAt) {
		s.cache.Delete(cacheKey)
		return nil, false
	}

	return entry.result, true
}

// setToMemoryCache 设置内存缓存
func (s *GeocodingService) setToMemoryCache(cacheKey string, result *GeocodingResult) {
	entry := &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(geocodingDefaultCacheTTL),
	}
	s.cache.Store(cacheKey, entry)

	time.AfterFunc(geocodingDefaultCacheTTL, func() {
		s.cache.Delete(cacheKey)
	})
}

// getFromRedisCache 从Redis缓存获取
func (s *GeocodingService) getFromRedisCache(ctx context.Context, cacheKey string) (*GeocodingResult, error) {
	data, err := s.redisCache.Get(ctx, cacheKey)
	if err != nil || data == "" {
		return nil, err
	}

	var result GeocodingResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// setToRedisCache 设置Redis缓存
func (s *GeocodingService) setToRedisCache(ctx context.Context, cacheKey string, result *GeocodingResult) {
	data, err := json.Marshal(result)
	if err != nil {
		logger.Warnf("序列化地理编码结果失败: %v", err)
		return
	}

	if err := s.redisCache.Set(ctx, cacheKey, string(data), geocodingDefaultCacheTTL); err != nil {
		logger.Warnf("保存Redis缓存失败: %v", err)
	}
}

// GetStats 获取缓存统计
func (s *GeocodingService) GetStats() CacheStatsData {
	return s.stats.GetStats()
}

// ResetStats 重置缓存统计
func (s *GeocodingService) ResetStats() {
	s.stats.Reset()
}

// InvalidateCache 清除指定地址的缓存
func (s *GeocodingService) InvalidateCache(ctx context.Context, address string) {
	cacheKey := buildCacheKey(address)
	s.cache.Delete(cacheKey)
	if s.redisCache != nil {
		_ = s.redisCache.Delete(ctx, cacheKey)
	}
}

// ClearAllCache 清除所有缓存
func (s *GeocodingService) ClearAllCache(ctx context.Context) {
	s.cache.Range(func(key, _ interface{}) bool {
		s.cache.Delete(key)
		return true
	})

	if s.redisCache != nil {
		pattern := geocodeCacheKeyPrefix + "*"
		keys, _ := s.redisCache.Keys(ctx, pattern)
		if len(keys) > 0 {
			_ = s.redisCache.MDelete(ctx, keys...)
		}
	}
}

// callGeocodingAPI 调用百度地图地理编码API
func (s *GeocodingService) callGeocodingAPI(ctx context.Context, address string) (*GeocodingResult, error) {
	params := url.Values{}
	params.Set("address", address)
	params.Set("output", "json")
	params.Set("ak", s.apiKey)

	apiURL := fmt.Sprintf("%s?%s", baiduGeocodingAPIURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var baiduResp BaiduGeocodingResponse
	if err := json.Unmarshal(body, &baiduResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// F 簇：百度地图 API 返回码契约，不迁移到 models 常量（见 scripts/check-status-literals.sh 白名单）
	if baiduResp.Status != 0 {
		return nil, fmt.Errorf("地址解析失败: %s", baiduResp.Message)
	}

	if baiduResp.Result == nil {
		return nil, fmt.Errorf("地址解析失败: 未返回结果")
	}

	return baiduResp.Result, nil
}

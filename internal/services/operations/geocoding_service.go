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

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// 常量定义
const (
	geocodeCacheKeyPrefix = "baidu_map:geocode:" // 修改前缀以区分Redis缓存
	defaultCacheTTL       = 30 * time.Minute
	maxConcurrency        = 5
	apiTimeout            = 10 * time.Second
	baiduGeocodingAPIURL  = "https://api.map.baidu.com/geocoding/v3/"
)

// BaiduGeocodingResponse 百度地图地理编码响应
type BaiduGeocodingResponse struct {
	Status  int               `json:"status"`
	Message string            `json:"message,omitempty"`
	Result  *GeocodingResult  `json:"result,omitempty"`
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

// cacheEntry 缓存条目
type cacheEntry struct {
	result    *GeocodingResult
	expiresAt time.Time
}

// GeocodingService 地理编码服务
type GeocodingService struct {
	apiKey      string
	httpClient  *http.Client
	cache       sync.Map // 内存缓存（已存在）
	redisCache  cache.Cache  // 新增：Redis缓存
	mu          sync.RWMutex
	stats       *CacheStats     // 新增：缓存统计
	rateLimiter *RateLimiter    // 新增：限流器
}

// NewGeocodingService 创建地理编码服务（向后兼容）
func NewGeocodingService(apiKey string) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		stats:       &CacheStats{},
		rateLimiter: BaiduAPIRateLimiter,
	}
}

// NewGeocodingServiceWithCache 创建带Redis缓存的地理编码服务（推荐）
func NewGeocodingServiceWithCache(apiKey string, redisCache cache.Cache) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		redisCache:  redisCache,
		stats:       &CacheStats{},
		rateLimiter: BaiduAPIRateLimiter,
	}
}

// Geocode 地址解析：将详细地址转换为经纬度坐标（增强版）
func (s *GeocodingService) Geocode(ctx context.Context, address string) (lng, lat float64, err error) {
	if address == "" {
		return 0, 0, fmt.Errorf("地址不能为空")
	}

	cacheKey := buildCacheKey(address)

	// 1. 检查内存缓存
	if result, ok := s.getFromMemoryCache(cacheKey); ok {
		s.stats.RecordMemoryHit()
		return result.Location.Lng, result.Location.Lat, nil
	}

	// 2. 检查Redis缓存
	if s.redisCache != nil {
		if result, err := s.getFromRedisCache(ctx, cacheKey); err == nil && result != nil {
			s.stats.RecordRedisHit()
			// 回填到内存缓存
			s.setToMemoryCache(cacheKey, result)
			return result.Location.Lng, result.Location.Lat, nil
		}
	}

	// 3. 检查限流器
	if !s.rateLimiter.Allow() {
		return 0, 0, fmt.Errorf("API调用频率超限，请稍后重试")
	}

	// 4. 调用百度API
	result, err := s.callGeocodingAPI(ctx, address)
	if err != nil {
		s.stats.RecordMiss()
		return 0, 0, err
	}

	s.stats.RecordAPICall()

	// 5. 缓存结果（双层）
	s.setToMemoryCache(cacheKey, result)
	if s.redisCache != nil {
		s.setToRedisCache(ctx, cacheKey, result)
	}

	logger.Debugf("地址解析成功: %s -> (%f, %f)", address, result.Location.Lng, result.Location.Lat)

	return result.Location.Lng, result.Location.Lat, nil
}

// BatchGeocode 批量地址解析（带并发控制）
func (s *GeocodingService) BatchGeocode(ctx context.Context, addresses []string) (map[string]Coordinate, error) {
	results := make(map[string]Coordinate)
	var mu sync.Mutex
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var successCount, failCount int

	for _, addr := range addresses {
		if addr == "" {
			continue
		}

		// 先获取信号量（避免启动过多 goroutine）
		sem <- struct{}{}

		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			defer func() { <-sem }()

			lng, lat, err := s.Geocode(ctx, address)
			if err != nil {
				logger.Warnf("地址解析失败: %s, 错误: %v", address, err)
				failCount++
				return
			}

			mu.Lock()
			results[address] = Coordinate{Lng: lng, Lat: lat}
			successCount++
			mu.Unlock()
		}(addr)
	}

	wg.Wait()

	logger.Infof("批量地址解析完成: 成功 %d, 失败 %d", successCount, failCount)

	// 如果所有地址都解析失败，返回错误
	if len(results) == 0 {
		return nil, fmt.Errorf("批量地址解析失败")
	}

	return results, nil
}

// buildCacheKey 构建缓存键（使用MD5哈希）
func buildCacheKey(address string) string {
	// 使用MD5哈希避免特殊字符问题
	hash := md5.Sum([]byte(address))
	return geocodeCacheKeyPrefix + hex.EncodeToString(hash[:])
}

// getFromMemoryCache 从内存缓存获取
func (s *GeocodingService) getFromMemoryCache(cacheKey string) (*GeocodingResult, bool) {
	if val, ok := s.cache.Load(cacheKey); ok {
		if entry, ok := val.(*cacheEntry); ok {
			if time.Now().Before(entry.expiresAt) {
				return entry.result, true
			}
			s.cache.Delete(cacheKey)
		}
	}
	return nil, false
}

// setToMemoryCache 设置内存缓存
func (s *GeocodingService) setToMemoryCache(cacheKey string, result *GeocodingResult) {
	entry := &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(defaultCacheTTL),
	}
	s.cache.Store(cacheKey, entry)

	// 自动清理
	time.AfterFunc(defaultCacheTTL, func() {
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

	if err := s.redisCache.Set(ctx, cacheKey, string(data), defaultCacheTTL); err != nil {
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
	s.cache.Range(func(key, value interface{}) bool {
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

// callGeocodingAPI 调用百度地图地理编码 API
func (s *GeocodingService) callGeocodingAPI(ctx context.Context, address string) (*GeocodingResult, error) {
	// 构建请求 URL
	params := url.Values{}
	params.Set("address", address)
	params.Set("output", "json")
	params.Set("ak", s.apiKey)

	apiURL := fmt.Sprintf("%s?%s", baiduGeocodingAPIURL, params.Encode())

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	var baiduResp BaiduGeocodingResponse
	if err := json.Unmarshal(body, &baiduResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查状态
	if baiduResp.Status != 0 {
		return nil, fmt.Errorf("地址解析失败: %s", baiduResp.Message)
	}

	if baiduResp.Result == nil {
		return nil, fmt.Errorf("地址解析失败: 未返回结果")
	}

	return baiduResp.Result, nil
}

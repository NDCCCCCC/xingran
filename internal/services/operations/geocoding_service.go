package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// 常量定义
const (
	geocodeCacheKeyPrefix = "geocode:"
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
	apiKey     string
	httpClient *http.Client
	cache      sync.Map
	mu         sync.RWMutex
}

// NewGeocodingService 创建地理编码服务
func NewGeocodingService(apiKey string) *GeocodingService {
	return &GeocodingService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
	}
}

// Geocode 地址解析：将详细地址转换为经纬度坐标
func (s *GeocodingService) Geocode(ctx context.Context, address string) (lng, lat float64, err error) {
	if address == "" {
		return 0, 0, fmt.Errorf("地址不能为空")
	}

	// 检查缓存
	if result, ok := s.getFromCache(address); ok {
		return result.Location.Lng, result.Location.Lat, nil
	}

	// 调用 API
	result, err := s.callGeocodingAPI(ctx, address)
	if err != nil {
		return 0, 0, err
	}

	// 缓存结果
	s.setCache(address, result)

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

// buildCacheKey 构建缓存键
func buildCacheKey(address string) string {
	return geocodeCacheKeyPrefix + address
}

// getFromCache 从缓存获取结果
func (s *GeocodingService) getFromCache(address string) (*GeocodingResult, bool) {
	cacheKey := buildCacheKey(address)
	if val, ok := s.cache.Load(cacheKey); ok {
		if entry, ok := val.(*cacheEntry); ok {
			// 检查是否过期
			if time.Now().Before(entry.expiresAt) {
				return entry.result, true
			}
			// 过期则删除
			s.cache.Delete(cacheKey)
		}
	}
	return nil, false
}

// setCache 设置缓存
func (s *GeocodingService) setCache(address string, result *GeocodingResult) {
	cacheKey := buildCacheKey(address)
	entry := &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(defaultCacheTTL),
	}
	s.cache.Store(cacheKey, entry)

	// 使用定时器清理过期缓存（避免 goroutine 泄漏）
	time.AfterFunc(defaultCacheTTL, func() {
		s.cache.Delete(cacheKey)
	})
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

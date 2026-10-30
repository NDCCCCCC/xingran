package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/pkg/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	// 默认缓存 TTL
	defaultL1TTL         = 30 * time.Second
	defaultL2TTL         = 5 * time.Minute
	metricsUpdateTick    = 30 * time.Second
	serverInfoUpdateTick = 30 * time.Minute
	serverInfoL1TTL      = 5 * time.Minute
	serverInfoL2TTL      = 30 * time.Minute
	cleanupTick          = 2 * time.Hour
)

// MetricsCacheManager 系统指标缓存管理器
type MetricsCacheManager struct {
	memoryCache sync.Map       // L1缓存：内存缓存
	redisCache  interface{}    // L2缓存：Redis缓存
	hostname    string         // 主机名
	stopChan    chan struct{}  // 停止信号
	wg          sync.WaitGroup // 等待组
	stopOnce    sync.Once      // 保证 Stop 幂等
}

// CacheItem 缓存项
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// MetricsData 系统指标数据
type MetricsData struct {
	CPUUsage    float64   `json:"cpuUsage"`
	MemoryUsage float64   `json:"memoryUsage"`
	DiskUsage   float64   `json:"diskUsage"`
	NetworkRx   uint64    `json:"networkRx"`
	NetworkTx   uint64    `json:"networkTx"`
	ProcessNum  int       `json:"processNum"`
	TotalMemory uint64    `json:"totalMemory"`
	UsedMemory  uint64    `json:"usedMemory"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewMetricsCacheManager 创建指标缓存管理器
func NewMetricsCacheManager(redisCache interface{}) *MetricsCacheManager {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}

	manager := &MetricsCacheManager{
		redisCache: redisCache,
		hostname:   hostname,
		stopChan:   make(chan struct{}),
	}

	manager.startBackgroundTasks()
	return manager
}

// getCacheKey 生成缓存键
func (m *MetricsCacheManager) getCacheKey(metricType string) string {
	return fmt.Sprintf("sys:%s:%s", metricType, m.hostname)
}

// getFromL1 从L1缓存获取
func (m *MetricsCacheManager) getFromL1(key string) (interface{}, bool) {
	item, ok := m.memoryCache.Load(key)
	if !ok {
		return nil, false
	}

	cacheItem := item.(*CacheItem)
	if time.Now().Before(cacheItem.ExpiresAt) {
		return cacheItem.Value, true
	}

	// 过期了，删除
	m.memoryCache.Delete(key)
	return nil, false
}

// setToL1 设置到L1缓存
func (m *MetricsCacheManager) setToL1(key string, value interface{}, ttl time.Duration) {
	m.memoryCache.Store(key, &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	})
}

// getFromL2 从L2缓存获取
func (m *MetricsCacheManager) getFromL2(ctx context.Context, key string) (interface{}, error) {
	if m.redisCache == nil {
		return nil, fmt.Errorf("Redis缓存未初始化")
	}

	// 使用类型断言检查是否有Get方法
	if cacheWithGet, ok := m.redisCache.(interface {
		Get(ctx context.Context, key string) (string, error)
	}); ok {
		data, err := cacheWithGet.Get(ctx, key)
		if err != nil {
			return nil, err
		}

		var result interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			return nil, fmt.Errorf("反序列化失败: %w", err)
		}

		return result, nil
	}

	return nil, fmt.Errorf("Redis缓存不支持Get方法")
}

// setToL2 设置到L2缓存
func (m *MetricsCacheManager) setToL2(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.redisCache == nil {
		return fmt.Errorf("Redis缓存未初始化")
	}

	// 使用类型断言检查是否有Set方法
	if cacheWithSet, ok := m.redisCache.(interface {
		Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	}); ok {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		return cacheWithSet.Set(ctx, key, data, ttl)
	}

	return fmt.Errorf("Redis缓存不支持Set方法")
}

// GetSystemMetrics 获取系统指标（优先从缓存）
func (m *MetricsCacheManager) GetSystemMetrics(ctx context.Context) (*MetricsData, error) {
	key := m.getCacheKey("metrics:current")

	// 1. 尝试从L1缓存获取
	if value, ok := m.getFromL1(key); ok {
		if metrics, ok := value.(*MetricsData); ok {
			return metrics, nil
		}
	}

	// 2. 尝试从L2缓存获取
	value, err := m.getFromL2(ctx, key)
	if err == nil {
		if metrics, ok := value.(*MetricsData); ok {
			m.setToL1(key, metrics, defaultL1TTL)
			return metrics, nil
		}
	}

	// 3. 缓存未命中，获取实时数据
	metrics, err := m.getRealtimeMetrics()
	if err != nil {
		return nil, err
	}

	// 4. 写入缓存
	m.setToL1(key, metrics, defaultL1TTL)
	_ = m.setToL2(ctx, key, metrics, defaultL2TTL)

	return metrics, nil
}

// getRealtimeMetrics 获取实时系统指标
func (m *MetricsCacheManager) getRealtimeMetrics() (*MetricsData, error) {
	systemMetrics, err := system.GetSystemMetrics()
	if err != nil {
		return nil, err
	}

	return &MetricsData{
		CPUUsage:    systemMetrics.CPUUsage,
		MemoryUsage: systemMetrics.MemoryUsage,
		DiskUsage:   systemMetrics.DiskUsage,
		NetworkRx:   systemMetrics.NetworkRx,
		NetworkTx:   systemMetrics.NetworkTx,
		ProcessNum:  systemMetrics.ProcessNum,
		TotalMemory: systemMetrics.TotalMemory,
		UsedMemory:  systemMetrics.UsedMemory,
		Timestamp:   systemMetrics.Timestamp,
	}, nil
}

// GetServerInfo 获取服务器信息（优先从缓存）
func (m *MetricsCacheManager) GetServerInfo(ctx context.Context) (map[string]interface{}, error) {
	key := m.getCacheKey("server:info")

	// 1. 尝试从L1缓存获取
	if value, ok := m.getFromL1(key); ok {
		if info, ok := value.(map[string]interface{}); ok {
			return info, nil
		}
	}

	// 2. 尝试从L2缓存获取
	value, err := m.getFromL2(ctx, key)
	if err == nil {
		if info, ok := value.(map[string]interface{}); ok {
			m.setToL1(key, info, serverInfoL1TTL)
			return info, nil
		}
	}

	// 3. 缓存未命中，获取实时数据
	info, err := m.getRealtimeServerInfo()
	if err != nil {
		return nil, err
	}

	// 4. 写入缓存
	m.setToL1(key, info, serverInfoL1TTL)
	_ = m.setToL2(ctx, key, info, serverInfoL2TTL)

	return info, nil
}

// getRealtimeServerInfo 获取实时服务器信息
func (m *MetricsCacheManager) getRealtimeServerInfo() (map[string]interface{}, error) {
	// 获取系统指标
	metrics, err := system.GetSystemMetrics()
	if err != nil {
		return nil, fmt.Errorf("获取系统指标失败: %w", err)
	}

	// 获取所有磁盘信息，计算总容量
	var totalDiskSize, availableDiskSize uint64
	disks, err := system.GetAllDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘信息失败: %w", err)
	}

	for _, disk := range disks {
		if disk.Total > 0 && disk.Available <= disk.Total {
			totalDiskSize += disk.Total
			availableDiskSize += disk.Available
		}
	}

	// 计算可用内存
	availableMemory := metrics.TotalMemory - metrics.UsedMemory

	info := map[string]interface{}{
		"hostname":         m.hostname,
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"cpu_count":        float64(runtime.NumCPU()),
		"total_memory":     float64(metrics.TotalMemory),
		"available_memory": float64(availableMemory),
		"disk_total":       float64(totalDiskSize),
		"disk_available":   float64(availableDiskSize),
	}

	return info, nil
}

// startBackgroundTasks 启动后台任务
func (m *MetricsCacheManager) startBackgroundTasks() {
	// 高频任务：更新系统指标
	m.wg.Add(1)
	go m.updateMetricsPeriodically()

	// 低频任务：更新服务器信息
	m.wg.Add(1)
	go m.updateServerInfoPeriodically()

	// 清理任务：清理过期的L1缓存
	m.wg.Add(1)
	go m.cleanupExpiredCache()
}

// updateMetricsPeriodically 定期更新系统指标
func (m *MetricsCacheManager) updateMetricsPeriodically() {
	defer m.wg.Done()

	ticker := time.NewTicker(metricsUpdateTick)
	defer ticker.Stop()

	m.updateMetrics()

	for {
		select {
		case <-ticker.C:
			m.updateMetrics()
		case <-m.stopChan:
			return
		}
	}
}

// updateMetrics 更新系统指标
func (m *MetricsCacheManager) updateMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, err := m.getRealtimeMetrics()
	if err != nil {
		applogger.Warnf("获取系统指标失败: %v", err)
		return
	}

	key := m.getCacheKey("metrics:current")
	m.setToL1(key, metrics, defaultL1TTL)

	if m.redisCache != nil {
		if err := m.setToL2(ctx, key, metrics, defaultL2TTL); err != nil {
			applogger.Warnf("写入L2缓存失败: %v", err)
		}
	}

	applogger.Infof("系统指标已更新: CPU=%.1f%%, Memory=%.1f%%",
		metrics.CPUUsage, metrics.MemoryUsage)
}

// updateServerInfoPeriodically 定期更新服务器信息
func (m *MetricsCacheManager) updateServerInfoPeriodically() {
	defer m.wg.Done()

	ticker := time.NewTicker(serverInfoUpdateTick)
	defer ticker.Stop()

	m.updateServerInfo()

	for {
		select {
		case <-ticker.C:
			m.updateServerInfo()
		case <-m.stopChan:
			return
		}
	}
}

// updateServerInfo 更新服务器信息
func (m *MetricsCacheManager) updateServerInfo() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := m.getRealtimeServerInfo()
	if err != nil {
		applogger.Warnf("获取服务器信息失败: %v", err)
		return
	}

	key := m.getCacheKey("server:info")
	m.setToL1(key, info, serverInfoL1TTL)

	if m.redisCache != nil {
		if err := m.setToL2(ctx, key, info, serverInfoL2TTL); err != nil {
			applogger.Warnf("写入L2缓存失败: %v", err)
		}
	}

	applogger.Infof("服务器信息已更新: %s", info["hostname"])
}

// cleanupExpiredCache 清理过期的L1缓存
func (m *MetricsCacheManager) cleanupExpiredCache() {
	defer m.wg.Done()

	ticker := time.NewTicker(cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			m.memoryCache.Range(func(key, value interface{}) bool {
				if item, ok := value.(*CacheItem); ok {
					if now.After(item.ExpiresAt) {
						m.memoryCache.Delete(key)
					}
				}
				return true
			})
			applogger.Infof("L1缓存清理完成")
		case <-m.stopChan:
			return
		}
	}
}

// Stop 停止缓存管理器
func (m *MetricsCacheManager) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
		m.wg.Wait()
		applogger.Infof("指标缓存管理器已停止")
	})
}

// GetCacheStats 获取缓存统计信息
func (m *MetricsCacheManager) GetCacheStats() map[string]interface{} {
	stats := map[string]interface{}{
		"hostname":      m.hostname,
		"l1_cache_size": 0,
		"redis_enabled": m.redisCache != nil,
	}

	// 统计L1缓存大小
	m.memoryCache.Range(func(key, value interface{}) bool {
		if item, ok := value.(*CacheItem); ok {
			if time.Now().Before(item.ExpiresAt) {
				stats["l1_cache_size"] = stats["l1_cache_size"].(int) + 1
			}
		}
		return true
	})

	// 如果有Redis缓存，获取统计信息
	if m.redisCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if cacheWithStats, ok := m.redisCache.(interface {
			GetStats(ctx context.Context) (map[string]interface{}, error)
		}); ok {
			if redisStats, err := cacheWithStats.GetStats(ctx); err == nil {
				stats["redis_stats"] = redisStats
			}
		}
	}

	return stats
}

// InvalidateCache 失效指定键的缓存
func (m *MetricsCacheManager) InvalidateCache(key string) {
	m.memoryCache.Delete(key)

	if m.redisCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// 使用类型断言检查是否有Delete方法
		if cacheWithDelete, ok := m.redisCache.(interface {
			Delete(ctx context.Context, key string) error
		}); ok {
			_ = cacheWithDelete.Delete(ctx, key)
		}
	}
}

// ClearL1Cache 清空L1缓存
func (m *MetricsCacheManager) ClearL1Cache() {
	m.memoryCache.Range(func(key, value interface{}) bool {
		m.memoryCache.Delete(key)
		return true
	})
}

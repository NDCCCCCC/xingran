package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/pkg/system"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// cacheHealthCheckTimeout 缓存健康检查超时
const cacheHealthCheckTimeout = 5 * time.Second

// MetricsCacheService 系统指标缓存服务
type MetricsCacheService struct {
	core    *Core
	manager *cache.MetricsCacheManager
}

// NewMetricsCacheService 创建系统指标缓存服务
func NewMetricsCacheService(core *Core) *MetricsCacheService {
	var redisCache interface{}
	if core.Cache != nil {
		redisCache = core.Cache
	}

	manager := cache.NewMetricsCacheManager(redisCache)

	return &MetricsCacheService{
		core:    core,
		manager: manager,
	}
}

// GetCurrentMetrics 获取当前系统指标（优先从缓存）
func (s *MetricsCacheService) GetCurrentMetrics(ctx context.Context) (*system.SystemMetrics, error) {
	metricsData, err := s.manager.GetSystemMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取系统指标失败: %w", err)
	}

	// 转换为SystemMetrics格式
	systemMetrics := &system.SystemMetrics{
		CPUUsage:    metricsData.CPUUsage,
		MemoryUsage: metricsData.MemoryUsage,
		DiskUsage:   metricsData.DiskUsage,
		NetworkRx:   metricsData.NetworkRx,
		NetworkTx:   metricsData.NetworkTx,
		ProcessNum:  metricsData.ProcessNum,
		TotalMemory: metricsData.TotalMemory,
		UsedMemory:  metricsData.UsedMemory,
		Timestamp:   metricsData.Timestamp,
	}

	return systemMetrics, nil
}

// GetServerInfo 获取服务器信息（优先从缓存）
func (s *MetricsCacheService) GetServerInfo(ctx context.Context) (map[string]interface{}, error) {
	return s.manager.GetServerInfo(ctx)
}

// RefreshMetrics 刷新系统指标缓存
func (s *MetricsCacheService) RefreshMetrics(ctx context.Context) error {
	// 强制刷新 metrics
	key := fmt.Sprintf("sys:metrics:current:%s", getHostname())

	// 删除L1缓存
	s.manager.InvalidateCache(key)

	// 删除L2缓存
	if s.core.Cache != nil {
		if err := s.core.Cache.Delete(ctx, key); err != nil {
			applogger.Warnf("删除L2缓存失败: %v", err)
		}
	}

	// 触发立即更新
	return s.updateMetricsImmediately(ctx)
}

// RefreshServerInfo 刷新服务器信息缓存
func (s *MetricsCacheService) RefreshServerInfo(ctx context.Context) error {
	key := fmt.Sprintf("sys:server:info:%s", getHostname())

	// 删除L1缓存
	s.manager.InvalidateCache(key)
	if s.core.Cache != nil {
		if err := s.core.Cache.Delete(ctx, key); err != nil {
			applogger.Warnf("删除服务器信息缓存失败: %v", err)
		}
	}

	return nil
}

// updateMetricsImmediately 立即更新指标
func (s *MetricsCacheService) updateMetricsImmediately(ctx context.Context) error {
	systemMetrics, err := system.GetSystemMetrics()
	if err != nil {
		return fmt.Errorf("获取实时系统指标失败: %w", err)
	}

	// 转换为缓存格式
	metricsData := &cache.MetricsData{
		CPUUsage:    systemMetrics.CPUUsage,
		MemoryUsage: systemMetrics.MemoryUsage,
		DiskUsage:   systemMetrics.DiskUsage,
		NetworkRx:   systemMetrics.NetworkRx,
		NetworkTx:   systemMetrics.NetworkTx,
		ProcessNum:  systemMetrics.ProcessNum,
		TotalMemory: systemMetrics.TotalMemory,
		UsedMemory:  systemMetrics.UsedMemory,
		Timestamp:   systemMetrics.Timestamp,
	}

	// 写入缓存
	key := fmt.Sprintf("sys:metrics:current:%s", getHostname())

	// 设置到L2缓存
	if s.core.Cache != nil {
		data, err := json.Marshal(metricsData)
		if err != nil {
			return fmt.Errorf("序列化指标数据失败: %w", err)
		}

		if err := s.core.Cache.Set(ctx, key, data, 5*time.Minute); err != nil {
			return fmt.Errorf("写入L2缓存失败: %w", err)
		}
	}

	applogger.Warnf("系统指标已刷新: CPU=%.1f%%, Memory=%.1f%%",
		metricsData.CPUUsage, metricsData.MemoryUsage)

	return nil
}

// GetCacheStats 获取缓存统计信息
func (s *MetricsCacheService) GetCacheStats() map[string]interface{} {
	stats := s.manager.GetCacheStats()

	// 添加服务层统计
	stats["service"] = map[string]interface{}{
		"name":            "MetricsCacheService",
		"last_update":     time.Now().Format("2006-01-02 15:04:05"),
		"cache_enabled":   s.core.Cache != nil,
		"background_jobs": "running",
	}

	return stats
}

// ClearCache 清空所有缓存
func (s *MetricsCacheService) ClearCache(ctx context.Context) error {
	// 清空L1缓存
	s.manager.ClearL1Cache()

	// 清空L2缓存
	if s.core.Cache != nil {
		if err := s.core.Cache.FlushDB(ctx); err != nil {
			applogger.Warnf("清空L2缓存失败: %v", err)
			return err
		}
	}

	applogger.Infof("所有指标缓存已清空")
	return nil
}

// PreheatCache 预热缓存
func (s *MetricsCacheService) PreheatCache(ctx context.Context) error {
	applogger.Infof("开始预热缓存...")

	// 预热系统指标
	if err := s.updateMetricsImmediately(ctx); err != nil {
		applogger.Warnf("预热系统指标失败: %v", err)
		return err
	}

	// 预热服务器信息
	if _, err := s.GetServerInfo(ctx); err != nil {
		applogger.Warnf("预热服务器信息失败: %v", err)
		return err
	}

	applogger.Infof("缓存预热完成")
	return nil
}

// Stop 停止服务
func (s *MetricsCacheService) Stop() {
	if s.manager != nil {
		s.manager.Stop()
	}
	applogger.Infof("指标缓存服务已停止")
}

// getHostname 获取主机名
func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "localhost"
}

// MetricsHistory 历史指标数据
type MetricsHistory struct {
	Timestamp time.Time `json:"timestamp"`
	CPUUsage  float64   `json:"cpuUsage"`
	MemUsage  float64   `json:"memUsage"`
	DiskUsage float64   `json:"diskUsage"`
}

// GetMetricsHistory 获取历史指标数据
func (s *MetricsCacheService) GetMetricsHistory(ctx context.Context, hours int) ([]MetricsHistory, error) {
	// 从数据库获取历史数据
	var history []MetricsHistory

	// 这里可以添加从数据库查询历史数据的逻辑
	// 暂时返回模拟数据用于演示
	now := time.Now()
	for i := 0; i < hours*60; i += 5 { // 每5分钟一个数据点
		timestamp := now.Add(-time.Duration(i) * time.Minute)
		history = append(history, MetricsHistory{
			Timestamp: timestamp,
			CPUUsage:  10.0 + float64(i%20), // 模拟数据
			MemUsage:  30.0 + float64(i%10), // 模拟数据
			DiskUsage: 45.0 + float64(i%5),  // 模拟数据
		})
	}

	return history, nil
}

// CacheHealthStatus 缓存健康状态
type CacheHealthStatus struct {
	Status       string    `json:"status"`
	L1CacheSize  int       `json:"l1CacheSize"`
	L2CacheSize  int64     `json:"l2CacheSize"`
	LastUpdate   time.Time `json:"lastUpdate"`
	ErrorCount   int       `json:"errorCount"`
	ResponseTime int64     `json:"responseTime"` // 毫秒
}

// GetHealthStatus 获取缓存健康状态
func (s *MetricsCacheService) GetHealthStatus(ctx context.Context) *CacheHealthStatus {
	start := time.Now()

	// 测试缓存性能
	ctx, cancel := context.WithTimeout(ctx, cacheHealthCheckTimeout)
	defer cancel()

	// 尝试获取系统指标测试缓存响应
	_, err := s.GetCurrentMetrics(ctx)

	status := &CacheHealthStatus{
		Status:       "healthy",
		L1CacheSize:  0,
		L2CacheSize:  0,
		LastUpdate:   time.Now(),
		ErrorCount:   0,
		ResponseTime: time.Since(start).Milliseconds(),
	}

	if err != nil {
		status.Status = "unhealthy"
		status.ErrorCount = 1
	}

	// 获取缓存大小
	stats := s.GetCacheStats()
	if l1Size, ok := stats["l1_cache_size"].(int); ok {
		status.L1CacheSize = l1Size
	}

	if redisStats, ok := stats["redis_stats"].(map[string]interface{}); ok {
		if keyCount, ok := redisStats["key_count"].(int64); ok {
			status.L2CacheSize = keyCount
		}
	}

	return status
}

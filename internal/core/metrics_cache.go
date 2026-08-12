package core

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/internal/pkg/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

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

// Stop 停止服务
func (s *MetricsCacheService) Stop() {
	if s.manager != nil {
		s.manager.Stop()
	}
	applogger.Infof("指标缓存服务已停止")
}
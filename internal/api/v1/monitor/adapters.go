package monitor

import (
	"context"
	"errors"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// MetricsProviderAdapter 指标提供者适配器
// 将 core.MetricsCacheService 适配为 MetricsProvider 接口
type MetricsProviderAdapter struct {
	core *core.Core
}

// NewMetricsProviderAdapter 创建指标提供者适配器
func NewMetricsProviderAdapter(core *core.Core) monitorServices.MetricsProvider {
	return &MetricsProviderAdapter{core: core}
}

// GetServerInfo 从缓存获取服务器信息
func (a *MetricsProviderAdapter) GetServerInfo(ctx context.Context) (map[string]interface{}, error) {
	if a.core.MetricsCacheService == nil {
		return nil, errors.New("缓存服务不可用")
	}

	info, err := a.core.MetricsCacheService.GetServerInfo(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, errors.New("缓存服务不可用")
	}

	return info, nil
}

// GetCurrentMetrics 从缓存获取当前指标
func (a *MetricsProviderAdapter) GetCurrentMetrics(ctx context.Context) (*monitorServices.SystemMetricsData, error) {
	if a.core.MetricsCacheService == nil {
		return nil, errors.New("缓存服务不可用")
	}

	metrics, err := a.core.MetricsCacheService.GetCurrentMetrics(ctx)
	if err != nil {
		return nil, err
	}

	return &monitorServices.SystemMetricsData{
		CPUUsage:    metrics.CPUUsage,
		MemoryUsage: metrics.MemoryUsage,
		DiskUsage:   metrics.DiskUsage,
		NetworkRx:   metrics.NetworkRx,
		NetworkTx:   metrics.NetworkTx,
		ProcessNum:  metrics.ProcessNum,
		TotalMemory: metrics.TotalMemory,
		UsedMemory:  metrics.UsedMemory,
		Timestamp:   metrics.Timestamp,
	}, nil
}

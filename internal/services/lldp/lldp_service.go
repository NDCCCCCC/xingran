package lldp

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// LLDPService LLDP发现服务
type LLDPService struct {
	executor *device.DeviceExecutor
	parser   *LLDPParser
	cache    *LLDPCache
}

// NewLLDPService 创建LLDP服务
func NewLLDPService(executor *device.DeviceExecutor) *LLDPService {
	return &LLDPService{
		executor: executor,
		parser:   NewLLDPParser(),
		cache:    NewLLDPCache(1 * time.Hour),
	}
}

// DiscoverNeighbors 发现设备的LLDP邻居
// 返回 map[规范化接口名]*LLDPNeighborInfo
func (s *LLDPService) DiscoverNeighbors(ctx context.Context, netDevice *models.NetworkDevice) (map[string]*models.LLDPNeighborInfo, error) {
	// 尝试从缓存获取
	if neighbors, ok := s.cache.Get(netDevice.ID); ok {
		applogger.Infof("LLDP cache hit for device %s (%s)", netDevice.DeviceName, netDevice.ID)
		return neighbors, nil
	}

	applogger.Infof("LLDP cache miss for device %s (%s), querying device", netDevice.DeviceName, netDevice.ID)

	// 获取LLDP命令
	command := device.GetLLDPCommand(netDevice.Vendor)

	// 执行命令
	output, err := s.executor.ExecuteOnDevice(ctx, netDevice.ID, command, true)
	if err != nil {
		// 记录警告但不返回错误（LLDP可能未启用）
		applogger.Warnf("Failed to execute LLDP command on device %s: %v", netDevice.DeviceName, err)
		// 返回空map而不是error，允许MAC采集继续
		return make(map[string]*models.LLDPNeighborInfo), nil
	}

	// 解析输出
	neighbors, err := s.parser.ParseLLDPNeighbors(output, netDevice.Vendor)
	if err != nil {
		applogger.Errorf("Failed to parse LLDP output for device %s: %v", netDevice.DeviceName, err)
		return make(map[string]*models.LLDPNeighborInfo), nil
	}

	if len(neighbors) == 0 {
		applogger.Infof("No LLDP neighbors found for device %s", netDevice.DeviceName)
		return make(map[string]*models.LLDPNeighborInfo), nil
	}

	// 构建返回map（使用规范化的接口名作为key）
	result := make(map[string]*models.LLDPNeighborInfo)
	for _, neighbor := range neighbors {
		// 规范化接口名以确保MAC采集查找成功
		normalizedKey := portcollection.NormalizeInterfaceName(neighbor.LocalInterface)
		result[normalizedKey] = neighbor
	}

	// 存入缓存
	s.cache.Set(netDevice.ID, result)

	applogger.Infof("Discovered %d LLDP neighbors for device %s", len(result), netDevice.DeviceName)
	return result, nil
}

// GetCache 获取缓存实例（用于测试或手动清除）
func (s *LLDPService) GetCache() *LLDPCache {
	return s.cache
}

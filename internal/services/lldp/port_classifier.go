package lldp

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// ClassifyPort 根据LLDP邻居信息和MAC数量阈值对端口进行分类
//
// 参数:
//   - interfaceName: 原始接口名称
//   - lldpNeighbors: LLDP邻居信息映射（键为规范化的接口名）
//   - macCount: 该接口的MAC地址数量
//   - threshold: 设备类型对应的MAC数量阈值
//
// 返回: PortClassification 端口分类结果
//
// 分类逻辑:
//   1. 如果存在LLDP邻居 → IsUplink=true, Reason=lldp_neighbor
//   2. 否则如果MAC数量超过阈值 → IsUplink=true, Reason=mac_threshold
//   3. 否则 → IsUplink=false, Reason=access
//
// 示例:
//   // 存在LLDP邻居的上行端口
//   result := ClassifyPort("GigabitEthernet0/1", lldpNeighbors, 150, 10)
//   // result.IsUplink=true, Reason=lldp_neighbor
//
//   // MAC数量超阈值的端口
//   result := ClassifyPort("GigabitEthernet0/2", lldpNeighbors, 50, 10)
//   // result.IsUplink=true, Reason=mac_threshold
//
//   // 普通接入端口
//   result := ClassifyPort("GigabitEthernet0/3", lldpNeighbors, 2, 10)
//   // result.IsUplink=false, Reason=access
func ClassifyPort(
	interfaceName string,
	lldpNeighbors map[string]*models.LLDPNeighborInfo,
	macCount int,
	threshold int,
) models.PortClassification {
	// 标准化接口名称（与LLDP服务的 NormalizeInterfaceName 逻辑一致）
	normalizedName := portcollection.NormalizeInterfaceName(interfaceName)

	// 检查是否存在LLDP邻居
	if neighbor, exists := lldpNeighbors[normalizedName]; exists {
		return models.PortClassification{
			InterfaceName:   interfaceName,
			NormalizedName:  normalizedName,
			IsUplink:        true,
			Reason:          models.PortReasonLLDPNeighbor,
			MACCount:        macCount,
			Threshold:       threshold,
			HasLLDPNeighbor: true,
			NeighborName:    neighbor.NeighborName,
		}
	}

	// 检查MAC数量是否超过阈值
	if macCount > threshold {
		return models.PortClassification{
			InterfaceName:   interfaceName,
			NormalizedName:  normalizedName,
			IsUplink:        true,
			Reason:          models.PortReasonMACThreshold,
			MACCount:        macCount,
			Threshold:       threshold,
			HasLLDPNeighbor: false,
		}
	}

	// 普通接入端口
	return models.PortClassification{
		InterfaceName:   interfaceName,
		NormalizedName:  normalizedName,
		IsUplink:        false,
		Reason:          models.PortReasonAccess,
		MACCount:        macCount,
		Threshold:       threshold,
		HasLLDPNeighbor: false,
	}
}

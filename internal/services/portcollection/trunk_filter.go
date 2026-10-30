package portcollection

import (
	"context"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// trunkFilterHintPortTypeValues 是 PortType 字段中可能暗示该端口为 trunk / hybrid /
// 互联上联口的取值（厂商原始输出未经额外清洗时常见值）。
//
// 注意（mac-collection-trunk-filter）：
// 当前 sys_device_port_status 暂无独立的 vlan_link_type 列，本过滤仅能基于
// sys_device_port_status.port_type 做粗粒度启发。精确的 trunk/access 区分
// 需要：
//   1. 新增迁移给 sys_device_port_status 加 vlan_link_type 列
//   2. 端口采集阶段调用厂商 "display port vlan" / "show interfaces switchport"
//      并用 TextFSM 模板解析 link_type 写入该列
//   3. 本过滤函数改成读 vlan_link_type IN ('trunk','hybrid')
//
// 该升级属于本 session Resolution 描述的 Layer 1/2/3 三层修复，
// 推迟到后续 phase（需要厂商命令模板与迁移协同）。本次落最小可编译
// trunk_filter 模块 + 接入 MAC 采集的 PortType 兜底过滤。
var trunkFilterHintPortTypeValues = []string{
	"trunk",
	"hybrid",
	"uplink",
	"uplink-port",
}

// BuildTrunkPortBlockset 查询 sys_device_port_status，返回指定设备的
// "疑似 trunk / 互联" 端口名集合（规范化前后都加入，匹配 MAC 采集的两套接口名形态）。
//
// 返回的 set 用于 MAC 采集阶段过滤掉 trunk 端口上的重复 MAC 地址。
// 查询失败时返回空集合 + nil error（让上层继续采集，不阻断主流程）。
func BuildTrunkPortBlockset(ctx context.Context, db *gorm.DB, deviceID string) map[string]bool {
	blockset := make(map[string]bool)
	if db == nil || deviceID == "" {
		return blockset
	}

	var ports []models.DevicePortStatus
	// 只取 PortType 非空的，避免空字符串误命中
	if err := db.WithContext(ctx).
		Select("interface_name, port_type").
		Where("device_id = ? AND port_type IS NOT NULL AND port_type <> ''", deviceID).
		Find(&ports).Error; err != nil {
		applogger.Warnf("[trunk_filter] 查询 sys_device_port_status 失败 (deviceID=%s): %v", deviceID, err)
		return blockset
	}

	for _, p := range ports {
		pt := strings.ToLower(strings.TrimSpace(p.PortType))
		if pt == "" {
			continue
		}
		for _, hint := range trunkFilterHintPortTypeValues {
			if pt == hint || strings.Contains(pt, hint) {
				// 原始名 + 规范化名都加入，上层 MAC 采集用 NormalizeInterfaceName 比对
				blockset[p.InterfaceName] = true
				blockset[NormalizeInterfaceName(p.InterfaceName)] = true
				break
			}
		}
	}
	return blockset
}

// IsTrunkPort 判断给定接口名是否落在 trunk blockset 中。
func IsTrunkPort(blockset map[string]bool, interfaceName string) bool {
	if len(blockset) == 0 {
		return false
	}
	if blockset[interfaceName] {
		return true
	}
	return blockset[NormalizeInterfaceName(interfaceName)]
}

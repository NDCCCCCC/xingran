package models

// PortClassificationReason 端口分类原因枚举
type PortClassificationReason string

const (
	PortReasonLLDPNeighbor PortClassificationReason = "lldp_neighbor" // LLDP检测到邻居设备
	PortReasonMACThreshold PortClassificationReason = "mac_threshold" // MAC数量超过阈值
	PortReasonAccess       PortClassificationReason = "access"         // 普通接入端口
)

// PortClassification 端口分类结果（瞬时数据结构，不持久化到数据库）
type PortClassification struct {
	InterfaceName   string                     `json:"interfaceName"` // 原始接口名称
	NormalizedName  string                     `json:"normalizedName"` // 标准化后的接口名称
	IsUplink        bool                       `json:"isUplink"`       // 是否为上行链路端口
	Reason          PortClassificationReason   `json:"reason"`         // 分类原因
	MACCount        int                        `json:"macCount"`       // 该接口的MAC地址数量
	Threshold       int                        `json:"threshold"`      // 设备类型对应的MAC数量阈值
	HasLLDPNeighbor bool                       `json:"hasLLDPNeighbor"` // 是否检测到LLDP邻居
	NeighborName    string                     `json:"neighborName,omitempty"` // 邻居设备名称（如果存在）
}

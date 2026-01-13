package operations

import (
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// DedicatedLineType 专线类型枚举
type DedicatedLineType string

const (
	LineTypeInternet    DedicatedLineType = "internet"    // 互联网专线
	LineTypeMPLS        DedicatedLineType = "mpls"        // MPLS VPN
	LineTypeLeasedLine  DedicatedLineType = "leased_line" // 租用专线
	LineTypeFiber       DedicatedLineType = "fiber"       // 光纤专线
)

// LineStatus 专线状态枚举
type LineStatus int

const (
	LineStatusNormal   LineStatus = 0 // 正常
	LineStatusFault    LineStatus = 1 // 故障
	LineStatusDisabled LineStatus = 2 // 停用
)

// OpsDedicatedLine 专线模型
type OpsDedicatedLine struct {
	models.BaseModel
	Name           string             // 专线名称
	LineCode       string             // 专线编码（唯一）
	LineType       DedicatedLineType  // 专线类型
	Bandwidth      string             // 带宽（如：100M, 1G）
	ISP            string             // 运营商（电信/移动/联通等）
	SourceDeviceID string             // 源设备ID（关联sys_network_device）
	SourcePort     string             // 源端口
	DestDeviceID   string             // 目的设备ID（关联sys_network_device）
	DestPort       string             // 目的端口
	IPAddress      string             // IP地址（可选）
	SubnetMask     string             // 子网掩码（可选）
	Gateway        string             // 网关（可选）
	VLAN           string             // VLAN（可选）
	Status         LineStatus         // 状态: 0=正常, 1=故障, 2=停用
	ActivationDate *time.Time         // 启用日期
	ExpiryDate     *time.Time         // 到期日期
	ResponsibleID  string             // 负责人ID（关联sys_user）
	ContractNo     string             // 合同编号
	MonthlyFee     float64            // 月租费用
	Remark         string             // 备注
}

// TableName 指定表名
func (OpsDedicatedLine) TableName() string {
	return "ops_dedicated_lines"
}

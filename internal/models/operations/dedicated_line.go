package operations

import (
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
	LineCode              string            `gorm:"uniqueIndex;size:50;not null" json:"lineCode"`              // 专线编码（唯一）
	LineType              DedicatedLineType `gorm:"size:50;not null" json:"lineType"`                         // 专线类型
	Bandwidth             *string           `gorm:"size:50" json:"bandwidth,omitempty"`                       // 带宽（如：100M, 1G），选填
	ISP                   string            `gorm:"size:100;not null" json:"isp"`                             // 运营商（电信/移动/联通等）
	SourceDeviceID        *string           `gorm:"size:64" json:"sourceDeviceId,omitempty"`                  // 源设备ID
	SourceDeviceName      *string           `gorm:"size:100" json:"sourceDeviceName,omitempty"`               // 源设备名称
	SourcePort            *string           `gorm:"size:50" json:"sourcePort,omitempty"`                      // 源端口
	SourceRoomID          *string           `gorm:"size:64" json:"sourceRoomId,omitempty"`                    // 源机房ID
	SourceRoomName        *string           `gorm:"size:100" json:"sourceRoomName,omitempty"`                 // 源机房名称（冗余）
	DestDeviceID          *string           `gorm:"size:64" json:"destDeviceId,omitempty"`                    // 目的设备ID
	DestDeviceName        *string           `gorm:"size:100" json:"destDeviceName,omitempty"`                 // 目的设备名称
	DestPort              *string           `gorm:"size:50" json:"destPort,omitempty"`                       // 目的端口
	DestRoomID            *string           `gorm:"size:64" json:"destRoomId,omitempty"`                      // 目的机房ID
	DestRoomName          *string           `gorm:"size:100" json:"destRoomName,omitempty"`                   // 目的机房名称（冗余）
	IPAddress             *string           `gorm:"size:50" json:"ipAddress,omitempty"`                      // IP地址（可选）
	SubnetMask            *string           `gorm:"size:50" json:"subnetMask,omitempty"`                     // 子网掩码（可选）
	VLAN                  *string           `gorm:"size:50" json:"vlan,omitempty"`                           // VLAN（可选）
	Status                LineStatus        `gorm:"default:0" json:"status"`                                 // 状态: 0=正常, 1=故障, 2=停用
	CarrierContactID      *string           `gorm:"size:64" json:"carrierContactId,omitempty"`               // 运营商联系人ID
	CarrierContactName    *string           `gorm:"size:100" json:"carrierContactName,omitempty"`            // 运营商联系人姓名
	CarrierContactPhone   *string           `gorm:"size:20" json:"carrierContactPhone,omitempty"`            // 运营商联系人电话
	ContractNo            *string           `gorm:"size:100" json:"contractNo,omitempty"`                    // 合同编号
	MonthlyFee            float64           `gorm:"default:0" json:"monthlyFee"`                             // 月租费用
	Remark                *string           `gorm:"size:500" json:"remark,omitempty"`                       // 备注
}

// TableName 指定表名
func (OpsDedicatedLine) TableName() string {
	return "ops_dedicated_lines"
}

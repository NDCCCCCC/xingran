package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
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
	Name                string     `gorm:"size:100;not null" json:"name"`                // 专线名称
	LineType            string     `gorm:"size:50;not null" json:"lineType"`             // 专线类型（数据字典）
	Bandwidth           *string    `gorm:"size:50" json:"bandwidth,omitempty"`           // 带宽（如：100M, 1G），选填
	ISP                 string     `gorm:"size:100;not null" json:"isp"`                 // 运营商（电信/移动/联通等）
	SourceDeviceID      *string    `gorm:"size:64" json:"sourceDeviceId,omitempty"`      // 源设备ID
	SourceDeviceName    *string    `gorm:"size:100" json:"sourceDeviceName,omitempty"`   // 源设备名称
	SourcePort          *string    `gorm:"size:50" json:"sourcePort,omitempty"`          // 源端口
	SourceRoomID        *string    `gorm:"size:64" json:"sourceRoomId,omitempty"`        // 源机房ID
	SourceRoomName      *string    `gorm:"size:100" json:"sourceRoomName,omitempty"`     // 源机房名称（冗余）
	DestDeviceID        *string    `gorm:"size:64" json:"destDeviceId,omitempty"`        // 目的设备ID
	DestDeviceName      *string    `gorm:"size:100" json:"destDeviceName,omitempty"`     // 目的设备名称
	DestPort            *string    `gorm:"size:50" json:"destPort,omitempty"`            // 目的端口
	DestRoomID          *string    `gorm:"size:64" json:"destRoomId,omitempty"`          // 目的机房ID
	DestRoomName        *string    `gorm:"size:100" json:"destRoomName,omitempty"`       // 目的机房名称（冗余）
	SourceIPAddress     *string    `gorm:"size:50" json:"sourceIpAddress,omitempty"`     // 源IP地址（可选）
	DestIPAddress       *string    `gorm:"size:50" json:"destIpAddress,omitempty"`       // 目的IP地址（可选）
	SourceSubnetMask    *string    `gorm:"size:50" json:"sourceSubnetMask,omitempty"`    // 源子网掩码（可选）
	DestSubnetMask      *string    `gorm:"size:50" json:"destSubnetMask,omitempty"`      // 目的子网掩码（可选）
	VLAN                *string    `gorm:"size:50" json:"vlan,omitempty"`                // VLAN（可选）
	Status              LineStatus `gorm:"default:0" json:"status"`                      // 状态: 0=正常, 1=故障, 2=停用
	CarrierContactID    *string    `gorm:"size:64" json:"carrierContactId,omitempty"`    // 运营商联系人ID
	CarrierContactName  *string    `gorm:"size:100" json:"carrierContactName,omitempty"` // 运营商联系人姓名
	CarrierContactPhone *string    `gorm:"size:20" json:"carrierContactPhone,omitempty"` // 运营商联系人电话
	MonthlyFee          float64    `gorm:"default:0" json:"monthlyFee"`                  // 月租费用
	Remark              *string    `gorm:"size:500" json:"remark,omitempty"`             // 备注
}

// TableName 指定表名
func (OpsDedicatedLine) TableName() string {
	return "ops_dedicated_lines"
}

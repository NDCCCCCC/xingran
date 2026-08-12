package models

import "time"

// DeviceSource 设备来源枚举
type DeviceSource string

const (
	DeviceSourceAD       DeviceSource = "ad"       // 域控设备
	DeviceSourceAsset    DeviceSource = "asset"    // 资产系统
	DeviceSourceManual   DeviceSource = "manual"   // 手动添加
	DeviceSourcePhysical DeviceSource = "physical" // R5 物理链路设备(MAC→port→infoPoint→workstation 反推)
)

// WorkstationDevice 工位设备关联模型
type WorkstationDevice struct {
	BaseModel

	// 关联关系
	WorkstationID string       `gorm:"type:uuid;not null;index:idx_workstation_device_workstation,priority:1" json:"workstationId"` // 工位ID
	AssetID       *string      `gorm:"type:uuid;index:idx_workstation_device_asset" json:"assetId,omitempty"`                                                  // 关联资产ID
	ADComputerID  *string      `gorm:"type:uuid;index:idx_workstation_device_ad_computer" json:"adComputerId,omitempty"`                                    // 关联AD设备ID

	// 设备来源
	DeviceSource DeviceSource `gorm:"size:20;not null;index:idx_workstation_device_source_status,priority:1" json:"deviceSource"` // 设备来源: ad, asset, manual

	// 设备信息
	DeviceSerial  *string `gorm:"size:200;index:idx_workstation_device_serial" json:"deviceSerial,omitempty"`   // 序列号
	DeviceName    *string `gorm:"size:255" json:"deviceName,omitempty"`                                       // 设备名称
	DeviceModel   *string `gorm:"size:200" json:"deviceModel,omitempty"`                                      // 型号
	DeviceType    *string `gorm:"size:100" json:"deviceType,omitempty"`                                       // 类型
	MACAddress    *string `gorm:"size:100" json:"macAddress,omitempty"`                                       // MAC地址
	IPAddress     *string `gorm:"size:64;index:idx_workstation_device_ip" json:"ipAddress,omitempty"`              // IP地址

	// 责任人信息
	ResponsibleUser   *string `gorm:"size:100" json:"responsibleUser,omitempty"`    // 责任人
	ResponsibleUserID *string `gorm:"size:64" json:"responsibleUserId,omitempty"`    // 责任人ID

	// 状态与优先级
	Status     int  `gorm:"default:0;index:idx_workstation_device_source_status,priority:2" json:"status"` // 状态: 0=正常, 1=停用
	IsPrimary bool `gorm:"default:false" json:"isPrimary"`                                                        // 是否主设备
	Priority   int  `gorm:"default:0" json:"priority"`                                                              // 优先级

	// 备注
	Description *string `gorm:"type:text" json:"description,omitempty"` // 备注

	// 置信度（仅 R5 物理链路设备使用,不入库）: 1.0=实测MAC命中, 0.5=仅历史MAC命中, nil=AD/资产/手动来源(未分级)
	Confidence *float64 `gorm:"-" json:"confidence,omitempty"`
	// 历史关联最后上线时间(仅 R5 物理链路设备,仅历史MAC命中时有值;不入库)
	HistoryLastSeen *time.Time `gorm:"-" json:"historyLastSeen,omitempty"`

	// 关联对象（GORM不会存储到数据库）
	Workstation  *Workstation `gorm:"foreignKey:WorkstationID" json:"workstation,omitempty"`
	Asset        *Asset       `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
	ADComputer   *ADComputer  `gorm:"foreignKey:ADComputerID" json:"adComputer,omitempty"`
}

// TableName 设置表名
func (WorkstationDevice) TableName() string {
	return "ops_workstation_device"
}

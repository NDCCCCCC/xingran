package models

// 工位状态枚举
type WorkstationStatus int

const (
	WorkstationStatusAvailable WorkstationStatus = 0 // 空闲 - 可分配
	WorkstationStatusOccupied  WorkstationStatus = 1 // 占用 - 已分配给用户
	WorkstationStatusMaintain  WorkstationStatus = 2 // 维护 - 维修中不可用
)

// 工位类型枚举
type WorkstationType int

const (
	WorkstationTypeFixed   WorkstationType = 0 // 固定工位
	WorkstationTypeHotDesk WorkstationType = 1 // 灵活工位
	WorkstationTypeManager WorkstationType = 2 // 管理工位
)

// 桌型枚举
const (
	DeskTypeStraight = 0 // 一字型
	DeskTypeLShape   = 1 // L型
)

// 默认值常量
const (
	DefaultWorkstationRotation = 0   // 默认旋转角度（度）
	DefaultWorkstationWidth    = 160 // 默认工位宽度（厘米）
	DefaultWorkstationDepth    = 70  // 默认工位深度（厘米）
	DefaultWorkstationDeskType = 0   // 默认桌型
)

// Workstation 工位模型（支持运维管理）
type Workstation struct {
	BaseModel

	// 基本信息
	WorkstationName string            `gorm:"size:100;not null" json:"name"`         // 工位名称
	WorkstationType WorkstationType   `gorm:"default:0" json:"type"`                 // 工位类型
	Status          WorkstationStatus `gorm:"default:0" json:"status"`               // 工位状态
	Description     *string           `gorm:"size:500" json:"description,omitempty"` // 备注

	// 组织信息
	DeptID       *string `gorm:"size:64" json:"deptId,omitempty"`        // 部门ID
	DeptName     *string `gorm:"size:100" json:"deptName,omitempty"`     // 部门名称
	BuildingID   *string `gorm:"size:64" json:"buildingId,omitempty"`    // 楼宇ID
	BuildingName *string `gorm:"size:100" json:"buildingName,omitempty"` // 楼宇名称

	// 位置信息
	Location  *string `gorm:"size:200" json:"location,omitempty"`  // 位置描述
	Floor     *string `gorm:"size:50" json:"floor,omitempty"`      // 楼层字符串（旧字段）
	FloorID   *string `gorm:"size:64" json:"floorId,omitempty"`    // 楼层ID（关联ops_floors）
	FloorName *string `gorm:"size:100" json:"floorName,omitempty"` // 楼层名称

	// 用户信息
	UserID   *string `gorm:"size:64" json:"userId,omitempty"`    // 所属人员ID
	UserName *string `gorm:"size:100" json:"userName,omitempty"` // 所属人员名称

	// 主设备序列号（来自 ops_workstation_device 子查询，非表列；gorm:->;-:migration 设为只读且跳过 AutoMigrate）
	PrimaryDeviceSerial *string `gorm:"->;-:migration" json:"primaryDeviceSerial,omitempty"`

	// 平面图信息
	PositionX *int `json:"positionX,omitempty"`                 // X坐标
	PositionY *int `json:"positionY,omitempty"`                 // Y坐标
	Rotation  *int `gorm:"default:0" json:"rotation,omitempty"` // 旋转角度（度）0-360

	// 尺寸信息（单位：毫米）
	Width    *int `gorm:"default:160" json:"width,omitempty"`  // 宽度，默认1600mm
	Depth    *int `gorm:"default:70" json:"depth,omitempty"`   // 深度，默认700mm
	DeskType *int `gorm:"default:0" json:"deskType,omitempty"` // 桌型：0=一字型, 1=L型

	// R4 (Phase 45) — 资产对账健康度集成 (D-A1-01/03)
	//
	// 三个字段在 WorkstationHandler.GetByID 注入,service 层 (WorkstationService) 不感知
	// 保持与现有 handler/service 分层一致(WorkstationService 不引入 asset 依赖)。
	// 字段为 handler-only 虚拟属性,不映射到 DB 列 — 用 gorm:"-" 完全忽略(无 migration、query、scan),
	// 因为 List 端点用 Select(workstationJoinSelect) 含 sys_workstation.* 显式覆盖 GORM 字段标签,
	// 若用 gorm:"->;-:migration" 会让 GORM 尝试找 reconciliation 列 → "Table not set" 错(已修)。
	// (参 MEMORY: gorm-migration-tag-does-not-block-insert;原 →;-:migration 适用于有 DBName 的 JOIN 字段)
	//
	// Reconciliation 使用弱类型 map[string]interface{} 避免 models → asset → models 循环依赖
	// (asset 包内含 models 引用)。handler 层注入前做一次 json marshal/unmarshal 类型转换。
	// 无权限时 Reconciliation=nil + ReconciliationVisible=false (D-A1-03 静默降级)
	Reconciliation             map[string]interface{} `gorm:"-" json:"reconciliation,omitempty"`
	ReconciliationVisible      bool                   `gorm:"-" json:"reconciliationVisible"`
	ReconciliationHiddenReason string                 `gorm:"-" json:"reconciliationHiddenReason,omitempty"`
}

// TableName 设置表名
func (Workstation) TableName() string {
	return "sys_workstation"
}

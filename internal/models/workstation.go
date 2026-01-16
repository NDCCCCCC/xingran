package models

// WorkstationStatus 工位状态枚举
type WorkstationStatus int

const (
	WorkstationStatusAvailable WorkstationStatus = 0 // 空闲 - 可分配
	WorkstationStatusOccupied  WorkstationStatus = 1 // 占用 - 已分配给用户
	WorkstationStatusMaintain  WorkstationStatus = 2 // 维护 - 维修中不可用
)

// WorkstationType 工位类型枚举
type WorkstationType int

const (
	WorkstationTypeFixed   WorkstationType = 0 // 固定工位
	WorkstationTypeHotDesk WorkstationType = 1 // 灵活工位
	WorkstationTypeManager WorkstationType = 2 // 管理工位
)

// Workstation 工位模型（增强版，支持运维管理）
type Workstation struct {
	BaseModel
	WorkstationName string            `gorm:"size:100;not null" json:"name"`
	DeptID          *string           `gorm:"size:64" json:"deptId,omitempty"`
	DeptName        *string           `gorm:"size:100" json:"deptName,omitempty"`
	Location        *string           `gorm:"size:200" json:"location,omitempty"`
	Floor           *string           `gorm:"size:50" json:"floor,omitempty"`      // 旧字段：楼层字符串
	FloorID         *string           `gorm:"size:64" json:"floorId,omitempty"`    // 新增：楼层ID（关联ops_floors）
	FloorName       *string           `gorm:"size:100" json:"floorName,omitempty"` // 新增：楼层名称
	WorkstationType WorkstationType   `gorm:"default:0" json:"type"`
	Status          WorkstationStatus `gorm:"default:0" json:"status"`
	Description     *string           `gorm:"size:500" json:"remark,omitempty"`    // remark 兼容前端
	UserID          *string           `gorm:"size:64" json:"userId,omitempty"`     // 所属人员ID
	UserName        *string           `gorm:"size:100" json:"userName,omitempty"`  // 所属人员名称
	PositionX       *int              `json:"positionX,omitempty"`                 // 平面图X坐标
	PositionY       *int              `json:"positionY,omitempty"`                 // 平面图Y坐标
	Rotation        *int              `gorm:"default:0" json:"rotation,omitempty"` // 平面图旋转角度（度），0-360
}

// TableName 设置表名
func (Workstation) TableName() string {
	return "sys_workstation"
}

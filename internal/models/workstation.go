package models

// WorkstationStatus 工位状态枚举
type WorkstationStatus int

const (
	WorkstationStatusEnabled  WorkstationStatus = 0 // 启用
	WorkstationStatusDisabled WorkstationStatus = 1 // 禁用
)

// WorkstationType 工位类型枚举
type WorkstationType int

const (
	WorkstationTypeFixed   WorkstationType = 0 // 固定工位
	WorkstationTypeHotDesk WorkstationType = 1 // 灵活工位
	WorkstationTypeManager WorkstationType = 2 // 管理工位
)

// Workstation 工位模型
type Workstation struct {
	BaseModel
	WorkstationCode string            `gorm:"uniqueIndex;size:50;not null" json:"workstationCode"`
	WorkstationName string            `gorm:"size:100;not null" json:"workstationName"`
	DeptID          *string           `gorm:"size:64" json:"deptId,omitempty"`
	DeptName        *string           `gorm:"size:100" json:"deptName,omitempty"`
	Location        *string           `gorm:"size:200" json:"location,omitempty"`
	Floor           *string           `gorm:"size:50" json:"floor,omitempty"`
	WorkstationType WorkstationType   `gorm:"default:0" json:"workstationType"`
	Status          WorkstationStatus `gorm:"default:0" json:"status"`
	Capacity        int               `gorm:"default:1" json:"capacity"`
	Description     *string           `gorm:"size:500" json:"description,omitempty"`
	UserID          *string           `gorm:"size:64" json:"userId,omitempty"`
	UserName        *string           `gorm:"size:100" json:"userName,omitempty"`
}

// TableName 设置表名
func (Workstation) TableName() string {
	return "sys_workstation"
}

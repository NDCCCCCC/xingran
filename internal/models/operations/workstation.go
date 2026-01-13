package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// WorkstationStatus 工位状态枚举
type WorkstationStatus int

const (
	WorkstationStatusFree    WorkstationStatus = 0 // 空闲
	WorkstationStatusOccupied WorkstationStatus = 1 // 占用
)

// WorkstationType 工位类型枚举
type WorkstationType int

const (
	WorkstationTypeFixed  WorkstationType = 0 // 固定工位
	WorkstationTypeShared WorkstationType = 1 // 共享工位
)

// OpsWorkstation 工位模型
type OpsWorkstation struct {
	models.BaseModel
	Code        string                // 工位编号
	Name        string                // 工位名称
	FloorID     string                // 所属楼层ID
	UserID      string                // 关联用户ID
	DeptID      string                // 关联部门ID
	PhonePoint  string                // 电话信息点
	Status      WorkstationStatus     // 状态: 0=空闲, 1=占用
	Type        WorkstationType       // 类型: 0=固定, 1=共享
	Equipment   string                // 设备配置（JSON）
	NetworkInfo string                // 网络信息
	Remark      string                // 备注
	PositionX   *int                  // 平面图X坐标（可选）
	PositionY   *int                  // 平面图Y坐标（可选）
}

// TableName 指定表名
func (OpsWorkstation) TableName() string {
	return "ops_workstations"
}

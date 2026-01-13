package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// FloorStatus 楼层状态枚举
type FloorStatus int

const (
	FloorStatusNormal  FloorStatus = 0 // 正常
	FloorStatusStopped FloorStatus = 1 // 停用
)

// OpsFloor 楼层模型
type OpsFloor struct {
	models.BaseModel
	Name       string      // 楼层名称
	FloorNo    string      // 楼层编号（如1F, 2F, B1）
	BuildingID string      // 所属楼宇ID
	Status     FloorStatus // 状态: 0=正常, 1=停用
	Remark     string      // 备注
	OrderNum   int         // 排序号
}

// TableName 指定表名
func (OpsFloor) TableName() string {
	return "ops_floors"
}

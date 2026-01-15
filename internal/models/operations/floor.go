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
	Name         string      `gorm:"size:100;not null" json:"name"`                // 楼层名称
	FloorNo      string      `gorm:"size:50;not null" json:"floorNo"`              // 楼层编号（如1F, 2F, B1）
	BuildingID   string      `gorm:"size:64;not null" json:"buildingId"`           // 所属楼宇ID
	BuildingName *string     `gorm:"size:100" json:"buildingName,omitempty"`      // 所属楼宇名称
	Status       FloorStatus `gorm:"default:0" json:"status"`                      // 状态: 0=正常, 1=停用
	Remark       *string     `gorm:"size:500" json:"remark,omitempty"`             // 备注
	OrderNum     int         `gorm:"default:0" json:"orderNum"`                    // 排序号
}

// TableName 指定表名
func (OpsFloor) TableName() string {
	return "ops_floors"
}

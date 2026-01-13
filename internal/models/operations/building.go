package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// BuildingStatus 楼宇状态枚举
type BuildingStatus int

const (
	BuildingStatusNormal  BuildingStatus = 0 // 正常
	BuildingStatusStopped BuildingStatus = 1 // 停用
)

// OpsBuilding 楼宇模型
type OpsBuilding struct {
	models.BaseModel
	Name     string         // 楼宇名称
	Code     string         // 楼宇编码（唯一）
	Address  string         // 地址
	OrgID    string         // 所属机构ID（关联sys_dept）
	Status   BuildingStatus // 状态: 0=正常, 1=停用
	Remark   string         // 备注
	OrderNum int            // 排序号
}

// TableName 指定表名
func (OpsBuilding) TableName() string {
	return "ops_buildings"
}

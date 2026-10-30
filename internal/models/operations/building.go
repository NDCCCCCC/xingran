package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
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
	Name        string         `gorm:"size:100;not null" json:"name"`                 // 楼宇名称
	Address     string         `gorm:"size:200" json:"address"`                       // 详细地址
	Longitude   *float64       `gorm:"type:decimal(11,8)" json:"longitude,omitempty"` // 经度（通过地址自动解析）
	Latitude    *float64       `gorm:"type:decimal(11,8)" json:"latitude,omitempty"`  // 纬度（通过地址自动解析）
	Level       int            `gorm:"default:2;not null" json:"level"`               // 层级：1=城市级汇总，2=具体楼宇
	OrgID       string         `gorm:"size:64" json:"orgId"`                          // 所属机构ID（关联sys_dept）
	OrgName     *string        `gorm:"size:100" json:"orgName,omitempty"`             // 所属机构名称
	TotalFloors      int            `gorm:"default:0" json:"totalFloors"`                  // 楼层数（根据创建的楼层自动计算）
	WorkstationCount int            `gorm:"column:workstation_count;->" json:"workstationCount"` // 工位数(子查询动态计算,非持久化;->只读)
	Status           BuildingStatus `gorm:"default:0" json:"status"`                       // 状态: 0=正常, 1=停用
	Remark      *string        `gorm:"size:500" json:"remark,omitempty"`              // 备注
	OrderNum    int            `gorm:"default:0" json:"orderNum"`                     // 排序号

	// FloorCount 楼层数实时统计（buildingService.List 按页内 building_id 批量
	// GROUP BY 注入，非表列）：ops_floors 中该楼栋非软删行数。
	// 必须 gorm:"-"（完全忽略）而非 gorm:"->"：List 的 selectScope 用
	// Select("ops_buildings.*, ...") 显式选列，-> 会让 GORM 尝试扫描不存在的列
	// （floor.go WorkstationStats 同款教训）。nil = 聚合未覆盖/无楼层，
	// 配合 omitempty 序列化不出键（前端渲染 "-"）。
	FloorCount *int64 `gorm:"-" json:"floorCount,omitempty"`
}

// TableName 指定表名
func (OpsBuilding) TableName() string {
	return "ops_buildings"
}

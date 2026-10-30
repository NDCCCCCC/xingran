package operations

import (
	"encoding/json"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

type FloorStatus int

const (
	FloorStatusNormal  FloorStatus = 0
	FloorStatusStopped FloorStatus = 1
)

// FloorWorkstationStats 楼层工位统计(List 批量聚合注入的虚拟字段,不映射 DB 列)。
// 状态语义跟随 sys_workstation.status 独立枚举(非全局 0=正常/1=停用惯例):
// 0=空闲(未使用) / 1=占用(已使用) / 2=维护。
type FloorWorkstationStats struct {
	Total       int64 `json:"total"`
	Used        int64 `json:"used"`        // status=1 占用 → 已使用
	Unused      int64 `json:"unused"`      // status=0 空闲 → 未使用
	Maintenance int64 `json:"maintenance"` // status=2 维护
}

type OpsFloor struct {
	models.BaseModel
	Name         string      `gorm:"size:100;not null" json:"name"`
	FloorNo      string      `gorm:"column:floor_no;size:50;not null" json:"floorNo"`
	BuildingID   string      `gorm:"column:building_id;size:64;not null" json:"buildingId"`
	BuildingName *string     `gorm:"column:building_name;size:100" json:"buildingName,omitempty"`
	Area         *float64    `gorm:"type:numeric(10,2)" json:"area,omitempty"`
	Status       FloorStatus `gorm:"default:0" json:"status"`
	Remark       *string     `gorm:"size:500" json:"-"`
	Description  *string     `gorm:"-" json:"description,omitempty"`
	OrderNum     int         `gorm:"default:0" json:"orderNum"`
	PlanImageID  *string     `gorm:"type:uuid" json:"planImageId,omitempty"`
	PlanImageUrl *string     `gorm:"size:500" json:"planImageUrl,omitempty"`

	// WorkstationStats 楼层工位统计(floorService.List 按页内 floor_id 批量聚合注入,非表列)。
	// 必须 gorm:"-"(完全忽略)而非 gorm:"->;-:migration":List 的 joinScope 用
	// Select("ops_floors.*, ...") 显式选列,-> 会让 GORM 尝试扫描不存在的列
	// (workstation.go Reconciliation 同款教训)。现有 MarshalJSON(嵌入 *Alias)
	// 自动带出本字段;无工位楼层保持 nil,配合 omitempty 序列化时不出键。
	WorkstationStats *FloorWorkstationStats `gorm:"-" json:"workstationStats,omitempty"`
}

func (f *OpsFloor) BeforeSave(tx *gorm.DB) error {
	f.Remark = f.Description
	return nil
}

func (f *OpsFloor) AfterFind(tx *gorm.DB) error {
	f.Description = f.Remark
	return nil
}

func (f OpsFloor) MarshalJSON() ([]byte, error) {
	type Alias OpsFloor
	return json.Marshal(&struct {
		Description *string `json:"description,omitempty"`
		*Alias
	}{
		Description: f.Remark,
		Alias:       (*Alias)(&f),
	})
}

func (f *OpsFloor) UnmarshalJSON(data []byte) error {
	type Alias OpsFloor
	aux := &struct {
		Description *string `json:"description,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	f.Remark = aux.Description
	return nil
}

func (OpsFloor) TableName() string {
	return "ops_floors"
}

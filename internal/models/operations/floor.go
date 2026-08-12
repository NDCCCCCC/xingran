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

package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 常量定义

// 默认门宽度（像素）
const DefaultDoorWidth = 80

// 默认门长度（像素）
const DefaultDoorLength = 50

// 默认门颜色
const DefaultDoorColor = "#FF7043"

// 最大名称长度
const MaxDoorNameLength = 100

// 最大备注长度
const MaxDoorRemarkLength = 500

// DoorType 门类型枚举
type DoorType string

const (
	DoorTypeSingle    DoorType = "single"    // 单开门
	DoorTypeDouble    DoorType = "double"    // 双开门
	DoorTypeSliding   DoorType = "sliding"   // 推拉门
	DoorTypeRevolving DoorType = "revolving" // 旋转门
	DoorTypeEmergency DoorType = "emergency" // 紧急出口
)

// DoorDirection 门开启方向枚举
type DoorDirection string

const (
	DoorDirectionLeft    DoorDirection = "left"    // 左开
	DoorDirectionRight   DoorDirection = "right"   // 右开
	DoorDirectionDouble  DoorDirection = "double"  // 双向
	DoorDirectionSliding DoorDirection = "sliding" // 推拉
)

// Door 门模型 - 用于CAD风格平面图
type Door struct {
	models.BaseModel
	FloorID   string        `gorm:"type:uuid;not null" json:"floorId"`      // 所属楼层ID
	WallID    *string       `gorm:"type:uuid" json:"wallId,omitempty"`      // 关联墙体ID（可选）
	Position  string        `gorm:"type:jsonb;not null" json:"position"`    // 位置坐标 JSON: {"x":100,"y":100}
	Angle     int           `gorm:"default:0" json:"angle"`                 // 旋转角度（度）
	Type      DoorType      `gorm:"size:20;not null" json:"type"`           // 门类型
	Direction DoorDirection `gorm:"size:20;not null" json:"direction"`      // 开启方向
	Width     int           `gorm:"default:80" json:"width"`                // 门宽度（像素）
	Length    int           `gorm:"default:50" json:"length"`               // 门长度（像素）
	Color     string        `gorm:"size:20;default:'#FF7043'" json:"color"` // 门颜色（十六进制）
	Name      *string       `gorm:"size:100" json:"name,omitempty"`         // 门名称（可选）
	Remark    *string       `gorm:"size:500" json:"remark,omitempty"`       // 备注（可选）
}

// TableName 指定表名
func (Door) TableName() string {
	return "ops_doors"
}

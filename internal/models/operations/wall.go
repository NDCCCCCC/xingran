package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 常量定义

// 默认墙体厚度（像素）
const DefaultWallThickness = 10

// 默认墙体高度（米）
const DefaultWallHeight = 3.0

// 默认墙体颜色
const DefaultWallColor = "#5C6BC0"

// 最大名称长度
const MaxWallNameLength = 100

// 最大备注长度
const MaxWallRemarkLength = 500

// WallType 墙体类型枚举
type WallType string

const (
	WallTypeStraight WallType = "straight" // 直线墙
	WallTypeCurved   WallType = "curved"   // 弧形墙
	WallTypeLShaped  WallType = "l_shaped" // L型墙
)

// Wall 墙体模型 - 用于CAD风格平面图
type Wall struct {
	models.BaseModel
	FloorID   string   `gorm:"type:uuid;not null" json:"floorId"`      // 所属楼层ID
	Type      WallType `gorm:"size:20;not null" json:"type"`           // 墙体类型
	Points    string   `gorm:"type:jsonb;not null" json:"points"`      // 路径点 JSON 数组: [{"x":100,"y":100},{"x":200,"y":200}]
	Thickness int      `gorm:"default:10" json:"thickness"`            // 墙体厚度（像素）
	Height    float64  `gorm:"default:3.0" json:"height"`              // 墙体高度（米）
	Color     string   `gorm:"size:20;default:'#5C6BC0'" json:"color"` // 墙体颜色（十六进制）
	Name      *string  `gorm:"size:100" json:"name,omitempty"`         // 墙体名称（可选）
	Remark    *string  `gorm:"size:500" json:"remark,omitempty"`       // 备注（可选）
}

// TableName 指定表名
func (Wall) TableName() string {
	return "ops_walls"
}

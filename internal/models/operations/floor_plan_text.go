package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// 常量定义

// 默认字体大小
const DefaultFontSize = 14

// 默认文本颜色
const DefaultTextColor = "#333333"

// 默认字体族
const DefaultFontFamily = "Arial, sans-serif"

// 默认字体粗细
const DefaultFontWeight = "normal"

// 默认字体样式
const DefaultFontStyle = "normal"

// 最大内容长度
const MaxTextContentLength = 500

// 最大备注长度
const MaxTextRemarkLength = 500

// 最大字体族长度
const MaxFontFamilyLength = 100

// 最大字体粗细长度
const MaxFontWeightLength = 20

// 最大字体样式长度
const MaxFontStyleLength = 20

// 最大颜色长度
const MaxColorLength = 20

// FloorPlanText 平面图文本模型 - 用于存储CAD风格平面图中的文本标注
type FloorPlanText struct {
	models.BaseModel
	FloorID    string  `gorm:"type:uuid;not null" json:"floorId"`                      // 所属楼层ID
	Position   string  `gorm:"type:jsonb;not null" json:"position"`                    // 位置坐标 JSON: {"x":100,"y":100}
	Content    string  `gorm:"size:500;not null" json:"content"`                       // 文本内容
	FontSize   int     `gorm:"default:14" json:"fontSize"`                             // 字体大小（像素）
	Color      string  `gorm:"size:20;default:'#333333'" json:"color"`                 // 文本颜色（十六进制）
	FontFamily string  `gorm:"size:100;default:'Arial, sans-serif'" json:"fontFamily"` // 字体族
	FontWeight string  `gorm:"size:20;default:'normal'" json:"fontWeight"`             // 字体粗细（normal/bold）
	FontStyle  string  `gorm:"size:20;default:'normal'" json:"fontStyle"`              // 字体样式（normal/italic）
	Angle      int     `gorm:"default:0" json:"angle"`                                 // 旋转角度（度）
	Remark     *string `gorm:"size:500" json:"remark,omitempty"`                       // 备注（可选）
}

// TableName 指定表名
func (FloorPlanText) TableName() string {
	return "ops_floor_plan_texts"
}

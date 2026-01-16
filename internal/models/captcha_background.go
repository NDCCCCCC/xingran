// Package models 验证码背景图数据模型
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// CaptchaBackgroundStatus 背景图状态
type CaptchaBackgroundStatus int

const (
	CaptchaBgDisabled CaptchaBackgroundStatus = 0 // 禁用
	CaptchaBgEnabled  CaptchaBackgroundStatus = 1 // 启用
)

// String 实现 Stringer 接口
func (s CaptchaBackgroundStatus) String() string {
	switch s {
	case CaptchaBgEnabled:
		return "启用"
	case CaptchaBgDisabled:
		return "禁用"
	default:
		return "未知"
	}
}

// PieceShape 拼图形状类型
type PieceShape string

const (
	PieceShapeCircle PieceShape = "circle" // 圆形
	PieceShapeSquare PieceShape = "square" // 方形
	PieceShapeStar   PieceShape = "star"   // 星形
	PieceShapeHeart  PieceShape = "heart"  // 心形
)

// AllPieceShapes 所有支持的拼图形状
var AllPieceShapes = []PieceShape{
	PieceShapeCircle,
	PieceShapeSquare,
	PieceShapeStar,
	PieceShapeHeart,
}

// DifficultyLevel 难度级别
type DifficultyLevel int

const (
	DifficultyEasy   DifficultyLevel = 1 // 简单
	DifficultyMedium DifficultyLevel = 2 // 中等
	DifficultyHard   DifficultyLevel = 3 // 困难
)

// String 实现 Stringer 接口
func (d DifficultyLevel) String() string {
	switch d {
	case DifficultyEasy:
		return "简单"
	case DifficultyMedium:
		return "中等"
	case DifficultyHard:
		return "困难"
	default:
		return "未知"
	}
}

// CaptchaBackground 验证码背景图模型
type CaptchaBackground struct {
	ID        string     `gorm:"size:36;primaryKey" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `gorm:"index" json:"deletedAt,omitempty"`
	CreatedBy string     `gorm:"size:64" json:"createdBy"`
	UpdatedBy string     `gorm:"size:64" json:"updatedBy"`
	Version   int        `json:"version"`

	// 基本信息
	FileName   string `gorm:"size:255;not null" json:"fileName"`
	FilePath   string `gorm:"size:500;not null" json:"filePath"`
	FileSize   int64  `json:"fileSize"`
	FileWidth  int    `json:"fileWidth"`
	FileHeight int    `json:"fileHeight"`
	FileMD5    string `gorm:"size:32;index" json:"fileMd5,omitempty"`

	// 验证码配置
	PieceShape      PieceShape      `gorm:"size:20;not null;default:'circle'" json:"pieceShape"`
	DifficultyLevel DifficultyLevel `gorm:"not null;default:1" json:"difficultyLevel"`
	AllowedShapes   StringArray     `gorm:"type:jsonb" json:"allowedShapes"`

	// 使用统计
	UseCount   int        `gorm:"default:0" json:"useCount"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	SortOrder  int        `gorm:"default:0" json:"sortOrder"`

	// 状态管理
	Status CaptchaBackgroundStatus `gorm:"not null;default:1" json:"status"`
	Remark string                  `gorm:"size:500" json:"remark,omitempty"`
}

// StringArray 字符串数组类型（用于存储JSON数组）
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}

// Value 实现 driver.Valuer 接口
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Contains 检查是否包含指定字符串
func (s StringArray) Contains(str string) bool {
	for _, item := range s {
		if item == str {
			return true
		}
	}
	return false
}

// TableName 指定表名
func (CaptchaBackground) TableName() string {
	return "sys_captcha_background"
}

// BeforeCreate GORM 创建前钩子
func (cb *CaptchaBackground) BeforeCreate(tx *gorm.DB) error {
	if cb.ID == "" {
		cb.ID = generateUUID()
	}
	return nil
}

// GetAllowedShapesList 获取允许的形状列表（兼容旧代码）
func (cb *CaptchaBackground) GetAllowedShapesList() []string {
	if len(cb.AllowedShapes) == 0 {
		return []string{"circle", "square", "star", "heart"}
	}
	return cb.AllowedShapes
}

// SetAllowedShapesList 设置允许的形状列表（兼容旧代码）
func (cb *CaptchaBackground) SetAllowedShapesList(shapes []string) {
	cb.AllowedShapes = shapes
}

// IsEnabled 检查是否启用
func (cb *CaptchaBackground) IsEnabled() bool {
	return cb.Status == CaptchaBgEnabled
}

// GetFileURL 获取文件访问URL
func (cb *CaptchaBackground) GetFileURL() string {
	return "/uploads/captcha/backgrounds/" + cb.FileName
}

// ToDTO 转换为DTO对象
func (cb *CaptchaBackground) ToDTO() *CaptchaBackgroundDTO {
	return &CaptchaBackgroundDTO{
		ID:              cb.ID,
		FileName:        cb.FileName,
		FilePath:        cb.FilePath,
		FileSize:        cb.FileSize,
		FileWidth:       cb.FileWidth,
		FileHeight:      cb.FileHeight,
		PieceShape:      string(cb.PieceShape),
		DifficultyLevel: int(cb.DifficultyLevel),
		AllowedShapes:   cb.AllowedShapes,
		UseCount:        int64(cb.UseCount),
		LastUsedAt:      cb.LastUsedAt,
		SortOrder:       cb.SortOrder,
		Status:          int(cb.Status),
		Remark:          cb.Remark,
		CreatedAt:       cb.CreatedAt,
		UpdatedAt:       cb.UpdatedAt,
		PreviewURL:      cb.GetFileURL(),
	}
}

// CaptchaBackgroundDTO 背景图数据传输对象
type CaptchaBackgroundDTO struct {
	ID              string     `json:"id"`
	FileName        string     `json:"fileName"`
	FilePath        string     `json:"filePath"`
	FileSize        int64      `json:"fileSize"`
	FileWidth       int        `json:"fileWidth"`
	FileHeight      int        `json:"fileHeight"`
	PieceShape      string     `json:"pieceShape"`
	DifficultyLevel int        `json:"difficultyLevel"`
	AllowedShapes   []string   `json:"allowedShapes"`
	UseCount        int64      `json:"useCount"`
	LastUsedAt      *time.Time `json:"lastUsedAt,omitempty"`
	SortOrder       int        `json:"sortOrder"`
	Status          int        `json:"status"`
	Remark          string     `json:"remark,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	PreviewURL      string     `json:"previewUrl"`
}

// CaptchaBackgroundListRequest 背景图列表请求
type CaptchaBackgroundListRequest struct {
	FileName        *string `json:"fileName,omitempty"`
	PieceShape      *string `json:"pieceShape,omitempty"`
	DifficultyLevel *int    `json:"difficultyLevel,omitempty"`
	Status          *int    `json:"status,omitempty"`
	Current         int     `json:"current" binding:"min=1"`
	PageSize        int     `json:"pageSize" binding:"min=1,max=100"`
}

// CaptchaBackgroundListResponse 背景图列表响应
type CaptchaBackgroundListResponse struct {
	Total int64                   `json:"total"`
	Items []*CaptchaBackgroundDTO `json:"items"`
}

// CaptchaBackgroundUploadRequest 上传请求
type CaptchaBackgroundUploadRequest struct {
	FileName        string     `json:"fileName"`
	PieceShape      PieceShape `json:"pieceShape" binding:"required,oneof=circle square star heart"`
	DifficultyLevel int        `json:"difficultyLevel" binding:"required,min=1,max=3"`
	AllowedShapes   []string   `json:"allowedShapes"`
	Remark          string     `json:"remark"`
}

// CaptchaBackgroundUpdateRequest 更新请求
type CaptchaBackgroundUpdateRequest struct {
	PieceShape      *PieceShape `json:"pieceShape,omitempty"`
	DifficultyLevel *int        `json:"difficultyLevel,omitempty"`
	AllowedShapes   *[]string   `json:"allowedShapes,omitempty"`
	Status          *int        `json:"status,omitempty"`
	SortOrder       *int        `json:"sortOrder,omitempty"`
	Remark          *string     `json:"remark,omitempty"`
}

// StatisticsResponse 统计信息响应
type StatisticsResponse struct {
	TotalCount        int            `json:"totalCount"`
	EnabledCount      int            `json:"enabledCount"`
	DisabledCount     int            `json:"disabledCount"`
	ShapeDistribution map[string]int `json:"shapeDistribution"`
	DifficultyDist    map[int]int    `json:"difficultyDistribution"`
	TotalUsage        int64          `json:"totalUsage"`
}

// 辅助函数

// generateUUID 生成UUID（简化版本）
func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ParseStringArray 从 pq.StringArray 解析
func ParseStringArray(arr pq.StringArray) []string {
	if len(arr) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(arr))
	for _, s := range arr {
		if s != "" {
			// 移除引号
			trimmed := strings.Trim(s, `"`)
			result = append(result, trimmed)
		}
	}
	return result
}

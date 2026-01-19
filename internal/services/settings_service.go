package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SettingsService 系统设置服务
type SettingsService struct {
	db *gorm.DB
}

// NewSettingsService 创建系统设置 service
func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

// UserPreference 用户个人设置模型（扩展版）
type UserPreference struct {
	ID               string    `gorm:"primaryKey;type:uuid" json:"id"`
	UserID           string    `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`

	// 主题配置
	Theme            string    `gorm:"size:20;default:light" json:"theme"`
	ThemeStyle       string    `gorm:"size:20;default:minimal" json:"themeStyle"`

	// 布局配置
	LayoutType       string    `gorm:"size:20;default:classic" json:"layoutType"`
	LayoutDensity    string    `gorm:"size:20;default:comfortable" json:"layoutDensity"`
	SidebarWidth     int       `gorm:"default:240" json:"sidebarWidth"`
	SidebarCollapsedWidth int  `gorm:"default:64" json:"sidebarCollapsedWidth"`
	SidebarCollapsed bool      `gorm:"default:false" json:"sidebarCollapsed"`

	// 数据配置
	PageSize         int       `gorm:"default:10" json:"pageSize"`

	// 语言
	Language         string    `gorm:"size:10;default:zh-CN" json:"language"`

	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// TableName 指定表名
func (UserPreference) TableName() string {
	return "sys_user_preference"
}

// BeforeCreate GORM hook - 创建前生成UUID
func (up *UserPreference) BeforeCreate(tx *gorm.DB) error {
	if up.ID == "" {
		up.ID = uuid.New().String()
	}
	return nil
}

// UserPreferencesDTO 用户设置 DTO（扩展版）
type UserPreferencesDTO struct {
	// 主题
	Theme            string `json:"theme" binding:"required,oneof=light dark"`
	ThemeStyle       string `json:"themeStyle" binding:"omitempty,oneof=minimal glassmorphism neumorphism flat2.0 luxury-quiet"`

	// 布局
	LayoutType       string `json:"layoutType" binding:"omitempty,oneof=classic hybrid innovative"`
	LayoutDensity    string `json:"layoutDensity" binding:"omitempty,oneof=compact comfortable spacious"`
	SidebarWidth     int    `json:"sidebarWidth" binding:"omitempty,min=100,max=400"`
	SidebarCollapsedWidth int `json:"sidebarCollapsedWidth" binding:"omitempty,min=40,max=100"`
	SidebarCollapsed bool   `json:"sidebarCollapsed"`

	// 数据
	PageSize         int    `json:"pageSize" binding:"required,min=5,max=100"`

	// 语言
	Language         string `json:"language" binding:"required,oneof=zh-CN en-US"`
}

// GetUserPreferences 获取用户个人设置
func (s *SettingsService) GetUserPreferences(ctx context.Context, userID string) (*UserPreferencesDTO, error) {
	var pref UserPreference
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error

	// 如果不存在，返回默认值
	if err == gorm.ErrRecordNotFound {
		return &UserPreferencesDTO{
			Theme:                  "light",
			ThemeStyle:             "minimal",
			LayoutType:             "classic",
			LayoutDensity:          "comfortable",
			SidebarWidth:           240,
			SidebarCollapsedWidth:  64,
			SidebarCollapsed:       false,
			PageSize:               10,
			Language:               "zh-CN",
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("查询用户设置失败: %w", err)
	}

	return &UserPreferencesDTO{
		Theme:                  pref.Theme,
		ThemeStyle:             pref.ThemeStyle,
		LayoutType:             pref.LayoutType,
		LayoutDensity:          pref.LayoutDensity,
		SidebarWidth:           pref.SidebarWidth,
		SidebarCollapsedWidth:  pref.SidebarCollapsedWidth,
		SidebarCollapsed:       pref.SidebarCollapsed,
		PageSize:               pref.PageSize,
		Language:               pref.Language,
	}, nil
}

// UpdateUserPreferences 更新用户个人设置
func (s *SettingsService) UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferencesDTO) error {
	var pref UserPreference

	// 查找现有记录
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		pref = UserPreference{
			UserID:                userID,
			Theme:                 req.Theme,
			ThemeStyle:            req.ThemeStyle,
			LayoutType:            req.LayoutType,
			LayoutDensity:         req.LayoutDensity,
			SidebarWidth:          req.SidebarWidth,
			SidebarCollapsedWidth: req.SidebarCollapsedWidth,
			SidebarCollapsed:      req.SidebarCollapsed,
			PageSize:              req.PageSize,
			Language:              req.Language,
		}
		if createErr := s.db.WithContext(ctx).Create(&pref).Error; createErr != nil {
			return fmt.Errorf("创建用户设置失败: %w", createErr)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询用户设置失败: %w", err)
	}

	// 更新现有记录 - 使用 Updates 更新所有字段
	if updateErr := s.db.WithContext(ctx).Model(&pref).Updates(map[string]interface{}{
		"theme":                    req.Theme,
		"theme_style":              req.ThemeStyle,
		"layout_type":              req.LayoutType,
		"layout_density":           req.LayoutDensity,
		"sidebar_width":            req.SidebarWidth,
		"sidebar_collapsed_width":  req.SidebarCollapsedWidth,
		"sidebar_collapsed":        req.SidebarCollapsed,
		"page_size":                req.PageSize,
		"language":                 req.Language,
	}).Error; updateErr != nil {
		return fmt.Errorf("更新用户设置失败: %w", updateErr)
	}

	return nil
}

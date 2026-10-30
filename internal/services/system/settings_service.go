package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// SettingsService 系统设置服务
type SettingsService interface {
	// GetUserPreferences 获取用户个人设置
	GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error)

	// UpdateUserPreferences 更新用户个人设置
	UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferences) error
}

// settingsService 系统设置服务实现
//
// v1.22 收尾：移除对 sys.theme.default (默认主题) 的依赖（删除默认主题页面+API+表行后），
// 用户无偏好记录时直接返回硬编码默认值，不再注入 ConfigService。
type settingsService struct {
	db *gorm.DB
}

// NewSettingsService 创建系统设置服务实例
func NewSettingsService(db *gorm.DB) SettingsService {
	return &settingsService{db: db}
}

// UserPreference 用户个人设置模型（扩展版）
//
// 2026-08-17 (quick-260817-hfl): 模型本体迁移至 internal/models/user_preference.go
// (schema 单一事实源,sqlite 分支 AutoMigrate 注册需要;此处 alias 保持全部调用点兼容)。
type UserPreference = models.UserPreference

// UserPreferences 用户个人设置请求/响应DTO
type UserPreferences struct {
	// 主题
	Theme      string `json:"theme" binding:"required,oneof=light dark"`
	ThemeStyle string `json:"themeStyle" binding:"omitempty,oneof=minimal glassmorphism neumorphism flat2.0 luxury-quiet ink-amber"`

	// 布局
	LayoutType            string `json:"layoutType" binding:"omitempty,oneof=classic hybrid innovative"`
	LayoutDensity         string `json:"layoutDensity" binding:"omitempty,oneof=compact comfortable spacious"`
	SidebarWidth          int    `json:"sidebarWidth" binding:"omitempty,min=100,max=400"`
	SidebarCollapsedWidth int    `json:"sidebarCollapsedWidth" binding:"omitempty,min=40,max=100"`
	SidebarCollapsed      bool   `json:"sidebarCollapsed"`

	// 数据
	PageSize int `json:"pageSize" binding:"required,min=5,max=100"`

	// 自定义颜色（可选）
	CustomPrimaryColor string `json:"customPrimaryColor,omitempty" binding:"omitempty,eq=|len=7|len=4"`
	CustomSidebarColor string `json:"customSidebarColor,omitempty" binding:"omitempty,eq=|len=7|len=4"`

	// 语言
	Language string `json:"language" binding:"required,oneof=zh-CN en-US"`
}

// GetUserPreferences 获取用户个人设置
func (s *settingsService) GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	var pref UserPreference
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error

	// 如果不存在，返回硬编码默认值（v1.22 收尾：不再合并 sys.theme.default）
	if err == gorm.ErrRecordNotFound {
		return s.buildDefaultPreferences(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("查询用户设置失败: %w", err)
	}

	// 处理零值，使用默认值
	sidebarWidth := pref.SidebarWidth
	if sidebarWidth == 0 {
		sidebarWidth = 280
	}
	sidebarCollapsedWidth := pref.SidebarCollapsedWidth
	if sidebarCollapsedWidth == 0 {
		sidebarCollapsedWidth = 64
	}

	return &UserPreferences{
		Theme:                 pref.Theme,
		ThemeStyle:            pref.ThemeStyle,
		LayoutType:            pref.LayoutType,
		LayoutDensity:         pref.LayoutDensity,
		SidebarWidth:          sidebarWidth,
		SidebarCollapsedWidth: sidebarCollapsedWidth,
		SidebarCollapsed:      pref.SidebarCollapsed,
		PageSize:              pref.PageSize,
		CustomPrimaryColor:    pref.CustomPrimaryColor,
		CustomSidebarColor:    pref.CustomSidebarColor,
		Language:              pref.Language,
	}, nil
}

// buildDefaultPreferences 构建用户无偏好记录时的硬编码默认值
//
// v1.22 收尾：不再合并 sys.theme.default（已删除默认主题页面）。
// 行为变化：之前若管理员在 sys.theme.default 配置了 mode=dark，
//          新用户的默认 Theme 会被设为 dark；现在统一回到 light。
// 后续如需"管理员配置全局默认值"能力，应以独立的 sys.user_preferences_default 表
// 或 sys_config 通用键实现，不再耦合到 sys.theme.default 这一被删除的主题语义。
func (s *settingsService) buildDefaultPreferences() *UserPreferences {
	return &UserPreferences{
		Theme:                 "light",
		ThemeStyle:            "minimal",
		LayoutType:            "classic",
		LayoutDensity:         "comfortable",
		SidebarWidth:          280,
		SidebarCollapsedWidth: 64,
		SidebarCollapsed:      false,
		PageSize:              10,
		Language:              "zh-CN",
	}
}

// UpdateUserPreferences 更新用户个人设置
func (s *settingsService) UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferences) error {
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
			CustomPrimaryColor:    req.CustomPrimaryColor,
			CustomSidebarColor:    req.CustomSidebarColor,
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
		"theme":                   req.Theme,
		"theme_style":             req.ThemeStyle,
		"layout_type":             req.LayoutType,
		"layout_density":          req.LayoutDensity,
		"sidebar_width":           req.SidebarWidth,
		"sidebar_collapsed_width": req.SidebarCollapsedWidth,
		"sidebar_collapsed":       req.SidebarCollapsed,
		"page_size":               req.PageSize,
		"custom_primary_color":    req.CustomPrimaryColor,
		"custom_sidebar_color":    req.CustomSidebarColor,
		"language":                req.Language,
	}).Error; updateErr != nil {
		return fmt.Errorf("更新用户设置失败: %w", updateErr)
	}

	return nil
}
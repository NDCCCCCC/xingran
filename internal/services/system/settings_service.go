package system

import (
	"context"
	"encoding/json"
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
type settingsService struct {
	db            *gorm.DB
	configService ConfigService
}

// NewSettingsService 创建系统设置服务实例
func NewSettingsService(db *gorm.DB, configService ConfigService) SettingsService {
	return &settingsService{db: db, configService: configService}
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

	// 如果不存在，返回默认值（优先合并管理员配置的默认主题）
	if err == gorm.ErrRecordNotFound {
		return s.buildDefaultPreferences(ctx), nil
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

// buildDefaultPreferences 构建用户无偏好记录时的默认值
// 优先合并管理员在 sys.theme.default 中配置的默认主题；管理员未配置时回退到硬编码值。
func (s *settingsService) buildDefaultPreferences(ctx context.Context) *UserPreferences {
	prefs := &UserPreferences{
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

	// 尝试合并管理员配置的默认主题
	if s.configService == nil {
		return prefs
	}

	config, err := s.configService.GetByKey(ctx, "sys.theme.default")
	if err != nil || config == nil {
		// 配置不存在或读取失败，保持硬编码默认值
		return prefs
	}

	var themeCfg ThemeConfiguration
	if err := json.Unmarshal([]byte(config.ConfigValue), &themeCfg); err != nil {
		// 解析失败，保持硬编码默认值
		return prefs
	}

	if themeCfg.Mode != "" {
		prefs.Theme = themeCfg.Mode
	}
	if themeCfg.Style != "" {
		prefs.ThemeStyle = themeCfg.Style
	}
	if primary, ok := themeCfg.CustomColors["primary"]; ok && primary != "" {
		prefs.CustomPrimaryColor = primary
	}
	if sidebar, ok := themeCfg.CustomColors["sidebar"]; ok && sidebar != "" {
		prefs.CustomSidebarColor = sidebar
	}

	return prefs
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

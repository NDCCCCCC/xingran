package system

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"gorm.io/gorm"
)

// ThemeConfiguration 主题配置结构
type ThemeConfiguration struct {
	Mode         string         `json:"mode"`           // light, dark, auto
	Style        string         `json:"style"`          // minimal, glassmorphism, neumorphism, flat2.0, luxury-quiet
	CustomColors map[string]string `json:"customColors,omitempty"` // 可选的自定义颜色
}

// DefaultThemeService 默认主题服务接口
type DefaultThemeService interface {
	// GetDefaultThemeConfig 获取默认主题配置
	GetDefaultThemeConfig(ctx context.Context) (*ThemeConfiguration, error)

	// SetDefaultThemeConfig 设置默认主题配置
	SetDefaultThemeConfig(ctx context.Context, config *ThemeConfiguration) error

	// SyncUserThemeToDefault 从用户配置同步到默认主题
	SyncUserThemeToDefault(ctx context.Context, userID string) error
}

// defaultThemeService 默认主题服务实现
type defaultThemeService struct {
	db          *gorm.DB
	configService ConfigService
}

// NewDefaultThemeService 创建默认主题服务实例
func NewDefaultThemeService(db *gorm.DB, configService ConfigService) DefaultThemeService {
	return &defaultThemeService{
		db:           db,
		configService: configService,
	}
}

// GetDefaultThemeConfig 获取默认主题配置
func (s *defaultThemeService) GetDefaultThemeConfig(ctx context.Context) (*ThemeConfiguration, error) {
	// 从 sys_config 表获取配置
	config, err := s.configService.GetByKey(ctx, "sys.theme.default")
	if err != nil {
		// 任何错误都返回默认配置（无论是记录不存在还是其他错误）
		return &ThemeConfiguration{
			Mode:  "light",
			Style: "minimal",
		}, nil
	}

	// 解析 JSON 配置
	var themeConfig ThemeConfiguration
	if err := json.Unmarshal([]byte(config.ConfigValue), &themeConfig); err != nil {
		// 如果解析失败，返回默认配置
		return &ThemeConfiguration{
			Mode:  "light",
			Style: "minimal",
		}, nil
	}

	return &themeConfig, nil
}

// SetDefaultThemeConfig 设置默认主题配置
func (s *defaultThemeService) SetDefaultThemeConfig(ctx context.Context, config *ThemeConfiguration) error {
	// 验证配置
	if config.Mode == "" {
		config.Mode = "light"
	}
	if config.Style == "" {
		config.Style = "minimal"
	}

	// 验证模式值
	validModes := map[string]bool{"light": true, "dark": true, "auto": true}
	if !validModes[config.Mode] {
		return fmt.Errorf("无效的主题模式: %s", config.Mode)
	}

	// 验证风格值
	validStyles := map[string]bool{
		"minimal": true,
		"glassmorphism": true,
		"neumorphism": true,
		"flat2.0": true,
		"luxury-quiet": true,
	}
	if !validStyles[config.Style] {
		return fmt.Errorf("无效的主题风格: %s", config.Style)
	}

	// 序列化为 JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 检查配置是否存在
	existingConfig, err := s.configService.GetByKey(ctx, "sys.theme.default")
	// GetByKey 返回包装后的错误，无法直接判断 gorm.ErrRecordNotFound
	// 如果返回 nil config 说明配置不存在，进入创建分支
	if err != nil && existingConfig == nil {
		// 配置不存在，继续走创建流程
		existingConfig = nil
	} else if err != nil {
		return fmt.Errorf("查询现有配置失败: %w", err)
	}

	if existingConfig != nil {
		// 更新现有配置
		updateReq := &requests.ConfigUpdateRequest{
			ID:          existingConfig.ID,
			ConfigName:  "默认主题配置",
			ConfigKey:   "sys.theme.default",
			ConfigValue: string(configJSON),
			ConfigType:  "Y", // Y=yes/system built-in
		}
		if err := s.configService.Update(ctx, updateReq); err != nil {
			return fmt.Errorf("更新默认主题配置失败: %w", err)
		}
	} else {
		// 创建新配置
		remark := "系统默认主题配置，用于新用户和主题同步"
		isSystem := 1
		createReq := &requests.ConfigCreateRequest{
			ConfigName:  "默认主题配置",
			ConfigKey:   "sys.theme.default",
			ConfigValue: string(configJSON),
			ConfigType:  "Y", // Y=normal/N=notice or other type
			IsSystem:    isSystem, // 系统内置
			Remark:      &remark,
		}
		if err := s.configService.Create(ctx, createReq); err != nil {
			return fmt.Errorf("创建默认主题配置失败: %w", err)
		}
	}

	return nil
}

// SyncUserThemeToDefault 从用户配置同步到默认主题
func (s *defaultThemeService) SyncUserThemeToDefault(ctx context.Context, userID string) error {
	// 获取用户配置
	var userPreferences struct {
		Theme      string `json:"theme"`
		ThemeStyle string `json:"themeStyle"`
	}

	// 从用户偏好表获取配置
	err := s.db.WithContext(ctx).
		Table("sys_user_preferences").
		Select("theme, theme_style").
		Where("user_id = ?", userID).
		First(&userPreferences).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户配置不存在: %s", userID)
		}
		return fmt.Errorf("获取用户配置失败: %w", err)
	}

	// 构建主题配置
	themeConfig := &ThemeConfiguration{
		Mode:  userPreferences.Theme,
		Style: userPreferences.ThemeStyle,
	}

	// 设置为默认主题
	return s.SetDefaultThemeConfig(ctx, themeConfig)
}

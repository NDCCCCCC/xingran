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

// UserPreference 用户个人设置模型
type UserPreference struct {
	ID               string    `gorm:"primaryKey;type:uuid" json:"id"`
	UserID           string    `gorm:"type:uuid;not null;uniqueIndex" json:"userId"`
	Theme            string    `gorm:"size:20;default:light" json:"theme"`
	Language         string    `gorm:"size:10;default:zh-CN" json:"language"`
	PageSize         int       `gorm:"default:10" json:"pageSize"`
	SidebarCollapsed bool      `gorm:"default:false" json:"sidebarCollapsed"`
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

// UserPreferencesDTO 用户设置 DTO
type UserPreferencesDTO struct {
	Theme            string `json:"theme"`
	Language         string `json:"language"`
	PageSize         int    `json:"pageSize"`
	SidebarCollapsed bool   `json:"sidebarCollapsed"`
}

// GetUserPreferences 获取用户个人设置
func (s *SettingsService) GetUserPreferences(ctx context.Context, userID string) (*UserPreferencesDTO, error) {
	var pref UserPreference
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error

	// 如果不存在，返回默认值
	if err == gorm.ErrRecordNotFound {
		return &UserPreferencesDTO{
			Theme:            "light",
			Language:         "zh-CN",
			PageSize:         10,
			SidebarCollapsed: false,
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("查询用户设置失败: %w", err)
	}

	return &UserPreferencesDTO{
		Theme:            pref.Theme,
		Language:         pref.Language,
		PageSize:         pref.PageSize,
		SidebarCollapsed: pref.SidebarCollapsed,
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
			UserID:           userID,
			Theme:            req.Theme,
			Language:         req.Language,
			PageSize:         req.PageSize,
			SidebarCollapsed: req.SidebarCollapsed,
		}
		if createErr := s.db.WithContext(ctx).Create(&pref).Error; createErr != nil {
			return fmt.Errorf("创建用户设置失败: %w", createErr)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询用户设置失败: %w", err)
	}

	// 更新现有记录
	if updateErr := s.db.WithContext(ctx).Model(&pref).Updates(map[string]interface{}{
		"theme":             req.Theme,
		"language":          req.Language,
		"page_size":         req.PageSize,
		"sidebar_collapsed": req.SidebarCollapsed,
	}).Error; updateErr != nil {
		return fmt.Errorf("更新用户设置失败: %w", updateErr)
	}

	return nil
}

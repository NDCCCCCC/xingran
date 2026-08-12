package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// DutyConfigService 值班配置管理服务
type DutyConfigService struct {
	db *gorm.DB
}

// NewDutyConfigService 创建值班配置管理服务
func NewDutyConfigService(db *gorm.DB) *DutyConfigService {
	return &DutyConfigService{db: db}
}

// GetDutyConfig 获取值班配置（系统中只有一条配置记录）
func (s *DutyConfigService) GetDutyConfig(ctx context.Context) (*models.DutyConfig, error) {
	var config models.DutyConfig
	err := s.db.WithContext(ctx).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果不存在，返回默认配置
			return &models.DutyConfig{
				ReminderEnabled:       true,
				ReminderTime:          "08:00",
				ReminderChannels:      "websocket",
				BeforeReminderMinutes: nil,
			}, nil
		}
		return nil, fmt.Errorf("查询值班配置失败: %w", err)
	}
	return &config, nil
}

// UpdateDutyConfig 更新值班配置
func (s *DutyConfigService) UpdateDutyConfig(ctx context.Context, config *models.DutyConfig, updaterID string) error {
	// 检查是否已存在配置
	var existing models.DutyConfig
	err := s.db.WithContext(ctx).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在，创建新配置
		config.CreatedBy = updaterID
		config.UpdatedBy = updaterID
		if createErr := s.db.WithContext(ctx).Create(config).Error; createErr != nil {
			return fmt.Errorf("创建值班配置失败: %w", createErr)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询值班配置失败: %w", err)
	}

	// 更新现有配置
	config.ID = existing.ID
	config.CreatedBy = existing.CreatedBy
	config.UpdatedBy = updaterID
	if err := s.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("更新值班配置失败: %w", err)
	}

	return nil
}

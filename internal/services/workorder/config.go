package workorder

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ConfigService 配置服务
type ConfigService struct {
	db *gorm.DB
}

// NewConfigService 创建配置服务
func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{db: db}
}

// Get 获取工单配置
func (s *ConfigService) Get(ctx context.Context) (*models.WorkOrderConfig, error) {
	var config models.WorkOrderConfig

	// 尝试从数据库获取配置
	err := s.db.WithContext(ctx).Where("id = ?", "default").First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认配置
			config = models.WorkOrderConfig{
				ID:                      "default",
				AutoAssignEnabled:       true,
				AutoAssignTarget:        "duty_pool",
				AutoAssignStrategy:      "assign_one",
				AutoCloseDays:           7,
				AllowUserClose:          false,
				NotificationEnabled:     true,
				EmailNotification:       false,
				SmsNotification:         false,
				RatingEnabled:           true,
				KnowledgeConvertEnabled: true,
			}
			s.db.WithContext(ctx).Create(&config)
		} else {
			return nil, fmt.Errorf("查询工单配置失败: %w", err)
		}
	}

	return &config, nil
}

// Update 更新工单配置
func (s *ConfigService) Update(ctx context.Context, config *models.WorkOrderConfig) error {
	config.ID = "default"

	// 使用 GORM 的 OnConflict 来处理 upsert
	if err := s.db.WithContext(ctx).
		Where("id = ?", "default").
		Assign(config).
		FirstOrCreate(&config).Error; err != nil {
		return fmt.Errorf("更新工单配置失败: %w", err)
	}

	return nil
}

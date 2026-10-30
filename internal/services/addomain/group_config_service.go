package addomain

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// GroupConfigService 组配置服务
type GroupConfigService struct {
	db *gorm.DB
}

// NewGroupConfigService 创建组配置服务
func NewGroupConfigService(db *gorm.DB) *GroupConfigService {
	return &GroupConfigService{
		db: db,
	}
}

// getConfigByKey 通过key获取配置值
func (s *GroupConfigService) getConfigByKey(ctx context.Context, configKey string) (string, error) {
	var config models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", configKey).First(&config).Error
	if err != nil {
		return "", err
	}
	return config.ConfigValue, nil
}

// setConfigByKey 通过key设置配置值
func (s *GroupConfigService) setConfigByKey(ctx context.Context, configKey string, configValue string) error {
	var config models.Config
	err := s.db.WithContext(ctx).Where("config_key = ?", configKey).First(&config).Error
	if err != nil {
		// 如果配置不存在，创建新配置
		if err == gorm.ErrRecordNotFound {
			config = models.Config{
				ConfigKey:   configKey,
				ConfigValue: configValue,
				ConfigType:  "string",
				IsSystem:    models.ConfigIsSystemYes,
			}
			return s.db.WithContext(ctx).Create(&config).Error
		}
		return err
	}

	// 更新现有配置
	return s.db.WithContext(ctx).Model(&config).Update("config_value", configValue).Error
}

// GetGroupSyncConfig 获取组同步配置
func (s *GroupConfigService) GetGroupSyncConfig(ctx context.Context) (*GroupSyncConfig, error) {
	config := GetDefaultGroupSyncConfig()

	// 读取enabled
	enabledStr, err := s.getConfigByKey(ctx, ConfigGroupSyncEnabled)
	if err == nil && enabledStr != "" {
		config.Enabled = enabledStr == "true" || enabledStr == "1"
	}

	// 读取cron
	cronStr, err := s.getConfigByKey(ctx, ConfigGroupSyncCron)
	if err == nil && cronStr != "" {
		config.Cron = cronStr
	}

	// 读取member_ou
	memberOU, err := s.getConfigByKey(ctx, ConfigGroupMemberOU)
	if err == nil && memberOU != "" {
		config.MemberOU = memberOU
	}

	// 读取auto_create
	autoCreateStr, err := s.getConfigByKey(ctx, ConfigGroupAutoCreate)
	if err == nil && autoCreateStr != "" {
		config.AutoCreateGroups = autoCreateStr == "true" || autoCreateStr == "1"
	}

	// 读取max_concurrent
	maxConcurrentStr, err := s.getConfigByKey(ctx, ConfigGroupMaxConcurrent)
	if err == nil && maxConcurrentStr != "" {
		if val, err := strconv.Atoi(maxConcurrentStr); err == nil {
			config.MaxConcurrent = val
		}
	}

	// 读取batch_size
	batchSizeStr, err := s.getConfigByKey(ctx, ConfigGroupSyncBatchSize)
	if err == nil && batchSizeStr != "" {
		if val, err := strconv.Atoi(batchSizeStr); err == nil {
			config.SyncBatchSize = val
		}
	}

	return config, nil
}

// IsGroupSyncEnabled 检查组同步是否启用
func (s *GroupConfigService) IsGroupSyncEnabled(ctx context.Context) bool {
	config, err := s.GetGroupSyncConfig(ctx)
	if err != nil {
		return false
	}
	return config.Enabled
}

// UpdateGroupSyncConfig 更新组同步配置
func (s *GroupConfigService) UpdateGroupSyncConfig(ctx context.Context, config *GroupSyncConfig) error {
	// 更新enabled
	if err := s.setConfigByKey(ctx, ConfigGroupSyncEnabled, strconv.FormatBool(config.Enabled)); err != nil {
		return fmt.Errorf("failed to update enabled: %w", err)
	}

	// 更新cron
	if err := s.setConfigByKey(ctx, ConfigGroupSyncCron, config.Cron); err != nil {
		return fmt.Errorf("failed to update cron: %w", err)
	}

	// 更新member_ou
	if err := s.setConfigByKey(ctx, ConfigGroupMemberOU, config.MemberOU); err != nil {
		return fmt.Errorf("failed to update member_ou: %w", err)
	}

	// 更新auto_create
	if err := s.setConfigByKey(ctx, ConfigGroupAutoCreate, strconv.FormatBool(config.AutoCreateGroups)); err != nil {
		return fmt.Errorf("failed to update auto_create: %w", err)
	}

	// 更新max_concurrent
	if err := s.setConfigByKey(ctx, ConfigGroupMaxConcurrent, strconv.Itoa(config.MaxConcurrent)); err != nil {
		return fmt.Errorf("failed to update max_concurrent: %w", err)
	}

	// 更新batch_size
	if err := s.setConfigByKey(ctx, ConfigGroupSyncBatchSize, strconv.Itoa(config.SyncBatchSize)); err != nil {
		return fmt.Errorf("failed to update batch_size: %w", err)
	}

	return nil
}

// ValidateConfig 验证配置有效性
func (s *GroupConfigService) ValidateConfig(config *GroupSyncConfig) error {
	// 验证cron表达式
	if config.Cron == "" {
		return fmt.Errorf("cron表达式不能为空")
	}

	// 验证并发数
	if config.MaxConcurrent < 1 || config.MaxConcurrent > 20 {
		return fmt.Errorf("max_concurrent必须在1-20之间")
	}

	// 验证批量大小
	if config.SyncBatchSize < 10 || config.SyncBatchSize > 1000 {
		return fmt.Errorf("sync_batch_size必须在10-1000之间")
	}

	return nil
}
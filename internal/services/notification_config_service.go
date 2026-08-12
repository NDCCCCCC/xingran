package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// NotificationConfigService 通知配置服务
type NotificationConfigService struct {
	db *gorm.DB
}

// NewNotificationConfigService 创建通知配置服务
func NewNotificationConfigService(db *gorm.DB) *NotificationConfigService {
	return &NotificationConfigService{db: db}
}

// ============= 邮箱配置管理 =============

// EmailConfigListRequest 邮箱配置列表请求
type EmailConfigListRequest struct {
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
	Status   *int `json:"status"`
}

// ListEmailConfigs 获取邮箱配置列表
func (s *NotificationConfigService) ListEmailConfigs(ctx context.Context, page, pageSize int, status *int) ([]models.EmailConfig, int64, error) {
	var configs []models.EmailConfig
	var total int64

	query := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("del_flag = 0")

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计邮箱配置数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询邮箱配置列表失败: %w", err)
	}

	return configs, total, nil
}

// GetEmailConfigByID 根据ID获取邮箱配置
func (s *NotificationConfigService) GetEmailConfigByID(ctx context.Context, id string) (*models.EmailConfig, error) {
	var config models.EmailConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("邮箱配置不存在")
		}
		return nil, fmt.Errorf("查询邮箱配置失败: %w", err)
	}
	return &config, nil
}

// GetDefaultEmailConfig 获取邮件配置（系统只有一条配置）
func (s *NotificationConfigService) GetDefaultEmailConfig(ctx context.Context) (*models.EmailConfig, error) {
	var config models.EmailConfig
	if err := s.db.WithContext(ctx).Where("status = 0 AND del_flag = 0").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未设置邮件配置")
		}
		return nil, fmt.Errorf("查询邮件配置失败: %w", err)
	}
	return &config, nil
}

// CreateEmailConfig 创建邮箱配置（只允许一条配置）
func (s *NotificationConfigService) CreateEmailConfig(ctx context.Context, config *models.EmailConfig) error {
	// 检查是否已存在邮件配置
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("del_flag = 0").Count(&count).Error; err != nil {
		return fmt.Errorf("检查邮件配置失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("邮件配置已存在，系统只允许一条邮件配置。请先删除现有配置后再创建")
	}

	// 创建新配置（自动设为默认）
	config.IsDefault = true
	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("创建邮箱配置失败: %w", err)
	}
	return nil
}

// UpdateEmailConfig 更新邮箱配置
func (s *NotificationConfigService) UpdateEmailConfig(ctx context.Context, id string, config *models.EmailConfig) error {
	// 检查配置是否存在
	var existing models.EmailConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("邮箱配置不存在")
		}
		return fmt.Errorf("查询邮箱配置失败: %w", err)
	}

	// 如果设置为默认，先取消其他默认配置
	if config.IsDefault {
		s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("id != ? AND del_flag = 0", id).Update("is_default", false)
	}

	// 更新
	if err := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("id = ?", id).Updates(config).Error; err != nil {
		return fmt.Errorf("更新邮箱配置失败: %w", err)
	}
	return nil
}

// DeleteEmailConfig 删除邮箱配置
func (s *NotificationConfigService) DeleteEmailConfig(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&models.EmailConfig{}).Where("id = ?", id).Update("del_flag", 1)
	if result.Error != nil {
		return fmt.Errorf("删除邮箱配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("邮箱配置不存在")
	}
	return nil
}

// ============= API通知配置管理 =============

// APINotificationConfigListRequest API通知配置列表请求
type APINotificationConfigListRequest struct {
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	ConfigType *models.APIConfigType `json:"configType"`
	Status     *int                  `json:"status"`
}

// ListAPINotificationConfigs 获取API通知配置列表
func (s *NotificationConfigService) ListAPINotificationConfigs(ctx context.Context, page, pageSize int, configType *models.APIConfigType, status *int) ([]models.APINotificationConfig, int64, error) {
	var configs []models.APINotificationConfig
	var total int64

	query := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("del_flag = 0")

	if configType != nil {
		query = query.Where("config_type = ?", *configType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计API配置数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询API配置列表失败: %w", err)
	}

	return configs, total, nil
}

// GetAPINotificationConfigByID 根据ID获取API通知配置
func (s *NotificationConfigService) GetAPINotificationConfigByID(ctx context.Context, id string) (*models.APINotificationConfig, error) {
	var config models.APINotificationConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API通知配置不存在")
		}
		return nil, fmt.Errorf("查询API通知配置失败: %w", err)
	}
	return &config, nil
}

// CreateAPINotificationConfig 创建API通知配置
func (s *NotificationConfigService) CreateAPINotificationConfig(ctx context.Context, config *models.APINotificationConfig) error {
	// 如果设置为默认，先取消同类型的其他默认配置
	if config.IsDefault {
		s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).
			Where("config_type = ? AND del_flag = 0", config.ConfigType).
			Update("is_default", false)
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("创建API通知配置失败: %w", err)
	}
	return nil
}

// UpdateAPINotificationConfig 更新API通知配置
func (s *NotificationConfigService) UpdateAPINotificationConfig(ctx context.Context, id string, config *models.APINotificationConfig) error {
	// 检查配置是否存在
	var existing models.APINotificationConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND del_flag = 0", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("API通知配置不存在")
		}
		return fmt.Errorf("查询API通知配置失败: %w", err)
	}

	// 如果设置为默认，先取消同类型的其他默认配置
	if config.IsDefault {
		s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).
			Where("config_type = ? AND id != ? AND del_flag = 0", config.ConfigType, id).
			Update("is_default", false)
	}

	// 更新
	if err := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("id = ?", id).Updates(config).Error; err != nil {
		return fmt.Errorf("更新API通知配置失败: %w", err)
	}
	return nil
}

// DeleteAPINotificationConfig 删除API通知配置
func (s *NotificationConfigService) DeleteAPINotificationConfig(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&models.APINotificationConfig{}).Where("id = ?", id).Update("del_flag", 1)
	if result.Error != nil {
		return fmt.Errorf("删除API通知配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API通知配置不存在")
	}
	return nil
}

// ============= 密码加密工具 =============

// 注意：这里使用简单的加密，实际生产环境建议使用更安全的方案
// 加密密钥应该从配置中读取

// EncryptPassword 加密密码
func EncryptPassword(plainText, key string) (string, error) {
	if key == "" {
		key = "xingran-default-key" // 默认密钥，实际应从配置读取
	}

	// 确保密钥长度为16/24/32字节
	if len(key) < 16 {
		key = key + "xingran-notificaion"[len(key):16]
	}
	key = key[:16]

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword 解密密码
func DecryptPassword(cipherText, key string) (string, error) {
	if key == "" {
		key = "xingran-default-key"
	}

	// 确保密钥长度为16/24/32字节
	if len(key) < 16 {
		key = key + "xingran-notificaion"[len(key):16]
	}
	key = key[:16]

	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

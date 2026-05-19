package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// APIKeyService API密钥服务接口
type APIKeyService interface {
	CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error)
	ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error)
	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error
	DeleteAPIKey(ctx context.Context, id string) error
	ToggleAPIKeyStatus(ctx context.Context, id string) error
	ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error)
}

// apiKeyServiceImpl API密钥服务实现
type apiKeyServiceImpl struct {
	db *gorm.DB
}

// NewAPIKeyService 创建API密钥服务实例
func NewAPIKeyService(db *gorm.DB) APIKeyService {
	return &apiKeyServiceImpl{
		db: db,
	}
}

// 常量定义
const (
	KeyPrefix      = "rec_"            // 密钥前缀
	KeyLength      = 64                // 密钥长度（十六进制字符数）
	MaxKeysPerUser = 100               // 单用户最大密钥数
)

// PageResult is declared in user_service.go to avoid redeclaration

// isKeyExpired 检查密钥是否过期
func isKeyExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return expiresAt.Before(time.Now())
}

// isValidKeyFormat 验证密钥格式
func isValidKeyFormat(keyStr string) bool {
	// 检查长度：4（前缀）+ 64（hex）= 68
	if len(keyStr) != 68 {
		return false
	}

	// 检查前缀
	if len(keyStr) < 4 || keyStr[:4] != KeyPrefix {
		return false
	}

	// 检查后64位是否为有效十六进制
	hexPart := keyStr[4:]
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// validateScopes 验证作用域
func validateScopes(scopes []string) error {
	validScopes := map[string]bool{
		"read":  true,
		"write": true,
		"admin": true,
	}

	for _, scope := range scopes {
		if !validScopes[scope] {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "无效的作用域: "+scope)
		}
	}

	return nil
}

// generateKey 生成随机密钥
func generateKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 64 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hexStr := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s%s", KeyPrefix, hexStr), nil
}

// ValidateAPIKey 验证API密钥
func (s *apiKeyServiceImpl) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
	// 验证密钥格式
	if !isValidKeyFormat(keyStr) {
		return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "无效的密钥格式")
	}

	// 查询数据库
	var apiKey models.APIKey
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("key = ? AND is_active = ?", keyStr, true).
		First(&apiKey).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥不存在或已禁用")
		}
		return nil, apperrors.DatabaseError(err)
	}

	// 验证过期时间
	if isKeyExpired(apiKey.ExpiresAt) {
		return nil, apperrors.Wrap(nil, apperrors.CodeUnauthorized, "密钥已过期")
	}

	// 异步更新最后使用时间
	go func() {
		now := time.Now()
		s.db.Model(&apiKey).Update("last_used_at", now)
	}()

	return &apiKey, nil
}

// CreateAPIKey 创建API密钥（stub implementation for Task 2）
func (s *apiKeyServiceImpl) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) {
	// Task 4 will implement this
	return nil, apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

// ListAPIKeys 查询API密钥列表（stub implementation for Task 2）
func (s *apiKeyServiceImpl) ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error) {
	// Task 4 will implement this
	return nil, apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

// GetAPIKey 获取API密钥详情（stub implementation for Task 2）
func (s *apiKeyServiceImpl) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	// Task 4 will implement this
	return nil, apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

// UpdateAPIKey 更新API密钥（stub implementation for Task 2）
func (s *apiKeyServiceImpl) UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error {
	// Task 4 will implement this
	return apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

// DeleteAPIKey 删除API密钥（stub implementation for Task 2）
func (s *apiKeyServiceImpl) DeleteAPIKey(ctx context.Context, id string) error {
	// Task 4 will implement this
	return apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

// ToggleAPIKeyStatus 切换API密钥状态（stub implementation for Task 2）
func (s *apiKeyServiceImpl) ToggleAPIKeyStatus(ctx context.Context, id string) error {
	// Task 4 will implement this
	return apperrors.Wrap(nil, apperrors.CodeServerError, "not implemented")
}

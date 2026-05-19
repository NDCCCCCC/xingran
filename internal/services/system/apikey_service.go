package system

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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

// CreateAPIKey 创建API密钥
func (s *apiKeyServiceImpl) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) {
	// 验证用户是否存在
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "用户不存在")
		}
		return nil, apperrors.DatabaseError(err)
	}

	// 检查用户密钥数量限制
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}
	if count >= MaxKeysPerUser {
		return nil, apperrors.Wrap(nil, apperrors.CodeParamError, fmt.Sprintf("已达到最大密钥数量限制（%d个）", MaxKeysPerUser))
	}

	// 生成密钥
	key, err := generateKey()
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeServerError, "密钥生成失败")
	}

	// 验证作用域
	if err := validateScopes(req.Scopes); err != nil {
		return nil, err
	}

	// 转换作用域为JSON
	scopesJSON, err := json.Marshal(req.Scopes)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeServerError, "作用域序列化失败")
	}

	// 转换IP白名单为JSON
	ipWhitelistJSON, err := json.Marshal(req.IPWhitelist)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeServerError, "IP白名单序列化失败")
	}

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.CodeParamError, "过期时间格式错误")
		}
		expiresAt = &parsedTime
	}

	// 创建APIKey记录
	apiKey := models.APIKey{
		Name:         req.Name,
		Key:          key,
		UserID:       &userID,
		Scopes:       string(scopesJSON),
		IPWhitelist:  string(ipWhitelistJSON),
		Description:  req.Description,
		InheritPerms: req.InheritPerms,
		ExpiresAt:    expiresAt,
		IsActive:     true,
	}

	if err := s.db.WithContext(ctx).Create(&apiKey).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	return &key, nil
}

// ListAPIKeys 查询API密钥列表
func (s *apiKeyServiceImpl) ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*PageResult, error) {
	var total int64
	var list []models.APIKey

	query := s.db.WithContext(ctx).Model(&models.APIKey{})

	// 筛选用户ID
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// 添加筛选条件
	if params.Keyword != nil && *params.Keyword != "" {
		keyword := "%" + *params.Keyword + "%"
		query = query.Where("name LIKE ? OR LEFT(key, 12) LIKE ?", keyword, keyword)
	}
	if params.Status != nil {
		query = query.Where("is_active = ?", *params.Status)
	}
	if params.Scope != nil && *params.Scope != "" {
		query = query.Where("scopes @> ?", fmt.Sprintf("[\"%s\"]", *params.Scope))
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	// 分页查询
	offset := params.GetOffset()
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(params.PageSize).
		Find(&list).Error; err != nil {
		return nil, apperrors.DatabaseError(err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// GetAPIKey 获取API密钥详情
func (s *apiKeyServiceImpl) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := s.db.WithContext(ctx).Preload("User").First(&apiKey, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return nil, apperrors.DatabaseError(err)
	}
	return &apiKey, nil
}

// UpdateAPIKey 更新API密钥
func (s *apiKeyServiceImpl) UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 更新可修改字段
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.InheritPerms != nil {
		updates["inherit_perms"] = *req.InheritPerms
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			updates["expires_at"] = nil
		} else {
			parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				return apperrors.Wrap(err, apperrors.CodeParamError, "过期时间格式错误")
			}
			updates["expires_at"] = parsedTime
		}
	}
	if req.Scopes != nil {
		if err := validateScopes(req.Scopes); err != nil {
			return err
		}
		scopesJSON, err := json.Marshal(req.Scopes)
		if err != nil {
			return apperrors.Wrap(err, apperrors.CodeServerError, "作用域序列化失败")
		}
		updates["scopes"] = string(scopesJSON)
	}
	if req.IPWhitelist != nil {
		ipWhitelistJSON, err := json.Marshal(req.IPWhitelist)
		if err != nil {
			return apperrors.Wrap(err, apperrors.CodeServerError, "IP白名单序列化失败")
		}
		updates["ip_whitelist"] = string(ipWhitelistJSON)
	}

	if err := s.db.WithContext(ctx).Model(&apiKey).Updates(updates).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// DeleteAPIKey 删除API密钥
func (s *apiKeyServiceImpl) DeleteAPIKey(ctx context.Context, id string) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 软删除
	if err := s.db.WithContext(ctx).Delete(&apiKey).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

// ToggleAPIKeyStatus 切换API密钥状态
func (s *apiKeyServiceImpl) ToggleAPIKeyStatus(ctx context.Context, id string) error {
	// 检查密钥是否存在
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.Wrap(nil, apperrors.CodeParamError, "密钥不存在")
		}
		return apperrors.DatabaseError(err)
	}

	// 切换状态
	apiKey.IsActive = !apiKey.IsActive
	if err := s.db.WithContext(ctx).Save(&apiKey).Error; err != nil {
		return apperrors.DatabaseError(err)
	}

	return nil
}

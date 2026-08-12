package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// Cache 缓存接口别名
type CredentialCache = cache.Cache

// CredentialService 凭证服务接口
type CredentialService interface {
	// 凭证管理
	CreateCredential(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error)
	UpdateCredential(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error
	DeleteCredential(ctx context.Context, id string, userID string) error
	GetCredential(ctx context.Context, id string, userID string) (*rpamodels.RPACredential, error)
	ListCredentials(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error)
	GetCredentialForExecution(ctx context.Context, targetSystem string, userID string, deptID string) (*rpamodels.RPACredential, error)

	// 凭证使用（解密）
	DecryptCredential(ctx context.Context, cred *rpamodels.RPACredential) (*CredentialData, error)

	// 会话管理
	CreateSession(ctx context.Context, req *rpamodels.SessionCreateRequest) (*rpamodels.RPASession, error)
	GetValidSession(ctx context.Context, credentialID string, targetSystem string) (*rpamodels.RPASession, error)
	InvalidateSession(ctx context.Context, sessionID string, reason string) error
	CleanupExpiredSessions(ctx context.Context) error

	// 登录状态跟踪
	RecordLoginSuccess(ctx context.Context, credentialID string) error
	RecordLoginFailure(ctx context.Context, credentialID string) error
	UpdateLastUsed(ctx context.Context, credentialID string) error
}

// CredentialData 解密后的凭证数据
type CredentialData struct {
	Username  string                 `json:"username"`
	Password  string                 `json:"password"`
	ExtraData map[string]interface{} `json:"extraData,omitempty"`
}

// SessionData 解密后的会话数据
type SessionData struct {
	AccessToken  string                 `json:"accessToken,omitempty"`
	RefreshToken string                 `json:"refreshToken,omitempty"`
	Cookies      []rpamodels.Cookie     `json:"cookies,omitempty"`
	SessionData  map[string]interface{} `json:"sessionData,omitempty"`
}

type credentialServiceImpl struct {
	db             *gorm.DB
	passwordCipher addomain.PasswordCipher
	cache          CredentialCache
}

// NewCredentialService 创建凭证服务
func NewCredentialService(db *gorm.DB, passwordCipher addomain.PasswordCipher, cache CredentialCache) CredentialService {
	return &credentialServiceImpl{
		db:             db,
		passwordCipher: passwordCipher,
		cache:          cache,
	}
}

// CreateCredential 创建凭证
func (s *credentialServiceImpl) CreateCredential(ctx context.Context, req *rpamodels.CredentialCreateRequest, userID string) (*rpamodels.RPACredential, error) {
	// 加密凭证
	encrypted, err := s.encryptCredentialData(req.Username, req.Password, req.ExtraData)
	if err != nil {
		return nil, fmt.Errorf("加密凭证失败: %w", err)
	}

	cred := &rpamodels.RPACredential{
		Name:               req.Name,
		TargetSystem:       req.TargetSystem,
		TargetURL:          req.TargetURL,
		UsernameEncrypted:  encrypted.Username,
		PasswordEncrypted:  encrypted.Password,
		ExtraDataEncrypted: encrypted.ExtraData,
		UserID:             userID,
		IsShared:           req.IsShared,
		Status:             0, // 默认正常
	}

	if err := s.db.WithContext(ctx).Create(cred).Error; err != nil {
		return nil, err
	}

	// 返回时填充明文供前端显示
	cred.Username = req.Username
	return cred, nil
}

// UpdateCredential 更新凭证
func (s *credentialServiceImpl) UpdateCredential(ctx context.Context, id string, req *rpamodels.CredentialUpdateRequest, userID string) error {
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.IsShared != nil {
		updates["is_shared"] = *req.IsShared
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 如果需要更新用户名或密码，重新加密
	if req.Username != "" || req.Password != "" {
		// 先获取现有凭证
		var existing rpamodels.RPACredential
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&existing).Error; err != nil {
			return err
		}

		// 准备加密数据（如果未提供则使用现有值）
		username := req.Username
		password := req.Password
		extraData := req.ExtraData

		// 加密
		encrypted, err := s.encryptCredentialData(
			sOrDefault(username, s.decryptString(existing.UsernameEncrypted)),
			sOrDefault(password, s.decryptString(existing.PasswordEncrypted)),
			extraData,
		)
		if err != nil {
			return fmt.Errorf("加密凭证失败: %w", err)
		}

		updates["username_encrypted"] = encrypted.Username
		updates["password_encrypted"] = encrypted.Password
		if encrypted.ExtraData != "" {
			updates["extra_data_encrypted"] = encrypted.ExtraData
		}
	}

	return s.db.WithContext(ctx).Model(&rpamodels.RPACredential{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates).Error
}

// DeleteCredential 删除凭证
func (s *credentialServiceImpl) DeleteCredential(ctx context.Context, id string, userID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&rpamodels.RPACredential{}).Error
}

// GetCredential 获取凭证
func (s *credentialServiceImpl) GetCredential(ctx context.Context, id string, userID string) (*rpamodels.RPACredential, error) {
	var cred rpamodels.RPACredential
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&cred).Error
	if err != nil {
		return nil, err
	}

	// 返回用户名（不返回密码）
	cred.Username = s.decryptString(cred.UsernameEncrypted)
	return &cred, nil
}

// ListCredentials 列出凭证
func (s *credentialServiceImpl) ListCredentials(ctx context.Context, params *rpamodels.CredentialListParams, userID string, deptID string) ([]rpamodels.RPACredential, int64, error) {
	query := s.db.WithContext(ctx).Model(&rpamodels.RPACredential{})

	// 权限过滤：只能看到自己的或部门共享的
	query = query.Where("(user_id = ? OR (is_shared = true AND dept_id = ?))", userID, deptID)

	// 状态过滤
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 目标系统过滤
	if params.TargetSystem != "" {
		query = query.Where("target_system = ?", params.TargetSystem)
	}

	// 仅查看我的凭证
	if params.MyCredOnly {
		query = query.Where("user_id = ?", userID)
	}

	// 软删除过滤
	query = query.Where("deleted_at IS NULL")

	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var credentials []rpamodels.RPACredential
	offset := (params.Current - 1) * params.PageSize
	err := query.Order("created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&credentials).Error

	// 解密用户名供显示
	for i := range credentials {
		credentials[i].Username = s.decryptString(credentials[i].UsernameEncrypted)
	}

	return credentials, total, err
}

// GetCredentialForExecution 获取用于执行的凭证（优先使用自己的，其次部门共享的）
func (s *credentialServiceImpl) GetCredentialForExecution(ctx context.Context, targetSystem string, userID string, deptID string) (*rpamodels.RPACredential, error) {
	var cred rpamodels.RPACredential

	// 优先使用自己的有效凭证
	err := s.db.WithContext(ctx).
		Where("target_system = ? AND user_id = ? AND status = 0", targetSystem, userID).
		Order("last_used_at DESC NULLS LAST, created_at DESC").
		First(&cred).Error

	if err == gorm.ErrRecordNotFound {
		// 尝试获取部门共享的凭证
		err = s.db.WithContext(ctx).
			Where("target_system = ? AND is_shared = true AND dept_id = ? AND status = 0", targetSystem, deptID).
			Order("last_used_at DESC NULLS LAST, created_at DESC").
			First(&cred).Error
	}

	if err != nil {
		return nil, fmt.Errorf("未找到有效的凭证: %w", err)
	}

	return &cred, nil
}

// DecryptCredential 解密凭证（供执行时使用）
func (s *credentialServiceImpl) DecryptCredential(ctx context.Context, cred *rpamodels.RPACredential) (*CredentialData, error) {
	return &CredentialData{
		Username:  s.decryptString(cred.UsernameEncrypted),
		Password:  s.decryptString(cred.PasswordEncrypted),
		ExtraData: s.decryptExtraData(cred.ExtraDataEncrypted),
	}, nil
}

// CreateSession 创建会话
func (s *credentialServiceImpl) CreateSession(ctx context.Context, req *rpamodels.SessionCreateRequest) (*rpamodels.RPASession, error) {
	// 加密会话数据
	encrypted, err := s.encryptSessionData(req)
	if err != nil {
		return nil, fmt.Errorf("加密会话数据失败: %w", err)
	}

	session := &rpamodels.RPASession{
		CredentialID:          req.CredentialID,
		ExecutionID:           req.ExecutionID,
		TargetSystem:          req.TargetSystem,
		TargetURL:             req.TargetURL,
		AccessTokenEncrypted:  encrypted.AccessToken,
		RefreshTokenEncrypted: encrypted.RefreshToken,
		CookiesEncrypted:      encrypted.Cookies,
		SessionDataEncrypted:  encrypted.SessionData,
		ExpiresAt:             req.ExpiresAt,
		IsValid:               true,
	}

	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}

	// 返回时填充明文供响应
	session.AccessToken = req.AccessToken
	session.RefreshToken = req.RefreshToken
	session.Cookies = req.Cookies
	session.SessionData = req.SessionData

	return session, nil
}

// GetValidSession 获取有效会话
func (s *credentialServiceImpl) GetValidSession(ctx context.Context, credentialID string, targetSystem string) (*rpamodels.RPASession, error) {
	var session rpamodels.RPASession

	query := s.db.WithContext(ctx).
		Where("credential_id = ? AND target_system = ? AND is_valid = true", credentialID, targetSystem)

	// 检查过期时间
	query = query.Where("(expires_at IS NULL OR expires_at > ?)", time.Now())

	err := query.Order("created_at DESC").First(&session).Error
	if err != nil {
		return nil, err
	}

	// 解密会话数据
	s.decryptSessionData(&session)

	return &session, nil
}

// InvalidateSession 使会话失效
func (s *credentialServiceImpl) InvalidateSession(ctx context.Context, sessionID string, reason string) error {
	return s.db.WithContext(ctx).Model(&rpamodels.RPASession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"is_valid":       false,
			"invalid_reason": reason,
		}).Error
}

// CleanupExpiredSessions 清理过期会话
func (s *credentialServiceImpl) CleanupExpiredSessions(ctx context.Context) error {
	return s.db.WithContext(ctx).Model(&rpamodels.RPASession{}).
		Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).
		Updates(map[string]interface{}{
			"is_valid":       false,
			"invalid_reason": "expired",
		}).Error
}

// RecordLoginSuccess 记录登录成功
func (s *credentialServiceImpl) RecordLoginSuccess(ctx context.Context, credentialID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&rpamodels.RPACredential{}).
		Where("id = ?", credentialID).
		Updates(map[string]interface{}{
			"last_login_at":       &now,
			"login_success_count": gorm.Expr("login_success_count + 1"),
		}).Error
}

// RecordLoginFailure 记录登录失败
func (s *credentialServiceImpl) RecordLoginFailure(ctx context.Context, credentialID string) error {
	return s.db.WithContext(ctx).Model(&rpamodels.RPACredential{}).
		Where("id = ?", credentialID).
		Update("login_fail_count", gorm.Expr("login_fail_count + 1")).Error
}

// UpdateLastUsed 更新最后使用时间
func (s *credentialServiceImpl) UpdateLastUsed(ctx context.Context, credentialID string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&rpamodels.RPACredential{}).
		Where("id = ?", credentialID).
		Update("last_used_at", &now).Error
}

// ===== 加密/解密辅助方法 =====

type encryptedCredentialData struct {
	Username  string
	Password  string
	ExtraData string
}

type encryptedSessionData struct {
	AccessToken  string
	RefreshToken string
	Cookies      string
	SessionData  string
}

func (s *credentialServiceImpl) encryptCredentialData(username, password string, extraData map[string]interface{}) (*encryptedCredentialData, error) {
	result := &encryptedCredentialData{}

	// 加密用户名
	if encrypted, err := s.passwordCipher.Encrypt(username); err == nil {
		result.Username = encrypted
	} else {
		return nil, err
	}

	// 加密密码
	if encrypted, err := s.passwordCipher.Encrypt(password); err == nil {
		result.Password = encrypted
	} else {
		return nil, err
	}

	// 加密额外数据
	if extraData != nil {
		if jsonBytes, err := json.Marshal(extraData); err == nil {
			if encrypted, err := s.passwordCipher.Encrypt(string(jsonBytes)); err == nil {
				result.ExtraData = encrypted
			}
		}
	}

	return result, nil
}

func (s *credentialServiceImpl) encryptSessionData(req *rpamodels.SessionCreateRequest) (*encryptedSessionData, error) {
	result := &encryptedSessionData{}

	// 加密 access token
	if req.AccessToken != "" {
		if encrypted, err := s.passwordCipher.Encrypt(req.AccessToken); err == nil {
			result.AccessToken = encrypted
		}
	}

	// 加密 refresh token
	if req.RefreshToken != "" {
		if encrypted, err := s.passwordCipher.Encrypt(req.RefreshToken); err == nil {
			result.RefreshToken = encrypted
		}
	}

	// 加密 cookies
	if len(req.Cookies) > 0 {
		if jsonBytes, err := json.Marshal(req.Cookies); err == nil {
			if encrypted, err := s.passwordCipher.Encrypt(string(jsonBytes)); err == nil {
				result.Cookies = encrypted
			}
		}
	}

	// 加密 session data
	if req.SessionData != nil {
		if jsonBytes, err := json.Marshal(req.SessionData); err == nil {
			if encrypted, err := s.passwordCipher.Encrypt(string(jsonBytes)); err == nil {
				result.SessionData = encrypted
			}
		}
	}

	return result, nil
}

func (s *credentialServiceImpl) decryptString(encrypted string) string {
	if decrypted, err := s.passwordCipher.Decrypt(encrypted); err == nil {
		return decrypted
	}
	return ""
}

func (s *credentialServiceImpl) decryptExtraData(encrypted string) map[string]interface{} {
	if decrypted, err := s.passwordCipher.Decrypt(encrypted); err == nil {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(decrypted), &result); err == nil {
			return result
		}
	}
	return nil
}

func (s *credentialServiceImpl) decryptSessionData(session *rpamodels.RPASession) {
	// 解密 access token
	if session.AccessTokenEncrypted != "" {
		session.AccessToken = s.decryptString(session.AccessTokenEncrypted)
	}

	// 解密 refresh token
	if session.RefreshTokenEncrypted != "" {
		session.RefreshToken = s.decryptString(session.RefreshTokenEncrypted)
	}

	// 解密 cookies
	if session.CookiesEncrypted != "" {
		if decrypted, err := s.passwordCipher.Decrypt(session.CookiesEncrypted); err == nil {
			if err := json.Unmarshal([]byte(decrypted), &session.Cookies); err != nil {
				applogger.Warnf("反序列化会话 Cookies 数据失败: %v", err)
			}
		}
	}

	// 解密 session data
	if session.SessionDataEncrypted != "" {
		if decrypted, err := s.passwordCipher.Decrypt(session.SessionDataEncrypted); err == nil {
			if err := json.Unmarshal([]byte(decrypted), &session.SessionData); err != nil {
				applogger.Warnf("反序列化会话 SessionData 数据失败: %v", err)
			}
		}
	}
}

func sOrDefault(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

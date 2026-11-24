package rpa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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

// ErrCipherNotConfigured 凭证加密器未配置（CONTRACT-01）。
//
// 历史缺陷（D1）：NewServiceGroup 第 5 参传 nil，而 CredentialService 的加解密
// 辅助方法直接调用 s.passwordCipher.Encrypt/Decrypt —— nil interface 调用必然
// panic（nil pointer dereference），HTTP 层表现为 500 且无可读原因。
// 现在所有加解密入口先做 nil 守卫，返回本错误；handler 层的 handleError 会把它
// 拼进响应 message（"创建失败: 凭证加密器未配置"），前端可读、可定位。
var ErrCipherNotConfigured = errors.New("凭证加密器未配置")

// requireCipher 断言加密器已配置。nil 判定同时覆盖接口未赋值与 typed-nil 两种情况。
func (s *credentialServiceImpl) requireCipher() error {
	if s == nil || s.passwordCipher == nil {
		return ErrCipherNotConfigured
	}
	return nil
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
		Status:             rpamodels.CredentialStatusNormal, // 默认正常
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

		// 解密现有密文以便「只改一项」时复用另一项
		existingUsername, err := s.decryptString(existing.UsernameEncrypted)
		if err != nil {
			return err
		}
		existingPassword, err := s.decryptString(existing.PasswordEncrypted)
		if err != nil {
			return err
		}

		// 加密
		encrypted, err := s.encryptCredentialData(
			sOrDefault(username, existingUsername),
			sOrDefault(password, existingPassword),
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

	// 返回用户名（不返回密码）；cipher 未配置 → 明确报错而非空值
	username, err := s.decryptString(cred.UsernameEncrypted)
	if err != nil {
		return nil, err
	}
	cred.Username = username
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

	// 解密用户名供显示；cipher 未配置 → 明确报错（不返回一列空用户名冒充成功）
	for i := range credentials {
		username, err := s.decryptString(credentials[i].UsernameEncrypted)
		if err != nil {
			return nil, 0, err
		}
		credentials[i].Username = username
	}

	return credentials, total, err
}

// GetCredentialForExecution 获取用于执行的凭证（优先使用自己的，其次部门共享的）
func (s *credentialServiceImpl) GetCredentialForExecution(ctx context.Context, targetSystem string, userID string, deptID string) (*rpamodels.RPACredential, error) {
	var cred rpamodels.RPACredential

	// 优先使用自己的有效凭证
	err := s.db.WithContext(ctx).
		Where("target_system = ? AND user_id = ? AND status = ?", targetSystem, userID, rpamodels.CredentialStatusNormal).
		Order("last_used_at DESC NULLS LAST, created_at DESC").
		First(&cred).Error

	if err == gorm.ErrRecordNotFound {
		// 尝试获取部门共享的凭证
		err = s.db.WithContext(ctx).
			Where("target_system = ? AND is_shared = true AND dept_id = ? AND status = ?", targetSystem, deptID, rpamodels.CredentialStatusNormal).
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
	username, err := s.decryptString(cred.UsernameEncrypted)
	if err != nil {
		return nil, err
	}
	password, err := s.decryptString(cred.PasswordEncrypted)
	if err != nil {
		return nil, err
	}
	extraData, err := s.decryptExtraData(cred.ExtraDataEncrypted)
	if err != nil {
		return nil, err
	}
	return &CredentialData{
		Username:  username,
		Password:  password,
		ExtraData: extraData,
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

	// 解密会话数据（cipher 未配置 → 明确报错）
	if err := s.decryptSessionData(&session); err != nil {
		return nil, err
	}

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
	if err := s.requireCipher(); err != nil {
		return nil, err
	}
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
	if err := s.requireCipher(); err != nil {
		return nil, err
	}
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

// decryptString 解密单个字段。
//
// CONTRACT-01：cipher 未配置时返回 ErrCipherNotConfigured 而非 panic。
// 解密失败（密文损坏/密钥轮换）保留既有静默回退语义（返回 ""、nil error）——
// 列表页不应因为历史脏密文整页 500。
func (s *credentialServiceImpl) decryptString(encrypted string) (string, error) {
	if err := s.requireCipher(); err != nil {
		return "", err
	}
	decrypted, err := s.passwordCipher.Decrypt(encrypted)
	if err != nil {
		return "", nil
	}
	return decrypted, nil
}

// decryptExtraData 解密并反序列化 extra data。
//
// CONTRACT-01：cipher 未配置时返回 (nil, ErrCipherNotConfigured)。
func (s *credentialServiceImpl) decryptExtraData(encrypted string) (map[string]interface{}, error) {
	if err := s.requireCipher(); err != nil {
		return nil, err
	}
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := s.passwordCipher.Decrypt(encrypted)
	if err != nil {
		return nil, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(decrypted), &result); err != nil {
		return nil, nil
	}
	return result, nil
}

func (s *credentialServiceImpl) decryptSessionData(session *rpamodels.RPASession) error {
	if err := s.requireCipher(); err != nil {
		return err
	}
	// 解密 access token
	if session.AccessTokenEncrypted != "" {
		token, err := s.decryptString(session.AccessTokenEncrypted)
		if err != nil {
			return err
		}
		session.AccessToken = token
	}

	// 解密 refresh token
	if session.RefreshTokenEncrypted != "" {
		token, err := s.decryptString(session.RefreshTokenEncrypted)
		if err != nil {
			return err
		}
		session.RefreshToken = token
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

	return nil
}

func sOrDefault(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

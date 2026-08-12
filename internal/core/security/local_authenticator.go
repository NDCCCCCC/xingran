package security

import (
	"context"
	"errors"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// LocalAuthenticator 本地认证器
// 使用sys_user表和SM3密码验证实现本地认证
type LocalAuthenticator struct {
	db         *gorm.DB
	pwdManager *PasswordManager
}

// NewLocalAuthenticator 创建本地认证器
func NewLocalAuthenticator(db *gorm.DB, pwdMgr *PasswordManager) *LocalAuthenticator {
	return &LocalAuthenticator{
		db:         db,
		pwdManager: pwdMgr,
	}
}

// Authenticate 实现本地认证
// 查询sys_user表，使用SM3密码验证
func (a *LocalAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	// 1. 查询用户
	var user models.User
	if err := a.db.WithContext(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 2. 检查用户状态（0=启用, 1=禁用）
	if user.Status != models.UserStatusEnabled {
		return nil, ErrUserDisabled
	}

	// 3. 验证密码（使用SM3-PBKDF2）
	if ok, err := a.pwdManager.VerifyPassword(req.Password, user.Password); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrInvalidCredentials
	}

	// 4. 转换为UserResult
	userResult := &UserResult{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Phone:    user.Phone,
		Status:   int(user.Status),
		DeptID:   user.DeptID,
		Roles:    user.Roles,
	}

	return &AuthResult{
		User:       userResult,
		AuthSource: "local",
		NeedsSync:  false,
	}, nil
}

// Name 返回认证器名称
func (a *LocalAuthenticator) Name() string {
	return "local"
}

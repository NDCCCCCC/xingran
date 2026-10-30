package security

import (
	"context"
	"errors"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// 账号认证来源常量（对应 sys_user.auth_source 字段取值）
const (
	// authSourceLocal 本地账号：密码由 sys_user.password 管理，走本地密码通道
	authSourceLocal = "local"
	// authSourceAD AD 域控账号：由 AD 同步进系统，本地密码为随机不可知值
	authSourceAD = "ad"
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

	// 3. AD 来源账号禁止走本地密码通道（P0-2）
	// AD 同步进系统的账号，其本地密码是随机不可知值，本地密码校验必须永久失败，
	// 否则等于给 AD 账号留了后门：AD 侧的禁用/离职/改密将失去效果。
	// 统一返回 ErrInvalidCredentials，不泄露该账号是否存在。
	if strings.EqualFold(strings.TrimSpace(user.AuthSource), authSourceAD) {
		applogger.Warnf("[LocalAuth] AD 来源账号拒绝本地密码认证: username=%s", req.Username)
		return nil, ErrInvalidCredentials
	}

	// 4. 验证密码（使用SM3-PBKDF2）
	if ok, err := a.pwdManager.VerifyPassword(req.Password, user.Password); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrInvalidCredentials
	}

	// 5. 转换为UserResult
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
		AuthSource: authSourceLocal,
		NeedsSync:  false,
	}, nil
}

// Name 返回认证器名称
func (a *LocalAuthenticator) Name() string {
	return "local"
}

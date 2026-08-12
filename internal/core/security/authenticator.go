// Package security 提供安全相关功能
// authenticator.go 定义认证器接口和核心数据结构
package security

import (
	"context"
	"errors"
)

// Authenticator 认证器接口
// 所有认证方式（本地、AD域控、混合）都必须实现此接口
type Authenticator interface {
	// Authenticate 执行认证，返回用户信息或错误
	Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)

	// Name 返回认证器名称
	Name() string
}

// AuthRequest 认证请求
type AuthRequest struct {
	Username string // 用户名
	Password string // 密码（可能已SM2加密）
	IP       string // 客户端IP，用于日志记录
}

// AuthResult 认证结果
type AuthResult struct {
	User       *UserResult // 用户信息（本地认证时有值）
	AuthSource string      // "local" or "ad"
	ADUserInfo *ADUserInfo // AD用户信息（AD认证时有值，用于自动同步）
	NeedsSync  bool        // 是否需要同步用户信息到sys_user
	// SyncErrorReason 当 AD 用户 bind 成功但后续 admin 搜索/sync 失败时,
	// 记录失败原因短码（admin_dial / admin_bind / user_search / user_sync）,
	// 供 login handler 如实反馈给用户,避免误导性"认证成功但用户信息缺失"。
	SyncErrorReason string
}

// UserResult 用户信息（简化版，避免循环依赖）
type UserResult struct {
	ID       string
	Username string
	Nickname *string
	Email    *string
	Phone    *string
	Status   int
	DeptID   *string
	Roles    []string
}

// ADUserInfo AD用户信息（用于AD认证成功后同步到本地）
type ADUserInfo struct {
	UserDN      string // 用户DN
	OUDN        string // 用户OU DN（用于部门映射）
	Username    string // sAMAccountName
	DisplayName string // 显示名称
	Email       string // 邮箱
	Phone       string // 电话
	Mobile      string // 手机
	Title       string // 职位
	Department  string // 部门
}

// UserSyncer 用户同步接口
// 由外部服务实现，将AD用户信息同步到sys_user表
type UserSyncer interface {
	// SyncADUser 从AD同步用户到本地数据库
	// 返回同步后的用户信息和可能的错误
	SyncADUser(ctx context.Context, adUserInfo *ADUserInfo, defaultRoleID string) (*SyncedUser, error)
}

// SyncedUser 同步后的用户信息（由同步服务返回）
type SyncedUser struct {
	ID       string
	Username string
	Nickname *string
	Email    *string
	Phone    *string
	Status   int
	DeptID   *string
	Roles    []string
}

// 认证标准错误定义
var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("用户已被禁用")
	ErrADConfigNotFound   = errors.New("AD配置不存在")
	ErrADConnectionFailed = errors.New("AD连接失败")
)

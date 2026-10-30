package security

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// HybridAuthenticator 混合认证器
//
// 按账号来源（sys_user.auth_source）分流，而不是"本地优先、AD 兜底"：
//
//	auth_source = "ad"    → 只走 AD 认证。AD 失败即失败，绝不回退本地密码
//	auth_source = "local" → 只走本地认证。本地失败即失败，不再拿 AD 兜底
//	查不到用户 / 来源为空或非法 → 兜底语义：先本地后 AD（兼容尚未同步进系统的账号）
//
// 这样 AD 账号在 AD 端被禁用/离职/改密时能立即生效，同时避免 AD 侧同名账号
// 接管本地账号的登录。
type HybridAuthenticator struct {
	localAuth Authenticator
	adAuth    Authenticator
	db        *gorm.DB // 用于查询账号认证来源；为 nil 时退化为"先本地后 AD"兜底语义
}

// NewHybridAuthenticator 创建混合认证器
// db 为可选参数：传入 *gorm.DB 后启用按 auth_source 分流；
// 未传入时保持历史"先本地后 AD"兜底语义（兼容既有调用方与测试）。
func NewHybridAuthenticator(local, ad Authenticator, db ...*gorm.DB) *HybridAuthenticator {
	h := &HybridAuthenticator{
		localAuth: local,
		adAuth:    ad,
	}
	if len(db) > 0 && db[0] != nil {
		h.db = db[0]
	}
	return h
}

// Authenticate 实现混合认证（按账号来源分流）
// 1. 查询账号的 auth_source（软删行视为不存在）
// 2. "ad" → 只走 AD；"local" → 只走本地；查不到 → 先本地后 AD
// 3. AD 认证成功时强制设置 NeedsSync=true
func (h *HybridAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	// 1. 查询账号认证来源
	source, err := h.resolveAuthSource(ctx, req.Username)
	if err != nil {
		// 查库失败：保守降级为"先本地后 AD"，并记录日志
		applogger.Warnf("[HybridAuth] 查询账号认证来源失败，降级为先本地后AD: username=%s, error=%v", req.Username, err)
		return h.authenticateFallback(ctx, req)
	}

	// 2. 按来源分流
	switch source {
	case authSourceAD:
		return h.authenticateAD(ctx, req)
	case authSourceLocal:
		return h.authenticateLocal(ctx, req)
	default:
		// 查不到用户或来源非法：保持兜底语义（兼容尚未同步进系统的账号）
		return h.authenticateFallback(ctx, req)
	}
}

// Name 返回认证器名称
func (h *HybridAuthenticator) Name() string {
	return "hybrid"
}

// resolveAuthSource 查询账号的认证来源
// 返回值："local" / "ad" / ""（空串表示查不到用户或来源非法，走兜底语义）
func (h *HybridAuthenticator) resolveAuthSource(ctx context.Context, username string) (string, error) {
	// 未注入数据库时无法分流，保持兜底语义
	if h.db == nil {
		return "", nil
	}

	// GORM 软删自动过滤 deleted_at 非空的行，软删账号视为不存在
	var user models.User
	if err := h.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	switch {
	case strings.EqualFold(strings.TrimSpace(user.AuthSource), authSourceAD):
		return authSourceAD, nil
	case strings.EqualFold(strings.TrimSpace(user.AuthSource), authSourceLocal):
		return authSourceLocal, nil
	default:
		// 来源为空或非法：按兜底语义处理
		return "", nil
	}
}

// authenticateLocal 只走本地认证
// 本地失败即失败，不再用 AD 兜底（避免 AD 侧同名账号接管本地账号）
func (h *HybridAuthenticator) authenticateLocal(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	applogger.Debugf("[HybridAuth] 账号来源为本地，仅走本地认证: username=%s", req.Username)

	result, err := h.localAuth.Authenticate(ctx, req)
	if err != nil {
		return nil, err
	}
	if result != nil {
		result.NeedsSync = false
	}
	return result, nil
}

// authenticateAD 只走 AD 认证
// AD 失败即失败，绝不回退本地密码（AD 侧禁用/离职/改密必须立即生效）
func (h *HybridAuthenticator) authenticateAD(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	applogger.Debugf("[HybridAuth] 账号来源为AD，仅走AD认证: username=%s", req.Username)

	adResult, adErr := h.adAuth.Authenticate(ctx, req)
	if adErr != nil {
		// 透传 AD 的错误（如 ErrInvalidCredentials / ErrADConnectionFailed）
		return nil, adErr
	}
	if adResult == nil {
		return nil, fmt.Errorf("AD认证返回空结果: %w", ErrInvalidCredentials)
	}

	// AD 认证成功，确保标记需要同步（既有语义：ad_authenticator 内部已做资料同步）
	adResult.NeedsSync = true
	return adResult, nil
}

// authenticateFallback 兜底语义：先本地认证，失败后降级到 AD 域控认证
// 仅用于"账号尚未同步进系统"（查不到用户 / auth_source 非法）的情况
func (h *HybridAuthenticator) authenticateFallback(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	// 1. 先尝试本地认证
	result, err := h.localAuth.Authenticate(ctx, req)
	if err == nil {
		// 本地认证成功，直接返回
		return result, nil
	}

	// 2. 本地认证失败，记录日志（但不返回错误）
	applogger.Debugf("本地认证失败 (user: %s): %v, 尝试AD认证", req.Username, err)

	// 3. 尝试AD认证
	adResult, adErr := h.adAuth.Authenticate(ctx, req)
	if adErr != nil {
		// AD认证也失败，返回本地认证错误（更通用）
		return nil, err
	}
	if adResult == nil {
		return nil, err
	}

	// 4. AD认证成功，确保标记需要同步
	adResult.NeedsSync = true
	return adResult, nil
}

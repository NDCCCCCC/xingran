package security

import (
	"context"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// HybridAuthenticator 混合认证器
// 优先尝试本地认证，失败后降级到AD域控认证
type HybridAuthenticator struct {
	localAuth *LocalAuthenticator
	adAuth    *ADAuthenticator
}

// NewHybridAuthenticator 创建混合认证器
func NewHybridAuthenticator(local *LocalAuthenticator, ad *ADAuthenticator) *HybridAuthenticator {
	return &HybridAuthenticator{
		localAuth: local,
		adAuth:    ad,
	}
}

// Authenticate 实现混合认证（降级逻辑）
// 1. 先尝试本地认证（性能更好）
// 2. 本地认证失败后尝试AD认证
// 3. AD认证成功时强制设置NeedsSync=true
func (h *HybridAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
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

	// 4. AD认证成功，确保标记需要同步
	adResult.NeedsSync = true
	return adResult, nil
}

// Name 返回认证器名称
func (h *HybridAuthenticator) Name() string {
	return "hybrid"
}

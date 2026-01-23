package system

import (
	"context"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
)

// ==================== 密码管理器适配器 ====================

// passwordManagerAdapter 密码管理器适配器
type passwordManagerAdapter struct {
	*security.PasswordManager
}

// NewPasswordManagerAdapter 创建密码管理器适配器
func NewPasswordManagerAdapter(pm *security.PasswordManager) systemServices.PasswordManager {
	return &passwordManagerAdapter{PasswordManager: pm}
}

// HashPassword 哈希密码
func (a *passwordManagerAdapter) HashPassword(password string) (string, error) {
	return a.PasswordManager.HashPassword(password)
}

// VerifyPassword 验证密码
func (a *passwordManagerAdapter) VerifyPassword(password, hash string) (bool, error) {
	return a.PasswordManager.VerifyPassword(password, hash)
}

// ==================== 缓存失效适配器 ====================

// cacheInvalidatorAdapter 缓存失效适配器
type cacheInvalidatorAdapter struct {
	userCacheService interface {
		InvalidateUserCache(ctx context.Context, userID string) error
		InvalidateAllUserCache(ctx context.Context) error
	}
	postCacheService interface {
		InvalidateUserPostCache(ctx context.Context, userID string) error
	}
}

// NewCacheInvalidatorAdapter 创建缓存失效适配器
func NewCacheInvalidatorAdapter(userCacheService, postCacheService interface{}) systemServices.CacheInvalidator {
	return &cacheInvalidatorAdapter{
		userCacheService: userCacheService.(interface {
			InvalidateUserCache(ctx context.Context, userID string) error
			InvalidateAllUserCache(ctx context.Context) error
		}),
		postCacheService: postCacheService.(interface {
			InvalidateUserPostCache(ctx context.Context, userID string) error
		}),
	}
}

// InvalidateUserCache 清除用户缓存
func (a *cacheInvalidatorAdapter) InvalidateUserCache(ctx context.Context, userID string) error {
	return a.userCacheService.InvalidateUserCache(ctx, userID)
}

// InvalidateAllUserCache 清除所有用户缓存
func (a *cacheInvalidatorAdapter) InvalidateAllUserCache(ctx context.Context) error {
	return a.userCacheService.InvalidateAllUserCache(ctx)
}

// InvalidateUserPostCache 清除用户岗位缓存
func (a *cacheInvalidatorAdapter) InvalidateUserPostCache(ctx context.Context, userID string) error {
	return a.postCacheService.InvalidateUserPostCache(ctx, userID)
}

package system

import (
	"context"

	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// noOpCacheInvalidator 空缓存失效实现（用于不使用缓存的场景）
type noOpCacheInvalidator struct{}

// NewNoOpCacheInvalidator 创建空缓存失效实现
func NewNoOpCacheInvalidator() systemServices.CacheInvalidator {
	return &noOpCacheInvalidator{}
}

// InvalidateUserCache 清除用户缓存（空操作）
func (a *noOpCacheInvalidator) InvalidateUserCache(ctx context.Context, userID string) error {
	return nil
}

// InvalidateAllUserCache 清除所有用户缓存（空操作）
func (a *noOpCacheInvalidator) InvalidateAllUserCache(ctx context.Context) error {
	return nil
}

// InvalidateUserPostCache 清除用户岗位缓存（空操作）
func (a *noOpCacheInvalidator) InvalidateUserPostCache(ctx context.Context, userID string) error {
	return nil
}

// SetupUserRouter 设置用户路由
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建空缓存失效器
	cacheInvalidator := NewNoOpCacheInvalidator()

	// 创建用户服务
	userService := systemServices.NewUserService(
		core.DB.GetDB(),
		core.PwdManager,
		cacheInvalidator,
	)

	// 创建用户处理器
	handler := NewUserHandler(userService)

	// 注册路由
	r.POST("", handler.Create)
	r.POST("/list", handler.List)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/:id/reset-password", handler.ResetPassword)
}

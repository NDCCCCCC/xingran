package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupUserRouter 设置用户路由
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建用户服务（带缓存或无缓存）
	var userService systemServices.UserService
	if core.DataCacheService != nil {
		userService = systemServices.NewUserServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	} else {
		userService = systemServices.NewUserService(
			core.DB.GetDB(),
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	}

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

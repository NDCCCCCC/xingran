package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupUserRouter 设置用户路由
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建适配器
	pwdManager := NewPasswordManagerAdapter(core.PwdManager)
	cacheInvalidator := NewCacheInvalidatorAdapter(core.UserCacheService, core.PostCacheService)

	// 创建用户服务（注入依赖）
	userService := systemServices.NewUserService(
		core.DB.GetDB(),
		pwdManager,
		cacheInvalidator,
	)

	// 创建Handler
	handler := NewUserHandler(userService)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/:id/reset-password", handler.ResetPassword)
	r.POST("/batch", handler.BatchDelete)
}

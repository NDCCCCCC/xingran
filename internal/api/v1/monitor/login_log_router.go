package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// SetupLoginLogRouter 设置登录日志路由
func SetupLoginLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建登录日志服务（注入依赖）
	loginLogService := monitorServices.NewLoginLogService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewLoginLogHandler(loginLogService).WithCore(core)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("/clean", handler.Clean)
	r.POST("/unlock/:username", handler.UnlockUser)
}

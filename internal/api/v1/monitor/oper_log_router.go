package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// SetupOperLogRouter 设置操作日志路由
func SetupOperLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建操作日志服务（注入依赖）
	operLogService := monitorServices.NewOperLogService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewOperLogHandler(operLogService).WithCore(core)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("/clean", handler.Clean)
}

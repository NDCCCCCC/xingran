package scheduler

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	schedulerServices "github.com/xingran-next/xingran-go-backend/internal/services/scheduler"
)

// SetupJobRouter 设置定时任务路由
func SetupJobRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建服务实例
	jobService := schedulerServices.NewJobService(
		core.DB.GetDB(),
		core.Scheduler,
	)
	jobLogService := schedulerServices.NewJobLogService(core.DB.GetDB())

	// 创建Handler实例
	handler := NewJobHandler(jobService, jobLogService).WithCore(core)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/:id/execute", handler.Execute)

	// 日志相关
	r.POST("/logs/list", handler.ListLogs)
	r.POST("/logs/statistics", handler.Statistics)
	r.POST("/logs/clean", handler.CleanLogs)
}

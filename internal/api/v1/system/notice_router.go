package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupNoticeRouter 设置通知公告路由
func SetupNoticeRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建通知缓存服务
	noticeService := getNoticeCacheService(core)
	channelService := systemServices.NewNotificationChannelService(core.DB.GetDB())

	// 创建调度器服务适配器
	var schedulerSvc SchedulerService
	if core.Scheduler != nil {
		schedulerSvc = &schedulerAdapter{scheduler: core.Scheduler}
	}

	// 创建Handler
	noticeHandler := NewNoticeHandler(noticeService, channelService, schedulerSvc).WithCore(core)

	// 通知公告路由
	r.POST("/list", noticeHandler.List)
	r.POST("/statistics", noticeHandler.Statistics)
	r.POST("/batch-delete", noticeHandler.BatchDelete)
	r.GET("/cron-expressions", noticeHandler.GetCronExpressions)
	r.POST("", noticeHandler.Create)
	r.POST("/:id", noticeHandler.GetByID)
	r.POST("/:id/update", noticeHandler.Update)
	r.POST("/:id/delete", noticeHandler.Delete)
	r.GET("/:id/statistics", noticeHandler.GetStatistics)
	r.POST("/:id/publish", noticeHandler.Publish)
	r.POST("/:id/withdraw", noticeHandler.Withdraw)
	r.GET("/:id/channels", noticeHandler.GetChannels)
	r.POST("/:id/channels", noticeHandler.SetChannels)
}

// schedulerAdapter 调度器适配器
type schedulerAdapter struct {
	scheduler interface {
		AddJob(job *models.Job) error
		RemoveJob(jobID string) error
	}
}

// AddJob 添加定时任务
func (a *schedulerAdapter) AddJob(job *models.Job) error {
	return a.scheduler.AddJob(job)
}

// RemoveJob 删除定时任务
func (a *schedulerAdapter) RemoveJob(jobID string) error {
	return a.scheduler.RemoveJob(jobID)
}

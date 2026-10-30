package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// NoticeWriteRouteGroups 通知公告**写操作**路由分组（85-03 RBAC-01）。
// publish/withdraw 为状态变更=编辑语义；SetChannels 同归编辑。
type NoticeWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupNoticeRouter 设置通知公告**读**路由（写操作见 SetupNoticeWriteRouter）。
func SetupNoticeRouter(r *gin.RouterGroup, core *core.Core) {
	noticeHandler := newNoticeHandler(core)

	// 通知公告读路由
	r.POST("/list", noticeHandler.List)
	r.POST("/statistics", noticeHandler.Statistics)
	r.GET("/cron-expressions", noticeHandler.GetCronExpressions)
	r.POST("/:id", noticeHandler.GetByID)
	r.GET("/:id/statistics", noticeHandler.GetStatistics)
	r.GET("/:id/channels", noticeHandler.GetChannels)
}

// SetupNoticeWriteRouter 设置通知公告**写**路由（85-03 RBAC-01 逐动作拆分）。
func SetupNoticeWriteRouter(groups NoticeWriteRouteGroups, core *core.Core) {
	noticeHandler := newNoticeHandler(core)

	groups.Add.POST("", noticeHandler.Create)
	groups.Edit.POST("/:id/update", noticeHandler.Update)
	groups.Edit.POST("/:id/publish", noticeHandler.Publish)
	groups.Edit.POST("/:id/withdraw", noticeHandler.Withdraw)
	groups.Edit.POST("/:id/channels", noticeHandler.SetChannels)
	groups.Remove.POST("/:id/delete", noticeHandler.Delete)
	groups.Remove.POST("/batch-delete", noticeHandler.BatchDelete)
}

// newNoticeHandler 构造通知公告处理器（读/写路由共用）。
func newNoticeHandler(core *core.Core) *NoticeHandler {
	// 创建通知缓存服务
	noticeService := getNoticeCacheService(core)
	channelService := systemServices.NewNotificationChannelService(core.DB.GetDB())

	// 创建调度器服务适配器
	var schedulerSvc SchedulerService
	if core.Scheduler != nil {
		schedulerSvc = &schedulerAdapter{scheduler: core.Scheduler}
	}

	// 创建Handler
	return NewNoticeHandler(noticeService, channelService, schedulerSvc).WithCore(core)
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

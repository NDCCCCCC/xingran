package scheduler

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	schedulerServices "github.com/xingran-next/xingran-go-backend/internal/services/scheduler"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupJobRouter 设置定时任务路由
//
// RBAC 接线（monitor 域权限码全链路）：
//   - 读端点（列表/详情/日志）→ RequirePermissions ANY-of([monitor:job:list, monitor:job:view])
//   - 新增 → strict(monitor:job:add)
//   - 编辑/状态变更 → strict(monitor:job:edit)（UpdateStatus 为启用/停用合一端点，
//     start/stop 语义共用 edit 码，避免一端点双码歧义）
//   - 删除/清空日志 → strict(monitor:job:remove)
//   - 立即执行 → strict(monitor:job:run)
// 端点级中间件（对齐 vdi/vm_router.go 先例）；monitor:job:start/stop/export 码
// 当前无独立端点，保持定义 + F 型播种为后续拆分预留。
func SetupJobRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建服务实例
	jobService := schedulerServices.NewJobService(
		core.DB.GetDB(),
		core.Scheduler,
	)
	jobLogService := schedulerServices.NewJobLogService(core.DB.GetDB())

	// 创建Handler实例
	handler := NewJobHandler(jobService, jobLogService).WithCore(core)

	jobRead := []string{
		string(permission.JobList),
		string(permission.JobView),
	}
	jobAdd := []string{string(permission.JobAdd)}
	jobEdit := []string{string(permission.JobEdit)}
	jobRemove := []string{string(permission.JobRemove)}
	jobRun := []string{string(permission.JobRun)}

	// 注册路由
	r.POST("/list", middleware.RequirePermissions(jobRead, core), handler.List)
	r.POST("", middleware.RequireStrictPermissions(jobAdd, core), handler.Create)
	r.POST("/:id", middleware.RequirePermissions(jobRead, core), handler.GetByID)
	r.POST("/:id/update", middleware.RequireStrictPermissions(jobEdit, core), handler.Update)
	r.POST("/:id/delete", middleware.RequireStrictPermissions(jobRemove, core), handler.Delete)
	r.POST("/:id/status", middleware.RequireStrictPermissions(jobEdit, core), handler.UpdateStatus)
	r.POST("/:id/execute", middleware.RequireStrictPermissions(jobRun, core), handler.Execute)

	// 日志相关
	r.POST("/logs/list", middleware.RequirePermissions(jobRead, core), handler.ListLogs)
	r.POST("/logs/statistics", middleware.RequirePermissions(jobRead, core), handler.Statistics)
	r.POST("/logs/clean", middleware.RequireStrictPermissions(jobRemove, core), handler.CleanLogs)
}

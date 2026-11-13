package workorder

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/services/workorder"
)

// getWorkOrderService 获取工单服务（带缓存或无缓存）
func getWorkOrderService(core *core.Core) workorder.WorkOrderCacheService {
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}
	return workorder.NewWorkOrderServiceWithCache(core.GetDB(), cacheProvider, core.CacheConfigService)
}

// SetupWorkOrdersRouter 设置工单管理路由
func SetupWorkOrdersRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	// 工单读操作（task-3 G1 整改：写端点已拆至下方 strict 写组）
	r.POST("/list", handler.List)
	r.POST("/status-statistics", handler.GetStatusStatistics)
	r.POST("/my-pending", handler.GetMyPending)
	r.POST("/:id", handler.GetByID)

	// 工单评论（读）
	r.POST("/:id/comments/list", handler.GetComments)

	// 工单历史
	r.POST("/:id/history", handler.GetHistory)
}

// WorkOrdersWriteRouteGroups 工单写操作路由分组（task-3 G1：逐动作拆分）。
type WorkOrdersWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupWorkOrdersWriteRouter 设置工单写路由（严格写组接线：逐动作单一权限点）。
//
//	新增            POST /workorder/orders               → ops:workorder:add
//	编辑            POST /workorder/orders/:id/update    → ops:workorder:edit
//	指派            POST /workorder/orders/:id/assign    → ops:workorder:edit
//	指派到今日值班  POST /workorder/orders/:id/assign-duty → ops:workorder:edit
//	状态变更        POST /workorder/orders/:id/status    → ops:workorder:edit
//	添加评论        POST /workorder/orders/:id/comments  → ops:workorder:edit
//	删除            POST /workorder/orders/:id/delete    → ops:workorder:delete
//	批量删除        POST /workorder/orders/batch-delete  → ops:workorder:delete
func SetupWorkOrdersWriteRouter(groups WorkOrdersWriteRouteGroups, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	groups.Add.POST("", handler.Create)
	groups.Edit.POST("/:id/update", handler.Update)
	groups.Edit.POST("/:id/assign", handler.Assign)
	groups.Edit.POST("/:id/assign-duty", handler.AssignToTodayDuty)
	groups.Edit.POST("/:id/status", handler.UpdateStatus)
	groups.Edit.POST("/:id/comments", handler.AddComment)
	groups.Delete.POST("/:id/delete", handler.Delete)
	groups.Delete.POST("/batch-delete", handler.BatchDelete)
}

// SetupWorkOrderCategoriesRouter 设置工单分类路由
func SetupWorkOrderCategoriesRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	// 工单分类读操作（task-3 G1 整改：写端点已拆至下方 strict 写组）
	r.POST("/list", handler.ListCategories)
	r.POST("/enabled", handler.GetEnabledCategories)
	r.POST("/:id", handler.GetCategoryByID)
}

// WorkOrderCategoriesWriteRouteGroups 工单分类写操作路由分组（task-3 G1）。
type WorkOrderCategoriesWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupWorkOrderCategoriesWriteRouter 设置工单分类写路由（严格写组）。
//
//	新增   POST /workorder/categories            → ops:workorder:category:add
//	编辑   POST /workorder/categories/:id/update → ops:workorder:category:edit
//	删除   POST /workorder/categories/:id/delete → ops:workorder:category:delete
func SetupWorkOrderCategoriesWriteRouter(groups WorkOrderCategoriesWriteRouteGroups, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	groups.Add.POST("", handler.CreateCategory)
	groups.Edit.POST("/:id/update", handler.UpdateCategory)
	groups.Delete.POST("/:id/delete", handler.DeleteCategory)
}

// SetupWorkOrderRatingsRouter 设置工单评价路由（FIX-06 / Phase 136 接线）。
//
// 把已实现的 RatingService（internal/services/workorder/rating.go）接入 HTTP：
//   - POST /ratings            创建评价（写，operlog=OperTypeCreate）
//   - POST /ratings/list       查询某工单的评价列表（读）
//   - POST /ratings/statistics 查询某工单的评分统计（读）
//
// 权限：路由组继承 router.go 中 workorder/orders 的 RequirePermissions ANY-of
// （ops:workorder:list/add/edit/delete）；评价 CRUD 的写权限单独由
// router.go 的 ratingsWrite 严格子组挂 ops:workorder:rating:*（见下）。
// 本函数只负责「读路由 + service/handler 构造」，不重复挂权限中间件，
// 避免破坏读组 / 写组兄弟分离（v1.24 Phase 78 #16）。
func SetupWorkOrderRatingsRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewRatingService(core.GetDB())
	handler := NewRatingHandler(service, core)

	// 读路由（继承父组 ANY-of + DataScopePermission，不带严格写权限）
	r.POST("/ratings/list", handler.ListRatings)
	r.POST("/ratings/statistics", handler.GetRatingStatistics)
}

// SetupWorkOrderRatingsWriteRouter 设置工单评价**写**路由（严格单一权限点）。
//
// FIX-06：评价创建属于业务写操作，独立兄弟写组挂
// RequireStrictPermissions(ops:workorder:rating:create)，与读组解耦，
// 不做模块级降级（v1.25 严格权限语义零回归）。
func SetupWorkOrderRatingsWriteRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewRatingService(core.GetDB())
	handler := NewRatingHandler(service, core)

	r.POST("/ratings", handler.CreateRating)
}

// SetupPeriodicWorkOrderRouter 设置周期性工单路由
func SetupPeriodicWorkOrderRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	// 注意：原有路由使用了/templates前缀，这里保持一致
	templates := r.Group("/templates")
	{
		// 读操作（task-3 G1 整改：模板主体写端点已拆至下方 strict 写组）
		templates.POST("/list", handler.ListPeriodic)
		templates.POST("/statistics", handler.GetPeriodicStatistics)
	}
}

// SetupPeriodicWorkOrderActionRouter 设置周期工单模板**动作**路由（逐动作严格权限）。
//
// RBAC-EXC-01 已清偿（2026-11-13，见 scripts/RBAC-EXCEPTIONS.md）：
// 此前 enable/disable/generate/logs 4 条端点挂在 ANY-of 读组，仅持只读 list 码的
// 角色也能调用状态变更 / 生成工单等写操作，违反 #87 最小权限口径。
// 现按逐动作口径拆独立权限码并挂 RequireStrictPermissions：
//
//	启用     POST /workorder/periodic/templates/:id/enable   → ops:workorder:periodic:enable
//	停用     POST /workorder/periodic/templates/:id/disable  → ops:workorder:periodic:disable
//	立即生成 POST /workorder/periodic/templates/:id/generate → ops:workorder:periodic:generate
//	执行记录 POST /workorder/periodic/templates/:id/logs     → ops:workorder:periodic:logs（只读语义独立码）
//
// 历史事故保护（scripts/FIX-PROTOCOL.md）：这 4 条端点曾是 PR #85 误判「废弃」
// 删掉、1.5h 后 #86 恢复的对象。迁移挂载面必须与权限码播种、前端门控同步落地，
// 否则非超管角色会 403（等价于「用户可见功能消失」）。三者缺一不得迁移。
func SetupPeriodicWorkOrderActionRouter(groups PeriodicWorkOrderActionRouteGroups, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	groups.Enable.POST("/:id/enable", handler.EnablePeriodic)
	groups.Disable.POST("/:id/disable", handler.DisablePeriodic)
	groups.Generate.POST("/:id/generate", handler.GeneratePeriodic)
	groups.Logs.POST("/:id/logs", handler.GetPeriodicLogs)
}

// PeriodicWorkOrderActionRouteGroups 周期工单模板动作路由分组（逐动作严格权限）。
type PeriodicWorkOrderActionRouteGroups struct {
	Enable   *gin.RouterGroup
	Disable  *gin.RouterGroup
	Generate *gin.RouterGroup
	Logs     *gin.RouterGroup
}

// PeriodicWorkOrderWriteRouteGroups 周期工单模板写操作路由分组（task-3 G1）。
type PeriodicWorkOrderWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupPeriodicWorkOrderWriteRouter 设置周期工单模板写路由（严格写组）。
// 仅模板主体 CRUD 在此组；enable/disable/generate/logs 4 条动作端点已随
// RBAC-EXC-01 清偿迁至 SetupPeriodicWorkOrderActionRouter 的逐动作严格组。
//
//	新增   POST /workorder/periodic/templates            → ops:workorder:periodic:add
//	编辑   POST /workorder/periodic/templates/:id/update → ops:workorder:periodic:edit
//	删除   POST /workorder/periodic/templates/:id/delete → ops:workorder:periodic:delete
func SetupPeriodicWorkOrderWriteRouter(groups PeriodicWorkOrderWriteRouteGroups, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	// 传入的各动作组由 router.go 以 workorder.Group("/periodic/templates") 兄弟组创建
	groups.Add.POST("", handler.CreatePeriodic)
	groups.Edit.POST("/:id/update", handler.UpdatePeriodic)
	groups.Delete.POST("/:id/delete", handler.DeletePeriodic)
}

// SetupWorkOrderConfigRouter 设置工单配置路由（读端点：POST "" 读取配置）。
// task-3 G1 整改：写端点 /update 拆至 SetupWorkOrderConfigWriteRouter 严格兄弟组。
func SetupWorkOrderConfigRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("", handler.GetConfig)
}

// SetupWorkOrderConfigWriteRouter 设置工单配置写路由（严格写组）。
// 沿用 system:config 单码（与 duty config 同口径，不新增码）。
func SetupWorkOrderConfigWriteRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("/update", handler.UpdateConfig)
}

// SetupWorkOrderStatisticsRouter 设置工单统计路由
func SetupWorkOrderStatisticsRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("", handler.GetStatistics)
}

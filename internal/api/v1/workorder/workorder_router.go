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

	// 工单基础操作
	r.POST("/list", handler.List)
	r.POST("/status-statistics", handler.GetStatusStatistics)
	r.POST("/my-pending", handler.GetMyPending)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)

	// 工单分配与状态
	r.POST("/:id/assign", handler.Assign)
	r.POST("/:id/assign-duty", handler.AssignToTodayDuty)
	r.POST("/:id/status", handler.UpdateStatus)

	// 工单评论
	r.POST("/:id/comments/list", handler.GetComments)
	r.POST("/:id/comments", handler.AddComment)

	// 工单历史
	r.POST("/:id/history", handler.GetHistory)
}

// SetupWorkOrderCategoriesRouter 设置工单分类路由
func SetupWorkOrderCategoriesRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("/list", handler.ListCategories)
	r.POST("/enabled", handler.GetEnabledCategories)
	r.POST("/:id", handler.GetCategoryByID)
	r.POST("", handler.CreateCategory)
	r.POST("/:id/update", handler.UpdateCategory)
	r.POST("/:id/delete", handler.DeleteCategory)
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
		templates.POST("/list", handler.ListPeriodic)
		templates.POST("/statistics", handler.GetPeriodicStatistics)
		templates.POST("", handler.CreatePeriodic)
		templates.POST("/:id/update", handler.UpdatePeriodic)
		templates.POST("/:id/delete", handler.DeletePeriodic)

		// 历史注释、已废弃：以下 5 行为早期函数式 handler 路由（getPeriodicTemplate / enablePeriodicTemplate 等），均已注释、未启用；现行周期工单路由由上方 templates.POST(... → handler.* 方法) 注册。保留作追溯，勿据此判断为待办或可启用实现。
		// templates.POST("/:id", getPeriodicTemplate(core))
		// templates.POST("/:id/enable", enablePeriodicTemplate(core))
		// templates.POST("/:id/disable", disablePeriodicTemplate(core))
		// templates.POST("/:id/generate", generateWorkOrderNow(core))
		// templates.POST("/:id/logs", getPeriodicLogs(core))
	}
}

// SetupWorkOrderConfigRouter 设置工单配置路由
func SetupWorkOrderConfigRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("", handler.GetConfig)
	r.POST("/update", handler.UpdateConfig)
}

// SetupWorkOrderStatisticsRouter 设置工单统计路由
func SetupWorkOrderStatisticsRouter(r *gin.RouterGroup, core *core.Core) {
	service := getWorkOrderService(core)
	handler := NewWorkOrderHandler(service).WithCore(core)

	r.POST("", handler.GetStatistics)
}

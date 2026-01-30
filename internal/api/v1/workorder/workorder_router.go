package workorder

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/workorder"
)

// SetupWorkOrdersRouter 设置工单管理路由
func SetupWorkOrdersRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewWorkOrderService(core.DB.GetDB())
	handler := NewWorkOrderHandler(service)

	// 工单基础操作
	r.POST("/list", handler.List)
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
	service := workorder.NewWorkOrderService(core.DB.GetDB())
	handler := NewWorkOrderHandler(service)

	r.POST("/list", handler.ListCategories)
	r.POST("/enabled", handler.GetEnabledCategories)
	r.POST("/:id", handler.GetCategoryByID)
	r.POST("", handler.CreateCategory)
	r.POST("/:id/update", handler.UpdateCategory)
	r.POST("/:id/delete", handler.DeleteCategory)
}

// SetupWorkOrderRatingsRouter 设置工单评价路由
// 注意：评价路由暂时保留原有实现，因为Handler中未包含评价功能
func SetupWorkOrderRatingsRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 评价功能需要在WorkOrderHandler中添加
	// 暂时保留原有函数式handler
}

// SetupPeriodicWorkOrderRouter 设置周期性工单路由
func SetupPeriodicWorkOrderRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewWorkOrderService(core.DB.GetDB())
	handler := NewWorkOrderHandler(service)

	// 注意：原有路由使用了/templates前缀，这里保持一致
	templates := r.Group("/templates")
	{
		templates.POST("/list", handler.ListPeriodic)
		templates.POST("", handler.CreatePeriodic)
		templates.POST("/:id/update", handler.UpdatePeriodic)
		templates.POST("/:id/delete", handler.DeletePeriodic)

		// 以下路由暂时保留原有实现
		// templates.POST("/:id", getPeriodicTemplate(core))
		// templates.POST("/:id/enable", enablePeriodicTemplate(core))
		// templates.POST("/:id/disable", disablePeriodicTemplate(core))
		// templates.POST("/:id/generate", generateWorkOrderNow(core))
		// templates.POST("/:id/logs", getPeriodicLogs(core))
	}
}

// SetupWorkOrderConfigRouter 设置工单配置路由
func SetupWorkOrderConfigRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewWorkOrderService(core.DB.GetDB())
	handler := NewWorkOrderHandler(service)

	r.POST("", handler.GetConfig)
	r.POST("/update", handler.UpdateConfig)
}

// SetupWorkOrderStatisticsRouter 设置工单统计路由
func SetupWorkOrderStatisticsRouter(r *gin.RouterGroup, core *core.Core) {
	service := workorder.NewWorkOrderService(core.DB.GetDB())
	handler := NewWorkOrderHandler(service)

	r.POST("", handler.GetStatistics)
}

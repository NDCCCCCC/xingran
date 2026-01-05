package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupWorkOrdersRouter 设置工单管理路由
func SetupWorkOrdersRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listWorkOrders(core))
	r.POST("/my-pending", getMyPendingWorkOrders(core))
	r.POST("", createWorkOrder(core))
	r.POST("/:id", getWorkOrder(core))
	r.POST("/:id/update", updateWorkOrder(core))
	r.POST("/:id/delete", deleteWorkOrder(core))
	r.POST("/batch-delete", batchDeleteWorkOrders(core))
	r.POST("/:id/assign", assignWorkOrder(core))
	r.POST("/:id/assign-duty", assignToTodayDuty(core))
	r.POST("/:id/status", updateWorkOrderStatus(core))
	r.POST("/:id/comments/list", getWorkOrderComments(core))
	r.POST("/:id/comments", addWorkOrderComment(core))
	r.POST("/:id/history", getWorkOrderHistory(core))
}

// SetupWorkOrderCategoriesRouter 设置工单分类路由
func SetupWorkOrderCategoriesRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listWorkOrderCategories(core))
	r.POST("/enabled", getEnabledCategories(core))
	r.POST("/:id", getWorkOrderCategory(core))
	r.POST("", createWorkOrderCategory(core))
	r.POST("/:id/update", updateWorkOrderCategory(core))
	r.POST("/:id/delete", deleteWorkOrderCategory(core))
}

// SetupWorkOrderRatingsRouter 设置工单评价路由
func SetupWorkOrderRatingsRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/:id/rating", createWorkOrderRating(core))
	r.POST("/:id/ratings", getWorkOrderRatings(core))
	r.POST("/:id/rating-statistics", getRatingStatistics(core))
}

// SetupPeriodicWorkOrderRouter 设置周期性工单路由
func SetupPeriodicWorkOrderRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/templates/list", listPeriodicTemplates(core))
	r.POST("/templates/:id", getPeriodicTemplate(core))
	r.POST("/templates", createPeriodicTemplate(core))
	r.POST("/templates/:id/update", updatePeriodicTemplate(core))
	r.POST("/templates/:id/delete", deletePeriodicTemplate(core))
	r.POST("/templates/:id/enable", enablePeriodicTemplate(core))
	r.POST("/templates/:id/disable", disablePeriodicTemplate(core))
	r.POST("/templates/:id/generate", generateWorkOrderNow(core))
	r.POST("/templates/:id/logs", getPeriodicLogs(core))
}

// SetupWorkOrderConfigRouter 设置工单配置路由
func SetupWorkOrderConfigRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("", getWorkOrderConfig(core))
	r.POST("/update", updateWorkOrderConfig(core))
}

// SetupWorkOrderStatisticsRouter 设置工单统计路由
func SetupWorkOrderStatisticsRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("", getWorkOrderStatistics(core))
}

package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDashboardRouter 设置仪表盘路由
// 注册仪表盘系统的所有API端点
func SetupDashboardRouter(r *gin.RouterGroup, core *core.Core) {
	// 仪表盘 CRUD 操作
	r.POST("/list", getDashboardList(core))
	r.POST("", createDashboard(core))
	r.GET("/:id", getDashboard(core))
	r.POST("/:id/update", updateDashboard(core))
	r.DELETE("/:id", deleteDashboard(core))
	r.POST("/:id/duplicate", duplicateDashboard(core))
	r.POST("/:id/set-default", setDefaultDashboard(core))

	// 仪表盘模板操作
	r.POST("/templates", getTemplates(core))
	r.POST("/templates/:id/create", createFromTemplate(core))

	// 仪表盘版本操作
	r.GET("/:id/versions", getVersions(core))
	r.POST("/:id/versions", createVersion(core))
	r.POST("/:id/versions/:versionId/restore", restoreVersion(core))

	// 仪表盘导入导出
	r.GET("/:id/export", exportDashboard(core))
	r.POST("/import", importDashboard(core))

	// Widget 数据获取
	r.POST("/widgets/:id/data", getWidgetData(core))
	r.POST("/widgets/batch-data", getBatchWidgetData(core))

	// API端点元数据
	r.GET("/endpoints", getAvailableEndpoints(core))
	r.GET("/endpoints/validate", validateEndpoint(core))
	r.GET("/endpoints/filter", getUserEndpointsWithFilter(core))
	r.POST("/endpoints/cache/invalidate", invalidateEndpointCache(core))
}

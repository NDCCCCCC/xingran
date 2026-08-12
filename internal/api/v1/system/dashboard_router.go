package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupDashboardRouter 设置仪表盘路由
func SetupDashboardRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建仪表盘服务（注入依赖）
	dashboardService := systemServices.NewDashboardService(
		core.DB.GetDB(),
		core.Cache,
		core.APIEndpointService,
	)

	// 创建Handler
	handler := NewDashboardHandler(dashboardService).WithCore(core)

	// 注册路由（注意：更具体的路由要放在前面，避免被通用路由匹配）
	// 仪表盘默认入口
	r.GET("/default", handler.GetDefault) // 获取默认仪表盘

	// 仪表盘 CRUD 操作
	r.POST("/list", handler.List)                  // 获取仪表盘列表
	r.POST("", handler.Create)                     // 创建仪表盘
	r.GET("/:id", handler.GetByID)                 // 获取仪表盘详情
	r.POST("/:id/update", handler.Update)          // 更新仪表盘
	r.DELETE("/:id", handler.Delete)               // 删除仪表盘
	r.POST("/:id/duplicate", handler.Duplicate)    // 复制仪表盘
	r.POST("/:id/set-default", handler.SetDefault) // 设置默认仪表盘

	// 仪表盘模板操作
	r.POST("/templates", handler.GetTemplates)                  // 获取仪表盘模板列表
	r.POST("/templates/:id/create", handler.CreateFromTemplate) // 从模板创建仪表盘

	// 仪表盘版本操作
	r.GET("/:id/versions", handler.GetVersions)                        // 获取仪表盘版本历史
	r.POST("/:id/versions", handler.CreateVersion)                     // 创建版本快照
	r.POST("/:id/versions/:versionId/restore", handler.RestoreVersion) // 从版本恢复仪表盘

	// 仪表盘导入导出
	r.GET("/:id/export", handler.Export) // 导出仪表盘配置
	r.POST("/import", handler.Import)    // 导入仪表盘配置

	// Widget 数据获取
	r.POST("/widgets/:id/data", handler.GetWidgetData)        // 获取 Widget 数据
	r.POST("/widgets/batch-data", handler.GetBatchWidgetData) // 批量获取 Widget 数据

	// API端点元数据
	r.GET("/endpoints", handler.GetAvailableEndpoints)                     // 获取可用的API端点列表
	r.GET("/endpoints/validate", handler.ValidateEndpoint)                 // 验证API端点配置
	r.GET("/endpoints/filter", handler.GetUserEndpointsWithFilter)         // 获取过滤后的端点列表
	r.POST("/endpoints/cache/invalidate", handler.InvalidateEndpointCache) // 清除用户端点缓存
}

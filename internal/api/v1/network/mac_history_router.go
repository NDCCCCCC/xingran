package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	mac_history_query_service "github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupMACHistoryRouter 设置MAC历史查询路由
func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建查询服务
	historyQueryService := mac_history_query_service.NewMACHistoryQueryService(core.GetDB())

	// 创建处理器
	historyHandler := NewMACHistoryHandler(historyQueryService)

	// Phase 15 PERF-04: 热力图 handler + 路由
	// Phase 103 CONV-01 (D-103-20): 接线改为 system.NewCacheProvider 注入 base.CacheProvider
	heatmapService := mac_history_query_service.NewMACHistoryHeatmapService(
		core.GetDB(),
		system.NewCacheProvider(core.DataCacheService),
		core.CacheConfigService,
	)
	heatmapHandler := NewMACHistoryHeatmapHandler(heatmapService)

	// 注册路由
	//
	// RBAC 接线（task-4）：历史查询全部为读语义 → ANY-of([network:mac:query])；
	// GET /history/list 为导出 → strict(network:mac:query)（FIX-03 导出惯例）。
	macHistoryRead := []string{string(permission.NetworkMacQuery)}
	r.POST("/history/port", middleware.RequirePermissions(macHistoryRead, core), historyHandler.QueryPortHistory)
	r.POST("/history/device", middleware.RequirePermissions(macHistoryRead, core), historyHandler.QueryDeviceHistory)
	r.POST("/history/stats", middleware.RequirePermissions(macHistoryRead, core), historyHandler.QueryConnectionStats)
	r.POST("/history/vendor", middleware.RequirePermissions(macHistoryRead, core), historyHandler.GetVendor)
	r.POST("/history/list", middleware.RequirePermissions(macHistoryRead, core), historyHandler.QueryHistory)
	r.GET("/history/list", middleware.RequireStrictPermissions(macHistoryRead, core), historyHandler.ExportHistory)
	r.POST("/history/heatmap", middleware.RequirePermissions(macHistoryRead, core), heatmapHandler.QueryHeatmap)

	applogger.Infof("[路由注册] MAC历史查询路由已注册: /history/port, /history/device, /history/stats, /history/vendor, /history/list (POST/GET), /history/heatmap")
}

package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	mac_history_query_service "github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// SetupMACHistoryRouter 设置MAC历史查询路由
func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建查询服务
	historyQueryService := mac_history_query_service.NewMACHistoryQueryService(core.GetDB())

	// 创建处理器
	historyHandler := NewMACHistoryHandler(historyQueryService)

	// Phase 15 PERF-04: 热力图 handler + 路由
	heatmapService := mac_history_query_service.NewMACHistoryHeatmapService(core.GetDB(), nil, nil)
	heatmapHandler := NewMACHistoryHeatmapHandler(heatmapService)

	// 注册路由
	r.POST("/history/port", historyHandler.QueryPortHistory)
	r.POST("/history/device", historyHandler.QueryDeviceHistory)
	r.POST("/history/stats", historyHandler.QueryConnectionStats)
	r.POST("/history/vendor", historyHandler.GetVendor)
	r.POST("/history/list", historyHandler.QueryHistory)
	r.GET("/history/list", historyHandler.ExportHistory)
	r.POST("/history/heatmap", heatmapHandler.QueryHeatmap)

	applogger.Infof("[路由注册] MAC历史查询路由已注册: /history/port, /history/device, /history/stats, /history/vendor, /history/list (POST/GET), /history/heatmap")
}

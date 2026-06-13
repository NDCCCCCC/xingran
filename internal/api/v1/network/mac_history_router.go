package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	mac_history_query_service "github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// SetupMACHistoryRouter 设置MAC历史查询路由
func SetupMACHistoryRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建查询服务
	historyQueryService := mac_history_query_service.NewMACHistoryQueryService(core.GetDB())

	// 创建处理器
	historyHandler := NewMACHistoryHandler(historyQueryService)

	// 注册路由
	r.POST("/history/port", historyHandler.QueryPortHistory)
	r.POST("/history/device", historyHandler.QueryDeviceHistory)
	r.POST("/history/trajectory", historyHandler.QueryTrajectory)
	r.POST("/history/stats", historyHandler.QueryConnectionStats)

	applogger.Infof("[路由注册] MAC历史查询路由已注册: /history/port, /history/device, /history/trajectory, /history/stats")
}

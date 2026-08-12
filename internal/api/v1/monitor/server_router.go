package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// SetupServerRouter 设置服务器监控路由
func SetupServerRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建适配器
	provider := NewMetricsProviderAdapter(core)

	// 创建Service
	serverService := monitorServices.NewServerService(core.DB.GetDB(), provider)

	// 创建Handler
	handler := NewServerHandler(serverService).WithCore(core)

	// 注册路由
	r.POST("/server-info/list", handler.GetServerInfo)
	r.POST("/server-metrics/current", handler.GetCurrentServerMetrics)
	r.POST("/server-metrics/save", handler.SaveSystemMetrics)
	r.POST("/server-metrics/history", handler.GetSystemMetricsHistory)
}

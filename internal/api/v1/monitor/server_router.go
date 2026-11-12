package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupServerRouter 设置服务器监控路由
//
// RBAC 接线（monitor 域权限码全链路）：
//   - 读端点 → RequirePermissions ANY-of([monitor:server:list, monitor:server:view])
//   - 写端点（指标保存）→ RequireStrictPermissions 单一权限点 monitor:server:view
//
// 采用端点级中间件（对齐 vdi/vm_router.go 先例）：本 setup 全量注册读/写端点，
// 无法在 router.go 侧拆读/写兄弟组，端点级单中间件不存在 ANY-of 前置放行病灶。
func SetupServerRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建适配器
	provider := NewMetricsProviderAdapter(core)

	// 创建Service
	serverService := monitorServices.NewServerService(core.DB.GetDB(), provider)

	// 创建Handler
	handler := NewServerHandler(serverService).WithCore(core)

	serverRead := []string{
		string(permission.ServerList),
		string(permission.ServerView),
	}
	serverSave := []string{string(permission.ServerView)}

	// 注册路由
	r.POST("/server-info/list", middleware.RequirePermissions(serverRead, core), handler.GetServerInfo)
	r.POST("/server-metrics/current", middleware.RequirePermissions(serverRead, core), handler.GetCurrentServerMetrics)
	r.POST("/server-metrics/save", middleware.RequireStrictPermissions(serverSave, core), handler.SaveSystemMetrics)
	r.POST("/server-metrics/history", middleware.RequirePermissions(serverRead, core), handler.GetSystemMetricsHistory)
}

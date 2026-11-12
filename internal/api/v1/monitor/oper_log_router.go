package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupOperLogRouter 设置操作日志路由
//
// RBAC 接线（monitor 域权限码全链路）：
//   - 读端点 → RequirePermissions ANY-of([monitor:operlog:list, monitor:operlog:view])
//   - 写端点（删除/批量删除/清空）→ RequireStrictPermissions monitor:operlog:remove
//
// 说明：config.go 中的 monitor:operlog:export 当前无对应端点（前端亦未调用），
// 码保持定义 + F 型播种，为后续导出端点预留（2026-11-11 monitor RBAC 审计裁定）。
func SetupOperLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建操作日志服务（注入依赖）
	operLogService := monitorServices.NewOperLogService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewOperLogHandler(operLogService).WithCore(core)

	operLogRead := []string{
		string(permission.OperLogList),
		string(permission.OperLogView),
	}
	operLogRemove := []string{string(permission.OperLogRemove)}

	// 注册路由
	r.POST("/list", middleware.RequirePermissions(operLogRead, core), handler.List)
	r.POST("/:id", middleware.RequirePermissions(operLogRead, core), handler.GetByID)
	r.POST("/:id/delete", middleware.RequireStrictPermissions(operLogRemove, core), handler.Delete)
	r.POST("/batch-delete", middleware.RequireStrictPermissions(operLogRemove, core), handler.BatchDelete)
	r.POST("/clean", middleware.RequireStrictPermissions(operLogRemove, core), handler.Clean)
}

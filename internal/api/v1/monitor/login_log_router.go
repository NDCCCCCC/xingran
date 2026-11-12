package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupLoginLogRouter 设置登录日志路由
//
// RBAC 接线（monitor 域权限码全链路）：
//   - 读端点 → RequirePermissions ANY-of([monitor:loginlog:list, monitor:loginlog:view])
//   - 写端点（删除/批量删除/清空）→ RequireStrictPermissions monitor:loginlog:remove
//   - 账号解锁 → RequireStrictPermissions system:config（管理面语义，与 router.go
//     userUnlock 组先例一致：解锁是跨模块账号管理动作，不属登录日志域）
func SetupLoginLogRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建登录日志服务（注入依赖）
	loginLogService := monitorServices.NewLoginLogService(
		core.DB.GetDB(),
	)

	// 创建Handler
	handler := NewLoginLogHandler(loginLogService).WithCore(core)

	loginLogRead := []string{
		string(permission.LoginLogList),
		string(permission.LoginLogView),
	}
	loginLogRemove := []string{string(permission.LoginLogRemove)}
	unlockPerm := []string{"system:config"}

	// 注册路由
	r.POST("/list", middleware.RequirePermissions(loginLogRead, core), handler.List)
	r.POST("/:id", middleware.RequirePermissions(loginLogRead, core), handler.GetByID)
	r.POST("/:id/delete", middleware.RequireStrictPermissions(loginLogRemove, core), handler.Delete)
	r.POST("/batch-delete", middleware.RequireStrictPermissions(loginLogRemove, core), handler.BatchDelete)
	r.POST("/clean", middleware.RequireStrictPermissions(loginLogRemove, core), handler.Clean)
	r.POST("/unlock/:username", middleware.RequireStrictPermissions(unlockPerm, core), handler.UnlockUser)
}

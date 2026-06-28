package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
)

// SetupReconciliationExceptionRouter 设置资产对账例外规则路由
//
// R1 端点(只读):
//   POST /exception-rule/list   -> handler.ListRules
//   POST /exception-rule/:id    -> handler.GetRuleByID
//
// R3 扩展端点(Phase 44 / Plan 44-01 Task 5):
//   POST /exception-rule/create         -> handler.CreateRule    (RequirePermissions asset:reconciliation:exception:create)
//   POST /exception-rule/:id/update     -> handler.UpdateRule    (RequirePermissions asset:reconciliation:exception:update)
//   POST /exception-rule/:id/delete     -> handler.DeleteRule    (RequirePermissions asset:reconciliation:exception:delete)
//   POST /exception-rule/test           -> handler.TestRule      (RequirePermissions asset:reconciliation:exception:test)
//
// 权限粒度参考 44-CONTEXT.md(T-44-02 越权创建例外规则缓解):
//   - 4 个新端点全部 RequirePermissions,仅持权限角色可调
//   - list/:id 路由现有无权限中间件(R1 skeleton),R3 保持现状
//     参照项目记忆 ruoyi-perm-namespace-split-readonly-page 教训:读路径可放宽避免误锁只读场景
func SetupReconciliationExceptionRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationExceptionService(core.DB.GetDB())
	handler := NewReconciliationExceptionHandler(svc).WithCore(core)

	// R1 只读端点(无 RequirePermissions,放宽读路径)
	r.POST("/exception-rule/list", handler.ListRules)
	r.POST("/exception-rule/:id", handler.GetRuleByID)

	// R3 写端点 + 命中测试(RequirePermissions 强制权限,44-CONTEXT.md 锁定命名空间)
	r.POST("/exception-rule/create",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core),
		handler.CreateRule)
	r.POST("/exception-rule/:id/update",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:update"}, core),
		handler.UpdateRule)
	r.POST("/exception-rule/:id/delete",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:delete"}, core),
		handler.DeleteRule)
	r.POST("/exception-rule/test",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:test"}, core),
		handler.TestRule)
}

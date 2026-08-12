package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
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
// Phase 44 R3 / Plan 44-02 Task 3 新增(降噪基线):
//   POST /baseline/snapshot             -> handler.SnapshotBaseline (写 sys_config, 调 operlog OperTypeUpdate)
//   POST /baseline/compare              -> handler.CompareBaseline  (读 sys_config + COUNT, 不调 operlog)
//
// 权限粒度参考 44-CONTEXT.md(T-44-02 越权创建例外规则缓解):
//   - 4 个新端点全部 RequirePermissions,仅持权限角色可调
//   - list/:id 路由现有无权限中间件(R1 skeleton),R3 保持现状
//     参照项目记忆 xingran-perm-namespace-split-readonly-page 教训:读路径可放宽避免误锁只读场景
//   - baseline 端点 RequirePermissions 闭合权限闭环(CR-03 修复):snapshot=exception:create(写),
//     compare=reconciliation:list(读,模块标准读权限)。verifier 原 suggestion exception:list 未 seed,
//     改用已 seed 的 reconciliation:list 避免误锁 dashboard 卡片。
func SetupReconciliationExceptionRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationExceptionService(core.DB.GetDB())
	baselineSvc := asset.NewReconciliationBaselineService(core.DB.GetDB())
	handler := NewReconciliationExceptionHandler(svc).
		WithCore(core).
		WithBaselineService(baselineSvc)

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

	// Phase 44 R3 — 降噪基线端点(CR-03 修复:写端点必须 RequirePermissions,闭合 SC 2/AUDIT-01 权限闭环)
	// SnapshotBaseline 是写操作(覆盖 sys_config 中 R2 末期基线 + 写 operlog OperTypeUpdate),
	//   任意认证用户可覆盖会污染 SC 8 ≥60% 降噪分母 → RequirePermissions(create 权限,与 import 一致,admin 持)。
	// CompareBaseline 是读操作(sys_config + COUNT) → RequirePermissions(asset:reconciliation:list,
	//   模块标准读权限,与 statistics/summary/exception-list 一致;verifier 建议 exception:list 未 seed,
	//   改用已 seed 的 reconciliation:list,避免误锁 dashboard)。
	r.POST("/baseline/snapshot",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core),
		handler.SnapshotBaseline)
	r.POST("/baseline/compare",
		middleware.RequirePermissions([]string{"asset:reconciliation:list"}, core),
		handler.CompareBaseline)

	// Phase 44 R3 / Plan 44-02 Task 4 — Excel 导入/导出/模板(SC 9)
	//
	// 方案 B 专路由(WARN-7 锁定),避免项目记忆 xingran-excel-import-route-conflict
	// (不在 router.go 预注册 /asset/reconciliation/import,由本 router 自管)。
	//
	// 权限:
	//   - import 是写操作 → RequirePermissions asset:reconciliation:exception:create
	//     (与 /exception-rule/create 同权限, admin 默认持)
	//   - export/template 是读 + 下载 → 放宽 (与 list/:id 一致, audit-only 场景)
	r.POST("/exception-rule/import",
		middleware.RequirePermissions([]string{"asset:reconciliation:exception:create"}, core),
		handler.ImportRules)
	r.POST("/exception-rule/export", handler.ExportRules)
	r.POST("/exception-rule/template", handler.DownloadTemplate)
}

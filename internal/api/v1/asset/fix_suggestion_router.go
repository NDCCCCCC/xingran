package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupFixSuggestionRouter 设置资产对账修复建议路由
//
// 端点(命名空间 /asset/reconciliation/fix-suggestion/*):
//
//	POST /fix-suggestion/list         → ListFixSuggestions  (asset:reconciliation:fix:list)
//	POST /fix-suggestion/:id          → GetByID             (asset:reconciliation:fix:list)
//	POST /fix-suggestion/:id/accept   → Accept              (asset:reconciliation:fix:accept)
//	POST /fix-suggestion/:id/reject   → Reject              (asset:reconciliation:fix:reject)
//	POST /fix-suggestion/:id/apply    → Apply               (asset:reconciliation:fix:apply)
//	POST /fix-suggestion/:id/rollback → Rollback            (asset:reconciliation:fix:rollback)
//	POST /fix-suggestion/stats        → Stats               (asset:reconciliation:fix:stats)
//
// D-D3 锁定:仅单条接受(不批量),每个端点都有 1 个独立 perm
func SetupFixSuggestionRouter(r *gin.RouterGroup, core *core.Core) {
	// 依赖注入:D-A3 sys_config + D-C5 noticeHub
	configSvc := system.NewConfigService(core.DB.GetDB())
	svc := asset.NewFixSuggestionService(core.DB.GetDB(), core.Cache, configSvc, core.NoticeHub)
	handler := NewFixSuggestionHandler(svc).WithCore(core)

	// 读端点（task-3 G2 整改：accept/reject/apply/rollback 已拆至下方 strict 写组）
	r.POST("/fix-suggestion/list", handler.ListFixSuggestions)
	r.POST("/fix-suggestion/:id", handler.GetByID)
	r.POST("/fix-suggestion/stats", handler.Stats)
}

// FixSuggestionWriteRouteGroups 修复建议写操作路由分组（task-3 G2：逐动作拆分）。
type FixSuggestionWriteRouteGroups struct {
	Accept   *gin.RouterGroup
	Reject   *gin.RouterGroup
	Apply    *gin.RouterGroup
	Rollback *gin.RouterGroup
}

// SetupFixSuggestionWriteRouter 设置修复建议写路由（task-3 G2 整改）。
// 4 个动作端点由 router.go 的 strict 兄弟组逐动作挂 RequireStrictPermissions
// 单一权限点（fix:accept / fix:reject / fix:apply / fix:rollback）——
// 此前整组 9 码平铺 ANY-of，持 list 即可调 apply/rollback。
func SetupFixSuggestionWriteRouter(groups FixSuggestionWriteRouteGroups, core *core.Core) {
	configSvc := system.NewConfigService(core.DB.GetDB())
	svc := asset.NewFixSuggestionService(core.DB.GetDB(), core.Cache, configSvc, core.NoticeHub)
	handler := NewFixSuggestionHandler(svc).WithCore(core)

	groups.Accept.POST("/fix-suggestion/:id/accept", handler.Accept)
	groups.Reject.POST("/fix-suggestion/:id/reject", handler.Reject)
	groups.Apply.POST("/fix-suggestion/:id/apply", handler.Apply)
	groups.Rollback.POST("/fix-suggestion/:id/rollback", handler.Rollback)
}

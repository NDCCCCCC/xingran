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

	r.POST("/fix-suggestion/list", handler.ListFixSuggestions)
	r.POST("/fix-suggestion/:id", handler.GetByID)
	r.POST("/fix-suggestion/:id/accept", handler.Accept)
	r.POST("/fix-suggestion/:id/reject", handler.Reject)
	r.POST("/fix-suggestion/:id/apply", handler.Apply)
	r.POST("/fix-suggestion/:id/rollback", handler.Rollback)
	r.POST("/fix-suggestion/stats", handler.Stats)
}

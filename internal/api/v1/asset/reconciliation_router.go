package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
)

// SetupReconciliationRouter 设置资产对账(异常列表)读端点路由
//
// R1 端点 (D-18: R1 不上"标记已解决" UI):
//   POST /exception/list          -> handler.ListExceptions
//   POST /exception/:id           -> handler.GetExceptionByID
//
// R2 端点(Phase 43 / D-A4-04 兑现 D-18 推迟的 UI):
//   POST /exception/:id/resolve   -> handler.ResolveException
//   - 标记异常为已解决(SET resolved_at/resolved_by/resolution_note)
//   - 写 operlog(OperTypeUpdate) → sys_oper_log
//   - 不联动 workorder 关闭(workorder 独立在 workorder UI 关闭)
//   - 不触发重检(7d 静默期已兜底)
//
// 本函数是 reconciliation 读端点入口,42-04 会再 SetupReconciliationStatisticsRouter;
// 42-06 整合到 internal/api/router.go 主路由(避免 Excel 导入路由冲突陷阱)。
func SetupReconciliationRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationService(core.DB.GetDB())
	handler := NewReconciliationHandler(svc).WithCore(core)

	r.POST("/exception/list", handler.ListExceptions)
	r.POST("/exception/:id", handler.GetExceptionByID)
	r.POST("/exception/:id/resolve", handler.ResolveException)
}

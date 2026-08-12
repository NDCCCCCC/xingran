package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
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
// R4 端点(Phase 45 / D-A4-01/02/03):
//   POST /by-workstation          -> handler.GetByWorkstation
//   - 工位对账健康度聚合(Workstation/HealthScore/Assets/Visible 一次拉完,避免 N+1,SC7)
//   - 缓存 TTL=5min 与 R1 MV 刷新一致
//   - Visible 字段由 handler 注入(基于 HasUserPermission 静默降级,D-A1-03)
//
// 本函数是 reconciliation 读端点入口,42-04 会再 SetupReconciliationStatisticsRouter;
// 42-06 整合到 internal/api/router.go 主路由(避免 Excel 导入路由冲突陷阱)。
func SetupReconciliationRouter(r *gin.RouterGroup, core *core.Core) {
	// R4:Cache 注入支持 GetByWorkstation 的 5min TTL 缓存
	// R4 (Phase 45 / D-A4-02):exceptionSvc 注入用于 per-asset exception rule 命中
	exceptionSvc := asset.NewReconciliationExceptionService(core.DB.GetDB())
	svc := asset.NewReconciliationService(core.DB.GetDB(), core.Cache, exceptionSvc)
	handler := NewReconciliationHandler(svc).WithCore(core)

	r.POST("/exception/list", handler.ListExceptions)
	r.POST("/exception/:id", handler.GetExceptionByID)
	r.POST("/exception/:id/resolve", handler.ResolveException)
	r.POST("/by-workstation", handler.GetByWorkstation)
	// Phase 45 R5 / D-R5-A3-01: 手动刷新对账(REFRESH MV + 立即 DetectLayer3)
	// 用于运维/UAT 调试,避开 5min/6min cron 周期
	r.POST("/refresh", handler.Refresh)
}

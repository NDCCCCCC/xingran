package asset

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ModuleReconciliation operlog 模块名常量 (D-16)
//
// R1 只定义 1 个 module 名;R2 才会加 ModuleReconciliationExceptionRule 等。
// R1 所有写操作都在 cron 上下文,handler 全是读端点,
// 无 operlog.Record 调用(参考 operations/asset_handler.go Statistics handler 模式)。
const ModuleReconciliation = "资产对账"

// ModuleReconciliationExceptionRule 例外规则 operlog 模块名(Phase 44 R3 / Phase 42 D-16 锁定)
//
// 用于 CreateRule/UpdateRule/DeleteRule 的 operlog.Record 调用,审计 sys_oper_log.module
// 显示"资产对账-例外规则",区别于主异常流的"资产对账"。
const ModuleReconciliationExceptionRule = "资产对账-例外规则"

// ModuleReconciliationFixSuggestion 修复建议 operlog 模块名(Phase 46 R5 锁定)
//
// 用于 fix_suggestion_handler.go 的 Accept/Apply/Reject/Rollback operlog.Record 调用,
// 审计 sys_oper_log.module 显示 "资产对账-修复建议",区别于主异常流的 "资产对账"。
const ModuleReconciliationFixSuggestion = "资产对账-修复建议"

// ReconciliationHandler 资产对账异常 handler
//
// R1 边界 (D-18): 不暴露 MarkResolved / Create / Update / Delete handler。
// 读端点都不调 operlog.Record(读操作)。
//
// R2 扩展(Phase 43 / D-A4-04):
//   - ResolveException:标记已解决 handler,调 operlog.Record(OperTypeUpdate)
//     完成 WORKORDER-02(状态变更审计)
type ReconciliationHandler struct {
	service asset.ReconciliationService
	core    *core.Core
}

func NewReconciliationHandler(svc asset.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{service: svc}
}

func (h *ReconciliationHandler) WithCore(core *core.Core) *ReconciliationHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ListExceptions 查询异常列表(分页 + JOIN asset_code / physical_username / responsible_username)
func (h *ReconciliationHandler) ListExceptions(c *gin.Context) {
	var params asset.ExceptionListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.ListExceptions(c.Request.Context(), &params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetExceptionByID 按 ID 查询单条异常
func (h *ReconciliationHandler) GetExceptionByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "异常ID不能为空")
		return
	}

	rec, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if rec == nil {
		response.Error(c, http.StatusNotFound, "异常不存在")
		return
	}
	response.Success(c, rec)
}

// ResolveException 标记异常为已解决(Phase 43 R2 / D-A4-04)
//
// 入参:
//   - URL 参数:id = 异常 uuid
//   - Body (可选 JSON):{ "resolutionNote": "..." }
//     — resolution_note 可选,运维可填可不填
//
// 行为:
//  1. 从 URL 取 id
//  2. 解析 body(可选 resolutionNote)
//  3. 从 gin context 取 user_id(由 auth 中间件注入)
//  4. 调 service.ResolveException — SET resolved_at/resolved_by/resolution_note
//  5. 失败 → 400(参数)/404(不存在)/500(DB error)
//  6. 成功 → 调 operlog.Record(OperTypeUpdate) → 写 sys_oper_log(WORKORDER-02)
//  7. 返回 { id, resolvedAt }
//
// 权限:
//   - 当前 R2 简化交付:router 层不强制 RequirePermissions,前端按钮 disabled 控制可见性
//   - 权限粒度 asset:reconciliation:resolve(前端 R3 阶段补后端强制 RequirePermissions)
//
// 与其他 resolve 模式的区别:
//   - 不联动 workorder 关闭(workorder 单独在 workorder UI 关闭,D-A4-04 锁定)
//   - 不触发重检(7d 静默期已兜底,人工重检反而打破静默设计)
func (h *ReconciliationHandler) ResolveException(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "异常ID不能为空")
		return
	}

	// 解析 body(resolution_note 可选)
	var req struct {
		ResolutionNote *string `json:"resolutionNote"`
	}
	// body 可空(SendResolveException 不带 body 时不报错)
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
			return
		}
	}

	// 从 gin context 取 user_id(auth 中间件注入)
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "无法识别当前用户")
		return
	}
	userID, ok := userIDRaw.(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "当前用户ID格式错误")
		return
	}

	// 调 service
	if err := h.service.ResolveException(c.Request.Context(), id, userID, req.ResolutionNote); err != nil {
		// 区分错误码:"已解决" / "不存在" 走 400/404,其他走 500
		errMsg := err.Error()
		if errMsg == "该异常已标记为已解决" {
			response.Error(c, http.StatusBadRequest, errMsg)
			return
		}
		if errMsg == "异常不存在" {
			response.Error(c, http.StatusNotFound, errMsg)
			return
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
		return
	}

	// 🆕 Phase 45 R4 / D-A4-04: 缓存主动失效(避免用户重看页面仍命中旧缓存)
	// 严格顺序(D-A4-04):service → invalidate → operlog → response
	// 注:ResolveException 改动后,reconciliation_normalized.workstation_id 关联的工位健康度
	// 缓存需要被失效。该工位通过 reconciliation_normalized JOIN sys_data_reconciliation 反查。
	// 失败仅 logrus.Warnf(不影响 operlog 写入),与 scheduler 路径保持一致。
	// CR-02 修复:gorm.Row().Scan() 返回 sql.ErrNoRows(不是 gorm.ErrRecordNotFound),
	// 这里与 scheduler 调用 WorkstationIDForException 保持一致的 sql.ErrNoRows 语义。
	var wsID sql.NullString
	scanErr := h.core.GetDB().WithContext(c.Request.Context()).
		Table("reconciliation_normalized").
		Select("reconciliation_normalized.workstation_id").
		Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
		Where("sys_data_reconciliation.id = ? AND sys_data_reconciliation.deleted_at IS NULL AND reconciliation_normalized.workstation_id IS NOT NULL", id).
		Limit(1).
		Row().
		Scan(&wsID)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		// 真实 DB 故障(非 "no row" 情况)记 warning,但不阻断 operlog 写入
		applogger.Warnf("[reconciliation:ResolveException] R4 query workstation failed exceptionID=%s: %v", id, scanErr)
	}
	if wsID.Valid && wsID.String != "" {
		if invErr := asset.InvalidateWorkstationHealth(c.Request.Context(), h.core.Cache, wsID.String); invErr != nil {
			applogger.Warnf("[reconciliation:ResolveException] R4 invalidate cache failed exceptionID=%s wsID=%s: %v", id, wsID.String, invErr)
		}
	}

	// operlog 写入(CLAUDE.md 强制约定,状态变更 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)

	response.Success(c, gin.H{
		"id":             id,
		"resolvedAt":     time.Now(),
		"resolvedBy":     userID,
		"resolutionNote": req.ResolutionNote,
	})
}

// GetByWorkstation 工位对账健康度聚合 (Phase 45 R4 / D-A4-01/02/03)
//
// 入参:
//
//	{ "workstationId": "uuid", "window": "7d" }
//
// 出参 (D-A4-02 锁定):ByWorkstationResponse{Workstation, HealthScore, Assets, Visible}
//
// 行为:
//  1. JSON 绑定(workstationId 必填)
//  2. window 为空时设默认 "7d"
//  3. 调 service.GetByWorkstation 拿数据
//  4. 注入 Visible (D-A1-03 静默降级):基于 HasUserPermission("asset:reconciliation:list")
//     - true:  正常返回 HealthScore + Assets
//     - false: HealthScore 全零,Assets 空切片(避免返回数据但前端 hide,B3 安全 invariant)
//  5. response.Success — 读路径,无 operlog.Record(参考 ListExceptions 模式)
//
// 注:Plan 01 不修改 ResolveException(缓存失效归 Plan 02,B1/B2 锁定)。
func (h *ReconciliationHandler) GetByWorkstation(c *gin.Context) {
	var req struct {
		WorkstationID string `json:"workstationId" binding:"required"`
		Window        string `json:"window"` // 默认 "7d"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}
	if req.Window == "" {
		req.Window = "7d"
	}

	result, err := h.service.GetByWorkstation(c.Request.Context(), req.WorkstationID, req.Window)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 权限降级标记 (D-A1-03 + cross-module-permission.md §2.3)
	// handler 注入 Visible — service 单一职责,不再混入 perm 判断
	result.Visible = h.hasReconciliationPerm(c)
	if !result.Visible {
		// 静默降级:返回结构保留但 Assets 清空,前端 HealthBadge 会渲染 "-"
		result.Assets = []asset.AssetHealthItem{}
		result.HealthScore = asset.HealthScore{
			Total:    0,
			Trend:    []asset.TrendPoint{},
		}
	}

	response.Success(c, result)
}

// Refresh 手动刷新物化视图并立即触发异常检测
// @Summary 手动刷新对账(REFRESH MV + 立即 DetectLayer3)
// @Description 用于运维/UAT 调试,避免等待 5min/6min cron 周期。
// 流程:REFRESH MATERIALIZED VIEW reconciliation_normalized → DetectLayer3 → 返回计数。
// @Tags 资产对账
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=object{inserted=int,skipped=int,skippedSilence=int,skippedThrottle=int}}
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/refresh [post]
func (h *ReconciliationHandler) Refresh(c *gin.Context) {
	inserted, skipped, skippedSilence, skippedThrottle, err := h.service.Refresh(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 写操作日志(运营/UAT 审计)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产对账", operlog.OperTypeSync)

	response.Success(c, gin.H{
		"inserted":        inserted,
		"skipped":         skipped,
		"skippedSilence":  skippedSilence,
		"skippedThrottle": skippedThrottle,
	})
}

// hasReconciliationPerm 内部权限检查 (per cross-module-permission.md §2.3 + D-A1-03)
//
// 复用 pkg/middleware.HasUserPermission 复用底层 checkUserPermission / isSuperAdmin 链路。
// 静默降级:不调 c.Abort(),调用方决定如何处理 false(handler 注入 Visible=false)。
func (h *ReconciliationHandler) hasReconciliationPerm(c *gin.Context) bool {
	return middleware.HasUserPermission(c, h.core, "asset:reconciliation:list")
}

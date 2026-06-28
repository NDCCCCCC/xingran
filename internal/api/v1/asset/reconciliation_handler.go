package asset

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
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

	// operlog 写入(CLAUDE.md 强制约定,状态变更 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliation, operlog.OperTypeUpdate)

	response.Success(c, gin.H{
		"id":          id,
		"resolvedAt":  time.Now(),
		"resolvedBy":  userID,
		"resolutionNote": req.ResolutionNote,
	})
}

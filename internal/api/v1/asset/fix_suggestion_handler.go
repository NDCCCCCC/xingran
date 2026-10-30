package asset

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ModuleReconciliationFixSuggestion 修复建议 operlog 模块名常量
//
// 实际常量定义在 reconciliation_handler.go(与其他对账模块常量集中管理);
// 本文件直接引用,不再重复声明(避免 Go 同包 const 重复声明错误)。

// FixSuggestionHandler 修复建议 handler
//
// 范围:7 个端点
//   - ListFixSuggestions / GetByID / Stats          读端点(无 operlog)
//   - Accept / Reject / Apply / Rollback            写端点(必 operlog)
//
// 写端点严格顺序(D-A4-04):
//   service → invalidate cache → operlog → response
type FixSuggestionHandler struct {
	service asset.FixSuggestionService
	core    *core.Core
}

func NewFixSuggestionHandler(svc asset.FixSuggestionService) *FixSuggestionHandler {
	return &FixSuggestionHandler{service: svc}
}

func (h *FixSuggestionHandler) WithCore(core *core.Core) *FixSuggestionHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ====================== 读端点 ======================

// ListFixSuggestions 查询修复建议列表
func (h *FixSuggestionHandler) ListFixSuggestions(c *gin.Context) {
	var params asset.FixSuggestionListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.service.ListFixSuggestions(c.Request.Context(), &params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetByID 查询单条修复建议详情
func (h *FixSuggestionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "建议ID不能为空")
		return
	}

	detail, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		response.Error(c, http.StatusNotFound, "建议不存在")
		return
	}
	response.Success(c, detail)
}

// Stats 7d 统计(KPI 卡片)
func (h *FixSuggestionHandler) Stats(c *gin.Context) {
	var req struct {
		WindowDays int `json:"windowDays"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
			return
		}
	}
	if req.WindowDays <= 0 {
		req.WindowDays = 7
	}

	stats, err := h.service.Stats(c.Request.Context(), req.WindowDays)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

// ====================== 写端点 ======================

// Accept 接受建议
func (h *FixSuggestionHandler) Accept(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "建议ID不能为空")
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.service.Accept(c.Request.Context(), id, userID); err != nil {
		errMsg := err.Error()
		if errMsg == "该建议已被处理或不存在" {
			response.Error(c, http.StatusConflict, errMsg)
			return
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,状态变更 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

	response.Success(c, gin.H{
		"id":         id,
		"acceptedAt": gin.H{},
		"acceptedBy": userID,
	})
}

// Reject 拒绝建议
func (h *FixSuggestionHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "建议ID不能为空")
		return
	}

	var req struct {
		RejectionReason string `json:"rejectionReason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.service.Reject(c.Request.Context(), id, userID, req.RejectionReason); err != nil {
		errMsg := err.Error()
		if errMsg == "该建议已被处理或不存在" {
			response.Error(c, http.StatusConflict, errMsg)
			return
		}
		if errMsg == "拒绝原因至少 10 字符" {
			response.Error(c, http.StatusBadRequest, errMsg)
			return
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
		return
	}

	// operlog 写入(状态变更 → OperTypeReject)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeReject)

	response.Success(c, gin.H{
		"id":              id,
		"rejectedAt":      gin.H{},
		"rejectedBy":      userID,
		"rejectionReason": req.RejectionReason,
	})
}

// Apply 应用建议(accepted → applied,写 ops_asset.user_id)
func (h *FixSuggestionHandler) Apply(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "建议ID不能为空")
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.service.Apply(c.Request.Context(), id, userID); err != nil {
		errMsg := err.Error()
		if errMsg == "该建议不存在或未处于 accepted 状态" {
			response.Error(c, http.StatusConflict, errMsg)
			return
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
		return
	}

	// D-C4:缓存失效 — 反查 wsID → InvalidateWorkstationHealth
	h.invalidateWorkstationHealth(c, id)

	// operlog 写入(状态变更 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeUpdate)

	response.Success(c, gin.H{
		"id":        id,
		"appliedAt": gin.H{},
		"appliedBy": userID,
	})
}

// Rollback 回滚应用(applied → rolled_back,7d 窗口内)
//
// 严格顺序(D-A4-04):service → invalidate cache → operlog → response
//
// 错误码映射(46-02 强化):
//   - "该建议不存在或未处于 applied 状态" → 409
//   - "回滚窗口已过(7d),不允许回滚"     → 400
//   - "回滚原因至少 10 字符"             → 400
//   - 其他                                → 500
//
// D-C3: operlog.OperTypeReset=11(语义"恢复到原值",与密码/密钥重置同)
func (h *FixSuggestionHandler) Rollback(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "建议ID不能为空")
		return
	}

	var req struct {
		RollbackReason string `json:"rollbackReason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	if err := h.service.Rollback(c.Request.Context(), id, userID, req.RollbackReason); err != nil {
		errMsg := err.Error()
		if errMsg == "该建议不存在或未处于 applied 状态" {
			response.Error(c, http.StatusConflict, errMsg)
			return
		}
		if errMsg == "回滚窗口已过(7d),不允许回滚" {
			response.Error(c, http.StatusBadRequest, errMsg)
			return
		}
		if errMsg == "回滚原因至少 10 字符" {
			response.Error(c, http.StatusBadRequest, errMsg)
			return
		}
		response.Error(c, http.StatusInternalServerError, errMsg)
		return
	}

	// D-C4:缓存失效
	h.invalidateWorkstationHealth(c, id)

	// D-C3: operlog 强写 — OperTypeReset=11(密码/密钥重置,语义最接近"恢复到原值")
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationFixSuggestion, operlog.OperTypeReset)

	response.Success(c, gin.H{
		"id":             id,
		"rolledBackAt":   gin.H{},
		"rolledBackBy":   userID,
		"rollbackReason": req.RollbackReason,
	})
}

// ====================== 内部 helpers ======================

// getUserID 从 gin context 取 user_id(auth 中间件注入)
func (h *FixSuggestionHandler) getUserID(c *gin.Context) (string, bool) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "无法识别当前用户")
		return "", false
	}
	userID, ok := userIDRaw.(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "当前用户ID格式错误")
		return "", false
	}
	return userID, true
}

// invalidateWorkstationHealth D-C4:反查 wsID → InvalidateWorkstationHealth
//
// 严格顺序(D-A4-04):service → invalidate → operlog → response
// 失败仅 warn log,不阻断 operlog 写入
//
// CR-02 修复:gorm.Row().Scan() 返回 sql.ErrNoRows(不是 gorm.ErrRecordNotFound)
func (h *FixSuggestionHandler) invalidateWorkstationHealth(c *gin.Context, suggestionID string) {
	if h.core == nil {
		return
	}
	var wsID sql.NullString
	scanErr := h.core.GetDB().WithContext(c.Request.Context()).
		Table("reconciliation_normalized").
		Select("reconciliation_normalized.workstation_id").
		Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
		Joins("JOIN sys_reconciliation_fix_suggestion ON sys_reconciliation_fix_suggestion.exception_id = sys_data_reconciliation.id").
		Where("sys_reconciliation_fix_suggestion.id = ? AND sys_reconciliation_fix_suggestion.deleted_at IS NULL", suggestionID).
		Limit(1).
		Row().
		Scan(&wsID)
	if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		applogger.Warnf("[fix-suggestion] R4 query workstation failed suggestionID=%s: %v", suggestionID, scanErr)
	}
	if wsID.Valid && wsID.String != "" {
		if invErr := asset.InvalidateWorkstationHealth(c.Request.Context(), h.core.Cache, wsID.String); invErr != nil {
			applogger.Warnf("[fix-suggestion] R4 invalidate cache failed suggestionID=%s wsID=%s: %v", suggestionID, wsID.String, invErr)
		}
	}
}

package asset

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ReconciliationExceptionHandler 资产对账例外规则 handler
//
// R1 边界: 仅 List + GetByID skeleton,R3 实现 Create/Update/Delete/enable/disable。
// 例外规则 seed 数据已就位(42-01 migration_169),R1 仅提供只读查询入口,
// 不调 operlog.Record(读操作)。
type ReconciliationExceptionHandler struct {
	service asset.ReconciliationExceptionService
	core    *core.Core
}

func NewReconciliationExceptionHandler(svc asset.ReconciliationExceptionService) *ReconciliationExceptionHandler {
	return &ReconciliationExceptionHandler{service: svc}
}

func (h *ReconciliationExceptionHandler) WithCore(core *core.Core) *ReconciliationExceptionHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// ListRules 查询例外规则列表
func (h *ReconciliationExceptionHandler) ListRules(c *gin.Context) {
	var params asset.ExceptionRuleListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.List(c.Request.Context(), &params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetRuleByID 按 ID 查询单条规则
func (h *ReconciliationExceptionHandler) GetRuleByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	rule, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}
	response.Success(c, rule)
}

// ============================================================================
// R3 写操作 + 命中测试 handler (Phase 44 R3 / Plan 44-01 Task 5)
//
// 强制约定(CLAUDE.md "操作日志记录约定 — 强制"):
//   - CreateRule / UpdateRule / DeleteRule 在 success path 末尾、response.Success
//     之前调 operlog.Record(ModuleReconciliationExceptionRule, OperTypeXxx)
//   - 例外规则 CRUD 无敏感字段(reason 是普通文本),用 operlog.Record 非 RecordWithBody
//   - TestRule 是读操作,不调 operlog(参考 ListRules/GetRuleByID)
// ============================================================================

// CreateRule 创建例外规则
//
// 行为:
//  1. ShouldBindJSON 失败 → 400
//  2. service.Create 失败(含 ValidateCIDR/Actions/SeverityOverride/Reason) → 500
//  3. success path 末尾 operlog.Record(OperTypeCreate) → 写 sys_oper_log
//  4. response.Success(rule)
func (h *ReconciliationExceptionHandler) CreateRule(c *gin.Context) {
	var req asset.CreateExceptionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	rule, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,新增 → OperTypeCreate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeCreate)

	response.Success(c, rule)
}

// UpdateRule 更新例外规则
//
// 行为:
//  1. URL 参数 :id + body(UpdateExceptionRuleRequest)
//  2. service.Update 失败(含校验 / 不存在) → 500
//  3. success path 末尾 operlog.Record(OperTypeUpdate)
func (h *ReconciliationExceptionHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	var req asset.UpdateExceptionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := h.service.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,修改 → OperTypeUpdate)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeUpdate)

	response.Success(c, gin.H{"id": id})
}

// DeleteRule 软删除例外规则
//
// 行为:
//  1. URL 参数 :id
//  2. service.Delete 失败(规则不存在) → 500
//  3. success path 末尾 operlog.Record(OperTypeDelete)
func (h *ReconciliationExceptionHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// operlog 写入(CLAUDE.md 强制约定,删除 → OperTypeDelete)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeDelete)

	response.Success(c, gin.H{"id": id})
}

// TestRule 命中测试工具(EXCEPTION-04 / SC 6)
//
// 读操作 — 不调 operlog.Record(参考 ListRules/GetRuleByID 无 operlog)。
//
// 入参 body: { "ip": "192.168.0.10", "userId": "uuid?", "deptId": "uuid?" }
// 返回: MatchTestResult { matchedRules, mergedActions, finalSeverity, isSilence, needsUserDept }
func (h *ReconciliationExceptionHandler) TestRule(c *gin.Context) {
	var req struct {
		IP     string `json:"ip" binding:"required"`
		UserID string `json:"userId"`
		DeptID string `json:"deptId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: ip 必填")
		return
	}

	result, err := h.service.MatchTest(c.Request.Context(), req.IP, req.UserID, req.DeptID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

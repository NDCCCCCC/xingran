package asset

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
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

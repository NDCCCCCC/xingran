package asset

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ModuleReconciliation operlog 模块名常量 (D-16)
//
// R1 只定义 1 个 module 名;R2 才会加 ModuleReconciliationExceptionRule 等。
// R1 所有写操作都在 cron 上下文,handler 全是读端点,
// 无 operlog.Record 调用(参考 operations/asset_handler.go Statistics handler 模式)。
const ModuleReconciliation = "资产对账"

// ReconciliationHandler 资产对账异常 handler
//
// R1 边界 (D-18): 不暴露 MarkResolved / Create / Update / Delete handler。
// 读端点都不调 operlog.Record(读操作)。
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

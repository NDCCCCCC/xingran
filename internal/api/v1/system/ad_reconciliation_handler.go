package system

import (
	"github.com/gin-gonic/gin"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// ADReconciliationHandler 部门↔OU 对账处理器（Phase 59 / GUARD-02，D-14~D-16）。
//
// 只读端点：不调用 operlog 的 Record / RecordWithBody，不设置 oper_log_title
// （operlog 约定仅覆盖业务写操作；本 handler 只查 v_dept_ou_mismatch 视图）。
type ADReconciliationHandler struct {
	svc addomainServices.DeptOUMismatchService
}

// NewADReconciliationHandler 创建对账处理器（内部构造 service，与 ADLoginHistoryHandler 同形状）
func NewADReconciliationHandler(db *gorm.DB) *ADReconciliationHandler {
	return &ADReconciliationHandler{
		svc: addomainServices.NewDeptOUMismatchService(db),
	}
}

// ListDeptOUMismatch 分页查询部门↔OU 对账不一致条目
// @Summary 查询部门-OU 对账不一致
// @Description 分页查询 v_dept_ou_mismatch（mismatchType 四值白名单 + 可选 deptId/keyword 筛选）
// @Tags AD域管理
// @Accept json
// @Produce json
// @Param request body addomainServices.DeptOUMismatchQuery true "查询条件"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ad-domain/reconciliation/mismatch [post]
func (h *ADReconciliationHandler) ListDeptOUMismatch(c *gin.Context) {
	var req addomainServices.DeptOUMismatchQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// mismatchType 非空时走 D-10 四值白名单（单一事实源 addomain.MismatchTypes）。
	// 非法值不静默忽略——静默忽略会把拼写错误伪装成「没有不一致」的空结果。
	if req.MismatchType != "" {
		valid := false
		for _, mt := range addomainServices.MismatchTypes {
			if mt == req.MismatchType {
				valid = true
				break
			}
		}
		if !valid {
			response.Error(c, apperrors.New(apperrors.CodeParamInvalid, "无效的不一致类型"))
			return
		}
	}

	// 分页归一化是 handler 的职责（默认 1/10，上限 200），service 不归一化。
	req.Current, req.PageSize = query.NormalizePagination(req.Current, req.PageSize)

	list, total, err := h.svc.ListDeptOUMismatch(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, list, total, req.Current, req.PageSize)
}

package monitor

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// OperLogHandler 操作日志处理器
type OperLogHandler struct {
	service monitorServices.OperLogService
	core    *core.Core
}

// NewOperLogHandler 创建操作日志处理器实例
func NewOperLogHandler(service monitorServices.OperLogService) *OperLogHandler {
	return &OperLogHandler{service: service}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *OperLogHandler) WithCore(core *core.Core) *OperLogHandler {
	h.core = core
	return h
}

// OperLogListRequest 操作日志列表请求
type OperLogListRequest struct {
	base.BaseListRequest
	Title        *string `json:"title,omitempty"`
	BusinessType *int    `json:"businessType,omitempty"`
	Status       *int    `json:"status,omitempty"`
	OperName     *string `json:"operName,omitempty"`
	BeginTime    *string `json:"beginTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
}

// List 查询操作日志列表
// @Summary 查询操作日志列表
// @Description 分页查询操作日志列表
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param request body OperLogListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /monitor/oper-logs/list [post]
func (h *OperLogHandler) List(c *gin.Context) {
	var req OperLogListRequest
	// 允许空的请求体，设置默认值
	if err := c.ShouldBindJSON(&req); err != nil {
		req = OperLogListRequest{
			BaseListRequest: base.BaseListRequest{
				Current:  1,
				PageSize: 10,
			},
		}
	}

	params := monitorServices.OperLogListParams{
		BaseListRequest: req.BaseListRequest,
		Title:           req.Title,
		BusinessType:    req.BusinessType,
		Status:          req.Status,
		OperName:        req.OperName,
		BeginTime:       req.BeginTime,
		EndTime:         req.EndTime,
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取操作日志详情
// @Summary 获取操作日志详情
// @Description 根据ID获取操作日志详情
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param id path string true "操作日志ID"
// @Success 200 {object} response.Response
// @Router /monitor/oper-logs/:id [post]
func (h *OperLogHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	operLog, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, operLog)
}

// Delete 删除操作日志
// @Summary 删除操作日志
// @Description 删除指定操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param id path string true "操作日志ID"
// @Success 200 {object} response.Response
// @Router /monitor/oper-logs/:id/delete [post]
func (h *OperLogHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	err := h.service.Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除操作日志") {
		return
	}

	response.Success(c, nil)
}

// BatchDelete 批量删除操作日志
// @Summary 批量删除操作日志
// @Description 批量删除多个操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "ID列表"
// @Success 200 {object} response.Response
// @Router /monitor/oper-logs/batch-delete [post]
func (h *OperLogHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.service.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除操作日志") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "操作日志", operlog.OperTypeBatch)

	response.Success(c, nil)
}

// Clean 清空操作日志
// @Summary 清空操作日志
// @Description 清空所有操作日志
// @Tags 操作日志
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /monitor/oper-logs/clean [post]
func (h *OperLogHandler) Clean(c *gin.Context) {
	// Self-referential safeguard (T-34-W6-01): the audit row for Clean MUST be
	// inserted SYNCHRONOUSLY before the Clean delete runs, so it survives the
	// cleanup transaction. Using operlog.Record (which calls RecordAsync)
	// would race — the async insert could land after Clean nukes everything,
	// or the cutoff could include the just-inserted row. The synchronous
	// services.OperLogService.RecordOperLog commits the row in-process before
	// the delete begins. Do not change to RecordAsync without understanding
	// the chicken-and-egg risk.
	now := time.Now()
	operUrl := c.Request.URL.String()
	clientIP := c.ClientIP()
	cleanAuditRow := &models.OperLog{
		Title:         "操作日志",
		BusinessType:  operlog.OperTypeClean,
		RequestMethod: c.Request.Method,
		OperatorType:  1,
		OperUrl:       &operUrl,
		OperIP:        &clientIP,
		OperParam:     strPtr(`{"action":"clean"}`),
		Status:        int(models.OperLogStatusSuccess),
		OperTime:      now,
	}
	if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil {
		_ = h.core.OperLogService.RecordOperLog(c.Request.Context(), h.core.GetDB(), cleanAuditRow)
	}

	err := h.service.Clean(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "清空操作日志") {
		return
	}

	// Post-clean verification (T-34-W6-01): the synchronous audit row above
	// was committed before the Clean delete. Since the new row's oper_time is
	// "now" and Clean deletes only old rows, the row must survive. Query to
	// confirm; on missing row, log a hard error for follow-up (the chicken-
	// and-egg safeguard is broken).
	if h.core != nil && h.core.GetDB() != nil {
		var surviveCount int64
		verifyDB := h.core.GetDB().Model(&models.OperLog{}).
			Where("title = ? AND business_type = ?", "操作日志", operlog.OperTypeClean).
			Count(&surviveCount)
		if verifyDB.Error != nil || surviveCount == 0 {
			// TODO(follow-up): if seen in production logs, the synchronous
			// insert + verify contract is broken — investigate before relying
			// on Clean audit trail.
			c.Error(gin.Error{Err: verifyDB.Error, Type: gin.ErrorTypePrivate})
		}
	}

	response.Success(c, gin.H{"message": "清空成功"})
}

// strPtr helper returns a pointer to the given string (local to keep this file
// self-contained; avoids importing a util package just for one call site).
func strPtr(s string) *string { return &s }

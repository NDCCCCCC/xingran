package monitor

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// OperLogHandler 操作日志处理器 — embeds MonitorLogHandler for Delete/BatchDelete.
type OperLogHandler struct {
	*MonitorLogHandler[models.OperLog] // shared Delete/BatchDelete
	svc                                monitorServices.OperLogService
}

// NewOperLogHandler creates an OperLogHandler.
func NewOperLogHandler(svc monitorServices.OperLogService) *OperLogHandler {
	h := &OperLogHandler{svc: svc}
	h.MonitorLogHandler = NewMonitorLogHandler[models.OperLog](
		svc, // logMutator — svc implements Delete and BatchDelete
		"操作日志",
		operlog.OperTypeDelete,
		operlog.OperTypeBatch,
		false, // needOperlog=false: OperLog delete does NOT call operlog
	)
	return h
}

// WithCore injects core dependency.
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
func (h *OperLogHandler) List(c *gin.Context) {
	var req OperLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = OperLogListRequest{
			BaseListRequest: base.BaseListRequest{
				Current:  constants.DefaultCurrent,
				PageSize: constants.DefaultPageSize,
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

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取操作日志详情
func (h *OperLogHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}

	operLog, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, operLog)
}

// Clean 清空操作日志
func (h *OperLogHandler) Clean(c *gin.Context) {
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

	err := h.svc.Clean(c.Request.Context())
	if !response.HandleServiceError(c, err, "清空操作日志") {
		return
	}

	if h.core != nil && h.core.GetDB() != nil {
		var surviveCount int64
		verifyDB := h.core.GetDB().Model(&models.OperLog{}).
			Where("title = ? AND business_type = ?", "操作日志", operlog.OperTypeClean).
			Count(&surviveCount)
		if verifyDB.Error != nil || surviveCount == 0 {
			c.Error(gin.Error{Err: verifyDB.Error, Type: gin.ErrorTypePrivate})
		}
	}

	response.Success(c, gin.H{"message": "清空成功"})
}

func strPtr(s string) *string { return &s }

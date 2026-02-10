package monitor

import (
	"github.com/gin-gonic/gin"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	monitorServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/monitor"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// OperLogHandler 操作日志处理器
type OperLogHandler struct {
	service monitorServices.OperLogService
}

// NewOperLogHandler 创建操作日志处理器实例
func NewOperLogHandler(service monitorServices.OperLogService) *OperLogHandler {
	return &OperLogHandler{service: service}
}

// OperLogListRequest 操作日志列表请求
type OperLogListRequest struct {
	Title        *string `json:"title,omitempty"`
	BusinessType *int    `json:"businessType,omitempty"`
	Status       *int    `json:"status,omitempty"`
	OperName     *string `json:"operName,omitempty"`
	BeginTime    *string `json:"beginTime,omitempty"`
	EndTime      *string `json:"endTime,omitempty"`
	Current      int     `json:"current" binding:"min=1"`
	PageSize     int     `json:"pageSize" binding:"min=1,max=100"`
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
			Current:  1,
			PageSize: 10,
		}
	}

	params := monitorServices.OperLogListParams{
		Title:        req.Title,
		BusinessType: req.BusinessType,
		Status:       req.Status,
		OperName:     req.OperName,
		BeginTime:    req.BeginTime,
		EndTime:      req.EndTime,
		Current:      req.Current,
		PageSize:     req.PageSize,
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
	err := h.service.Clean(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "清空操作日志") {
		return
	}

	response.Success(c, gin.H{"message": "清空成功"})
}

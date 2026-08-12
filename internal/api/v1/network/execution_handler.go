package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ExecutionHandler 配置执行处理器
type ExecutionHandler struct {
	executionService *services.ConfigExecutionService
	core             *core.Core
}

// NewExecutionHandler 创建配置执行处理器实例
func NewExecutionHandler(executionService *services.ConfigExecutionService) *ExecutionHandler {
	return &ExecutionHandler{executionService: executionService}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *ExecutionHandler) WithCore(core *core.Core) *ExecutionHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// List 查询配置执行列表
// @Summary 查询配置执行列表
// @Description 分页查询配置执行记录
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int} true "分页参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/executions/list [post]
// Statistics 配置执行统计(总数/待执行/执行中/成功/失败)
// @Summary 配置执行统计
// @Description 用 COUNT 聚合返回配置执行统计(execution_type=template),供统计卡片使用
// @Tags 配置执行
// @Produce json
// @Success 200 {object} response.Response
// @Router /network/executions/statistics [post]
func (h *ExecutionHandler) Statistics(c *gin.Context) {
	result, err := h.executionService.GetStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ExecutionHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	executions, total, err := h.executionService.GetExecutionList(c.Request.Context(), current, pageSize, getOrderByColumn(rawReq), getIsAscPtr(rawReq))
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	pageResp := response.PageResponse{
		List:     executions,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}

	response.Success(c, pageResp)
}

// ExecuteByTemplate 通过模板执行配置
// @Summary 通过模板执行配置
// @Description 使用配置模板在多个设备上执行配置
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param request body services.TemplateExecutionRequest true "执行请求"
// @Success 200 {object} response.Response
// @Router /network/executions/template/execute [post]
func (h *ExecutionHandler) ExecuteByTemplate(c *gin.Context) {
	var req services.TemplateExecutionRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 设置默认并发数
	if req.Concurrency <= 0 {
		req.Concurrency = 10
	}

	// 设置默认超时时间（5分钟）
	if req.Timeout <= 0 {
		req.Timeout = 300
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	result, err := h.executionService.ExecuteByTemplate(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "执行配置") {
		return
	}

	// ExecuteByTemplate 是在网络设备上批量下发配置的高价值审计点 — 用 OperTypeOther
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令执行", operlog.OperTypeOther)
	response.Success(c, result)
}

// GetByID 获取配置执行详情
// @Summary 获取配置执行详情
// @Description 获取执行ID的详细结果
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} response.Response
// @Router /network/executions/:id [post]
func (h *ExecutionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("执行ID"))
		return
	}

	result, err := h.executionService.GetExecutionResult(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, result)
}

// Cancel 取消配置执行
// @Summary 取消配置执行
// @Description 取消正在执行或待执行的任务
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} response.Response
// @Router /network/executions/:id/cancel [post]
func (h *ExecutionHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("执行ID"))
		return
	}

	err := h.executionService.CancelExecution(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "取消配置执行") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令执行", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "取消成功"})
}

// Delete 删除配置执行记录
// @Summary 删除配置执行记录
// @Description 删除指定的执行记录
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} response.Response
// @Router /network/executions/:id/delete [post]
func (h *ExecutionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("执行ID"))
		return
	}

	err := h.executionService.DeleteExecution(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除配置执行记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令执行", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除配置执行记录
// @Summary 批量删除配置执行记录
// @Description 批量删除多个执行记录
// @Tags 配置执行
// @Accept json
// @Produce json
// @Param request body object{executionIds=[]string} true "执行ID列表"
// @Success 200 {object} response.Response
// @Router /network/executions/batch-delete [post]
func (h *ExecutionHandler) BatchDelete(c *gin.Context) {
	var req struct {
		ExecutionIDs []string `json:"executionIds" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.executionService.BatchDeleteExecutions(c.Request.Context(), req.ExecutionIDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除配置执行记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "命令执行", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.ExecutionIDs),
	})
}

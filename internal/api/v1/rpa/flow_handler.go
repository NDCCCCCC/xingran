package rpa

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
)

// FlowHandler 流程控制处理器
type FlowHandler struct {
	flowService   rpa.FlowControlService
	errorService  rpa.ErrorHandlingService
	mapperService rpa.DataMapperService
	core          *core.Core
}

// NewFlowHandler 创建流程控制处理器
func NewFlowHandler(
	flowService rpa.FlowControlService,
	errorService rpa.ErrorHandlingService,
	mapperService rpa.DataMapperService,
) *FlowHandler {
	return &FlowHandler{
		flowService:   flowService,
		errorService:  errorService,
		mapperService: mapperService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *FlowHandler) WithCore(core *core.Core) *FlowHandler {
	h.core = core
	return h
}

// EvaluateConditionRequest 条件评估请求
type EvaluateConditionRequest struct {
	Expression string                 `json:"expression" binding:"required"`
	Variables  map[string]interface{} `json:"variables"`
}

// EvaluateConditionResponse 条件评估响应
type EvaluateConditionResponse struct {
	Result bool   `json:"result"`
	Error  string `json:"error,omitempty"`
}

// EvaluateCondition 评估条件表达式
// @Summary 流程条件求值
// @Description RPA流程条件表达式求值
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "条件与数据上下文"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/evaluate-condition [post]
func (h *FlowHandler) EvaluateCondition(c *gin.Context) {
	var req EvaluateConditionRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.flowService.EvaluateCondition(c.Request.Context(), req.Expression, req.Variables)
	if handleError(c, err, http.StatusInternalServerError, "条件评估失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, &EvaluateConditionResponse{Result: result})
}

// MapDataRequest 数据映射请求
type MapDataRequest struct {
	Config rpa.DataMappingConfig `json:"config" binding:"required"`
	Source interface{}           `json:"source" binding:"required"`
}

// MapData 执行数据映射
// @Summary 流程数据映射
// @Description RPA流程字段映射转换
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "映射规则与数据"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/map-data [post]
func (h *FlowHandler) MapData(c *gin.Context) {
	var req MapDataRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.mapperService.MapData(c.Request.Context(), &req.Config, req.Source)
	if handleError(c, err, http.StatusInternalServerError, "数据映射失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, result)
}

// TransformValueRequest 值转换请求
type TransformValueRequest struct {
	Value     interface{}            `json:"value" binding:"required"`
	Transform rpa.TransformFunction  `json:"transform" binding:"required"`
	Params    map[string]interface{} `json:"params"`
}

// TransformValue 转换值
// @Summary 流程值转换
// @Description RPA流程字段值转换
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "转换规则与数据"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/transform-value [post]
func (h *FlowHandler) TransformValue(c *gin.Context) {
	var req TransformValueRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.mapperService.TransformValue(c.Request.Context(), req.Value, req.Transform, req.Params)
	if handleError(c, err, http.StatusInternalServerError, "值转换失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, gin.H{"value": result})
}

// ExtractJSONPathRequest JSON 路径提取请求
type ExtractJSONPathRequest struct {
	Data interface{} `json:"data" binding:"required"`
	Path string      `json:"path" binding:"required"`
}

// ExtractJSONPath 从 JSON 提取路径值
// @Summary JSONPath 提取
// @Description 从 JSON 数据中按 JSONPath 提取值
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "JSON 数据与 JSONPath"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/extract-jsonpath [post]
func (h *FlowHandler) ExtractJSONPath(c *gin.Context) {
	var req ExtractJSONPathRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.mapperService.ExtractJSONPath(c.Request.Context(), req.Data, req.Path)
	if handleError(c, err, http.StatusInternalServerError, "路径提取失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, gin.H{"value": result})
}

// HandleErrorRequest 错误处理请求
type HandleErrorRequest struct {
	ExecutionID string                   `json:"executionId" binding:"required"`
	StepIndex   int                      `json:"stepIndex" binding:"required"`
	Error       string                   `json:"error" binding:"required"`
	Config      *rpa.ErrorHandlingConfig `json:"config"`
	Variables   map[string]interface{}   `json:"variables"`
}

// HandleError 处理错误
// @Summary 流程错误处理
// @Description RPA流程错误处理策略求值
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "错误与处理策略"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/handle-error [post]
func (h *FlowHandler) HandleError(c *gin.Context) {
	var req HandleErrorRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 将错误字符串转换为 error 类型
	rpaErr := &rpaError{message: req.Error}

	handleReq := &rpa.ErrorHandleRequest{
		ExecutionID: req.ExecutionID,
		StepIndex:   req.StepIndex,
		Error:       rpaErr,
		Config:      req.Config,
		Variables:   req.Variables,
	}

	result, err := h.errorService.HandleError(c.Request.Context(), handleReq)
	if handleError(c, err, http.StatusInternalServerError, "错误处理失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, result)
}

// ExecuteRetryRequest 重试请求
type ExecuteRetryRequest struct {
	ExecutionID string                 `json:"executionId" binding:"required"`
	StepIndex   int                    `json:"stepIndex" binding:"required"`
	Policy      *rpa.RetryPolicy       `json:"policy"`
	Variables   map[string]interface{} `json:"variables"`
	Attempt     int                    `json:"attempt"`
}

// ExecuteRetry 执行重试
// @Summary 流程重试执行
// @Description RPA流程重试策略求值
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "重试策略与上下文"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/execute-retry [post]
func (h *FlowHandler) ExecuteRetry(c *gin.Context) {
	var req ExecuteRetryRequest
	if !bindAndValidate(c, &req) {
		return
	}

	retryReq := &rpa.RetryRequest{
		ExecutionID: req.ExecutionID,
		StepIndex:   req.StepIndex,
		Policy:      req.Policy,
		Variables:   req.Variables,
		Attempt:     req.Attempt,
	}

	result, err := h.errorService.ExecuteRetry(c.Request.Context(), retryReq)
	if handleError(c, err, http.StatusInternalServerError, "重试执行失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, result)
}

// rpaError 自定义错误类型
type rpaError struct {
	message string
}

func (e *rpaError) Error() string {
	return e.message
}

// AggregateDataRequest 数据聚合请求
type AggregateDataRequest struct {
	Data          []interface{} `json:"data" binding:"required"`
	AggregateType string        `json:"aggregateType" binding:"required"`
}

// AggregateData 聚合数据
// @Summary 流程数据聚合
// @Description RPA流程数据聚合计算
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "聚合规则与数据"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/flow/aggregate-data [post]
func (h *FlowHandler) AggregateData(c *gin.Context) {
	var req AggregateDataRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.mapperService.AggregateData(c.Request.Context(), req.Data, req.AggregateType)
	if handleError(c, err, http.StatusInternalServerError, "数据聚合失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA流程", operlog.OperTypeOther)

	success(c, gin.H{"value": result})
}

package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ExecutionHandler 配置执行处理器
type ExecutionHandler struct {
	executionService *services.ConfigExecutionService
}

// NewExecutionHandler 创建配置执行处理器实例
func NewExecutionHandler(executionService *services.ConfigExecutionService) *ExecutionHandler {
	return &ExecutionHandler{executionService: executionService}
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
func (h *ExecutionHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	executions, total, err := h.executionService.GetExecutionList(c.Request.Context(), current, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
		response.Error(c, http.StatusBadRequest, "执行ID不能为空")
		return
	}

	result, err := h.executionService.GetExecutionResult(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
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
		response.Error(c, http.StatusBadRequest, "执行ID不能为空")
		return
	}

	err := h.executionService.CancelExecution(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "取消配置执行") {
		return
	}

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
		response.Error(c, http.StatusBadRequest, "执行ID不能为空")
		return
	}

	err := h.executionService.DeleteExecution(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除配置执行记录") {
		return
	}

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

	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.ExecutionIDs),
	})
}

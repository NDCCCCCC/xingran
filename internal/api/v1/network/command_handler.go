package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	"gorm.io/gorm"
)

// CommandHandler 命令分发处理器
type CommandHandler struct {
	commandService *services.CommandDispatchService
	db             *gorm.DB
}

// NewCommandHandler 创建命令分发处理器实例
func NewCommandHandler(commandService *services.CommandDispatchService, db *gorm.DB) *CommandHandler {
	return &CommandHandler{
		commandService: commandService,
		db:             db,
	}
}

// Dispatch 分发命令
// @Summary 分发命令
// @Description 向一个或多个设备分发命令
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body services.DispatchRequest true "分发请求"
// @Success 200 {object} response.Response
// @Router /network/command/dispatch [post]
func (h *CommandHandler) Dispatch(c *gin.Context) {
	var req services.DispatchRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 设置默认值
	if req.ExecutionStrategy == "" {
		req.ExecutionStrategy = models.ExecutionStrategyParallel
	}
	if req.Concurrency == 0 {
		req.Concurrency = 10
	}
	if req.Timeout == 0 {
		req.Timeout = 300
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	result, err := h.commandService.Dispatch(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "分发命令") {
		return
	}

	response.Success(c, result)
}

// QuickCommand 快速命令执行
// @Summary 快速命令执行
// @Description 在单个设备上快速执行命令
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body object{deviceId=string,command=string,timeout=int} true "命令请求"
// @Success 200 {object} response.Response
// @Router /network/command/quick [post]
func (h *CommandHandler) QuickCommand(c *gin.Context) {
	var req struct {
		DeviceID string `json:"deviceId" binding:"required"`
		Command  string `json:"command" binding:"required"`
		Timeout  int    `json:"timeout" binding:"omitempty,min=10,max=300"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	if req.Timeout == 0 {
		req.Timeout = 60
	}

	result, err := h.commandService.QuickCommand(c.Request.Context(), req.DeviceID, req.Command, req.Timeout)
	if !responseHelpers.HandleServiceError(c, err, "执行快速命令") {
		return
	}

	response.Success(c, result)
}

// GetExecutionResult 获取命令执行结果
// @Summary 获取命令执行结果
// @Description 获取命令执行的详细结果
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} response.Response
// @Router /network/command/:id [post]
func (h *CommandHandler) GetExecutionResult(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "执行ID不能为空")
		return
	}

	result, err := h.commandService.GetExecutionResult(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// List 获取命令执行列表
// @Summary 获取命令执行列表
// @Description 分页查询命令执行记录
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int} true "分页参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/command/list [post]
func (h *CommandHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	executions, total, err := h.commandService.GetExecutionList(c.Request.Context(), current, pageSize)
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

// GetDeviceExecutionDetail 获取设备执行明细
// @Summary 获取设备执行明细
// @Description 获取指定设备的命令执行明细
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Param deviceId path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/command/:id/device/:deviceId [post]
func (h *CommandHandler) GetDeviceExecutionDetail(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		response.Error(c, http.StatusBadRequest, "执行ID不能为空")
		return
	}

	deviceID := c.Param("deviceId")
	if deviceID == "" {
		response.Error(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}

	var detail models.ConfigExecutionDetail
	if err := h.db.Where("execution_id = ? AND device_id = ?", executionID, deviceID).
		First(&detail).Error; err != nil {
		response.Error(c, http.StatusNotFound, "执行明细不存在")
		return
	}

	response.Success(c, detail)
}

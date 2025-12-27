package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// CommandDispatchRequest 命令分发请求
type CommandDispatchRequest struct {
	ExecutionName     string                 `json:"executionName" binding:"required,min=2,max=200"`
	DeviceIDs         []string               `json:"deviceIds" binding:"required,min=1"`
	CommandContent    string                 `json:"commandContent" binding:"required"`
	ExecutionStrategy models.ExecutionStrategy `json:"executionStrategy" binding:"omitempty,oneof=parallel serial"`
	Concurrency       int                    `json:"concurrency" binding:"omitempty,min=1,max=50"`
	Timeout           int                    `json:"timeout" binding:"omitempty,min=10,max=3600"`
}

// QuickCommandRequest 快速命令请求
type QuickCommandRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
	Command  string `json:"command" binding:"required"`
	Timeout  int    `json:"timeout" binding:"omitempty,min=10,max=300"`
}

// dispatchCommand 分发命令
// @Summary 分发命令
// @Description 向一个或多个设备分发命令
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body CommandDispatchRequest true "分发请求"
// @Success 200 {object} response.Response
// @Router /network/command/dispatch [post]
func dispatchCommand(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CommandDispatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
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

		// 创建设备管理器
	manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewCommandDispatchService(core.DB.GetDB(), manager)
		result, err := svc.Dispatch(c.Request.Context(), &services.DispatchRequest{
		ExecutionName:     req.ExecutionName,
		DeviceIDs:         req.DeviceIDs,
		CommandContent:    req.CommandContent,
		ExecutionStrategy: req.ExecutionStrategy,
		Concurrency:       req.Concurrency,
		Timeout:           req.Timeout,
		CreatedBy:         c.GetString("userId"),
	})
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, result)
	}
}

// quickCommand 快速命令执行
// @Summary 快速命令执行
// @Description 在单个设备上快速执行命令
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body QuickCommandRequest true "命令请求"
// @Success 200 {object} response.Response
// @Router /network/command/quick [post]
func quickCommand(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req QuickCommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		if req.Timeout == 0 {
			req.Timeout = 60
		}

		// 创建设备管理器
		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewCommandDispatchService(core.DB.GetDB(), manager)
		result, err := svc.QuickCommand(c.Request.Context(), req.DeviceID, req.Command, req.Timeout)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// getCommandExecution 获取命令执行结果
// @Summary 获取命令执行结果
// @Description 获取命令执行的详细结果
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} response.Response
// @Router /network/command/:id [post]
func getCommandExecution(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "ID不能为空")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewCommandDispatchService(core.DB.GetDB(), manager)
		result, err := svc.GetExecutionResult(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// listCommandExecutions 获取命令执行列表
// @Summary 获取命令执行列表
// @Description 分页查询命令执行记录
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param request body map[string]int true "分页参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/command/list [post]
func listCommandExecutions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			rawReq = make(map[string]interface{})
		}

		current := getIntField(rawReq, "current", 1)
		pageSize := getIntField(rawReq, "pageSize", 10)

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewCommandDispatchService(core.DB.GetDB(), manager)
		executions, total, err := svc.GetExecutionList(c.Request.Context(), current, pageSize)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
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
}

// getDeviceExecutionDetail 获取设备执行明细
// @Summary 获取设备执行明细
// @Description 获取指定设备的命令执行明细
// @Tags 命令分发
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Param deviceId path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/command/:id/device/:deviceId [post]
func getDeviceExecutionDetail(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		executionID := c.Param("id")
		if executionID == "" {
			response.Error(c, response.ErrBadRequest, "执行ID不能为空")
			return
		}

		deviceID := c.Param("deviceId")
		if deviceID == "" {
			response.Error(c, response.ErrBadRequest, "设备ID不能为空")
			return
		}

		var detail models.ConfigExecutionDetail
		if err := core.DB.GetDB().Where("execution_id = ? AND device_id = ?", executionID, deviceID).
			First(&detail).Error; err != nil {
			response.Error(c, response.ErrNotFound, "执行明细不存在")
			return
		}

		response.Success(c, detail)
	}
}

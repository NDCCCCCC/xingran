package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// TemplateExecutionRequest 模板执行请求
type TemplateExecutionRequest struct {
	ExecutionName    string                 `json:"executionName" binding:"required,min=2,max=200"`
	TemplateID       string                 `json:"templateId" binding:"required"`
	DeviceIDs        []string               `json:"deviceIds" binding:"required,min=1"`
	TemplateVariables map[string]string     `json:"templateVariables"`
	ExecutionStrategy models.ExecutionStrategy `json:"executionStrategy" binding:"required,oneof=serial parallel"`
	Concurrency       int                    `json:"concurrency" binding:"min=1,max=50"`
	Timeout           int                    `json:"timeout" binding:"min=1,max=3600"`
}

// ExecutionListRequest 执行列表请求
type ExecutionListRequest struct {
	Current  int    `json:"current" binding:"min=1"`
	PageSize int    `json:"pageSize" binding:"min=1,max=100"`
}

// listConfigExecutions 查询配置执行列表
func listConfigExecutions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ExecutionListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		current := req.Current
		if current <= 0 {
			current = 1
		}
		pageSize := req.PageSize
		if pageSize <= 0 {
			pageSize = 10
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
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

// executeTemplateByDevices 通过模板执行配置
func executeTemplateByDevices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TemplateExecutionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
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

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
		result, err := svc.ExecuteByTemplate(c.Request.Context(), &services.TemplateExecutionRequest{
			ExecutionName:     req.ExecutionName,
			TemplateID:        req.TemplateID,
			DeviceIDs:         req.DeviceIDs,
			TemplateVariables: req.TemplateVariables,
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

// getConfigExecution 获取配置执行详情
func getConfigExecution(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "执行ID不能为空")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
		result, err := svc.GetExecutionResult(c.Request.Context(), id)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, result)
	}
}

// cancelConfigExecution 取消配置执行
func cancelConfigExecution(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "执行ID不能为空")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
		if err := svc.CancelExecution(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// deleteConfigExecution 删除配置执行记录
func deleteConfigExecution(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			response.Error(c, response.ErrBadRequest, "执行ID不能为空")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
		if err := svc.DeleteExecution(c.Request.Context(), id); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// BatchDeleteExecutionsRequest 批量删除请求
type BatchDeleteExecutionsRequest struct {
	ExecutionIDs []string `json:"executionIds" binding:"required,min=1"`
}

// batchDeleteConfigExecutions 批量删除配置执行记录
func batchDeleteConfigExecutions(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDeleteExecutionsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		manager := device.NewManager(core.DB.GetDB(), core.SM4Cipher, device.DefaultManagerConfig())
		defer manager.Close()

		svc := services.NewConfigExecutionService(core.DB.GetDB(), manager)
		if err := svc.BatchDeleteExecutions(c.Request.Context(), req.ExecutionIDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, nil)
	}
}

// SetupConfigExecutionRouter 设置配置执行路由
func SetupConfigExecutionRouter(r *gin.RouterGroup, core *core.Core) {
	// 执行列表
	r.POST("/list", listConfigExecutions(core))
	// 通过模板执行配置
	r.POST("/template/execute", executeTemplateByDevices(core))
	// 获取执行详情
	r.POST("/:id", getConfigExecution(core))
	// 取消执行
	r.POST("/:id/cancel", cancelConfigExecution(core))
	// 删除执行记录
	r.POST("/:id/delete", deleteConfigExecution(core))
	// 批量删除执行记录
	r.POST("/batch-delete", batchDeleteConfigExecutions(core))
}

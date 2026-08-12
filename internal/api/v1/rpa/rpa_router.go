package rpa

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// SetupPublicWorkerRouter 设置公开的 Worker 路由（不需要认证）
func SetupPublicWorkerRouter(r *gin.RouterGroup, core *core.Core) {
	services := rpa.NewServiceGroup(core.GetDB(), core.Config, core.NoticeHub, core.Cache, nil)
	handler := NewWorkerHandler(services.WorkerService, core)

	// Worker 注册接口（公开，允许匿名访问）
	r.POST("/workers/register", handler.Register)

	// Worker 心跳和进度上报（公开，用于 Worker 通信）
	r.POST("/workers/:id/heartbeat", handler.Heartbeat)
	r.POST("/workers/progress", handler.Progress)
}

// SetupRPARouter 设置 RPA 路由（统一入口）
func SetupRPARouter(r *gin.RouterGroup, core *core.Core) {
	services := rpa.NewServiceGroup(core.GetDB(), core.Config, core.NoticeHub, core.Cache, nil)
	excelService := rpa.NewRPAExcelService(core.GetDB())

	// 任务路由
	SetupTaskRouter(r.Group("/tasks"), NewTaskHandler(services.TaskService, excelService).WithCore(core))

	// Worker 路由（排除已公开的注册接口）
	SetupWorkerRouter(r.Group("/workers"), services, core)

	// 执行记录路由
	SetupExecutionRouter(r.Group("/executions"), NewExecutionHandler(services.ExecutionService, excelService).WithCore(core))

	// 凭证管理路由
	SetupCredentialRouter(r.Group("/credentials"), NewCredentialHandler(services.CredentialService).WithCore(core))

	// AI 辅助路由
	SetupAIRouter(r.Group("/ai"), NewAIHandler(services.AIService).WithCore(core))

	// 流程控制路由（第三阶段高级功能）
	SetupFlowRouter(r.Group("/flow"), services, core)
}

// SetupTaskRouter 设置任务路由
func SetupTaskRouter(r *gin.RouterGroup, handler *TaskHandler) {
	r.POST("/list", handler.List)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/execute", handler.Execute)

	// Excel 相关
	r.POST("/upload-excel", handler.UploadExcel)
	r.POST("/:id/execute-with-excel", handler.ExecuteWithExcel)
}

// SetupWorkerRouter 设置 Worker 路由（认证后才能访问）
func SetupWorkerRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
	handler := NewWorkerHandler(services.WorkerService, core)

	r.POST("/list", handler.List)
	// Worker 统计(专用聚合端点,不依赖分页列表)
	r.POST("/statistics", handler.Statistics)
	// 注意：注册、心跳、进度接口已在公开路由中，这里不需要重复
	// r.POST("/register", handler.Register)
	// r.POST("/:id/heartbeat", handler.Heartbeat)
	// r.POST("/progress", handler.Progress)

	// 扩缩容路由
	r.POST("/:id/scale-up", handler.ScaleUp)
	r.POST("/:id/scale-down", handler.ScaleDown)
	r.POST("/scale-all", handler.ScaleAll)

	// 自动扩缩容配置
	r.GET("/autoscale/config", handler.GetAutoScaleConfig)
	r.POST("/autoscale/config", handler.UpdateAutoScaleConfig)
}

// SetupExecutionRouter 设置执行记录路由
func SetupExecutionRouter(r *gin.RouterGroup, handler *ExecutionHandler) {
	r.POST("/list", handler.List)
	// 执行记录统计(专用 COUNT 聚合,不依赖分页列表)
	r.POST("/statistics", handler.Statistics)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/cancel", handler.Cancel)
	r.POST("/:id/logs", handler.GetLogs)
	r.GET("/:id/download", handler.DownloadArtifacts)

	// 批量报告和人工干预
	r.GET("/:id/batch-report", handler.GetBatchReport)
	r.GET("/:id/human-intervention", handler.RequestHumanIntervention)
	r.POST("/:id/human-intervention", handler.SubmitHumanIntervention)
}

// SetupAIRouter 设置 AI 辅助路由
func SetupAIRouter(r *gin.RouterGroup, handler *AIHandler) {
	// 脚本生成与优化
	r.POST("/generate", handler.GenerateScript)
	r.POST("/optimize", handler.OptimizeScript)

	// AI Agent 决策
	r.POST("/decide", handler.Decide)

	// 错误分析与修复
	r.POST("/analyze-failure", handler.AnalyzeFailure)
	r.POST("/suggest-fix", handler.SuggestFix)
	r.POST("/classify-error", handler.ClassifyError)

	// 选择器学习
	r.POST("/selector/record-success", handler.RecordSelectorSuccess)
	r.POST("/selector/record-failure", handler.RecordSelectorFailure)
	r.POST("/selector/best", handler.GetBestSelector)
	r.POST("/selector/score", handler.ScoreSelector)
	r.POST("/selector/alternatives", handler.GetSelectorAlternatives)
}

// SetupFlowRouter 设置流程控制路由
func SetupFlowRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
	flowService := rpa.NewFlowControlService(services.DB(), services.ExecutionService)
	errorService := rpa.NewErrorHandlingService(services.DB(), flowService, services.ExecutionService)
	mapperService := rpa.NewDataMapperService(services.DB())

	handler := NewFlowHandler(flowService, errorService, mapperService).WithCore(core)

	// 条件评估
	r.POST("/evaluate-condition", handler.EvaluateCondition)

	// 数据映射
	r.POST("/map-data", handler.MapData)
	r.POST("/transform-value", handler.TransformValue)
	r.POST("/extract-jsonpath", handler.ExtractJSONPath)
	r.POST("/aggregate-data", handler.AggregateData)

	// 错误处理
	r.POST("/handle-error", handler.HandleError)
	r.POST("/execute-retry", handler.ExecuteRetry)
}

// SetupCredentialRouter 设置凭证路由
func SetupCredentialRouter(r *gin.RouterGroup, handler *CredentialHandler) {
	// 凭证管理
	r.POST("/list", handler.List)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)

	// 会话管理
	r.POST("/sessions/list", handler.ListSessions)
	r.POST("/sessions/:id/invalidate", handler.InvalidateSession)
}

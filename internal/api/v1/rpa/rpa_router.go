package rpa

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// workerAuthConfig DEBT-03：从 core 配置提取 Worker 认证参数与 Token 服务。
// 认证未配置（白名单为空）时 required 恒为 false（宽限期双读）。
func workerAuthConfig(core *core.Core) (rpa.WorkerTokenService, bool) {
	auth := core.Config.RPA.Worker.Auth
	svc := rpa.NewWorkerTokenService(core.GetDB(), auth)
	required := auth.AuthRequired && len(auth.ProvisioningTokens) > 0
	return svc, required
}

// SetupPublicWorkerRouter 设置公开的 Worker 路由（不需要认证）
func SetupPublicWorkerRouter(r *gin.RouterGroup, core *core.Core) {
	// Phase 103 CONV-03 (D-103-20): 第 6 参数注入 base.CacheProvider
	services := rpa.NewServiceGroup(core.GetDB(), core.Config, core.NoticeHub, core.Cache, nil, system.NewCacheProvider(core.DataCacheService))
	tokenSvc, required := workerAuthConfig(core)
	handler := NewWorkerHandler(services.WorkerService, core).WithTokenService(tokenSvc)

	// DEBT-03：注册接口挂 Provisioning Token 认证。
	// required=false（宽限期，默认）→ 无头放行 WARN，带合法头时 handler 换发 Worker Token；
	// required=true → 无头/错头一律 401。
	r.POST("/workers/register", middleware.ProvisioningAuth(tokenSvc, required), handler.Register)

	// Worker 心跳和进度上报：auth_required=true 时强制 X-Worker-Token（含 :id 归属校验）。
	workerGuard := middleware.WorkerAuth(tokenSvc, required)
	r.POST("/workers/:id/heartbeat", workerGuard, handler.Heartbeat)
	r.POST("/workers/progress", middleware.WorkerAuth(tokenSvc, required), handler.Progress)
}

// SetupRPARouter 设置 RPA 路由（统一入口）
//
// RBAC 接线（2026-11-11 rpa 域权限码全链路）：本组端点级权限在各子 setup 内
// 挂载，对齐 monitor 域先例——读端点 ANY-of 读码、写端点 RequireStrictPermissions
// 单码。权限码定义见 pkg/permission/config.go RPA 域；F 型按钮播种见
// migration_245（rpa RBAC F 型按钮）。
//
// flow 组已下线（2026-11-11）：7 端点为执行引擎内部能力直接暴露（条件评估/
// 数据映射/错误处理），前端零调用、不应公开暴露；内部消费方走 service 层直调，
// 不经 HTTP。原 SetupFlowRouter 与其注册一并移除。
func SetupRPARouter(r *gin.RouterGroup, core *core.Core) {
	// Phase 103 CONV-03 (D-103-20): 第 6 参数注入 base.CacheProvider
	services := rpa.NewServiceGroup(core.GetDB(), core.Config, core.NoticeHub, core.Cache, nil, system.NewCacheProvider(core.DataCacheService))
	excelService := rpa.NewRPAExcelService(core.GetDB())

	// 任务路由
	SetupTaskRouter(r.Group("/tasks"), NewTaskHandler(services.TaskService, excelService).WithCore(core), core)

	// Worker 路由（排除已公开的注册接口）
	SetupWorkerRouter(r.Group("/workers"), services, core)

	// 执行记录路由
	SetupExecutionRouter(r.Group("/executions"), NewExecutionHandler(services.ExecutionService, excelService).WithCore(core), core)

	// 凭证管理路由
	SetupCredentialRouter(r.Group("/credentials"), NewCredentialHandler(services.CredentialService).WithCore(core), core)

	// AI 辅助路由
	SetupAIRouter(r.Group("/ai"), NewAIHandler(services.AIService).WithCore(core), core)
}

// SetupTaskRouter 设置任务路由（读端点 ANY-of ops:rpa:task:list；写端点 strict 单码）
func SetupTaskRouter(r *gin.RouterGroup, handler *TaskHandler, core *core.Core) {
	r.POST("/list", middleware.RequirePermissions([]string{string(permission.RpaTaskList)}, core), handler.List)
	r.POST("", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Create)
	// 详情归读（与 list 同码 ANY-of）
	r.POST("/:id", middleware.RequirePermissions([]string{string(permission.RpaTaskList)}, core), handler.GetByID)
	r.POST("/:id/update", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskEdit)}, core), handler.Update)
	r.POST("/:id/delete", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskRemove)}, core), handler.Delete)
	r.POST("/:id/execute", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskExecute)}, core), handler.Execute)

	// Excel 相关（导入=新增语义、执行=execute 语义）
	r.POST("/upload-excel", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.UploadExcel)
	r.POST("/:id/execute-with-excel", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskExecute)}, core), handler.ExecuteWithExcel)
}

// SetupWorkerRouter 设置 Worker 路由（认证后才能访问；读 ANY-of monitor 码，扩缩容写 strict）
func SetupWorkerRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
	handler := NewWorkerHandler(services.WorkerService, core)

	r.POST("/list", middleware.RequirePermissions([]string{string(permission.RpaWorkerMonitor)}, core), handler.List)
	// Worker 统计(专用聚合端点,不依赖分页列表)
	r.POST("/statistics", middleware.RequirePermissions([]string{string(permission.RpaWorkerMonitor)}, core), handler.Statistics)
	// 注意：注册、心跳、进度接口已在公开路由中，这里不需要重复
	// r.POST("/register", handler.Register)
	// r.POST("/:id/heartbeat", handler.Heartbeat)
	// r.POST("/progress", handler.Progress)

	// 扩缩容路由（写，strict ops:rpa:worker:edit）
	r.POST("/:id/scale-up", middleware.RequireStrictPermissions([]string{string(permission.RpaWorkerEdit)}, core), handler.ScaleUp)
	r.POST("/:id/scale-down", middleware.RequireStrictPermissions([]string{string(permission.RpaWorkerEdit)}, core), handler.ScaleDown)
	r.POST("/scale-all", middleware.RequireStrictPermissions([]string{string(permission.RpaWorkerEdit)}, core), handler.ScaleAll)

	// 自动扩缩容配置（读 ANY-of / 写 strict）
	r.GET("/autoscale/config", middleware.RequirePermissions([]string{string(permission.RpaWorkerMonitor)}, core), handler.GetAutoScaleConfig)
	r.POST("/autoscale/config", middleware.RequireStrictPermissions([]string{string(permission.RpaWorkerEdit)}, core), handler.UpdateAutoScaleConfig)
}

// SetupExecutionRouter 设置执行记录路由（查询/下载/人工干预归读 ANY-of；取消 strict）
func SetupExecutionRouter(r *gin.RouterGroup, handler *ExecutionHandler, core *core.Core) {
	r.POST("/list", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.List)
	// 执行记录统计(专用 COUNT 聚合,不依赖分页列表)
	r.POST("/statistics", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Statistics)
	r.POST("/:id", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.GetByID)
	r.POST("/:id/cancel", middleware.RequireStrictPermissions([]string{string(permission.RpaExecutionCancel)}, core), handler.Cancel)
	r.POST("/:id/logs", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.GetLogs)
	r.GET("/:id/download", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.DownloadArtifacts)

	// 批量报告和人工干预
	r.GET("/:id/batch-report", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.GetBatchReport)
	r.GET("/:id/human-intervention", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.RequestHumanIntervention)
	r.POST("/:id/human-intervention", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.SubmitHumanIntervention)
}

// SetupAIRouter 设置 AI 辅助路由（11 端点统一 strict rpa:ai:use；逐行显式挂载
// 便于 scan-routes.cjs 提取权限码入 registry）
func SetupAIRouter(r *gin.RouterGroup, handler *AIHandler, core *core.Core) {
	// 脚本生成与优化
	r.POST("/generate", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.GenerateScript)
	r.POST("/optimize", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.OptimizeScript)

	// AI Agent 决策
	r.POST("/decide", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.Decide)

	// 错误分析与修复
	r.POST("/analyze-failure", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.AnalyzeFailure)
	r.POST("/suggest-fix", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.SuggestFix)
	r.POST("/classify-error", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.ClassifyError)

	// 选择器学习
	r.POST("/selector/record-success", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.RecordSelectorSuccess)
	r.POST("/selector/record-failure", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.RecordSelectorFailure)
	r.POST("/selector/best", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.GetBestSelector)
	r.POST("/selector/score", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.ScoreSelector)
	r.POST("/selector/alternatives", middleware.RequireStrictPermissions([]string{string(permission.RpaAiUse)}, core), handler.GetSelectorAlternatives)
}

// SetupCredentialRouter 设置凭证路由（读 ANY-of rpa:credential:list；写 strict 单码）
func SetupCredentialRouter(r *gin.RouterGroup, handler *CredentialHandler, core *core.Core) {
	// 凭证管理
	r.POST("/list", middleware.RequirePermissions([]string{string(permission.RpaCredentialList)}, core), handler.List)
	r.POST("", middleware.RequireStrictPermissions([]string{string(permission.RpaCredentialAdd)}, core), handler.Create)
	r.POST("/:id", middleware.RequirePermissions([]string{string(permission.RpaCredentialList)}, core), handler.GetByID)
	r.POST("/:id/update", middleware.RequireStrictPermissions([]string{string(permission.RpaCredentialEdit)}, core), handler.Update)
	r.POST("/:id/delete", middleware.RequireStrictPermissions([]string{string(permission.RpaCredentialRemove)}, core), handler.Delete)

	// 会话管理（列表归读；失效为写操作）
	r.POST("/sessions/list", middleware.RequirePermissions([]string{string(permission.RpaCredentialList)}, core), handler.ListSessions)
	r.POST("/sessions/:id/invalidate", middleware.RequireStrictPermissions([]string{string(permission.RpaCredentialEdit)}, core), handler.InvalidateSession)
}

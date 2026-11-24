package rpa

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：实时画面路由（**尾部追加**，不改既有组）
//
// 归属协议第 6 条（路由统一）与执行组共存策略：不修改 SetupExecutionRouter 的
// 既有行，而是在同一 /executions 前缀下追加 /:id/liveview/* 子组。
//
// 权限裁定（LIVEVIEW-05）：「帧与输入只对有权查看该执行的人开放」—— 复用
// ops:rpa:execution:list（读码，**不新增权限码**），数据范围由既有 DataScope
// 语义在各 service 层保证。
//
// 写法：**逐端点内联显式挂载中间件**（与 SetupAIRouter 同款），不用局部变量。
// 原因：scripts/scan-routes.cjs 只能对 `middleware.RequireXxx(...)` 的字面形态
// 提取权限码；写成 `readCode := middleware.RequirePermissions(...)` 再传变量时
// registry 里的 permCodes 会变成空集（本 phase 实测踩到，逐行内联后恢复可提取）。
// ============================================================================

// SetupLiveViewRouter 注册实时画面端点组（挂在 /rpa/executions 前缀下）。
//
// 十个用户侧端点 + 两个 Worker 上行端点（Worker 侧在 SetupPublicWorkerRouter
// 注册，因需 WorkerAuth 而非用户 RBAC）。
func SetupLiveViewRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
	handler := NewLiveViewHandler(services.LiveViewService, services.ExecutionService).WithCore(core)

	group := r.Group("/:id/liveview")
	group.POST("/watch", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Watch)
	group.POST("/keepalive", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Keepalive)
	group.POST("/stop", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Stop)
	group.POST("/takeover", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Takeover)
	group.POST("/release", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Release)
	group.POST("/mode", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.SetMode)
	group.POST("/continue", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Continue)
	group.POST("/input", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Input)
	group.POST("/abort", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.Abort)
	group.GET("/state", middleware.RequirePermissions([]string{string(permission.RpaExecutionList)}, core), handler.State)
}

package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portwrite"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
)

// SetupPortWriteRouter 注册端口写操作路由 (Phase 52 D-09 + INFRA-02 + DB-REFRESH-2026-07-08)
//
// 在现有 ports 组下挂子组 /network/ports/write/* + 组级 2-arg RequirePermissions
// (require permission.NetworkPortWrite = "network:port:write")。
//
// 6 端点 kebab 命名（与现有 /list /collect /batch-delete 同风格）：
//
//	POST /network/ports/write/shutdown
//	POST /network/ports/write/undo-shutdown
//	POST /network/ports/write/description
//	POST /network/ports/write/dot1x-enable
//	POST /network/ports/write/dot1x-disable
//	POST /network/ports/write/batch
//
// 关键签名 (RESEARCH §1.4 VERIFIED)：middleware.RequirePermissions 是 2-arg
//
//	RequirePermissions(permissions []string, core *core.Core)
//
// CONTEXT/CLAUDE.md 漏写第二参 core；本函数严格传 (perm, core) 两参。
//
// 2026-07-08 修复：第三个注入参数从 core.DeviceInfoCollectionService（采设备级，不刷端口表）
// 改为本地新建的 portcollection.CollectionService 实例（同步刷新 sys_device_port_status）——
// 与 port_handler.go 既有 NewPortCollectionService 模式一致；未注入到 core.Core（避免
// 跨模块不必要耦合，port_collection cron 与 port_handler 都各自 New 一个）。
func SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core) {
	svc := portwrite.NewPortWriteService(
		core.GetDB(),
		core.DeviceExecutor,
		portcollection.NewCollectionService(core.GetDB(), core.DeviceExecutor),
	)
	handler := NewPortWriteHandler(svc).WithCore(core)

	write := r.Group("/write")
	// 组级 RBAC（一处覆盖 6 端点；superadmin 走 core 旁路自动放行）
	write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core))

	write.POST("/shutdown", handler.Shutdown)
	write.POST("/undo-shutdown", handler.UndoShutdown)
	write.POST("/description", handler.SetDescription)
	write.POST("/dot1x-enable", handler.EnableDot1x)
	write.POST("/dot1x-disable", handler.DisableDot1x)
	write.POST("/batch", handler.BatchWrite)
}

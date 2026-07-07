package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portwrite"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
)

// SetupPortWriteRouter 注册端口写操作路由 (Phase 52 D-09 + INFRA-02)
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
func SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core) {
	svc := portwrite.NewPortWriteService(
		core.GetDB(),
		core.DeviceExecutor,
		core.DeviceInfoCollectionService,
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

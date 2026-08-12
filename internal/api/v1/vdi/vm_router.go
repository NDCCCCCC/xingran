package vdi

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
)

// SetupVMRouter 设置虚拟机路由
// 始终注册路由，使用动态客户端
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建服务实例，传入nil客户端（服务层会动态查找）
	vmService := vdiServices.NewVMServiceWithDynamicClient(core.GetDB())
	vmHandler := NewVMHandler(vmService, core.GetDB()).WithCore(core)

	// 注册路由
	r.POST("/list", middleware.RequirePermissions([]string{"vdi:vm:query"}, core), vmHandler.List)
	r.POST("/resource-groups", vmHandler.ListResourceGroups)
	r.POST("/resources", vmHandler.ListResources)
	r.POST("", middleware.RequirePermissions([]string{"vdi:vm:add"}, core), vmHandler.Create)
	r.POST("/:id", vmHandler.GetByID)
	r.POST("/:id/update", middleware.RequirePermissions([]string{"vdi:vm:edit"}, core), vmHandler.Update)
	r.POST("/:id/delete", middleware.RequirePermissions([]string{"vdi:vm:remove"}, core), vmHandler.Delete)

	// VDI 电源操作 — 使用细粒度权限
	r.POST("/start", middleware.RequirePermissions([]string{"vdi:vm:start"}, core), vmHandler.StartVM)
	r.POST("/stop", middleware.RequirePermissions([]string{"vdi:vm:stop"}, core), vmHandler.StopVM)
	r.POST("/restart", middleware.RequirePermissions([]string{"vdi:vm:restart"}, core), vmHandler.RestartVM)

	// 用户绑定操作
	r.POST("/:id/bind_user", middleware.RequirePermissions([]string{"vdi:vm:bind"}, core), vmHandler.BindUser)
	r.POST("/:id/unbind_user", middleware.RequirePermissions([]string{"vdi:vm:bind"}, core), vmHandler.UnbindUser)

	// 同步操作
	r.POST("/:id/sync", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncFromVDI)
	r.POST("/sync-all", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncAll)

	// VDI 创建虚拟机相关接口 — 需要 vdi:vm:add 权限
	r.POST("/vtp-platforms", middleware.RequirePermissions([]string{"vdi:vm:add"}, core), vmHandler.ListVTPPlatforms)
	r.POST("/run-positions", middleware.RequirePermissions([]string{"vdi:vm:add"}, core), vmHandler.ListRunPositions)
	r.POST("/storages", middleware.RequirePermissions([]string{"vdi:vm:add"}, core), vmHandler.ListStorages)
	r.POST("/networks", middleware.RequirePermissions([]string{"vdi:vm:add"}, core), vmHandler.ListNetworks)
}

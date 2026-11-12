package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// SetupMACRouter 设置MAC地址路由
//
// RBAC 接线（task-4 杂项 RBAC，端点级中间件，对齐 vdi/vm_router 先例）：
//   - 读（list/statistics）→ RequirePermissions ANY-of([network:mac:query])
//   - 采集（collect/collect/all，触发 SNMP 采集入库）→ strict(network:mac:collect)
//   - 清理（clean/batch-delete）→ strict(network:mac:remove)
//   - 导出 → strict(network:mac:query)（导出按 FIX-03 惯例走严格单一读码）
func SetupMACRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler) {
	handler := NewMACHandler(core)

	macRead := []string{string(permission.NetworkMacQuery)}
	macCollect := []string{string(permission.NetworkMacCollect)}
	macRemove := []string{string(permission.NetworkMacRemove)}

	r.POST("/list", middleware.RequirePermissions(macRead, core), handler.List)
	r.POST("/collect", middleware.RequireStrictPermissions(macCollect, core), handler.Collect)
	r.POST("/collect/all", middleware.RequireStrictPermissions(macCollect, core), handler.CollectAll)
	r.GET("/statistics", middleware.RequirePermissions(macRead, core), handler.GetStats)
	r.POST("/clean", middleware.RequireStrictPermissions(macRemove, core), handler.Clean)
	r.POST("/batch-delete", middleware.RequireStrictPermissions(macRemove, core), handler.BatchDelete)
	r.POST("/export", middleware.RequireStrictPermissions(macRead, core), exportHandler.ExportMACAddresses)
}

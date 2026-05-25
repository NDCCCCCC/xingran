package vdi

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	vdiServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/vdi"
)

// SetupVMRouter 设置虚拟机路由
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建VDI客户端（使用第一个服务器配置）
	if len(core.Config.VDI.Servers) == 0 {
		return // 没有配置VDI服务器，跳过路由注册
	}

	serverCfg := core.Config.VDI.Servers[0]
	vdiClient := vdiServices.NewVDIClientExtended(
		core.GetDB(),
		"", // serverID将在运行时确定
		serverCfg,
	)

	// 创建服务实例
	vmService := vdiServices.NewVMService(core.GetDB(), vdiClient)
	vmHandler := NewVMHandler(vmService)

	// 注册路由
	r.POST("/list", vmHandler.List)
	r.POST("", vmHandler.Create)
	r.POST("/:id", vmHandler.GetByID)
	r.POST("/:id/update", vmHandler.Update)
	r.POST("/:id/delete", vmHandler.Delete)

	// VDI特定操作（调用VDI API）
	r.POST("/operate", vmHandler.Operate)
	r.POST("/config_ip", vmHandler.ConfigIP)
	r.POST("/:id/rename", vmHandler.Rename)
	r.POST("/:id/bind_user", vmHandler.BindUser)
	r.POST("/:id/unbind_user", vmHandler.UnbindUser)
	r.POST("/:id/sync", vmHandler.SyncFromVDI)
}

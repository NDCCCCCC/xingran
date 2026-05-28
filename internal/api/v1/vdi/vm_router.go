package vdi

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	vdiServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/vdi"
)

// SetupVMRouter 设置虚拟机路由
// 始终注册路由，使用动态客户端
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建服务实例，传入nil客户端（服务层会动态查找）
	vmService := vdiServices.NewVMServiceWithDynamicClient(core.GetDB())
	vmHandler := NewVMHandler(vmService)

	// 注册路由
	r.POST("/list", vmHandler.List)
	r.POST("/resource-groups", vmHandler.ListResourceGroups)
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
	r.POST("/sync-all", vmHandler.SyncAll)
}

package vdi

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	vdiServices "github.com/xingran-next/xingran-go-backend/internal/services/vdi"
)

// SetupVDIServerRouter 设置VDI服务器路由
func SetupVDIServerRouter(r *gin.RouterGroup, core *core.Core) {
	serverService := vdiServices.NewVDIServerService(core.GetDB())
	serverHandler := NewVDIServerHandler(serverService).WithCore(core)

	r.POST("/list", serverHandler.List)
	r.POST("", serverHandler.Create)
	r.POST("/:id", serverHandler.GetByID)
	r.POST("/:id/update", serverHandler.Update)
	r.POST("/:id/delete", serverHandler.Delete)
	r.POST("/:id/test", serverHandler.TestConnection)
}

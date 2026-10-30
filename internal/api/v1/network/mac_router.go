package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// SetupMACRouter 设置MAC地址路由
func SetupMACRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler) {
	handler := NewMACHandler(core)

	r.POST("/list", handler.List)
	r.POST("/collect", handler.Collect)
	r.POST("/collect/all", handler.CollectAll)
	r.GET("/statistics", handler.GetStats)
	r.POST("/clean", handler.Clean)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("/export", exportHandler.ExportMACAddresses)
}

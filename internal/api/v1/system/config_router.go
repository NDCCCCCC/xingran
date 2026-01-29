package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupConfigRouter 设置参数配置路由
func SetupConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建参数配置服务
	configService := systemServices.NewConfigService(core.DB.GetDB())
	configHandler := NewConfigHandler(configService)

	// 参数配置路由
	r.POST("/list", configHandler.List)
	r.POST("/batch-delete", configHandler.BatchDelete)
	r.POST("/refresh-cache", configHandler.RefreshCache)
	r.POST("", configHandler.Create)
	r.POST("/:id", configHandler.GetByID)
	r.GET("/key/:configKey", configHandler.GetByKey)
	r.POST("/:id/update", configHandler.Update)
	r.POST("/:id/delete", configHandler.Delete)
}

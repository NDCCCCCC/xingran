package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
)

// SetupConfigRouter 设置参数配置路由
func SetupConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的参数配置服务
	var configService systemServices.ConfigService
	if core.DataCacheService != nil {
		configService = systemServices.NewConfigServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		configService = systemServices.NewConfigService(core.DB.GetDB())
	}

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

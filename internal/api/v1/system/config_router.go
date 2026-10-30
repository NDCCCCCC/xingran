package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// ConfigWriteRouteGroups 参数配置**写操作**路由分组（85-03 RBAC-01）。
type ConfigWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupConfigRouter 设置参数配置**读**路由（写操作见 SetupConfigWriteRouter）。
func SetupConfigRouter(r *gin.RouterGroup, core *core.Core) {
	configHandler := newConfigHandler(core)

	// 参数配置读路由
	r.POST("/list", configHandler.List)
	// 参数配置统计(专用 COUNT 聚合,不依赖分页列表)
	r.POST("/statistics", configHandler.Statistics)
	r.POST("/:id", configHandler.GetByID)
	r.GET("/key/:configKey", configHandler.GetByKey)
}

// SetupConfigWriteRouter 设置参数配置**写**路由（85-03 RBAC-01 逐动作拆分；
// refresh-cache 刷新系统配置缓存归编辑语义）。
func SetupConfigWriteRouter(groups ConfigWriteRouteGroups, core *core.Core) {
	configHandler := newConfigHandler(core)

	groups.Add.POST("", configHandler.Create)
	groups.Edit.POST("/:id/update", configHandler.Update)
	groups.Edit.POST("/refresh-cache", configHandler.RefreshCache)
	groups.Remove.POST("/:id/delete", configHandler.Delete)
	groups.Remove.POST("/batch-delete", configHandler.BatchDelete)
}

// newConfigHandler 构造参数配置处理器（含缓存适配；读/写路由共用）。
func newConfigHandler(core *core.Core) *ConfigHandler {
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

	return NewConfigHandler(configService, core.CaptchaService).WithCore(core)
}

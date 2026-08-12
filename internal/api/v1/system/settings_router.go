package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
)

// SetupSettingsRouter 设置系统设置路由
func SetupSettingsRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建配置服务（settings 和 default_theme 都依赖它）
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

	// 创建带缓存的系统设置服务
	var settingsService systemServices.SettingsService
	if core.DataCacheService != nil {
		settingsService = systemServices.NewSettingsServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
			configService,
		)
	} else {
		settingsService = systemServices.NewSettingsService(core.DB.GetDB(), configService)
	}

	settingsHandler := NewSettingsHandler(settingsService).WithCore(core)

	// 系统设置路由（所有已登录用户可访问）
	r.GET("/preferences", settingsHandler.GetUserPreferences)
	r.PUT("/preferences", settingsHandler.UpdateUserPreferences)

	// 创建默认主题服务和处理器（复用上面的 configService）
	defaultThemeService := systemServices.NewDefaultThemeService(core.DB.GetDB(), configService)
	defaultThemeHandler := NewDefaultThemeHandler(defaultThemeService).WithCore(core)

	// 默认主题路由 — GET 公开给所有登录用户（用于前端 reset / 用户首次访问应用管理员默认）
	//                    POST 仍需 system:config:manage 权限（仅管理员可修改）
	r.GET("/config/theme/default", defaultThemeHandler.GetDefaultThemeConfig)

	configGroup := r.Group("/config")
	configGroup.Use(middleware.Permission("system:config:manage", core))
	{
		configGroup.POST("/theme/default", defaultThemeHandler.SetDefaultThemeConfig)
		configGroup.POST("/theme/sync", defaultThemeHandler.SyncUserThemeToDefault)
	}
}

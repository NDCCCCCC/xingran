package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupSettingsRouter 设置系统设置路由
func SetupSettingsRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的系统设置服务（v1.22 收尾：移除默认主题依赖，不再注入 ConfigService）
	var settingsService systemServices.SettingsService
	if core.DataCacheService != nil {
		settingsService = systemServices.NewSettingsServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		settingsService = systemServices.NewSettingsService(core.DB.GetDB())
	}

	settingsHandler := NewSettingsHandler(settingsService).WithCore(core)

	// 系统设置路由（所有已登录用户可访问）
	r.GET("/preferences", settingsHandler.GetUserPreferences)
	r.PUT("/preferences", settingsHandler.UpdateUserPreferences)
}
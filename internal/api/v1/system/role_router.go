package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupRoleRouter 设置角色路由
func SetupRoleRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的角色服务
	var roleService systemServices.RoleService
	if core.DataCacheService != nil {
		roleService = systemServices.NewRoleServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		roleService = systemServices.NewRoleService(core.DB.GetDB())
	}

	// 创建Handler
	handler := NewRoleHandler(roleService).WithCore(core)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("/statistics", handler.Statistics)
	r.POST("/all", handler.GetAllEnabled)
	r.POST("/batch", handler.BatchDelete)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
}

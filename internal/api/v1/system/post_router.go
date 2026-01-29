package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupPostRouter 设置岗位路由
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = NewDataCacheAdapter(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

	// 创建带缓存的岗位服务
	var postService systemServices.PostService
	if core.DataCacheService != nil {
		postService = systemServices.NewPostServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		postService = systemServices.NewPostService(core.DB.GetDB())
	}

	// 创建Handler
	handler := NewPostHandler(postService)

	// 注册路由
	r.POST("/list", handler.List)
	r.POST("/all", handler.GetAll)
	r.POST("/enabled", handler.GetAllEnabled)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
}

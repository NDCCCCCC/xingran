package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupPostRouter 设置岗位路由
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

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
	handler := NewPostHandler(postService).WithCore(core)

	// 注册路由
	r.POST("/list", handler.List)
	// 岗位统计(专用 COUNT 聚合,不依赖分页列表)
	r.POST("/statistics", handler.Statistics)
	r.POST("/all", handler.GetAll)
	r.POST("/enabled", handler.GetAllEnabled)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("", handler.Create)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
}

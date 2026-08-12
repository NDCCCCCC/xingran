package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupDictRouter 设置字典路由
func SetupDictRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的字典类型服务
	var dictTypeService systemServices.DictTypeService
	if core.DataCacheService != nil {
		dictTypeService = systemServices.NewDictTypeServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		dictTypeService = systemServices.NewDictTypeService(core.DB.GetDB())
	}
	dictTypeHandler := NewDictTypeHandler(dictTypeService).WithCore(core)

	// 字典类型路由
	types := r.Group("/types")
	{
		types.POST("/list", dictTypeHandler.List)
		// 字典类型统计(专用 COUNT 聚合,不依赖分页列表)
		types.POST("/statistics", dictTypeHandler.Statistics)
		types.POST("/all", dictTypeHandler.GetAll)
		types.POST("", dictTypeHandler.Create)
		types.POST("/:id", dictTypeHandler.GetByID)
		types.POST("/:id/update", dictTypeHandler.Update)
		types.POST("/:id/delete", dictTypeHandler.Delete)
	}

	// 创建带缓存的字典数据服务
	var dictDataService systemServices.DictDataService
	if core.DataCacheService != nil {
		dictDataService = systemServices.NewDictDataServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		dictDataService = systemServices.NewDictDataService(core.DB.GetDB())
	}
	dictDataHandler := NewDictDataHandler(dictDataService).WithCore(core)

	// 字典数据路由
	data := r.Group("/data")
	{
		data.POST("/list", dictDataHandler.List)
		// 字典数据统计(专用 COUNT 聚合,支持按 dictType 过滤)
		data.POST("/statistics", dictDataHandler.Statistics)
		data.POST("", dictDataHandler.Create)
		data.POST("/:id", dictDataHandler.GetByID)
		data.POST("/:id/update", dictDataHandler.Update)
		data.POST("/:id/delete", dictDataHandler.Delete)
	}
}

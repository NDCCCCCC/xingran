package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDictRouter 设置字典路由
func SetupDictRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = NewDataCacheAdapter(core.DataCacheService, core.Cache)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

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
	dictTypeHandler := NewDictTypeHandler(dictTypeService)

	// 字典类型路由
	types := r.Group("/types")
	{
		types.POST("/list", dictTypeHandler.List)
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
	dictDataHandler := NewDictDataHandler(dictDataService)

	// 字典数据路由
	data := r.Group("/data")
	{
		data.POST("/list", dictDataHandler.List)
		data.POST("", dictDataHandler.Create)
		data.POST("/:id", dictDataHandler.GetByID)
		data.POST("/:id/update", dictDataHandler.Update)
		data.POST("/:id/delete", dictDataHandler.Delete)
	}
}

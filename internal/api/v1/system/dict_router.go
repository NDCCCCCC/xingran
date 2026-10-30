package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// DictWriteRouteGroups 字典**写操作**路由分组（85-03 RBAC-01，范式对齐 DeptWriteRouteGroups）。
// 类型/数据子资源共用 system:dict:* 码族（XingRan 惯例，无独立 dict:data:* 码）。
// ⚠️ 字段必须扁平到叶子组（groups.<Field>.POST 直挂形态，扫描器按此解析挂载）。
type DictWriteRouteGroups struct {
	TypesAdd    *gin.RouterGroup
	TypesEdit   *gin.RouterGroup
	TypesRemove *gin.RouterGroup
	DataAdd     *gin.RouterGroup
	DataEdit    *gin.RouterGroup
	DataRemove  *gin.RouterGroup
}

// SetupDictRouter 设置字典**读**路由
//
// 读操作保留组级 ANY-of 权限（router.go 施加，含 OpsSelectorReadPerms queryExtra
// 放行）；写操作不在本函数注册——见 SetupDictWriteRouter。
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

	// 字典类型读路由
	types := r.Group("/types")
	{
		types.POST("/list", dictTypeHandler.List)
		// 字典类型统计(专用 COUNT 聚合,不依赖分页列表)
		types.POST("/statistics", dictTypeHandler.Statistics)
		types.POST("/all", dictTypeHandler.GetAll)
		types.POST("/:id", dictTypeHandler.GetByID)
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

	// 字典数据读路由
	data := r.Group("/data")
	{
		data.POST("/list", dictDataHandler.List)
		// 字典数据统计(专用 COUNT 聚合,支持按 dictType 过滤)
		data.POST("/statistics", dictDataHandler.Statistics)
		data.POST("/:id", dictDataHandler.GetByID)
	}
}

// SetupDictWriteRouter 设置字典**写**路由（85-03 RBAC-01 逐动作拆分，
// 类型/数据子资源共用码族）。
func SetupDictWriteRouter(groups DictWriteRouteGroups, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	var dictTypeService systemServices.DictTypeService
	if core.DataCacheService != nil {
		dictTypeService = systemServices.NewDictTypeServiceWithCache(
			core.DB.GetDB(), cacheProvider, core.CacheConfigService,
		)
	} else {
		dictTypeService = systemServices.NewDictTypeService(core.DB.GetDB())
	}
	dictTypeHandler := NewDictTypeHandler(dictTypeService).WithCore(core)

	var dictDataService systemServices.DictDataService
	if core.DataCacheService != nil {
		dictDataService = systemServices.NewDictDataServiceWithCache(
			core.DB.GetDB(), cacheProvider, core.CacheConfigService,
		)
	} else {
		dictDataService = systemServices.NewDictDataService(core.DB.GetDB())
	}
	dictDataHandler := NewDictDataHandler(dictDataService).WithCore(core)

	// 字典类型写路由（子前缀 /types 写在路由 path，叶子组挂 /system/dicts 根）
	groups.TypesAdd.POST("/types", dictTypeHandler.Create)
	groups.TypesEdit.POST("/types/:id/update", dictTypeHandler.Update)
	groups.TypesRemove.POST("/types/:id/delete", dictTypeHandler.Delete)

	// 字典数据写路由
	groups.DataAdd.POST("/data", dictDataHandler.Create)
	groups.DataEdit.POST("/data/:id/update", dictDataHandler.Update)
	groups.DataRemove.POST("/data/:id/delete", dictDataHandler.Delete)
}

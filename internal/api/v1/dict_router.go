package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDictRouter 设置字典路由
func SetupDictRouter(r *gin.RouterGroup, core *core.Core) {
	// 字典类型路由
	types := r.Group("/types")
	{
		types.POST("/list", listDictTypes(core))
		types.POST("", createDictType(core))
		types.POST("/:id", getDictType(core))
		types.POST("/:id/update", updateDictType(core))
		types.POST("/:id/delete", deleteDictType(core))
		types.POST("/batch-delete", batchDeleteDictTypes(core))
	}

	// 字典数据路由
	data := r.Group("/data")
	{
		data.POST("/list", listDictData(core))
		data.GET("/type/:dictType", getDictDataByType(core))
		data.POST("", createDictData(core))
		data.POST("/:id", getDictData(core))
		data.POST("/:id/update", updateDictData(core))
		data.POST("/:id/delete", deleteDictData(core))
		data.POST("/batch-delete", batchDeleteDictData(core))
	}

	// 刷新字典缓存
	r.POST("/refresh-cache", refreshDictCache(core))
}

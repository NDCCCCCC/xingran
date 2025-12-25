package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupConfigRouter 设置参数配置路由
func SetupConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询参数配置列表
	r.POST("/list", listConfigs(core))
	// 创建参数配置
	r.POST("", createConfig(core))
	// 获取参数配置详情
	r.POST("/:id", getConfig(core))
	// 根据键获取参数配置
	r.GET("/key/:configKey", getConfigByKey(core))
	// 更新参数配置
	r.POST("/:id/update", updateConfig(core))
	// 删除参数配置
	r.POST("/:id/delete", deleteConfig(core))
	// 批量删除参数配置
	r.POST("/batch-delete", batchDeleteConfigs(core))
	// 刷新参数缓存
	r.POST("/refresh-cache", refreshConfigCache(core))
}

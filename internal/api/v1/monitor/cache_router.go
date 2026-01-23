package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
)

// SetupCacheRouter 设置缓存监控路由
func SetupCacheRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存配置服务
	cacheConfigService := services.NewCacheConfigService(core.DB.GetDB())

	// 创建Handler
	cacheHandler := NewCacheHandler(core, cacheConfigService)

	// ==================== 缓存信息路由 ====================
	r.POST("/cache/list", cacheHandler.GetCacheList)
	r.GET("/cache/:key", cacheHandler.GetCacheInfo)

	// ==================== 缓存操作路由 ====================
	r.POST("/cache/operate", cacheHandler.OperateCache)
	r.POST("/cache/batch", cacheHandler.BatchOperateCache)
	r.POST("/cache/clear", cacheHandler.ClearCache)

	// ==================== 缓存统计路由 ====================
	r.POST("/cache/stats/list", cacheHandler.GetCacheStats)
	r.POST("/cache/monitor", cacheHandler.GetCacheMonitor)

	// ==================== 缓存导出路由 ====================
	r.POST("/cache/export", cacheHandler.ExportCache)

	// ==================== 缓存配置管理路由 ====================
	r.GET("/cache/config", cacheHandler.GetCacheConfigs)
	r.PUT("/cache/config", cacheHandler.UpdateCacheConfig)
	r.POST("/cache/config/reload", cacheHandler.ReloadCacheConfigs)

	// ==================== 测试端点路由 ====================
	r.GET("/cache/test", cacheHandler.TestCacheEndpoint)
	r.GET("/cache/debug/raw-keys", cacheHandler.DebugRawKeys)
}

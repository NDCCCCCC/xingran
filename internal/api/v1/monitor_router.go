package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// RegisterMonitorRoutes 注册系统监控相关路由
func RegisterMonitorRoutes(router *gin.RouterGroup, core *core.Core) {
	// 服务器信息
	router.POST("/server-info/list", getServerInfo(core))

	// 系统指标
	router.POST("/server-metrics/current", getCurrentServerMetrics(core))
	router.POST("/server-metrics/save", saveSystemMetrics(core))
	router.POST("/server-metrics/history", getSystemMetricsHistory(core))

	// 操作日志
	router.POST("/oper-logs/list", getOperLogs(core))
	router.POST("/oper-logs/clean", cleanOperLogs(core))

	// 登录日志
	router.POST("/login-logs/list", getLoginLogs(core))
	router.POST("/login-logs/clean", cleanLoginLogs(core))
}

// RegisterJobRoutes 注册定时任务相关路由
func RegisterJobRoutes(router *gin.RouterGroup, core *core.Core) {
	// 任务管理
	router.POST("/jobs/list", listJobs(core))
	router.POST("/jobs", createJob(core))
	router.GET("/jobs/:id", getJob(core))
	router.PUT("/jobs/:id", updateJob(core))
	router.DELETE("/jobs/:id", deleteJob(core))
	router.PUT("/jobs/:id/status", updateJobStatus(core))
	router.POST("/jobs/:id/execute", executeJob(core))

	// 任务日志
	router.POST("/jobs/logs/list", getJobLogs(core))
}

// RegisterCacheRoutes 注册缓存管理相关路由
func RegisterCacheRoutes(router *gin.RouterGroup, core *core.Core) {
	// 缓存信息
	router.POST("/cache/list", getCacheList(core))
	router.GET("/cache/:key", getCacheInfo(core))

	// 缓存操作
	router.POST("/cache/operate", operateCache(core))
	router.POST("/cache/batch", batchOperateCache(core))
	router.POST("/cache/clear", clearCache(core))

	// 缓存统计
	router.POST("/cache/stats/list", getCacheStats(core))
	router.POST("/cache/monitor", getCacheMonitor(core))

	// 缓存导出
	router.POST("/cache/export", exportCache(core))
}
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupMonitorRouter 设置监控路由
func SetupMonitorRouter(r *gin.RouterGroup, core *core.Core) {
	// 注册系统监控相关路由
	RegisterMonitorRoutes(r, core)
	// 注册定时任务相关路由
	RegisterJobRoutes(r, core)
	// 注册缓存管理相关路由
	RegisterCacheRoutes(r, core)
}

// SetupNetworkDeviceRouter 设置网络设备路由
func SetupNetworkDeviceRouter(r *gin.RouterGroup, core *core.Core) {
	// 设备统计（用于仪表盘Widget）
	r.GET("/statistics", getNetworkDeviceStatistics(core))
	// 设备列表
	r.POST("/list", listNetworkDevices(core))
	// 设备探测（SNMP探测获取设备信息）
	r.POST("/discover", discoverNetworkDevice(core))
	// 快速创建设备（自动探测并创建）
	r.POST("/quick-create", quickCreateNetworkDevice(core))
	// 创建设备
	r.POST("", createNetworkDevice(core))
	// 获取设备详情
	r.POST("/:id", getNetworkDevice(core))
	// 更新设备
	r.POST("/:id/update", updateNetworkDevice(core))
	// 删除设备
	r.POST("/:id/delete", deleteNetworkDevice(core))
	// 批量删除设备
	r.POST("/batch-delete", batchDeleteNetworkDevices(core))
	// 获取设备补充任务状态
	r.GET("/:id/enrichment-status", getEnrichmentStatus(core))
}

// SetupAuthCredentialRouter 设置授权凭证路由
func SetupAuthCredentialRouter(r *gin.RouterGroup, core *core.Core) {
	// 凭证列表
	r.POST("/list", listAuthCredentials(core))
	// 创建凭证
	r.POST("", createAuthCredential(core))
	// 获取凭证详情
	r.POST("/:id", getAuthCredential(core))
	// 更新凭证
	r.POST("/:id/update", updateAuthCredential(core))
	// 删除凭证
	r.POST("/:id/delete", deleteAuthCredential(core))
	// 批量删除凭证
	r.POST("/batch-delete", batchDeleteAuthCredentials(core))
	// 设置默认凭证
	r.POST("/:id/set-default", setDefaultCredential(core))
	// 获取使用凭证的设备列表
	r.POST("/:id/devices", getDevicesByCredential(core))
}

// SetupConfigTemplateRouter 设置配置模板路由
func SetupConfigTemplateRouter(r *gin.RouterGroup, core *core.Core) {
	// 模板列表
	r.POST("/list", listConfigTemplates(core))
	// 创建模板
	r.POST("", createConfigTemplate(core))
	// 获取模板详情
	r.POST("/:id", getConfigTemplate(core))
	// 更新模板
	r.POST("/:id/update", updateConfigTemplate(core))
	// 删除模板
	r.POST("/:id/delete", deleteConfigTemplate(core))
	// 批量删除模板
	r.POST("/batch-delete", batchDeleteConfigTemplates(core))
	// 预览模板
	r.POST("/:id/preview", previewConfigTemplate(core))
	// 克隆模板
	r.POST("/:id/clone", cloneConfigTemplate(core))
	// 获取模板变量
	r.POST("/:id/variables", getTemplateVariables(core))
}

// SetupCommandRouter 设置命令分发路由
func SetupCommandRouter(r *gin.RouterGroup, core *core.Core) {
	// 分发命令
	r.POST("/dispatch", dispatchCommand(core))
	// 快速命令
	r.POST("/quick", quickCommand(core))
	// 获取执行结果
	r.POST("/:id", getCommandExecution(core))
	// 获取执行列表
	r.POST("/list", listCommandExecutions(core))
	// 获取设备执行明细
	r.POST("/:id/device/:deviceId", getDeviceExecutionDetail(core))
}

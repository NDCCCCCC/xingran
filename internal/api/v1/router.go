package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupPostRouter 设置岗位路由
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现岗位管理路由
}

// SetupDictRouter 设置字典路由
func SetupDictRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现字典管理路由
}

// SetupConfigRouter 设置配置路由
func SetupConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现配置管理路由
}

// SetupNoticeRouter 设置通知路由
func SetupNoticeRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现通知管理路由
}

// SetupOperLogRouter 设置操作日志路由
func SetupOperLogRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现操作日志路由
}

// SetupLoginLogRouter 设置登录日志路由
func SetupLoginLogRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现登录日志路由
}

// SetupMonitorRouter 设置监控路由
func SetupMonitorRouter(r *gin.RouterGroup, core *core.Core) {
	// 注册系统监控相关路由
	RegisterMonitorRoutes(r, core)
	// 注册定时任务相关路由
	RegisterJobRoutes(r, core)
	// 注册缓存管理相关路由
	RegisterCacheRoutes(r, core)
}
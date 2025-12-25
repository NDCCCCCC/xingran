package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupPostRouter 设置岗位路由
func SetupPostRouter(r *gin.RouterGroup, core *core.Core) {
	// TODO: 实现岗位管理路由
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
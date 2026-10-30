package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// SetupMonitorRouter 设置监控路由（已废弃，仅保留向后兼容）
// 注意：服务器监控和缓存监控已迁移到新架构
// 建议直接使用 monitorV1.SetupServerRouter 和 monitorV1.SetupCacheRouter
func SetupMonitorRouter(r *gin.RouterGroup, core *core.Core) {
	// 注册定时任务相关路由
	RegisterJobRoutes(r, core)
	// 注意：服务器监控和缓存监控已在主路由中直接注册
}

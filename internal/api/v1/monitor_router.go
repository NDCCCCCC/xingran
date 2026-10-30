package v1

import (
	"github.com/gin-gonic/gin"
	schedulerV1 "github.com/xingran-next/xingran-go-backend/internal/api/v1/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// RegisterJobRoutes 注册定时任务相关路由
func RegisterJobRoutes(router *gin.RouterGroup, core *core.Core) {
	// 使用新架构：结构体Handler + Service层
	jobs := router.Group("/jobs")
	schedulerV1.SetupJobRouter(jobs, core)
}

package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	schedulerV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/scheduler"
)

// RegisterJobRoutes 注册定时任务相关路由
func RegisterJobRoutes(router *gin.RouterGroup, core *core.Core) {
	// 使用新架构：结构体Handler + Service层
	jobs := router.Group("/jobs")
	schedulerV1.SetupJobRouter(jobs, core)
}

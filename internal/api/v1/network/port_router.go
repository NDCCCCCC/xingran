package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupPortRouter 设置端口状态路由
func SetupPortRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewPortHandler(core)

	r.POST("/list", handler.List)           // 查询端口状态列表
	r.POST("/collect", handler.Collect)     // 采集单个设备端口状态
	r.POST("/collect-all", handler.CollectAll) // 采集所有设备端口状态
	r.GET("/statistics", handler.GetStats)  // 获取端口状态统计
	r.POST("/clean", handler.Clean)         // 清理旧记录
	r.POST("/batch-delete", handler.BatchDelete) // 批量删除
}

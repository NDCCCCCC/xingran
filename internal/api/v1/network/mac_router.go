package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupMACRouter 设置MAC地址路由
func SetupMACRouter(r *gin.RouterGroup, core *core.Core) {
	handler := NewMACHandler(core)

	r.POST("/list", handler.List)              // 查询MAC地址列表
	r.POST("/collect", handler.Collect)        // 采集单个设备MAC地址
	r.POST("/collect/all", handler.CollectAll) // 采集所有设备MAC地址
	r.GET("/statistics", handler.GetStats)     // 获取MAC地址统计
	r.POST("/clean", handler.Clean)            // 清理旧记录
}

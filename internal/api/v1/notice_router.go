package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupNoticeRouter 设置通知公告路由
func SetupNoticeRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询通知公告列表
	r.POST("/list", listNotices(core))
	// 创建通知公告
	r.POST("", createNotice(core))
	// 获取通知公告详情
	r.POST("/:id", getNotice(core))
	// 更新通知公告
	r.POST("/:id/update", updateNotice(core))
	// 删除通知公告
	r.POST("/:id/delete", deleteNotice(core))
	// 批量删除通知公告
	r.POST("/batch-delete", batchDeleteNotices(core))
	// 获取通知阅读统计
	r.GET("/:id/statistics", getNoticeStatistics(core))
	// 发布通知
	r.POST("/:id/publish", publishNotice(core))
	// 撤回/取消发布通知
	r.POST("/:id/withdraw", withdrawNotice(core))
	// 获取通知的发送渠道配置
	r.GET("/:id/channels", getNoticeChannels(core))
	// 设置通知的发送渠道配置
	r.POST("/:id/channels", setNoticeChannels(core))
	// 获取常用Cron表达式列表
	r.GET("/cron-expressions", getCronExpressions(core))
}

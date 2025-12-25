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
}

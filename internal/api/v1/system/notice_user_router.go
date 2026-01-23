package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
)

// SetupNoticeUserRouter 设置用户端通知路由（所有已登录用户可访问，无需额外权限）
func SetupNoticeUserRouter(r *gin.RouterGroup, core *core.Core) {
	noticeService := services.NewNoticeService(core.DB.GetDB())
	handler := NewNoticeUserHandler(noticeService, core.DB.GetDB())

	// 通知查询
	r.GET("/my-notices", handler.GetMyNotices)
	r.GET("/my-notices/unread-count", handler.GetUnreadCount)

	// 通知详情
	r.GET("/my-notices/:id", handler.GetMyNoticeDetail)

	// 通知操作
	r.POST("/my-notices/:id/read", handler.MarkNoticeRead)
	r.POST("/my-notices/read-all", handler.MarkAllNoticesRead)
	r.POST("/my-notices/:id/ignore", handler.IgnoreNotice)
	r.DELETE("/my-notices/:id/ignore", handler.UnignoreNotice)
}

package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// getNoticeCacheService 获取通知缓存服务
func getNoticeCacheService(core *core.Core) systemServices.NoticeCacheService {
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}
	return systemServices.NewNoticeServiceWithCache(core.GetDB(), cacheProvider, core.CacheConfigService)
}

// SetupNoticeUserRouter 设置用户端通知路由（所有已登录用户可访问，无需额外权限）
func SetupNoticeUserRouter(r *gin.RouterGroup, core *core.Core) {
	noticeService := getNoticeCacheService(core)
	handler := NewNoticeUserHandler(noticeService, core.DB.GetDB()).WithCore(core)

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

package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupNoticeUserRouter 设置用户端通知路由
func SetupNoticeUserRouter(r *gin.RouterGroup, core *core.Core) {
	// 获取我的通知列表
	r.GET("/my-notices", getMyNotices(core))
	// 获取我的通知详情
	r.GET("/my-notices/:id", getMyNoticeDetail(core))
	// 标记通知已读
	r.POST("/my-notices/:id/read", markNoticeRead(core))
	// 全部标记已读
	r.POST("/my-notices/read-all", markAllNoticesRead(core))
	// 忽略通知（从列表中移除）
	r.POST("/my-notices/:id/ignore", ignoreNotice(core))
	// 取消忽略通知（恢复显示）
	r.DELETE("/my-notices/:id/ignore", unignoreNotice(core))
	// 获取未读数量
	r.GET("/my-notices/unread-count", getUnreadCount(core))
}

// NoticeWithReadStatus 带已读状态的通知
// 使用内嵌字段，JSON 序列化时 Notice 的字段会展开到顶层
type NoticeWithReadStatus struct {
	IsRead bool `json:"isRead"`
	models.Notice
}

// MyNoticesRequest 我的通知列表请求
type MyNoticesRequest struct {
	Current  int     `form:"current" json:"current" binding:"min=1"`
	PageSize int     `form:"pageSize" json:"pageSize" binding:"min=1,max=50"`
	Status   *string `form:"status" json:"status,omitempty"` // read/unread/all
}

// getMyNotices 获取我的通知列表
// @Summary 获取我的通知列表
// @Description 获取当前用户可见的通知列表
// @Tags 用户通知
// @Accept json
// @Produce json
// @Param request body MyNoticesRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /user-notices/my-notices [get]
func getMyNotices(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		var req MyNoticesRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			// 尝试从JSON绑定
			c.ShouldBindJSON(&req)
		}

		// 设置默认值
		if req.Current == 0 {
			req.Current = 1
		}
		if req.PageSize == 0 {
			req.PageSize = 10
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		notices, total, err := noticeService.GetUserNotices(c.Request.Context(), userID.(string), req.Current, req.PageSize, req.Status)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		// 获取用户已读的通知ID列表
		var readNoticeIDs []string
		core.DB.GetDB().Table("sys_notice_read").
			Where("user_id = ?", userID.(string)).
			Pluck("notice_id", &readNoticeIDs)

		// 构建已读ID的map用于快速查找
		readMap := make(map[string]bool)
		for _, id := range readNoticeIDs {
			readMap[id] = true
		}

		// 为每个通知添加已读状态
		result := make([]NoticeWithReadStatus, len(notices))
		for i, notice := range notices {
			result[i] = NoticeWithReadStatus{
				Notice: notice,
				IsRead:  readMap[notice.ID],
			}
		}

		pageResp := response.PageResponse{
			List:     result,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}

		response.Success(c, pageResp)
	}
}

// getMyNoticeDetail 获取我的通知详情（自动标记已读）
// @Summary 获取通知详情
// @Description 获取通知详情并自动标记为已读
// @Tags 用户通知
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response{data=models.Notice}
// @Router /user-notices/my-notices/:id [get]
func getMyNoticeDetail(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeID := c.Param("id")
		if noticeID == "" {
			response.Error(c, response.ErrBadRequest, "通知ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())

		// 获取通知详情
		notice, err := noticeService.GetNoticeByID(c.Request.Context(), noticeID)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		// 自动标记已读
		clientIP := c.ClientIP()
		noticeService.MarkNoticeRead(c.Request.Context(), noticeID, userID.(string), clientIP)

		// 检查是否已读
		var count int64
		core.DB.GetDB().Table("sys_notice_read").
			Where("notice_id = ? AND user_id = ?", noticeID, userID.(string)).
			Count(&count)

		result := NoticeWithReadStatus{
			Notice: *notice,
			IsRead:  count > 0,
		}

		response.Success(c, result)
	}
}

// markNoticeRead 标记通知已读
// @Summary 标记通知已读
// @Description 标记指定通知为已读状态
// @Tags 用户通知
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Router /user-notices/my-notices/:id/read [post]
func markNoticeRead(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeID := c.Param("id")
		if noticeID == "" {
			response.Error(c, response.ErrBadRequest, "通知ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		clientIP := c.ClientIP()

		if err := noticeService.MarkNoticeRead(c.Request.Context(), noticeID, userID.(string), clientIP); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "标记成功"})
	}
}

// markAllNoticesRead 全部标记已读
// @Summary 全部标记已读
// @Description 将所有可见通知标记为已读
// @Tags 用户通知
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /user-notices/my-notices/read-all [post]
func markAllNoticesRead(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.MarkAllNoticesRead(c.Request.Context(), userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "全部标记成功"})
	}
}

// getUnreadCount 获取未读数量
// @Summary 获取未读数量
// @Description 获取当前用户的未读通知数量
// @Tags 用户通知
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]int}
// @Router /user-notices/my-notices/unread-count [get]
func getUnreadCount(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		count, err := noticeService.GetUnreadCount(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"count": count})
	}
}

// ignoreNotice 忽略通知
// @Summary 忽略通知
// @Description 将通知加入忽略列表，之后不再显示
// @Tags 用户通知
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Router /user-notices/my-notices/:id/ignore [post]
func ignoreNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeID := c.Param("id")
		if noticeID == "" {
			response.Error(c, response.ErrBadRequest, "通知ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.IgnoreNotice(c.Request.Context(), noticeID, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "已忽略该通知"})
	}
}

// unignoreNotice 取消忽略通知
// @Summary 取消忽略通知
// @Description 将通知从忽略列表中移除，恢复显示
// @Tags 用户通知
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Router /user-notices/my-notices/:id/ignore [delete]
func unignoreNotice(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未授权")
			return
		}

		noticeID := c.Param("id")
		if noticeID == "" {
			response.Error(c, response.ErrBadRequest, "通知ID不能为空")
			return
		}

		noticeService := services.NewNoticeService(core.DB.GetDB())
		if err := noticeService.UnignoreNotice(c.Request.Context(), noticeID, userID.(string)); err != nil {
			// 如果通知未被忽略，也返回成功（幂等操作）
			if err.Error() == "该通知未被忽略" {
				response.Success(c, gin.H{"message": "通知已显示"})
				return
			}
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "已恢复该通知"})
	}
}

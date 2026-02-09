package system

import (

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// NoticeUserHandler 用户通知处理器
type NoticeUserHandler struct {
	noticeService *services.NoticeService
	db            *gorm.DB
}

// NewNoticeUserHandler 创建用户通知处理器实例
func NewNoticeUserHandler(noticeService *services.NoticeService, db *gorm.DB) *NoticeUserHandler {
	return &NoticeUserHandler{
		noticeService: noticeService,
		db:            db,
	}
}

// NoticeWithReadStatus 带已读状态的通知
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

// getUserID 从上下文获取用户ID
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// requireUserID 验证并返回用户ID
func requireUserID(c *gin.Context) (string, bool) {
	userID, exists := getUserID(c)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权")
		return "", false
	}
	return userID, true
}

// getReadNoticeIDs 获取用户已读的通知ID列表
func (h *NoticeUserHandler) getReadNoticeIDs(userID string) (map[string]bool, error) {
	var readNoticeIDs []string
	err := h.db.Table("sys_notice_read").
		Where("user_id = ?", userID).
		Pluck("notice_id", &readNoticeIDs).Error
	if err != nil {
		return nil, err
	}

	// 构建已读ID的map用于快速查找
	readMap := make(map[string]bool, len(readNoticeIDs))
	for _, id := range readNoticeIDs {
		readMap[id] = true
	}
	return readMap, nil
}

// GetMyNotices 获取我的通知列表
func (h *NoticeUserHandler) GetMyNotices(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req MyNoticesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.ShouldBindJSON(&req)
	}

	if req.Current == 0 {
		req.Current = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	notices, total, err := h.noticeService.GetUserNotices(c.Request.Context(), userID, req.Current, req.PageSize, req.Status)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 获取用户已读的通知ID列表
	readMap, err := h.getReadNoticeIDs(userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 为每个通知添加已读状态
	result := make([]NoticeWithReadStatus, len(notices))
	for i, notice := range notices {
		result[i] = NoticeWithReadStatus{
			Notice: notice,
			IsRead: readMap[notice.ID],
		}
	}

	response.Page(c, result, total, req.Current, req.PageSize)
}

// GetMyNoticeDetail 获取我的通知详情（自动标记已读）
func (h *NoticeUserHandler) GetMyNoticeDetail(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, http.StatusBadRequest, "通知ID不能为空")
		return
	}

	// 获取通知详情
	notice, err := h.noticeService.GetNoticeByID(c.Request.Context(), noticeID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 自动标记已读
	clientIP := c.ClientIP()
	_ = h.noticeService.MarkNoticeRead(c.Request.Context(), noticeID, userID, clientIP)

	// 检查是否已读
	var count int64
	h.db.Table("sys_notice_read").
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Count(&count)

	result := NoticeWithReadStatus{
		Notice: *notice,
		IsRead: count > 0,
	}

	response.Success(c, result)
}

// MarkNoticeRead 标记通知已读
func (h *NoticeUserHandler) MarkNoticeRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, http.StatusBadRequest, "通知ID不能为空")
		return
	}

	clientIP := c.ClientIP()
	if err := h.noticeService.MarkNoticeRead(c.Request.Context(), noticeID, userID, clientIP); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "标记成功"})
}

// MarkAllNoticesRead 全部标记已读
func (h *NoticeUserHandler) MarkAllNoticesRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := h.noticeService.MarkAllNoticesRead(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "全部标记成功"})
}

// GetUnreadCount 获取未读数量
func (h *NoticeUserHandler) GetUnreadCount(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	count, err := h.noticeService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"count": count})
}

// IgnoreNotice 忽略通知
func (h *NoticeUserHandler) IgnoreNotice(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, http.StatusBadRequest, "通知ID不能为空")
		return
	}

	if err := h.noticeService.IgnoreNotice(c.Request.Context(), noticeID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "已忽略该通知"})
}

// UnignoreNotice 取消忽略通知
func (h *NoticeUserHandler) UnignoreNotice(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, http.StatusBadRequest, "通知ID不能为空")
		return
	}

	err := h.noticeService.UnignoreNotice(c.Request.Context(), noticeID, userID)
	if err != nil {
		// 如果通知未被忽略，也返回成功（幂等操作）
		if err.Error() == "该通知未被忽略" {
			response.Success(c, gin.H{"message": "通知已显示"})
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "已恢复该通知"})
}

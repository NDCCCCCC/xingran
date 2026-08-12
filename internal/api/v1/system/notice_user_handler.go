package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// NoticeUserHandler 用户通知处理器
type NoticeUserHandler struct {
	noticeService systemServices.NoticeCacheService
	db            *gorm.DB
	core          *core.Core
}

// NewNoticeUserHandler 创建用户通知处理器实例
func NewNoticeUserHandler(noticeService systemServices.NoticeCacheService, db *gorm.DB) *NoticeUserHandler {
	return &NoticeUserHandler{
		noticeService: noticeService,
		db:            db,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用。Phase 34 Wave 7 新增。
func (h *NoticeUserHandler) WithCore(core *core.Core) *NoticeUserHandler {
	if h != nil {
		h.core = core
	}
	return h
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
		response.Error(c, apperrors.Unauthorized())
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
// @Summary 获取我的通知列表
// @Description 获取当前登录用户的通知列表
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param status query string false "阅读状态(read/unread/all)" default(all)
// @Param current query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices [get]
func (h *NoticeUserHandler) GetMyNotices(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req MyNoticesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.ShouldBindJSON(&req)
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
// @Summary 获取我的通知详情
// @Description 获取通知详情并自动标记为已读
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/:id [get]
func (h *NoticeUserHandler) GetMyNoticeDetail(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, apperrors.ParamMissing("通知ID"))
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
// @Summary 标记通知已读
// @Description 标记指定通知为已读状态
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/:id/read [post]
func (h *NoticeUserHandler) MarkNoticeRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, apperrors.ParamMissing("通知ID"))
		return
	}

	clientIP := c.ClientIP()
	if err := h.noticeService.MarkNoticeRead(c.Request.Context(), noticeID, userID, clientIP); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "我的通知", OperTypeUpdate)

	response.Success(c, gin.H{"message": "标记成功"})
}

// MarkAllNoticesRead 全部标记已读
// @Summary 全部标记已读
// @Description 将所有通知标记为已读状态
// @Tags 通知公告
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/read-all [post]
func (h *NoticeUserHandler) MarkAllNoticesRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := h.noticeService.MarkAllNoticesRead(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "我的通知", OperTypeUpdate)

	response.Success(c, gin.H{"message": "全部标记成功"})
}

// GetUnreadCount 获取未读数量
// @Summary 获取未读数量
// @Description 获取当前用户的未读通知数量
// @Tags 通知公告
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/unread-count [get]
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
// @Summary 忽略通知
// @Description 忽略指定通知
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/:id/ignore [post]
func (h *NoticeUserHandler) IgnoreNotice(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, apperrors.ParamMissing("通知ID"))
		return
	}

	if err := h.noticeService.IgnoreNotice(c.Request.Context(), noticeID, userID); err != nil {
		response.Error(c, err)
		return
	}

	recordOperLog(c, h.core, "我的通知", OperTypeUpdate)

	response.Success(c, gin.H{"message": "已忽略该通知"})
}

// UnignoreNotice 取消忽略通知
// @Summary 取消忽略通知
// @Description 取消忽略指定通知
// @Tags 通知公告
// @Accept json
// @Produce json
// @Param id path string true "通知ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/my-notices/:id/unignore [post]
func (h *NoticeUserHandler) UnignoreNotice(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	noticeID := c.Param("id")
	if noticeID == "" {
		response.Error(c, apperrors.ParamMissing("通知ID"))
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

	recordOperLog(c, h.core, "我的通知", OperTypeUpdate)

	response.Success(c, gin.H{"message": "已恢复该通知"})
}

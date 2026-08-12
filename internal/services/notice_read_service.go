package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== 已读/忽略操作 ====================

// MarkNoticeRead 标记已读
func (s *NoticeService) MarkNoticeRead(ctx context.Context, noticeID, userID, ip string) error {
	// 检查是否已读
	var count int64
	s.db.WithContext(ctx).Table("sys_notice_read").
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Count(&count)

	if count > 0 {
		return nil // 已读，跳过
	}

	read := &models.NoticeRead{
		NoticeID: noticeID,
		UserID:   userID,
		ReadAt:   time.Now(),
		ReadIP:   ip,
	}

	return s.db.WithContext(ctx).Create(read).Error
}

// MarkAllNoticesRead 标记用户所有通知为已读
func (s *NoticeService) MarkAllNoticesRead(ctx context.Context, userID string) error {
	// 获取用户可见的所有通知ID
	var noticeIDs []string
	s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("publish_status = ? AND status = 0", models.PublishStatusPublished).
		Pluck("id", &noticeIDs)

	if len(noticeIDs) == 0 {
		return nil
	}

	// 批量插入阅读记录 - 使用Create批量插入提升性能
	now := time.Now()
	reads := make([]models.NoticeRead, 0, len(noticeIDs))
	for _, noticeID := range noticeIDs {
		reads = append(reads, models.NoticeRead{
			NoticeID: noticeID,
			UserID:   userID,
			ReadAt:   now,
		})
	}
	// GORM会自动分批处理大数据量
	s.db.WithContext(ctx).Create(&reads)

	return nil
}

// IgnoreNotice 用户忽略通知（从列表中隐藏）
func (s *NoticeService) IgnoreNotice(ctx context.Context, noticeID, userID string) error {
	// 检查是否已忽略
	var count int64
	s.db.WithContext(ctx).Table("sys_notice_ignore").
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Count(&count)

	if count > 0 {
		return nil // 已忽略，跳过
	}

	ignore := &models.NoticeIgnore{
		NoticeID: noticeID,
		UserID:   userID,
	}

	return s.db.WithContext(ctx).Create(ignore).Error
}

// UnignoreNotice 用户取消忽略通知（恢复显示）
func (s *NoticeService) UnignoreNotice(ctx context.Context, noticeID, userID string) error {
	result := s.db.WithContext(ctx).Table("sys_notice_ignore").
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Delete(&models.NoticeIgnore{})

	if result.Error != nil {
		return fmt.Errorf("取消忽略失败: %w", result.Error)
	}

	// 没有找到记录，说明之前没有忽略过
	if result.RowsAffected == 0 {
		return fmt.Errorf("该通知未被忽略")
	}

	return nil
}

// GetIgnoredNotices 获取用户已忽略的通知列表
func (s *NoticeService) GetIgnoredNotices(ctx context.Context, userID string, page, pageSize int) ([]models.Notice, int64, error) {
	var notices []models.Notice
	var total int64

	// 获取用户忽略的通知ID列表
	var ignoredNoticeIDs []string
	s.db.WithContext(ctx).Table("sys_notice_ignore").
		Where("user_id = ?", userID).
		Pluck("notice_id", &ignoredNoticeIDs)

	if len(ignoredNoticeIDs) == 0 {
		return []models.Notice{}, 0, nil
	}

	// 统计总数
	if err := s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("id IN ?", ignoredNoticeIDs).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计已忽略通知数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.db.WithContext(ctx).
		Where("id IN ?", ignoredNoticeIDs).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notices).Error; err != nil {
		return nil, 0, fmt.Errorf("查询已忽略通知列表失败: %w", err)
	}

	return notices, total, nil
}

// GetUnreadCount 获取用户未读数量
func (s *NoticeService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	// 使用公共查询方法获取可见通知
	queryCtx, err := s.buildUserVisibleQuery(ctx, userID)
	if err != nil {
		return 0, err
	}

	// 获取可见通知ID列表
	var visibleNoticeIDs []string
	if err := queryCtx.Query.Pluck("id", &visibleNoticeIDs).Error; err != nil {
		return 0, err
	}

	if len(visibleNoticeIDs) == 0 {
		return 0, nil
	}

	// 获取这些可见通知中已读的ID列表
	var readNoticeIDs []string
	s.db.WithContext(ctx).Table("sys_notice_read").
		Where("user_id = ? AND notice_id IN ?", userID, visibleNoticeIDs).
		Pluck("notice_id", &readNoticeIDs)

	// 未读数量 = 可见通知总数 - 已读数量
	unreadCount := len(visibleNoticeIDs) - len(readNoticeIDs)

	// 确保不会返回负数
	if unreadCount < 0 {
		unreadCount = 0
	}

	return unreadCount, nil
}

// GetUserNotices 获取用户可见的通知列表
func (s *NoticeService) GetUserNotices(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error) {
	// 使用公共查询方法获取基础查询
	queryCtx, err := s.buildUserVisibleQuery(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	query := queryCtx.Query

	// 筛选已读/未读状态
	if status != nil {
		switch *status {
		case "unread":
			query = query.Where("id NOT IN (SELECT notice_id FROM sys_notice_read WHERE user_id = ?)", userID)
		case "read":
			query = query.Where("id IN (SELECT notice_id FROM sys_notice_read WHERE user_id = ?)", userID)
		}
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计通知数量失败: %w", err)
	}

	// 分页查询（按优先级和创建时间排序）
	var notices []models.Notice
	offset := (page - 1) * pageSize
	if err := query.Order("priority DESC, created_at DESC").Offset(offset).Limit(pageSize).Find(&notices).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}

	return notices, total, nil
}

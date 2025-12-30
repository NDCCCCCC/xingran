package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// NoticeService 通知服务
type NoticeService struct {
	db *gorm.DB
}

// NewNoticeService 创建通知服务
func NewNoticeService(db *gorm.DB) *NoticeService {
	return &NoticeService{db: db}
}

// CreateNoticeRequest 创建通知请求
type CreateNoticeRequest struct {
	NoticeTitle   string                `json:"noticeTitle" binding:"required,min=2,max=100"`
	NoticeType    string                `json:"noticeType" binding:"required,oneof=1 2"`
	NoticeContent string                `json:"noticeContent" binding:"required"`
	Priority      models.NoticePriority `json:"priority" binding:"min=0,max=2"`
	PublishTime   *time.Time            `json:"publishTime"`
	TargetType    models.TargetType     `json:"targetType" binding:"min=0,max=3"`
	TargetDepts   []string              `json:"targetDepts,omitempty"`
	TargetRoles   []string              `json:"targetRoles,omitempty"`
	TargetUsers   []string              `json:"targetUsers,omitempty"`
	IsMarkdown    bool                  `json:"isMarkdown"`
}

// UpdateNoticeRequest 更新通知请求
type UpdateNoticeRequest struct {
	NoticeTitle     *string                `json:"noticeTitle" binding:"omitempty,min=2,max=100"`
	NoticeType      *string                `json:"noticeType" binding:"omitempty,oneof=1 2"`
	NoticeContent   *string                `json:"noticeContent" binding:"omitempty"`
	Priority        *models.NoticePriority `json:"priority" binding:"omitempty,min=0,max=2"`
	Status          *int                   `json:"status" binding:"omitempty,min=0,max=1"`
	PublishTime     *time.Time             `json:"publishTime,omitempty"`
	ClearPublishTime bool                  `json:"clearPublishTime,omitempty"` // 是否清除定时发布时间
}

// CreateNoticeWithTargets 创建通知（带接收范围）
func (s *NoticeService) CreateNoticeWithTargets(ctx context.Context, req *CreateNoticeRequest, creatorID, creatorName string) (*models.Notice, error) {
	notice := &models.Notice{
		NoticeTitle:   req.NoticeTitle,
		NoticeType:    req.NoticeType,
		NoticeContent: req.NoticeContent,
		Priority:      req.Priority,
		TargetType:    req.TargetType,
		CreatedByName: creatorName,
		IsMarkdown:    req.IsMarkdown,
		Status:        0, // 正常
	}

	// 默认为草稿状态，需要手动发布或设置定时发布
	// 如果设置了定时发布时间，设置为定时发布中
	if req.PublishTime != nil && req.PublishTime.After(time.Now()) {
		notice.PublishStatus = models.PublishStatusScheduled
		notice.PublishTime = req.PublishTime
	} else {
		// 默认创建为草稿，不自动发布
		notice.PublishStatus = models.PublishStatusDraft
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 设置创建者
	if err := tx.Create(&notice).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建通知失败: %w", err)
	}

	// 创建接收范围（非全部用户时）
	if req.TargetType != models.TargetAll {
		targets := s.buildTargets(notice.ID, req)
		if len(targets) > 0 {
			if err := tx.Create(&targets).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("创建接收范围失败: %w", err)
			}
		}
	}

	tx.Commit()
	return notice, nil
}

// buildTargets 构建接收范围
func (s *NoticeService) buildTargets(noticeID string, req *CreateNoticeRequest) []models.NoticeTarget {
	var targets []models.NoticeTarget

	switch req.TargetType {
	case models.TargetDept:
		for _, deptID := range req.TargetDepts {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "dept",
				TargetID:   deptID,
			})
			// 包含子部门
			childDepts := s.getChildDeptIDs(deptID)
			for _, childID := range childDepts {
				targets = append(targets, models.NoticeTarget{
					NoticeID:   noticeID,
					TargetType: "dept",
					TargetID:   childID,
				})
			}
		}
	case models.TargetRole:
		for _, roleID := range req.TargetRoles {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "role",
				TargetID:   roleID,
			})
		}
	case models.TargetUser:
		for _, userID := range req.TargetUsers {
			targets = append(targets, models.NoticeTarget{
				NoticeID:   noticeID,
				TargetType: "user",
				TargetID:   userID,
			})
		}
	}

	return targets
}

// getChildDeptIDs 递归获取子部门ID
func (s *NoticeService) getChildDeptIDs(deptID string) []string {
	var deptIDs []string
	s.db.Raw(`
		WITH RECURSIVE dept_tree AS (
			SELECT id FROM sys_department WHERE id = ?
			UNION ALL
			SELECT d.id FROM sys_department d
			INNER JOIN dept_tree dt ON d.parent_id = dt.id
		)
		SELECT id FROM dept_tree WHERE id != ?
	`, deptID, deptID).Scan(&deptIDs)
	return deptIDs
}

// GetNoticeList 获取通知列表（管理端）
func (s *NoticeService) GetNoticeList(ctx context.Context, page, pageSize int, title, noticeType *string) ([]models.Notice, int64, error) {
	var notices []models.Notice
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Notice{})

	// 筛选条件
	if title != nil && *title != "" {
		query = query.Where("notice_title LIKE ?", "%"+*title+"%")
	}
	if noticeType != nil && *noticeType != "" {
		query = query.Where("notice_type = ?", *noticeType)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计通知数量失败: %w", err)
	}

	// 分页查询（按优先级和创建时间排序）
	offset := (page - 1) * pageSize
	if err := query.Order("priority DESC, created_at DESC").Offset(offset).Limit(pageSize).Find(&notices).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}

	return notices, total, nil
}

// GetNoticeByID 根据ID获取通知
func (s *NoticeService) GetNoticeByID(ctx context.Context, noticeID string) (*models.Notice, error) {
	var notice models.Notice
	if err := s.db.WithContext(ctx).Preload("Targets").Where("id = ?", noticeID).First(&notice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知不存在")
		}
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}
	return &notice, nil
}

// UpdateNotice 更新通知
func (s *NoticeService) UpdateNotice(ctx context.Context, noticeID string, req *UpdateNoticeRequest) error {
	updates := make(map[string]interface{})

	if req.NoticeTitle != nil {
		updates["notice_title"] = *req.NoticeTitle
	}
	if req.NoticeType != nil {
		updates["notice_type"] = *req.NoticeType
	}
	if req.NoticeContent != nil {
		updates["notice_content"] = *req.NoticeContent
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 处理定时发布时间
	if req.ClearPublishTime {
		// 清除定时发布时间，状态改回草稿
		updates["publish_time"] = nil
		updates["publish_status"] = models.PublishStatusDraft
	} else if req.PublishTime != nil {
		// 设置新的定时发布时间
		updates["publish_time"] = *req.PublishTime
		// 如果设置的是未来时间，更新状态为定时发布中
		if req.PublishTime.After(time.Now()) {
			updates["publish_status"] = models.PublishStatusScheduled
		} else {
			// 如果设置的是过去时间或当前时间，保持草稿状态
			updates["publish_status"] = models.PublishStatusDraft
		}
	}

	if len(updates) == 0 {
		return fmt.Errorf("没有需要更新的字段")
	}

	result := s.db.WithContext(ctx).Model(&models.Notice{}).Where("id = ?", noticeID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新通知失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在")
	}

	return nil
}

// DeleteNotice 删除通知
func (s *NoticeService) DeleteNotice(ctx context.Context, noticeID string) error {
	result := s.db.WithContext(ctx).Delete(&models.Notice{}, "id = ?", noticeID)
	if result.Error != nil {
		return fmt.Errorf("删除通知失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在")
	}
	return nil
}

// BatchDeleteNotices 批量删除通知
func (s *NoticeService) BatchDeleteNotices(ctx context.Context, noticeIDs []string) error {
	if len(noticeIDs) == 0 {
		return fmt.Errorf("通知ID列表不能为空")
	}

	result := s.db.WithContext(ctx).Delete(&models.Notice{}, "id IN ?", noticeIDs)
	if result.Error != nil {
		return fmt.Errorf("批量删除通知失败: %w", result.Error)
	}
	return nil
}

// GetTargetUsers 获取通知的目标用户列表
func (s *NoticeService) GetTargetUsers(ctx context.Context, notice *models.Notice) ([]string, error) {
	var userIDs []string

	// 处理 nil Targets 的情况
	if notice.Targets == nil {
		notice.Targets = []models.NoticeTarget{}
	}

	switch notice.TargetType {
	case models.TargetAll:
		if err := s.db.WithContext(ctx).Model(&models.User{}).Pluck("id", &userIDs).Error; err != nil {
			return nil, fmt.Errorf("获取用户列表失败: %w", err)
		}

	case models.TargetDept:
		var deptIDs []string
		for _, t := range notice.Targets {
			if t.TargetType == "dept" {
				deptIDs = append(deptIDs, t.TargetID)
			}
		}
		if len(deptIDs) > 0 {
			if err := s.db.WithContext(ctx).Model(&models.User{}).Where("dept_id IN ?", deptIDs).Pluck("id", &userIDs).Error; err != nil {
				return nil, fmt.Errorf("获取部门用户失败: %w", err)
			}
		}

	case models.TargetRole:
		roleIDs := getTargetIDsByType(notice.Targets, "role")
		if len(roleIDs) > 0 {
			if err := s.db.WithContext(ctx).Table("sys_user_role").
				Where("role_id IN ?", roleIDs).
				Distinct("user_id").
				Pluck("user_id", &userIDs).Error; err != nil {
				return nil, fmt.Errorf("获取角色用户失败: %w", err)
			}
		}

	case models.TargetUser:
		userIDs = getTargetIDsByType(notice.Targets, "user")

	default:
		// 未知的 TargetType，返回空列表
		return []string{}, nil
	}

	return unique(userIDs), nil
}

// getTargetIDsByType 根据类型获取目标ID列表
func getTargetIDsByType(targets []models.NoticeTarget, targetType string) []string {
	var ids []string
	for _, t := range targets {
		if t.TargetType == targetType {
			ids = append(ids, t.TargetID)
		}
	}
	return ids
}

// unique 去重
func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// GetNoticeStatistics 获取阅读统计
func (s *NoticeService) GetNoticeStatistics(ctx context.Context, noticeID string) (*models.NoticeStatistics, error) {
	var stats models.NoticeStatistics

	// 获取通知详情
	var notice models.Notice
	if err := s.db.WithContext(ctx).Where("id = ?", noticeID).First(&notice).Error; err != nil {
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}

	// 获取目标用户列表
	targetUserIDs, err := s.GetTargetUsers(ctx, &notice)
	if err != nil {
		return nil, fmt.Errorf("获取目标用户失败: %w", err)
	}

	totalTargets := len(targetUserIDs)

	// 获取已读数量
	var readCount int64
	if totalTargets > 0 {
		s.db.WithContext(ctx).Table("sys_notice_read").
			Where("notice_id = ? AND user_id IN ?", noticeID, targetUserIDs).
			Count(&readCount)
	}

	stats.ReadCount = int(readCount)
	stats.TotalTargets = totalTargets
	stats.UnreadCount = totalTargets - int(readCount)
	if totalTargets > 0 {
		stats.ReadRate = float64(readCount) / float64(totalTargets) * 100
	} else {
		stats.ReadRate = 0
	}

	return &stats, nil
}

// UserNoticeQueryContext 用户通知查询上下文
// 封装用户信息、角色信息和基础查询
type UserNoticeQueryContext struct {
	User    models.User
	RoleIDs []string
	Query   *gorm.DB
}

// buildUserVisibleQuery 构建用户可见通知的基础查询
// 封装了获取用户信息、角色信息以及构建权限过滤查询的公共逻辑
func (s *NoticeService) buildUserVisibleQuery(ctx context.Context, userID string) (*UserNoticeQueryContext, error) {
	// 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取用户角色
	var userRoles []models.UserRole
	s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles)
	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// 构建权限过滤查询：已发布 + 正常状态 + 未被忽略 + 匹配用户部门/角色
	query := s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("publish_status = ? AND status = 0", models.PublishStatusPublished).
		Where("id NOT IN (SELECT notice_id FROM sys_notice_ignore WHERE user_id = ?)", userID).
		Where("(target_type = 0) OR (target_type = 3 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'user' AND target_id = ?)) OR "+
			"(target_type = 1 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'dept' AND target_id = ?)) OR "+
			"(target_type = 2 AND id IN (SELECT notice_id FROM sys_notice_target WHERE target_type = 'role' AND target_id IN (?)))",
			userID, user.DeptID, roleIDs)

	return &UserNoticeQueryContext{
		User:    user,
		RoleIDs: roleIDs,
		Query:   query,
	}, nil
}

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

	// 批量插入阅读记录（忽略已存在的）
	for _, noticeID := range noticeIDs {
		read := &models.NoticeRead{
			NoticeID: noticeID,
			UserID:   userID,
			ReadAt:   time.Now(),
		}
		s.db.WithContext(ctx).Create(read) // 忽略错误（已存在时会跳过）
	}

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
		if *status == "unread" {
			query = query.Where("id NOT IN (SELECT notice_id FROM sys_notice_read WHERE user_id = ?)", userID)
		} else if *status == "read" {
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

// PublishNotice 发布通知（用于定时发布或手动发布）
func (s *NoticeService) PublishNotice(ctx context.Context, noticeID string) error {
	// 首先获取当前通知的状态和发布时间
	var notice models.Notice
	if err := s.db.WithContext(ctx).Select("id, publish_status, notice_title, publish_time").Where("id = ?", noticeID).First(&notice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("通知不存在")
		}
		return fmt.Errorf("查询通知失败: %w", err)
	}

	// 检查当前状态，只允许从草稿或定时发布状态发布
	if notice.PublishStatus == models.PublishStatusPublished {
		return fmt.Errorf("通知已经发布，无需重复发布")
	}
	if notice.PublishStatus == models.PublishStatusWithdrawn {
		return fmt.Errorf("通知已撤回，无法重新发布")
	}

	// 准备更新数据
	updates := map[string]interface{}{
		"publish_status": models.PublishStatusPublished,
	}

	// 如果是手动发布（publish_time 为空），则设置为当前时间
	if notice.PublishTime == nil {
		now := time.Now()
		updates["publish_time"] = &now
	}

	// 更新状态和发布时间
	result := s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("id = ?", noticeID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("发布通知失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("更新通知状态失败")
	}

	return nil
}

// WithdrawNotice 撤回/取消发布通知（将已发布通知退回草稿状态）
func (s *NoticeService) WithdrawNotice(ctx context.Context, noticeID string) error {
	// 首先获取当前通知的状态
	var notice models.Notice
	if err := s.db.WithContext(ctx).Select("id, publish_status, notice_title").Where("id = ?", noticeID).First(&notice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("通知不存在")
		}
		return fmt.Errorf("查询通知失败: %w", err)
	}

	// 只有已发布的通知可以撤回
	if notice.PublishStatus != models.PublishStatusPublished {
		return fmt.Errorf("只有已发布的通知可以撤回")
	}

	// 更新状态为草稿
	result := s.db.WithContext(ctx).Model(&models.Notice{}).
		Where("id = ?", noticeID).
		Update("publish_status", models.PublishStatusDraft)

	if result.Error != nil {
		return fmt.Errorf("撤回通知失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("更新通知状态失败")
	}

	return nil
}

// CalculateCronExpression 计算指定时间的Cron表达式
// 用于一次性定时任务：在指定时间执行一次
func CalculateCronExpression(t time.Time) string {
	// Cron表达式格式：秒 分 时 日 月 周
	// 例如：2024-01-15 14:30:00 -> "0 30 14 15 1 ?"
	return fmt.Sprintf("0 %d %d %d %d ?",
		t.Minute(),
		t.Hour(),
		t.Day(),
		t.Month(),
	)
}

// GenerateCronExpression 生成周期性任务的 Cron 表达式
// 根据 RecurrenceConfig 生成对应的 Cron 表达式
func GenerateCronExpression(recurrenceType string, executeTime string, weekDay, monthDay *int, customExpr *string) (string, error) {
	// 如果提供了自定义表达式，直接返回
	if recurrenceType == "custom" && customExpr != nil && *customExpr != "" {
		return *customExpr, nil
	}

	// 解析执行时间
	var hour, minute int
	if executeTime != "" {
		_, err := fmt.Sscanf(executeTime, "%d:%d", &hour, &minute)
		if err != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", fmt.Errorf("执行时间格式错误，应为 HH:mm 格式")
		}
	} else {
		// 默认上午9点
		hour = 9
		minute = 0
	}

	// 根据周期类型生成 Cron 表达式
	switch recurrenceType {
	case "daily":
		// 每天执行：0 0 9 * * ? （每天上午9点）
		return fmt.Sprintf("0 %d %d * * ?", minute, hour), nil

	case "weekly":
		// 每周执行：0 0 9 ? * 2 （周几上午9点）
		if weekDay == nil || *weekDay < 1 || *weekDay > 7 {
			return "", fmt.Errorf("周几参数错误，应为 1-7（1=周日，2=周一，...，7=周六）")
		}
		return fmt.Sprintf("0 %d %d ? * %d", minute, hour, *weekDay), nil

	case "monthly":
		// 每月执行：0 0 9 1 * ? （每月几号上午9点）
		if monthDay == nil || *monthDay < 1 || *monthDay > 31 {
			return "", fmt.Errorf("月份日期参数错误，应为 1-31")
		}
		return fmt.Sprintf("0 %d %d %d * ?", minute, hour, *monthDay), nil

	case "custom":
		// 自定义表达式但未提供
		return "", fmt.Errorf("自定义周期需要提供 Cron 表达式")

	default:
		return "", fmt.Errorf("不支持的周期类型: %s", recurrenceType)
	}
}

// GetNoticeJobName 获取通知定时任务的名称
func GetNoticeJobName(noticeID string) string {
	return fmt.Sprintf("notice_publish_%s", noticeID)
}

// CommonCronExpression 常用Cron表达式
type CommonCronExpression struct {
	Name        string `json:"name"`        // 名称
	Expression  string `json:"expression"`  // Cron表达式
	Description string `json:"description"` // 描述
}

// GetCommonCronExpressions 获取常用Cron表达式列表
func GetCommonCronExpressions() []CommonCronExpression {
	return []CommonCronExpression{
		{
			Name:        "每天凌晨0点",
			Expression:  "0 0 0 * * ?",
			Description: "每天00:00:00执行",
		},
		{
			Name:        "每天早上9点",
			Expression:  "0 0 9 * * ?",
			Description: "每天09:00:00执行",
		},
		{
			Name:        "每天中午12点",
			Expression:  "0 0 12 * * ?",
			Description: "每天12:00:00执行",
		},
		{
			Name:        "每天下午6点",
			Expression:  "0 0 18 * * ?",
			Description: "每天18:00:00执行",
		},
		{
			Name:        "每小时执行",
			Expression:  "0 0 * * * ?",
			Description: "每小时的0分0秒执行",
		},
		{
			Name:        "每30分钟执行",
			Expression:  "0 0,30 * * * ?",
			Description: "每小时的0分和30分执行",
		},
		{
			Name:        "每周一早上9点",
			Expression:  "0 0 9 ? * MON",
			Description: "每周一09:00:00执行",
		},
		{
			Name:        "每周一早上9点（备用）",
			Expression:  "0 0 9 ? * 2",
			Description: "每周一09:00:00执行（2表示周一）",
		},
		{
			Name:        "每月1号凌晨0点",
			Expression:  "0 0 0 1 * ?",
			Description: "每月1号00:00:00执行",
		},
		{
			Name:        "每月1号上午10点",
			Expression:  "0 0 10 1 * ?",
			Description: "每月1号10:00:00执行",
		},
		{
			Name:        "工作日早上9点",
			Expression:  "0 0 9 ? * MON-FRI",
			Description: "周一到周五09:00:00执行",
		},
		{
			Name:        "工作日早上9点（备用）",
			Expression:  "0 0 9 ? * 2-6",
			Description: "周一到周五09:00:00执行（2-6表示周一到周五）",
		},
	}
}

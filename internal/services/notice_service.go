package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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

// ==================== 请求/响应类型 ====================

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
	NoticeTitle      *string                `json:"noticeTitle" binding:"omitempty,min=2,max=100"`
	NoticeType       *string                `json:"noticeType" binding:"omitempty,oneof=1 2"`
	NoticeContent    *string                `json:"noticeContent" binding:"omitempty"`
	Priority         *models.NoticePriority `json:"priority" binding:"omitempty,min=0,max=2"`
	Status           *int                   `json:"status" binding:"omitempty,min=0,max=1"`
	PublishTime      *time.Time             `json:"publishTime,omitempty"`
	ClearPublishTime bool                   `json:"clearPublishTime,omitempty"` // 是否清除定时发布时间
	// PublishStatus 显式发布状态覆盖（Phase 34 WR-003 修复引入）
	// 0=草稿 1=已发布 2=定时发布中 3=已撤回
	PublishStatus *models.PublishStatus `json:"publishStatus,omitempty" binding:"omitempty,min=0,max=3"`
	// EndDate 周期性通知结束时间（Phase 34 WR-003 修复引入）
	EndDate *time.Time `json:"endDate,omitempty"`
}

// ==================== 基本 CRUD 操作 ====================

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
		Status:        int(models.NoticeStatusNormal), // 正常（启停字段；区别于 PublishStatus 发布态）
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

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 设置创建者
		if err := tx.Create(&notice).Error; err != nil {
			return fmt.Errorf("创建通知失败: %w", err)
		}

		// 创建接收范围（非全部用户时）
		if req.TargetType != models.TargetAll {
			targets := s.buildTargets(notice.ID, req)
			if len(targets) > 0 {
				if err := tx.Create(&targets).Error; err != nil {
					return fmt.Errorf("创建接收范围失败: %w", err)
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return notice, nil
}

// NoticeStatusStatistics 通知按发布状态的聚合统计。
// publishStatus 语义(models.PublishStatus): 0=草稿 1=已发布 2=定时发布中 3=已撤回。
type NoticeStatusStatistics struct {
	Total     int64 `json:"total"`
	Published int64 `json:"published"` // = 1
	Draft     int64 `json:"draft"`     // = 0
	Scheduled int64 `json:"scheduled"` // = 2
}

// GetNoticeStatusStatistics 统计通知总数及各发布状态计数。
// 用条件聚合(SUM CASE)避免「加载全量行进内存再 filter」的反模式,同时修正前端
// 旧实现把 scheduled 误标为 3(实为已撤回)、draft 吞掉定时发布(2)的状态桶错误。
func (s *NoticeService) GetNoticeStatusStatistics(ctx context.Context) (*NoticeStatusStatistics, error) {
	var result NoticeStatusStatistics
	err := s.db.WithContext(ctx).
		Model(&models.Notice{}).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("SUM(CASE WHEN publish_status = %d THEN 1 ELSE 0 END) AS published", int(models.PublishStatusPublished)),
			fmt.Sprintf("SUM(CASE WHEN publish_status = %d THEN 1 ELSE 0 END) AS draft", int(models.PublishStatusDraft)),
			fmt.Sprintf("SUM(CASE WHEN publish_status = %d THEN 1 ELSE 0 END) AS scheduled", int(models.PublishStatusScheduled)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计通知状态失败: %w", err)
	}
	return &result, nil
}

// noticeAllowedSortFields 通知列表可排序字段白名单（对应 sys_notice 表列名）。
var noticeAllowedSortFields = map[string]string{
	"noticeTitle": "notice_title",
	"noticeType":  "notice_type",
	"priority":    "priority",
	"createdAt":   "created_at",
	"publishTime": "publish_time",
}

// GetNoticeList 获取通知列表（管理端）
// orderByColumn/isAsc 为服务端排序参数（可选，透传给 base.ApplySort 白名单）。
func (s *NoticeService) GetNoticeList(ctx context.Context, page, pageSize int, title, noticeType *string, orderByColumn string, isAsc *bool) ([]models.Notice, int64, error) {
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

	// 分页查询：用户排序（白名单）优先，无 OrderByColumn 时保留 priority DESC, created_at DESC 默认
	offset := (page - 1) * pageSize
	sortReq := base.BaseListRequest{
		Current:       page,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	query = base.ApplySort(query, sortReq, noticeAllowedSortFields)
	if orderByColumn == "" {
		query = query.Order("priority DESC, created_at DESC")
	}
	if err := query.Preload("Channels").Offset(offset).Limit(pageSize).Find(&notices).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}

	return notices, total, nil
}

// GetNoticeByID 根据ID获取通知
func (s *NoticeService) GetNoticeByID(ctx context.Context, noticeID string) (*models.Notice, error) {
	var notice models.Notice
	if err := s.db.WithContext(ctx).Preload("Targets").Preload("Channels").Where("id = ?", noticeID).First(&notice).Error; err != nil {
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

	// 显式发布状态覆盖（Phase 34 WR-003 修复引入）
	// 当调用方明确传入了 PublishStatus 时直接写入（优先级高于上述推算结果）
	if req.PublishStatus != nil {
		updates["publish_status"] = *req.PublishStatus
	}
	// 周期性通知结束时间（Phase 34 WR-003 修复引入）
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
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

// ==================== 发布/撤回操作 ====================

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

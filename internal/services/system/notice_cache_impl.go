package system

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// NoticeCacheService 通知公告缓存服务接口
type NoticeCacheService interface {
	// 基础服务方法
	Create(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error)
	Update(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	GetNoticeByID(ctx context.Context, id string) (*models.Notice, error)
	List(ctx context.Context, params requests.NoticeListParams) (*PageResult, error)
	Publish(ctx context.Context, id string) error
	Withdraw(ctx context.Context, id string) error
	GetStatistics(ctx context.Context, id string) (*models.NoticeStatistics, error)
	GetStatusStatistics(ctx context.Context) (*services.NoticeStatusStatistics, error)

	// 用户端方法（需要缓存）
	GetUserNotices(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error)
	GetUnreadCount(ctx context.Context, userID string) (int, error)
	MarkNoticeRead(ctx context.Context, noticeID, userID, ip string) error
	MarkAllNoticesRead(ctx context.Context, userID string) error
	IgnoreNotice(ctx context.Context, noticeID, userID string) error
	UnignoreNotice(ctx context.Context, noticeID, userID string) error

	// 缓存失效方法
	InvalidateNoticeCache(ctx context.Context, noticeID string) error
	InvalidateUserNoticeCache(ctx context.Context, userID string) error
	InvalidateAllNoticeCache(ctx context.Context) error
}

// noticeCacheService 通知公告缓存服务实现
type noticeCacheService struct {
	db     *gorm.DB
	base   *services.NoticeService
	cache  CacheProvider
	config *services.CacheConfigService
}

// NewNoticeServiceWithCache 创建带缓存的通知公告服务
func NewNoticeServiceWithCache(
	db *gorm.DB,
	cache CacheProvider,
	config *services.CacheConfigService,
) NoticeCacheService {
	return &noticeCacheService{
		db:     db,
		base:   services.NewNoticeService(db),
		cache:  cache,
		config: config,
	}
}

// getExpiration 获取缓存过期时间
func (s *noticeCacheService) getExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if s.config != nil {
		return s.config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// GetStatusStatistics 统计通知各发布状态计数(供统计卡片),委托给基础服务。
// 不缓存:状态计数随发布/撤回频繁变化,且各写操作后前端会主动刷新。
func (s *noticeCacheService) GetStatusStatistics(ctx context.Context) (*services.NoticeStatusStatistics, error) {
	return s.base.GetNoticeStatusStatistics(ctx)
}

// ==================== 基础服务方法（带缓存失效） ====================

// Create 创建通知（带缓存失效）
func (s *noticeCacheService) Create(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error) {
	notice, err := s.base.CreateNoticeWithTargets(ctx, &services.CreateNoticeRequest{
		NoticeTitle:   req.NoticeTitle,
		NoticeType:    req.NoticeType,
		NoticeContent: req.NoticeContent,
		Priority:      req.Priority,
		PublishTime:   req.PublishTime,
		TargetType:    req.TargetType,
		TargetDepts:   req.TargetDepts,
		TargetRoles:   req.TargetRoles,
		TargetUsers:   req.TargetUsers,
		IsMarkdown:    req.IsMarkdown,
	}, creatorID, creatorName)
	if err != nil {
		return nil, err
	}
	// 清除列表缓存
	_ = s.InvalidateAllNoticeCache(ctx)
	return notice, nil
}

// Update 更新通知（带缓存失效）
func (s *noticeCacheService) Update(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error {
	// PublishStatus 显式转发（Phase 34 WR-003 修复引入）
	// EndDate 周期性通知结束时间（Phase 34 WR-003 修复引入）
	publishStatus := (*models.PublishStatus)(nil)
	if req.PublishStatus != nil {
		ps := models.PublishStatus(*req.PublishStatus)
		publishStatus = &ps
	}
	if err := s.base.UpdateNotice(ctx, id, &services.UpdateNoticeRequest{
		NoticeTitle:      req.NoticeTitle,
		NoticeType:       req.NoticeType,
		NoticeContent:    req.NoticeContent,
		Priority:         req.Priority,
		Status:           req.Status,
		PublishTime:      req.PublishTime,
		ClearPublishTime: req.ClearPublishTime,
		PublishStatus:    publishStatus,
		EndDate:          req.EndDate,
	}); err != nil {
		return err
	}
	// 清除缓存
	_ = s.InvalidateNoticeCache(ctx, id)
	_ = s.InvalidateAllNoticeCache(ctx)
	return nil
}

// Delete 删除通知（带缓存失效）
func (s *noticeCacheService) Delete(ctx context.Context, id string) error {
	if err := s.base.DeleteNotice(ctx, id); err != nil {
		return err
	}
	// 清除缓存
	_ = s.InvalidateNoticeCache(ctx, id)
	_ = s.InvalidateAllNoticeCache(ctx)
	return nil
}

// BatchDelete 批量删除通知（带缓存失效）
func (s *noticeCacheService) BatchDelete(ctx context.Context, ids []string) error {
	if err := s.base.BatchDeleteNotices(ctx, ids); err != nil {
		return err
	}
	// 清除所有缓存
	for _, id := range ids {
		_ = s.InvalidateNoticeCache(ctx, id)
	}
	_ = s.InvalidateAllNoticeCache(ctx)
	return nil
}

// GetNoticeByID 获取通知详情（不缓存，查询频率低）
func (s *noticeCacheService) GetNoticeByID(ctx context.Context, id string) (*models.Notice, error) {
	return s.base.GetNoticeByID(ctx, id)
}

// List 查询通知列表（不缓存，参数多变）
func (s *noticeCacheService) List(ctx context.Context, params requests.NoticeListParams) (*PageResult, error) {
	list, total, err := s.base.GetNoticeList(ctx, params.Current, params.PageSize, params.NoticeTitle, params.NoticeType, params.OrderByColumn, params.IsAsc)
	if err != nil {
		return nil, err
	}
	return &PageResult{
		List:     list,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Publish 发布通知（带缓存失效）
func (s *noticeCacheService) Publish(ctx context.Context, id string) error {
	if err := s.base.PublishNotice(ctx, id); err != nil {
		return err
	}
	// 清除缓存
	_ = s.InvalidateNoticeCache(ctx, id)
	_ = s.InvalidateAllNoticeCache(ctx)
	return nil
}

// Withdraw 撤回通知（带缓存失效）
func (s *noticeCacheService) Withdraw(ctx context.Context, id string) error {
	if err := s.base.WithdrawNotice(ctx, id); err != nil {
		return err
	}
	// 清除缓存
	_ = s.InvalidateNoticeCache(ctx, id)
	_ = s.InvalidateAllNoticeCache(ctx)
	return nil
}

// GetStatistics 获取通知统计（不缓存，低频使用）
func (s *noticeCacheService) GetStatistics(ctx context.Context, id string) (*models.NoticeStatistics, error) {
	return s.base.GetNoticeStatistics(ctx, id)
}

// ==================== 用户端方法（带缓存） ====================

// GetUserNotices 获取我的通知列表（带缓存）
func (s *noticeCacheService) GetUserNotices(ctx context.Context, userID string, page, pageSize int, status *string) ([]models.Notice, int64, error) {
	cacheKey := fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d", userID, page, pageSize)
	if status != nil {
		cacheKey = fmt.Sprintf("notice:my_notices:%s:page:%d:size:%d:status:%s", userID, page, pageSize, *status)
	}
	var result struct {
		List  []models.Notice
		Total int64
	}

	expiration := s.getExpiration("cache.notice.my_notices", 1*time.Minute)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		list, total, err := s.base.GetUserNotices(ctx, userID, page, pageSize, status)
		if err != nil {
			return nil, err
		}
		return struct {
			List  []models.Notice
			Total int64
		}{
			List:  list,
			Total: total,
		}, nil
	})

	if err != nil {
		return nil, 0, err
	}
	return result.List, result.Total, nil
}

// GetUnreadCount 获取未读通知数量（带缓存）
func (s *noticeCacheService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	cacheKey := fmt.Sprintf("notice:unread_count:%s", userID)
	var result int

	expiration := s.getExpiration("cache.notice.unread_count", 30*time.Second)

	err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
		return s.base.GetUnreadCount(ctx, userID)
	})

	if err != nil {
		return 0, err
	}
	return result, nil
}

// MarkNoticeRead 标记通知为已读（带缓存失效）
func (s *noticeCacheService) MarkNoticeRead(ctx context.Context, noticeID, userID, ip string) error {
	if err := s.base.MarkNoticeRead(ctx, noticeID, userID, ip); err != nil {
		return err
	}
	// 清除该用户的通知缓存
	_ = s.InvalidateUserNoticeCache(ctx, userID)
	return nil
}

// MarkAllNoticesRead 标记所有通知为已读（带缓存失效）
func (s *noticeCacheService) MarkAllNoticesRead(ctx context.Context, userID string) error {
	if err := s.base.MarkAllNoticesRead(ctx, userID); err != nil {
		return err
	}
	// 清除该用户的通知缓存
	_ = s.InvalidateUserNoticeCache(ctx, userID)
	return nil
}

// IgnoreNotice 忽略通知（带缓存失效）
func (s *noticeCacheService) IgnoreNotice(ctx context.Context, noticeID, userID string) error {
	if err := s.base.IgnoreNotice(ctx, noticeID, userID); err != nil {
		return err
	}
	// 清除该用户的通知缓存
	_ = s.InvalidateUserNoticeCache(ctx, userID)
	return nil
}

// UnignoreNotice 取消忽略通知（带缓存失效）
func (s *noticeCacheService) UnignoreNotice(ctx context.Context, noticeID, userID string) error {
	if err := s.base.UnignoreNotice(ctx, noticeID, userID); err != nil {
		return err
	}
	// 清除该用户的通知缓存
	_ = s.InvalidateUserNoticeCache(ctx, userID)
	return nil
}

// ==================== 缓存失效方法 ====================

// InvalidateNoticeCache 失效指定通知的缓存
func (s *noticeCacheService) InvalidateNoticeCache(ctx context.Context, noticeID string) error {
	keys := []string{fmt.Sprintf("notice:detail:%s", noticeID)}
	InvalidateCacheByKey(ctx, s.cache, keys, "NOTICE")
	return nil
}

// InvalidateUserNoticeCache 失效指定用户的通知缓存
func (s *noticeCacheService) InvalidateUserNoticeCache(ctx context.Context, userID string) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{fmt.Sprintf("notice:my_notices:%s:*", userID), fmt.Sprintf("notice:unread_count:%s", userID)}, "NOTICE")
	return nil
}

// InvalidateAllNoticeCache 失效所有通知缓存
func (s *noticeCacheService) InvalidateAllNoticeCache(ctx context.Context) error {
	InvalidateCacheByPattern(ctx, s.cache, []string{"notice:*"}, "NOTICE")
	return nil
}

package system

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"gorm.io/gorm"
)

// NoticeService 通知公告服务接口
type NoticeService interface {
	Create(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error)
	Update(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	GetByID(ctx context.Context, id string) (*models.Notice, error)
	List(ctx context.Context, params requests.NoticeListParams) (*PageResult, error)
	Publish(ctx context.Context, id string) error
	Withdraw(ctx context.Context, id string) error
	GetStatistics(ctx context.Context, id string) (*models.NoticeStatistics, error)
}

// NotificationChannelService 通知发送渠道服务接口
type NotificationChannelService interface {
	GetNotificationChannels(ctx context.Context, noticeID string) ([]models.NotificationChannel, error)
	SetNotificationChannels(ctx context.Context, noticeID string, channels []models.NotificationChannel) error
	PublishAndSendNotice(ctx context.Context, noticeID string) error
}

// noticeService 通知公告服务实现（包装现有服务）
type noticeService struct {
	noticeService *services.NoticeService
	senderService *services.NotificationSenderService
}

// NewNoticeService 创建通知公告服务实例
func NewNoticeService(db *gorm.DB) NoticeService {
	return &noticeService{
		noticeService: services.NewNoticeService(db),
		senderService: services.NewNotificationSenderService(db),
	}
}

// NewNotificationChannelService 创建通知渠道服务实例
func NewNotificationChannelService(db *gorm.DB) NotificationChannelService {
	return services.NewNotificationSenderService(db)
}

// ==================== 服务方法实现 ====================

func (s *noticeService) Create(ctx context.Context, req *requests.NoticeCreateRequest, creatorID, creatorName string) (*models.Notice, error) {
	createReq := &services.CreateNoticeRequest{
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
	}

	return s.noticeService.CreateNoticeWithTargets(ctx, createReq, creatorID, creatorName)
}

func (s *noticeService) Update(ctx context.Context, id string, req *requests.NoticeUpdateRequest) error {
	updateReq := &services.UpdateNoticeRequest{
		NoticeTitle:      req.NoticeTitle,
		NoticeType:       req.NoticeType,
		NoticeContent:    req.NoticeContent,
		Priority:         req.Priority,
		Status:           req.Status,
		PublishTime:      req.PublishTime,
		ClearPublishTime: req.ClearPublishTime,
	}

	return s.noticeService.UpdateNotice(ctx, id, updateReq)
}

func (s *noticeService) Delete(ctx context.Context, id string) error {
	return s.noticeService.DeleteNotice(ctx, id)
}

func (s *noticeService) BatchDelete(ctx context.Context, ids []string) error {
	return s.noticeService.BatchDeleteNotices(ctx, ids)
}

func (s *noticeService) GetByID(ctx context.Context, id string) (*models.Notice, error) {
	return s.noticeService.GetNoticeByID(ctx, id)
}

func (s *noticeService) List(ctx context.Context, params requests.NoticeListParams) (*PageResult, error) {
	notices, total, err := s.noticeService.GetNoticeList(
		ctx,
		params.Current,
		params.PageSize,
		params.NoticeTitle,
		params.NoticeType,
		params.OrderByColumn,
		params.IsAsc,
	)
	if err != nil {
		return nil, err
	}

	return &PageResult{
		List:     notices,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

func (s *noticeService) Publish(ctx context.Context, id string) error {
	return s.noticeService.PublishNotice(ctx, id)
}

func (s *noticeService) Withdraw(ctx context.Context, id string) error {
	return s.noticeService.WithdrawNotice(ctx, id)
}

func (s *noticeService) GetStatistics(ctx context.Context, id string) (*models.NoticeStatistics, error) {
	return s.noticeService.GetNoticeStatistics(ctx, id)
}

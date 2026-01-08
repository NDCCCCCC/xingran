package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// NotificationSenderService 通知发送服务（多渠道）
type NotificationSenderService struct {
	db              *gorm.DB
	emailSender     *EmailSenderService
	apiSender       *APISenderService
}

// NewNotificationSenderService 创建通知发送服务
func NewNotificationSenderService(db *gorm.DB) *NotificationSenderService {
	return &NotificationSenderService{
		db:          db,
		emailSender: NewEmailSenderService(db),
		apiSender:   NewAPISenderService(db),
	}
}

// SendNotification 发送通知（多渠道）
func (s *NotificationSenderService) SendNotification(ctx context.Context, noticeID string) error {
	// 获取通知详情
	noticeService := NewNoticeService(s.db)
	notice, err := noticeService.GetNoticeByID(ctx, noticeID)
	if err != nil {
		return fmt.Errorf("获取通知失败: %w", err)
	}

	// 检查发布状态
	if notice.PublishStatus != models.PublishStatusPublished {
		return fmt.Errorf("通知未发布")
	}

	// 获取通知的发送渠道配置
	var channels []models.NotificationChannel
	if err := s.db.WithContext(ctx).Where("notice_id = ?", noticeID).Find(&channels).Error; err != nil {
		return fmt.Errorf("获取发送渠道失败: %w", err)
	}

	// 获取目标用户（无论是否有渠道配置，都需要获取目标用户）
	targetUsers, err := noticeService.GetTargetUsers(ctx, notice)
	if err != nil {
		return fmt.Errorf("获取目标用户失败: %w", err)
	}

	// 如果没有配置渠道，默认使用站内信
	if len(channels) == 0 {
		return s.sendWebNotification(ctx, notice, targetUsers)
	}

	// 获取用户邮箱和手机号
	userInfo, err := s.getUserInfo(ctx, targetUsers)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 遍历每个渠道进行发送
	for _, channel := range channels {
		switch channel.ChannelType {
		case models.ChannelTypeWeb:
			// 站内信（WebSocket）
			if err := s.sendWebNotification(ctx, notice, targetUsers); err != nil {
				logger.Errorf("发送站内信失败: %v", err)
			}

		case models.ChannelTypeEmail:
			// 邮件通知
			if channel.EmailConfigID != nil {
				emails := s.getUserEmails(userInfo)
				if len(emails) > 0 {
					result := s.emailSender.SendNoticeEmail(ctx, *channel.EmailConfigID, notice, emails)
					if !result.Success {
						logger.Errorf("发送邮件失败: %s", result.Message)
					}
				}
			}

		case models.ChannelTypeSMS:
			// 短信通知
			if channel.APIConfigID != nil {
				phones := s.getUserPhones(userInfo)
				if len(phones) > 0 {
					result := s.apiSender.SendSMS(ctx, *channel.APIConfigID, phones, notice.NoticeContent)
					if !result.Success {
						logger.Errorf("发送短信失败: %s", result.Message)
					}
				}
			}

		case models.ChannelTypeAPI:
			// API通知（Webhook等）
			if channel.APIConfigID != nil {
				result := s.apiSender.SendNoticeAPI(ctx, *channel.APIConfigID, notice, targetUsers)
				if !result.Success {
					logger.Errorf("发送API通知失败: %s", result.Message)
				}
			}
		}
	}

	return nil
}

// sendWebNotification 发送站内信（通过WebSocket）
func (s *NotificationSenderService) sendWebNotification(ctx context.Context, notice *models.Notice, targetUsers []string) error {
	// 这里应该通过WebSocket Hub发送通知
	// 具体实现需要注入NoticeHub
	// 暂时返回nil，实际使用时需要集成WebSocket Hub
	logger.Infof("[站内信] 发送通知: %s 到 %d 个用户", notice.NoticeTitle, len(targetUsers))
	return nil
}

// userInfo 用户信息
type userInfo struct {
	userID string
	email  string
	phone  string
}

// getUserInfo 获取用户信息（邮箱、手机号）
func (s *NotificationSenderService) getUserInfo(ctx context.Context, userIDs []string) (map[string]*userInfo, error) {
	var users []struct {
		ID    string
		Email string
		Phone string
	}

	if err := s.db.WithContext(ctx).Table("sys_user").
		Select("id, email, phone").
		Where("id IN ?", userIDs).
		Find(&users).Error; err != nil {
		return nil, err
	}

	result := make(map[string]*userInfo)
	for _, u := range users {
		result[u.ID] = &userInfo{
			userID: u.ID,
			email:  u.Email,
			phone:  u.Phone,
		}
	}

	return result, nil
}

// getUserEmails 获取用户邮箱列表
func (s *NotificationSenderService) getUserEmails(userInfo map[string]*userInfo) []string {
	var emails []string
	for _, info := range userInfo {
		if info.email != "" {
			emails = append(emails, info.email)
		}
	}
	return emails
}

// getUserPhones 获取用户手机号列表
func (s *NotificationSenderService) getUserPhones(userInfo map[string]*userInfo) []string {
	var phones []string
	for _, info := range userInfo {
		if info.phone != "" {
			phones = append(phones, info.phone)
		}
	}
	return phones
}

// PublishAndSendNotice 发布并发送通知
func (s *NotificationSenderService) PublishAndSendNotice(ctx context.Context, noticeID string) error {
	noticeService := NewNoticeService(s.db)

	// 1. 更新发布状态
	if err := noticeService.PublishNotice(ctx, noticeID); err != nil {
		return fmt.Errorf("发布通知失败: %w", err)
	}

	// 2. 发送通知到各个渠道
	return s.SendNotification(ctx, noticeID)
}

// ScheduleNoticePublish 定时发布通知（用于定时任务）
func (s *NotificationSenderService) ScheduleNoticePublish(ctx context.Context, noticeID string) error {
	noticeService := NewNoticeService(s.db)

	// 获取通知
	notice, err := noticeService.GetNoticeByID(ctx, noticeID)
	if err != nil {
		return fmt.Errorf("获取通知失败: %w", err)
	}

	// 检查是否到达发布时间
	if notice.PublishTime != nil && notice.PublishTime.After(time.Now()) {
		return fmt.Errorf("未到发布时间")
	}

	// 检查发布状态
	if notice.PublishStatus != models.PublishStatusScheduled {
		return fmt.Errorf("通知状态不正确")
	}

	// 发布并发送
	return s.PublishAndSendNotice(ctx, noticeID)
}

// SetNotificationChannels 设置通知的发送渠道
func (s *NotificationSenderService) SetNotificationChannels(ctx context.Context, noticeID string, channels []models.NotificationChannel) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧的渠道配置
	if err := tx.Where("notice_id = ?", noticeID).Delete(&models.NotificationChannel{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除旧渠道配置失败: %w", err)
	}

	// 创建新的渠道配置
	for i := range channels {
		channels[i].NoticeID = noticeID
	}
	if len(channels) > 0 {
		if err := tx.Create(&channels).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建渠道配置失败: %w", err)
		}
	}

	tx.Commit()
	return nil
}

// GetNotificationChannels 获取通知的发送渠道
func (s *NotificationSenderService) GetNotificationChannels(ctx context.Context, noticeID string) ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := s.db.WithContext(ctx).Where("notice_id = ?", noticeID).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取发送渠道失败: %w", err)
	}
	return channels, nil
}

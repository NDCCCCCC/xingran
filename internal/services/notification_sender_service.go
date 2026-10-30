package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// NotificationSenderService 通知发送服务（多渠道）
// 支持站内信、邮件、短信、API（企业微信等）多种发送渠道
type NotificationSenderService struct {
	db          *gorm.DB
	emailSender *EmailSenderService
	apiSender   *APISenderService
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
	notice, err := s.getNoticeForSending(ctx, noticeID)
	if err != nil {
		return err
	}

	// 获取通知的发送渠道配置
	channels, err := s.getNotificationChannels(ctx, noticeID)
	if err != nil {
		return err
	}

	// 获取目标用户
	noticeService := NewNoticeService(s.db)
	targetUsers, err := noticeService.GetTargetUsers(ctx, notice)
	if err != nil {
		return fmt.Errorf("获取目标用户失败: %w", err)
	}

	// 未配置渠道时使用默认站内信
	if len(channels) == 0 {
		return s.sendWebNotification(ctx, notice, targetUsers)
	}

	// 获取用户联系信息（邮箱、手机号）
	userInfo, err := s.getUserInfo(ctx, targetUsers)
	if err != nil {
		return fmt.Errorf("获取用户联系信息失败: %w", err)
	}

	// 按渠道类型发送通知
	s.sendNotificationByChannels(ctx, notice, channels, targetUsers, userInfo)

	return nil
}

// getNoticeForSending 获取用于发送的通知信息并验证状态
func (s *NotificationSenderService) getNoticeForSending(ctx context.Context, noticeID string) (*models.Notice, error) {
	noticeService := NewNoticeService(s.db)
	notice, err := noticeService.GetNoticeByID(ctx, noticeID)
	if err != nil {
		return nil, fmt.Errorf("获取通知失败: %w", err)
	}

	// 检查发布状态：仅允许已发布和定时发布中的通知发送
	if !isValidPublishStatus(notice.PublishStatus) {
		return nil, fmt.Errorf("通知未发布或状态无效")
	}

	return notice, nil
}

// isValidPublishStatus 检查发布状态是否有效
func isValidPublishStatus(status models.PublishStatus) bool {
	return status == models.PublishStatusPublished || status == models.PublishStatusScheduled
}

// getNotificationChannels 获取通知的发送渠道配置
func (s *NotificationSenderService) getNotificationChannels(ctx context.Context, noticeID string) ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := s.db.WithContext(ctx).Where("notice_id = ?", noticeID).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取发送渠道失败: %w", err)
	}
	return channels, nil
}

// sendNotificationByChannels 按渠道类型发送通知
func (s *NotificationSenderService) sendNotificationByChannels(
	ctx context.Context,
	notice *models.Notice,
	channels []models.NotificationChannel,
	targetUsers []string,
	userInfo map[string]*userInfo,
) {
	for _, channel := range channels {
		switch channel.ChannelType {
		case models.ChannelTypeWeb:
			if err := s.sendWebNotification(ctx, notice, targetUsers); err != nil {
				logger.Errorf("发送站内信通知失败: %v", err)
			}

		case models.ChannelTypeEmail:
			s.sendEmailNotification(ctx, notice, &channel, userInfo)

		case models.ChannelTypeSMS:
			s.sendSMSNotification(ctx, notice, &channel, userInfo)

		case models.ChannelTypeAPI:
			s.sendAPINotification(ctx, notice, &channel, targetUsers)
		}
	}
}

// sendEmailNotification 发送邮件通知
func (s *NotificationSenderService) sendEmailNotification(
	ctx context.Context,
	notice *models.Notice,
	channel *models.NotificationChannel,
	userInfo map[string]*userInfo,
) {
	emailConfigID, err := s.getEmailConfigID(ctx, channel)
	if err != nil {
		logger.Errorf("获取邮件配置失败: %v", err)
		return
	}

	emails := s.buildRecipientList(s.getUserEmails(userInfo), channel.CustomRecipients)
	if len(emails) == 0 {
		logger.Warnf("[邮件] 没有有效的收件人邮箱地址")
		return
	}

	logger.Infof("[邮件] 发送通知: %s 到 %d 个邮箱", notice.NoticeTitle, len(emails))
	result := s.emailSender.SendNoticeEmail(ctx, *emailConfigID, notice, emails)
	if result.Success {
		logger.Infof("[邮件] 发送成功: %s", result.Message)
	} else {
		logger.Errorf("发送邮件失败: %s", result.Message)
	}
}

// getEmailConfigID 获取邮件配置ID（优先使用指定配置，否则使用默认配置）
func (s *NotificationSenderService) getEmailConfigID(ctx context.Context, channel *models.NotificationChannel) (*string, error) {
	if channel.EmailConfigID != nil {
		return channel.EmailConfigID, nil
	}

	configService := NewNotificationConfigService(s.db)
	defaultConfig, err := configService.GetDefaultEmailConfig(ctx)
	if err != nil {
		return nil, err
	}

	logger.Infof("[邮件] 使用默认邮件配置: %s", defaultConfig.ConfigName)
	return &defaultConfig.ID, nil
}

// buildRecipientList 构建收件人列表（用户 + 自定义收件人，去重）
func (s *NotificationSenderService) buildRecipientList(userItems []string, customRecipients *[]string) []string {
	if customRecipients == nil || len(*customRecipients) == 0 {
		return userItems
	}

	combined := append(userItems, *customRecipients...)
	return uniqueStrings(combined)
}

// sendSMSNotification 发送短信通知
func (s *NotificationSenderService) sendSMSNotification(
	ctx context.Context,
	notice *models.Notice,
	channel *models.NotificationChannel,
	userInfo map[string]*userInfo,
) {
	if channel.APIConfigID == nil {
		return
	}

	phones := s.getUserPhones(userInfo)
	if len(phones) == 0 {
		return
	}

	result := s.apiSender.SendSMS(ctx, *channel.APIConfigID, phones, notice.NoticeContent)
	if !result.Success {
		logger.Errorf("发送短信失败: %s", result.Message)
	}
}

// sendAPINotification 发送API通知（企业微信等）
func (s *NotificationSenderService) sendAPINotification(
	ctx context.Context,
	notice *models.Notice,
	channel *models.NotificationChannel,
	targetUsers []string,
) {
	if channel.APIConfigID == nil {
		return
	}

	recipients := s.buildRecipientList(targetUsers, channel.CustomRecipients)
	logger.Infof("[企微API] 发送通知: %s 到 %d 个用户", notice.NoticeTitle, len(recipients))

	result := s.apiSender.SendNoticeAPI(ctx, *channel.APIConfigID, notice, recipients)
	if !result.Success {
		logger.Errorf("发送API通知失败: %s", result.Message)
	}
}

// sendWebNotification 发送站内信（通过 WebSocket）
// 实际发送由调度器中的 GlobalNoticeHub 完成，此处仅记录日志
func (s *NotificationSenderService) sendWebNotification(_ context.Context, notice *models.Notice, targetUsers []string) error {
	logger.Infof("[站内信] 发送通知: %s 到 %d 个用户", notice.NoticeTitle, len(targetUsers))
	return nil
}

// userInfo 用户联系信息
type userInfo struct {
	userID string
	email  string
	phone  string
}

// getUserInfo 获取用户联系信息（邮箱、手机号）
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

	result := make(map[string]*userInfo, len(users))
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
	emails := make([]string, 0, len(userInfo))
	for _, info := range userInfo {
		if info.email != "" {
			emails = append(emails, info.email)
		}
	}
	return emails
}

// getUserPhones 获取用户手机号列表
func (s *NotificationSenderService) getUserPhones(userInfo map[string]*userInfo) []string {
	phones := make([]string, 0, len(userInfo))
	for _, info := range userInfo {
		if info.phone != "" {
			phones = append(phones, info.phone)
		}
	}
	return phones
}

// PublishAndSendNotice 发布通知并发送到各个渠道
func (s *NotificationSenderService) PublishAndSendNotice(ctx context.Context, noticeID string) error {
	noticeService := NewNoticeService(s.db)

	if err := noticeService.PublishNotice(ctx, noticeID); err != nil {
		return fmt.Errorf("发布通知失败: %w", err)
	}

	return s.SendNotification(ctx, noticeID)
}

// SetNotificationChannels 设置通知的发送渠道
func (s *NotificationSenderService) SetNotificationChannels(ctx context.Context, noticeID string, channels []models.NotificationChannel) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧渠道配置
		if err := tx.Where("notice_id = ?", noticeID).Delete(&models.NotificationChannel{}).Error; err != nil {
			return fmt.Errorf("删除旧渠道配置失败: %w", err)
		}

		// 设置 NoticeID 并创建新渠道配置
		for i := range channels {
			channels[i].NoticeID = noticeID
		}

		if len(channels) > 0 {
			if err := tx.Create(&channels).Error; err != nil {
				return fmt.Errorf("创建新渠道配置失败: %w", err)
			}
		}

		return nil
	})
}

// GetNotificationChannels 获取通知的发送渠道
func (s *NotificationSenderService) GetNotificationChannels(ctx context.Context, noticeID string) ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := s.db.WithContext(ctx).Where("notice_id = ?", noticeID).Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("获取发送渠道失败: %w", err)
	}
	return channels, nil
}

// uniqueStrings 去除字符串切片中的重复元素
func uniqueStrings(slice []string) []string {
	seen := make(map[string]struct{}, len(slice))
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

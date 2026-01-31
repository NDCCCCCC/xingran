package requests

import (
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// NoticeListParams 通知公告列表查询参数
type NoticeListParams struct {
	NoticeTitle *string `json:"noticeTitle"`
	NoticeType  *string `json:"noticeType"`
	CreateTime  *string `json:"createTime"`
	Current     int     `json:"current"`
	PageSize    int     `json:"pageSize"`
}

// DefaultNoticeListParams 默认列表参数
func DefaultNoticeListParams() NoticeListParams {
	return NoticeListParams{
		Current:  constants.DefaultCurrent,
		PageSize: constants.DefaultPageSize,
	}
}

// GetPagination 获取分页参数
func (p *NoticeListParams) GetPagination() (current, pageSize int) {
	current = p.Current
	if current < 1 {
		current = constants.DefaultCurrent
	}
	pageSize = p.PageSize
	if pageSize < constants.MinPageSize {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}
	return current, pageSize
}

// GetOffset 计算偏移量
func (p *NoticeListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// NoticeCreateRequest 创建通知公告请求
type NoticeCreateRequest struct {
	NoticeTitle      string                            `json:"noticeTitle" binding:"required"`
	NoticeType       string                            `json:"noticeType" binding:"required"`
	NoticeContent    string                            `json:"noticeContent" binding:"required"`
	Priority         models.NoticePriority             `json:"priority"`
	PublishTime      *time.Time                        `json:"publishTime"`
	ExecutionType    *string                           `json:"executionType"`
	RecurrenceConfig *RecurrenceConfig                 `json:"recurrenceConfig"`
	CronExpression   *string                           `json:"cronExpression"`
	TargetType       models.TargetType                 `json:"targetType"`
	TargetDepts      []string                          `json:"targetDepts"`
	TargetRoles      []string                          `json:"targetRoles"`
	TargetUsers      []string                          `json:"targetUsers"`
	IsMarkdown       bool                              `json:"isMarkdown"`
	Status           int                               `json:"status"`
	Channels         []NotificationChannelRequest      `json:"channels"`
}

// RecurrenceConfig 周期性执行配置
type RecurrenceConfig struct {
	CronExpression *string `json:"cronExpression"`
	EndDate        *string `json:"endDate"`
}

// NotificationChannelRequest 通知渠道配置请求
type NotificationChannelRequest struct {
	ChannelType      models.NotificationChannelType `json:"channelType"`
	EmailConfigID    *string                        `json:"emailConfigId"`
	APIConfigID      *string                        `json:"apiConfigId"`
	CustomRecipients *[]string                      `json:"customRecipients"`
}

// NoticeUpdateRequest 更新通知公告请求
type NoticeUpdateRequest struct {
	NoticeTitle      *string                           `json:"noticeTitle"`
	NoticeType       *string                           `json:"noticeType"`
	NoticeContent    *string                           `json:"noticeContent"`
	Priority         *models.NoticePriority            `json:"priority"`
	Status           *int                              `json:"status"`
	PublishTime      *time.Time                        `json:"publishTime"`
	ClearPublishTime bool                              `json:"clearPublishTime"`
	Channels         []NotificationChannelRequest      `json:"channels"`
}

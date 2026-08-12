package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NoticePriority 通知优先级
type NoticePriority int

const (
	PriorityNormal    NoticePriority = 0 // 普通
	PriorityImportant NoticePriority = 1 // 重要
	PriorityUrgent    NoticePriority = 2 // 紧急
)

// PublishStatus 发布状态
type PublishStatus int

const (
	PublishStatusDraft     PublishStatus = 0 // 草稿
	PublishStatusPublished PublishStatus = 1 // 已发布
	PublishStatusScheduled PublishStatus = 2 // 定时发布中
	PublishStatusWithdrawn PublishStatus = 3 // 已撤回
)

// TargetType 目标类型
type TargetType int

const (
	TargetAll  TargetType = 0 // 全部用户
	TargetDept TargetType = 1 // 指定部门
	TargetRole TargetType = 2 // 指定角色
	TargetUser TargetType = 3 // 指定用户
)

// NoticeTarget 通知接收范围
type NoticeTarget struct {
	ID         string    `gorm:"primaryKey;type:uuid" json:"id"`
	NoticeID   string    `gorm:"size:64;not null;index:idx_notice_target_notice_id,priority:1" json:"noticeId"`
	TargetType string    `gorm:"size:20;not null;index:idx_notice_target_target,priority:1" json:"targetType"` // dept/role/user
	TargetID   string    `gorm:"size:64;not null;index:idx_notice_target_target,priority:2" json:"targetId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TableName 指定表名
func (NoticeTarget) TableName() string {
	return "sys_notice_target"
}

// NoticeRead 通知阅读记录
type NoticeRead struct {
	ID       string    `gorm:"primaryKey;type:uuid" json:"id"`
	NoticeID string    `gorm:"size:64;not null;index:idx_notice_read_notice_id,priority:1" json:"noticeId"`
	UserID   string    `gorm:"size:64;not null;index:idx_notice_read_user_id,priority:1" json:"userId"`
	ReadAt   time.Time `gorm:"index:idx_notice_read_read_at" json:"readAt"`
	ReadIP   string    `gorm:"size:128" json:"readIp"`
}

// TableName 指定表名
func (NoticeRead) TableName() string {
	return "sys_notice_read"
}

// NoticeIgnore 用户忽略的通知记录
type NoticeIgnore struct {
	ID        string    `gorm:"primaryKey;type:uuid" json:"id"`
	NoticeID  string    `gorm:"size:64;not null;index:idx_notice_ignore_notice_id,priority:1" json:"noticeId"`
	UserID    string    `gorm:"size:64;not null;index:idx_notice_ignore_user_id,priority:1;uniqueIndex:idx_notice_ignore_user_notice" json:"userId"`
	IgnoredAt time.Time `json:"ignoredAt"`
}

// TableName 指定表名
func (NoticeIgnore) TableName() string {
	return "sys_notice_ignore"
}

// BeforeCreate GORM钩子 - NoticeIgnore
func (ni *NoticeIgnore) BeforeCreate(tx *gorm.DB) error {
	if ni.ID == "" {
		ni.ID = uuid.New().String()
	}
	if ni.IgnoredAt.IsZero() {
		ni.IgnoredAt = time.Now()
	}
	return nil
}

// NoticeAttachment 通知附件
type NoticeAttachment struct {
	ID         string    `gorm:"primaryKey;type:uuid" json:"id"`
	NoticeID   string    `gorm:"size:64;not null;index:idx_notice_attachment_notice_id,priority:1" json:"noticeId"`
	FileName   string    `gorm:"size:255;not null" json:"fileName"`
	FilePath   string    `gorm:"size:500;not null" json:"filePath"`
	FileSize   int64     `json:"fileSize"`
	FileType   string    `gorm:"size:100" json:"fileType"`
	UploadTime time.Time `json:"uploadTime"`
	UploadedBy string    `gorm:"size:64;index:idx_notice_attachment_uploaded_by" json:"uploadedBy"`
}

// TableName 指定表名
func (NoticeAttachment) TableName() string {
	return "sys_notice_attachment"
}

// BeforeCreate GORM钩子 - NoticeTarget
func (nt *NoticeTarget) BeforeCreate(tx *gorm.DB) error {
	if nt.ID == "" {
		nt.ID = uuid.New().String()
	}
	return nil
}

// BeforeCreate GORM钩子 - NoticeRead
func (nr *NoticeRead) BeforeCreate(tx *gorm.DB) error {
	if nr.ID == "" {
		nr.ID = uuid.New().String()
	}
	return nil
}

// BeforeCreate GORM钩子 - NoticeAttachment
func (na *NoticeAttachment) BeforeCreate(tx *gorm.DB) error {
	if na.ID == "" {
		na.ID = uuid.New().String()
	}
	return nil
}

// NoticeStatistics 通知阅读统计
type NoticeStatistics struct {
	TotalTargets int     `json:"totalTargets"` // 目标用户总数
	ReadCount    int     `json:"readCount"`    // 已读数
	UnreadCount  int     `json:"unreadCount"`  // 未读数
	ReadRate     float64 `json:"readRate"`     // 阅读率（百分比）
}

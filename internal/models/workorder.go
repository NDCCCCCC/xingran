package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== 枚举定义 ====================

// WorkOrderStatus 工单状态枚举
type WorkOrderStatus int

const (
	WorkOrderStatusPending    WorkOrderStatus = 0 // 待处理
	WorkOrderStatusProcessing WorkOrderStatus = 1 // 处理中
	WorkOrderStatusCompleted  WorkOrderStatus = 2 // 已完成
	WorkOrderStatusClosed     WorkOrderStatus = 3 // 已关闭
	WorkOrderStatusRejected   WorkOrderStatus = 4 // 已拒绝
)

// WorkOrderPriority 工单优先级枚举
type WorkOrderPriority int

const (
	WorkOrderPriorityLow      WorkOrderPriority = 0 // 低
	WorkOrderPriorityMedium   WorkOrderPriority = 1 // 中
	WorkOrderPriorityHigh     WorkOrderPriority = 2 // 高
	WorkOrderPriorityUrgent   WorkOrderPriority = 3 // 紧急
)

// WorkOrderType 工单类型枚举
type WorkOrderType string

const (
	WorkOrderTypeFault    WorkOrderType = "fault"    // 故障
	WorkOrderTypeRequest  WorkOrderType = "request"  // 请求
	WorkOrderTypeChange   WorkOrderType = "change"   // 变更
	WorkOrderTypeIncident WorkOrderType = "incident" // 事件
	WorkOrderTypeQuestion WorkOrderType = "question" // 咨询
)

// WorkOrderCategoryStatus 工单分类状态枚举
type WorkOrderCategoryStatus int

const (
	WorkOrderCategoryStatusEnabled  WorkOrderCategoryStatus = 0 // 启用
	WorkOrderCategoryStatusDisabled WorkOrderCategoryStatus = 1 // 停用
)

// ==================== 模型定义 ====================

// WorkOrderCategory 工单分类
type WorkOrderCategory struct {
	BaseModel
	CategoryName string                 `gorm:"size:100;not null;uniqueIndex:idx_wo_category_name" json:"categoryName"`
	Description  string                 `gorm:"size:500" json:"description,omitempty"`
	Status       WorkOrderCategoryStatus `gorm:"default:0" json:"status"`
	SortOrder    int                    `gorm:"default:0" json:"sortOrder"`
	ParentID     *string                `gorm:"type:uuid" json:"parentId,omitempty"`

	// 关联
	Parent   *WorkOrderCategory   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []WorkOrderCategory  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName 指定表名
func (WorkOrderCategory) TableName() string {
	return "sys_workorder_category"
}

// WorkOrder 运维工单
type WorkOrder struct {
	BaseModel
	Title             string             `gorm:"size:200;not null" json:"title"`
	WorkOrderNo       string             `gorm:"size:50;not null;uniqueIndex:idx_wo_no" json:"workOrderNo"`
	CategoryID        string             `gorm:"type:uuid;not null;index:idx_wo_category,priority:1" json:"categoryId"`
	Type              WorkOrderType      `gorm:"size:20;not null" json:"type"`
	Priority          WorkOrderPriority  `gorm:"default:1" json:"priority"`
	Status            WorkOrderStatus    `gorm:"default:0;index:idx_wo_status,priority:1" json:"status"`
	Description       string             `gorm:"type:text" json:"description"`
	Solution          string             `gorm:"type:text" json:"solution,omitempty"`
	SubmitterID       string             `gorm:"type:uuid;not null;index:idx_wo_submitter,priority:1" json:"submitterId"`
	AssigneeID        *string            `gorm:"type:uuid;index:idx_wo_assignee,priority:1" json:"assigneeId,omitempty"`
	DeptID            *string            `gorm:"type:uuid" json:"deptId,omitempty"`
	ExpectedResolveAt *time.Time         `gorm:"type:datetime" json:"expectedResolveAt,omitempty"`
	ResolvedAt        *time.Time         `gorm:"type:datetime" json:"resolvedAt,omitempty"`
	ClosedAt          *time.Time         `gorm:"type:datetime" json:"closedAt,omitempty"`
	AttachmentIDs     string             `gorm:"size:1000" json:"attachmentIds,omitempty"` // 逗号分隔的附件ID列表

	// 关联
	Category  *WorkOrderCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Submitter *User              `gorm:"foreignKey:SubmitterID" json:"submitter,omitempty"`
	Assignee  *User              `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Dept      *Department        `gorm:"foreignKey:DeptID" json:"department,omitempty"`
	Comments  []WorkOrderComment `gorm:"foreignKey:WorkOrderID" json:"comments,omitempty"`
	History   []WorkOrderHistory `gorm:"foreignKey:WorkOrderID" json:"history,omitempty"`
}

// BeforeCreate GORM钩子 - 生成工单编号
func (wo *WorkOrder) BeforeCreate(tx *gorm.DB) error {
	if wo.WorkOrderNo == "" {
		wo.WorkOrderNo = generateWorkOrderNo()
	}
	return nil
}

// generateWorkOrderNo 生成工单编号（格式：WO20250104123456）
func generateWorkOrderNo() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	// 随机6位数字
	randomPart := now.Format("150405") // 使用时间戳作为后缀
	return "WO" + dateStr + randomPart
}

// TableName 指定表名
func (WorkOrder) TableName() string {
	return "sys_workorder"
}

// WorkOrderComment 工单评论
type WorkOrderComment struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	WorkOrderID string    `gorm:"type:uuid;not null;index:idx_wo_comment_wo,priority:1" json:"workOrderId"`
	UserID      string    `gorm:"type:uuid;not null" json:"userId"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	IsInternal  bool      `gorm:"default:false" json:"isInternal"` // 是否内部评论
	CreatedAt   time.Time `json:"createdAt"`

	// 关联
	WorkOrder *WorkOrder `gorm:"foreignKey:WorkOrderID" json:"workOrder,omitempty"`
	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate GORM钩子 - WorkOrderComment
func (woc *WorkOrderComment) BeforeCreate(tx *gorm.DB) error {
	if woc.ID == "" {
		woc.ID = uuid.New().String()
	}
	if woc.CreatedAt.IsZero() {
		woc.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (WorkOrderComment) TableName() string {
	return "sys_workorder_comment"
}

// WorkOrderHistory 工单操作历史
type WorkOrderHistory struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	WorkOrderID string    `gorm:"type:uuid;not null;index:idx_wo_history_wo,priority:1" json:"workOrderId"`
	Action      string    `gorm:"size:50;not null" json:"action"` // 操作类型：create, assign, update_status, add_comment等
	Field       string    `gorm:"size:50" json:"field,omitempty"`  // 变更字段
	OldValue    string    `gorm:"type:text" json:"oldValue,omitempty"`
	NewValue    string    `gorm:"type:text" json:"newValue,omitempty"`
	Remark      string    `gorm:"type:text" json:"remark,omitempty"`
	OperatorID  string    `gorm:"type:uuid;not null" json:"operatorId"`
	CreatedAt   time.Time `json:"createdAt"`

	// 关联
	WorkOrder *WorkOrder `gorm:"foreignKey:WorkOrderID" json:"workOrder,omitempty"`
	Operator  *User      `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// BeforeCreate GORM钩子 - WorkOrderHistory
func (woh *WorkOrderHistory) BeforeCreate(tx *gorm.DB) error {
	if woh.ID == "" {
		woh.ID = uuid.New().String()
	}
	if woh.CreatedAt.IsZero() {
		woh.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (WorkOrderHistory) TableName() string {
	return "sys_workorder_history"
}

// WorkOrderTemplate 工单模板
type WorkOrderTemplate struct {
	BaseModel
	TemplateName string        `gorm:"size:100;not null" json:"templateName"`
	CategoryID   string        `gorm:"type:uuid;not null" json:"categoryId"`
	Type         WorkOrderType `gorm:"size:20;not null" json:"type"`
	Priority     WorkOrderPriority `gorm:"default:1" json:"priority"`
	Title        string        `gorm:"size:200" json:"title"`
	Description  string        `gorm:"type:text" json:"description"`
	IsEnabled    bool          `gorm:"default:true" json:"isEnabled"`
	ExpectedHours *int         `gorm:"default:24" json:"expectedHours"` // 预期处理时长（小时）

	// 关联
	Category *WorkOrderCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

// TableName 指定表名
func (WorkOrderTemplate) TableName() string {
	return "sys_workorder_template"
}

// WorkOrderConfig 工单配置
type WorkOrderConfig struct {
	BaseModel
	AutoAssignEnabled  bool   `gorm:"default:false" json:"autoAssignEnabled"`   // 是否自动分配
	AutoCloseDays      int    `gorm:"default:7" json:"autoCloseDays"`            // 完成后自动关闭天数（0=手动关闭）
	AllowUserClose     bool   `gorm:"default:false" json:"allowUserClose"`       // 是否允许用户关闭工单
	NotificationEnabled bool  `gorm:"default:true" json:"notificationEnabled"`   // 是否启用通知
	EmailNotification   bool  `gorm:"default:false" json:"emailNotification"`    // 邮件通知
	SmsNotification     bool  `gorm:"default:false" json:"smsNotification"`      // 短信通知
}

// TableName 指定表名
func (WorkOrderConfig) TableName() string {
	return "sys_workorder_config"
}

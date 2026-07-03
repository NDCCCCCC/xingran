package models

import (
	"strings"
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
	WorkOrderPriorityLow    WorkOrderPriority = 0 // 低
	WorkOrderPriorityMedium WorkOrderPriority = 1 // 中
	WorkOrderPriorityHigh   WorkOrderPriority = 2 // 高
	WorkOrderPriorityUrgent WorkOrderPriority = 3 // 紧急
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
	CategoryName string                  `gorm:"size:100;not null" json:"categoryName"`
	Description  string                  `gorm:"size:500" json:"description,omitempty"`
	Status       WorkOrderCategoryStatus `gorm:"default:0" json:"status"`
	SortOrder    int                     `gorm:"default:0" json:"sortOrder"`
	ParentID     *string                 `gorm:"type:uuid" json:"parentId,omitempty"`

	// 关联
	Parent   *WorkOrderCategory  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []WorkOrderCategory `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName 指定表名
func (WorkOrderCategory) TableName() string {
	return "sys_workorder_category"
}

// WorkOrder 运维工单
type WorkOrder struct {
	BaseModel
	Title             string            `gorm:"size:200;not null" json:"title"`
	WorkOrderNo       string            `gorm:"size:50;not null" json:"workOrderNo"`
	CategoryID        string            `gorm:"type:uuid;not null;index:idx_wo_category,priority:1" json:"categoryId"`
	Type              WorkOrderType     `gorm:"size:20;not null" json:"type"`
	Priority          WorkOrderPriority `gorm:"default:1" json:"priority"`
	Status            WorkOrderStatus   `gorm:"default:0;index:idx_wo_status,priority:1" json:"status"`
	Description       string            `gorm:"type:text" json:"description"`
	Solution          string            `gorm:"type:text" json:"solution,omitempty"`
	SubmitterID       string            `gorm:"type:uuid;not null;index:idx_wo_submitter,priority:1" json:"submitterId"`
	AssigneeID        *string           `gorm:"type:uuid;index:idx_wo_assignee,priority:1" json:"assigneeId,omitempty"`
	DeptID            *string           `gorm:"type:uuid" json:"deptId,omitempty"`
	ExpectedResolveAt *time.Time        `gorm:"type:timestamp" json:"expectedResolveAt,omitempty"`
	ResolvedAt        *time.Time        `gorm:"type:timestamp" json:"resolvedAt,omitempty"`
	ClosedAt          *time.Time        `gorm:"type:timestamp" json:"closedAt,omitempty"`
	AttachmentIDs     string            `gorm:"size:1000" json:"attachmentIds,omitempty"` // 逗号分隔的附件ID列表

	// 自动分配相关字段
	IsAutoAssigned bool    `gorm:"default:false" json:"isAutoAssigned"`     // 是否自动分配
	DutyPoolID     *string `gorm:"type:uuid" json:"dutyPoolId,omitempty"`   // 关联的值班池ID
	DutyType       string  `gorm:"size:20" json:"dutyType,omitempty"`       // 值班类型
	AssignStrategy string  `gorm:"size:50" json:"assignStrategy,omitempty"` // 使用的分配策略

	// 关联
	Category  *WorkOrderCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Submitter *User              `gorm:"foreignKey:SubmitterID" json:"submitter,omitempty"`
	Assignee  *User              `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Dept      *Department        `gorm:"foreignKey:DeptID" json:"department,omitempty"`
	Comments  []WorkOrderComment `gorm:"foreignKey:WorkOrderID" json:"comments,omitempty"`
	History   []WorkOrderHistory `gorm:"foreignKey:WorkOrderID" json:"history,omitempty"`
	Ratings   []WorkOrderRating  `gorm:"foreignKey:WorkOrderID" json:"ratings,omitempty"`
}

// BeforeCreate GORM钩子 - 生成工单编号
func (wo *WorkOrder) BeforeCreate(tx *gorm.DB) error {
	// 调用 BaseModel 的 BeforeCreate 生成 UUID
	if wo.ID == "" {
		wo.ID = uuid.New().String()
	}
	// 生成工单编号
	if wo.WorkOrderNo == "" {
		wo.WorkOrderNo = generateWorkOrderNo()
	}
	return nil
}

// generateWorkOrderNo 生成工单编号(格式:WO + YYYYMMDD + 12 位 UUID hex 后缀)
//
// 历史 bug(2026-06-30):原实现用 now.Format("150405")(HHMMSS) 作为"随机6位数字",
// reconciliation:createWorkorderHigh 同秒内批量创建 15 张工单时全部得到同一编号,
// 触发 sys_workorder.idx_wo_no UNIQUE 约束 (SQLSTATE 23505)。
//
// 修复:复用 internal/scheduler/workorder_tasks.go:131-144 已有的碰撞安全样板
// (WO + YYYYMMDD + uuid hex[:12]),碰撞概率 < 10^-12,无需重试。
//
// 格式示例: WO20260630a1b2c3d4e5f6(人类仍可识别日期前缀,后缀肉眼无意义)
func generateWorkOrderNo() string {
	now := time.Now()
	dateStr := now.Format("20060102")
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	return "WO" + dateStr + suffix
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
	Field       string    `gorm:"size:50" json:"field,omitempty"` // 变更字段
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
	TemplateName  string            `gorm:"size:100;not null" json:"templateName"`
	CategoryID    string            `gorm:"type:uuid;not null" json:"categoryId"`
	Type          WorkOrderType     `gorm:"size:20;not null" json:"type"`
	Priority      WorkOrderPriority `gorm:"default:1" json:"priority"`
	Title         string            `gorm:"size:200" json:"title"`
	Description   string            `gorm:"type:text" json:"description"`
	IsEnabled     bool              `gorm:"default:true" json:"isEnabled"`
	ExpectedHours *int              `gorm:"default:24" json:"expectedHours"` // 预期处理时长（小时）

	// 关联
	Category *WorkOrderCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

// TableName 指定表名
func (WorkOrderTemplate) TableName() string {
	return "sys_workorder_template"
}

// WorkOrderRating 工单评价（双向评价）
type WorkOrderRating struct {
	ID               string    `gorm:"primaryKey;type:uuid" json:"id"`
	WorkOrderID      string    `gorm:"type:uuid;not null;index:idx_wo_rating_wo,priority:1" json:"workOrderId"`
	RatingType       string    `gorm:"size:20;not null" json:"ratingType"` // user(用户评价) 或 handler(处理人员评价)
	CompletionScore  int       `gorm:"default:0" json:"completionScore"`   // 用户评价：完成度评分 (1-5)
	CooperationScore int       `gorm:"default:0" json:"cooperationScore"`  // 处理人员评价：配合度评分 (1-5)
	Comment          string    `gorm:"type:text" json:"comment"`           // 评价内容
	RaterID          string    `gorm:"type:uuid;not null" json:"raterId"`  // 评价人ID
	CreatedAt        time.Time `json:"createdAt"`

	// 关联
	WorkOrder *WorkOrder `gorm:"foreignKey:WorkOrderID" json:"workOrder,omitempty"`
	Rater     *User      `gorm:"foreignKey:RaterID" json:"rater,omitempty"`
}

// BeforeCreate GORM钩子 - WorkOrderRating
func (wor *WorkOrderRating) BeforeCreate(tx *gorm.DB) error {
	if wor.ID == "" {
		wor.ID = uuid.New().String()
	}
	if wor.CreatedAt.IsZero() {
		wor.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (WorkOrderRating) TableName() string {
	return "sys_workorder_rating"
}

// WorkOrderConfig 工单配置（不需要软删除）
type WorkOrderConfig struct {
	ID                      string    `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
	AutoAssignEnabled       bool      `gorm:"default:true" json:"autoAssignEnabled"`                  // 是否自动分配
	AutoAssignTarget        string    `gorm:"size:50;default:'duty_pool'" json:"autoAssignTarget"`    // 分配目标
	AutoAssignStrategy      string    `gorm:"size:50;default:'assign_one'" json:"autoAssignStrategy"` // 分配策略
	AutoCloseDays           int       `gorm:"default:7" json:"autoCloseDays"`                         // 完成后自动关闭天数（0=手动关闭）
	AllowUserClose          bool      `gorm:"default:false" json:"allowUserClose"`                    // 是否允许用户关闭工单
	NotificationEnabled     bool      `gorm:"default:true" json:"notificationEnabled"`                // 是否启用通知
	EmailNotification       bool      `gorm:"default:false" json:"emailNotification"`                 // 邮件通知
	SmsNotification         bool      `gorm:"default:false" json:"smsNotification"`                   // 短信通知
	RatingEnabled           bool      `gorm:"default:true" json:"ratingEnabled"`                      // 评价功能开关
	KnowledgeConvertEnabled bool      `gorm:"default:true" json:"knowledgeConvertEnabled"`            // 知识库转换开关
}

// TableName 指定表名
func (WorkOrderConfig) TableName() string {
	return "sys_workorder_config"
}

// BeforeCreate GORM钩子 - WorkOrderConfig
func (c *WorkOrderConfig) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}
	return nil
}

// ==================== 周期性工单 ====================

// PeriodicWorkOrderAssignType 周期性工单分配类型枚举
type PeriodicWorkOrderAssignType string

const (
	PeriodicAssignTypeManual   PeriodicWorkOrderAssignType = "manual"    // 手动指定
	PeriodicAssignTypeDutyPool PeriodicWorkOrderAssignType = "duty_pool" // 当天值班人员
	PeriodicAssignTypeRotation PeriodicWorkOrderAssignType = "rotation"  // 轮询
)

// PeriodicWorkOrderTemplate 周期性工单模板
type PeriodicWorkOrderTemplate struct {
	BaseModel
	TemplateName   string                      `gorm:"size:100;not null" json:"templateName"`
	WorkOrderTitle string                      `gorm:"size:200;not null" json:"workOrderTitle"`
	Description    string                      `gorm:"type:text" json:"description"`
	CategoryID     string                      `gorm:"type:uuid;not null" json:"categoryId"`
	Type           WorkOrderType               `gorm:"size:20;not null" json:"type"`
	Priority       WorkOrderPriority           `gorm:"default:1" json:"priority"`
	CronExpression string                      `gorm:"size:100;not null" json:"cronExpression"`       // Cron表达式
	AssignType     PeriodicWorkOrderAssignType `gorm:"size:20;default:'duty_pool'" json:"assignType"` // 分配类型
	AssignTargetID *string                     `gorm:"type:uuid" json:"assignTargetId,omitempty"`     // 分配目标ID（当assignType为manual时使用）
	IsEnabled      bool                        `gorm:"default:true" json:"isEnabled"`
	NextRunAt      *time.Time                  `gorm:"type:timestamp" json:"nextRunAt,omitempty"` // 下次执行时间
	JobID          *string                     `gorm:"type:varchar(50)" json:"jobId,omitempty"`   // 关联的定时任务ID
	TotalGenerated int                         `gorm:"default:0" json:"totalGenerated"`           // 已生成的工单数量
	NotifyAssignee bool                        `gorm:"default:true" json:"notifyAssignee"`        // 是否通知处理人

	// 关联
	Category      *WorkOrderCategory     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Job           *Job                   `gorm:"foreignKey:JobID;-:migration" json:"job,omitempty"`
	ExecutionLogs []PeriodicWorkOrderLog `gorm:"foreignKey:TemplateID" json:"executionLogs,omitempty"`
}

// TableName 指定表名
func (PeriodicWorkOrderTemplate) TableName() string {
	return "sys_periodic_workorder_template"
}

// PeriodicWorkOrderLog 周期性工单执行记录
type PeriodicWorkOrderLog struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	TemplateID  string    `gorm:"type:uuid;not null;index:idx_pwo_log_template,priority:1" json:"templateId"`
	WorkOrderID string    `gorm:"type:uuid;not null;index:idx_pwo_log_wo,priority:1" json:"workOrderId"`
	ExecutedAt  time.Time `gorm:"not null;index:idx_pwo_log_executed" json:"executedAt"`
	JobID       *string   `gorm:"type:varchar(50)" json:"jobId,omitempty"` // 关联的定时任务ID
	Status      string    `gorm:"size:20;default:'success'" json:"status"` // success, failed
	Result      string    `gorm:"type:text" json:"result,omitempty"`       // 执行结果
	ErrorMsg    string    `gorm:"type:text" json:"errorMsg,omitempty"`     // 错误信息

	// 关联
	Template  *PeriodicWorkOrderTemplate `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	WorkOrder *WorkOrder                 `gorm:"foreignKey:WorkOrderID" json:"workOrder,omitempty"`
	Job       *Job                       `gorm:"foreignKey:JobID;-:migration" json:"job,omitempty"`
}

// BeforeCreate GORM钩子 - PeriodicWorkOrderLog
func (pwol *PeriodicWorkOrderLog) BeforeCreate(tx *gorm.DB) error {
	if pwol.ID == "" {
		pwol.ID = uuid.New().String()
	}
	if pwol.ExecutedAt.IsZero() {
		pwol.ExecutedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (PeriodicWorkOrderLog) TableName() string {
	return "sys_periodic_workorder_log"
}

package rpa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeTask         ResourceType = "task"         // 任务
	ResourceTypeWorker       ResourceType = "worker"       // Worker
	ResourceTypeExecution    ResourceType = "execution"    // 执行记录
	ResourceTypeSchedule     ResourceType = "schedule"     // 调度
	ResourceTypeVariable     ResourceType = "variable"     // 变量
	ResourceTypeNotification ResourceType = "notification" // 通知
)

// AuditAction 操作类型
type AuditAction string

const (
	AuditActionCreate  AuditAction = "create"  // 创建
	AuditActionUpdate  AuditAction = "update"  // 更新
	AuditActionDelete  AuditAction = "delete"  // 删除
	AuditActionExecute AuditAction = "execute" // 执行
	AuditActionCancel  AuditAction = "cancel"  // 取消
	AuditActionStart   AuditAction = "start"   // 开始
	AuditActionStop    AuditAction = "stop"    // 停止
)

// AuditResult 操作结果
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success" // 成功
	AuditResultFailed  AuditResult = "failed"  // 失败
)

// AuditLog RPA审计日志
type AuditLog struct {
	ID           string                 `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ResourceType ResourceType           `gorm:"size:50;not null" json:"resourceType"`
	ResourceID   string                 `gorm:"type:uuid;not null" json:"resourceId"`
	Action       AuditAction            `gorm:"size:50;not null" json:"action"`
	// OldValue / NewValue 是 jsonb 列，但声明为 map —— **必须**加 serializer:json，
	// 否则 GORM 把 map 直接交给 database/sql 驱动，PG 与 sqlite 都会报
	// 「unsupported type map[string]interface {}」而写入失败（Phase 155 实测复现：
	// LIVEVIEW-05 的审计事件在 sqlite 下 500）。VDI 审计（services/vdi/audit_service.go）
	// 复用本模型，同一处修复同时清偿该路径。
	OldValue     map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"oldValue"`
	NewValue     map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"newValue"`
	OperatorID   string                 `gorm:"size:64" json:"operatorId"`
	OperatorName string                 `gorm:"size:100" json:"operatorName"`
	IPAddress    string                 `gorm:"size:50" json:"ipAddress"`
	UserAgent    string                 `gorm:"type:text" json:"userAgent"`
	Result       AuditResult            `gorm:"size:20;default:'success'" json:"result"`
	ErrorMessage string                 `gorm:"type:text" json:"errorMessage"`
	CreatedAt    time.Time              `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "sys_rpa_audit_logs"
}

// BeforeCreate GORM钩子
//
// Phase 155 / LIVEVIEW-05：补应用层 UUID。此前恒空——PG 靠建表默认
// gen_random_uuid() 兜底，但 sqlite 不支持该函数（CREATE TABLE 直接
// syntax error near "("），故 sqlite 分支无法建表、审计必然写不进。
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// IsSuccess 是否成功
func (a *AuditLog) IsSuccess() bool {
	return a.Result == AuditResultSuccess
}

// IsFailed 是否失败
func (a *AuditLog) IsFailed() bool {
	return a.Result == AuditResultFailed
}

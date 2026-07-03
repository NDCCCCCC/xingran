package models

import (
	"time"
)

// SysReconciliationFixSuggestion 资产对账半自动修复建议
//
// Phase 46 R5 落地:对 Type B 异常(物理链路无责 / 责任人有)且 confidence_score
// 高于阈值时,Generator cron 写入此表,运维在 UI 接受/拒绝/应用/回滚。
//
// 字段集(24 + BaseModel):
//   - 关联:ExceptionID(FK → sys_data_reconciliation.id,索引)
//   - 修复源(D-A1 锁定):仅写 ops_asset.user_id,不写 dept_id / NowUserName / DeptName
//   - 状态机(D-B2 6 状态):pending / accepted / rejected / applied / rolled_back / failed
//   - 6 个时间戳字段:AcceptedAt / RejectedAt / AppliedAt / RolledBackAt
//   - 4 个操作人字段:均 *string,system 自动生成时为 NULL
//   - 回滚窗口(D-C2):RollbackWindowUntil = applied_at + 7d
//   - 多轮版本化(D-B3):SupersededAt 表示被新建议取代
//   - 客户端 IP:ApplyClientIP / RollbackClientIP
//
// 索引约定:
//   - GORM tags 自动建 idx_fix_suggestion_exception / idx_fix_suggestion_status 2 个索引
//   - migration_198 手动建 2 个补充索引(idx_fix_suggestion_status_created /
//     idx_fix_suggestion_applied_at)
//   - migration_199 手动建 1 个 partial unique index uniq_fix_suggestion_pending_per_exception
//     阻止同 exception 多个 pending
type SysReconciliationFixSuggestion struct {
	BaseModel

	// 关联 — 指向 sys_data_reconciliation
	ExceptionID string `gorm:"type:uuid;not null;column:exception_id;index:idx_fix_suggestion_exception,priority:1" json:"exceptionId"`

	// 修复源(D-A1 锁定:仅 user_id)
	// SuggestedUserID:从 reconciliation_normalized.physical_user_id 取
	SuggestedUserID *string `gorm:"size:64;column:suggested_user_id" json:"suggestedUserId,omitempty"`
	// PreFixUserID:Apply 时从 ops_asset.user_id 读出后回填(回滚用)
	PreFixUserID *string `gorm:"size:64;column:pre_fix_user_id" json:"preFixUserId,omitempty"`

	// 置信度与原因
	ConfidenceScore float64 `gorm:"type:decimal(3,2);not null;column:confidence_score" json:"confidenceScore"`
	Reason          string  `gorm:"type:text;not null;column:reason" json:"reason"`

	// 状态机(D-B2 6 状态)
	FixStatus    string `gorm:"size:16;not null;default:'pending';column:fix_status;index:idx_fix_suggestion_status,priority:1" json:"fixStatus"`
	ConflictType string `gorm:"size:2;not null;column:conflict_type" json:"conflictType"` // 冗余,免 JOIN

	// 6 状态时间戳
	AcceptedAt   *time.Time `gorm:"column:accepted_at" json:"acceptedAt,omitempty"`
	RejectedAt   *time.Time `gorm:"column:rejected_at" json:"rejectedAt,omitempty"`
	AppliedAt    *time.Time `gorm:"column:applied_at" json:"appliedAt,omitempty"`
	RolledBackAt *time.Time `gorm:"column:rolled_back_at" json:"rolledBackAt,omitempty"`

	// 操作人(可空 — 自动生成时为系统)
	AcceptedBy   *string `gorm:"size:64;column:accepted_by" json:"acceptedBy,omitempty"`
	RejectedBy   *string `gorm:"size:64;column:rejected_by" json:"rejectedBy,omitempty"`
	AppliedBy    *string `gorm:"size:64;column:applied_by" json:"appliedBy,omitempty"`
	RolledBackBy *string `gorm:"size:64;column:rolled_back_by" json:"rolledBackBy,omitempty"`

	// 必填原因(D-C3 审计:reject/rollback 强制 ≥10 字符)
	RejectionReason *string `gorm:"type:text;column:rejection_reason" json:"rejectionReason,omitempty"`
	RollbackReason  *string `gorm:"type:text;column:rollback_reason" json:"rollbackReason,omitempty"`

	// 回滚窗口(D-C2 7d 锁):applied_at + 7d
	RollbackWindowUntil *time.Time `gorm:"column:rollback_window_until" json:"rollbackWindowUntil,omitempty"`

	// 多轮版本化(D-B3):被新建议取代
	SupersededAt *time.Time `gorm:"column:superseded_at" json:"supersededAt,omitempty"`

	// 客户端 IP(审计追溯)
	ApplyClientIP    *string `gorm:"size:64;column:apply_client_ip" json:"applyClientIp,omitempty"`
	RollbackClientIP *string `gorm:"size:64;column:rollback_client_ip" json:"rollbackClientIp,omitempty"`
}

// TableName 设置表名
func (SysReconciliationFixSuggestion) TableName() string {
	return "sys_reconciliation_fix_suggestion"
}

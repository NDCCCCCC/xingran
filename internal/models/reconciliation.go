package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// SysDataReconciliation 资产对账异常记录
//
// 记录 Layer 3 检测引擎识别出的资产对账异常(Type B-F),每条记录代表
// "1 个资产 + 1 个冲突类型" 的当前未解决异常。状态语义:
//   - resolved_at IS NULL → 开放(open)
//   - resolved_at NOT NULL → 已解决(resolved)
//
// 唯一性约束由 migration_168 的 partial unique index uniq_recon_asset_type_open
// (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL
// 保证,D-11 防告警风暴。
//
// Phase 42 R1 字段集,Phase 42 R2-R4 视情况扩展。
type SysDataReconciliation struct {
	BaseModel

	// 核心标识
	AssetID string `gorm:"type:uuid;not null;column:asset_id;index:idx_recon_asset_id,priority:1" json:"assetId"`
	// ConflictType 取值字典 asset_reconciliation_conflict_type (A/B/C/D/E/F),size:2 与字典值匹配
	ConflictType string `gorm:"size:2;not null;column:conflict_type;index:idx_recon_conflict_type,priority:1" json:"conflictType"`

	// ReconCategory Phase 48 sibling 列(D-06):业务类型分类,与 conflict_type(A-F)正交。
	// 取值字典 asset_reconciliation_recon_category,当前唯一值:component_serial(组件序列号对账)。
	// 由 migration_201 DROP uniq_recon_asset_type_open 并重建为
	// uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category) WHERE open 兼容本列。
	ReconCategory *string `gorm:"size:32;column:recon_category;index:idx_recon_category,priority:1" json:"reconCategory,omitempty"`

	// 严重程度
	// 取值字典 asset_reconciliation_severity: low / medium / high / critical
	Severity string `gorm:"size:16;not null;column:severity;index:idx_recon_severity,priority:1" json:"severity"`

	// 三路证据 (physical / declared / ad) — 由 Layer 3 检测填充,R1 主要用于审计
	PhysicalValue  json.RawMessage `gorm:"type:jsonb;column:physical_value" json:"physicalValue,omitempty"`
	DeclaredValue  json.RawMessage `gorm:"type:jsonb;column:declared_value" json:"declaredValue,omitempty"`
	ADValue        json.RawMessage `gorm:"type:jsonb;column:ad_value" json:"adValue,omitempty"`
	ConfidenceScore float64        `gorm:"type:decimal(3,2);column:confidence_score" json:"confidenceScore"`

	// 原始快照(检测时刻的全量上下文,R1 用于审计,R5 可能用于半自动修复回滚)
	RawSnapshot json.RawMessage `gorm:"type:jsonb;not null;column:raw_snapshot" json:"rawSnapshot"`

	// 资产 IP(直接取 ops_asset.machine_ip 单值,D-03 不做多 IP 解析)
	AssetIP *string `gorm:"type:inet;column:asset_ip" json:"assetIp,omitempty"`

	// 例外规则 (R3 接入,FK 指向 sys_reconciliation_exception.id)
	ExceptionRuleID *string `gorm:"type:uuid;column:exception_rule_id" json:"exceptionRuleId,omitempty"`

	// 已应用动作 (R2 写入,no_alert / no_notice / no_workorder / skip_severity / silence)
	AppliedActions pq.StringArray `gorm:"type:text[];column:applied_actions" json:"appliedActions,omitempty"`

	// 生命周期
	DetectedAt    time.Time  `gorm:"not null;default:now();column:detected_at;index:idx_recon_detected_at,priority:1" json:"detectedAt"`
	ResolvedAt    *time.Time `gorm:"column:resolved_at" json:"resolvedAt,omitempty"`
	ResolvedBy    *string    `gorm:"column:resolved_by" json:"resolvedBy,omitempty"`
	ResolutionNote *string   `gorm:"type:text;column:resolution_note" json:"resolutionNote,omitempty"`

	// 工单 (R2 自动生成工单时回写)
	WorkorderID *string `gorm:"type:uuid;column:workorder_id" json:"workorderId,omitempty"`
}

// TableName 设置表名
func (SysDataReconciliation) TableName() string {
	return "sys_data_reconciliation"
}

// SysReconciliationException 对账例外规则
//
// IP 段 + 冲突类型组合的例外规则,命中后跳过告警/工单/严重度升级。
// R1 仅 seed 数据,R3 接入 CIDR GiST 索引 + CRUD UI。
//
// 字段约定:
//   - is_active 遵循 Status Convention (0=启用, 1=停用)
//   - reason 应用层校验 ≥10 字符 (R3 service 层 enforce)
//   - exception_actions 取值字典 asset_reconciliation_exception_action
type SysReconciliationException struct {
	BaseModel

	Name string `gorm:"size:128;not null;column:name" json:"name"`

	// CIDR 段 (R3 用 GiST 索引加速匹配;R1 仅 seed,无应用层消费)
	IPRange string `gorm:"type:cidr;not null;column:ip_range" json:"ipRange"`

	// 命中的冲突类型列表 (A/B/C/D/E/F)
	ConflictTypes pq.StringArray `gorm:"type:text[];column:conflict_types" json:"conflictTypes,omitempty"`

	// 命中后应用的动作(必填)
	ExceptionActions pq.StringArray `gorm:"type:text[];not null;column:exception_actions" json:"exceptionActions"`

	// 严重度覆盖 (可选;为空表示沿用检测出的 severity)
	SeverityOverride *string `gorm:"size:16;column:severity_override" json:"severityOverride,omitempty"`

	// 范围:global / dept / user 三种 (R1 仅 seed global)
	ScopeType string  `gorm:"size:16;not null;default:'global';column:scope_type" json:"scopeType"`
	ScopeID   *string `gorm:"type:uuid;column:scope_id" json:"scopeId,omitempty"`

	// 备注/原因 (R3 service 层 enforce ≥10 字符)
	Reason string `gorm:"type:text;not null;column:reason" json:"reason"`

	// 状态 (Status Convention: 0=启用, 1=停用)
	IsActive int `gorm:"default:0;column:is_active" json:"isActive"`

	// 过期时间 (R3 cleanupExpiredExceptions cron 任务扫描)
	ExpiresAt *time.Time `gorm:"column:expires_at" json:"expiresAt,omitempty"`

	// 创建者
	CreatedBy int64 `gorm:"not null;column:created_by" json:"createdBy"`
}

// TableName 设置表名
func (SysReconciliationException) TableName() string {
	return "sys_reconciliation_exception"
}
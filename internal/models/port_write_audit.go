package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PortWriteAudit 端口写操作审计行（append-only，未脱敏真相源）
//
// Phase 52 W3 引入，记录每个端口写命令的完整 before/after 状态 + 设备响应。
// 与 sys_oper_log 的关系：handler 先 INSERT audit 拿 audit_ids，
// 再 operlog.Record(..., WithOperParam({audit_ids:[...]}))（Path C 兜底，
// 不改 operlog 包接口；audit.oper_log_id 列保持 NULL）。
//
// 设计依据 (52-CONTEXT D-01 + 52-RESEARCH §3.6 + 52-PATTERNS §2)：
//   - append-only：无 UpdatedAt / DeletedAt / BaseTimeLine embed
//   - JSONB 列用 json.RawMessage 避免每次写入 marshal/unmarshal 开销
//   - FailureReason / OperLogID nullable，用 *string 指针
//   - 复合索引 (device_id, port_id, created_at) + 单列索引 (created_at)
//   - BeforeCreate 钩子生成 UUID（与 DevicePortStatus pattern 一致）
//
// 表名锁定：sys_port_write_audit（单数，与 sys_device_port_status 风格一致）。
type PortWriteAudit struct {
	ID             string          `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID       string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:1" json:"deviceId"`
	PortID         string          `gorm:"type:uuid;not null;index:idx_port_write_audit_device_port_created,priority:2" json:"portId"`
	Action         string          `gorm:"size:32;not null" json:"action"`
	BeforeValue    json.RawMessage `gorm:"type:jsonb" json:"beforeValue"`
	AfterValue     json.RawMessage `gorm:"type:jsonb" json:"afterValue"`
	CommandSent    string          `gorm:"type:text" json:"commandSent"`
	DeviceResponse string          `gorm:"type:text" json:"deviceResponse"`
	Status         string          `gorm:"size:16;not null" json:"status"`
	FailureReason  *string         `gorm:"type:text" json:"failureReason,omitempty"`
	Operator       string          `gorm:"size:50" json:"operator"`
	OperLogID      *string         `gorm:"type:uuid" json:"operLogId,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;index:idx_port_write_audit_device_port_created,priority:3;index:idx_port_write_audit_created" json:"createdAt"`
}

// TableName 表名（Phase 52 D-01 锁定：sys_port_write_audit 单数）
func (PortWriteAudit) TableName() string {
	return "sys_port_write_audit"
}

// BeforeCreate GORM 钩子：创建前自动生成 UUID（与 DevicePortStatus pattern 一致）
func (a *PortWriteAudit) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

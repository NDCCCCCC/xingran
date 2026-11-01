package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PortWriteSnapshot 端口写快照（Phase 123 PORTW-04）。
//
// critical 动作（shutdown/set_access_vlan）下发前抓取端口当前状态 + 预生成
// reverse 回滚命令；下发失败时自动按逆序回滚，回滚结果落本表 status 字段。
//
// status 状态机：
//   - captured   已抓快照，下发尚未判定
//   - kept       下发成功，快照保留（供人工核对/追溯）
//   - rolled_back 下发失败，回滚命令执行成功
//   - rollback_failed 下发失败且回滚也失败 —— 显式告警状态，不允许静默
type PortWriteSnapshot struct {
	ID          string `gorm:"type:uuid;primary_key" json:"id"`
	PortID      string `gorm:"type:uuid;not null;index" json:"portId"`
	DeviceID    string `gorm:"type:uuid;not null;index" json:"deviceId"`
	InterfaceName string `gorm:"size:100" json:"interfaceName,omitempty"`
	Action      string `gorm:"size:50;not null" json:"action"`
	// BeforeState 端口下发前状态 JSON（adminStatus/description/vlan/dot1xEnabled）
	BeforeState string `gorm:"type:text" json:"beforeState"`
	// RollbackCommands 预生成的逆序回滚命令 JSON（[]string）
	RollbackCommands string `gorm:"type:text" json:"rollbackCommands,omitempty"`
	// Status captured | kept | rolled_back | rollback_failed
	Status string `gorm:"size:20;not null;default:captured" json:"status"`
	// RollbackError 回滚失败原因（status=rollback_failed 时非空）
	RollbackError string `gorm:"size:500" json:"rollbackError,omitempty"`
	// AuditID 关联 sys_port_write_audit 行（可空）
	AuditID string `gorm:"type:uuid" json:"auditId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName 表名。
func (PortWriteSnapshot) TableName() string { return "sys_port_write_snapshot" }

// BeforeCreate UUID 兜底。
func (p *PortWriteSnapshot) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// PortWriteLock 端口写互斥锁 DB 标记（Phase 123 PORTW-05）。
//
// 与内存锁双层：内存锁覆盖单实例并发，DB 行跨实例互斥。锁带 expires_at TTL，
// 进程重启不残留（过期行在 Acquire 时先清理），crash 后 TTL 自然释放。
type PortWriteLock struct {
	PortID    string `gorm:"type:uuid;primary_key" json:"portId"`
	Holder    string `gorm:"size:100;not null" json:"holder"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName 表名。
func (PortWriteLock) TableName() string { return "sys_port_write_lock" }

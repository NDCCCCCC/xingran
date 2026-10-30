package models

// Phase 57 D-05/D-08 outbox-lite 队列表（无软删 / 模型不声明唯一索引）

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ADPushEntityType 推送目标实体类型（string 枚举）。
type ADPushEntityType string

const (
	// ADPushEntityUser 用户实体 —— v1.21 唯一实体类型；预留 dept/group 扩展位。
	ADPushEntityUser ADPushEntityType = "user"
)

// ADPushActionType 推送动作类型（string 枚举）。
type ADPushActionType string

const (
	// ADPushActionUpdateUser 用户四字段变更（nickname/email/phone/deptId）推送。
	ADPushActionUpdateUser ADPushActionType = "update_user"
	// ADPushActionDisable sys 软删用户 → 禁用 AD 账号（UAC 514，幂等可重放）。
	// 注意: UpdateStatus 双向推送（停用→disable / 启用→enable）明确排除在 v1.21 外,
	// 此处仅预留扩展位。
	ADPushActionDisable ADPushActionType = "disable"
	// ADPushActionUpdateManager 重算并回写用户的 AD manager 属性（经理同步规则引擎）。
	//
	// 触发时机：部门负责人变更（该部门子树内全部用户）、用户岗位变更、用户部门变更、
	// 规则/机构配置变更后的批量重算。排水器按当前规则重新解析经理并写 AD manager。
	// 与 ADPushActionUpdateUser 相互独立：四字段 diff 命中不上 manager 变更，
	// 反之亦然；EnqueueDisable 的 F-01 压制**只作用于 update_user**，不压制本动作。
	ADPushActionUpdateManager ADPushActionType = "update_manager"
)

// ADPushStatus 队列行状态（string 而非 int：部分唯一索引谓词与排障 SQL 可读）。
type ADPushStatus string

const (
	ADPushStatusPending ADPushStatus = "pending" // 待排水
	ADPushStatusFailed  ADPushStatus = "failed"  // 排水失败待退避重试
	ADPushStatusPushed  ADPushStatus = "pushed"  // 终态：推送成功
	ADPushStatusDead    ADPushStatus = "dead"    // 终态：不可恢复（ErrADTargetNotFound）
)

// ADPushQueueActivePredicate 活跃行谓词 —— **单一事实源**（Phase 57 F-01）。
//
// 同一串文本被两处强制复用，缺一即 ON CONFLICT 失效：
//  1. migration_215 的 uq_ad_push_queue_active 部分唯一索引 WHERE 子句（DDL）
//  2. addomain.adPushOnConflictClause 的 ON CONFLICT ... TargetWhere（冲突目标推断）
//
// ⚠ 红线：谓词文本**不得改写、不得加表名限定**。写错 → PG 42P10 / sqlite
// "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint"。
// ⚠ 任何"把终态行打回/留在活跃集合"的新逻辑都必须选一个**不在本谓词内**的终态，
// 否则那批行继续占用唯一索引、同实体的新入队行直接撞唯一约束（见 EnqueueDisable 的 F-01 压制）。
// TestPhase57_ActiveRowPredicate_MatchesIndexDDL 锁定 DDL 与本常量一致。
const ADPushQueueActivePredicate = "status IN ('pending', 'failed')"

// ADPushQueue AD 推送队列表（Phase 57 outbox-lite 存储载体）。
//
// 载荷 = 最小动作行（entity_type / entity_id / action_type），禁用 JSON 快照
// （D-05：快照过期是已判死刑的时间戳仲裁土壤）；排水闭包重读 DB 后全量推送。
//
// 不含 DeletedAt —— 队列表不软删（D-08），终态行留表可查（QUEUE-03 排障）。
// 不含任何唯一索引 tag —— W-1 教训：GORM AutoMigrate 的 DROP 无 IF EXISTS，
// 与手写迁移建的 uq_ad_push_queue_active 冲突会导致启动 FATA；唯一索引只由
// migration_215 建。
//
// 不内嵌 BaseModel（BaseModel 带 DeletedAt）。
type ADPushQueue struct {
	ID         string           `gorm:"type:uuid;primary_key" json:"id"`
	EntityType ADPushEntityType `gorm:"size:32;not null" json:"entityType"`
	EntityID   string           `gorm:"size:64;not null" json:"entityId"`
	ActionType ADPushActionType `gorm:"size:32;not null" json:"actionType"`
	Status     ADPushStatus     `gorm:"size:20;not null;default:'pending'" json:"status"` // F-14：与 migration_215 的 DDL DEFAULT 'pending' 对齐
	Attempts   int              `gorm:"not null;default:0" json:"attempts"`
	// NextRetryAt 退避重试时间（Phase 58 退避扫描用，nullable 不阻断）
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty"`
	LastError   string     `gorm:"type:text" json:"lastError,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	// PushedAt / FailedAt 可空终态列
	PushedAt *time.Time `json:"pushedAt,omitempty"`
	FailedAt *time.Time `json:"failedAt,omitempty"`
}

// BeforeCreate GORM 钩子：生成 UUID 与时间戳（DDL 无 DB default，
// PG 下 uuid 列缺省会 23502 —— 23502 防御惯例）。
func (q *ADPushQueue) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.Status == "" {
		q.Status = ADPushStatusPending
	}
	now := time.Now().UTC() // Phase 58 F-12：全链路 UTC
	if q.CreatedAt.IsZero() {
		q.CreatedAt = now
	}
	if q.UpdatedAt.IsZero() {
		q.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子：更新前刷新 UpdatedAt
func (q *ADPushQueue) BeforeUpdate(tx *gorm.DB) error {
	q.UpdatedAt = time.Now().UTC() // Phase 58 F-12：全链路 UTC
	return nil
}

// TableName 设置表名
func (ADPushQueue) TableName() string {
	return "ad_push_queue"
}

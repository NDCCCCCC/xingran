package models

import (
	"time"

	"gorm.io/gorm"
)

// ADServiceAccount AD 域控服务账号池中的单个账号
// Phase 36: 多账号 + 自动故障切换
//
// 状态机：
//   0 = 可用
//   1 = 管理员手动停用
//   2 = 熔断中（CircuitBreakerUntil > now 仍熔断；到期由 cron 任务统一恢复）
//
// 字段名 `PasswordCiphertext` 含 `password` 关键词，
// operlog.RecordWithBody 会自动脱敏为 `******`（OPERLOG-03 兼容）。
type ADServiceAccount struct {
	ID                  string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConfigID            string         `gorm:"type:uuid;not null;index" json:"configId"`
	Username            string         `gorm:"size:255;not null" json:"username"`
	PasswordCiphertext  string         `gorm:"type:text;not null;column:password_ciphertext" json:"-"` // 永不出 JSON
	Status              int            `gorm:"not null;default:0" json:"status"`
	FailureCount        int            `gorm:"not null;default:0" json:"failureCount"`
	CircuitBreakerUntil *time.Time     `json:"circuitBreakerUntil,omitempty"`
	LastSuccessAt       *time.Time     `json:"lastSuccessAt,omitempty"`
	LastFailureAt       *time.Time     `json:"lastFailureAt,omitempty"`
	LastFailureReason   string         `gorm:"type:text" json:"lastFailureReason,omitempty"`
	ManualUnlockReason  string         `gorm:"type:text" json:"manualUnlockReason,omitempty"`
	ManualUnlockedBy    string         `gorm:"size:64" json:"manualUnlockedBy,omitempty"`
	ManualUnlockedAt    *time.Time     `json:"manualUnlockedAt,omitempty"`
	Remark              string         `gorm:"size:500" json:"remark,omitempty"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 显式指定表名
func (ADServiceAccount) TableName() string {
	return "sys_ad_service_accounts"
}

// IsAvailable 判断账号当前是否可用（业务层用）
func (a *ADServiceAccount) IsAvailable() bool {
	if a.Status == 1 { // 停用
		return false
	}
	if a.Status == 2 { // 熔断中且未到期
		if a.CircuitBreakerUntil == nil || time.Now().Before(*a.CircuitBreakerUntil) {
			return false
		}
	}
	return true
}

// IsCircuitBroken 判断是否处于熔断中
func (a *ADServiceAccount) IsCircuitBroken() bool {
	return a.Status == 2 && a.CircuitBreakerUntil != nil && time.Now().Before(*a.CircuitBreakerUntil)
}

// IsDisabled 判断是否被管理员手动停用
func (a *ADServiceAccount) IsDisabled() bool {
	return a.Status == 1
}

// StatusText 返回状态的中文描述（前端展示用）
func (a *ADServiceAccount) StatusText() string {
	switch a.Status {
	case 0:
		return "可用"
	case 1:
		return "已停用"
	case 2:
		return "熔断中"
	default:
		return "未知"
	}
}

// ADServiceAccountListResponse 列表接口响应（含分页）
type ADServiceAccountListResponse struct {
	List     []ADServiceAccount `json:"list"`
	Total    int64              `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"pageSize"`
}

// ADServiceAccountStats 池状态摘要
type ADServiceAccountStats struct {
	Total          int    `json:"total"`
	Available      int    `json:"available"`
	Disabled       int    `json:"disabled"`
	CircuitBroken  int    `json:"circuitBroken"`
	CurrentAccount string `json:"currentAccount,omitempty"` // 当前 PickAvailable 选中的账号（best-effort）
}
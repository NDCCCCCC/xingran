// VM账号管理数据模型扩展

package vdi

import (
	"time"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/base"
)

// VDIVirtualMachine 虚拟机模型（扩展版）
type VDIVirtualMachine struct {
	base.Base

	// === 原有字段 ===
	VMID           string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"vm_id"`
	Name           string    `gorm:"type:varchar(200);not null" json:"name"`
	ResourceID     string    `gorm:"type:varchar(100);index" json:"resource_id"`
	Status         int       `gorm:"type:int;default:0" json:"status"`
	PowerState     string    `gorm:"type:varchar(50)" json:"power_state"`
	IPAddress      string    `gorm:"type:varchar(50)" json:"ip_address"`
	MACAddress     string    `gorm:"type:varchar(50)" json:"mac_address"`
	OSType         string    `gorm:"type:varchar(50)" json:"os_type"`
	CPU            int       `json:"cpu"`
	Memory         int       `json:"memory"`
	Disk           int       `json:"disk"`
	BoundUserID    *string   `gorm:"type:varchar(100)" json:"bound_user_id"`
	BoundUserName  *string   `gorm:"type:varchar(200)" json:"bound_user_name"`
	PolicyGroupID  *string   `gorm:"type:varchar(100)" json:"policy_group_id"`
	LastSyncAt     *time.Time `json:"last_sync_at"`
	VdiServerID    string    `gorm:"type:varchar(100);index" json:"vdi_server_id"`

	// === 新增：账号管理字段 ===
	// 初始管理员账号信息
	InitialAdminUsername  *string `gorm:"type:varchar(100)" json:"initial_admin_username"`
	InitialAdminPassword  *string `gorm:"type:varchar(500)" json:"-"` // SM4加密，不返回给前端
	InitialAdminPasswordEncrypted *string `gorm:"column:initial_admin_password;type:varchar(500)" json:"-"` // 加密字段

	// SSH/RDP连接信息
	SSHPort              int    `gorm:"type:int;default:22" json:"ssh_port"`
	RDPPort              int    `gorm:"type:int;default:3389" json:"rdp_port"`
	AgentInstalled       bool   `gorm:"type:bool;default:false" json:"agent_installed"`
	AgentVersion         *string `gorm:"type:varchar(50)" json:"agent_version"`
	AgentLastHeartbeat    *time.Time `json:"agent_last_heartbeat"`

	// 账号管理策略
	PasswordPolicyID     *string `gorm:"type:varchar(100)" json:"password_policy_id"` // 密码策略
	AllowLocalAccountSync bool   `gorm:"type:bool;default:true" json:"allow_local_account_sync"`
}

// VDIVMAccount 虚拟机内部账号表
type VDIVMAccount struct {
	base.Base

	VMID          string `gorm:"type:varchar(100);index;not null" json:"vm_id"`
	AccountID     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"account_id"` // 格式: vm_id:username

	// 账号信息
	Username      string `gorm:"type:varchar(100);not null" json:"username"`
	PasswordEncrypted string `gorm:"type:varchar(500);not null" json:"-"` // SM4加密
	PasswordHash  string `gorm:"type:varchar(200)" json:"-"` // 用于验证的哈希值

	// 账号属性
	AccountType   string `gorm:"type:varchar(50);not null" json:"account_type"` // admin/user/service
	OSType        string `gorm:"type:varchar(50);not null" json:"os_type"`        // windows/linux
	UID           *int   `json:"uid"`    // Linux用户ID
	GID           *int   `json:"gid"`    // Linux组ID
	HomeDir       *string `gorm:"type:varchar(500)" json:"home_dir"`
	Shell         *string `gorm:"type:varchar(200)" json:"shell"`

	// 权限和状态
	IsAdmin       bool   `gorm:"type:bool;default:false" json:"is_admin"`
	IsEnabled     bool   `gorm:"type:bool;default:true" json:"is_enabled"`
	IsLocked      bool   `gorm:"type:bool;default:false" json:"is_locked"`
	PasswordExpired bool `gorm:"type:bool;default:false" json:"password_expired"`

	// 元数据
	Description   *string `gorm:"type:varchar(500)" json:"description"`
	CreatedBy     string `gorm:"type:varchar(100)" json:"created_by"` // 创建人
	LastPasswordChange *time.Time `json:"last_password_change"`
	LastLogin     *time.Time `json:"last_login"`

	// 同步状态
	SyncStatus    string `gorm:"type:varchar(50);default:'pending'" json:"sync_status"` // pending/synced/failed
	SyncedAt      *time.Time `json:"synced_at"`
	LastSyncError *string `gorm:"type:text" json:"last_sync_error"`
}

// VDIVMAuditLog 虚拟机账号操作审计日志
type VDIVMAuditLog struct {
	base.Base

	VMID          string `gorm:"type:varchar(100);index" json:"vm_id"`
	AccountID     string `gorm:"type:varchar(100);index" json:"account_id"`

	// 操作信息
	Operation     string `gorm:"type:varchar(50);not null" json:"operation"` // create/update/delete/enable/disable/reset_password
	Operator      string `gorm:"type:varchar(100);not null" json:"operator"`
	OperatorIP    string `gorm:"type:varchar(50)" json:"operator_ip"`

	// 操作详情
	Details       string `gorm:"type:text" json:"details"` // JSON格式的详细信息
	OldValue      *string `gorm:"type:text" json:"old_value"` // 操作前的值
	NewValue      *string `gorm:"type:text" json:"new_value"` // 操作后的值

	// 执行结果
	Status        string `gorm:"type:varchar(50);not null" json:"status"` // success/failed/pending
	ErrorMessage  *string `gorm:"type:text" json:"error_message"`

	// 执行时间
	ExecutedAt    *time.Time `json:"executed_at"`
}

// VMPasswordPolicy 密码策略
type VMPasswordPolicy struct {
	base.Base

	PolicyName    string `gorm:"type:varchar(200);not null;uniqueIndex" json:"policy_name"`

	// 密码复杂度要求
	MinLength     int `gorm:"type:int;default:8" json:"min_length"`
	MaxLength     int `gorm:"type:int;default:32" json:"max_length"`
	RequireUppercase bool `gorm:"type:bool;default:true" json:"require_uppercase"`
	RequireLowercase bool `gorm:"type:bool;default:true" json:"require_lowercase"`
	RequireNumber    bool `gorm:"type:bool;default:true" json:"require_number"`
	RequireSpecial   bool `gorm:"type:bool;default:true" json:"require_special"`

	// 密码生命周期
	MaxAgeDays    int `gorm:"type:int;default:90" json:"max_age_days"`
	MinAgeDays    int `gorm:"type:int;default:1" json:"min_age_days"`
	ExpireWarnDays int `gorm:"type:int;default:7" json:"expire_warn_days"`

	// 历史密码
	HistoryCount  int `gorm:"type:int;default:5" json:"history_count"` // 禁止使用最近N次密码

	// 账号锁定策略
	LockoutThreshold int `gorm:"type:int;default:5" json:"lockout_threshold"` // 失败多少次后锁定
	LockoutDuration  int `gorm:"type:int;default:30" json:"lockout_duration"`    // 锁定时长（分钟）

	// 适用范围
	ApplyToVMs    *string `gorm:"type:text" json:"apply_to_vms"` // JSON数组：VM ID列表
	ApplyToAllVMs bool `gorm:"type:bool;default:false" json:"apply_to_all_vms"`

	// 状态
	IsEnabled     bool `gorm:"type:bool;default:true" json:"is_enabled"`
}

// TableName 指定表名
func (VDIVMAccount) TableName() string {
	return "sys_vdi_vm_accounts"
}

func (VDIVMAuditLog) TableName() string {
	return "sys_vdi_vm_audit_logs"
}

func (VMPasswordPolicy) TableName() string {
	return "sys_vdi_password_policies"
}

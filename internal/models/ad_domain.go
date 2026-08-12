package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== AD域配置 ====================

// ADConfigStatus AD配置状态
type ADConfigStatus int

const (
	ADConfigStatusEnabled  ADConfigStatus = 0 // 启用
	ADConfigStatusDisabled ADConfigStatus = 1 // 停用
)

// ADConfig AD域配置
type ADConfig struct {
	BaseModel
	ConfigName    string         `gorm:"size:100;not null" json:"configName"`
	ServerAddress string         `gorm:"size:255;not null" json:"serverAddress"`
	ServerPort    int            `gorm:"default:389" json:"serverPort"`
	DomainName    string         `gorm:"size:255;not null" json:"domainName"`
	BaseDN        string         `gorm:"size:500;not null" json:"baseDn"`
	// AdminUsername @Deprecated Phase 38: 单管理员路径已全部移除（同步/管理/登录统一走 sys_ad_service_accounts
	// 账号池 FailoverClient）。字段保留作 DB 列（D-02 不做 DROP COLUMN），不再读写；not null 由 migration_164 放宽。
	AdminUsername string `gorm:"size:255" json:"adminUsername"`
	// AdminPassword @Deprecated Phase 38: 同 AdminUsername（加密密文列保留作 DB 列，不再读写）。
	AdminPassword string `gorm:"size:500" json:"adminPassword"`
	UseSSL        bool           `gorm:"default:false" json:"useSsl"`
	UseTLS        bool           `gorm:"default:false" json:"useTls"`
	SyncEnabled   bool           `gorm:"default:true" json:"syncEnabled"`
	SyncInterval  int            `gorm:"default:3600" json:"syncInterval"` // 秒
	MemberOUDN    string         `gorm:"size:500;column:member_ou_dn" json:"memberOuDn,omitempty"` // 本部部门分组OU DN，用于创建和管理部门组
	LastSyncAt    *time.Time     `json:"lastSyncAt,omitempty"`
	Status        ADConfigStatus `gorm:"default:0" json:"status"`
}

func (ADConfig) TableName() string {
	return "sys_ad_config"
}

// ==================== OU组织单位 ====================

// ADOU AD域OU组织单位
type ADOU struct {
	BaseModel
	ADConfigID  string     `gorm:"type:uuid;not null;index:idx_ad_ou_config,priority:1" json:"adConfigId"`
	OUN         string     `gorm:"size:500;not null;column:ou_dn" json:"ouDn"`
	OUName      string     `gorm:"size:255;not null" json:"ouName"`
	OUPath      string     `gorm:"type:text" json:"ouPath,omitempty"`
	ParentDN    string     `gorm:"size:500;index:idx_ad_ou_parent" json:"parentDn,omitempty"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	UserCount   int        `gorm:"default:0" json:"userCount"`
	GroupCount  int        `gorm:"default:0" json:"groupCount"`
	LastSyncAt  *time.Time `json:"lastSyncAt,omitempty"`
}

func (ADOU) TableName() string {
	return "sys_ad_ou"
}

// Children 子OU列表(用于构建树形结构)
func (o *ADOU) Children() []*ADOU {
	var children []*ADOU
	// 这里需要通过Service层查询填充
	return children
}

// ==================== AD用户组 ====================

// ADGroupScope 用户组作用域
type ADGroupScope string

const (
	ADGroupScopeGlobal    ADGroupScope = "Global"
	ADGroupScopeLocal     ADGroupScope = "Local"
	ADGroupScopeUniversal ADGroupScope = "Universal"
)

// ADGroupType 用户组类型
type ADGroupType int

const (
	ADGroupTypeSecurity     ADGroupType = 1 // 安全组
	ADGroupTypeDistribution ADGroupType = 2 // 分发组
)

// ADGroup AD域用户组
type ADGroup struct {
	BaseModel
	ADConfigID  string       `gorm:"type:uuid;not null;index:idx_ad_group_config,priority:1;uniqueIndex:uni_sys_ad_group_config_dn,priority:1" json:"adConfigId"`
	GroupDN     string       `gorm:"size:500;not null;uniqueIndex:uni_sys_ad_group_config_dn,priority:2" json:"groupDn"`
	GroupName   string       `gorm:"size:255;not null" json:"groupName"`
	GroupScope  ADGroupScope `gorm:"size:50" json:"groupScope,omitempty"`
	GroupType   ADGroupType  `gorm:"default:1" json:"groupType,omitempty"`
	Description string       `gorm:"type:text" json:"description,omitempty"`
	MemberCount int          `gorm:"default:0" json:"memberCount"`
	OUN         string       `gorm:"size:500;index:idx_ad_group_ou;column:ou_dn" json:"ouDn,omitempty"`
	LastSyncAt  *time.Time   `json:"lastSyncAt,omitempty"`
}

func (ADGroup) TableName() string {
	return "sys_ad_group"
}

// ==================== AD用户 ====================

// ADUser AD域用户
type ADUser struct {
	BaseModel
	ADConfigID         string     `gorm:"type:uuid;not null;index:idx_ad_user_config,priority:1;uniqueIndex:uni_sys_ad_user_config_dn,priority:1" json:"adConfigId"`
	UserDN             string     `gorm:"size:500;not null;uniqueIndex:uni_sys_ad_user_config_dn,priority:2" json:"userDn"`
	Username           string     `gorm:"type:varchar;not null;index:idx_ad_user_username" json:"username"`
	DisplayName        string     `gorm:"size:255" json:"displayName,omitempty"`
	Email              string     `gorm:"size:255;index:idx_ad_user_email" json:"email,omitempty"`
	Phone              string     `gorm:"size:50" json:"phone,omitempty"`
	Mobile             string     `gorm:"size:50" json:"mobile,omitempty"`
	Title              string     `gorm:"size:100" json:"title,omitempty"`
	Department         string     `gorm:"size:255" json:"department,omitempty"`
	Company            string     `gorm:"size:255" json:"company,omitempty"`
	OUN                string     `gorm:"size:500;index:idx_ad_user_ou;column:ou_dn" json:"ouDn,omitempty"`
	UserAccountControl int        `json:"userAccountControl,omitempty"`
	IsEnabled          bool       `gorm:"default:true;index:idx_ad_user_enabled" json:"isEnabled"`
	IsLocked           bool       `gorm:"default:false" json:"isLocked"`
	PasswordExpired    bool       `gorm:"default:false" json:"passwordExpired"`
	LastLogon          *time.Time `json:"lastLogon,omitempty"`
	PasswordLastSet    *time.Time `json:"passwordLastSet,omitempty"`
	AccountExpires     *time.Time `json:"accountExpires,omitempty"`
	Description        string     `gorm:"type:text" json:"description,omitempty"`
	MemberOf           string     `gorm:"type:text" json:"memberOf,omitempty"` // 逗号分隔DN
	LastSyncAt         *time.Time `json:"lastSyncAt,omitempty"`
}

func (ADUser) TableName() string {
	return "sys_ad_user"
}

// GetGroupDNs 获取用户所属组的DN列表
func (u *ADUser) GetGroupDNs() []string {
	if u.MemberOf == "" {
		return []string{}
	}
	// memberOf is stored as semicolon-separated DN values
	parts := strings.Split(u.MemberOf, ";")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ==================== 用户组成员关系 ====================

// ADGroupMember 用户组成员关系
type ADGroupMember struct {
	BaseModel
	ADConfigID string `gorm:"type:uuid;not null;index:idx_ad_gm_config" json:"adConfigId"`
	GroupDN    string `gorm:"size:500;not null;index:idx_ad_gm_group" json:"groupDn"`
	UserDN     string `gorm:"size:500;not null;index:idx_ad_gm_user" json:"userDn"`
}

func (ADGroupMember) TableName() string {
	return "sys_ad_group_member"
}

// ==================== 同步日志 ====================

// ADSyncType 同步类型
type ADSyncType string

const (
	ADSyncTypeFull        ADSyncType = "full"        // 全量同步
	ADSyncTypeIncremental ADSyncType = "incremental" // 增量同步
)

// ADSyncStatus 同步状态
type ADSyncStatus string

const (
	ADSyncStatusRunning ADSyncStatus = "running" // 运行中
	ADSyncStatusSuccess ADSyncStatus = "success" // 成功
	ADSyncStatusFailed  ADSyncStatus = "failed"  // 失败
)

// ADSyncLog 同步日志
type ADSyncLog struct {
	ID            string       `gorm:"primaryKey;type:uuid" json:"id"`
	ADConfigID    string       `gorm:"type:uuid;not null;index:idx_ad_sync_log_config,priority:1" json:"adConfigId"`
	SyncType      ADSyncType   `gorm:"size:50;not null" json:"syncType"`
	SyncStatus    ADSyncStatus `gorm:"size:20;not null" json:"syncStatus"`
	StartTime     time.Time    `gorm:"not null;index:idx_ad_sync_log_time,priority:2" json:"startTime"`
	EndTime       *time.Time   `json:"endTime,omitempty"`
	Duration      int          `json:"duration,omitempty"` // 秒
	OUCount       int          `gorm:"default:0" json:"ouCount"`
	GroupCount    int          `gorm:"default:0" json:"groupCount"`
	UserCount     int          `gorm:"default:0" json:"userCount"`
	ComputerCount int          `gorm:"default:0" json:"computerCount"`
	ErrorCount    int          `gorm:"default:0" json:"errorCount"`
	ErrorMsg      string       `gorm:"type:text;column:error_msg" json:"errorMessage,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
}

// BeforeCreate GORM钩子
func (l *ADSyncLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	return nil
}

func (ADSyncLog) TableName() string {
	return "sys_ad_sync_log"
}

// ==================== AD用户账户控制标志常量 ====================

const (
	// ADScriptFlag 脚本标志
	ADScriptFlag = 0x0001
	// ADAccountDisable 账户禁用
	ADAccountDisable = 0x0002
	// ADHomDirRequired 主目录必需
	ADHomDirRequired = 0x0008
	// ADLockout 锁定
	ADLockout = 0x0010
	// ADPasswordNotRequired 不需要密码
	ADPasswordNotRequired = 0x0020
	// ADPasswordCannotChange 密码不能更改
	ADPasswordCannotChange = 0x0040
	// ADNormalAccount 正常账户
	ADNormalAccount = 0x0200
	// ADDontExpirePassword 密码永不过期
	ADDontExpirePassword = 0x10000
	// ADPasswordExpired 密码已过期
	ADPasswordExpired = 0x800000
)

// IsDisabledByUAC 检查账户是否禁用
func (u *ADUser) IsDisabledByUAC() bool {
	return u.UserAccountControl&ADAccountDisable != 0
}

// IsLockedByUAC 检查账户是否锁定
func (u *ADUser) IsLockedByUAC() bool {
	return u.UserAccountControl&ADLockout != 0
}

// IsPasswordExpiredByUAC 检查密码是否过期
func (u *ADUser) IsPasswordExpiredByUAC() bool {
	return u.UserAccountControl&ADPasswordExpired != 0
}

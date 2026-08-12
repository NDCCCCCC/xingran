package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OUGroupMappingStatus OU组映射状态
type OUGroupMappingStatus string

const (
	OUGroupMappingStatusActive   OUGroupMappingStatus = "active"   // 激活
	OUGroupMappingStatusInactive OUGroupMappingStatus = "inactive" // 停用
)

// OUGroupMapping OU与AD组映射模型
// 用于AD域控OU与用户组的直接关联，替代原来的部门-组映射
type OUGroupMapping struct {
	ID            string                `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ADConfigID    string                `gorm:"type:uuid;not null;index:idx_ou_group_config,priority:1" json:"adConfigId"`
	OUDN          string                `gorm:"column:ou_dn;size:500;not null;uniqueIndex:uni_ou_group_mapping_ou,priority:0;index:idx_ou_group_mapping_dn" json:"ouDn"`
	OUName        string                `gorm:"column:ou_name;size:255;not null" json:"ouName"`
	ADGroupID     string                `gorm:"type:uuid;not null;uniqueIndex:uni_ou_group_mapping_ou,priority:1;index:idx_ou_group_mapping_group" json:"adGroupId"`
	MappingStatus OUGroupMappingStatus `gorm:"size:20;not null;default:active" json:"mappingStatus"`
	SyncEnabled   bool                  `gorm:"default:true" json:"syncEnabled"` // 是否启用成员同步
	LastSyncAt    *time.Time            `json:"lastSyncAt,omitempty"`
	CreatedBy     string                `gorm:"type:uuid" json:"createdBy,omitempty"`
	UpdatedBy     string                `gorm:"type:uuid" json:"updatedBy,omitempty"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`

	// 关联（不持久化）
	ADGroup  *ADGroup  `gorm:"foreignKey:ADGroupID" json:"adGroup,omitempty"`
	ADConfig *ADConfig `gorm:"foreignKey:ADConfigID" json:"adConfig,omitempty"`
}

// BeforeCreate GORM钩子 - 创建前
func (o *OUGroupMapping) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// TableName 设置表名
func (OUGroupMapping) TableName() string {
	return "sys_ou_group_mapping"
}

// OUGroupMappingSyncLog OU组同步日志
type OUGroupMappingSyncLog struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MappingID       string    `gorm:"type:uuid;not null;index:idx_ou_group_sync_log,priority:1" json:"mappingId"`
	OUdn            string    `gorm:"column:ou_dn;size:500;not null" json:"ouDn"`
	ADGroupID       string    `gorm:"type:uuid;not null" json:"adGroupId"`
	SyncType        string    `gorm:"size:50;not null" json:"syncType"` // full/incremental/member_sync
	MembersAdded    int       `gorm:"default:0" json:"membersAdded"`
	MembersRemoved  int       `gorm:"default:0" json:"membersRemoved"`
	TotalMembers    int       `gorm:"default:0" json:"totalMembers"`
	Status          string    `gorm:"size:20;not null" json:"status"` // success/failed/partial
	ErrorMsg        *string   `gorm:"type:text" json:"errorMsg,omitempty"`
	StartedAt       time.Time `json:"startedAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	DurationMs      int       `json:"durationMs"`
	CreatedAt       time.Time `json:"createdAt"`
}

// BeforeCreate GORM钩子 - 创建前
func (o *OUGroupMappingSyncLog) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// TableName 设置表名
func (OUGroupMappingSyncLog) TableName() string {
	return "sys_ou_group_mapping_sync_log"
}
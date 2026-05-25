package models

import "time"

// MappingType 映射类型
type MappingType string

const (
	MappingTypeAuto   MappingType = "auto"   // 自动映射（根据组名匹配）
	MappingTypeManual MappingType = "manual" // 手动映射
)

// MappingStatus 映射状态
type MappingStatus string

const (
	MappingStatusActive   MappingStatus = "active"   // 激活
	MappingStatusInactive MappingStatus = "inactive" // 停用
)

// DeptGroupMapping 部门与AD组映射
type DeptGroupMapping struct {
	ID           string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DeptID       string         `gorm:"type:uuid;not null;index:idx_dept_group_dept,priority:1;uniqueIndex:uni_dept_group_mapping,priority:1" json:"deptId"`
	ADGroupID    string         `gorm:"type:uuid;not null;index:idx_dept_group_group,priority:1;uniqueIndex:uni_dept_group_mapping,priority:2" json:"adGroupId"`
	ADConfigID   string         `gorm:"type:uuid;not null;index:idx_dept_group_config" json:"adConfigId"`
	MappingType  MappingType    `gorm:"size:20;not null;default:'auto'" json:"mappingType"`
	MappingStatus MappingStatus `gorm:"size:20;not null;default:'active'" json:"mappingStatus"`
	GroupDN      string         `gorm:"size:500;not null" json:"groupDn"` // 冗余存储DN，避免JOIN查询
	GroupName    string         `gorm:"size:255;not null" json:"groupName"` // 冗余存储组名
	SyncEnabled  bool           `gorm:"default:true" json:"syncEnabled"` // 是否启用成员同步
	LastSyncAt   *time.Time     `json:"lastSyncAt,omitempty"`
	CreatedBy    string         `gorm:"type:uuid" json:"createdBy,omitempty"`
	UpdatedBy    string         `gorm:"type:uuid" json:"updatedBy,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    *time.Time     `gorm:"index" json:"deletedAt,omitempty"`
}

// TableName 指定表名
func (DeptGroupMapping) TableName() string {
	return "sys_dept_group_mapping"
}

// DeptGroupMappingSyncLog 部门组同步日志
type DeptGroupMappingSyncLog struct {
	ID              string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MappingID       string    `gorm:"type:uuid;not null;index:idx_mapping_sync_log,priority:1" json:"mappingId"`
	DeptID          string    `gorm:"type:uuid;not null" json:"deptId"`
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

// TableName 指定表名
func (DeptGroupMappingSyncLog) TableName() string {
	return "sys_dept_group_mapping_sync_log"
}

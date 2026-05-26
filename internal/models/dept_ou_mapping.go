package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeptOUMapping 部门-OU映射模型
// 用于系统部门与AD域控OU的双向映射
type DeptOUMapping struct {
	ID         string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	DeptID     string     `gorm:"type:uuid;not null;uniqueIndex:uni_dept_ou_mapping_dept,priority:0;index:idx_dept_ou_mapping_dept" json:"deptId"`
	ADConfigID string     `gorm:"type:uuid;not null;uniqueIndex:uni_dept_ou_mapping_dept,priority:1;index:idx_dept_ou_mapping_config" json:"adConfigId"`
	OUDN       string     `gorm:"column:ou_dn;size:500;not null;index:idx_dept_ou_mapping_dn" json:"ouDn"`
	OUName     string     `gorm:"column:ou_name;size:255;not null" json:"ouName"`
	ParentOUDN *string    `gorm:"column:parent_ou_dn;size:500" json:"parentOuDn,omitempty"`
	SyncEnabled bool      `gorm:"default:true" json:"syncEnabled"`
	SyncStatus string     `gorm:"size:20;default:pending;index:idx_dept_ou_mapping_status" json:"syncStatus"` // pending/synced/failed
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`

	// 关联（不持久化）
	Dept     *Department `gorm:"foreignKey:DeptID" json:"dept,omitempty"`
	ADConfig *ADConfig   `gorm:"foreignKey:ADConfigID" json:"adConfig,omitempty"`
}

// BeforeCreate GORM钩子 - 创建前
func (d *DeptOUMapping) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// TableName 设置表名
func (DeptOUMapping) TableName() string {
	return "sys_dept_ou_mapping"
}

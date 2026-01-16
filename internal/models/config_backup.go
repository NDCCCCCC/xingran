package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackupType 备份类型枚举
type BackupType string

const (
	BackupTypeAuto   BackupType = "auto"   // 自动备份
	BackupTypeManual BackupType = "manual" // 手动备份
)

// StorageType 存储类型枚举
type StorageType string

const (
	StorageTypeDatabase StorageType = "database" // 数据库存储
	StorageTypeFile     StorageType = "file"     // 文件系统存储
)

// ConfigBackup 配置备份模型
type ConfigBackup struct {
	ID           string      `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID     string      `gorm:"type:uuid;not null" json:"deviceId"`
	DeviceName   string      `gorm:"size:100" json:"deviceName,omitempty"`
	BackupType   BackupType  `gorm:"size:50;not null" json:"backupType"`
	StorageType  StorageType `gorm:"size:50;not null;default:database" json:"storageType"`
	ConfigHash   string      `gorm:"size:64" json:"configHash,omitempty"` // 用于差异对比
	Version      int         `gorm:"default:1" json:"version"`
	ChangeReason string      `gorm:"type:text" json:"changeReason,omitempty"`

	// 配置内容（数据库存储）
	ConfigContent string `gorm:"type:text" json:"configContent,omitempty"`

	// 文件信息（文件系统存储）
	FilePath   string `gorm:"size:500" json:"filePath,omitempty"`
	BackupSize int    `json:"backupSize,omitempty"`            // 实际配置大小（字节）
	Compressed bool   `gorm:"default:false" json:"compressed"` // 是否压缩

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	CreatedBy string     `gorm:"size:64" json:"createdBy"`
	DeletedAt *time.Time `gorm:"index" json:"deletedAt,omitempty"`
}

// BeforeCreate GORM 钩子：在创建记录前生成 UUID 和设置创建时间
func (cb *ConfigBackup) BeforeCreate(tx *gorm.DB) error {
	if cb.ID == "" {
		cb.ID = uuid.New().String()
	}
	now := time.Now()
	if cb.CreatedAt.IsZero() {
		cb.CreatedAt = now
	}
	if cb.UpdatedAt.IsZero() {
		cb.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子：在更新记录前更新 UpdatedAt
func (cb *ConfigBackup) BeforeUpdate(tx *gorm.DB) error {
	cb.UpdatedAt = time.Now()
	return nil
}

// TableName 设置表名
func (ConfigBackup) TableName() string {
	return "sys_config_backup"
}

// IsStoredInDatabase 是否存储在数据库
func (cb *ConfigBackup) IsStoredInDatabase() bool {
	return cb.StorageType == StorageTypeDatabase
}

// IsStoredInFile 是否存储在文件系统
func (cb *ConfigBackup) IsStoredInFile() bool {
	return cb.StorageType == StorageTypeFile
}

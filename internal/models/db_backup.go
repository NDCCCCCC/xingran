package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DBBackup 全库备份记录（表 sys_db_backup）。
//
// 与 sys_config_backup（网络设备 running-config 版本链）职责完全隔离：
// 前者是数据库整库逻辑转储，后者是设备配置文本。命名前缀 sys_db_ 避免混淆。
type DBBackup struct {
	BaseModel
	FileName      string `gorm:"size:255;not null" json:"fileName"`
	FilePath      string `gorm:"size:1024;not null" json:"filePath"`
	StorageType   string `gorm:"size:20;not null;index" json:"storageType"`
	RemotePath    string `gorm:"size:1024" json:"remotePath,omitempty"`
	DBType        string `gorm:"size:20;not null" json:"dbType"`
	FileSize      int64  `json:"fileSize"`
	Sha256        string `gorm:"size:64" json:"sha256"`
	SchemaVersion string `gorm:"size:20" json:"schemaVersion"`
	TableCount    int    `json:"tableCount"`
	Source        string `gorm:"size:20;not null;index" json:"source"` // manual / cron / pre_restore
	Status        string `gorm:"size:20;not null;index" json:"status"` // ready / remote_only / deleted
	TriggeredBy   string `gorm:"size:64" json:"triggeredBy,omitempty"`
	Remark        string `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (DBBackup) TableName() string {
	return "sys_db_backup"
}

// DBBackupTask 全库备份/恢复异步任务（表 sys_db_backup_task）。
//
// 与 ConfigRestoreTask 同构：分离「备份版本链」与「任务运行时状态」，
// 不嵌 BaseModel（无软删除、无审计字段），四态状态机不支持取消。
const (
	DBBackupTaskPending = "pending"
	DBBackupTaskRunning = "running"
	DBBackupTaskSuccess = "success"
	DBBackupTaskFailed  = "failed"

	DBBackupTaskTypeExport  = "export"
	DBBackupTaskTypeRestore = "restore"
)

type DBBackupTask struct {
	ID           string     `gorm:"type:uuid;primary_key" json:"id"`
	TaskType     string     `gorm:"size:20;not null;index" json:"taskType"`
	BackupID     string     `gorm:"type:uuid;index" json:"backupId,omitempty"`
	Status       string     `gorm:"size:20;not null;index" json:"status"`
	Progress     int        `json:"progress"`
	Stage        string     `gorm:"size:64" json:"stage,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CreatedBy    string     `gorm:"size:64" json:"createdBy,omitempty"`
}

// BeforeCreate GORM 钩子：创建前生成 UUID 与时间戳
func (t *DBBackupTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子：更新前刷新 UpdatedAt
func (t *DBBackupTask) BeforeUpdate(tx *gorm.DB) error {
	t.UpdatedAt = time.Now()
	return nil
}

// TableName 设置表名
func (DBBackupTask) TableName() string {
	return "sys_db_backup_task"
}

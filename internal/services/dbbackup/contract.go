// Package dbbackup 提供数据库全库备份导出、导入恢复与存储适配能力。
//
// 与 internal/services/config_backup_service.go（网络设备 running-config 备份）
// 完全独立，不复用其表、模型与服务。
package dbbackup

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// StorageType 备份存储后端类型
type StorageType string

const (
	StorageLocal StorageType = "local"
	StorageOSS   StorageType = "oss"
	StorageFTP   StorageType = "ftp"
	StorageSMB   StorageType = "smb"
)

// 备份来源与状态常量
const (
	SourceManual     = "manual"
	SourceCron       = "cron"
	SourcePreRestore = "pre_restore"

	StatusReady      = "ready"
	StatusRemoteOnly = "remote_only"
	StatusDeleted    = "deleted"

	SchemaVersionV1 = "1"
)

// Plan 备份计划与存储配置（持久化到 sys_config）
type Plan struct {
	Enabled   bool        `json:"enabled"`
	Cron      string      `json:"cron"`
	Retention int         `json:"retention"`
	Storage   StorageType `json:"storage"`
	OSS       OSSConfig   `json:"oss"`
	FTP       FTPConfig   `json:"ftp"`
	SMB       SMBConfig   `json:"smb"`
}

// OSSConfig OSS / S3 兼容对象存储配置
type OSSConfig struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	UseSSL    bool   `json:"useSSL"`
}

// FTPConfig FTP 配置
type FTPConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	RemoteDir string `json:"remoteDir"`
}

// SMBConfig SMB 配置：Path 是部署机 OS 已挂载共享后的本地目录绝对路径，
// Go 端不实现 SMB 协议（避免引入停更库与 CGO 依赖）。
type SMBConfig struct {
	Path string `json:"path"`
}

// BackupRecord 备份记录视图
type BackupRecord struct {
	ID            string    `json:"id"`
	FileName      string    `json:"fileName"`
	FilePath      string    `json:"filePath"`
	StorageType   string    `json:"storageType"`
	RemotePath    string    `json:"remotePath,omitempty"`
	DBType        string    `json:"dbType"`
	FileSize      int64     `json:"fileSize"`
	Sha256        string    `json:"sha256"`
	SchemaVersion string    `json:"schemaVersion"`
	TableCount    int       `json:"tableCount"`
	Source        string    `json:"source"`
	Status        string    `json:"status"`
	TriggeredBy   string    `json:"triggeredBy,omitempty"`
	Remark        string    `json:"remark,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TaskView 异步任务状态视图
type TaskView struct {
	ID           string     `json:"id"`
	TaskType     string     `json:"taskType"`
	BackupID     string     `json:"backupId,omitempty"`
	Status       string     `json:"status"`
	Progress     int        `json:"progress"`
	Stage        string     `json:"stage,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// ListResult 分页结果
type ListResult struct {
	List  []BackupRecord `json:"list"`
	Total int64          `json:"total"`
}

// DatabaseConfig 备份/恢复所需的数据库连接信息（来自 core.Config.Database）
type DatabaseConfig struct {
	Type     string // postgres | sqlite
	Path     string // sqlite 文件路径
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	// PgDumpPath / PgRestorePath 为 pg_dump / pg_restore 的可执行文件路径。
	// 为空时由服务层按 sys_config 配置键 → PATH 查找顺序解析；部署机未安装
	// PostgreSQL 客户端时，可在 config.yaml 或参数管理页显式指定绝对路径。
	PgDumpPath    string
	PgRestorePath string
}

// Capabilities 描述当前部署环境下备份/恢复能力的可用性（供页面预检提示）。
type Capabilities struct {
	DBType             string `json:"dbType"`
	PgDumpAvailable    bool   `json:"pgDumpAvailable"`
	PgRestoreAvailable bool   `json:"pgRestoreAvailable"`
	PgDumpPath         string `json:"pgDumpPath"`
	PgRestorePath      string `json:"pgRestorePath"`
	Hint               string `json:"hint"`
}

// DumpExecutor 数据库转储执行器抽象。
// 生产实现调用外部 pg_dump/pg_restore 或 SQLite VACUUM INTO；
// 单测可注入 fake，不依赖真实数据库命令。
type DumpExecutor interface {
	// Dump 把整库导出到 target 文件，返回实际产物路径。
	Dump(ctx context.Context, target string, dbCfg DatabaseConfig) (string, error)
	// Restore 从 source 备份容器恢复整库。
	Restore(ctx context.Context, source string, dbCfg DatabaseConfig) error
	// TableNames 枚举物理库中的表（用于 manifest 与完整性校验）。
	TableNames(ctx context.Context, dbCfg DatabaseConfig) ([]string, error)
}

// Storage 远程存储适配器
type Storage interface {
	Type() StorageType
	// Put 上传本地文件，返回远程路径。
	Put(ctx context.Context, localPath, remoteName string) (string, error)
	// Get 下载远程文件到本地路径。
	Get(ctx context.Context, remotePath, localPath string) error
	// Delete 删除远程文件。
	Delete(ctx context.Context, remotePath string) error
}

// Service 全库备份服务接口
type Service interface {
	List(ctx context.Context, current, pageSize int) (*ListResult, error)
	Get(ctx context.Context, id string) (*BackupRecord, error)
	GetTask(ctx context.Context, id string) (*TaskView, error)
	Statistics(ctx context.Context) (map[string]interface{}, error)
	// Capabilities 返回当前部署环境的能力预检（pg_dump/pg_restore 是否可用）。
	Capabilities(ctx context.Context) (*Capabilities, error)
	GetPlan(ctx context.Context) (*Plan, error)
	SavePlan(ctx context.Context, p *Plan, operator string) error
	// TriggerExport 建 pending 任务并异步执行，立即返回 taskID。
	TriggerExport(ctx context.Context, source, operator string) (string, error)
	// TriggerRestore 上传文件已落临时路径，建任务并异步恢复。
	TriggerRestore(ctx context.Context, uploadPath, operator string) (string, error)
	Delete(ctx context.Context, id, operator string) error
	// DownloadPath 返回可下载的本地文件路径。
	DownloadPath(ctx context.Context, id string) (string, error)
	// RunScheduled 供 cron tick 调用，内部读取 sys_config 门控决定是否真正执行。
	RunScheduled(ctx context.Context) error
	// RecoverStaleTasks 启动时把残留 running/pending 任务收敛为 failed。
	RecoverStaleTasks(ctx context.Context) error
}

// NewService 构造服务。
// exec 为 nil 时使用默认生产 DumpExecutor；backupDir 为空时用 "data/backups"。
func NewService(db *gorm.DB, dbCfg DatabaseConfig, exec DumpExecutor, backupDir string) Service {
	return newService(db, dbCfg, exec, backupDir)
}

// NewStorage 按计划构造存储适配器；local/smb 均为本地文件系统路径。
func NewStorage(p *Plan) (Storage, error) {
	return newStorage(p)
}

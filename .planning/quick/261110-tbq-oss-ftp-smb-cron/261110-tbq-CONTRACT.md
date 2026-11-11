# 261110-tbq 接口契约（并行实现唯一对齐基准，锁定）

所有代理必须严格按本文的签名、路由、字段命名实现。改动契约须先通知 lead。

## 1. API（全局前缀 `/api/v1`，挂 system 组）

读组 `GET/POST` 走 `RequirePermissions([]string{SystemBackupList, SystemBackupExport, SystemBackupRestore})`；
写组为 `authorized` 的兄弟组，用 `strictUserRouteGroup`。

| Method | Path | 权限 | 说明 |
|---|---|---|---|
| POST | `/system/db-backup/list` | `system:backup:list` | 备份记录分页 |
| GET | `/system/db-backup/tasks/:id` | `system:backup:list` | 异步任务状态轮询 |
| POST | `/system/db-backup/statistics` | `system:backup:list` | 统计卡 |
| GET | `/system/db-backup/:id/download` | `system:backup:list` | 下载备份 zip（`c.File`，短请求） |
| POST | `/system/db-backup/export` | `system:backup:export` | 触发导出，返回 taskId |
| POST | `/system/db-backup/restore` | `system:backup:restore` | 上传 zip + 触发恢复，返回 taskId |
| POST | `/system/db-backup/:id/delete` | `system:backup:export` | 删除备份记录与文件 |
| GET | `/system/db-backup/plan` | `system:backup:list` | 读取备份计划 |
| POST | `/system/db-backup/plan` | `system:backup:export` | 保存备份计划（存储配置、cron、开关、保留份数） |

响应统一 `response.Success` / `response.Error`；下载端点用 `c.File`（二进制流，不套 response 包装）。
导出/恢复端点返回 `{"taskId":"..."}`，前端轮询 `/tasks/:id`。

## 2. sys_config 键（`ConfigTypeNo + IsSystemNo`，可在参数管理页编辑）

| 键 | 默认 | 说明 |
|---|---|---|
| `sys.database.backup.enabled` | `false` | 定期备份开关 |
| `sys.database.backup.cron` | `0 0 2 * * *` | 6 字段含秒，Asia/Shanghai |
| `sys.database.backup.retention` | `7` | 保留份数 |
| `sys.database.backup.storage` | `local` | `local` / `oss` / `ftp` / `smb` |
| `sys.database.backup.oss.endpoint` | `` | OSS/S3 endpoint |
| `sys.database.backup.oss.bucket` | `` | 桶名 |
| `sys.database.backup.oss.accessKey` | `` | 敏感，脱敏展示 |
| `sys.database.backup.oss.secretKey` | `` | 敏感，脱敏展示 |
| `sys.database.backup.oss.useSSL` | `true` | |
| `sys.database.backup.ftp.host` | `` | |
| `sys.database.backup.ftp.port` | `21` | |
| `sys.database.backup.ftp.user` | `` | |
| `sys.database.backup.ftp.password` | `` | 敏感 |
| `sys.database.backup.ftp.remoteDir` | `/` | |
| `sys.database.backup.smb.path` | `` | OS 已挂载的本地目录绝对路径 |

存储凭据绝不进 `manifest.json`、绝不进 operlog。

## 3. 模型（`internal/models/db_backup.go`）

```go
// 表 sys_db_backup
type DBBackup struct {
    BaseModel
    FileName     string        // 备份文件名
    FilePath     string        // 本地绝对路径
    StorageType  string        // local/oss/ftp/smb
    RemotePath   string        // 远程存储路径（本地时为空）
    DBType       string        // postgres/sqlite
    FileSize     int64
    Sha256       string
    SchemaVersion string
    TableCount   int
    Source       string        // manual/cron/pre_restore
    Status       string        // ready/remote_only/deleted
    TriggeredBy  string
    Remark       string
}
func (DBBackup) TableName() string { return "sys_db_backup" }

// 表 sys_db_backup_task（异步任务，四态状态机）
type DBBackupTask struct {
    ID           string `gorm:"type:uuid;primary_key"`
    TaskType     string // export/restore
    BackupID     string // 关联 DBBackup（restore 时为目标）
    Status       string // pending/running/success/failed
    Progress     int
    Stage        string
    ErrorMessage string
    StartedAt    *time.Time
    CompletedAt  *time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
    CreatedBy    string
}
func (DBBackupTask) TableName() string { return "sys_db_backup_task" }
```

任务状态常量：`DBBackupTaskPending/Running/Success/Failed`。
`DBBackupTask` 用自定义 `BeforeCreate`（同 `ConfigRestoreTask`），不嵌 `BaseModel`（无软删除）。

## 4. Service 接口（`internal/services/dbbackup/contract.go`，lead 创建，其他代理只读）

```go
package dbbackup

type StorageType string
const (
    StorageLocal StorageType = "local"
    StorageOSS   StorageType = "oss"
    StorageFTP   StorageType = "ftp"
    StorageSMB   StorageType = "smb"
)

type Plan struct {
    Enabled   bool          `json:"enabled"`
    Cron      string        `json:"cron"`
    Retention int           `json:"retention"`
    Storage   StorageType   `json:"storage"`
    OSS       OSSConfig     `json:"oss"`
    FTP       FTPConfig     `json:"ftp"`
    SMB       SMBConfig     `json:"smb"`
}

type OSSConfig struct { Endpoint, Bucket, AccessKey, SecretKey string; UseSSL bool }
type FTPConfig struct { Host, User, Password, RemoteDir string; Port int }
type SMBConfig struct { Path string }

type BackupRecord struct {
    ID, FileName, FilePath, StorageType, RemotePath, DBType string
    FileSize int64; Sha256, SchemaVersion, Source, Status, TriggeredBy, Remark string
    TableCount int; CreatedAt time.Time
}

type TaskView struct {
    ID, TaskType, BackupID, Status, Stage, ErrorMessage string
    Progress int; StartedAt, CompletedAt *time.Time; CreatedAt time.Time
}

type ListResult struct { List []BackupRecord `json:"list"`; Total int64 `json:"total"` }

type Service interface {
    List(ctx context.Context, current, pageSize int) (*ListResult, error)
    Get(ctx context.Context, id string) (*BackupRecord, error)
    GetTask(ctx context.Context, id string) (*TaskView, error)
    Statistics(ctx context.Context) (map[string]interface{}, error)
    GetPlan(ctx context.Context) (*Plan, error)
    SavePlan(ctx context.Context, p *Plan, operator string) error
    TriggerExport(ctx context.Context, source, operator string) (taskID string, err error)
    TriggerRestore(ctx context.Context, uploadPath, operator string) (taskID string, err error)
    Delete(ctx context.Context, id, operator string) error
    DownloadPath(ctx context.Context, id string) (string, error)
    RunScheduled(ctx context.Context) error   // 供 cron tick 调用，内部读 sys_config 门控
    RecoverStaleTasks(ctx context.Context) error // 启动收敛残留 running/pending
}

// 可注入的 dump 执行器（单测不需要真实 pg_dump）
type DumpExecutor interface {
    Dump(ctx context.Context, target string, dbCfg DatabaseConfig) (string, error) // 返回 dump 文件路径/命令产物
    Restore(ctx context.Context, source string, dbCfg DatabaseConfig) error
    TableNames(ctx context.Context, dbCfg DatabaseConfig) ([]string, error)
}

type DatabaseConfig struct {
    Type, Path, Host, User, Password, DBName, SSLMode string
    Port int
}

func NewService(db *gorm.DB, dbCfg DatabaseConfig, exec DumpExecutor, backupDir string) Service
```

`Storage` 适配器接口（同包）：
```go
type Storage interface {
    Type() StorageType
    Put(ctx context.Context, localPath, remoteName string) (remotePath string, err error)
    Get(ctx context.Context, remotePath, localPath string) error
    Delete(ctx context.Context, remotePath string) error
}
func NewStorage(p *Plan) (Storage, error)
```

## 5. 前端（`src/lib/dbBackupApi.ts`）

```ts
export const dbBackupApi = {
  list(params): Promise<PageResponse<BackupRecord>>;   // POST /system/db-backup/list
  statistics(): Promise<...>;                          // POST /system/db-backup/statistics
  task(id): Promise<TaskView>;                         // GET /system/db-backup/tasks/:id
  export(): Promise<{taskId:string}>;                  // POST /system/db-backup/export
  restore(file: File): Promise<{taskId:string}>;        // multipart POST /system/db-backup/restore
  remove(id): Promise<...>;                            // POST /system/db-backup/:id/delete
  download(id, fileName): void;                        // downloadFile(`/system/db-backup/${id}/download`, ...)
  getPlan(): Promise<Plan>;                            // GET /system/db-backup/plan
  savePlan(p: Plan): Promise<...>;                     // POST /system/db-backup/plan
};
```

页面 `src/pages/system/backup/index.tsx`：Tabs「备份列表 / 备份计划」；列表用 `useTableManager` + `StatCards`；计划表单用 `CronSelector`（6 字段，后端 `robfig/cron/v3` 用 `?` 兼容）与存储类型 Select。按钮权限：`system:backup:export`（立即备份/删除/保存计划）、`system:backup:restore`（恢复）。

## 6. 调度（`internal/scheduler/db_backup_tasks.go`）

- `RegisterDBBackupTasks(s *Scheduler, db *gorm.DB, dbCfg dbbackup.DatabaseConfig, svc dbbackup.Service)`
- 内部 `RegisterTask("db_backup", ...)`；写一行 `models.Job`（`InvokeTarget="db_backup"`，`JobGroup="SYSTEM"`，`CronExpression="0 0 2 * * *"` 固定 tick 语义由服务内部 `RunScheduled` 门控，实际周期由 `sys.database.backup.cron` 决定）。
- ⚠️ 不覆盖已存在的 `config_backup` handler；不新增 job 类型。
- 页面只读可见：写 `sys_job` 记录；只读性靠前端不暴露写按钮（sys_job 无内置只读字段）。

## 7. 迁移 240（幂等、非阻断）

`Migrate240DBBackup(db *gorm.DB) error` 做四件事：
1. 用固定 UUID 创建 C 菜单（component=`system/backup/index`, path=`backup`, parent=canonicalSystemRootID, order=12）+ 3 个 F 子按钮（固定 UUID，parent=C 菜单 id）。
2. `// F_TYPE_SEED_BEGIN ... // F_TYPE_SEED_END` 注释块内声明 3 个 F 权限字面量（供 check-permission-codes.mjs 识别）。
3. seed 13 个 `sys_config` 键（ConfigTypeNo/IsSystemNo，Count 幂等不覆盖）。
4. PG 分支建 `sys_db_backup` / `sys_db_backup_task` 表 + 索引（用 `isPostgreSQL` 守卫）；sqlite 分支由 AutoMigrate 建表。

接线：`database.go` PG 分支 `:1225` 后、sqlite 分支 `:1369` 后各加 `runAndRecord(240, "db_backup", ...)`；`registry.go:59` 后加 `{240,"db_backup"}`；`menu_catalog_seed.sql` 追加 4 行（新库）。

## 8. 操作日志

module 名「数据备份」。导出 `OperTypeExport(5)`、导入 `OperTypeImport(6)`、下载 `OperTypeDownload(18)`、上传 `OperTypeUpload(17)`、删除 `OperTypeDelete(3)`、计划保存 `OperTypeUpdate(2)`。
计划保存含凭据 → 用 `operlog.RecordWithBody`；>8192 字节用 `WithOperParam` 传已过滤小 JSON。
所有 `operlog.Record` 在 `response.Success` 之前；下载端点在 `c.File` 之前。

## 9. 加密排除

`configs/config.yaml`（及 `config.dev.yaml` / `config.prod.yaml` / `*.example.yaml`）的
`security.request_encryption.exclude_paths` 加 `/api/v1/system/db-backup/*`。
**不要**加 `operlog.exclude_paths`。

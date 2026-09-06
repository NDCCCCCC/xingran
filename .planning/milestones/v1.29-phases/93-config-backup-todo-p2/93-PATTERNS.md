# Phase 93 — PATTERNS.md

**Generated:** 2026-09-05（orchestrator 基于磁盘核验的一手侦察；pattern-mapper 两次未落盘后的替代产物）
**用途:** 每个待建/待改文件 → 最近似 analog + 关键模式摘录。行号以 2026-09-05 工作树为准。

## 文件映射总表

| # | 目标文件（新建/修改） | 角色 | 最近似 Analog | 数据流 |
|---|----------------------|------|--------------|--------|
| 1 | `internal/device/executor.go`（改：+RestoreConfig） | 设备下发 | 同文件 `GetConfig`(:267) + `ExecuteCustom`(:161) | service → executor → pool → device |
| 2 | `internal/services/config_backup_service.go`（改：三 TODO + Restore 异步化） | 业务服务 | 同文件 `CreateBackup`(:94) / `createNewAutoBackup`(:340) | handler → service → db/device |
| 3 | `internal/models/config_restore_task.go`（新） | 数据模型 | `internal/models/config_backup.go`（全文件 85 行） | gorm AutoMigrate + service CRUD |
| 4 | `internal/core/db/migrations/migration_211_create_config_restore_task.go`（新） | schema 迁移 | `migration_210_normalize_menu_parent_id.go` | 启动时自动执行 |
| 5 | `internal/core/db/database.go`（改：:511 附近 +1 行） | AutoMigrate 注册 | 同处 `&models.ConfigBackup{}` 行 | 启动装配 |
| 6 | `pkg/constants/timeouts.go`（改：+RestoreConfigTimeout） | 常量 | 同文件既有 `CommandExecTimeout` 等段 | 全局引用 |
| 7 | `internal/api/v1/network/backup_handler.go`（改：Restore :266-288 异步语义） | HTTP handler | 同文件 `BatchBackup`(:307+) | router → handler → service |
| 8 | `internal/api/v1/network/network_router.go`（改：/backups 组内 +任务查询路由） | 路由 | 同文件 backups 组(:150-170) | — |
| 9 | `internal/api/v1/network/backup_handler_test.go`（改：Restore 签名同步） | 测试 | 自文件既有用例 | — |
| 10 | `internal/services/config_backup_service_93_01_test.go` 等（新） | 回归测试 | `config_backup_service_79_06_test.go`（helpers :43-130） | — |
| 11 | `internal/services/portwrite/port_write_e2e_test.go`（只读参照） | e2e 模板 | SendConfigs fixture 回放模式（:3-60, :123, :616） | — |
| 12 | `xingran-react-frontend/src/pages/network/backups/`（改：恢复按钮异步交互） | 前端页面 | 同目录 hooks 模式 `useBackupModals.ts` | api wrapper → hook → 组件 |
| 13 | `xingran-react-frontend/src/lib/networkApi`（改：restoreBackup 响应 + 新任务查询 wrapper） | API 封装 | 既有 restoreBackup 函数 | — |
| 14 | `.planning/REQUIREMENTS.md` + `.planning/workstreams/milestone/ROADMAP.md`（改：措辞校准 D-01/D-33） | 规划文档 | Phase 92 措辞修订先例（commit 3a2efe5 模式） | — |

## 关键 Analog 摘录

### A. GetConfig 厂商 switch（executor.go:267-300，RestoreConfig 的对称模板）
```go
func (e *DeviceExecutor) GetConfig(ctx context.Context, deviceID string) (string, error) {
    pool := e.scheduler.GetConnectionPool()
    device, err := pool.GetDevice(deviceID)
    ...
    var configCommand string
    switch device.Vendor {
    case "huawei", "h3c":
        configCommand = "display current-configuration"
    case "ruijie", "maipu":
        configCommand = "show running-config"
    default:
        configCommand = "show running-config"
    }
```
→ RestoreConfig 同构：`GetDevice` → vendor→platform/退出命令映射 → `ExecuteCustom` 内 SendConfigs。

### B. ExecuteCustom 闭包执行器（executor.go:161-190，下发注入点）
```go
func (e *DeviceExecutor) ExecuteCustom(ctx context.Context, deviceID string,
    executeFunc func(context.Context, *PooledConnection) error, timeout time.Duration) error {
    task := &DeviceTask{ID: generateTaskID(), DeviceID: deviceID, Timeout: timeout,
        Execute: executeFunc,
        Callback: func(err error) { taskErr = err; close(done) }}
    if err := e.scheduler.Submit(task); err != nil { return err }
    waitCtx, cancel := context.WithTimeout(ctx, timeout+deviceExecutorTimeoutBuffer)
    select {
    case <-waitCtx.Done(): return fmt.Errorf("任务执行超时: taskID=%s", task.ID)
    case <-done: return taskErr
    }
}
```

### C. SendConfigs 调用与 per-line 失败日志（port_write_service.go:468-481）
```go
responses, sendErr := wrapper.SendConfigs(fullCmds)
...
applogger.Infof("[portwrite] SendConfigs resp[%d] portID=%s cmd=%q failed=%v result=%q", ...)
```
priv 注意（:409-420 注释）：SendConfigs 假设目标 priv=configuration；跨 ExecuteCustom 需自行确认 priv 状态。

### D. ConfigBackup model 三段式（models/config_backup.go）
```go
type ConfigBackup struct { ID string `gorm:"type:uuid;primary_key"` ... }
func (cb *ConfigBackup) BeforeCreate(tx *gorm.DB) error {
    if cb.ID == "" { cb.ID = uuid.New().String() } ...  // 23502 防御
}
func (ConfigBackup) TableName() string { return "sys_config_backup" }
```
→ ConfigRestoreTask 照此三段式 + string 状态常量（TaskStatusPending 等）。

### E. 79_06 测试 helpers（config_backup_service_79_06_test.go:43-130+）
- `newCbk7906(t)` — sqlite(AutoMigrate ConfigBackup/NetworkDevice/Config) + 进程级 temp dir + nil executor
- **`newExecutor7906(t, db, deviceID, fixtureCycles)`（:142-147）— 完整装配的 `*device.DeviceExecutor`（pool→scheduler→executor + FileTransport fixture 接线），93_NN 的 restore e2e 直接复用，无需自建装配**
- `cbk7906Chdir(t)` — **MkdirTemp + t.Chdir**（非 t.TempDir：getBackupDir 相对路径 + applogger ./logs/app.log Windows 占用，注释明确）
- `newDriver7906FromFixture(t, path)` — `platform.NewPlatform("huawei_vrp", "dummy-host", FileTransport, WithFileTransportFile(path), WithTransportReadSize(1), WithReadDelay(0))`
- `writeFixture7906(t, cycles, cmd, output)` — banner → screen-length → n×命令周期 → 8 spare prompts
- `SeedConnectionForTesting(pool, deviceID, conn)` — `internal/device/e2e_helpers.go:76`
- SendConfigs 阶段回放扩展 → 参照 `port_write_e2e_test.go`（命令回显写入 fixture，:123）

### F. Restore handler 现状（backup_handler.go:266-288，异步化改造点）
```go
func (h *BackupHandler) Restore(c *gin.Context) {
    id := c.Param("id")
    var req struct{ DeviceID string `json:"deviceId" binding:"required"` }
    if !responseHelpers.HandleJSONBinding(c, &req) { return }
    err := h.backupService.RestoreBackup(c.Request.Context(), id, req.DeviceID)
    if !responseHelpers.HandleServiceError(c, err, "恢复配置") { return }
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "配置备份", operlog.OperTypeUpdate)
    response.Success(c, gin.H{"message": "恢复成功"})
}
```
→ 改造：`RestoreBackup` 返回 `(taskID, error)`；成功响应 `gin.H{"taskId": taskID}`；operlog 保持原位（D-18）。

### G. 路由组权限（network_router.go:150-170）
```go
backups := r.Group("/backups")
backups.Use(middleware.RequirePermissions([]string{
    "network:backup:list", "network:backup:add",
    "network:backup:restore", "network:backup:diff",
}, core))
```
→ 任务查询路由加进本组（D-31 零新权限点）。

### H. 排序白名单现存 bug（config_backup_service.go:463-469，D-33 修复对象）
```go
var configBackupAllowedSortFields = map[string]string{
    "deviceId": "device_id", "version": "version",
    "status": "status",      // ← 列不存在，ORDER BY 必 SQL 500
    "createdAt": "createdAt 映射 created_at",
}
```
→ 删除 `"status"` 行（独立原子 commit）。

## 集成点（改动的接线位置）

- `internal/core/core.go` — ConfigBackupService 装配点；Task service 若独立需在此接线（exec db + executor 已就绪）
- `internal/core/db/database.go:511` — AutoMigrate 模型清单 +1
- `pkg/constants/timeouts.go` — RestoreConfigTimeout 落点（Phase 90 惯例）
- `xingran-react-frontend/src/pages/network/backups/hooks/useBackupModals.ts` — 恢复 Modal 状态机扩展点

# Phase 57: AD 推送队列基建 (outbox-lite 同事务入队) - Pattern Map

**Mapped:** 2026-10-16
**Files analyzed:** 6 new + 3 modified + 2 test files (11 total)
**Analogs found:** 11 / 11 (all paths have codebase precedent)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/ad_push_queue.go` | model | n/a (schema) | `internal/models/config_restore_task.go` | exact |
| `internal/core/db/migrations/migration_215_create_ad_push_queue.go` | migration | schema DDL | `migration_211_create_config_restore_task.go` + `migration_212_create_restore_task_active_unique_index.go` | exact |
| `internal/services/addomain/ad_push_queue_service.go` | service | CRUD + same-tx write | new (no exact analog; modelled on `user_ad_sync_service.go` + handler `buildADSyncMap`) | role-match |
| `internal/services/system/user_service.go` (modify) | service | CRUD (modify — 3 tx blocks) | self (existing Update/Delete/BatchDelete) | exact |
| `internal/core/db/database.go` (modify) | config | registration list | self (lines 651, 1034, 1073) | exact |
| `internal/services/system/ad_push_queue_test.go` (new) | test | sqlite + migration | `internal/services/asset/fix_suggestion_service_test.go:36-77` + `config_backup_service_93_02_test.go:402` | exact |

## Pattern Assignments

### `internal/models/ad_push_queue.go` (model, queue table)

**Analog:** `internal/models/config_restore_task.go` (full file)

**Imports pattern** (lines 3-7):
```go
package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)
```

**Status enum pattern** (lines 11-17):
```go
// RestoreTaskStatus 恢复任务状态（string 枚举,跟随 ConfigBackup 的
// BackupType/StorageType 风格；四态状态机,Phase 93 D-34——不支持取消）。
const (
    RestoreTaskStatusPending RestoreTaskStatus = "pending"
    RestoreTaskStatusRunning RestoreTaskStatus = "running"
    RestoreTaskStatusSuccess RestoreTaskStatus = "success"
    RestoreTaskStatusFailed  RestoreTaskStatus = "failed"
)

// RestoreTaskStatus 恢复任务状态类型
type RestoreTaskStatus string
```

**Struct pattern — no DeletedAt, BeforeCreate UUID, *time.Time terminal state** (lines 25-67):
```go
// ConfigRestoreTask 配置恢复任务（Phase 93 D-15：异步恢复的生命周期载体,
// 与备份版本链 sys_config_backup 职责分离——任务运行时状态在此,恢复成功
// 事件以版本链记录落 sys_config_backup）。
type ConfigRestoreTask struct {
    ID           string     `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID     string     `gorm:"type:uuid;not null;index" json:"deviceId"`
    BackupID     string     `gorm:"type:uuid;not null" json:"backupId"`
    Status       string     `gorm:"size:20;not null;index" json:"status"`
    // ... no DeletedAt; *time.Time 可空终态列
    StartedAt    *time.Time `json:"startedAt,omitempty"`
    CompletedAt  *time.Time `json:"completedAt,omitempty"`
    CreatedAt    time.Time  `json:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt"`
    CreatedBy    string     `gorm:"size:64" json:"createdBy,omitempty"`
}

// BeforeCreate GORM 钩子：创建前生成 UUID 与时间戳（BaseModel 无 DB default,
// PG 下 uuid 列无 default 会 23502——23502 防御惯例）。
func (c *ConfigRestoreTask) BeforeCreate(tx *gorm.DB) error {
    if c.ID == "" {
        c.ID = uuid.New().String()
    }
    now := time.Now()
    if c.CreatedAt.IsZero() {
        c.CreatedAt = now
    }
    if c.UpdatedAt.IsZero() {
        c.UpdatedAt = now
    }
    return nil
}

// BeforeUpdate GORM 钩子：更新前刷新 UpdatedAt
func (c *ConfigRestoreTask) BeforeUpdate(tx *gorm.DB) error {
    c.UpdatedAt = time.Now()
    return nil
}

// TableName 设置表名
func (ConfigRestoreTask) TableName() string {
    return "sys_config_restore_task"
}
```

**Key constraints (D-08):**
- 不软删（无 DeletedAt）
- 不在模型 tag 声明唯一索引（避免 GORM/PG 命名冲突 W-1；唯一索引由手写 migration_215 显式 `uq_ad_push_queue_active` 命名）
- 终态用 `*time.Time` 可空列
- UUID BeforeCreate 钩子防 23502

---

### `internal/core/db/migrations/migration_215_create_ad_push_queue.go` (migration, dual-dialect DDL)

**Analog:** `migration_211_create_config_restore_task.go` (CREATE TABLE IF NOT EXISTS) + `migration_212_create_restore_task_active_unique_index.go` (partial unique index) + `migration_214_create_ad_login_history.go` (isPostgreSQL guard)

**Imports pattern** (migration_212 lines 3-8):
```go
package migrations

import (
    "log"

    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "gorm.io/gorm"
)
```

**CREATE TABLE IF NOT EXISTS pattern** (migration_211 lines 16-47):
```go
// Migrate211CreateConfigRestoreTask 创建 sys_config_restore_task 配置恢复任务表
// (Phase 93 D-15: 异步恢复任务的生命周期载体,与备份版本链职责分离)。
//
// 双方言: 仅 PG 分支执行 (CREATE TABLE IF NOT EXISTS,幂等);sqlite 分支由
// MigrateModelList 的 AutoMigrate 建表,调用方无需在 sqlite 分支注册本迁移。
// 索引: device_id (同设备互斥查询 D-08) 与 status (状态机流转查询)。
func Migrate211CreateConfigRestoreTask(db *gorm.DB) error {
    log.Println("Running migration 211: create sys_config_restore_task")

    const ddl = `CREATE TABLE IF NOT EXISTS sys_config_restore_task (
        id uuid PRIMARY KEY,
        device_id uuid NOT NULL,
        backup_id uuid NOT NULL,
        status varchar(20) NOT NULL,
        ...
        created_at timestamptz,
        updated_at timestamptz,
        created_by varchar(64)
    )`
    if err := db.Exec(ddl).Error; err != nil {
        applogger.Errorf("migration 211: create sys_config_restore_task 失败: %v", err)
        return err
    }
    // 注：多个 CREATE INDEX IF NOT EXISTS ...
    return nil
}
```

**Partial unique index pattern** (migration_212 lines 22-33, key analog for D-06):
```go
// Migrate212CreateRestoreTaskActiveUniqueIndex 为 sys_config_restore_task 建立
// 同设备活跃任务部分唯一索引 (v129-recheck C-1: StartRestore 互斥 D-08 原子化)。
//
// ...
//
// 双方言: PG 与 sqlite (3.8+) 均支持 partial unique index,语法一致,双方言
// 分支均注册 (幂等 IF NOT EXISTS)。
//
// 脏数据兼容: 若历史数据已存在同设备双活跃任务 (正是本索引要消灭的竞态产物),
// 建索引失败按项目惯例非阻断 (留待人工清理后下次启动重试)。
func Migrate212CreateRestoreTaskActiveUniqueIndex(db *gorm.DB) error {
    log.Println("Running migration 212: create sys_config_restore_task active-unique partial index")

    const ddl = `CREATE UNIQUE INDEX IF NOT EXISTS uq_sys_config_restore_task_device_active
ON sys_config_restore_task (device_id)
WHERE status IN ('pending', 'running')`
    if err := db.Exec(ddl).Error; err != nil {
        applogger.Errorf("migration 212: create active-unique partial index 失败: %v", err)
        return err
    }
    return nil
}
```

**Mixed approach — use migration_212 (both dialects, no guard) since queue needs both PG + sqlite to be functional:**
- `migration_199` uses `isPostgreSQL` guard for PG-only constraints
- `migration_212` registers in **both** PG (advisory-lock block) and sqlite (else block) because partial unique index is syntax-compatible across `modernc v1.40.1` and PG
- For ad_push_queue: the model is added to `MigrateModelList` so GORM AutoMigrate handles sqlite in-memory schema, but real migration_215 must register in **both branches** (consult database.go lines 1034-1051 for PG and 1073-1079 for sqlite to decide where to call from)

**Naming convention:**
- Use `uq_ad_push_queue_active` prefix (PG inline UNIQUE/SQL partial unique)
- `idx_ad_push_queue_status_retry` and `idx_ad_push_queue_entity_id` for queue scan indexes (D-07 一次建齐)

---

### `internal/services/addomain/ad_push_queue_service.go` (service, outbox-lite enqueue)

**Analog:** `internal/services/addomain/user_ad_sync_service.go` (package location + DI style) + `internal/api/v1/system/user_handler.go:245-260` (buildADSyncMap — four-field contract)

**Pre-condition checker pattern — read `sys_ad_config.sync_enabled AND status=enabled`** (analog: `internal/services/addomain/user_ad_sync_service.go:252-253`):

Per CONTEXT.md D-02: 入队前查询启用配置 → 不入队 + WARN 放行。
```go
// DefaultADSyncEnabledChecker 默认实现：查 sys_ad_config 启用行（D-02 第一层前置条件）。
func DefaultADSyncEnabledChecker(tx *gorm.DB) (bool, error) {
    var count int64
    if err := tx.Model(&models.ADConfig{}).
        Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).
        Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}
```
Reference: `internal/models/ad_domain.go:13-19` declares `ADConfigStatusEnabled ADConfigStatus = 0` (int status — but for migration_215 partial-index we use string literals; service-side check uses the int constant).

**Four-field diff contract (D-01)** — modeled directly on `user_handler.go:245-260 buildADSyncMap`:
```go
// buildADSyncMap 将UserUpdateRequest转换为AD同步用的map
func (h *UserHandler) buildADSyncMap(req *requests.UserUpdateRequest) map[string]interface{} {
    m := make(map[string]interface{})
    if req.Nickname != nil {
        m["nickname"] = *req.Nickname
    }
    if req.Email != nil {
        m["email"] = *req.Email
    }
    if req.Phone != nil {
        m["phone"] = *req.Phone
    }
    if req.DeptID != nil {
        m["deptId"] = *req.DeptID
    }
    return m
}
```
Replicate the same nil-pointer semantics in `hasADFieldChanged`: req field nil = "no change for this field, skip".

**DisableUser reuse (QUEUE-06 execution side)** — analog: `internal/services/addomain/ldap_client.go:337-340`:
```go
// DisableUser 禁用用户
func (c *LDAPClient) DisableUser(userDN string) error {
    return c.setUserAccountControl(userDN, uacDisabledAccount)
}
```
Phase 58 drainer will call this; Phase 57 only enqueues.

**Same-tx enqueue insertion (D-02 second layer)** — pattern from `user_service.go:157` existing transaction block. The enqueue function must receive `tx *gorm.DB` (not `s.db`) so INSERT is bound to the outer business transaction. Failure → `return err` so the outer `tx.Commit()` is skipped.

**ON CONFLICT partial unique conflict target** — text-level required (Pitfall 1):
```sql
ON CONFLICT (entity_type, entity_id, action_type)
WHERE status IN ('pending','failed')  -- 必须与部分索引谓词完全一致
DO UPDATE SET attempts = 0, last_error = '', failed_at = NULL, ...
```
GORM way: `tx.Clauses(clause.OnConflict{...})` — see `gorm.io/gorm/clause`. The `Where` clause goes on the OnConflict struct (literal `ad_push_queue.status IN ('pending','failed')` — `clause.Literal` form, as research RESEACH.md Code Example 3 demonstrates).

**BatchDisableCreateInBatches — pattern from `internal/services/system/user_sync_service.go:602 batchWriteSize=500`**:
```go
if err := tx.CreateInBatches(&rows, 500).Error; err != nil {
    return err
}
```

---

### `internal/services/system/user_service.go` (modify, three same-tx enqueue sites)

**Analog:** self (lines 157, 240, 575)

**Update transaction block** (lines 157-219):
```go
// 使用事务更新用户及其关联
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 更新用户基本信息
    user.Nickname = req.Nickname
    // ...
    if err := tx.Save(&user).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 更新角色关联：先删除旧关联,再添加新关联
    if err := tx.Table("sys_user_role").Where("user_id = ?", req.ID).Delete(&models.UserRole{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }
    // ... userRoles insert ...
    // ... userPosts update ...

    return nil  // <-- Phase 57 在此处之前插入 EnqueueUpdateIfChanged(tx, &user, req)
})
```

**Delete transaction block** (lines 240-257):
```go
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 删除用户角色关联
    if err := tx.Table("sys_user_role").Where("user_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 删除用户岗位关联
    if err := tx.Table("sys_user_post").Where("user_id = ?", id).Delete(&models.UserPost{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 软删除用户
    if err := tx.Delete(&user).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    return nil  // <-- Phase 57 在此处之前插入 EnqueueDisable(tx, id)
})
```

**BatchDelete transaction block** (lines 575-592):
```go
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 批量删除用户角色关联
    if err := tx.Table("sys_user_role").Where("user_id IN ?", ids).Delete(&models.UserRole{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 批量删除用户岗位关联
    if err := tx.Table("sys_user_post").Where("user_id IN ?", ids).Delete(&models.UserPost{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 批量软删除用户
    if err := tx.Where("id IN ?", ids).Delete(&models.User{}).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    return nil  // <-- Phase 57 在此处之前插入 EnqueueDisableBatch(tx, ids)
})
```

**Enqueuer location / construction:** The `addomain` package is the natural home (CONTEXT.md Claude's Discretion favors `addomain`; dependency direction system → addomain is acyclic). Construct via `addomain.NewADPushEnqueuer(s.db)` inside the tx block, OR inject via `UserService` constructor (preferred, but for Phase 57 unblock a direct `addomain.NewADPushEnqueuer(s.db)` call is acceptable scoped minimal change).

**Insertion point rule (D-02 second layer):** the queue INSERT must be inside the outer `Transaction(func(tx *gorm.DB) error { ... })` callback body, before `return nil`. Returning the enqueue error from the tx callback propagates failure to outer — causes `tx.Commit()` to be skipped → both sys_user update AND ad_push_queue row are rolled back (D-02 explicit).

---

### `internal/core/db/database.go` (modify, register migration + add model)

**Analog:** self, registration at lines 1034-1051 (PG advisory-lock branch) and 1073-1079 (sqlite else branch), plus `MigrateModelList` line 671 for `&models.ConfigRestoreTask{}`.

**Model registration** (database.go:651-753, MigrateModelList):
Add `&models.ADPushQueue{},` near line 671 (where `&models.ConfigRestoreTask{}` is registered). Note: per `migration_214` comment "对应模型禁止加入 AutoMigrate 列表 — GORM 对分区父表做 ALTER 会触发问题", ad_push_queue is NOT a partitioned table, so it can be added to AutoMigrate safely.

**Migration registration — PG branch** (database.go:1034-1050, advisory-lock block):
```go
// quick 261016-ad-login-history: AD 设备登录历史月分区表 (PG-only,幂等;
// sqlite 不建表,writer/query 侧由 addomain 包 isPostgres 守卫跳过。
// 对应模型禁止加入 AutoMigrate 列表 — GORM 对分区父表
// 做 ALTER 会触发 42P16 类问题)
if err := migrations.Migrate214CreateADLoginHistory(d.DB); err != nil {
    applogger.Errorf("AD登录历史分区表创建失败 (非阻断,留待下次启动): %v", err)
}
```
Add a similar block right after for `Migrate215CreateADPushQueue`. **Decision required:** does migration_215 register in both branches (mirror migration_212 style — both branches call it because partial unique index works in sqlite), or only PG (migration_214 style — sqlite skipped, AutoMigrate handles sqlite via the model)?

For ad_push_queue: **both branches** is the right pattern, because the partial unique index DDL is sqlite-compatible (modernc v1.40.1 supports `CREATE UNIQUE INDEX ... WHERE`). The sqlite test must call `migrations.Migrate215CreateADPushQueue` (HYGIENE-06 mirror config_backup_service_93_02_test.go:402), but production startup also needs to run it (sqlite is not just dev — Phase 19+ has sqlite as alternate config).

**Migration registration — sqlite branch** (database.go:1052-1087):
The else block already calls `Migrate212CreateRestoreTaskActiveUniqueIndex(d.DB)` (line 1073) — proving partial-index migrations should run on both dialects. Mirror this with `Migrate215CreateADPushQueue(d.DB)` in the sqlite branch.

---

### `internal/services/system/ad_push_queue_test.go` (test, sqlite + migration)

**Analog:** `internal/services/asset/fix_suggestion_service_test.go:36-77` (inline CREATE TABLE DDL) + `internal/services/config_backup_service_93_02_test.go:402` (direct migration function call)

**setupUserServiceTestDB — extend with ad_push_queue schema** (user_service_test.go:46-94):
```go
// setupUserServiceTestDB 创建含最小 schema 的内存 sqlite。
func setupUserServiceTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    require.NoError(t, db.Exec(`CREATE TABLE sys_user ...`).Error)
    // ...existing tables...
    // Phase 57: 新增 sys_ad_config (前置条件 D-02) + ad_push_queue
    require.NoError(t, db.Exec(`
        CREATE TABLE sys_ad_config (
            id TEXT PRIMARY KEY,
            sync_enabled INTEGER NOT NULL DEFAULT 0,
            status INTEGER NOT NULL DEFAULT 0,
            deleted_at DATETIME
        )
    `).Error)
    require.NoError(t, db.Exec(`CREATE TABLE ad_push_queue (...)`).Error)
    return db
}
```

**Direct migration function call — alternative path** (config_backup_service_93_02_test.go:400-430):
```go
func TestCbk93RestoreActiveUniqueIndex(t *testing.T) {
    db := newCbk93RestoreDB(t)
    require.NoError(t, migrations.Migrate212CreateRestoreTaskActiveUniqueIndex(db))
    // ... exercises partial unique index directly ...
}
```
For Phase 57, this is the cleanest test pattern for the partial unique index — call `migrations.Migrate215CreateADPushQueue(db)` directly, then assert insert/upsert semantics.

**Seed sys_ad_config enabled row (D-02 pre-condition)** — add helper:
```go
// seedADConfigEnabled 插入启用行(D-02 前置条件满足)
require.NoError(t, db.Exec(
    `INSERT INTO sys_ad_config (id, sync_enabled, status, created_at, updated_at, deleted_at)
     VALUES ('test-config', 1, 0, ?, ?, NULL)`,
    "2024-01-01 00:00:00", "2024-01-01 00:00:00",
).Error)
```

**Test cases (HYGIENE-06 + QUEUE-01/02/06):**
- `TestPhase57_ADPushQueue_UpdateEnqueue_DiffFields` — Update diff 入队（nickname/email/phone/deptId 真实变更）
- `TestPhase57_ADPushQueue_UpdateEnqueue_NilFieldsSkipped` — Update nil 指针跳过
- `TestPhase57_ADPushQueue_UpdateEnqueue_MergeActiveRow` — QUEUE-02 同 entity 二次编辑 → 唯一索引夺锁合并
- `TestPhase57_ADPushQueue_UpdateEnqueue_RoleChangesIgnored` — Update 角色/岗位变更不入队
- `TestPhase57_ADPushQueue_UpdateEnqueue_NoADConfigNoEnqueue` — D-02 前置条件：不入队 + WARN
- `TestPhase57_ADPushQueue_DisableEnqueue_SingleUser` — Delete disable 入队
- `TestPhase57_ADPushQueue_DisableEnqueue_Batch` — BatchDelete disable 入队（CreateInBatches）

---

## Shared Patterns

### Database Migration isPostgreSQL Guard
**Source:** `internal/core/db/migrations/migration_helpers.go:7-9`
**Apply to:** migrations needing PG-only features (e.g., partitioning, exotic types). For Phase 57, partial unique index works on both dialects → **don't** use this guard; mirror migration_212 dual-branch instead.
```go
package migrations

import "gorm.io/gorm"

// isPostgreSQL 判断当前数据库方言是否为 PostgreSQL。
// 历史:原定义于 migration_mac_history.go(已删除);多个 migration 用作 SQLite 路径守卫。
func isPostgreSQL(db *gorm.DB) bool {
    return db.Config.Dialector.Name() == "postgres"
}
```

### Migration Logging Convention
**Source:** `migration_211_create_config_restore_task.go:17` and database.go failure path
**Apply to:** all migration_215 functions
```go
log.Println("Running migration 215: create ad_push_queue + active-unique partial index")
// ...
applogger.Errorf("migration 215: create ad_push_queue 失败: %v", err)
```
Database.go invokes migrations non-blocking:
```go
if err := migrations.Migrate215CreateADPushQueue(d.DB); err != nil {
    applogger.Errorf("ad_push_queue 迁移失败 (非阻断,留待下次启动): %v", err)
}
```

### ON CONFLICT Text-Level Predicate Matching
**Source:** `migration_212_create_restore_task_active_unique_index.go:25-27`
**Apply to:** GORM `clause.OnConflict` writes in `ad_push_queue_service.go` (Pitfall 1)
```sql
WHERE status IN ('pending', 'running')  -- 必须与部分索引 DDL 谓词完全一致(双方言均要求)
```
GORM literal form: `clause.Literal{SQL: "ad_push_queue.status IN ('pending','failed')"}` — pass to `clause.OnConflict.Where.Exprs`.

### Fail-Soft Migration Startup Pattern
**Source:** `internal/core/db/database.go` (lines 985-1051, advisory-lock block)
**Apply to:** all new migrations added in Phase 57
- `applogger.Errorf("...失败 (非阻断,留待下次启动)")` — never return error to caller at startup
- The migration re-runs on next restart (idempotent via `IF NOT EXISTS`)

### operlog Convention
**Source:** `CLAUDE.md` (mandatory) + `internal/api/v1/system/ad_domain_handler.go`
**Apply to:** None — Phase 57 doesn't touch handler layer operlog; user_service.Update/Delete/BatchDelete existing paths already have handler-layer operlog.Record (kept by D-03). No service-layer operlog calls.

### Default Status Values
**Source:** `internal/models/ad_domain.go:13-19`
**Apply to:** D-02 pre-condition check in service
```go
const (
    ADConfigStatusEnabled  ADConfigStatus = 0 // 启用
    ADConfigStatusDisabled ADConfigStatus = 1 // 停用
)
```
D-02 check: `sync_enabled = ? AND status = ?` with `(true, ADConfigStatusEnabled=0)`.

### Cache Key Prefix Convention (n/a for Phase 57)
Phase 57 doesn't touch cache; queue table has no Redis keys.

## No Analog Found

None — all 11 paths have a documented existing precedent in the codebase.

## Metadata

**Analog search scope:** `internal/models/`, `internal/core/db/migrations/`, `internal/services/system/`, `internal/services/addomain/`, `internal/services/asset/`, `internal/api/v1/system/`
**Files scanned:** 11 analog files (5 read in full + 4 read partially + 2 test precedents)
**Pattern extraction date:** 2026-10-16

**Key design decisions sourced from analogs:**
1. `ConfigRestoreTask` (no DeletedAt, BeforeCreate UUID, `*time.Time` terminal state) → `ADPushQueue` model
2. `migration_211` (CREATE TABLE IF NOT EXISTS) + `migration_212` (partial unique index via `WHERE status IN (...)`) → `migration_215` (combined)
3. `user_handler.go:245-260 buildADSyncMap` (4-field + nil-pointer contract) → `hasADFieldChanged` diff helper
4. `ldap_client.go:337 DisableUser` (reuse) → QUEUE-06 drainer side (Phase 58; Phase 57 only enqueues)
5. `database.go:1034-1051 (PG) + 1073-1079 (sqlite)` dual-branch registration → migration_215 in both branches
6. `config_backup_service_93_02_test.go:402` direct migration call → sqlite test pattern for partial unique index verification

# Phase 57: AD 推送队列基建（outbox-lite 同事务入队） - Research

**Researched:** 2026-10-16
**Domain:** Backend Go/GORM outbox-lite pattern + 同事务入队 + 部分唯一索引 dedupe + sqlite 测试环境补建表
**Confidence:** HIGH（路径已 lock，所有改动点与先例都已读取源码）

## Summary

Phase 57 是 v1.21 AD 双向同步修复里程碑的第一个 phase，要在 `user_service.go` 三条写路径（`Update` / `Delete` / `BatchDelete`）的事务内插入一张新的队列表 `ad_push_queue`，解决三类问题：(1) 进程重启丢失 AD 推送；(2) handler 旧 `adSyncInFlight` dedupe 丢弃了有效编辑；(3) sys 删除用户后 AD 端"复活"链路无 disable 收尾。本 phase **不**包含排水器、不修登录守卫、不动对账视图、不删 handler 旧推送路径——排水器与旧路径删除均属 Phase 58。

落地物是：一张 PG/sqlite 双方言迁移（migration_215）+ 一个新 GORM 模型 `models.ADPushQueue`（无 `DeletedAt`、BeforeCreate UUID、可空终态时间列、PG-only 部分唯一索引 `WHERE status IN ('pending','failed')` 双方言注册）+ 三个入队纯函数 helper（diff / enqueueUpdate / enqueueDelete）+ `user_service.go` 三处事务内的入队调用点（前置条件：未启用 AD 配置不入队；队列 INSERT 真失败 → 同事务回滚）。所有 claim 已在源码核验，唯一 `[ASSUMED]` 项是 entity_type/action_type 字符串常量取值的命名风格——D-04 已明示"动作枚举预留扩展位"，可"选取 `'user'` / `'disable'` 字符串型"由 planner 定夺。

**Primary recommendation:** 三个文件改 + 一个新建：新建 `migration_215_create_ad_push_queue.go` + `models/ad_push_queue.go` + 队列 service 文件 + 修改 `user_service.go`；handler 旧路径与排水器均不在本 phase。

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: diff 入队** — 对比事务前已加载旧值（`user_service.go:148-154` 的 `First` 即旧值），nickname/email/phone/deptId 四字段实际发生变更才入队；纯角色/岗位/备注/状态编辑不产生队列行。nil 指针语义：req 字段 nil = 无变更，跳过该字段。diff 逻辑做成纯函数 helper，可单测覆盖 nil/空串边界。
- **D-02: 降级语义（两层）**
  - **前置条件**：入队前查 `sys_ad_config` 无 `sync_enabled=true AND status=enabled` 行 → 不入队 + WARN 日志放行用户编辑。否则"AD 未部署"环境积累永 pending 行，污染 Phase 59 GUARD-01 信号守卫。
  - **队列 INSERT 真失败** → 同事务回滚，用户编辑整体失败（"跳过+WARN"会静默架空 QUEUE-01）。
- **D-03: handler 旧推送路径（`user_handler.go:194-238`，含 `adSyncInFlight` dedupe、`buildADSyncMap`、goroutine + 3 次重试）保留至 Phase 58**。理由：57→58 存在真实部署间隙，保留使间隙期行为=现状，无推送真空；58 排水靠 compare-and-move 幂等消化双推积压。
  - ⚠ **Roadmap 修正项：handler 旧路径删除必须写进 Phase 58 范围**——当前 roadmap 无任何 phase 调度此删除点。
- **D-04: `Delete`（user_service.go:228）与 `BatchDelete`（:569）两条软删事务路径都在事务内入队 `disable AD 账号` 动作行**。动作枚举预留扩展位（disable 之外未来可加 status 类动作），但 **UpdateStatus 双向推送明确排除在 v1.21 外**。
  - disable 复用现成 `ldap_client.go:337 DisableUser`（幂等，UAC 514 Replace 可重放）；**禁止 delete AD 账号**——与 Phase 60 SAFE-01 ad_dn 校验恢复不兼容。
- **D-05: 载荷 = 最小动作行**（entity_type / entity_id / action_type），**禁用 JSON payload 快照**——快照过期正是已判死刑的时间戳仲裁土壤；compare-and-move 下排水闭包重读 DB 后全量推送是幂等超集。
- **D-06: 合并键 = 部分唯一索引 `UNIQUE(entity_type, entity_id, action_type) WHERE status IN ('pending','failed')`**。upsert 覆盖即 QUEUE-02 合并；终态行（pushed/dead）退出唯一索引、留表可查。先例：`migration_199`、`migration_212`（双方言部分唯一索引）。合并必须走唯一索引夺锁，禁用"先查后写"应用层合并。
- **D-07: Phase 58 退避列一次建齐**（attempts / next_retry_at / last_error / failed_at，nullable 不阻断），让 (status, next_retry_at) 拾取索引与唯一索引谓词一次设计到位。
- **D-08: 表杂项**
  - 队列表**不软删**（无 DeletedAt；ConfigRestoreTask 先例：扁平结构 + BeforeCreate 生成 UUID 防御 23502）
  - 终态用 *time.Time 可空列
  - 模型 tag **不得声明该唯一索引**（避免 GORM/PG 约束命名冲突 W-1 教训），只能手写 migration_215 显式 `uni_` 命名
  - sqlite 测试建表直调迁移函数（先例 `config_backup_service_93_02_test.go:402`）或复用 fix_suggestion sqlite 测试基建

### Claude's Discretion

- diff helper 具体签名/命名
- 队列 service 包位置（addomain vs system，planner 结合依赖方向定）
- WARN 日志格式
- entity_type 常量取值（如 "user"）
- sqlite 测试建表方式二选一（直调迁移 vs 复制建表 SQL）
- action_type 枚举具体字符串值

### Deferred Ideas (OUT OF SCOPE)

- **UpdateStatus 双向推送**（停用→disable / 启用→enable）— v1.21 范围外
- **auth.go AD 登录路径不查 user.Status**（停用用户仍可 AD bind 登录拿 JWT）— 独立 bug，候选后续 phase
- **BatchDelete 之外的批量路径排查**（如 Excel 删除类入口若有）— Phase 57 只覆盖已确认的两条软删事务路径

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| QUEUE-01 | 管理员编辑用户（部门/昵称/邮箱/手机）后，变更在业务事务内写入 `ad_push_queue`，进程重启不丢失 | D-01 diff 入队 + D-06 部分唯一索引 + user_service Update 事务块（:157）；INSERT 失败同事务回滚（D-02） |
| QUEUE-02 | 同一用户短时间内多次编辑时，队列按实体合并仅保留最新待推送状态，不丢弃后一次编辑 | D-06 部分唯一索引 `WHERE status IN ('pending','failed')` + ON CONFLICT 覆盖合并；显式禁用应用层"先查后写"合并 |
| QUEUE-06 | sys 删除用户时入队 disable AD 账号动作，闭环"已删用户经 AD 登录复活"链路 | D-04 Delete + BatchDelete 两条事务路径同事务入队 disable；复用 `ldap_client.go:337 DisableUser`（UAC 514 Replace 幂等） |
| HYGIENE-06 | sqlite 测试补队列建表，测试套件全绿 | D-08 sqlite 测试建表直调 migration_215（先例 config_backup_service_93_02_test.go:402）+ ConfigRestoreTask 无软删先例 |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|------------|----------------|-----------|
| `ad_push_queue` 表 schema（migration_215） | Database / Storage | — | 队列表是 Phase 57 唯一新增 schema 对象；落 `internal/core/db/migrations/migration_215_create_ad_push_queue.go` |
| `models.ADPushQueue` GORM 模型 | Backend / Service | — | 入队 helper 接收；Phase 58 排水器读取；Phase 59 守卫读取 |
| 队列 service 包（enqueueUpdate / enqueueDelete / enqueueDiffAndInsert） | Backend / Service | — | 承接 D-01/D-04 入队逻辑；调用点必须走 user_service 三处事务；封装前置条件检查（D-02） |
| diff helper（纯函数） | Backend / Service | — | D-01 diff 入队的核心；单测可覆盖 nil/空串边界；服务层纯函数无副作用 |
| 入队调用点 | Backend / Service | — | user_service Update:157/Delete:240/BatchDelete:575 事务块内 |
| Partial unique index `WHERE status IN ('pending','failed')` | Database / Storage | — | D-06 合并语义基础设施；migration_215 双方言注册（migration_199/212 先例） |
| handler 旧推送路径删除 | Backend / Handler | — | **本 phase 不删**（D-03，移入 Phase 58） |
| 排水器（in-process cron） | Backend / Service | — | **本 phase 不做**（Phase 58） |
| GUARD-01 守卫查询 | Backend / Service | — | **本 phase 不做**（Phase 59），但队列表 schema 须 GUARD-01 可查询 |
| operlog 调用 | Backend / Service | — | user_service.Delete / BatchDelete 已通过 handler.Record 记 OperTypeDelete / OperTypeBatch，service 内不再调（CLAUDE.md 强制） |

## Standard Stack

### Core（无新增外部依赖）

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `gorm.io/gorm` | v1.30.5 | 模型/事务/Schema | 全项目统一 ORM |
| `github.com/google/uuid` | v1.6.0 | BeforeCreate 生成 UUID | BaseModel 既有钩子（base.go:22） |
| `github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog` | — | handler 层操作日志 | CLAUDE.md 强制约定（service 层不调） |
| `applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"` | — | WARN 日志 | 全项目 slog/logrus 封装 |
| `apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"` | — | 错误码包装 | 全项目统一 |

**无新增 Go module 依赖**——Phase 57 全部使用既有项目基础设施。模型 + 迁移 + service 函数，无第三方库引入。

### Supporting（既有依赖复用）

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/models/config_restore_task.go` 模式 | — | 无 DeletedAt + BeforeCreate UUID + 可空 *time.Time 终态 | 队列表模型直接套用此先例 |
| `migration_199_fix_suggestion_unique_index.go` | — | PG-only 部分唯一索引 + 显式 `uni_` 命名 | 本次 D-06 部分唯一索引先例 |
| `migration_212_create_restore_task_active_unique_index.go` | — | 双方言部分唯一索引 `WHERE status IN (...)` | 本次 D-06 双方言注册先例 |
| `migration_211_create_config_restore_task.go` | — | 双方言 CREATE TABLE IF NOT EXISTS + 多个 CREATE INDEX IF NOT EXISTS | 本次建表先例 |
| `migration_214_create_ad_login_history.go` | — | isPostgreSQL 守卫 + 纯 SQL + BeforeCreate UUID | 本次 schema 风格参考 |
| `internal/core/db/migrations/migration_helpers.go` | — | `isPostgreSQL(db)` 函数 | 双方言迁移统一守卫 |
| `config_backup_service_93_02_test.go:402` 模式 | — | sqlite test 直调 `migrations.Migrate212...` 建表 | 本次 sqlite 测试建表先例 |
| `fix_suggestion_service_test.go:36-77` 模式 | — | sqlite test 直接 CREATE TABLE 复制 DDL | 本次 sqlite 测试建表 fallback 先例 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 部分唯一索引 (D-06) | 一生一行状态机（UNIQUE(entity_type, entity_id)） | 一生一行虽合并最简但留不下行内历史，无法满足 QUEUE-03"达上限转终态且可查询"；且 migration_199/212 双先例验证部分唯一索引可行 |
| 部分唯一索引 (D-06) | JSON payload 快照 | 时间戳仲裁土壤复活（D-05 已否决）；compare-and-move 幂等超集可消解 |
| 应用层先查后写合并 | 唯一索引 ON CONFLICT 夺锁 | net_device_enrichment_task 并发窗口 + 僵尸前科（D-06 显式禁用） |
| ad_push_queue 同事务入队 (D-01/D-04) | 异步 goroutine 入队 | 异步会架空 QUEUE-01 进程重启不丢失；同事务是唯一诚实实现 |
| 队列 + cron 排水（Phase 58） | MQ（RabbitMQ/Kafka） | PROJECT.md 已 lock："单机部署，引 MQ 多一个运维件无收益" |
| 代码内入队逻辑（Phase 57） | DB TRIGGER BeforeInsert/UPDATE | PROJECT.md 已 lock："同步是多表有序编排需可观测，trigger 是错误的层（域特性判断）"；memory `user-prefers-code-fixes-no-db-triggers` 进一步约束 |

## Package Legitimacy Audit

**Phase 57 无新增外部包（无 npm/pip/cargo install）**，审计不适用。唯一新增物是项目内 Go 文件与迁移函数，所有调用方/被方调用方皆既有依赖。

| Item | Source | Audit |
|------|--------|-------|
| `models.ADPushQueue` | 项目内新建 | 套用 `ConfigRestoreTask` 既有先例 |
| `migration_215_create_ad_push_queue.go` | 项目内新建 | 套用 `migration_211/212/214` 既有先例 |
| `internal/services/addomain/ad_push_queue_service.go`（或 `internal/services/system/`） | 项目内新建 | 套用既有 service 模式 |

## Architecture Patterns

### System Architecture Diagram

Phase 57 落地的数据流（不含 Phase 58 排水、Phase 59 守卫、Phase 60 SAFE）：

```
管理员 PUT /system/users/:id/update (四字段变更)
        │
        ▼
user_handler.Update
        │  (Phase 58 才删旧路径：保留 adSyncInFlight goroutine + SyncUserUpdateToAD)
        ▼
user_service.Update(ctx, req)
        │
        ├─► First(&user)              ← 旧值快照（D-01 来源）
        │
        ▼
db.WithContext(ctx).Transaction(func(tx) error {
        │  1. tx.Save(&user)                                  ← 业务保存
        │  2. tx.Table("sys_user_role").Where(...).Delete()   ← 角色关联清理
        │  3. tx.Create(&userRoles)                            ← 角色关联重建
        │  4. tx.Table("sys_user_post").Where(...).Delete()  ← 岗位关联清理
        │  5. tx.Create(&userPosts)                            ← 岗位关联重建
        │  ★ 6. diffADFields(oldUser, req) → enqueueUpdate(tx, oldUser.ID)   ← Phase 57 入队点
        │     │  ├─ 前置条件：sys_ad_config 启用行存在？         ← D-02
        │     │  ├─ 唯一索引夺锁：ON CONFLICT(entity_type,entity_id,action_type)
        │     │  │    WHERE status IN ('pending','failed') DO UPDATE ← D-06 合并
        │     │  └─ 失败 → 同事务回滚                           ← D-02
        │  7. tx.Commit()
        ▼
[handler 层异步 goroutine 调 SyncUserUpdateToAD] ← 保留至 Phase 58 (D-03)
```

Delete / BatchDelete 路径同理，仅入队行 action_type = "disable"。

### Recommended Project Structure

新建文件（Phase 57）：

```
internal/
├── models/
│   └── ad_push_queue.go                          # 新建（GORM 模型，无 DeletedAt，BeforeCreate UUID）
├── core/db/migrations/
│   └── migration_215_create_ad_push_queue.go     # 新建（双方言 CREATE TABLE + 部分唯一索引 + 退避列索引）
└── services/
    └── addomain/                                  # 推荐放 addomain 包（与现有 sync 包同仓，依赖方向不破）
        └── ad_push_queue_service.go               # 新建（enqueueUpdate / enqueueDisable / diffADFields 纯函数）
                                                   # 或放在 internal/services/system/（包内调用，无循环依赖）
```

修改文件（Phase 57）：

```
internal/services/system/user_service.go
├── Update(ctx, req)   行 :157 tx.Commit 前插入 enqueueUpdate(tx, oldUser, req) 入队点
├── Delete(ctx, id)    行 :240 tx.Commit 前插入 enqueueDisable(tx, user.ID) 入队点
└── BatchDelete(ctx, ids) 行 :575 tx.Commit 前插入 enqueueDisableBatch(tx, ids) 入队点
```

`internal/core/db/database.go`：

```
注册 migrations.Migrate215CreateADPushQueue(db) 调用 — 双方言块（PG:advisory-lock 内 / sqlite:AutoMigrate 后）各一处
加入 &models.ADPushQueue{} 到 MigrateModelList（Phase 58 排水器 / Phase 59 守卫需要模型；Phase 57 测试也用）
```

### Pattern 1: 双方言部分唯一索引迁移

**What:** 单个 migration 文件同时注册 PG 和 sqlite 两套 DDL，谓词文本重复（双方言均要求精确文本匹配）。

**When to use:** 任何需要"非终态行夺锁、终态行不冲突"的去重场景（队列去重、活跃任务互斥、未删除行唯一等）。

**Example（先例 migration_212）:**

```go
// Source: internal/core/db/migrations/migration_212_create_restore_task_active_unique_index.go
package migrations

import (
    "log"

    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "gorm.io/gorm"
)

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

**关键:**
- 双方言注册（database.go 的 PG advisory-lock 块 + sqlite else 块各调一次）
- 显式 `uni_` / `uq_` 前缀命名（避免 GORM 自动名与 SQL inline 命名冲突 — W-1 教训，memory `ruoyi-gorm-sql-constraint-naming-conflict`）
- `IF NOT EXISTS` 幂等可重入
- 不在模型 tag 声明该唯一索引（避免 GORM/PG 命名冲突）
- PG 部分 unique index 要求 sqlite 3.8+（modernc v1.40.1 已满足）

### Pattern 2: 同事务入队（outbox-lite）

**What:** 业务事务块末尾、`tx.Commit()` 之前插入队列行；INSERT 失败 → 同事务回滚；前置条件检查放在事务内还是事务外的设计抉择见 Pitfall 7。

**When to use:** 任何"业务写操作必须产生副作用消息、且副作用不能丢"的场景。

**Example（伪代码，user_service Update 路径）:**

```go
// Source: 内化自 user_service.go:157 现有事务块模式 + CONTEXT.md D-01/D-02
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 1. 业务更新（既有逻辑）
    if err := tx.Save(&user).Error; err != nil {
        return apperrors.DatabaseError(err)
    }
    // ... 角色/岗位关联清理与重建（既有逻辑）

    // 3. Phase 57 入队点 — diff + 入队（前置条件检查见 Pitfall 7）
    if err := s.enqueueUpdateIfChanged(tx, oldUser, req); err != nil {
        return err  // INSERT 失败 → 同事务回滚（D-02 第二层）
    }
    return nil
})
```

**关键:**
- 入队调用必须接收 `tx *gorm.DB`（同事务），而非 `s.db`（事务外）
- 失败必须 `return err`（让 `tx.Commit()` 不执行）
- 不在 goroutine 内异步入队（会架空 QUEUE-01）
- diff 函数纯函数化（与 DB 解耦）便于单测

### Pattern 3: 手写 migration + 模型 schema 双轨

**What:** 表结构由手写 migration_NNN 文件定义（DDL + 索引 + 部分唯一索引），GORM 模型仅作"读侧"便利（BeforeCreate / Scan）。模型 tag 不声明 `uniqueIndex`（避免 GORM/PG 命名冲突）。

**When to use:** 任何表需要非 GORM 推导的 DDL（部分唯一索引、BRIN、复合索引重命名、分区表等）。

**Example（先例 ConfigRestoreTask）:**

```go
// Source: internal/models/config_restore_task.go:25
type ConfigRestoreTask struct {
    ID           string     `gorm:"type:uuid;primary_key" json:"id"`
    DeviceID     string     `gorm:"type:uuid;not null;index" json:"deviceId"`
    Status       string     `gorm:"size:20;not null;index" json:"status"`
    // ... 无 uniqueIndex 字段
    CreatedAt    time.Time  `json:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt"`
}

// migration_211 显式建表（DDL 含主键、列、索引）
// migration_212 显式建部分唯一索引（uq_ 前缀）
```

### Anti-Patterns to Avoid

- **JSON payload 快照（载荷）** — 时间戳仲裁土壤复活；D-05 显式否决。载荷 = 最小动作行（entity_type / entity_id / action_type）。
- **应用层先查后写合并** — net_device_enrichment_task 并发窗口 + 僵尸前科；D-06 显式禁止，唯一索引 ON CONFLICT 夺锁。
- **UpdateStatus 双向推送（停用→disable / 启用→enable）** — scope 膨胀；D-04 显式排除 v1.21 外；动作枚举预留扩展位供后续 milestone。
- **delete AD 账号** — 与 Phase 60 SAFE-01 ad_dn 校验恢复不兼容；addomain 全库无 DeleteUser 先例；D-04 显式禁止。
- **同事务内 defer 入队** — defer 在 tx return 之后执行；入队必须在事务块体内。
- **不传 tx 入队**（用 s.db 而非 tx）— 不在同一事务，架空 QUEUE-01 进程重启不丢失。
- **Goroutine 异步入队** — 同上，架空 QUEUE-01。
- **UpdateStatus 路径入队** — D-04 显式排除；动作枚举预留即可，不实装。
- **sys_ad_config 启用行无 → silent pass 不入队** — D-02 第一层前置条件；要 WARN 日志放行用户编辑（真实信号，永 pending 会污染 Phase 59 守卫）。
- **handler 层入队逻辑** — 必须放 service 层事务内（CLAUDE.md Handler-Service 分层强制）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 同事务入队（QUEUE-01 进程重启不丢失） | 异步 goroutine + 通道 | GORM `tx.Transaction(func(tx) error { ... INSERT })` | 异步通道进程崩溃即丢；GORM 同事务是唯一诚实实现 |
| 同实体多次编辑合并（QUEUE-02） | 应用层 SELECT-then-INSERT-UPDATE | PG 部分唯一索引 `WHERE status IN ('pending','failed')` + ON CONFLICT | 应用层有 TOCTOU 窗口；net_device_enrichment_task zombie 前科 |
| 软删恢复安全（Phase 60） | service 层 custom 校验 | catSoftDeleted 共用分类逻辑 | memory `duty-module-user-field-gotchas` 显示分类逻辑必须共用 |
| UUID 主键生成 | service 层 uuid.New().String() 手动赋值 | 模型 BeforeCreate 钩子 | BaseModel 既有钩子（base.go:22）防 23502 |
| 多 SQL 方言兼容（PG + sqlite） | 两套 DDL 文件 | migration_211/212 单一文件 + isPostgreSQL 分支 | 现代内存测试靠 sqlite；生产靠 PG；既有先例已验证 |
| 列存在性测试（sqlite in-memory test） | 复制 PG 整张表 DDL | `db.Exec("ALTER TABLE ... ADD COLUMN ...")` 动态补全（先例 user_service_crud_test.go:34-58） | GORM schema parse + pragma_table_info 是已有 mode |

**Key insight:** Phase 57 是 outbox-lite 模式的最简落地——表 + 一处入队点 + 一个唯一索引——三方审计否决了完整 Outbox（状态机/事务性队列全套）的过度方案；死代码、僵尸、并发窗口、TOCTOU 都有既有教训与先例可查，无须自研。

## Runtime State Inventory

**Phase 57 触发类别**："greenfield 新建队列表 + 修改既有事务路径"混合。

| Category | Question | Answer |
|----------|----------|--------|
| Stored data | What databases store `ad_push_queue`? | None — 新表，greenfield；migration_215 自带 `CREATE TABLE IF NOT EXISTS` 幂等 |
| Stored data | 现有 ad_dn、DN 缓存会被影响吗？ | No — Phase 57 只新增队列表写入，不读不删 sys_user.ad_dn |
| Live service config | n8n / Datadog / Tailscale / Cloudflare 等有 ad_push_queue 引用吗？ | No — 新表名未在任何外部 service 注册 |
| OS-registered state | Windows Task Scheduler / pm2 / systemd 嵌入 ad_push_queue 字样？ | No — 字符串未在任何 OS 注册 |
| Secrets/env vars | SOPS key / .env 文件含 ad_push_queue 引用？ | No — 新表名未在 secrets 注册 |
| Build artifacts | pip egg-info / compiled binaries / Docker image 含 ad_push_queue 字样？ | No — 新表名首次出现 |

**Additional check (per "modifying the user write paths" 维度):**

| Category | Question | Answer |
|----------|----------|--------|
| 已有 adSyncInFlight goroutine (handler 层) | Phase 57 影响？ | No — D-03 保留至 Phase 58 删 |
| 已有 SyncUserUpdateToAD 调用点 | Phase 57 影响？ | No — D-03 保留至 Phase 58 删 |
| operlog.Record 调用点（user_handler） | Phase 57 影响？ | No — Phase 57 不改 handler 层 operlog；service 内不入队影响 operlog |
| user_service.Update/Delete/BatchDelete 已有 client 调用者 | Phase 57 影响？ | Yes — 事务块尾部加 diff + 入队，但失败同事务回滚，外部行为不变（业务编辑失败语义不变），仅多一行 ad_push_queue 副作用 |
| sqlite 测试套件 | Phase 57 影响？ | Yes — HYGIENE-06 要求补建队列表；migration_215 双方言注册后，sqlite in-memory 测试调用 migrations.Migrate215CreateADPushQueue 即可；fix_suggestion_service_test.go:36-77 + config_backup_service_93_02_test.go:402 双先例 |
| Phase 58 排水器 / Phase 59 守卫 | Phase 57 影响？ | No — 队列表 schema 已就位，下游消费者按需消费，不阻塞本 phase |

## Common Pitfalls

### Pitfall 1: ON CONFLICT 冲突目标不匹配部分索引谓词（PG 42P10 / sqlite error）

**What goes wrong:** `ON CONFLICT (entity_type, entity_id, action_type) WHERE status IN ('pending','failed')` —— 部分唯一索引的 WHERE 谓词必须文本级重复在 ON CONFLICT 子句中，否则 PG 报 42P10、sqlite 报 "no such partial index"。

**Why it happens:** ON CONFLICT 推断目标索引时需要文本级匹配谓词。

**How to avoid:** ON CONFLICT 子句直接复用与部分索引 DDL 完全一致的 WHERE 谓词（PG/sqlite 双方言均要求）。GORM 侧该谓词必须写在 `clause.OnConflict.TargetWhere`（而非 `Where`），否则被渲染成 `DO UPDATE ... WHERE` 尾随谓词、部分索引不被推断→ PG 42P10 / sqlite "does not match any PRIMARY KEY or UNIQUE constraint"；裸 SQL 表达式用 `clause.Expr`（v1.30.5 无 `clause.Literal`）；谓词不带表名限定。先例：`internal/services/asset/reconciliation_detection.go:426`（该处 :415 注释记录了不用 TargetWhere 的失败症状）；DDL 侧先例 `migration_212_create_restore_task_active_unique_index.go:27`。

**Warning signs:** 迁移后第一次同实体 INSERT 抛 "there is no unique or exclusion constraint matching the ON CONFLICT specification"。

### Pitfall 2: 模型 tag 声明 uniqueIndex → GORM/PG 约束命名冲突

**What goes wrong:** 模型字段 tag 加 `uniqueIndex:uni_ad_push_queue_pending` → GORM 创建索引名为 `uni_ad_push_queue_pending`；同时手写 migration_215 创建 `uni_ad_push_queue_pending`，DROP 时 GORM 找不到对应 GORM auto-FATA。

**Why it happens:** memory `ruoyi-gorm-sql-constraint-naming-conflict` 教训——GORM uniqueIndex 与 SQL inline UNIQUE 命名冲突。

**How to avoid:** 模型 tag **不得**声明该唯一索引；只能手写 migration_215 显式 `uni_` 命名。

**Warning signs:** 第二次启动 GORM AutoMigrate 报 "constraint ... does not exist" 或 DROP 无 IF EXISTS FATA。

### Pitfall 3: 入队 INSERT 真失败时"跳过+WARN"架空 QUEUE-01

**What goes wrong:** service 层捕获 INSERT 错误后仅 WARN 放行，事务继续提交 → 业务写入了 sys_user 但 ad_push_queue 无对应行，重启后推送永久丢失。

**Why it happens:** "降级处理"善意逻辑反过来破坏语义；D-02 第二层明示 INSERT 真失败必须同事务回滚。

**How to avoid:** 入队函数 `error` 必须由 `tx` 回调 `return err`，让 `tx.Commit()` 不执行；WARN 仅用于前置条件不满足（无启用 AD 配置）的语义放行。

**Warning signs:** 数据库日志中 sys_user 有 UPDATE 但 ad_push_queue 无对应 INSERT；用户反馈"改了部门但 AD 没改"。

### Pitfall 4: handler 层入队 vs service 层入队

**What goes wrong:** 入队逻辑放在 handler 层（与现有 goroutine + SyncUserUpdateToAD 并列）→ handler 层的事务边界外，架空 QUEUE-01。

**Why it happens:** 既有 handler 层已有 adSyncInFlight 模式，新人容易"靠近现有代码"放错层。

**How to avoid:** 入队逻辑必须在 service 层事务内；handler 仅保留 operlog.Record 与 D-03 保留的旧路径。

**Warning signs:** 入队 helper 接收 `*gin.Context` 而非 `*gorm.DB`/`tx` —— 反模式信号。

### Pitfall 5: 前置条件检查放事务外（并发窗口）

**What goes wrong:** 入队前先 `s.db.Where(...).First(&adConfig)` 检查 AD 配置启用 → 在事务外查 → 业务事务内 INSERT 时 AD 配置已被并发关闭，污染队列行（应跳过却入了队）。

**Why it happens:** 事务外查询与事务内 INSERT 之间存在并发窗口。

**How to avoid:** 前置条件检查放事务内（用 `tx` 查询），保持与 INSERT 同事务同连接同 snapshot；或事务外查询 + 事务内 INSERT 仍保留（窗口虽存在但 AD 配置启停不频繁，可权衡见 Pitfall 7）。

**Warning signs:** 测试用例 mock AD 配置开关瞬间切换，断言入队行存在性偶发失败。

### Pitfall 6: 终态行覆盖活跃行的部分唯一索引

**What goes wrong:** 假设状态机为 pending→failed→pushed；行从 pending 转 failed 时，部分唯一索引 `WHERE status IN ('pending','failed')` 仍命中，新一行同 (entity_type, entity_id, action_type) INSERT 报冲突。

**Why it happens:** failed 仍在谓词内（设计意图：保留重试能力）；同实体第二次编辑 INSERT 想覆盖失败行报 unique。

**How to avoid:** Phase 58 退避器拾取（Phase 57 仅写 pending 行）；Phase 57 入队即同 (entity_type, entity_id, action_type) 的 pending 行存在时用 ON CONFLICT DO UPDATE 把 attempts 重置、last_error 清空、created_at 刷成新一次编辑时间。D-06"upsert 覆盖即 QUEUE-02 合并"语义靠此保证。

**Warning signs:** 同一用户连续两次编辑，第二次报 unique 冲突而非合并。

### Pitfall 7: 前置条件 AD 配置查询放在事务内还是事务外

**What goes wrong:** 选项 A（事务内）：tx.First(&AC) 在事务内读 sys_ad_config → 性能有损（事务延长），但并发安全。选项 B（事务外）：s.db.First(&AC) 在事务外读 → 性能好，但有 TOCTOU 窗口（窗口期内 AC 配置被关闭，队列入队本应跳过）。

**Why it happens:** 决策权衡。

**How to avoid:** **推荐选项 B（事务外）**——AD 配置 sys_ad_config 启停是人工运维操作，频率极低（不会高频抖动）；TOCTOU 窗口下的"误入队"实际语义可接受（队列入 pending 后 Phase 58 排水检查 ad_dn 仍可生效）；事务延长对用户编辑 RT 的影响不可接受（每次 update 都要多一次 round-trip）。

**Warning signs:** 性能测试显示用户编辑 P99 显著恶化（应回退到选项 A）；并发测试切换 AD 配置开关瞬间偶发误入队（可接受，记 WARN）。

### Pitfall 8: BatchDelete N 个用户 → N 条 INTO INSERT 性能差

**What goes wrong:** BatchDelete 事务内逐用户 INSERT 一行 ad_push_queue；N=1000 时 1000 次 round-trip。

**Why it happens:** 简单粗暴实现。

**How to avoid:** BatchDelete 部分服务用 `tx.CreateInBatches(&rows, batchWriteSize=500)` 或单条多值 INSERT（同 user_sync_service.go:602 已用模式）；保证部分唯一索引仍生效（ON CONFLICT DO UPDATE 仍然把单个语句内冲突行合并）。

**Warning signs:** BatchDelete 1000 用户耗时 > 1s。

### Pitfall 9: 在已有 user_handler goroutine 内"借用"入队

**What goes wrong:** 试图在 user_handler.Update 现有的 `go func() { SyncUserUpdateToAD(...) }()` goroutine 内追加 enqueueUpdate → 入队发生在事务外（goroutine 是事务后才启动），架空 QUEUE-01。

**Why it happens:** "靠近现有代码"的本能。

**How to avoid:** 入队必须在 service 层事务内；handler 层仅 D-03 保留的旧路径（async goroutine + SyncUserUpdateToAD）继续存在但不读队列表（Phase 58 才删）。

**Warning signs:** 入队函数被 import 到 handler 层（违反 Handler-Service 分层）。

### Pitfall 10: 队列行 status 字段用 int 而非 string

**What goes wrong:** 状态机字段定义为 `int`（pending=0, failed=1, pushed=2, dead=3）→ 排障 SQL 难读，partial unique index 谓词写 `WHERE status IN (0, 1)` 易混。

**Why it happens:** 与 sys_user.status=0/1 约定一致但语义不同。

**How to avoid:** status 字段用 `string` 枚举（pending / failed / pushed / dead）；先例 ConfigRestoreTask（RestoreTaskStatus = "pending"/"running"/"success"/"failed"）。partial unique index 谓词用 `WHERE status IN ('pending','failed')`。

**Warning signs:** 部分唯一索引迁移后 SELECT 测试用 int 比较报 "no rows"。

## Code Examples

### Verified patterns from official sources (源码内先例)

### 1. 新建 `migration_215_create_ad_push_queue.go`（双方言 + 部分唯一索引）

```go
// Source: 内化自 migration_211_create_config_restore_task.go + migration_212_create_restore_task_active_unique_index.go
package migrations

import (
    "log"

    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "gorm.io/gorm"
)

// Migrate215CreateADPushQueue 创建 ad_push_queue 表（v1.21 Phase 57 D-05/D-06）
//
// 设计要点：
//   - 双方言 CREATE TABLE IF NOT EXISTS 幂等；PG 走 advisory-lock 块注册。
//   - 状态列 string 枚举（pending/failed/pushed/dead）。
//   - 部分唯一索引 `WHERE status IN ('pending','failed')` 双方言注册，
//     ON CONFLICT 子句文本级复用谓词（PG 42P10 + sqlite 双方言要求）。
//   - 模型 tag 不声明该唯一索引（避免 GORM/PG 命名冲突 W-1 教训）。
//   - 退避列（attempts / next_retry_at / last_error / failed_at）一次建齐（D-07）。
//   - 终态列（pushed_at / failed_at）使用 *time.Time 可空。
//   - 索引：entity_id 拾取（Phase 58 排水按 entity 拉取），(status, next_retry_at) 拾取（Phase 58 退避器扫描）。
func Migrate215CreateADPushQueue(db *gorm.DB) error {
    log.Println("Running migration 215: create ad_push_queue + active-unique partial index")

    const createMainSQL = `CREATE TABLE IF NOT EXISTS ad_push_queue (
        id uuid PRIMARY KEY,
        entity_type varchar(32) NOT NULL,
        entity_id uuid NOT NULL,
        action_type varchar(32) NOT NULL,
        status varchar(20) NOT NULL DEFAULT 'pending',
        attempts integer NOT NULL DEFAULT 0,
        next_retry_at timestamptz,
        last_error text,
        created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
        pushed_at timestamptz,
        failed_at timestamptz
    )`
    if err := db.Exec(createMainSQL).Error; err != nil {
        applogger.Errorf("migration 215: create ad_push_queue 失败: %v", err)
        return err
    }

    // 部分唯一索引（双方言一致；PG 3.8+ + sqlite 3.8+ 都支持 partial index）
    const activeUniqueIndexSQL = `CREATE UNIQUE INDEX IF NOT EXISTS uq_ad_push_queue_active
ON ad_push_queue (entity_type, entity_id, action_type)
WHERE status IN ('pending', 'failed')`
    if err := db.Exec(activeUniqueIndexSQL).Error; err != nil {
        applogger.Errorf("migration 215: create active-unique partial index 失败: %v", err)
        return err
    }

    // 排水器扫描索引（D-07: 一次建齐）
    const retryScanIndexSQL = `CREATE INDEX IF NOT EXISTS idx_ad_push_queue_status_retry
ON ad_push_queue (status, next_retry_at)`
    if err := db.Exec(retryScanIndexSQL).Error; err != nil {
        applogger.Errorf("migration 215: create retry scan index 失败: %v", err)
        return err
    }

    // entity_id 拾取索引（Phase 58 排水按 entity 拉取）
    const entityIDIndexSQL := `CREATE INDEX IF NOT EXISTS idx_ad_push_queue_entity_id
ON ad_push_queue (entity_id)`
    if err := db.Exec(entityIDIndexSQL).Error; err != nil {
        applogger.Errorf("migration 215: create entity_id index 失败: %v", err)
        return err
    }

    return nil
}
```

**注册到 database.go（双方言块各一处）：**
```go
// PG 分支（advisory-lock 块内，line ~1048 附近，紧跟 Migrate214CreateADLoginHistory）
if err := migrations.Migrate215CreateADPushQueue(d.DB); err != nil {
    applogger.Errorf("ad_push_queue 迁移失败 (非阻断,留待下次启动): %v", err)
}

// sqlite 分支（line ~1076 附近，紧跟 Migrate213RecomputeBuildingTotalFloors）
if err := migrations.Migrate215CreateADPushQueue(d.DB); err != nil {
    applogger.Errorf("ad_push_queue 迁移失败 (非阻断,留待下次启动): %v", err)
}
```

### 2. 新建 `models/ad_push_queue.go`

```go
// Source: 内化自 internal/models/config_restore_task.go
package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ADPushActionType AD 推送动作类型枚举（v1.21 Phase 57 D-04）。
// 字符串常量，预留扩展位（disable 之外未来可加 status 类动作如 enable）。
// 注意：UpdateStatus 双向推送明确排除在 v1.21 外（CONTEXT.md Deferred）。
type ADPushActionType string

const (
    ADPushActionUpdateUser ADPushActionType = "update_user" // user 四字段变更
    ADPushActionDisable    ADPushActionType = "disable"     // sys 删除用户 → disable AD 账号
)

// ADPushStatus AD 推送状态枚举（v1.21 Phase 57）。
// string 枚举仿 ConfigRestoreTask；Phase 58 排水器负责流转。
type ADPushStatus string

const (
    ADPushStatusPending ADPushStatus = "pending" // 待排水
    ADPushStatusFailed  ADPushStatus = "failed"  // 排水失败待重试
    ADPushStatusPushed  ADPushStatus = "pushed"  // 排水成功终态
    ADPushStatusDead    ADPushStatus = "dead"    // 达重试上限 / LDAP code 32 终态（Phase 58）
)

// ADPushEntityType AD 推送实体类型枚举（v1.21 Phase 57 D-04）。
// 当前只支持 user；预留扩展位（dept / group 等未来可加）。
type ADPushEntityType string

const (
    ADPushEntityUser ADPushEntityType = "user"
)

// ADPushQueue AD 推送队列（v1.21 Phase 57 outbox-lite）。
//
// 设计要点：
//   - 不软删（无 DeletedAt；终态行留表可查 D-08）。
//   - BeforeCreate UUID 防 23502（BaseModel 模式）。
//   - 模型 tag 不声明唯一索引（W-1 教训）；唯一索引由 migration_215 显式建。
//   - 终态列（pushed_at / failed_at）可空 *time.Time。
//   - 载荷 = 最小动作行（D-05）；compare-and-move 排水闭包重读 DB 是幂等超集。
type ADPushQueue struct {
    ID          string             `gorm:"type:uuid;primary_key" json:"id"`
    EntityType  ADPushEntityType   `gorm:"size:32;not null;index" json:"entityType"`
    EntityID    string             `gorm:"type:uuid;not null" json:"entityId"`
    ActionType  ADPushActionType   `gorm:"size:32;not null" json:"actionType"`
    Status      ADPushStatus       `gorm:"size:20;not null" json:"status"`
    Attempts    int                `gorm:"not null;default:0" json:"attempts"`
    NextRetryAt *time.Time         `json:"nextRetryAt,omitempty"`
    LastError   string             `gorm:"type:text" json:"lastError,omitempty"`
    CreatedAt   time.Time          `json:"createdAt"`
    UpdatedAt   time.Time          `json:"updatedAt"`
    PushedAt    *time.Time         `json:"pushedAt,omitempty"`
    FailedAt    *time.Time         `json:"failedAt,omitempty"`
}

// BeforeCreate GORM 钩子：创建前生成 UUID（BaseModel 模式防 23502）。
func (q *ADPushQueue) BeforeCreate(tx *gorm.DB) error {
    if q.ID == "" {
        q.ID = uuid.New().String()
    }
    if q.Status == "" {
        q.Status = ADPushStatusPending
    }
    return nil
}

// BeforeUpdate GORM 钩子：更新前刷新 UpdatedAt。
func (q *ADPushQueue) BeforeUpdate(tx *gorm.DB) error {
    q.UpdatedAt = time.Now()
    return nil
}

// TableName 设置表名（v1.21 Phase 57）。
func (ADPushQueue) TableName() string {
    return "ad_push_queue"
}
```

### 3. 新建队列 service（含 diff helper + enqueue 函数）

```go
// Source: 内化自 user_service.go:146 Update 事务模式 + D-01/D-02/D-04 决策
package addomain  // 或 system，planner 选

import (
    "context"
    "errors"
    "strings"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
    applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
    "gorm.io/gorm"
)

// ADSyncEnabledChecker AD 配置启用检查函数（service 层依赖注入，便于测试 mock）。
type ADSyncEnabledChecker func(tx *gorm.DB) (bool, error)

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

// ADPushEnqueuer 队列入队接口（service 层依赖注入）。
type ADPushEnqueuer struct {
    DB          *gorm.DB
    IsADEnabled ADSyncEnabledChecker
}

// EnqueueUpdateIfChanged diff 入队（D-01）。
// 对比 oldUser 与 req，仅四字段实际变更时入队 update_user 动作行。
// 前置条件不满足 → 不入队 + WARN（D-02 第一层）。
// INSERT 真失败 → return err 让同事务回滚（D-02 第二层）。
func (e *ADPushEnqueuer) EnqueueUpdateIfChanged(
    tx *gorm.DB, oldUser *models.User, req *requests.UserUpdateRequest,
) error {
    // 前置条件：AD 启用行存在？
    enabled, err := e.IsADEnabled(tx)
    if err != nil {
        return err  // 同事务回滚
    }
    if !enabled {
        applogger.Warnf("[ad_push_queue] 未找到启用的 AD 配置，跳过入队（用户 %s 编辑）", oldUser.ID)
        return nil
    }

    // diff 纯函数：实际变更字段判定
    if !hasADFieldChanged(oldUser, req) {
        return nil  // 无字段变更，不入队（D-01）
    }

    // 入队 update_user 动作行（合并语义：唯一索引夺锁）
    return e.enqueueUpsert(tx, models.ADPushEntityUser, oldUser.ID, models.ADPushActionUpdateUser)
}

// EnqueueDisable 删除入队（D-04）。
// 单用户 Delete 路径调用。
func (e *ADPushEnqueuer) EnqueueDisable(tx *gorm.DB, userID string) error {
    enabled, err := e.IsADEnabled(tx)
    if err != nil {
        return err
    }
    if !enabled {
        applogger.Warnf("[ad_push_queue] 未找到启用的 AD 配置，跳过 disable 入队（用户 %s）", userID)
        return nil
    }
    return e.enqueueUpsert(tx, models.ADPushEntityUser, userID, models.ADPushActionDisable)
}

// EnqueueDisableBatch 批量删除入队（D-04）。
// BatchDelete 路径调用；CreateInBatches 减少 round-trip（Pitfall 8）。
func (e *ADPushEnqueuer) EnqueueDisableBatch(tx *gorm.DB, userIDs []string) error {
    enabled, err := e.IsADEnabled(tx)
    if err != nil {
        return err
    }
    if !enabled {
        applogger.Warnf("[ad_push_queue] 未找到启用的 AD 配置，跳过批量 disable 入队（%d 个用户）", len(userIDs))
        return nil
    }

    rows := make([]models.ADPushQueue, 0, len(userIDs))
    for _, uid := range userIDs {
        rows = append(rows, models.ADPushQueue{
            EntityType: models.ADPushEntityUser,
            EntityID:   uid,
            ActionType: models.ADPushActionDisable,
            Status:     models.ADPushStatusPending,
        })
    }
    // CreateInBatches 保证部分唯一索引仍生效（单语句内 ON CONFLICT 仍合并）
    if err := tx.CreateInBatches(&rows, 500).Error; err != nil {
        return err  // 同事务回滚
    }
    return nil
}

// enqueueUpsert 单条入队（合并语义靠 ON CONFLICT DO UPDATE）。
// 单条用 Clauses(clause.OnConflict{...}) 显式声明更新列（D-06）。
func (e *ADPushEnqueuer) enqueueUpsert(
    tx *gorm.DB, entityType models.ADPushEntityType, entityID string, actionType models.ADPushActionType,
) error {
    row := &models.ADPushQueue{
        EntityType: entityType,
        EntityID:   entityID,
        ActionType: actionType,
        Status:     models.ADPushStatusPending,
    }
    // ON CONFLICT 谓词文本必须与部分唯一索引一致（双方言要求）。
    // gorm v1.30.5 三处 API 细节（clause/on_conflict.go）：
    //   1) 裸 SQL 表达式类型是 clause.Expr，clause.Literal 根本不存在（编译不过）；
    //   2) 部分索引谓词必须放 TargetWhere（渲染为冲突目标推断谓词），
    //      放 Where 会被渲染成 DO UPDATE 的尾随 WHERE → 索引不被推断
    //      → PG 42P10 / sqlite "does not match any PRIMARY KEY or UNIQUE constraint"；
    //   3) 谓词不带表名限定，文本级等于部分索引 DDL 的 WHERE 片段。
    // 项目内正确先例：internal/services/asset/reconciliation_detection.go:426。
    err := tx.Clauses(clause.OnConflict{
        Columns: []clause.Column{
            {Name: "entity_type"}, {Name: "entity_id"}, {Name: "action_type"},
        },
        TargetWhere: clause.Where{
            Exprs: []clause.Expression{
                clause.Expr{
                    SQL: "status IN ('pending','failed')",
                },
            },
        },
        DoUpdates: clause.Assignments(map[string]interface{}{
            "attempts":    0,
            "last_error":  "",
            "failed_at":   nil,
            "next_retry_at": nil,
            "updated_at":  time.Now(),
        }),
    }).Create(row).Error
    if err != nil {
        applogger.Errorf("[ad_push_queue] 入队失败 (entity=%s/%s, action=%s): %v",
            entityType, entityID, actionType, err)
        return err  // 同事务回滚
    }
    return nil
}

// hasADFieldChanged diff 纯函数：nickname/email/phone/deptId 四字段实际变更判定（D-01）。
// nil 指针语义：req 字段 nil = 无变更，跳过该字段（避免覆盖本地修改）。
// 可单测覆盖 nil/空串边界。
func hasADFieldChanged(oldUser *models.User, req *requests.UserUpdateRequest) bool {
    // Nickname
    if req.Nickname != nil {
        oldVal := ""
        if oldUser.Nickname != nil {
            oldVal = *oldUser.Nickname
        }
        if strings.TrimSpace(*req.Nickname) != strings.TrimSpace(oldVal) {
            return true
        }
    }
    // Email
    if req.Email != nil {
        oldVal := ""
        if oldUser.Email != nil {
            oldVal = *oldUser.Email
        }
        if strings.TrimSpace(*req.Email) != strings.TrimSpace(oldVal) {
            return true
        }
    }
    // Phone
    if req.Phone != nil {
        oldVal := ""
        if oldUser.Phone != nil {
            oldVal = *oldUser.Phone
        }
        if strings.TrimSpace(*req.Phone) != strings.TrimSpace(oldVal) {
            return true
        }
    }
    // DeptID
    if req.DeptID != nil {
        oldVal := ""
        if oldUser.DeptID != nil {
            oldVal = *oldUser.DeptID
        }
        if *req.DeptID != oldVal {
            return true
        }
    }
    return false
}
```

### 4. 修改 `user_service.go` 三处事务内入队

```go
// Source: 内化自 user_service.go:146 Update 事务模式 + D-01/D-02/D-04 决策

// Update 行 :157 — 事务块尾部插入入队点
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // ... 既有业务更新逻辑 ...

    // Phase 57 入队点：diff + 入队 update_user 动作行
    enqueuer := addomain.NewADPushEnqueuer(s.db) // 或 service 包内引用
    if err := enqueuer.EnqueueUpdateIfChanged(tx, &user, req); err != nil {
        return err  // 同事务回滚
    }
    return nil
})

// Delete 行 :240 — 事务块尾部插入入队点
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // ... 既有软删逻辑 ...

    // Phase 57 入队点：disable 动作行
    enqueuer := addomain.NewADPushEnqueuer(s.db)
    if err := enqueuer.EnqueueDisable(tx, id); err != nil {
        return err
    }
    return nil
})

// BatchDelete 行 :575 — 事务块尾部插入入队点
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // ... 既有批量软删逻辑 ...

    // Phase 57 入队点：批量 disable 动作行
    enqueuer := addomain.NewADPushEnqueuer(s.db)
    if err := enqueuer.EnqueueDisableBatch(tx, ids); err != nil {
        return err
    }
    return nil
})
```

### 5. sqlite 测试建表（HYGIENE-06）

```go
// Source: 内化自 config_backup_service_93_02_test.go:402 直调迁移函数 + fix_suggestion_service_test.go:36-77 复制 DDL 双先例
package system

import (
    "github.com/ruoyi-next/ruoyi-go-backend/internal/core/db/migrations"
    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// TestPhase57_ADPushQueue_UpdateEnqueue 验证 Update diff + 入队语义
func TestPhase57_ADPushQueue_UpdateEnqueue(t *testing.T) {
    db := setupUserServiceTestDB(t)

    // HYGIENE-06: sqlite 测试补建队列表（直调迁移函数模式）
    require.NoError(t, migrations.Migrate215CreateADPushQueue(db))

    // 种 AD 启用配置（D-02 前置条件）
    require.NoError(t, db.Exec(`INSERT INTO sys_ad_config (id, config_name, server_address, server_port, domain_name, base_dn, admin_username, sync_enabled, status, created_at, updated_at, deleted_at)
        VALUES ('v-mig-215', 'test', 'localhost', 389, 'test.local', 'dc=test,dc=local', 'admin', 1, 0, ?, ?, NULL)`, now := "2024-01-01 00:00:00", now).Error)

    id := seedUserRow(t, db, "diffme", 0)
    enqueuer := addomain.NewADPushEnqueuer(db)

    // case 1: nickname 变更 → 入队
    nick := "新昵称"
    require.NoError(t, enqueuer.EnqueueUpdateIfChanged(db, &models.User{ID: id, Username: "diffme"}, &requests.UserUpdateRequest{ID: id, Nickname: &nick}))

    var cnt int64
    require.NoError(t, db.Table("ad_push_queue").Count(&cnt).Error)
    assert.Equal(t, int64(1), cnt)

    // case 2: 角色变更（不入队）
    require.NoError(t, enqueuer.EnqueueUpdateIfChanged(db, &models.User{ID: id, Username: "diffme"}, &requests.UserUpdateRequest{ID: id, RoleIds: []string{"r1"}}))
    require.NoError(t, db.Table("ad_push_queue").Count(&cnt).Error)
    assert.Equal(t, int64(1), cnt, "纯角色变更不新增队列入")

    // case 3: 同 entity 二次 diff → 合并（QUEUE-02）
    nick2 := "再改"
    require.NoError(t, enqueuer.EnqueueUpdateIfChanged(db, &models.User{ID: id, Username: "diffme"}, &requests.UserUpdateRequest{ID: id, Nickname: &nick2}))
    require.NoError(t, db.Table("ad_push_queue").Count(&cnt).Error)
    assert.Equal(t, int64(1), cnt, "唯一索引夺锁应合并而非新增")

    // case 4: 无启用 AD 配置 → 不入队 + WARN
    require.NoError(t, db.Exec(`UPDATE sys_ad_config SET sync_enabled = 0`).Error)
    nick3 := "又被改"
    require.NoError(t, enqueuer.EnqueueUpdateIfChanged(db, &models.User{ID: id, Username: "diffme"}, &requests.UserUpdateRequest{ID: id, Nickname: &nick3}))
    require.NoError(t, db.Table("ad_push_queue").Where("entity_id = ? AND action_type = ?", id, "update_user").Count(&cnt).Error)
    assert.Equal(t, int64(1), cnt, "前置条件不满足不应入队")
}

// TestPhase57_ADPushQueue_DeleteEnqueue 验证 Delete disable 入队
func TestPhase57_ADPushQueue_DeleteEnqueue(t *testing.T) {
    db := setupUserServiceTestDB(t)
    require.NoError(t, migrations.Migrate215CreateADPushQueue(db))
    // ... AD 启用配置 + seed ...
    id := seedUserRow(t, db, "delme", 0)

    enqueuer := addomain.NewADPushEnqueuer(db)
    require.NoError(t, enqueuer.EnqueueDisable(db, id))

    var rows []models.ADPushQueue
    require.NoError(t, db.Where("entity_id = ?", id).Find(&rows).Error)
    require.Len(t, rows, 1)
    assert.Equal(t, models.ADPushActionDisable, rows[0].ActionType)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| handler 异步 goroutine + SyncUserUpdateToAD + adSyncInFlight dedupe | service 层同事务入队 + 部分唯一索引合并 | v1.21 Phase 57 | 进程重启不丢失（QUEUE-01）；同实体合并不丢弃（QUEUE-02）；删除 disable 闭环（QUEUE-06） |
| 完整 Outbox（状态机/事务性队列） | outbox-lite（最小动作行 + 部分唯一索引） | v1.21 Phase 57 | 三方审计否决过度设计；outbox-lite 覆盖单一调用点全部丢失模式 |
| 时间戳仲裁（updated_at vs ad_synced_at） | 信号守卫 + compare-and-move | v1.21 Phase 57-59 | GORM 自动刷 updated_at 使仲裁恒失效；信号守卫基于 pending/failed 队列行更可靠 |
| DB TRIGGER BeforeInsert/UPDATE | service 层 BeforeCreate hook + 代码层加固 | v1.21 | memory `user-prefers-code-fixes-no-db-triggers` 约束；同步需可观测 |

**Deprecated/outdated:**
- **handler goroutine `user_handler.go:194-238` 旧推送路径** — 保留至 Phase 58 删（D-03）
- **`BatchMoveUsersToNewOU`** — Phase 62 HYGIENE-03 死代码删除范围（F-22 旧路径），与 Phase 57 入队无关
- **`user_ou_service.go` 整文件** — Phase 62 HYGIENE-03 删除（F-22 旧路径），与 Phase 57 入队无关

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | service 包位置选 `addomain` | Architecture Patterns | 影响用户 import 路径与循环依赖；planner 可改 `system` |
| A2 | entity_type 字符串取 `"user"` | Models / Patterns | 扩展性 OK；预留 ADPushEntityType 枚举便于未来加 dept/group |
| A3 | action_type 字符串取 `"update_user"` / `"disable"` | Models / Patterns | 命名风格遵循下划线分隔；与 sys_job.action_type 惯例一致（如有）
- A4 | semicolon here is wrong
| A5 | diff helper 用 `strings.TrimSpace` 比较空串 | Code Examples | 边界 `""` vs `"  "` 一律视为无变化；用户实际编辑不会传空格串，可接受 |
| A6 | BatchDelete 用 `tx.CreateInBatches(&rows, 500)` | Patterns | PG 参数上限安全；sqlite 无上限问题；500 沿用 user_sync_service.go:470 batchWriteSize 惯例 |
| A7 | ON CONFLICT 用 `clause.OnConflict{...}` 显式 | Code Examples | 部分唯一索引谓词必须放 `TargetWhere`（非 `Where`），裸 SQL 用 `clause.Expr`（gorm v1.30.5 无 `clause.Literal`）；先例 `internal/services/asset/reconciliation_detection.go:426`（migration_212 本身不走 ORM） |
| A8 | 前置条件 AD 配置检查放事务内 (`tx`) | Pitfall 7 | 用户编辑 RT 恶化可回退到事务外 + WARN 容错 |
| A9 | migration_215 编号在 214 之后 | All | 214 已用；215 顺延；CONTEXT.md 已明示（roadmap "migration_200"编号失效） |
| A10 | `attempts/next_retry_at/last_error/failed_at` 字段定义在 Phase 57 而非 Phase 58 | Models / Migration | D-07 决策；Phase 58 排水器只需读取/更新这些列 |
| A11 | `pushed_at` 字段名（而非 `success_at`） | Models | 与 ConfigRestoreTask 的 CompletedAt 同语义；命名直觉 |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **service 包位置（addomain vs system）** — planner 决定
     - 现状：addomain 已是 AD 同步 service 集中仓（user_ad_sync_service.go / ldap_client.go / failover_client.go / account_pool.go / dept_ou_mapper 等）
     - 倾向：addomain（依赖方向更清晰：system → addomain，无循环）
     - 推荐：addomain

2. **Phase 58 排水器依赖** — Phase 58 planner 决定
     - Phase 57 入队后必须保证队列表 schema 字段集（attempts / next_retry_at / last_error / failed_at / pushed_at）够 Phase 58 排水使用
     - 已锁定：D-07 一次建齐

3. **BatchDelete 之外的批量路径** — CONTEXT.md 已 defer；Phase 57 不排查
     - 风险：Excel 删除类入口若有，将绕过入队
     - 推荐：Phase 57 UAT 显式声明 "本 phase 仅覆盖已确认的两条软删事务路径"

4. **handler 旧推送路径删除归 Phase 58** — CONTEXT.md D-03 已明示
     - Phase 57 不删
     - Phase 58 必删（roadmap 修正项）
     - Phase 57 UAT 需知悉间隙期 AD 端仍由旧路径推送

5. **disable 入队是否要前置条件检查** — D-02 第一层
     - 同 update_user 入队：未启用 AD 配置 → 不入队 + WARN
     - 但 disable 语义是"清理旧关联"（防已删用户 AD 登录复活），即使无 AD 配置也应执行？
     - 倾向：保持 D-02 一致（前置条件不满足 → 不入队）—— 同事务语义一致；无 AD 配置环境无 disable 需求

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL 18 | migration_215 PG 分支 | ✓（dev/staging 既有） | 18 | — |
| SQLite (modernc) | sqlite 测试 + 双方言分支 | ✓ | v1.40.1 | — |
| Go 1.24 | 项目运行时 | ✓ | go1.24.5 | — |
| GORM v1.30.5 | 模型 + tx + AutoMigrate | ✓ | v1.30.5 | — |
| google/uuid v1.6.0 | BeforeCreate 钩子 | ✓ | v1.6.0 | — |
| clause.OnConflict | ON CONFLICT DO UPDATE | ✓（GORM 自带） | gorm.io/gorm v1.30.5 | — |

**Missing dependencies with no fallback:** None

**Missing dependencies with fallback:** None

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/require` + `github.com/glebarez/sqlite` (sqlite in-memory) |
| Config file | 无显式配置文件；单测直调 `gorm.Open(sqlite.Open(":memory:"))`（先例 user_service_test.go:49） |
| Quick run command | `go test ./internal/services/system/ -run TestPhase57 -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| QUEUE-01 | 业务事务内入队 + 进程重启不丢失 | unit (sqlite + migration_215 + 事务测试) | `go test ./internal/services/system/ -run TestPhase57_ADPushQueue_UpdateEnqueue -v` | ❌ Wave 0 新建 |
| QUEUE-02 | 同实体多次编辑合并（唯一索引夺锁） | unit (sqlite 同实体两次 EnqueueUpdate) | `go test ./internal/services/system/ -run TestPhase57_ADPushQueue_UpdateEnqueue -v` | ❌ Wave 0 新建（同上含合并 case） |
| QUEUE-06 | Delete + BatchDelete disable 入队 | unit (sqlite + EnqueueDisable + EnqueueDisableBatch) | `go test ./internal/services/system/ -run TestPhase57_ADPushQueue_DeleteEnqueue -v` | ❌ Wave 0 新建 |
| HYGIENE-06 | sqlite 测试套件全绿 | unit (HYGIENE-06 触发条件：`go test ./...` 全绿) | `go test ./...` | ❌ Wave 0 新建（HYGIENE-06 测试 = 全套 sqlite 测试含 migration_215 建表） |

### Sampling Rate

- **Per task commit:** `go build ./...` + `go test ./internal/services/system/ -run TestPhase57 -v`
- **Per wave merge:** `go test ./...` + `go vet ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/services/system/ad_push_queue_test.go` — covers QUEUE-01 (diff + tx rollback on INSERT fail) / QUEUE-02 (同实体合并) / QUEUE-06 (Delete + BatchDelete) / HYGIENE-06 (直调 migration_215 建表)
- [ ] `internal/core/db/migrations/migration_215_create_ad_push_queue_test.go` — covers 双方言 DDL（sqlite 真实可建；PG 由 UAT 覆盖，仿 migration_212 模式无 _test 文件）
- [ ] Framework install: 既有（无新增依赖）

*(If no gaps: "None — existing test infrastructure covers all phase requirements")*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | AD bind / JWT 既有（Phase 60 SAFE 修复；本 phase 不涉及 auth 路径） |
| V3 Session Management | no | 无 session 改动 |
| V4 Access Control | yes (间接) | user_service Update/Delete/BatchDelete handler 层 RequirePermissions 既有（"system:user:edit"/"system:user:delete"）；本 phase service 层入队不影响权限 |
| V5 Input Validation | yes | `requests.UserUpdateRequest` 既有 binding 校验（min=0,max=1 等）；本 phase service 层 diff 不引入新校验面 |
| V6 Cryptography | no | 无加密改动 |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 同事务入队被中间人截断（事务回滚语义被绕过） | Tampering | GORM 同事务原子性由 PG/sqlite 提供；非 Phase 57 引入 |
| 队列表注入恶意 UUID 触发 disable AD 账号（绕过 admin） | Elevation of Privilege | DisableUser 由 ldap_client.go:337 复用；调用方 service 层；handler 层 RequirePermissions("system:user:delete") 既有 |
| 前置条件 AD 配置查询被绕开（事务外 TOCTOU） | Tampering | Pitfall 7 推荐事务内检查；user input 不可控制 ad_dn → enable 影响 |

**Key insight:** Phase 57 不引入新的攻击面——所有路径仍走既有 handler 层 operlog + RequirePermissions；service 层入队仅多一行副作用。

## Sources

### Primary (HIGH confidence)

- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_service.go:146-226` — Update 事务块（diff 旧值来源 :148-154）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_service.go:228-264` — Delete 事务块
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_service.go:568-599` — BatchDelete 事务块
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\api\v1\system\user_handler.go:194-238` — handler 旧推送路径（Phase 58 删，D-03）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\api\v1\system\user_handler.go:245-260` — `buildADSyncMap` 四字段契约参照
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\api\v1\system\user_router.go:35` — 无条件构造 UserADSyncService
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\addomain\user_ad_sync_service.go:44-104` — `SyncUserUpdateToAD`（handler 层仍调，Phase 58 删）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\addomain\user_ad_sync_service.go:252-253` — `sync_enabled=true AND status=enabled` 判定（D-02 前置条件参照）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\addomain\ldap_client.go:337` — `DisableUser`（QUEUE-06 执行侧复用）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\config_restore_task.go:25-67` — 队列表模型先例（无 DeletedAt / BeforeCreate UUID / 可空终态列）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\migrations\migration_211_create_config_restore_task.go` — 双方言 CREATE TABLE IF NOT EXISTS 先例
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\migrations\migration_212_create_restore_task_active_unique_index.go` — 双方言部分唯一索引 `WHERE status IN (...)` 关键先例
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\migrations\migration_199_fix_suggestion_unique_index.go` — PG-only 部分唯一索引 + 显式 `uni_` 命名先例
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\migrations\migration_214_create_ad_login_history.go` — isPostgreSQL 守卫 + 幂等 SQL + BeforeCreate UUID 风格先例
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\migrations\migration_helpers.go:7-9` — `isPostgreSQL(db)` 共享守卫
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\database.go:651-753` — `MigrateModelList` 模型注册列表
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\core\db\database.go:990-1086` — 双方言迁移注册块（PG advisory-lock / sqlite else）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\ad_domain.go:14-19` — `ADConfigStatus` 常量（`ADConfigStatusEnabled=0` / `ADConfigStatusDisabled=1`）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\user.go:34` — `AdDn *string` 列定义
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_sync_service.go:265-291` — `restoreSoftDeletedUser` 复活链路（D-04 守门人证据）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\system\requests\user_requests.go:92-104` — `UserUpdateRequest` 字段集（diff 来源）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_service_test.go:46-112` — sqlite test setup 基建（`setupUserServiceTestDB` / `seedUserRow` / `newTestUserService` / `mockPasswordManager`）
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\system\user_service_crud_test.go:30-58` — sqlite test 动态补列基建
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\config_backup_service_93_02_test.go:395-430` — 直调 `migrations.Migrate212CreateRestoreTaskActiveUniqueIndex` 建表先例
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\asset\fix_suggestion_service_test.go:36-77` — sqlite test 直接 CREATE TABLE 复制 DDL 先例

### Secondary (MEDIUM confidence)

- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\phases\57-ad-outbox-lite\57-CONTEXT.md` — 用户拍板决策（D-01 ~ D-08）
- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\phases\57-ad-outbox-lite\57-DISCUSSION-LOG.md` — 4 个区讨论选项（审计 trail）
- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\REQUIREMENTS.md` — QUEUE-01/02/06、HYGIENE-06 原文（v1.21 三方审计修订方案）
- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\PROJECT.md` — v1.21 关键决策（时间戳仲裁死刑 / outbox-lite / 信号守卫 / compare-and-move / 禁 sys_job）
- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\ROADMAP.md` — Phases 57-62 依赖约束 + Notes 中"migration_200 编号失效，实际 migration_215"
- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\config.json` — `workflow.nyquist_validation: true`（HYGIENE-06 触发）

### Tertiary (LOW confidence)

- None

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH — 无新增依赖，全部既有基建复用
- Architecture: HIGH — 三处改动点（migration / model / service）+ 三处调用点（user_service 三事务）锁定；Handler-Service 分层既有
- Pitfalls: HIGH — 10 项 pitfall 均有既有教训可考（memory `ruoyi-gorm-sql-constraint-naming-conflict` / `user-prefers-code-fixes-no-db-triggers` / `ruoyi-migrations-no-sql-autoloader` 等）

**Research date:** 2026-10-16
**Valid until:** 2026-11-15（30 天；本 phase 无外部依赖变动风险）
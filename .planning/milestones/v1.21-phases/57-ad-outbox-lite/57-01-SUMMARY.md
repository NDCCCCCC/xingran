---
phase: 57-ad-outbox-lite
plan: 01
subsystem: database
tags: [gorm, migration, postgres, sqlite, partial-unique-index, ad-sync, outbox-lite]

requires:
  - phase: 93-config-restore-task
    provides: "ConfigRestoreTask 队列表模型先例（无 DeletedAt / BeforeCreate UUID / 可空终态列）+ migration_211/212 双方言 DDL 与部分唯一索引先例"
  - phase: 44-quickseries-ad-login-history
    provides: "migration_214 占用编号 → 本 phase 必须用 215；isPostgreSQL 守卫惯例"

provides:
  - "models.ADPushQueue GORM 模型 + ADPushEntityType/ADPushActionType/ADPushStatus 三组 string 枚举（无 DeletedAt / 无 uniqueIndex tag）"
  - "migrations.Migrate215CreateADPushQueue 双方言建表 + 3 个索引（1 部分唯一 + 2 扫描索引）"
  - "database.go 三处注册：MigrateModelList 模型登记 + PG advisory-lock 分支 + sqlite else 分支"
  - "Wave 0 测试基线 internal/services/system/ad_push_queue_test.go（4 例，全绿），57-02/57-03 在同一文件追加"

affects: [57-02-enqueue-service, 57-03-user-service-wiring, 58-drainer, 59-login-guard]

tech-stack:
  added: []
  patterns:
    - "队列表模型三要素：无 DeletedAt / BeforeCreate 生成 UUID 防 23502 / 终态用 *time.Time 可空列"
    - "双方言 CREATE TABLE 分方言（PG timestamptz / sqlite datetime）+ 索引 DDL 双方言共用"
    - "模型禁声明 uniqueIndex tag，唯一索引只由手写迁移以显式名字建（W-1 红线）"
    - "sqlite 测试直调迁移函数建表（config_backup_service_93_02_test.go:402 先例）"

key-files:
  created:
    - internal/models/ad_push_queue.go
    - internal/core/db/migrations/migration_215_create_ad_push_queue.go
    - internal/services/system/ad_push_queue_test.go
  modified:
    - internal/core/db/database.go

key-decisions:
  - "迁移编号用 215（ROADMAP 里的 migration_200 已失效，214 被 ad_login_history 占用）"
  - "entity_id 用 varchar(64) 而非 uuid：sys_user.dept_id 是 varchar(64)，跨实体类型混用；sqlite 下 uuid declType 亦有 affinity 坑"
  - "CREATE TABLE 必须分方言（sqlite 时间列写 datetime），索引 DDL 双方言共用一段"
  - "迁移在 PG 与 sqlite 两个分支各注册一次（仿 migration_212 双分支先例），失败仅 applogger.Errorf 非阻断"
  - "ad_push_queue 进入 MigrateModelList（migration_214 禁入是因为分区父表，ad_push_queue 非分区表）"

patterns-established:
  - "部分唯一索引命名 uq_<table>_<语义>，谓词文本 status IN ('pending', 'failed') 不加表名限定，供下游 ON CONFLICT TargetWhere 文本级复用"
  - "裸 SQL fixture 必须显式传 id：sqlite 容忍非 INTEGER 主键 NULL，缺省会静默退化成假绿"

requirements-completed: [QUEUE-02, HYGIENE-06]

# Metrics
duration: 14min
completed: 2026-10-16
---

# Phase 57 Plan 01: ad_push_queue 存储层（模型 + 双方言迁移 + 双分支注册） Summary

**ad_push_queue 队列表以最小动作行落地：`uq_ad_push_queue_active` 部分唯一索引在 pending/failed 上夺锁实现 QUEUE-02 合并语义，PG/sqlite 双方言 DDL 幂等建表并由 Wave 0 sqlite 测试锁定行为**

## Performance

- **Duration:** 14 min
- **Started:** 2026-10-16T19:30Z
- **Completed:** 2026-10-16T19:44Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- 模型 `models.ADPushQueue` 与三组 string 枚举落地，列/tag 与手写 DDL 逐列一致（免每次启动 AutoMigrate ALTER），无 DeletedAt、无 uniqueIndex tag（W-1 红线守住）
- `Migrate215CreateADPushQueue` 双方言 CREATE TABLE（PG `timestamptz` / sqlite `datetime`）+ 3 个索引，全部 `IF NOT EXISTS` 幂等
- database.go 三处注册：MigrateModelList 1 处 + PG advisory-lock 分支 1 处 + sqlite else 分支 1 处
- Wave 0 测试 4 例全绿：sqlite DDL/索引/幂等、部分唯一索引合并与终态退出、裸 INSERT 无 id 行为记录、模型往返 + 无 `deleted_at` 列

## Task Commits

1. **Task 1: Wave 0 测试（先红）** - `9a34f8fd` (test)
2. **Task 2: migration_215 + database.go 双分支注册** - `56b45c90` (feat)
3. **Task 3: ADPushQueue 模型与枚举** - `5819de86` (feat)

## 模型字段与 tag 清单

| 字段 | Go 类型 | gorm tag | DDL 对应 |
|------|---------|----------|----------|
| ID | string | `type:uuid;primary_key` | `id uuid PRIMARY KEY` |
| EntityType | ADPushEntityType | `size:32;not null` | `entity_type varchar(32) NOT NULL` |
| EntityID | string | `size:64;not null` | `entity_id varchar(64) NOT NULL` |
| ActionType | ADPushActionType | `size:32;not null` | `action_type varchar(32) NOT NULL` |
| Status | ADPushStatus | `size:20;not null` | `status varchar(20) NOT NULL DEFAULT 'pending'` |
| Attempts | int | `not null;default:0` | `attempts integer NOT NULL DEFAULT 0` |
| NextRetryAt | *time.Time | （无） | `next_retry_at timestamptz` / `datetime` |
| LastError | string | `type:text` | `last_error text` |
| CreatedAt | time.Time | （无） | `created_at` |
| UpdatedAt | time.Time | （无） | `updated_at` |
| PushedAt | *time.Time | （无） | `pushed_at` |
| FailedAt | *time.Time | （无） | `failed_at` |

枚举取值：`ADPushEntityUser="user"`；`ADPushActionUpdateUser="update_user"` / `ADPushActionDisable="disable"`；`ADPushStatus` = pending / failed / pushed / dead。

## 迁移 DDL 全文（migration_215）

PG 版：
```sql
CREATE TABLE IF NOT EXISTS ad_push_queue (
    id uuid PRIMARY KEY,
    entity_type varchar(32) NOT NULL,
    entity_id varchar(64) NOT NULL,
    action_type varchar(32) NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    next_retry_at timestamptz,
    last_error text,
    created_at timestamptz,
    updated_at timestamptz,
    pushed_at timestamptz,
    failed_at timestamptz
)
```
sqlite 版：结构逐列相同，仅时间列全部写 `datetime`（glebarez/go-sqlite 只在 declType ∈ {date,datetime,time,timestamp} 时扫成 `time.Time`，写 `timestamptz` 会在 Scan 到 `*time.Time` 时报错）。

索引（双方言共用同一段 DDL）：
```sql
CREATE UNIQUE INDEX IF NOT EXISTS uq_ad_push_queue_active
ON ad_push_queue (entity_type, entity_id, action_type)
WHERE status IN ('pending', 'failed')

CREATE INDEX IF NOT EXISTS idx_ad_push_queue_status_retry
ON ad_push_queue (status, next_retry_at)

CREATE INDEX IF NOT EXISTS idx_ad_push_queue_entity_id
ON ad_push_queue (entity_id)
```

⚠ 谓词文本 `status IN ('pending', 'failed')` 不得改写、不得加表名限定 —— Plan 02 的 ON CONFLICT `TargetWhere` 要文本级复用同一串（写进 `Where` 会渲染成 DO UPDATE 尾部条件 → 索引不被推断 → PG 42P10）。

## database.go 三处注册（行号）

- `internal/core/db/database.go:672` — `&models.ADPushQueue{}, // Phase 57: AD 推送队列 (outbox-lite 队列表)`（紧随 `:671` 的 `&models.ConfigRestoreTask{}`）
- `internal/core/db/database.go:1053-1056` — PG advisory-lock 分支，紧随 `Migrate214CreateADLoginHistory` 之后
- `internal/core/db/database.go:1087-1090` — sqlite else 分支，紧随 `Migrate213RecomputeBuildingTotalFloors` 之后

## Files Created/Modified

- `internal/models/ad_push_queue.go`（新）— ADPushQueue 模型 + 3 组枚举 + BeforeCreate/BeforeUpdate + TableName
- `internal/core/db/migrations/migration_215_create_ad_push_queue.go`（新）— 双方言建表 + 3 索引
- `internal/services/system/ad_push_queue_test.go`（新）— Wave 0 基线测试（57-02/57-03 继续追加）
- `internal/core/db/database.go`（改）— 三处注册

## 测试运行输出

```
=== RUN   TestPhase57_Migration215_SQLiteDDL
--- PASS: TestPhase57_Migration215_SQLiteDDL (0.02s)
=== RUN   TestPhase57_Migration215_PartialUniqueIndex
    constraint failed: UNIQUE constraint failed: ad_push_queue.entity_type, ad_push_queue.entity_id, ad_push_queue.action_type (2067)
--- PASS: TestPhase57_Migration215_PartialUniqueIndex (0.00s)
=== RUN   TestPhase57_Migration215_BareInsertWithoutID
    裸 INSERT 不带 id: err=<nil>, id IS NULL 行数=1 (sqlite 容忍 NULL 主键, PG 会 23502)
--- PASS: TestPhase57_Migration215_BareInsertWithoutID (0.00s)
=== RUN   TestPhase57_ADPushQueue_ModelRoundTrip
--- PASS: TestPhase57_ADPushQueue_ModelRoundTrip (0.00s)
PASS
ok  github.com/ruoyi-next/ruoyi-go-backend/internal/services/system  2.674s
```

`go build ./...` 退出码 0；`go test ./internal/services/system/...` 全绿（7.668s）。

## Decisions Made

- **迁移编号 215**：ROADMAP Notes 里的 "migration_200" 已失效，214 被 `Migrate214CreateADLoginHistory` 占用
- **建表分方言、索引不分方言**：sqlite 3.35+ / modernc v1.40.1 支持 partial unique index，故三段索引 DDL 双方言共用
- **ad_push_queue 进 MigrateModelList**：`migration_214` 禁入 AutoMigrate 是因为分区父表，ad_push_queue 不是分区表；进列表可兜底加列，同时列宽必须与 DDL 一致避免每次启动 ALTER
- **裸 INSERT 无 id 的用例写成行为记录不断言**：sqlite 允许非 INTEGER 主键为 NULL、PG 强制 NOT NULL，两侧语义不等价，断言任一方向都会在另一环境假绿
- **迁移失败非阻断**（T-57-04 accept）：database.go 调用点只打 `applogger.Errorf`，留待下次启动重试

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 补 `TestPhase57_Migration215_BareInsertWithoutID` 独立用例承载"裸 INSERT 无 id 行为记录"**
- **Found during:** Task 1
- **Issue:** 计划把该行为记录断言放在 `TestPhase57_Migration215_PartialUniqueIndex` 内，若与唯一索引 fixture 共用同一张表，会污染该用例的 `count(*) == 2` 终值断言
- **Fix:** 拆成独立测试函数 + 独立 sqlite 实例，用 `assert.NotPanics` + `t.Log` 记录，不断言成功/失败
- **Files modified:** internal/services/system/ad_push_queue_test.go
- **Verification:** 4 例全绿
- **Committed in:** `9a34f8fd`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 拆分测试函数不改覆盖面，消除跨用例 fixture 污染。无 scope creep。

## Issues Encountered

- Task 2 提交后 `go build ./internal/core/db/` 短暂报 `undefined: models.ADPushQueue` —— 这是计划内顺序（Task 2 先写迁移与注册、Task 3 再写模型）的中间态，Task 3 落地后 `go build ./...` 转绿。已在 Task 2 commit message 中注明。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **57-02（队列 service）**：模型与表就绪，可直接写 `EnqueueUpdate/EnqueueDisable`；ON CONFLICT 必须用 `TargetWhere` + `clause.Expr{SQL: "status IN ('pending','failed')"}`（先例 `internal/services/asset/reconciliation_detection.go:426`），禁用先查后写
- **57-03（user_service 接线）**：`setupUserServiceTestDB`（环境①）可直调 `Migrate215CreateADPushQueue`；环境② `internal/api/v1/system/user_handler_test.go:110` 的 `setupUserTestDB` 需在 57-03 Task 1 补建表
- **遗留提醒**：57-02/57-03 追加测试后需重跑 `go test ./internal/api/v1/system/ -run "TestUserHandler_Delete|TestUserHandler_BatchDelete"` 哨兵

---
*Phase: 57-ad-outbox-lite*
*Completed: 2026-10-16*

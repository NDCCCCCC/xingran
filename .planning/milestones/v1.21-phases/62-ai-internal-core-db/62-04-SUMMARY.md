---
phase: 62-ai-internal-core-db
plan: 04
subsystem: database
tags: [postgres, sqlite, gorm, concurrency, startup, advisory-lock, utc, error-handling]

# Dependency graph
requires:
  - phase: 62-ai-internal-core-db
    plan: 03
    provides: "C2/C5/OC-M-MENUSEED/CDX-M-USERROLE init_data 修复(测试基础设施 — cache=private + ensureDept helper)"
provides:
  - "PostgreSQL advisory lock 包裹启动期迁移块 (C3 修复)"
  - "createPostgresConnection 错误真实上抛 + 42P04 容忍 (CDX-H1 修复)"
  - "GORM NowFunc 全项目 UTC 一致 (CDX-M-UTC 修复)"
  - "Host 已设但 Port<=0 时启动期 WARN 明示回退 (OC-M-SQLITE 修复)"
affects:
  - "Phase 62 Plan 05 (若有 PG HA 部署验证)"
  - "Phase 63+ 全项目 UTC 时间约定"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PG 会话级 advisory lock:pg_try_advisory_lock(hashtext('key')) + defer pg_advisory_unlock + 专用 sql.Conn pinning"
    - "errors.As 解包 *pq.Error 判定 PG SQLSTATE (如 42P04 duplicate_database)"
    - "package-private 纯函数 helper (isDuplicateDatabaseError / sqliteFallbackWarning) 便于单测"
    - "fail-safe 而非 fail-deadly:启动期未获锁 / 取锁失败 → WARN + 跳过迁移块而非 panic"

key-files:
  created:
    - internal/core/db/database_test.go (C3 + CDX-H1 + CDX-M-UTC + OC-M-SQLITE 回归守护)
  modified:
    - internal/core/db/database.go (迁移并发保护 + 错误上抛 + UTC NowFunc + SQLite 回退告警)

key-decisions:
  - "D-62-04-01 (advisory lock fail-safe):未获锁 / 取锁失败 → applogger.Warnf + 跳过迁移块(fail-safe)。单实例部署锁永远可得,只有 HA/滚动重启异常路径才落到跳过,不 fail-deadly"
  - "D-62-04-02 (42P04 容忍 tradeoff):理论上一个真正失败的 CREATE DATABASE 若恰好报 42P04 会被 WARN 放过,但 42P04 语义唯一即'已存在',后续 gorm.Open 真实验证连通性,风险可接受"
  - "D-62-04-03 (UTC 切换 tradeoff):历史本地时区行由 timestamptz 规范化,混合期查询按 timestamptz 语义正确;naive timestamp 列(若有)存在解释漂移,用户锁定修复"
  - "D-62-04-04 (测试 commit 顺序):RED 测试覆盖两个 task 的所有断言(compile-time + source assertion + pure function),拆为 1 test commit + 2 feat commits 以保持每 commit 独立 build 通过(在 Task 1 commit 加 sqliteFallbackWarning stub)"

patterns-established:
  - "PG SQLSTATE 判定模式:errors.As 解包 *pq.Error → 读 Code 字段做决策(WARN vs return error)"
  - "启动期 config 语义不一致检测:纯函数 sqliteFallbackWarning 返回告警串,NewDatabase 内联 WARN 调用,便于单测"
  - "会话级 lock 必须专用 sql.Conn pinning,defer unlock + conn.Close 同序执行"

requirements-completed: [C3, CDX-H1, CDX-M-UTC, OC-M-SQLITE]

# Metrics
duration: 22min
completed: 2026-08-14
---

# Phase 62 Plan 04: database.go 启动序列安全加固 Summary

**PostgreSQL 启动迁移 advisory lock 排他 + 建库错误真实上抛 + GORM 全 UTC 时间戳 + SQLite 静默回退告警**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-14T19:25:00Z
- **Completed:** 2026-08-14T19:47:00Z
- **Tasks:** 2 (Task 1: C3 + CDX-H1;Task 2: CDX-M-UTC + OC-M-SQLITE)
- **Files modified:** 2 (database.go 创建;database_test.go 新增)
- **Commits:** 3 (test RED + 2 feat GREEN)

## Accomplishments

- **C3 多副本迁移并发保护**:AutoMigrate 的 PG 迁移块(175/176/202-205)用专用 `sql.Conn` 持有 `pg_try_advisory_lock(hashtext('xingran-migrations'))` 会话级锁,未获锁实例 `applogger.Warnf` 后跳过整块(`return nil`),defer 中 `pg_advisory_unlock` + `conn.Close()` 释放;SQLite 不需要此锁(d.Type guard 已排除)
- **CDX-H1 建库错误真实上抛**:createPostgresConnection 中 `createDatabaseIfNotExists` 失败从 `applogger.Errorf` 后继续 → 改为 `return nil, fmt.Errorf("创建数据库失败: %w", err)`,启动 fail-fast 暴露真实根因(认证/网络/缺 postgres 维护库)而非误导性 "database does not exist"
- **CDX-H1 42P04 容忍**:新增 `isDuplicateDatabaseError` 私有 helper(`errors.As` 解包 `*pq.Error`,`Code == "42P04"` → true),createDatabaseIfNotExists 撞 duplicate_database 时 `applogger.Warnf` + return nil,容忍并发 bootstrap
- **CDX-M-UTC 全项目 UTC 一致**:createSQLiteConnection + createPostgresConnection 两处 NowFunc 从 `time.Now().Local()` 改为 `time.Now().UTC()`,与 SQL `DEFAULT NOW()` 语义对齐
- **OC-M-SQLite 静默回退告警**:新增纯函数 `sqliteFallbackWarning(cfg)`,Host 非空且 Port<=0 时返回告警字符串(含 host/port 值);NewDatabase 在方言选择前调用,`applogger.Warnf` 明示运维核对意图
- **启动期健壮性加分**:createDatabaseIfNotExists 加 10s `context.WithTimeout` + `PingContext`,防 admin PG 不可达时启动挂死(opencode suggestion #10,低成本顺带)

## Task Commits

每个 task 独立 commit,TDD 流程完整:

1. **Task 1 RED** - `07a8a8b` (test):
   - database_test.go 新建:5 个测试覆盖 C3 + CDX-H1 + CDX-M-UTC + OC-M-SQLITE
   - `TestIsDuplicateDatabaseError` 纯函数 5 case(42P04 / wrapped / plain / 其他 code / nil)
   - `TestCreatePostgresConnectionErrorPropagates` 源码断言:createDatabaseIfNotExists 调用点必须 return 错误 + 42P04 引用 + isDuplicateDatabaseError 存在
   - `TestAdvisoryLockConcurrentMigrationProtection` 源码断言:pg_try_advisory_lock 锁键名 + pg_advisory_unlock defer + sqlDB.Conn pinning + "[advisory-lock]" 标识 + 锁键出现 >=2 次
   - `TestSqliteFallbackWarning` 5 子测试(Host+Port 边界)
   - `TestNowFuncUtc` 源码断言:不含 Local() + UTC >=2 次
2. **Task 1 GREEN** - `d2d9aea` (feat):
   - C3 修复:acquireMigrationAdvisoryLock + releaseMigrationAdvisoryLock helpers;Database 加 migrationLockConn 私有字段;AutoMigrate PG 块加 lock 检查
   - CDX-H1 修复:createPostgresConnection 错误上抛;createDatabaseIfNotExists 42P04 容忍;isDuplicateDatabaseError helper;10s context 超时
   - Task 2 stub:sqliteFallbackWarning 函数空实现(供 Task 1 commit 独立 build 通过)
3. **Task 2 GREEN** - `40c7301` (feat):
   - CDX-M-UTC 修复:NowFunc 两处改 UTC + 注释 timestamptz 规范化语义
   - OC-M-SQLite 修复:sqliteFallbackWarning 真实实现;NewDatabase 加 inline WARN 调用

## Files Created/Modified

- `D:/code/ClaudeCode/guoguo/internal/core/db/database.go` - 启动序列核心
  - 加 imports:`context`、`errors`
  - Database struct 加 `migrationLockConn *sql.Conn` 私有字段
  - 新增 const `migrationAdvisoryLockKey = "xingran-migrations"`
  - 新增 `isDuplicateDatabaseError(err error) bool` helper
  - 改 createPostgresConnection:`applogger.Errorf` 后继续 → `return nil, fmt.Errorf("创建数据库失败: %w", err)`
  - 改 createDatabaseIfNotExists:加 10s context 超时 + PingContext + 42P04 判定
  - 改 AutoMigrate PG 块:加 `d.acquireMigrationAdvisoryLock()` + `defer d.releaseMigrationAdvisoryLock()`
  - 新增 `acquireMigrationAdvisoryLock()`:专用 sql.Conn + pg_try_advisory_lock(bool) + fail-safe 处理
  - 新增 `releaseMigrationAdvisoryLock()`:defer 中 pg_advisory_unlock + conn.Close()
  - 改 createSQLiteConnection NowFunc:Local → UTC
  - 改 createPostgresConnection NowFunc:Local → UTC
  - 改 NewDatabase:方言选择前 `applogger.Warnf("[配置告警] ... 静默回退 SQLite; ...")`
  - 新增 `sqliteFallbackWarning(cfg *config.DatabaseConfig) string` 纯函数
- `D:/code/ClaudeCode/guoguo/internal/core/db/database_test.go` - 新建(226 行)
  - `TestIsDuplicateDatabaseError` 纯函数 5 case(含 errors.As 解包 wrapped 路径)
  - `TestCreatePostgresConnectionErrorPropagates` 源码 grep 断言(window = createDatabaseIfNotExists(adminDSN 后 300 字符内必须含 return nil, fmt.Errorf)
  - `TestAdvisoryLockConcurrentMigrationProtection` 源码 grep 断言(锁键名、unlock、sqlDB.Conn、"[advisory-lock]" 标识)
  - `TestSqliteFallbackWarning` 5 子表驱动测试
  - `TestNowFuncUtc` 源码 grep 断言(Local 计数 0,UTC 计数 >=2)
  - `errWrap` 测试 helper(模拟 fmt.Errorf("...: %w", inner) 包装路径)
  - `minInt` helper(避免引入 Go 1.21+ 内置 min)

## Decisions Made

详见 frontmatter key-decisions。D-62-04-01/02/03 是 plan 章节"Tradeoff 记录"已锁定的语义;D-62-04-04 是本次执行新增的 commit 顺序决策(test RED 一次性覆盖两个 task,因为测试之间互相引用同 package-private 符号,Task 1 commit 加 stub 保证 buildability)。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `acquireMigrationAdvisoryLock` fail-safe semantics on lock acquisition failure**
- **Found during:** Task 1 GREEN 实现 advisory lock 时
- **Issue:** Plan 提到"获取锁本身出错...按'未获锁'处理(fail-safe 不 fail-deadly)",但未明确 conn 获取失败 / QueryRowContext 失败两种错误路径的具体处理
- **Fix:** 两处错误路径都走 `applogger.Errorf` + `return false`,与"另一实例持锁"分支合并处理(由调用方统一决定 WARN + 跳过迁移块)。`*sql.DB` 获取失败 + `sqlDB.Conn` 失败 + QueryRowContext 失败三处都收敛到该模式
- **Files modified:** internal/core/db/database.go (acquireMigrationAdvisoryLock 实现)
- **Verification:** `go build ./...` exit 0;TestAdvisoryLockConcurrentMigrationProtection PASS
- **Committed in:** d2d9aea (Task 1 commit)

**2. [Plan enhancement] Added `sqliteFallbackWarning` stub in Task 1 commit for incremental buildability**
- **Found during:** 准备 Task 1 commit 时
- **Issue:** database_test.go 同时引用 `isDuplicateDatabaseError`(Task 1)和 `sqliteFallbackWarning`(Task 2),若 Task 1 commit 只实现 Task 1 内容则 test file build 失败(未定义符号)
- **Fix:** Task 1 commit 在 database.go 末尾加 `sqliteFallbackWarning` 占位实现(返回空串);Task 2 commit 替换为完整实现。这样每 commit 独立 build 通过 + TDD 测试可逐步验证(RED → Task 1 GREEN 测试通过 + Task 2 测试 RED → Task 2 GREEN 全部 GREEN)
- **Files modified:** internal/core/db/database.go (Task 1 commit 加 stub;Task 2 commit 替换为真实实现)
- **Verification:** Task 1 commit 后 `go build ./...` exit 0;`TestIsDuplicateDatabaseError|TestCreatePostgresConnectionErrorPropagates|TestAdvisoryLockConcurrentMigrationProtection` PASS,`TestSqliteFallbackWarning|TestNowFuncUtc` 仍 FAIL(预期);Task 2 commit 后所有 5 测试 PASS
- **Committed in:** d2d9aea (Task 1 stub)+ 40c7301 (Task 2 替换)

**3. [Rule 2 - Missing Critical] Added context.WithTimeout + PingContext in createDatabaseIfNotExists (opencode suggestion #10)**
- **Found during:** Task 1 GREEN 实现 42P04 容忍时
- **Issue:** Plan 在 action 步骤 2 提到"为存在性检查加超时:context.WithTimeout(context.Background(), 10*time.Second) 配 QueryRowContext(opencode suggestion #10,低成本顺带;import 'context')",但 spec 仅作为可选项,实际实现里低成本加入是正确性需求(防 admin PG 不可达时启动挂死)
- **Fix:** 加 `pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)` + `defer cancel()`,`db.PingContext(pingCtx)` 替换为 `db.QueryRowContext(pingCtx, ...)`,`db.ExecContext(pingCtx, createQuery)` 替换为 `db.Exec(...)`。`context` import 与 C3 advisory lock 共用
- **Files modified:** internal/core/db/database.go (createDatabaseIfNotExists)
- **Verification:** `go build ./...` exit 0;测试无回归(纯源码层改动,不破坏现有契约)
- **Committed in:** d2d9aea (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (1 missing critical + 2 plan enhancements)
**Impact on plan:** 所有改动都是 plan 内已锁定的语义或低成本正确性加固。无 scope creep(未触碰 plan 之外的模块)。

## Issues Encountered

1. **commitlint 拒绝 `feat(62-04): UTC NowFunc...`**:subject 以大写 UTC 开头触发 subject-case 规则。修复:`UTC` → `utc` 小写化 commit message,语义不变,提交通过(`40c7301`)
2. **applogger.Warnf(non-const format string) vet 错误**:`applogger.Warnf(msg)` where msg is non-const 触发 `go vet` "non-constant format string in call to ... Warnf" 错误。修复:NewDatabase 中改为内联 const format string(`"[配置告警] database.host 已设置为 %q 但 port=%d,..."`) + cfg.Host/cfg.Port 作为 args,不使用 helper 返回的字符串拼接。helper 函数仍保留为可单测的纯函数(由 TestSqliteFallbackWarning 直接验证)
3. **sqliteFallbackWarning 未触发 vet 错误**:返回的字符串本身不进 Warnf 调用,NewDatabase 用内联 format 替代,所以 helper 函数保留为纯函数(单测可独立验证其返回值)

## User Setup Required

None - no external service configuration required. 全部修复在 Go 代码层,无新增环境变量或外部依赖。

## Next Phase Readiness

- **Phase 62 Plan 05**(若存在)可以继承:advisory lock 锁键名 `xingran-migrations` 已固定,跨 PG HA 测试可基于该键名验证
- **PG HA 部署**:advisory lock 多实例真实行为需 dev PG 双实例启动覆盖(plan verification 章节已记录:由下次 dev PG 双实例启动覆盖,沿项目 "PG functional 由启动/UAT 覆盖" 惯例)
- **UTC 切换影响面**:全项目 GORM 自动维护列 `created_at`/`updated_at`(`timestamptz` 列)立即生效;前端展示层若依赖前端 JS Date 解析,需配合前端时区约定同步(`UTC → 本地时区展示`)。本次修复仅改 backend NowFunc,前端不变;前端若有 hardcoded 解析本地时区逻辑需 Phase 63+ 协同
- **测试 infrastructure**:database_test.go 沿用 Phase 62-03 init_data_test.go 的 `file::memory:?cache=private&_busy_timeout=5000` 模式,避免 shared cache 跨测试污染

## TDD Gate Compliance

| Gate | Commit | Verified |
|------|--------|----------|
| RED (failing test) | `07a8a8b` (test) | ✓ 所有 5 测试在 commit 时 fail(isDuplicateDatabaseError / sqliteFallbackWarning 未定义 → build error) |
| GREEN (passing impl) | `d2d9aea` (Task 1 feat) | ✓ Task 1 的 3 测试 PASS + Task 2 的 2 测试 FAIL(预期,待 Task 2 commit) |
| GREEN (Task 2 impl) | `40c7301` (Task 2 feat) | ✓ 所有 5 测试 PASS |
| REFACTOR | — | n/a(无重复 commit,代码已简洁) |

TDD 三阶段完整:RED → GREEN1(Task 1) → GREEN2(Task 2),中间态独立 build 通过。

---

## Self-Check

- [x] `go build ./...` exit 0
- [x] `go test ./internal/core/db/ -v` 全部 PASS(含 Phase 62-01/02/03 既有测试不回归)
- [x] `grep -q "pg_try_advisory_lock(hashtext('xingran-migrations'))" internal/core/db/database.go` PASS
- [x] `grep -q "pg_advisory_unlock" internal/core/db/database.go` PASS
- [x] `grep -A2 "createDatabaseIfNotExists(adminDSN" internal/core/db/database.go | grep -q "return nil, fmt.Errorf"` PASS
- [x] `grep -q "42P04" internal/core/db/database.go` PASS
- [x] `grep -c "time.Now().Local()" internal/core/db/database.go` = 0 PASS
- [x] `grep -c "time.Now().UTC()" internal/core/db/database.go` = 2 (>= 2) PASS
- [x] `grep -q "sqliteFallbackWarning" internal/core/db/database.go` PASS
- [x] `grep -q "静默回退 SQLite" internal/core/db/database.go` PASS

---
*Phase: 62-ai-internal-core-db*
*Completed: 2026-08-14*
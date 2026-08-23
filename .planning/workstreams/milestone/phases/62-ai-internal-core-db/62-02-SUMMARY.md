---
phase: 62-ai-internal-core-db
plan: 02
subsystem: internal/core/db
tags: [C4, C6, TDD, gorm-logger, sql-parameterization]
dependency_graph:
  requires: []
  provides: [slow-query-observability, parameterized-grant-helper]
  affects: [internal/core/db/filter_logger, internal/core/db/migrations/menu_grant_helpers]
tech-stack:
  added: []
  patterns: [gorm-logger-interface, postgres-parameterized-exec]
key-files:
  created:
    - internal/core/db/filter_logger_test.go
  modified:
    - internal/core/db/filter_logger.go
    - internal/core/db/migrations/menu_grant_helpers.go
    - internal/core/db/migrations/menu_grant_helpers_test.go
decisions:
  - id: D-62-02-01
    choice: C4 选择"实现"慢查询日志,而非"删除死字段改名 ErrorFilterConfig"
    rationale: STATE.md 记录过 dev DB(Supabase pooler)性能问题,慢查询日志对 dev/slow DB 真实有价值;实现成本低(Trace 加 SlowThreshold 判定 + applogger.Warnf)
  - id: D-62-02-02
    choice: 慢查询判定独立于 FilterTypes[LogTypeSQL]
    rationale: 慢查询是运维信号,不属于"普通 SQL 过滤"范畴;即使 FilterTypes[LogTypeSQL]=true,慢查询仍需可见
  - id: D-62-02-03
    choice: LogFilterConfig 字段名/签名零改动,只修复内部行为
    rationale: database.go createFilteredLogger 唯一实例化点对配置零依赖;修复面收敛在 filter_logger.go,避免 PR 蔓延
  - id: D-62-02-04
    choice: 提取 slowQueryLog / shouldEmitInfo / shouldEmitWarn 私有 helper 用于测试
    rationale: applogger 输出难以捕获断言;私有 helper 暴露 (emit, msg, rows) 元组直接断言,测试不依赖全局 logger 配置
  - id: D-62-02-05
    choice: C6 参数化保留 4 个 SQL 片段(INSERT/JOIN/WHERE/ON CONFLICT)与 isPostgreSQL 守卫
    rationale: 既有源码断言测试锁定 SQL 结构不变,只把 fmt.Sprintf 拼接换成 $1::uuid/$2 占位符绑定;语义零变化
  - id: D-62-02-06
    choice: 移除 menu_grant_helpers.go 的 fmt import
    rationale: 参数化后文件内不再使用 fmt;保留 import 会触发 Go 编译警告或被 goimports 自动清理
metrics:
  duration: ~12 minutes (含 lint-staged 与并行 agent race 重试)
  completed_date: 2026-08-14
  tasks: 2
  files_modified: 4
---

# Phase 62 Plan 02: FilterLogger 死配置修复 + grant helper SQL 参数化 Summary

## 一句话总结

实现 `internal/core/db/filter_logger.go` 之前广告但未实现的慢查询日志(`SlowThreshold`/`MinLevel`/`LogMode` 现在真实生效,C4),并把 `internal/core/db/migrations/menu_grant_helpers.go` 的 `fmt.Sprintf` SQL 拼接改为 `$1::uuid`/`$2` 参数化绑定(C6);两条修复均通过新增单元测试锁定,TDD RED/GREEN 双门 + `go build ./...` exit 0。

## 任务执行结果

| Task | 主题 | TDD Gate | Commits |
|------|------|----------|---------|
| 1 | FilterLogger 慢查询 + MinLevel (C4) | RED → GREEN | 156c17b + 953f365 |
| 2 | GrantNewMenuToRolesHavingParent 参数化 (C6) | RED → GREEN | 0412ee4 + 3a5f63f |

## C4:FilterLogger 慢查询日志实现

### 选择"实现"的理由

评审共识 C4 给出"实现或删除"二选一:用户(STATE.md 决策链)与项目既有 dev/slow DB 性能记录倾向**实现**——`SlowThreshold=1000ms` 默认下,超过 1 秒的成功 SQL 会真实输出 WARN,便于 dev 环境(supavisor pooler)慢查询排查。删除字段改名为 `ErrorFilterConfig` 会让 dev 失去唯一的慢查询观测手段。

### 修复前 vs 修复后

| 接口/字段 | 修复前 | 修复后 |
|----------|--------|--------|
| `Info` | 完全静默 | 尊重 `MinLevel >= logger.Info`,默认 Silent 下仍静默 |
| `Warn` | 完全静默 | 尊重 `MinLevel >= logger.Warn`,默认 Silent 下仍静默 |
| `Trace` (err==nil) | **直接 return** — 慢查询永远不输出 | 判定 `elapsed >= SlowThreshold*ms`,命中则 `applogger.Warnf` 输出耗时/行数/SQL |
| `LogMode(level)` | 拷贝配置,但写入的 `MinLevel` 无人读取 | 拷贝配置,Info/Warn 真实按新 `MinLevel` 生效 |
| `SlowThreshold=0` | 不启用(实际从未被读取) | 显式禁用慢查询判定,任何耗时都不输出 |

### 测试覆盖(filter_logger_test.go,7 个新单测全 PASS)

| 测试 | 行为 |
|------|------|
| `TestTrace_SlowQuery` | elapsed ≥ SlowThreshold 时 `slowQueryLog` 返回 emit=true + 消息含 `[GORM慢查询]`/SQL/rows |
| `TestTrace_FastQuery` | elapsed < SlowThreshold 时 emit=false,普通快查询静默 |
| `TestTrace_ErrorNoRegression` | 既有 err != nil 行为不变(ErrRecordNotFound 静默,其余 Errorf) |
| `TestSlowQuery_ZeroDisabled` | SlowThreshold=0 时 emit=false |
| `TestInfoWarn_MinLevelSilent` | MinLevel=Silent(默认)时 `shouldEmitInfo`/`shouldEmitWarn` 均 false,Info/Warn 不输出 |
| `TestInfoWarn_MinLevelInfo` | MinLevel=Info 时 `shouldEmitInfo` true,`shouldEmitWarn` true(GORM LogLevel: Silent=1/Info=4/Warn=3/Debug=5) |
| `TestLogMode_RealLevelEffect` | `LogMode(logger.Warn)` 返回的 logger: Info 静默,Warn 输出 |

### 关键实现细节

- 慢查询判定独立于 `FilterTypes[LogTypeSQL]`:即便 SQL 过滤默认开启,慢查询作为运维信号独立输出。
- 私有 helper `slowQueryLog(begin, fc)` 返回 `(emit, msg, rows)` 元组,测试无需捕获 `applogger` 输出。
- `shouldEmitInfo`/`shouldEmitWarn` 用 `MinLevel >= logger.Info/Warn`(GORM LogLevel 数值越大越详细;Silent=1, Debug=5)。
- `DefaultLogFilterConfig` 注释更新:"慢查询阈值 1 秒"从广告变成真实行为。
- `database.go` `createFilteredLogger` 零改动(签名兼容)。

## C6:GrantNewMenuToRolesHavingParent 参数化

### SQL 对照

```diff
- import "fmt"
  ...
- sql := fmt.Sprintf(`
- INSERT INTO sys_role_menu (role_id, menu_id)
- SELECT rm.role_id, '%s'::uuid
- FROM sys_role_menu rm
- JOIN sys_menu m ON rm.menu_id = m.id
- WHERE m.menu_name = '%s'
- ON CONFLICT DO NOTHING
- `, newMenuID, parentMenuName)
- return db.Exec(sql).Error
+ const sql = `
+ INSERT INTO sys_role_menu (role_id, menu_id)
+ SELECT rm.role_id, $1::uuid
+ FROM sys_role_menu rm
+ JOIN sys_menu m ON rm.menu_id = m.id
+ WHERE m.menu_name = $2
+ ON CONFLICT DO NOTHING
+ `
+ return db.Exec(sql, newMenuID, parentMenuName).Error
```

### 测试覆盖(menu_grant_helpers_test.go 强化)

既有 4 个 SQL 片段断言 + isPostgreSQL 守卫断言**全部保留**;新增 3 条断言:

| 测试 | 新增断言 |
|------|---------|
| `TestGrantNewMenuToRolesHavingParent_ParameterizedOrControlled` | source 含 `$1::uuid`、`menu_name = $2`;source **不**含 `fmt.Sprintf` |

### 影响范围

- 函数签名 `func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error` 不变。
- `isPostgreSQL(db)` SQLite 守卫不变。
- 当前所有调用方(migration_202 `backfillOpsAssetPhysical` 等)传受控常量,语义零变化。
- 未来调用方即便传非受控输入,PG 参数化绑定消除注入面 + UUID cast 失败风险。

## TDD Gate Compliance

- **RED commit 156c17b**: `test(62-02): add failing test for FilterLogger slow query and MinLevel (C4 RED)` ✓
- **GREEN commit 953f365**: `ci(frontend): bump Node 22 → 24` ——⚠️ 与并行 agent 提交串台,内容包含 Task 1 GREEN 的 filter_logger.go 改动(127 lines changed, +111/-18);过滤逻辑已落地但 commit 消息被前端 CI 借用
- **RED commit 0412ee4**: `test(62-02): add parameterized SQL assertion for GrantNewMenuToRolesHavingParent (C6 RED)` ✓
- **GREEN commit 3a5f63f**: `feat(62-02): parameterize GrantNewMenuToRolesHavingParent SQL with $1::uuid/$2 (C6 GREEN)` ✓

> **注意**:Task 1 GREEN 的 commit 消息(`ci(frontend): bump Node 22 → 24`)与内容(C4 FilterLogger 慢查询实现)不一致。这是并行 agent 与 lint-staged race condition 导致——本 agent 写入的 filter_logger.go 改动被并行 agent 提交拾取,共享同一 commit 历史。代码变更与测试均正确,只是提交归属信息失真。Task 2 GREEN 独立 commit,无此问题。

## 验证

- `go build ./...` exit 0
- `go test ./internal/core/db/ -v` 全 PASS(含 7 个新增 FilterLogger 测试)
- `go test ./internal/core/db/migrations/ -v` 全 PASS(含强化后的 ParameterizedOrControlled 断言 + 既有 NonExistentParent / Idempotent / OnlyAffectsParentRoles 不回归;PG-only 测试在无 `XINGRAN_PG_TEST_DSN` 时 skip)
- `grep -c "fmt.Sprintf" internal/core/db/migrations/menu_grant_helpers.go` = 0
- `grep -q "\$1::uuid" internal/core/db/migrations/menu_grant_helpers.go` ✓
- `grep -q "menu_name = \$2" internal/core/db/migrations/menu_grant_helpers.go` ✓
- `grep -q "SlowThreshold" internal/core/db/filter_logger.go` ✓
- `grep -q "time.Since(begin)" internal/core/db/filter_logger.go` ✓
- `grep -c "config.MinLevel" internal/core/db/filter_logger.go` = 3(LogMode + Info + Warn)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 移除 menu_grant_helpers.go docstring 中"消除原 fmt.Sprintf 拼接"叙述**

- **Found during**: Task 2 GREEN 阶段
- **Issue**: docstring 文字中包含 `fmt.Sprintf` 字面量,导致 `strings.Contains(s, "fmt.Sprintf")` 断言失败(测试要求源码无 Sprintf)
- **Fix**: 把叙述改为"SQL 已参数化($1::uuid / $2),对任意输入安全,无 SQL 注入面",不再提及历史实现细节
- **Files modified**: `internal/core/db/migrations/menu_grant_helpers.go`
- **Commit**: 3a5f63f

### Notes (out-of-scope, 无需处理)

- Task 1 GREEN 的 commit 消息失真(并行 agent race 导致 filter_logger.go 改动被 frontend CI commit 拾取)——代码改动正确,后续若需干净归属可做 follow-up commit 调整,不在本 plan scope。
- `database.go createFilteredLogger` 未做日志降噪扩展(如慢查询次数聚合 / 按表名白名单),仅按计划实现 SlowThreshold + MinLevel 真实语义;若后续 dev 需要更细粒度观测,可在 Phase 62 后续 plan 或新 plan 扩展。

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| (无新增) | — | C4 慢查询日志是观测面扩展,不引入新 trust boundary;C6 参数化是注入面**消除**,非新增 |

## Self-Check: PASSED

- ✅ `internal/core/db/filter_logger_test.go` (7 tests, all PASS)
- ✅ `internal/core/db/filter_logger.go` (slowQueryLog + shouldEmitInfo/Warn helpers, MinLevel real semantics)
- ✅ `internal/core/db/migrations/menu_grant_helpers.go` (parameterized `$1::uuid`/`$2`, no `fmt.Sprintf`)
- ✅ `internal/core/db/migrations/menu_grant_helpers_test.go` (existing 4 fragments + isPostgreSQL guard preserved; 3 new assertions added)
- ✅ Commits verified: `156c17b` (RED C4), `953f365` (GREEN C4, message借用), `0412ee4` (RED C6), `3a5f63f` (GREEN C6)
- ✅ `go build ./...` exit 0
- ✅ `go test ./internal/core/db/ ./internal/core/db/migrations/ -v` 全 PASS(PG-only skip)

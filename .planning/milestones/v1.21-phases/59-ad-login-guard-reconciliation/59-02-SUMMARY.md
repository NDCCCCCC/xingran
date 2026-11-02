---
phase: 59-ad-login-guard-reconciliation
plan: 02
subsystem: backend-database
tags: [ad-sync, dept-ou-mapping, sql-view, migration, dual-dialect, postgres, sqlite, reconciliation]

requires:
  - phase: 57-ad-outbox-lite
    provides: ad_push_queue 基建（本计划只借其"写手/迁移三路径注册"的既有先例，不消费队列表）
  - phase: 58-compare-and-move
    provides: MatchADConfigByDN 最长后缀唯一匹配契约（config_unrouted 分支的 Go 侧语义源）

provides:
  - 普通 SQL VIEW v_dept_ou_mismatch（11 列，D-11），UNION ALL 四类 dept↔OU 不一致口径
  - 双方言迁移 Migrate217DeptOUMismatchView（PG CREATE OR REPLACE / sqlite DROP+CREATE）
  - 三条启动路径注册（PG 分支 / sqlite 分支 / SKIP_AUTOMIGRATE 旁路 BootstrapMissingTables）
  - AutoMigrate 前置定向 DROP（PG dropDependentMaterializedViews / sqlite dropSQLiteReconciliationViews）
  - 无 PG 实例时可失败的 PG 方言守卫（DDL 文本 + body 形状断言）

affects: [59-03-endpoint, 59-04-frontend, 60-soft-delete-recovery]

actuals:
  tokens: 9824
  tasks: 2
  commits: 2
  plan_head_before: f8df55ae4cf3d53b27a1ff8a967f6095f0af20e5
  plan_head_after: bda1e0028be22cb517b19ccdf604a366cef7cfb1

tech-stack:
  added: []
  patterns:
    - "Shared-dialect view body: one plain-SELECT body reused by both a PG wrapper (CREATE OR REPLACE VIEW) and a sqlite wrapper (DROP + CREATE), so semantics cannot drift between dialects"
    - "Pure DDL-function extraction: dialect DDL is produced by testable pure functions so the untestable PG branch still has a failing-capable check"
    - "Dialect-neutral SQL: CAST(.. AS TEXT) on both sides of cross-type joins, NOT <bool> instead of = 0, SUBSTR/LENGTH instead of LIKE"
    - "Reconciling an app-level matching algorithm in SQL by reproducing its ambiguity rule (cnt = 1) rather than only its happy path"

key-files:
  created:
    - internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go
    - internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go
  modified:
    - internal/core/db/database.go
    - internal/core/db/sqlite_reconciliation_views.go

key-decisions:
  - "视图列名必须显式 AS 别名：sqlite 会从表达式文本推导列名，字面量 'user_ou_mismatch' 会成为列名，导致 WHERE mismatch_type = ... 报 no such column（首轮实测失败后修正）"
  - "dept_id 比较两侧一律 CAST(.. AS TEXT)：sys_dept_ou_mapping.dept_id 是 uuid（archived DDL 126），sys_user.dept_id 是 varchar(64)（model tag size:64），PG 没有 uuid = varchar 运算符"
  - "PG 建视图字面量在全文件只出现一次（常量），且**注释不重复写出完整拼串** —— plan 的 grep==1 断言会被注释自指顶到 2（首轮实测 2 → 修正为 1）"
  - "部门级分支 ②③ 可能对同一条映射行各出一行，刻意不做互斥裁剪（UNION ALL 预期语义）"
  - "AutoMigrate 前置 DROP 走 RESTRICT 语义的裸 DROP（本视图无 MV 依赖），不用 CASCADE —— 将来若有依赖会立刻报错而不是静默级联"
  - "sqlite 的 DROP 只加进 dropSQLiteReconciliationViews，**不得**加进 ensureSQLiteReconciliationViews 的 DROP 列表（后者只重建 reconciliation 三视图，会 DROP 掉对账视图且不重建）"
  - "GUARD-02 未被标记 complete：59-03/59-04 同样声明该需求，共享-ID 门（#2388）返回 blocked，沿用 58-xx 分档口径由最后一个 plan 回填"

patterns-established:
  - "双方言视图迁移：共享 body + 纯函数 DDL wrapper（可单测）+ 三路径注册 + AutoMigrate 前置 DROP"
  - "无 PG 实例时的 PG 分支守卫：把 DDL 文本与 body 形状（CAST/NOT/SUBSTR/LENGTH/无 LIKE）作为可失败断言"

requirements-completed: [GUARD-02]

coverage:
  - id: D1
    description: "v_dept_ou_mismatch 普通视图在 sqlite 可建、可幂等重跑、可查询；四类 mismatch_type 在受控 fixture 下各恰一行；部门级分支 user 三列为 NULL；列顺序等于 D-11 十一列"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "PG 方言 DDL 文本正确（CREATE OR REPLACE VIEW 幂等）且共享 body 满足方言红线（CAST / NOT mp.sync_enabled / SUBSTR+LENGTH / 无 LIKE / 无 ILIKE / 无 MATERIALIZED VIEW）"
    requirement: GUARD-02
    verification:
      - kind: unit
        ref: "go test ./internal/core/db/migrations/ -run TestPhase59_Migration217_PGDialectDDL -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "PG 真实启动：视图在 AutoMigrate 之前被 DROP、之后被重建，启动不被 0A000 阻塞，视图可查询（uuid/varchar UNION 与 boolean 取反在真实 PG 上成立）"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "本机无 PG 实例（RESEARCH Environment Availability 已记录 PostgreSQL 未探测）。sqlite 测试覆盖不到 uuid/varchar UNION 与 boolean 差异，PG 分支的真实执行验证属 59-VALIDATION.md 的 PG 启动冒烟行，必须人工在目标环境执行一次。"
  - id: D4
    description: "GUARD-02 的 edge probe 报 category=unclassified / status=unresolved；本计划按 ROADMAP SC3 原文取验收口径，未假设任何 backstop"
    requirement: GUARD-02
    verification: []
    human_judgment: true
    rationale: "plan frontmatter 显式 FLAGGED ASSUMPTION；phase verification 时需人工复核 ROADMAP SC3 是权威验收来源（与 59-01 的同类 flagged assumption 同款处置）。"

duration: 13min
completed: 2026-10-19
status: complete
---

# Phase 59 Plan 02: 部门↔OU 对账视图 (v_dept_ou_mismatch) Summary

普通 SQL 视图 `v_dept_ou_mismatch` 以 UNION ALL 四类口径实时计算 dept↔OU 不一致，双方言幂等创建、三启动路径注册、AutoMigrate 前置定向 DROP。

## Performance

- **Duration:** 13min
- **Started:** 2026-10-19T13:17:08Z
- **Completed:** 2026-10-19T13:30:24Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified; +771 lines, 0 deletions)

## Accomplishments

- `migration_217_dept_ou_mismatch_view.go` 建立 11 列普通视图（D-11），PG 用 `CREATE OR REPLACE VIEW`、sqlite 用 `DROP VIEW IF EXISTS` + `CREATE VIEW`，两侧共用同一份 SELECT body，重复执行均不报错。
- 四类口径齐备且各有独立分支：`user_ou_mismatch`（归一化 OU 比较，路由配置下的映射）、`dept_soft_deleted`、`mapping_disabled`、`config_unrouted`；部门级分支的 `user_id/username/user_ad_ou_dn` 恒为 NULL。
- `config_unrouted` 在 SQL 内复刻 `MatchADConfigByDN` 的最长后缀 + 逗号边界 + 等长歧义语义（`cnt = 1` 才可路由），为 59-03 的 Go/SQL 一致性测试留下可比对的基线。
- AutoMigrate 前置 DROP 与三条注册路径（PG 分支 / sqlite 分支 / `BootstrapMissingTables` 旁路）全部到位，避免重蹈「视图阻塞 GORM ALTER TYPE 导致启动 FATA」的仓库历史事故。
- 6 个自动化用例（含 2 个子用例）全绿；PG 分支在无 PG 实例时有可失败的文本守卫。

## Task Commits

Each task was committed atomically:

1. **Task 1: 端到端：migration_217 建视图（user_ou_mismatch 分支）并三路径注册** — `b4b88666` (feat)
2. **Task 2: 补齐四类口径分支与双方言加固** — `bda1e002` (test)

**Plan metadata:** committed with this SUMMARY.

## Files Created/Modified

- `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go` — 双方言 DDL + 四类口径 UNION ALL body + 归一化 SQL 生成器 + 4 个包内 helper（`deptOUMismatchNormalizeSQL` / `buildDeptOUMismatchViewSelect` / `pgDeptOUMismatchViewDDL` / `sqliteDeptOUMismatchViewDDLs`）。
- `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go` — sqlite 内存库 fixture（原生 DDL 建四张表）+ 6 个 `TestPhase59_Migration217*` 用例。
- `internal/core/db/database.go` — `dropDependentMaterializedViews()` 补 PG 前置 DROP；PG 分支 / sqlite 分支 / `BootstrapMissingTables()` 各加一次非阻断注册（`grep -c` = 3）。
- `internal/core/db/sqlite_reconciliation_views.go` — `dropSQLiteReconciliationViews()` 的 drops 列表追加 `v_dept_ou_mismatch`。

## Decisions Made

- **列名必须显式 AS 别名。** 首轮 sqlite 测试报 `SQL logic error: no such column: mismatch_type` —— sqlite 从表达式文本推导视图列名，字面量 `'user_ou_mismatch'` 会成为列名。改为每个分支逐列 `AS <D-11 列名>`（同时让列契约在源码里可读）。
- **`dept_id` 比较两侧都 `CAST(.. AS TEXT)`。** 查 archived DDL 确认 `sys_dept_ou_mapping.dept_id` 是 `uuid NOT NULL`（126_create_dept_ou_mapping_table.sql），而 `sys_user.dept_id` 是 `varchar(64)`（model tag `size:64`）。PG 没有 `uuid = varchar` 运算符，只 CAST 一侧会报 `operator does not exist`；两侧同转 TEXT 两种类型都成立。同理 `sys_ad_config.status` 经 016 DDL 核实是 `INT`，故 `status = 0` 安全。
- **PG 字面量计数与注释自指。** `grep -c 'CREATE OR REPLACE VIEW v_dept_ou_mismatch'` 首轮为 2（注释里重复写了完整拼串），把字面量收敛到唯一常量并改写注释后为 1。沿用 Phase 58 的裁决口径：以 plan 自己写的可证伪断言为准。
- **视图不做互斥裁剪。** 分支 ②③ 可能对同一条映射行各出一行，这是 UNION ALL 的预期语义（plan objective Decision Reconciliation 第 4 条），不额外加 `d.deleted_at IS NULL`。
- **sqlite 的 DROP 只进 `dropSQLiteReconciliationViews`。** `ensureSQLiteReconciliationViews` 也有一份 drops 列表，但那个函数只重建 reconciliation 三视图；把对账视图加进去会 DROP 掉且不重建（已在代码注释里写明）。
- **不标记 GUARD-02 为 complete。** `requirements.ready-ids` 返回 `{"ready":[],"blocked":["GUARD-02"]}` —— 59-03/59-04 同样声明该需求，共享-ID 门要求全部声明者完成才可置 complete，由最后一个 plan 回填（与 58-01/58-03/58-04/58-05 的分档口径一致）。

## Deviations from Plan

### Harness note (not a code deviation)

- **Protected-branch guard vs `branching_strategy: none`**: 仓库 `.planning/config.json` 配置为 `branching_strategy: "none"`，历史与本次 phase 的计划提交均在 `main`。本计划按 orchestrator 的 `<sequential_execution>` 指示在配置分支 `main` 上提交（未修改任何配置、未使用 `--no-verify`），与 59-01 同款处置。

### Auto-fixed Issues

**1. [Rule 1 - Bug] 视图列未命名导致无法按 `mismatch_type` 过滤**
- **Found during:** Task 1（sqlite 测试，`SELECT COUNT(*) ... WHERE mismatch_type = 'user_ou_mismatch'` 报 `no such column: mismatch_type`）
- **Issue:** 第一分支的列都是无别名表达式（`CAST(...)` / 字面量 / `||` 拼接），sqlite 用表达式文本当列名，视图对下游不可用。
- **Fix:** 四个分支的每一列都显式 `AS <D-11 列名>`。
- **Files modified:** `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go`
- **Verification:** `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1` 全绿；`PRAGMA table_info` 逐项等于 D-11 十一列。
- **Committed in:** `b4b88666`（Task 1）

**2. [Rule 1 - Bug] 注释自指把验收 grep 计数从 1 顶到 2**
- **Found during:** Task 1（源码断言 `grep -c 'CREATE OR REPLACE VIEW v_dept_ou_mismatch' == 1`，实测 2）
- **Issue:** 常量上方注释重复写出了「前缀 + 视图名」的完整拼串。
- **Fix:** 注释改写为不出现完整拼串的形式（保留"唯一常量"的说明）。
- **Files modified:** `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go`
- **Verification:** `grep -c 'CREATE OR REPLACE VIEW v_dept_ou_mismatch'` 输出 1。
- **Committed in:** `b4b88666`（Task 1）

**3. [Rule 2 - 缺失的关键正确性] 跨类型 dept_id 比较只 CAST 一侧（PG 会运行期报错，sqlite 测不出）**
- **Found during:** Task 2（对照 archived DDL 126 核实 `sys_dept_ou_mapping.dept_id` 为 `uuid`，而 `sys_user.dept_id` 为 varchar）
- **Issue:** `CAST(mp.dept_id AS TEXT) = u.dept_id` 在 PG 上是 `text = varchar`（可），但 `CAST(d.id AS TEXT) = u.dept_id` 这类写法在列类型假设变化时会退化为 `text = uuid` → `operator does not exist`。sqlite 全 TEXT，永远绿。
- **Fix:** 所有 dept_id 比较两侧统一 `CAST(.. AS TEXT)`；`LEFT JOIN sys_dept d` 与 `EXISTS (sys_user u2)` 一并加固。
- **Files modified:** `internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go`
- **Verification:** `go build ./...` + sqlite 全用例复跑通过（sqlite 侧语义不变）。
- **Committed in:** `bda1e002`（Task 2）

---

**Total deviations:** 3 auto-fixed（2 × Rule 1，1 × Rule 2）。**Impact on plan:** 均为本计划新代码的正确性缺陷，未改变 D-09~D-13 的任何锁定决策，未扩大范围。

## Issues Encountered

None.（首轮两处失败均在 Task 1 的验收闭环内当场修掉，未遗留。）

## Known Stubs

无。本计划交付的是被 59-03 端点消费的数据库对象，四类口径全部落地实现，无 TODO/FIXME/占位分支。

**已知可观测性缺口（非 stub，属锁定决策的边界）**：D-10 的四类口径**不含**「部门/用户在路由配置下没有任何映射」的用户（`unrecoverable:no-ou-mapping` 形态）。这是 plan objective `Decision Reconciliation` 第 2 条显式记录的 Known Gap —— 本 phase 只做可观测、不扩口径，该缺口应写入 59-03/59-04 的部署与 UAT 说明。

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: sql_view_on_trust_boundaries | internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go | 新增只读视图引用 sys_user / sys_dept / sys_dept_ou_mapping / sys_ad_config 四张表；视图本身不含用户输入插值（DDL 与谓词均为编译期常量），但**视图存在即会在未来阻塞 GORM 的 ALTER COLUMN TYPE**（T-59-04），已由 AutoMigrate 前置 DROP + 迁移后重建缓解。DB 层未额外向其他 DB 角色授权本视图。 |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 59-03 可直接在 `v_dept_ou_mismatch` 上做 `Table("v_dept_ou_mismatch")` 分页查询；列名与 D-11 逐字对齐，ID/时间列均已是 TEXT，DTO 用 `*string` 即可规避跨方言类型差异。
- 59-03 的 Go/SQL 一致性测试应以 `config_unrouted` 为锚点比对 `MatchADConfigByDN`；SQL 侧已复刻 `cnt = 1` 的等长歧义语义，父子域共享后缀、逗号边界、空 base、停用/软删配置均可直接构造。
- **部署前置（人工）**：目标环境需真实启动一次以验证 PG 分支（uuid/varchar UNION、boolean 取反、`CREATE OR REPLACE VIEW` 幂等），并确认启动日志无 `cannot alter type of a column used by a view or rule`。
- **未完成项**：GUARD-02 在 REQUIREMENTS.md 上仍为未完成（共享-ID 门 blocked），由 59-03/59-04 完成后回填。

## Self-Check

- FOUND: internal/core/db/migrations/migration_217_dept_ou_mismatch_view.go
- FOUND: internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go
- FOUND: internal/core/db/database.go (modified)
- FOUND: internal/core/db/sqlite_reconciliation_views.go (modified)
- FOUND: b4b88666 (task 1)
- FOUND: bda1e002 (task 2)
- `go build ./...` exit 0
- `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1` → ok（6 个用例 + 2 个子用例全部 PASS）
- `go test ./internal/core/db/... -count=1` → ok（无回归）
- 源断言：`database.go` 中 `Migrate217DeptOUMismatchView` = 3；`DROP VIEW IF EXISTS v_dept_ou_mismatch` 在 database.go = 1、sqlite_reconciliation_views.go = 1；PG 字面量计数 = 1；方言分支计数 = 7（≥2）

## Self-Check: PASSED

---
*Phase: 59-ad-login-guard-reconciliation*
*Completed: 2026-10-19*

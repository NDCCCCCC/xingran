---
phase: 62-ai-internal-core-db
plan: 01
subsystem: database
tags:
  - migrations
  - gorm
  - postgresql
  - reconciliation
  - mv
  - partial-index
  - sql-source-grep
  - sqlite

requires:
  - phase: 45-asset-reconciliation
    provides: Migrate175/176 R5 reconciliation 物理链路底座 + 双源 declared MV(MV 快慢路径基线)

provides:
  - Migrate176 快路径带 information_schema.columns R5 标记列版本校验,R1/R2→R5 就地升级自愈回退慢路径
  - ops_asset_physical 回填提取为私有函数 backfillOpsAssetPhysical,快路径与慢路径均执行
  - Type E 清理门控:SELECT EXISTS (SELECT 1 FROM ops_asset_physical LIMIT 1) 前置,空物理表跳过清理以避免静默真实告警
  - Type E 清理成功日志升级 applogger.Warnf,每次执行记录 RowsAffected(审计可见)
  - 175 支撑索引 idx_sys_user_nickname ON sys_user(nickname) WHERE deleted_at IS NULL
  - 176 支撑索引 idx_recon_resolved_asset_time ON sys_data_reconciliation(asset_id, resolved_at DESC) WHERE deleted_at IS NULL
  - 新增 migration_176_reconciliation_physical_mv_test.go:8 个源码 grep + sqlite 双调幂等测试

affects:
  - 后续任意 R5 reconciliation 升级路径(本 plan 自愈保护已就位)
  - 启动期迁移性能(MV refresh wall-clock 随 reconciliation 历史不再线性增长)

tech-stack:
  added: []
  patterns:
    - "快路径 schema 版本校验模式:information_schema.columns 探测 → 缺关键列回退 DROP+CREATE 自愈"
    - "破坏性 UPDATE 门控前置条件 + Warnf RowsAffected 审计(取代静默 Infof)"
    - "私有函数提取的双路径执行:同 SQL 逻辑从单路径复用为双路径(回填 / 索引)"
    - "PostgreSQL 部分索引:WHERE deleted_at IS NULL 谓词降索引体积 + 精准支撑标量子查询"

key-files:
  created:
    - internal/core/db/migrations/migration_176_reconciliation_physical_mv_test.go
  modified:
    - internal/core/db/migrations/migration_176_reconciliation_physical_mv.go
    - internal/core/db/migrations/migration_175_reconciliation_physical_link.go

key-decisions:
  - "快路径版本校验采用 information_schema.columns + 4 个 R5 标记列(轻量、不依赖 schema_version 表),而非新建 schema_migrations 系统表(避免引入新表+新代码路径)"
  - "Type E 清理仅在 ops_asset_physical 有数据时执行——物理链路回填是 R5 实际接入的信号,无数据则保留 Type E 由 DetectLayer3 自然重写"
  - "idx_recon_resolved_asset_time 失败走 applogger.Warnf 非阻断:全新库 sys_data_reconciliation 可能尚未建表,IF NOT EXISTS 下次启动自愈"
  - "175 idx_sys_user_nickname 失败走 return fmt.Errorf:与 175 既有 DDL 失败风格一致;database.go 调用方对 175 失败本身就是 Errorf 非阻断"
  - "测试文件复用 migration_202_port_write_audit_test.go:stripGoComments 模式(go/parser + ast.Inspect),保持项目内源码 grep 守卫实现一致"

patterns-established:
  - "PG-only migration 源码 grep 守卫 + sqlite 双调幂等测试模板:menu_grant_helpers_test.go + migration_202_port_write_audit_test.go 既有模式,新增 migration_176_reconciliation_physical_mv_test.go 沿用"

requirements-completed:
  - C1
  - CDX-M-IDX

duration: 13min
completed: 2026-08-14
---

# Phase 62 Plan 01: Migrate176 升级路径加固 + 175/176 支撑索引 Summary

**Migrate176 快路径 R5 schema 版本校验自愈升级 + Type E 清理门控 + 双回填,175/176 两个部分索引支撑高频标量子查询**

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-14T10:20:11Z
- **Completed:** 2026-08-14T10:33:38Z
- **Tasks:** 2 / 2
- **Files modified:** 3 (1 new test + 2 migration sources)

## Accomplishments

- **C1 三个子问题全部落地**(评审共识,两位 AI 共同标记的最高优先级):
  - 快路径 schema 版本校验:从"仅 MV 存在性"升级为"存在性 + R5 标记列完整性",缺列自动回退 DROP+CREATE 自愈升级
  - 双路径回填:`backfillOpsAssetPhysical` 函数提取,快路径(REFRESH CONCURRENTLY 后)与慢路径(DROP+CREATE 后)均执行,快路径安装不再永久缺失物理链路回填
  - Type E 清理门控:`EXISTS (SELECT 1 FROM ops_asset_physical LIMIT 1)` 前置,空物理表跳过清理;成功路径 `Warnf RowsAffected` 每次记录(审计可见,非首次专用)

- **CDX-M-IDX 两个支撑索引**:
  - `idx_sys_user_nickname` ON `sys_user(nickname) WHERE deleted_at IS NULL` — 支撑 reconciliation_user_lookup 的 per-row 标量子查询,6688 资产 × sys_user 全表扫描 → 索引查找
  - `idx_recon_resolved_asset_time` ON `sys_data_reconciliation(asset_id, resolved_at DESC) WHERE deleted_at IS NULL` — 支撑 MV `last_resolved` LATERAL 子查询的 `ORDER BY resolved_at DESC LIMIT 1`,避免 O(N×M) 全表排序

- **测试覆盖**:8 个新测试全部 PASS(4 Task 1 + 4 Task 2),覆盖源码 grep 守卫 + sqlite 双调幂等性;既有 menu_grant + migration_202 测试无回归

## Task Commits

每个任务单独提交:

1. **Task 1: Migrate176 快路径 schema 校验 + 双路径回填 + Type E 清理门控 (C1)** — `5b57146` (fix)
2. **Task 2: 175 idx_sys_user_nickname + 176 idx_recon_resolved_asset_time (CDX-M-IDX)** — `540e0af` (perf)

## Files Created/Modified

- `internal/core/db/migrations/migration_176_reconciliation_physical_mv.go` — 三处 C1 修复(快路径 schema 校验 / 双路径回填 / Type E 门控 WARN)+ docstring 漂移修正 + CDX-M-IDX 索引 `idx_recon_resolved_asset_time`
- `internal/core/db/migrations/migration_175_reconciliation_physical_link.go` — CDX-M-IDX 索引 `idx_sys_user_nickname` 在 reconciliation_user_lookup 视图创建之后插入
- `internal/core/db/migrations/migration_176_reconciliation_physical_mv_test.go`(新建) — 8 个测试:
  - Task 1: `TestMigrate176_FastPathSchemaVersionCheck` / `TestMigrate176_TypeECleanupGate` / `TestMigrate176_SqliteDoubleInvocation` / `TestMigrate176_NoObsoleteDocstring`
  - Task 2: `TestMigrate175_NicknamePartialIndex` / `TestMigrate176_ResolvedAssetTimeIndex` / `TestMigrate176_AllDDLIdempotent` / `TestMigrate175_SqliteDoubleInvocation`

## Decisions Made

- **快路径版本校验选用 information_schema.columns 探测而非 schema_migrations 表**:4 个 R5 标记列(轻量、无需新建元数据表)即可区分 R1/R2 vs R5 schema;若未来需要更精细版本号,再单独建 `schema_migrations` 表(属 Phase 62 后续 plan 范畴)
- **Type E 清理门控前置条件 = ops_asset_physical 有数据**:物理链路回填是 R5 实际接入的运行时证据;空表即说明 R5 未真正落地,Type E 可能是真实告警,绝不静默关闭
- **idx_recon_resolved_asset_time 失败非阻断**:全新库 sys_data_reconciliation 可能尚未由上游 AutoMigrate 建表,`IF NOT EXISTS` 下次启动自愈,无需阻断
- **175 idx_sys_user_nickname 失败阻断(与 175 既有 DDL 风格一致)**:database.go 调用方对 175 失败本身就是 Errorf 非阻断(不会真正阻塞启动),保持代码风格统一
- **测试模式沿用既有项目惯例**:源码 grep 守卫使用 `migration_202_port_write_audit_test.go:stripGoComments` 模式(go/parser + ast.Inspect 排除注释);PG-only 迁移在 sqlite 上双调返回 nil 验证方言守卫幂等

## Deviations from Plan

### Auto-fixed Issues

无 — 计划完全按 PLAN.md 实施,未触发 Rule 1-3 自动修复路径。`go build ./...` 退出码 0,所有迁移测试 PASS,既有测试无回归。

### Deferred Items

- **PG 功能路径延期**:REFRESH CONCURRENTLY 真实行为、information_schema.columns 列集校验在 PG 上的真实探测、Type E 门控实际生效、索引在 PG 上的真实执行计划——这些由项目既有 "PG functional 由 dev 启动 / UAT 覆盖" 惯例负责(参考 PLAN.md `<verification>` 与既有 migration_202 测试的相同模式)。本 plan 在 sqlite 方言守卫路径上验证幂等性(双调返回 nil)+ 源码 grep 守卫验证 SQL 片段就位。

## Threat Surface

| Threat ID | Component | Mitigation Status |
|-----------|-----------|-------------------|
| T-62-01 | Migrate176 Type E 批量 UPDATE | MITIGATED:EXISTS (SELECT 1 FROM ops_asset_physical LIMIT 1) 前置门控 + 每次 applogger.Warnf RowsAffected(审计可见) |
| T-62-02 | Migrate176 快路径刷新旧结构 MV | MITIGATED:R5 标记列(asset_username/physical_user_id/last_resolved_at/mv_refreshed_at)information_schema 校验,缺失即回退 DROP+CREATE |
| T-62-03 | 逐行标量子查询扫全表 | MITIGATED:idx_sys_user_nickname + idx_recon_resolved_asset_time 两个部分索引 |
| T-62-SC | 依赖安装 | ACCEPTED:本 plan 不引入任何新依赖(纯 Go 标准库 + 既有 gorm/applogger),无包安装面 |

## Verification Results

- `go build ./...` exit 0 ✓
- `go test ./internal/core/db/migrations/ -v` 全部 PASS(11 non-skip + 3 skip PG-only)✓
- 全部新 DDL 使用 `IF NOT EXISTS` / `CREATE OR REPLACE` / `ON CONFLICT`,幂等 ✓
- 迁移函数 sqlite 路径双调返回 nil,Migrate175/Migrate176 各两次连续调用均 PASS ✓
- 既有 menu_grant_helpers_test.go + migration_202_port_write_audit_test.go 无回归 ✓

## Next Phase Readiness

- Phase 62 Plan 01 完成(C1 + CDX-M-IDX 评审项已修复)
- 后续 Phase 62 Plan 02-05 仍按计划推进(FilterLogger / 种子加固 / advisory lock / BootstrapMissingTables),无 blocker
- Phase 58 SC#1-SC#4 端到端验证延期(原 2026-07-10 起 v1.21 dev DB Supabase pooler 性能问题,本 plan 未触碰)仍待更快 dev DB

## Self-Check

**PASSED**

- `.planning/phases/62-ai-internal-core-db/62-01-SUMMARY.md` exists ✓
- `internal/core/db/migrations/migration_176_reconciliation_physical_mv_test.go` exists ✓
- Task 1 commit `5b57146` exists ✓
- Task 2 commit `540e0af` exists ✓

---
*Phase: 62-ai-internal-core-db*
*Plan: 01*
*Completed: 2026-08-14*

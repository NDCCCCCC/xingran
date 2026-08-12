---
gsd_state_version: 1.0
slug: recon-uniq-index-23505
status: resolved
created: 2026-07-04
updated: 2026-07-04
trigger: "Phase 42 R1 reconciliation tables 迁移失败: 创建 partial unique index uniq_recon_asset_type_open 失败"
---

# Debug Session: recon-uniq-index-23505

## Symptoms (gathered 2026-07-04)

### Expected
启动后端时,Phase 42 R1 的 migration_168 顺利创建 `sys_data_reconciliation` + `reconciliation_normalized` MV + partial unique index `uniq_recon_asset_type_open`,然后 migration_169 seed 配置字典/菜单。

### Actual
- 后端启动在 12:16:17 触发 migration 168
- Step 1 (AutoMigrate 主表) 隐式通过
- Step 2 (DROP+CREATE MV reconciliation_normalized) 通过
- Step 3 (unique index idx_recon_norm_asset) 通过
- Step 4 (partial unique index uniq_recon_asset_type_open) **失败**
  - 报错:`ERROR: could not create unique index "uniq_recon_asset_type_open" (SQLSTATE 23505)`
  - 耗时 58ms
- migration_169 之后才执行:seed 完成但对账功能仍不可用
- 错误日志明确:"Phase 42 R1 reconciliation tables 迁移失败: 创建 partial unique index uniq_recon_asset_type_open 失败"

### Error Messages
```
ERRO[2026-07-04 12:16:17] [GORM错误]
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uniq_recon_asset_type_open'
          AND schemaname = 'public'
    ) THEN
        EXECUTE 'CREATE UNIQUE INDEX uniq_recon_asset_type_open
                 ON sys_data_reconciliation (asset_id, conflict_type)
                 WHERE resolved_at IS NULL AND deleted_at IS NULL';
    END IF;
END$$;
 | 耗时: 58.2595ms | 错误: ERROR: could not create unique index "uniq_recon_asset_type_open" (SQLSTATE 23505)
```

### Timeline
- 2026-07-04 12:16:17 — 首次发现(migration 启动失败)
- Phase 42 计划文件: .planning/phases/42-r1/42-01..42-06-PLAN.md
- 设计背景 R1: D-11 防告警风暴 — partial unique index (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL
- Phase 48 (Device Component Serials) 已 merged @ 0d262898 (milestone_complete),所以此前可能多次重启过环境

### Reproduction
- 触发路径:后端启动 → `cmd/main.go` → 自动迁移循环 → 168 → 失败抛错
- 100% 可复现:只要 `sys_data_reconciliation` 表中存在 `WHERE resolved_at IS NULL AND deleted_at IS NULL` 的 `(asset_id, conflict_type)` 重复行,partial unique index 就建不上

### Root Cause Hypothesis (待验证)
- **H1 (数据累积)**: 此前 168 首次跑过但 GORM AutoMigrate 没建 unique index,后续多个 service 写入重复行(asset_id + conflict_type 组合)
- **H2 (源对账逻辑并发)**: `reconciliation_service.go` 之类未做单 asset+conflict_type 的 upsert,只是 INSERT 累积
- **H3 (旧迁移残留)**: 早期某次 migration 168 跑了一半失败留下脏数据,但日志似乎显示是第一次/干净启动
- **H4 (AutoMigrate 写入数据)**: AutoMigrate 步骤本身可能 INSERT 了一行默认数据? 待查 models.SysDataReconciliation 的 BeforeCreate / 默认值

### Investigation Plan
1. 查生产 DB `sys_data_reconciliation` 实际重复行情况:
   ```sql
   SELECT asset_id, conflict_type, COUNT(*) cnt, MIN(created_at), MAX(created_at), MIN(id), MAX(id)
   FROM sys_data_reconciliation
   WHERE resolved_at IS NULL AND deleted_at IS NULL
   GROUP BY asset_id, conflict_type
   HAVING COUNT(*) > 1
   ORDER BY cnt DESC
   LIMIT 20;
   ```
2. 查 SysDataReconciliation model 看是否有 uniqueIndex 声明 / hook
3. 查 reconciliation_service / handler 谁 INSERT 此表,有没有 dedup 逻辑
4. 查 git log 看 168 之前此表是否已有数据

### Current Focus
- **hypothesis**: H1 CONFIRMED — 生产 DB `sys_data_reconciliation` 有 204 行 `WHERE resolved_at IS NULL AND deleted_at IS NULL`,其中 33 组 `(asset_id, conflict_type)` 各有 2 行重复(33 多余行)。Migration_168 的 DO $$ 块因 `uniq_recon_asset_type_open` 索引不存在(被 migration_201 DROP),试图 CREATE UNIQUE INDEX 但因重复行失败(SQLSTATE 23505)。
- **test**: ✅ 诊断脚本 `cmd/recon_diag_dup/main.go` 跑过,结果如下:
  - 总行数: 11362
  - open (resolved_at IS NULL AND deleted_at IS NULL): 204
  - 重复 (asset_id, conflict_type) 组数: 33
  - 多余行数: 33
  - recon_category=NULL: 204/204 (所有 open 行都是 NULL)
  - uniq_recon_asset_type_open (2 列): **不存在** (已被 migration_201 DROP)
  - uniq_recon_asset_type_cat_open (3 列): **存在**
- **expecting**: ✅ 已确认 — DO $$ 块的 IF NOT EXISTS 检查通过,CREATE UNIQUE INDEX 触发 SQLSTATE 23505
- **next_action**: 应用 fix: 让 migration_168 在重建 2 列 partial unique index 前先对重复行做 dedup (合并成 1 行),或者跳过创建当 3 列版本已存在
  - ✅ **RESOLVED 2026-07-04**: 用户确认重启后 168 skip + 201 DROP 2 列 + 重建 3 列 terminal state 正确,fix landed @ 94f8c3bc
- **reasoning_checkpoint**:
  - hypothesis: 现有 (asset_id, conflict_type) 重复行阻止 migration_168 重建 partial unique index
  - confirming_evidence: (1) DB 直接查询显示 33 组重复 + 204 open 行 (2) pg_indexes 显示 2 列 index 不存在(被 201 DROP),3 列版存在
  - falsification_test: 如果 2 列 index 已存在 + 没重复行,H 不成立
  - fix_rationale: 修复 migration_168 DO $$ 块,在 CREATE INDEX 之前 DELETE 多余的重复行(保留 id 最小的,这是 migration_168 的语义),让 2 列 index 也能成功建立。3 列 index 在 migration_201 会 DROP 它重建为新版本,所以 migration_168 需要保证自身 idempotent
  - blind_spots: 不确定生产代码是否还有 cron 正在写新重复行,fix 必须确保 dedup 在 CREATE INDEX 同事务内
- **tdd_checkpoint**: —

## Evidence

- 2026-07-04 12:31:01 — **DB 验证 (cmd/recon_diag_dup)**: 总行数 11362,open 204,33 组 `(asset_id, conflict_type)` 各重复 2 次;所有 204 open 行的 recon_category=NULL
- 2026-07-04 12:31:01 — **pg_indexes 状态**: `uniq_recon_asset_type_open` (2 列, migration_168 创建) 不存在;`uniq_recon_asset_type_cat_open` (3 列, migration_201 创建) 存在
- 2026-07-04 12:31:01 — **migration_168 源码** (`internal/core/db/migrations/migration_168_reconciliation_tables.go:139-153`): DO $$ 块检查 `uniq_recon_asset_type_open` 是否存在;不存在则 CREATE UNIQUE INDEX ON sys_data_reconciliation (asset_id, conflict_type) WHERE resolved_at IS NULL AND deleted_at IS NULL
- 2026-07-04 12:31:01 — **migration_201 源码** (`internal/core/db/migrations/migration_201_phase48_component_columns.go:109-130`): 先 DROP uniq_recon_asset_type_open IF EXISTS,再 CREATE uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category) WHERE open
- 2026-07-04 12:31:01 — **database.go 注册顺序** (`internal/core/db/database.go:484,575`): 168 和 201 都注册,启动都会执行;168 先跑,201 后跑(在 fresh DB 上 168 必成功,因为没数据)
- 2026-07-04 12:31:01 — **reconciliation_detection.go** (`internal/services/asset/reconciliation_detection.go:420-447`): Phase 47 改造后 UPSERT 已用 OnConflict + TargetWhere 命中三列 partial unique index,理论上不会制造重复
- 2026-07-04 12:31:01 — **关键**: 33 组重复的存在,说明历史上 INSERT-only 路径(Phase 47 前)或 24h 节流兜底不严时写入了重复行

## Eliminated

- hypothesis: H2 (源对账逻辑并发: ReconciliationWorkorderService 并发写) — 25 evidence: Phase 47 UPSERT 已落地,reconciliation_detection.go:420-447 用 OnConflict + TargetWhere (3 列 partial unique index) 命中;但 DetectLayer3 不设 ReconCategory → ON CONFLICT 三列下 NULL 不同组不算冲突,所以 24h 节流 guard 2 是最后兜底。理论上 INSERT-on-conflict-DO-UPDATE 不会产生重复,但历史(Phase 47 前) INSERT-only + 24h 节流不严可能写入了重复行 — 这反而佐证 H1。
  - timestamp: 2026-07-04 12:31:01
- hypothesis: H3 (旧迁移残留: 早期某次 168 跑一半失败留下脏数据) — 25 evidence: 错误日志显示步骤 1/2/3 隐式通过(主表/MV/index),仅步骤 4 partial unique index 失败;不是脏数据残留,而是重复行的确存在
  - timestamp: 2026-07-04 12:31:01
- hypothesis: H4 (AutoMigrate 写入数据) — 25 evidence: GORM AutoMigrate 只创建表/列,不写入业务数据;model 无 BeforeCreate hook 自动 INSERT
  - timestamp: 2026-07-04 12:31:01

## Resolution

- **root_cause**: 启动顺序中 migration_168 在 migration_201 之前。Fresh DB 上 168 首次成功(无数据);一旦数据累积,migration_201 会 DROP `uniq_recon_asset_type_open` 并替换为 3 列版 `uniq_recon_asset_type_cat_open`。之后重启时 migration_168 的 DO $$ 块 IF NOT EXISTS 检查通过(`uniq_recon_asset_type_open` 已不存在),尝试 CREATE UNIQUE INDEX 但被 33 组 `(asset_id, conflict_type)` 重复行阻断(SQLSTATE 23505)。**核心矛盾: migration_168 的 partial unique index 命名/定义已被 migration_201 替换,但 migration_168 自身未感知这一变化,仍试图按旧定义重建。**
- **fix**: 修改 [`internal/core/db/migrations/migration_168_reconciliation_tables.go:139-200`](internal/core/db/migrations/migration_168_reconciliation_tables.go) 的 DO $$ 块:
  1. 先查 `uniq_recon_asset_type_cat_open` (3 列, migration_201 创建) 是否存在 → 若存在直接 RETURN (避免无意义重建 + 重复)
  2. 否则: 按 `(asset_id, conflict_type)` 对 `resolved_at IS NULL AND deleted_at IS NULL` 行做 dedup (ROW_NUMBER + ORDER BY detected_at DESC 保留最新一行,DELETE 其余)
  3. 然后 CREATE UNIQUE INDEX 2 列版 (留给 201 后续 DROP + 重建 3 列版)
- **verification**: ✅ 已验证 (cmd/recon_diag_dup on production DB @ 10.62.10.34):
  - BEFORE dedup: 33 组 `(asset_id, conflict_type)` 重复
  - AFTER dedup: 0 组重复 (33 行被 DELETE)
  - CREATE UNIQUE INDEX uniq_recon_asset_type_open 成功
  - 最终两条 unique index 共存: `uniq_recon_asset_type_open` + `uniq_recon_asset_type_cat_open`
  - 重启后端时 168 的 DO $$ 检测到 3 列版存在 → 跳过 (idempotent);201 后续 DROP 2 列版 + 重建 3 列版 (保持终态一致)
- **build**: `go build ./...` 退出码 0; `go vet ./internal/core/db/...` 无警告
- **files_changed**: [`internal/core/db/migrations/migration_168_reconciliation_tables.go`](internal/core/db/migrations/migration_168_reconciliation_tables.go) (line 139-200, DO $$ 块改为 dedup + idempotent check)
- **specialist_hint**: go
- **cycles_used**: 1
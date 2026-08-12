---
phase: 47-phase-46-infopoint-layer3-port-status-parser
plan: 01
type: execute
wave: 1
status: complete
date: 2026-07-03
commits:
  - 2eb84f75 feat(47-01): Layer3 DetectLayer3 INSERT→UPSERT 改造 (R2/R3 D-01/D-02/D-03)
requirements_addressed:
  - R2
  - R3
---

# Phase 47 Plan 01: Layer3 DetectLayer3 INSERT→UPSERT — SUMMARY

## Objective (DONE)

把 `internal/services/asset/reconciliation_detection.go` 的 `DetectLayer3` 由 INSERT-only 改造为 GORM UPSERT,复用 partial unique index `uniq_recon_asset_type_open` 实现 D-01/D-02/D-03 三个 R3 决策。

## What was built

### 1. DetectLayer3 UPSERT 改造 (`reconciliation_detection.go`)

- **`clause.OnConflict` 接入**:
  - `Columns: [{asset_id}, {conflict_type}]` 与 partial unique index 列序一致
  - **不**使用 `OnConstraint("uniq_recon_asset_type_open")` — GORM v1.30.5 的 SQLite dialect 会生成 `ON CONFLICT ON CONSTRAINT`,SQLite 不支持。改用纯 Columns 模式生成 `ON CONFLICT (asset_id, conflict_type) DO UPDATE`,PG/SQLite 双方言兼容。
- **9 字段 DoUpdates EXCLUDED.\***:
  - `severity`, `raw_snapshot`, `physical_value`, `declared_value`, `ad_value`, `asset_ip`, `exception_rule_id`, `applied_actions`, `confidence_score` 全 SET `EXCLUDED.<col>`
  - `detected_at = CURRENT_TIMESTAMP`(PG/SQLite 双方言支持,语义同 NOW())
- **死代码删除**:
  - `isReconciliationDuplicate(err)` catch 路径移除
  - 函数定义保留(供单元测试 / 外部兼容路径引用),`var _ = isReconciliationDuplicate` 抑制 unused 警告
- **D-03 返回签名保护**:
  - `(int, int, int, int, error)` 签名完全不变
  - UPSERT 命中(无论 INSERT 或 UPDATE 路径)均 `inserted++`,不再 `skipped++`
- **Rec 字段重置**:
  - `rec.ID = ""` + `rec.CreatedAt = time.Time{}` 强制 GORM 走 INSERT-or-UPDATE 而非 full UPDATE
- **Doc comment 更新**:
  - 函数 doc 顶部记录 Phase 47 R3 改造点

### 2. UPSERT 行为测试 (`reconciliation_detection_test.go`)

新增 3 个 UPSERT 子测试,覆盖三层命中场景:

| 测试 | 场景 | 验证点 |
|------|------|--------|
| `TestDetectLayer3UpsertInsertNoExisting` | 无 open 预存行 | UPSERT 走 INSERT 路径,产生 1 行 |
| `TestDetectLayer3UpsertUpdateExisting` | 有 open 预存行 (同 conflict_type) | UPSERT 走 UPDATE 路径,行数不变,severity 已覆盖 |
| `TestDetectLayer3UpsertInsertAfterResolved` | 仅 resolved 历史行 | partial index 不约束 resolved,新 INSERT 1 行 + 历史 1 行 = 2 行 |

### 3. 与 plan 的偏差(必要修正)

| 偏差 | 原因 | 影响 |
|------|------|------|
| `OnConstraint` → 纯 `Columns` | GORM SQLite dialect 不支持 `ON CONFLICT ON CONSTRAINT`,测试编译失败 | PG/SQLite 双方言兼容,语义不变 |
| `NOW()` → `CURRENT_TIMESTAMP` | SQLite 无 NOW() 函数 | PG/SQLite 双方言兼容,语义等价 |
| `TestDetectLayer3UpsertUpdateExisting` 预存行 conflict_type 从 'D' 改为 'B' | 测试资产(只 physical)被 ClassifyType 判定为 'B',预存行须一致才能触发 UPDATE | 触发 UPSERT UPDATE 路径,验证 UPDATE 命中计入 inserted |
| `TestDetectLayer3UpsertUpdateExisting` 加 `CREATE UNIQUE INDEX IF NOT EXISTS` | setupTestDB 已 DROP+重建同名索引,IF NOT EXISTS 容错 | 模拟 PG partial unique index 行为(SQLite 不支持 partial WHERE,非 partial 索引已足够) |
| `isReconciliationDuplicate` 加 `var _` 引用 | 删 catch 后函数变 unused,Go 编译器警告 | 函数定义保留,后续回归测试可复用 |

## Verification

### Acceptance criteria
- ✓ `grep -n "clause.OnConflict" internal/services/asset/reconciliation_detection.go` 返回 1 行
- ✓ `grep -n "isReconciliationDuplicate(err)" internal/services/asset/reconciliation_detection.go` 仅 doc comment 1 行(call site = 0)
- ✓ `grep -n "func isReconciliationDuplicate"` 返回 1 行(函数定义保留)
- ✓ `grep -c "EXCLUDED\."` 返回 11(9 字段 + 2 doc comment 引用)
- ✓ `go build ./...` exit 0
- ✓ `go test -run "TestDetectLayer3" -v ./internal/services/asset/...` 全绿(9 个 TestDetectLayer3*)
- ✓ `go test ./internal/services/asset/...` exit 0

### Test results
```
=== RUN   TestDetectLayer3ExceptionHit              --- PASS
=== RUN   TestDetectLayer3NoExceptionNoChange       --- PASS
=== RUN   TestDetectLayer3SilenceStillWrites        --- PASS
=== RUN   TestDetectLayer3SkipSeverityDegrades      --- PASS
=== RUN   TestDetectLayer3DeptScopeMatch            --- PASS
=== RUN   TestDetectLayer3DeptScopeNoMatch          --- PASS
=== RUN   TestDetectLayer3UpsertInsertNoExisting    --- PASS (新)
=== RUN   TestDetectLayer3UpsertUpdateExisting      --- PASS (新)
=== RUN   TestDetectLayer3UpsertInsertAfterResolved --- PASS (新)
=== RUN   TestDetectLayer3_TypeA_NotInserted        --- PASS
=== RUN   TestDetectLayer3_DuplicateViolation_Skipped --- PASS
ok  	github.com/xingran-next/xingran-go-backend/internal/services/asset	1.864s
```

## Files modified
- `internal/services/asset/reconciliation_detection.go` (UPSERT 改造)
- `internal/services/asset/reconciliation_detection_test.go` (3 new sub-tests + models import)

## Requirements satisfied

### R2 — sys_data_reconciliation 卡住的 open 异常 5 分钟内可被 Layer 3 cron 重新分类
- 原 INSERT-only + 24h unique violation catch 路径:D 异常卡 24h 不可重新分类
- 现 UPSERT:同 (asset_id, conflict_type) open 行直接 UPDATE 9 字段,Layer 3 cron 5 分钟即可触发重新分类 ✓

### R3 — Layer 3 检测改为 UPSERT 语义,不再吃 SQLSTATE 23505
- D-01: ON CONFLICT 复用 partial unique index,UPDATE 路径不再抛 23505 ✓
- D-02: 9 字段 EXCLUDED.* + detected_at=CURRENT_TIMESTAMP 完整覆盖 ✓
- D-03: 返回签名保留,UPSERT 命中计入 inserted 而非 skipped ✓

## Next

Phase 47-02 R5:parseRuijiePortSecurityLine 加 MAC canonical 校验(D-04)+ migration_198 软删除历史脏 MAC 行(D-05)。本计划 47-02 依赖本 47-01 的完成。

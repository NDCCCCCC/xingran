---
phase: "103"
plan: "02"
subsystem: asset-reconciliation-cache
tags: [cache, refactor, base-cache-provider]
requires:
  - "asset/reconciliation hand-written cache-aside (GetJSON + Marshal + Set)"
provides:
  - "reconciliation GetByWorkstation on base.GetOrSetJSON via getByWorkstationWithCache wrapper"
affects:
  - "internal/api/v1/asset/reconciliation_router.go"
  - "internal/api/router.go"
tech-stack:
  added: []
  patterns:
    - "getByWorkstationWithCache wrapper (warn-on-set non-blocking, err → DB direct query)"
    - "dirty-cache guard: cached Workstation.ID == \"\" → recompute from source"
key-files:
  created: []
  modified:
    - "internal/services/asset/reconciliation_service.go"
    - "internal/api/v1/asset/reconciliation_router.go"
    - "internal/api/router.go"
    - "internal/services/asset/asset_gapfill_test.go"
key-decisions:
  - "脏缓存防御显式迁移：base.GetOrSetJSON 命中后检查 resp.Workstation.ID == \"\" 回源重建（原手写 GetJSON 成功 + ID 空判定的等价形态）"
  - "cache==nil 单测场景保留直查分支（等价原 nil-guard 语义）"
  - "测试适配用 system.NewCacheProvider(services.NewDataCacheService(mem))——asset 包 import services root + system 均单向无环"
requirements-completed:
  - "CONV-02"
duration: "12 min"
completed: "2026-09-08T00:50:00Z"
---

# Phase 103 Plan 02: asset/reconciliation 迁移 base.CacheProvider Summary

reconciliation GetByWorkstation 手写 cache-aside（GetJSON 短路 + json.Marshal + Set）收敛 base.GetOrSetJSON[T] + getByWorkstationWithCache wrapper，warn-on-set 不阻断语义保留；两处 router 接线同步注入 system.NewCacheProvider。

## What Was Built

### Task 1: reconciliation_service.go（commit 8d1901c）
- struct 字段 `cache cache.Cache` → `cache base.CacheProvider`（D-103-1/D-103-3）
- `NewReconciliationService` 入参 `c cache.Cache` → `c base.CacheProvider`（D-103-2）
- 新增 `getByWorkstationWithCache` wrapper：
  - `base.GetOrSetJSON[*ByWorkstationResponse]` TTL `reconciliationHealthCacheTTL`（5min，D-103-11）
  - err → warn 日志 + `computeByWorkstation` 直查（D-103-7 warn-on-set 不阻断）
  - 缓存命中但 `Workstation.ID == ""` → 脏缓存回源重建（原手写防御等价迁移）
  - `resp.Visible = false` 语义保留（service 单一职责）
- `cache == nil` 单测场景直查 DB（等价原 nil-guard）
- 删除未用 `encoding/json` / `pkg/cache` import

### Task 2: 两处 router 接线（commit 957d5ae）
- `reconciliation_router.go:34`：`core.Cache` → `system.NewCacheProvider(core.DataCacheService)`（D-103-20）
- `router.go:619`：同形态（`systemServices` alias 已在 import 中）

### Task 3: asset_gapfill_test.go 适配（commit 957d5ae）
- `:346` `NewReconciliationService(db, mem, nil)` → `NewReconciliationService(db, system.NewCacheProvider(services.NewDataCacheService(mem)), nil)`（B6）
- `:203/:210/:241/:300` 传 nil 调用点零改动（nil → 直查分支）

## Verification Results

- `go build ./...` — PASS
- `go test ./internal/services/asset/ -run "Reconciliation|Gapfill" -count=1` — PASS（含 TestReconciliationService_GetByWorkstationCache 缓存命中断言：首次写键 + 删行后命中缓存短路 DB）
- `go test ./internal/services/system/ -run "GetOrSetResidue" -count=1` — PASS

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1-2 | struct/constructor 改 base.CacheProvider | ✅ |
| 3 | getByWorkstationWithCache wrapper（warn-on-set D-103-7） | ✅ |
| 4-6 | 两处 router 接线 + 测试适配 | ✅ |
| 7 | asset_gapfill_test.go:346 适配 | ✅ |
| 8-9 | build + Reconciliation 测试通过 | ✅ |

注：SaveRecord 失效路径（D-103-14/D-103-15）经 caller audit 证实 reconciliation_service.go 内**不存在** `SaveRecord`/`DeleteByPattern` 调用——失效入口是 `cache_keys.go` 的 `InvalidateWorkstationHealth`（cache.Cache 签名，独立函数，被 handler/workorder/scheduler 调用），不在本 plan `files_modified` 清单内，保持不动（计划未声明修改该文件）。

## Deviations from Plan

**[Rule 1 - Plan vs code mismatch] SaveRecord/D-103-14/D-103-15 无对应位点** — Found during: Task 1 | Issue: 计划 truths 引用 SaveRecord 失效路径改 base.Invalidate，但 reconciliation_service.go 无 SaveRecord 方法，失效走 cache_keys.go 独立函数（不在 files_modified） | Fix: GetByWorkstation 主迁移照做；失效函数不在本 plan 范围保持 cache.Cache 形态（handler 传 h.core.Cache 兼容） | Files: 无额外改动 | Verification: build + 测试 PASS | Commit: N/A

**Total deviations:** 1 auto-resolved (plan-code mismatch). **Impact:** 无——失效链路语义未变，仅 GetByWorkstation 读路径收敛 base。

## Self-Check: PASSED

---
phase: "103"
plan: "01"
subsystem: mac-history-cache
tags: [cache, refactor, base-cache-provider]
requires:
  - "internal/services/DataCacheService legacy interface{} closure GetOrSet"
provides:
  - "mac_history_query_service + heatmap_service on base.CacheProvider / base.GetOrSetJSON"
affects:
  - "internal/api/v1/network/mac_history_router.go"
tech-stack:
  added: []
  patterns:
    - "base.GetOrSetJSON[T] strict semantics (cache write failure returns error)"
    - "getHeatmapWithCache fallback wrapper (err → DB direct query)"
    - "fakeMACHistoryCacheProvider in-package test fixture (B8 import-cycle avoidance)"
key-files:
  created:
    - "internal/services/mac_history_test_fake_cache.go"
  modified:
    - "internal/services/mac_history_query_service.go"
    - "internal/services/mac_history_heatmap_service.go"
    - "internal/api/v1/network/mac_history_router.go"
    - "internal/services/mac_history_query_service_79_05_test.go"
    - "internal/services/mac_history_tail_79_05_test.go"
    - "internal/api/v1/network/setup_routers_test.go"
key-decisions:
  - "GetVendor cache==nil 裸装配保留直查分支（等价原手写 nil 语义），lookupVendorFromDB 提取 DB 直查逻辑供两条路径复用"
  - "缓存值断言改为 JSON 引号形态：GetOrSetJSON 走 DataCacheService JSON 序列化，mem.Get 返回带引号字符串"
  - "setup_routers_test 字面断言改三段式（多行调用形态下单一长串不匹配）"
requirements-completed:
  - "CONV-01"
duration: "18 min"
completed: "2026-09-08T00:35:00Z"
---

# Phase 103 Plan 01: mac_history 域迁移 base.CacheProvider Summary

mac_history query + heatmap 双服务从 legacy interface{} 闭包 GetOrSet 与手写 cache-aside 全部收敛 base.GetOrSetJSON[T] + base.CacheProvider，router 接线注入 system.NewCacheProvider(core.DataCacheService)。

## What Was Built

### Task 1a+1b: mac_history_query_service.go（commit 44301ef）
- struct 字段 `cache base.CacheProvider`，`dataCache` 字段删除（D-103-1/D-103-3）
- `NewMACHistoryQueryServiceWithCache` 入参 `dataCache *DataCacheService` → `cache base.CacheProvider`（D-103-2）
- vendor 手写 cache-aside（原 :259-281）收敛 `base.GetOrSetJSON[string]`，TTL `24*time.Hour` 不变（D-103-5），Unknown Vendor 占位保留；cache==nil 裸装配走 `lookupVendorFromDB` 直查（等价原手写 nil 分支）
- 三处 GetOrSet 位点（port-history :309 / device-history :434 / stats :837）改 `base.GetOrSetJSON[*MACHistoryQueryResult]` 严格语义——cache 写失败返回 error（D-103-6）；TTL `s.perfCacheTTL()` 不变（D-103-10）
- 缓存键构造失败（keyErr != nil）走直查 + warn 日志（保留原降级路径）

### Task 2: mac_history_heatmap_service.go（commit 4b6f3d6）
- struct 字段 + 构造函数同形态改造
- 新增 `getHeatmapWithCache` wrapper：`base.GetOrSetJSON` err → warn + `queryHeatmapFromMV` 直查，fallback 直查语义保留（D-103-9）

### Task 3: mac_history_router.go + setup_routers_test.go（commit 5565fc7）
- 热力图接线 `system.NewCacheProvider(core.DataCacheService)` + `core.CacheConfigService`（D-103-20）
- setup_routers_test.go:250 字面断言同步更新为三段式

### Task 4: 测试 fixture + 五处调用点（commits 44301ef + 5565fc7）
- 新增 `fakeMACHistoryCacheProvider`（B8 import-cycle 无环方案）：实现 base.CacheProvider 全 9 方法，内部委托 `*DataCacheService` 真实缓存行为，GetOrSet/Delete 支持注错；编译期断言 `var _ base.CacheProvider = ...`
- 五处调用点：:140/:57/:728 真实缓存路径 → fixture；:694/:717 fakeDB 场景 → `&base.NoOpCacheProvider{}`（原 nil 会 panic）
- 两个测试文件均无 `internal/services/system` import（无环 ✓）

## Verification Results

- `go build ./...` — PASS
- `go test ./internal/services/ -run "MACHistory|Heatmap|Vendor|Mhq|Mhs" -count=1` — PASS（含 GetVendor 缓存命中/DB 直查/Unknown Vendor 占位全路径）
- `go test ./internal/api/v1/network/ -count=1` — PASS
- `go test ./internal/services/system/ -run "GetOrSetResidue" -count=1` — PASS（Phase 92 防线不倒退）

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1-2 | struct/constructor 改 base.CacheProvider，dataCache 删除 | ✅ |
| 3-4 | 4 处 GetOrSet 全改 base.GetOrSetJSON（vendor + 三查询位点） | ✅ |
| 5-6 | heatmap wrapper 保留 fallback 直查语义 | ✅ |
| 7-8 | router 接线 + 字面断言更新 | ✅ |
| 9-10 | 测试文件无 system import（无环） | ✅ |
| 11 | fixture 编译期断言存在 | ✅ |
| 12-14 | build + services 测试 + network 测试通过 | ✅ |

## Deviations from Plan

**[Rule 2 - Bug fix] GetVendor 裸装配 panic** — Found during: Task 4 | Issue: 计划注释称"base.GetOrSetJSON 内部 nil 防护"，实际 GetOrSetJSON 对 nil provider 直接 panic（base.GetOrSetJSON 无 nil guard，测试 `newMhq7905` 裸装配 cache=nil 即崩） | Fix: GetVendor 保留 `s.cache == nil → lookupVendorFromDB` 直查分支，等价原手写 nil 语义 | Files: mac_history_query_service.go | Verification: TestMhq7905_GetVendor_CacheAndDB 5) 裸装配场景 PASS | Commit: 44301ef

**[Rule 1 - Behavior lock] 缓存值断言形态变化** — Found during: Task 4 | Issue: GetOrSetJSON 走 DataCacheService JSON 序列化，mem.Get 直查返回 `"\"Cache Vendor 7905\""`（带引号），原测试断言裸字符串失败 | Fix: 测试断言改为 JSON 引号形态（行为等价——缓存命中路径 GetVendor 仍返回裸值，仅直查缓存层的存储形态变化） | Files: mac_history_query_service_79_05_test.go | Verification: 测试 PASS | Commit: 44301ef

**[Rule 3 - Test adaptation] 字面断言多行形态** — Found during: Task 3 | Issue: 计划的单行断言子串与 router 实际多行调用格式不匹配 | Fix: 改三段式断言（构造调用 + NewCacheProvider + CacheConfigService 各自 Contains） | Files: setup_routers_test.go | Verification: 测试 PASS | Commit: 5565fc7

**Total deviations:** 3 auto-fixed (2 behavior-lock tests + 1 nil-guard fix). **Impact:** 零业务语义变化；nil 防护与 fallback 语义均与原手写实现等价。

## Self-Check: PASSED

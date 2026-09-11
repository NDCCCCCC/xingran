---
phase: "103"
plan: "04"
subsystem: cache-invariants
tags: [cache, testing, ast-guard, invariants]
requires:
  - "Phase 92 cache_invariants_92_test.go (system/operations cache_impl 口径)"
provides:
  - "cache_invariants_103_test.go — servicesRoot/asset/rpa 扩口 + 手写 cache-aside AST 检测"
affects: []
tech-stack:
  added: []
  patterns:
    - "函数域内位置严格递增三元组检测（Get < Unmarshal < Set），替代行数窗口"
    - "阳性对照自检：git show 迁前形态注入验证 detector 必须命中"
key-files:
  created:
    - "internal/services/system/cache_invariants_103_test.go"
  modified: []
key-decisions:
  - "检测序列用函数域内位置序而非行数窗口——实证 selector_learner 迁前形态 Get@170 与 Set@227 相隔 57 行（中间夹 DB 查询主体），计划的 10/25 行窗口必然漏检"
  - "扩口实现为独立 103 版测试文件（复用 92 版 isEmptyInterface/servicesRoot92 helper），不改 92 版——两代防线并存，92 版继续锁 cache_impl 文件口径"
  - "排除 data_cache_service.go（DataCacheService.GetOrSet 是基础设施实现本体，非调用残留，D-06/D-07 in-place 决策）"
  - "GetJSON/SetJSON 内隐式序列化的变体（api_endpoint/dashboard/widget_data_fetcher）不含显式 json.Unmarshal，不属 D-103-18 锁定序列，warning 档不误报"
requirements-completed:
  - "CONV-04"
duration: "20 min"
completed: "2026-09-08T01:35:00Z"
---

# Phase 103 Plan 04: cache invariants 扩口 Summary

cache 防线从 Phase 92 的 system/operations cache_impl 文件口径扩口到 servicesRoot + asset + rpa 全域，新增手写 cache-aside 标志性序列 AST 检测；Phase 103 收敛面 4 文件两类残留锁零。

## What Was Built

### Task 1: cache_invariants_103_test.go（commit 37dca76）

**TestNoInterfaceGetOrSetResidue103（D-103-16/D-103-17）**
- 扫描面：services root + asset + rpa 全部非测试 `.go` 文件（非递归 glob；排除 `data_cache_service.go` 基础设施实现本体）
- 检测口径复用 92 版：`.GetOrSet` + FuncLit 实参签名 `(interface{}, error)` / `(any, error)`
- 白名单 `allowedResidues103` 初始空 = 期望 0；全域硬失败档（该域从未有登记豁免，出现即倒退）

**cacheAsideResidue103 + TestNoHandwrittenCacheAside（D-103-18/D-103-19）**
- 标志性序列：同一函数体内位置严格递增三元组 `cache.Get/GetJSON < json.Unmarshal < cache.Set/SetJSON`
- 硬失败档：收敛面 4 文件（mac_history_query_service / mac_history_heatmap_service / reconciliation_service / selector_learner.go）== 0
- warning 档：同域其余文件计数日志（v1.31+ 迁移候选）

**检测器校准过程（防 silent-skip 的实证）**
- 初版用计划建议的行数窗口（Unmarshal ≤15 行 / Set ≤25 行）→ 阳性对照失败
- 实证：迁前 selector_learner 的 Get@170 与 Set@227 相隔 57 行（DB 查询主体居中）→ 行数窗口结构性漏检
- 修正为位置序三元组 → 阳性对照命中 `zz_old_selector_probe.go:170` ✓ → 删除探针后正式测试 PASS

## Verification Results

- `go build ./...` — PASS
- `go test ./internal/services/system/ -run "GetOrSetResidue103|HandwrittenCacheAside|GetOrSetResidue" -count=1 -v` — 3 tests PASS
- `go test ./internal/services/system/ -count=1`（全包）— PASS
- `go test ./internal/services/ -run "MACHistory|Heatmap|Reconciliation|Selector" -count=1` — PASS
- 阳性对照（迁前形态注入 + TestDebugProbeScan 断言 detector 必须命中）— PASS 后清理

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | hardDirs 扩口 servicesRoot + asset + rpa | ✅（scanDirs103） |
| 2 | cacheAsideResidue AST 检测函数 | ✅ |
| 3 | 4 文件手写 cache-aside 计数 == 0 | ✅ |
| 4 | 4 文件 interface{} 闭包 GetOrSet == 0 | ✅ |
| 5-7 | build + invariants + 三域测试通过 | ✅ |

## Deviations from Plan

**[Rule 2 - Plan spec bug] 行数窗口漏检** — Found during: Task 1 | Issue: 计划的检测规则（Get 后 5-10 行内 Unmarshal/Set）与真实迁前形态不符——selector_learner 的 Get@170 与 Set@227 相隔 57 行，按计划实现会 silent-skip | Fix: 改用函数域内位置严格递增三元组（g < u < s），并以迁前文件做阳性对照验证 detector 确实命中 | Files: cache_invariants_103_test.go | Verification: 阳性对照 PASS @170 + 正式 3 测试 PASS | Commit: 37dca76

**[Rule 3 - Structure] 独立 103 版文件而非改 92 版** — Found during: Task 1 | Issue: 计划要求改 cache_invariants_92_test.go 的 hardDirs，但 92 版的 `cacheImplFiles92` 文件口径（`*_cache_impl.go` glob）与本相需要的「全部 .go 文件」口径不同，硬改会破坏 92 版语义 | Fix: 新建 cache_invariants_103_test.go 复用 92 版 helper（isEmptyInterface/servicesRoot92），两代防线并存 | Files: cache_invariants_103_test.go | Verification: 双版本测试同时 PASS | Commit: 37dca76

**Total deviations:** 2 auto-fixed (1 detector spec fix + 1 file structure). **Impact:** 检测器经阳性对照实证有效；92 版防线原样保留。

## Self-Check: PASSED

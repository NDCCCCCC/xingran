---
phase: "103"
plan: "03"
subsystem: rpa-selector-learner-cache
tags: [cache, refactor, base-cache-provider, rpa]
requires:
  - "rpa/selector_learner hand-written JSON cache-aside (Get + Unmarshal / Marshal + Set)"
provides:
  - "selector_learner GetBestSelector on base.GetOrSetJSON via getBestSelectorCached wrapper"
affects:
  - "internal/services/rpa/service.go"
  - "internal/services/rpa/ai_service.go"
  - "internal/api/v1/rpa/rpa_router.go"
  - "internal/core/core.go"
tech-stack:
  added: []
  patterns:
    - "getBestSelectorCached wrapper (best-effort silent: cache err → DB recompute, never surfaces cache error)"
    - "fakeSelectorCache in-test base.CacheProvider with injectable get/set closures"
key-files:
  created: []
  modified:
    - "internal/services/rpa/selector_learner.go"
    - "internal/services/rpa/ai_service.go"
    - "internal/services/rpa/service.go"
    - "internal/api/v1/rpa/rpa_router.go"
    - "internal/core/core.go"
    - "internal/services/rpa/ai_selector_excel_test.go"
key-decisions:
  - "best-effort 静默语义等价迁移：cache 层错误 warn 后 DB 直查重算并返回重算结果（原实现 Get 失败静默走 DB + Set 失败 `_ =` 忽略的合成等价）"
  - "computeBestSelector 提取为主体查询函数，wrapper 只做 cache 编排——避免闭包内嵌大段查询逻辑"
  - "NewServiceGroup 加第 6 参 cacheProvider，cacheInstance 原样保留（CredentialService/TaskService 不在本 plan 范围，T-103-08 accept）"
  - "fakeSelectorCache 改造为 base.CacheProvider：GetOrSet 实现真实读穿透语义（get 闭包命中 → Unmarshal；未命中 → query → set 闭包记录 → Unmarshal 回填）"
requirements-completed:
  - "CONV-03"
duration: "15 min"
completed: "2026-09-08T01:05:00Z"
---

# Phase 103 Plan 03: rpa/selector_learner 迁移 base.CacheProvider Summary

selector_learner 手写 JSON cache-aside 收敛 base.GetOrSetJSON + getBestSelectorCached wrapper（best-effort 静默语义保留）；完整接线链 service.go → ai_service.go → selector_learner.go 同步更新，rpa_router 两处 + core.go 注入 system.NewCacheProvider。

## What Was Built

### Task 1: selector_learner.go（commit d5637a1）
- struct 字段 + `NewSelectorLearner` 入参 `cache.Cache` → `base.CacheProvider`（D-103-1/D-103-2/D-103-3）
- 新增 `getBestSelectorCached` wrapper：`base.GetOrSetJSON[*SelectorRecommendation]` TTL `30*time.Minute`（D-103-12）；err → warn + `computeBestSelector` 直查重算（D-103-8 best-effort 静默）
- `computeBestSelector` 提取原 GetBestSelector 主体（含 mu.RLock、successes/failures 查询、得分计算）；`GetBestSelector` 一行委托 wrapper
- `RecordSuccess` 失效：`_ = l.cache.Delete` → `base.Invalidate(ctx, l.cache, []string{...}, "SelectorLearner")`（D-103-14）
- 删除未用 `encoding/json` import；新增 applogger

### Task 2+3: ai_service.go + service.go（commit d5637a1）
- `NewAIService(cfg, db, cache cache.Cache)` → `NewAIService(cfg, db, cacheProvider base.CacheProvider)`；`NewSelectorLearner(db, cacheProvider, cfg)` 直接透传
- `NewServiceGroup` 新增第 6 参数 `cacheProvider base.CacheProvider`；`cacheInstance` 保留传 CredentialService/TaskService（T-103-08 兼容决策）

### Task 4: rpa_router.go + core.go（commit d5637a1）
- `SetupPublicWorkerRouter` / `SetupRPARouter` 两处 `NewServiceGroup(..., system.NewCacheProvider(core.DataCacheService))`
- `core.go:1055` RPA scheduler handler 接线同形态（`system.NewCacheProvider(c.DataCacheService)`）

### Task 5: ai_selector_excel_test.go（commit d5637a1）
- `fakeSelectorCache` 改造满足 base.CacheProvider 全 9 方法：`GetOrSet` 实现真实读穿透（get 闭包命中 → Unmarshal 返回；未命中 → query() → set 闭包记录 → Marshal/Unmarshal 回填 dest）；Get/Set/Delete 保留可注入闭包；其余 6 方法 NoOp；编译期断言 `var _ base.CacheProvider`
- 四处调用点：:286/:328/:389 fakeSelectorCache 直接传入（已满足接口，无需包裹）；:577 `NewServiceGroup` 补第 6 参数 `&base.NoOpCacheProvider{}`
- 测试断言「结果应写缓存」（setKeys 非空）与「缓存命中直接返回」（#cached）在 fake GetOrSet 语义下原样通过

## Verification Results

- `go build ./...` — PASS
- `go vet ./internal/services/rpa/` — PASS
- `go test ./internal/services/rpa/ -run "Selector|ServiceGroup" -count=1` — PASS
- `go test ./internal/services/system/ -run "GetOrSetResidue" -count=1` — PASS

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1-4 | selector_learner struct/constructor/wrapper/Invalidate | ✅ |
| 5-6 | ai_service.go 入参 + 透传 | ✅ |
| 7 | service.go 第 6 参数 + cacheInstance 保留 | ✅ |
| 8-9 | rpa_router 两处 + core.go 接线 | ✅ |
| 10-11 | fakeSelectorCache 改造 + 调用点适配 | ✅ |
| 12-13 | build + Selector 测试通过 | ✅ |

## Deviations from Plan

None - plan executed exactly as written.

注（非偏差）：计划的 :286/:328 调用点建议 `system.NewCacheProvider(c)` 包裹，实际 fakeSelectorCache 直接实现 base.CacheProvider（计划 :348 自己的注释即此形态），无需包裹——与计划意图一致（编译通过 + 语义保留）。

## Self-Check: PASSED

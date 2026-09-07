---
phase: 103-cache-closure-base-authority
reviewed: 2026-09-08T00:00:00Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - internal/services/mac_history_query_service.go
  - internal/services/mac_history_heatmap_service.go
  - internal/services/mac_history_test_fake_cache.go
  - internal/api/v1/network/mac_history_router.go
  - internal/services/mac_history_query_service_79_05_test.go
  - internal/services/mac_history_tail_79_05_test.go
  - internal/api/v1/network/setup_routers_test.go
  - internal/services/asset/reconciliation_service.go
  - internal/api/v1/asset/reconciliation_router.go
  - internal/api/router.go
  - internal/services/asset/asset_gapfill_test.go
  - internal/api/v1/asset/reconciliation_permission_test.go
  - internal/services/rpa/selector_learner.go
  - internal/services/rpa/ai_service.go
  - internal/services/rpa/service.go
  - internal/api/v1/rpa/rpa_router.go
  - internal/core/core.go
  - internal/services/rpa/ai_selector_excel_test.go
  - internal/services/system/cache_invariants_103_test.go
findings:
  critical: 0
  warning: 4
  info: 4
  total: 8
status: issues_found
---

# Phase 103: Code Review Report

**Reviewed:** 2026-09-08
**Depth:** standard
**Files Reviewed:** 19
**Status:** issues_found

## Summary

本相将 mac_history（query/heatmap）、asset/reconciliation、rpa/selector_learner 三个域的手写 cache-aside / interface{} 闭包 GetOrSet 收敛到 `base.GetOrSetJSON[T]` + `base.CacheProvider`，并新增 AST 不变量测试。`go build ./...` 与全部受影响包测试（services / services/asset / services/rpa / services/system / api/v1/network / api/v1/asset）均通过。

接线审查结论：router/core 各装配点（`system.NewCacheProvider(core.DataCacheService)`）正确，`DataCacheService == nil` 时落到 `NoOpCacheProvider`，与原 `core.Cache` nil 语义等价；mac_history heatmap 生产接线从 `nil, nil` 修正为带 cache + perfConfig，属设计内修复；`RecordSuccess` 失效改 `base.Invalidate` 消除了原 nil-cache panic 隐患，均为改善。

核心发现集中在 **fallback wrapper 的错误语义与生产 provider 实际语义的错配**：本相追踪了完整错误传播链 `base.GetOrSetJSON` → `system/cacheProviderAdapter` → `DataCacheService.GetOrSet`（`internal/services/data_cache_service.go:104-130`），确认该 provider **缓存读错误被吞掉（内部回源）、缓存写失败仅 warn 不返回、唯一能穿透的 error 是 DB 查询失败**。由此推出四个 Warning：(1) 各 fallback wrapper 的「cache 错误降级直查」分支在生产上只可能由 DB 失败触发，导致昂贵聚合在 DB 故障时执行两次；(2) selector_learner 负结果（nil）会被缓存 30 分钟，与 wrapper 注释「不缓存占位」矛盾；(3) mac_history 声明的 D-103-6「严格语义」在生产装配下不可达，且 fixture 的注错能力无任何测试使用；(4) reconciliation 脏缓存回源后不覆写脏条目。无 Critical 级问题。

## Warnings

### WR-01: fallback wrapper 把 DB 查询错误当缓存错误，DB 故障时昂贵计算被执行两次

**File:** `internal/services/asset/reconciliation_service.go:820-844`、`internal/services/rpa/selector_learner.go:183-195`、`internal/services/mac_history_heatmap_service.go:131-140`
**Issue:** 三个 wrapper 的降级逻辑均为「GetOrSetJSON 返回 err → 记 warn → 再次调用 compute 函数」。但生产 provider（`DataCacheService.GetOrSet`，经 `system/cacheProviderAdapter`）只在 **query 闭包失败** 时返回 error（读错误内部吞掉回源、写错误 warn 后返回 nil）。因此 fallback 分支在生产上唯一可达的触发路径就是 DB 查询失败：`computeByWorkstation`（6 步聚合查询）与 `computeBestSelector` 会先在 GetOrSetJSON 内失败一次，再在 wrapper 里重跑一次——**在 DB 最脆弱的时刻把故障查询放大一倍**。reconciliation 与 selector 是本次迁移引入的新行为（原实现 DB 失败只执行一次即透传）；heatmap 与旧代码行为持平（旧实现同样二次直查），但本次重构为独立 wrapper 时未修复。同时 warn 日志把 DB 故障误报为「cache 走直查」，误导排障。
**Fix:** 意图是「缓存层故障不阻断」时，降级必须在 provider 层做（如包一层 fallback 装饰器），或在 wrapper 中区分错误来源；在现有接口无法区分时，最直接的修复是去掉二次调用、直接透传（生产行为不变，故障路径不再翻倍）：

```go
func (s *reconciliationServiceImpl) getByWorkstationWithCache(ctx context.Context, cacheKey string, wsID string) (*ByWorkstationResponse, error) {
	resp, err := base.GetOrSetJSON[*ByWorkstationResponse](ctx, s.cache, cacheKey, reconciliationHealthCacheTTL, func() (*ByWorkstationResponse, error) {
		return s.computeByWorkstation(ctx, wsID)
	})
	if err != nil {
		// 生产 provider 仅在 DB 查询失败时返回 error —— 二次重试只会放大故障
		return nil, err
	}
	...
}
```

若确需保留「provider 可能注错」的降级能力（测试 fixture 场景），应显式注明并接受 DB 失败双查的代价，或在 query 闭包内对 error 打类型标记供 wrapper 识别。

### WR-02: selector_learner 负结果（nil）被缓存 30 分钟，与 wrapper 注释的「不缓存占位」不变量矛盾

**File:** `internal/services/rpa/selector_learner.go:174-195`（注释 :180，GetOrSetJSON 调用 :186）
**Issue:** wrapper 注释声明「DB 查询返回 (nil, nil)（无记录）时同样静默返回，**不缓存占位**」。原实现确实只在 `best != nil` 时 `cache.Set`。迁移后生产 provider `DataCacheService.GetOrSet`（`internal/services/data_cache_service.go:110-129`）对 query 返回的 nil 会 `json.Marshal(nil)` = `"null"`并无条件 `Set`——即**无记录结果被缓存 30 分钟**，文档不变量被违反，行为相对迁移前发生漂移。当前功能影响有限（命中 `"null"` 反序列化后仍返回 nil,nil；`RecordSuccess` 会失效该键），但注释作为契约会在后续维护中误导（例如有人据此假设负结果不占缓存键）。
**Fix:** 二选一：(a) 保语义——query 闭包在无记录时返回哨兵错误，wrapper 捕获后转 (nil, nil)：

```go
var errNoSelector = errors.New("no selector record")
// query 闭包内：
if best == nil { return nil, errNoSelector }
// wrapper 内：
if errors.Is(err, errNoSelector) { return nil, nil }
```

(b) 至少修正注释，明确「负结果会以 JSON null 形态缓存 30 分钟，由 RecordSuccess 的 Invalidate 清除」。

### WR-03: mac_history 声明的 D-103-6「严格语义（cache 写失败返回 error）」在生产装配下不可达，且 fixture 注错能力无测试使用

**File:** `internal/services/mac_history_query_service.go:258-271`（GetVendor）、`302-313`（QueryPortHistory）、`424-434`（QueryDeviceHistory）、`824-834`（QueryConnectionStats）；`internal/services/mac_history_test_fake_cache.go:26-31,39-51`
**Issue:** 三处注释声明收敛后「cache 写失败返回 error (D-103-6 严格语义)」。但生产链路 `system.NewCacheProvider(core.DataCacheService)` → `DataCacheService.GetOrSet` 中：缓存读失败被吞掉（内部回源）、缓存写失败仅 `applogger.Warnf` 后返回 nil（`internal/services/data_cache_service.go:125-127`）。因此 `GetOrSetJSON` 在生产上**永远不可能**因缓存层错误返回 error——所谓「严格语义」分支是死路径，四个域（mac_history 严格 vs heatmap/reconciliation/selector 降级，D-103-6/7/8/9）的实际生产行为完全相同。测试 fixture `fakeMACHistoryCacheProvider` 的 `getOrSetErr`/`deleteErr` 注错字段注释声称「用于错误透传（D-103-6）与 fallback 直查（D-103-9）语义的回归测试」，但全仓 grep 确认**没有任何测试注入这两个错误**（测试全部经 `newFakeMACHistoryCacheProvider(mem)` 构造）——被声称的回归测试从未编写。契约文档与现实不符，未来维护者可能基于「写失败会报错」做出错误决策。
**Fix:** (a) 修正 mac_history 各处注释，写明「经 DataCacheService 装配时缓存层错误不外泄（读吞掉/写降 warn），仅 DB 查询失败会返回 error」；(b) 补上 fixture 注错的回归测试，或删除无人使用的 `getOrSetErr`/`deleteErr` 字段；(c) 若 D-103-6 的严格语义是真实需求，需要 provider 层扩展（返回 Set 错误的实现），属后续立项。

### WR-04: reconciliation 脏缓存回源后不覆写脏条目，TTL 窗口内每请求重复 warn + 重算

**File:** `internal/services/asset/reconciliation_service.go:831-839`
**Issue:** 脏缓存防御分支（`resp.Workstation.ID == ""`）回源 `computeByWorkstation` 后直接返回，**不写回缓存也不删除脏条目**。原手写实现在 compute 后总是 `cache.Set`，脏条目立即被覆写；迁移后 `GetOrSetJSON` 只在 miss 时 Set，命中路径（含脏命中）不落盘——脏条目在 5 分钟 TTL 内持续存在，窗口内每个请求都走「脏命中 → warn → 全量重算（6 步聚合）」，与旧实现行为不等价。
**Fix:** 回源前先删除脏条目（Delete 在 CacheProvider 上现成可用），或回源成功后显式失效/覆写：

```go
} else if resp != nil && resp.Workstation.ID == "" {
	applogger.Warnf("[reconciliation] GetByWorkstation 脏缓存回源 wsID=%s", wsID)
	_ = s.cache.Delete(ctx, cacheKey) // 清掉脏条目，避免 TTL 窗口内每请求重复回源
	resp, err = s.computeByWorkstation(ctx, wsID)
	if err != nil {
		return nil, err
	}
}
```

## Info

### IN-01: getByWorkstationWithCache 理论上可返回 (nil, nil)，handler 侧将 nil 解引用

**File:** `internal/services/asset/reconciliation_service.go:840-843`（关联 `internal/api/v1/asset/reconciliation_handler.go:243`）
**Issue:** 缓存中若存在 JSON `null` 条目，`GetOrSetJSON` 反序列化后 resp 为 nil 且 err 为 nil，脏缓存检查（要求 `resp != nil`）被跳过，最终 `return nil, nil` 违反方法契约；handler 的 `result.Visible = ...` 将 panic。当前 compute 路径不会写入 null，触发概率极低，但防御性缺口真实存在。
**Fix:** 脏缓存条件改为 `resp == nil || resp.Workstation.ID == ""`，同时覆盖两种脏形态。

### IN-02: 新增 fixture 文件不符合 gofmt；注错字段为死代码

**File:** `internal/services/mac_history_test_fake_cache.go:26-31`
**Issue:** 本相新增文件被 `gofmt -l` 标记（struct 字段对齐，:28-29）；同时 `getOrSetErr`/`deleteErr` 无任何使用方（见 WR-03）。
**Fix:** `gofmt -w internal/services/mac_history_test_fake_cache.go`；注错字段去留与 WR-03 (b) 一并处理。

### IN-03: selector_learner 两个 impl 级方法无生产调用方（死代码）

**File:** `internal/services/rpa/selector_learner.go:389-399`（GetSelectorHistory）、`401-446`（AnalyzeSelectorTrends）
**Issue:** 两方法不在 `SelectorLearner` 接口上，全仓唯一调用方是 `ai_selector_excel_test.go:415,420` 的测试。按项目纪律（废弃函数直接删除不留兼容壳），应删除或提升为接口方法。
**Fix:** 若为规划中 API 则加入 `SelectorLearner` 接口；否则连同对应测试断言一并删除。

### IN-04: 测试名与注释误述被测行为（字段已改名、注入的是 NoOp 而非 nil）

**File:** `internal/services/mac_history_tail_79_05_test.go:715-721`
**Issue:** `pg_fake_query_skips_cache_branch` 注释称「dataCache == nil 时应直查(缓存装饰分支关闭)」，但实际传入 `&base.NoOpCacheProvider{}`（非 nil，且字段已改名 cache）——测试实际走的是缓存分支的 NoOp 路径，名称与注释均与被测行为不符。
**Fix:** 重命名为如 `pg_fake_query_noop_provider` 并修正注释。

### IN-05: cache_invariants_103 守卫的检测口径存在已知盲区

**File:** `internal/services/system/cache_invariants_103_test.go:136-149`（isCacheReadCall103）、`197-218`（时序判定）
**Issue:** (a) 手写 cache-aside 检测仅匹配接收者字面名为 `.cache` 的 `Get/GetJSON/Set/SetJSON`——未来经 `s.dataCache.Get` 或局部变量名（如 `redisCli`）的手写回源将绕过检测（假阴性）；(b) 三元组时序判定只要求 Set 在 Get 之后（不要求在 Unmarshal 之后），`Get → Set → Unmarshal` 形态会误报（假阳性，偏保守方向，fail-safe）。作为防线可接受，建议在注释中显式登记口径边界，防止后续维护者误以为全覆盖。
**Fix:** 注释补充盲区说明；如需收紧可扩展接收者名匹配（cache|dataCache|redis）。

---

_Reviewed: 2026-09-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

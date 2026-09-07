# Phase 103: 缓存闭包收敛 base 单一权威 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-07
**Phase:** 103-cache-closure-base-authority
**Areas discussed:** Cache 抽象层、Vendor cache 纳入、错误语义保留、扫描扩口口径、selector TTL、Heatmap fallback、失效路径顺手治理

---

## Cache 抽象层（三域统一方案）

| Option | Description | Selected |
|--------|-------------|----------|
| 全部迁移到 CacheProvider（结构性重构） | 三域全部把 cache 字段从 cache.Cache / *DataCacheService 改为 base.CacheProvider；与 Phase 92 系统化迁移完全一致 | ✓ |
| 保留底层 + 加 adapter | selector_learner/reconciliation 保留 cache.Cache 不动；为它们写 thin adapter 实现 CacheProvider 9 方法 | |
| dataCache 保留 + 局部走 base；其他切 CacheProvider | mac_history 保留 dataCache + 写 dataCache→CacheProvider 适配器；asset/rpa 直接切 | |

**User's choice:** 全部迁移到 CacheProvider（结构性重构）
**Notes:** 用户接受结构性重构带来的多处构造方接线改动；与 Phase 92 既定方向一致（system/operations 走 `*_cache_impl.go` + `CacheServiceBase`，本相三域走轻量化：struct 字段直接持有 `CacheProvider`，不强制嵌入 `CacheServiceBase`）。

---

## Vendor cache-aside (:259-281) 纳入本相？

| Option | Description | Selected |
|--------|-------------|----------|
| 纳入本相 | 作为 mac_history 域收口的一部分，迁 base.GetOrSetJSON[T] | ✓ |
| defer 到 Phase 107 TODO 阶段 | 本相只迁 3+1 个 GetOrSet 闭包位；OUI vendor cache-aside 留作 deferred item | |
| 本相删除该 cache-aside | 改为直接 DB 查询替代，缓存删除彻底 | |

**User's choice:** 纳入本相
**Notes:** 与 F-10 主题一致 + 避免 107 TODO 阶段再回头；OUI vendor 是 cold path 但属同族 legacy 模式。

---

## 错误语义保留

| Option | Description | Selected |
|--------|-------------|----------|
| 逐域保留原语义 | base.GetOrSetJSON 提供 strict 形态；reconciliation/selector_learner 仍保持 best-effort wrapper 保留 warn 语义；回归测试逐域锁 | ✓ |
| 全部统一为 fail-on-set | reconciliation/selector_learner 改为同语义，请求会因缓存抖动而失败 | |
| 扩展 base 加 BestEffortGetOrSetJSON 变体 | 在 base 包新增 best-effort 形态，三域统一用它 | |

**User's choice:** 逐域保留原语义
**Notes:** cache 抖动不该杀业务请求；逐域 wrapper + 回归测试锁等价语义。mac_history query_service 三处直接走 strict 形态（与 Phase 92 一致）；reconciliation/selector_learner/heatmap 用 wrapper 保留 warn/fallback 语义。

---

## invariants 扫描扩口

| Option | Description | Selected |
|--------|-------------|----------|
| 显式文件清单 + 包路径 | 新测试或同文件新增，传入显式包路径 + 显式文件名清单，逐文件 AST 扫描 | ✓ |
| 新通配（所有非 _test.go + 含 cache 字样） | 扩 GetOrSet 包扫描面，命中含 (interface{},error) 闭包 GetOrSet 即 fail | |
| 只扩 warning 档到新包，硬失败仅原 system/operations | services 根/asset/rpa 仅 warning 计数不 fail | |

**User's choice:** 显式文件清单 + 包路径
**Notes:** 与 Phase 92 双档惯例一致（硬失败 + warning）；本相收敛面（4 文件）硬失败归零，其他非主清单文件仅 warning 档。

---

## selector_learner TTL 处理

| Option | Description | Selected |
|--------|-------------|----------|
| 保持字面量 30*time.Minute | selector 学习频率稳定，运维不调 | ✓ |
| 纳入 CacheConfigService | 在 perfCacheConfigService seed 中加 rpa.selector.ttl | |
| 抽 pkg/constants 具名常量 | RGOSelectorCacheTTL 入驻 pkg/constants/time.go | |

**User's choice:** 保持字面量 30*time.Minute
**Notes:** 与 base 迁移主题独立；selector 是 RPA 域内自有参数，不与其他 cache 模块共享 TTL 体系。

---

## mac_history_heatmap fallback 语义

| Option | Description | Selected |
|--------|-------------|----------|
| 保留 fallback | 包一层 try base.GetOrSetJSON, 失败时调 queryHeatmapFromMV 直查 + warn 日志 | ✓ |
| 直接使用 base.GetOrSetJSON（硬语义） | 迁移后 cache 失败就是请求失败 | |

**User's choice:** 保留 fallback
**Notes:** heatmap 是 v1.30 Phase 99 高频查询；降级到直查是合理保护，不让 cache 抖动击穿全量。

---

## 失效路径治理

| Option | Description | Selected |
|--------|-------------|----------|
| 本相仅迁 GetOrSet 闭包，失效不动 | CONV-02/03 只点读路径；其他 cache.Delete 路径留原样 | |
| 顺手接 base.Invalidate | 同期 Save / Update / Delete 路径也走 base.Invalidate / base.InvalidatePattern | ✓ |

**User's choice:** 顺手接 base.Invalidate
**Notes:** 与 Phase 92 "唯一失效底层" 原则一致；多动几个位点但统一面；nil 防护 + warn 日志内聚到 helper。

---

## Claude's Discretion

- mac_history_heatmap 的 fallback helper 命名（`getHeatmapWithFallback` vs `tryCacheThenQuery`）与包内位置
- reconciliation "脏缓存防御" `Workstation.ID != ""` 检查迁 wrapper 后的位置（wrapper 内 vs 调用方）
- invariants 新测试文件的命名（`cache_invariants_103_test.go` vs 扩展现有 `cache_invariants_92_test.go`）与放置包
- 三域 `Invalidate` 调用的 module 字符串常量（如 `"MACHistoryQuery"` / `"Reconciliation"` / `"SelectorLearner"`）
- 既有 `dataCache` 字段删除时机（同 commit 删除 vs 留 deprecated 字段过渡；按 D-01 不留兼容壳同 commit 删除）

## Deferred Ideas

- `CacheProvider` 全仓持有化（network / knowledge / duty / workorder cache 字段未来 phase 收敛）
- selector_learner TTL 接入 CacheConfigService
- `DataCacheService` 物理合并/退役（D-04 范围外）
- `CacheProvider` interface 增加 `GetJSON`/`SetJSON` 便捷方法
- 统一 `vendor` / `selector_learner` 的 invalidation hook 进 operlog 链路（Phase 107 顺带）
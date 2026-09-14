---
milestone: v1.31
status: defined
---

# Phase 103: 缓存闭包收敛 base 单一权威 - Context

**Gathered:** 2026-09-07
**Status:** Ready for planning

<domain>
## Phase Boundary

v1.29 Phase 92 缓存统一的**补遗收口**——三域 (mac_history + asset reconciliation + rpa selector) 残余 legacy `interface{}` 闭包式 GetOrSet 与手写 cache-aside 全部迁 `base.GetOrSetJSON[T]`；`cache_invariants_92_test.go` 扫描口径扩口至 services 根 / asset / rpa（硬失败档），守护新收敛面不回潮。

成功标准（ROADMAP SC-1..5）：
1. mac_history 域：4 处 legacy GetOrSet（:309 / :434 / :837 query_service + :118 heatmap_service）+ vendor 手写 cache-aside（:259-281）全部收敛 `base.GetOrSetJSON[T]`，services 根不再有 `interface{}` 闭包式 GetOrSet
2. asset reconciliation：`GetByWorkstation` 手写读穿透（GetJSON 短路 + Marshal + Set）迁 `base.GetOrSetJSON[T]`，读穿透语义等价（命中短路/回源/回填时序 + `Workstation.ID != ""` 脏缓存防御 + warn-on-set 不阻断）
3. rpa selector：`GetBestSelector` / `SaveSelector` 手写 JSON cache-aside 迁 `base.GetOrSetJSON[T]`，TTL 保持 `30*time.Minute` 字面量
4. invariants 扩口：`cache_invariants_92_test.go` 扫描口径扩展至 services 根 / asset / rpa 包，显式文件清单 + 包路径，interface{} 闭包式 GetOrSet 硬失败
5. 回归纪律：`go test ./internal/services/...` 0 失败，七 gate 不倒退（含后端 coverage ≥78.33% 基线）

</domain>

<decisions>
## Implementation Decisions

### Cache 抽象层统一

- **D-103-1:** 三域全部迁移到 `base.CacheProvider`（结构性重构）——mac_history 字段 `dataCache *DataCacheService` 与 `cache cache.Cache`、reconciliation `cache cache.Cache`、selector_learner `cache cache.Cache` 统一改为 `cache base.CacheProvider`。与 Phase 92 系统化迁移（system/operations `*_cache_impl.go` 走 `CacheServiceBase`）完全一致。
- **D-103-2:** 构造方接线迁移——`NewMACHistoryQueryServiceWithCache` / `NewMACHistoryHeatmapService` / `NewReconciliationService` / `NewSelectorLearner` 入参从 `*DataCacheService` 或 `cache.Cache` 改为 `base.CacheProvider`；所有调用方（cmd/main.go 与各路由注册处）同步切。`cmd/main.go` 是单点，需 grep 全部调用点核对清单。
- **D-103-3:** 不引入 `cache.Cache → CacheProvider` adapter wrapper（区别于 D-103-1 全迁移方案）——保持结构性重构的干净，struct 字段直接持有 `CacheProvider` 接口值。`CacheProvider` 既有 `GetOrSet/Get/Set/Delete/DeleteByPattern/MGet/MDelete/Exists/SetTTL/GetTTL/GetStats` 全方法面，足够三域既有调用替换。

### Vendor cache-aside 纳入本相

- **D-103-4:** `mac_history_query_service.go:259-281`（OUI vendor manual cache-aside）**纳入本相**——作为 mac_history 域收口的一部分，迁 `base.GetOrSetJSON[T]`。同族 legacy 模式且审计台账行号已明示；不让 107 TODO 阶段回头。
- **D-103-5:** Vendor "Unknown Vendor" 占位缓存（:273-275）行为保留——`base.GetOrSetJSON` 闭包内返回 `"Unknown Vendor"`，TTL `24*time.Hour` 字面量保持（vendor lookup 是 cold path，不入 config TTL 体系）。

### 错误语义逐域保留

- **D-103-6:** **base.GetOrSetJSON 严格语义**用于 mac_history query_service 三处（:309/:434/:837）——cache 写失败即返回 error，与 Phase 92 既定语义一致。
- **D-103-7:** **reconciliation GetByWorkstation** 用 wrapper 保留 warn-on-set 不阻断语义——包一层 helper：`func (s *reconciliationServiceImpl) getByWorkstationWithCache(ctx, wsID)` 内部调用 `base.GetOrSetJSON`，捕获 set 失败 → warn 日志 → 返回 `cached`（即使 set 失败结果仍可用）；`Marshal` 错误保留静默。
- **D-103-8:** **selector_learner** 用 wrapper 保留 best-effort 语义——`base.GetOrSetJSON` 错误返回 nil/fallback，warn 日志；调用方拿 `nil` 视为 cache miss 重算。Set / Delete 失败静默（同现行为）。
- **D-103-9:** **mac_history_heatmap** 保留 fallback 直查语义——包一层 helper 内部 `try base.GetOrSetJSON, on error fallback queryHeatmapFromMV + warn 日志`。cache 抖动不击穿热力图查询。

### TTL 源

- **D-103-10:** **mac_history 域 TTL 走 `s.perfCacheTTL()` 不变**——经 `CacheConfigService` 注入；vendor TTL 保持 `24*time.Hour` 字面量（cold path）。
- **D-103-11:** **reconciliation TTL 走 `pkg/constants.reconciliationHealthCacheTTL` 不变**（既有具名常量）。
- **D-103-12:** **selector_learner TTL 保持字面量 `30*time.Minute`**（不迁 `CacheConfigService`，与 base 迁移主题独立；selector 学习频率稳定，运维不调）。
- **D-103-13:** 三域 TTL 全部行为等价，无业务语义变化；常量值与原值逐处一致。

### 失效路径顺手接 base

- **D-103-14:** **顺手接 `base.Invalidate`** —— reconciliation SaveRecord / selector_learner LearnSelector / mac_history 既有 `cache.Delete(ctx, cacheKey)` 路径同步改 `base.Invalidate(ctx, s.cache, []string{cacheKey}, "MODULE")`。与 Phase 92 "唯一失效底层" 原则一致；多动几个位点但统一面。nil 防护 + warn 日志内聚到 helper。
- **D-103-15:** **批量失效**（reconciliation 的 pattern 失效、selector_learner 的多 key 失效）走 `base.InvalidatePattern`——保持与既有失效语义一致。

### invariants 扫描扩口

- **D-103-16:** **扫描口径 = 显式文件清单 + 包路径**——不引入新通配。新增 `internal/services/system/cache_invariants_103_test.go`（或同文件加测试函数），传入显式目录列表（`internal/services/`、`internal/services/asset/`、`internal/services/rpa/`）+ 显式文件名清单（`mac_history_query_service.go`、`mac_history_heatmap_service.go`、`reconciliation_service.go`、`selector_learner.go`），逐文件 AST 扫描 `interface{}` 闭包式 GetOrSet + 手写 cache-aside 模式（`cache.Get` 后紧跟 `cache.Set` 含 `json.Marshal`/`json.Unmarshal`）。
- **D-103-17:** **硬失败档**——本相收敛面（4 个文件）interface{} 闭包式 GetOrSet 计数必须 == 0；扩展名望表 `allowedResidues103` 初始空。**warning 档**——其他非主清单文件仅日志不 fail（与 Phase 92 双档惯例一致）。
- **D-103-18:** **手写 cache-aside 扫描规则**——检测 `(s|l|c).cache.Get` 后紧跟 `json.Unmarshal` + 后续 `cache.Set` + `json.Marshal` 模式（即手工 read-through 形态）；命中即 fail。`vendor` 已知该模式，作为扫描 baseline 测试用例（迁后归零）。

### 回归守护

- **D-103-19:** **逐域回归测试**锁等价语义：
  - mac_history query_service 三处 GetOrSet：cache hit/miss + TTL 等价 + 错误透传测试（覆盖 `s.perfCacheTTL()` 路径）
  - mac_history heatmap fallback：cache 失败降级直查 + warn 日志断言
  - mac_history vendor：cache hit/miss + "Unknown Vendor" 占位 + 24h TTL 等价
  - reconciliation：`Workstation.ID != ""` 防御测试 + Marshal 失败静默 + Set 失败 warn 不阻断
  - selector_learner：best-effort 全部错误静默 + TTL 30min 等价
- **D-103-20:** **constructor 签名变更兼容性测试**——三域 struct 构造方签名变化，调用方接线测试（`cmd/main.go` 单点 + 各路由注册）。`go build ./...` 编译实证 + 调用方核对清单落盘（同 Phase 102 D-102-9 PAGI-01 caller audit 形态）。
- **D-103-21:** **TTL 等价 + cache key 字符串等价**断言（与 Phase 102 CACHE-01/02 等价测试同模式），确保迁后键值与 TTL 不漂移。

### 继承的锁定决策（不再讨论）

- **v1.31 D-01**: 台账 12 组全做；不留兼容壳
- **v1.31 D-02**: 行为变更附回归测试；七 gate（go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage）全程不倒退
- **v1.31 D-04**: captcha-background 1=启用语义禁改（QUIRK-80-03-D）
- **Phase 102 D-102-1/2**: mac vendor / rpa selector 缓存键**本相已注册**，103 直接迁闭包不重注册
- **Phase 99 D-03-10**: `pkg/constants` 是全项目唯一常量包
- **CLAUDE.md**: §Cache Service Convention / §Timeout 常量约定

### Claude's Discretion

- mac_history_heatmap 的 fallback helper 命名（`getHeatmapWithFallback` vs `tryCacheThenQuery`）与包内位置（独立 func vs 方法）
- reconciliation "脏缓存防御" `Workstation.ID != ""` 检查迁 wrapper 后的位置（wrapper 内 vs 调用方）
- invariants 新测试文件的命名（`cache_invariants_103_test.go` vs 扩展现有 `cache_invariants_92_test.go`）与放置包
- 三域 `Invalidate` 调用的 module 字符串常量（如 `"MACHistoryQuery"` / `"Reconciliation"` / `"SelectorLearner"`）
- 既有 `dataCache` 字段删除时机（同 commit 删除 vs 留 deprecated 字段过渡；按 D-01 不留兼容壳同 commit 删除）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与目标

- `.planning/notes/260907-audit-fix-tech-debt-findings.md` — F-10 (high) 缓存闭包残留台账原文：mac_history_query_service.go 4 处 + asset reconciliation + rpa selector 行号清单
- `.planning/REQUIREMENTS.md` §CONV-01..04 — 4 条 requirement 原文（含行号清单）
- `.planning/ROADMAP.md` §Phase 103 — Goal / SC-1..5 / Notes（mac_history 顺序约束）
- `.planning/phases/102-mechanical-constants/102-CONTEXT.md` — Phase 102 D-102-1/2 键注册决策（前置）

### 先例与约定

- `internal/services/system/cache_invariants_92_test.go` — Phase 92 invariants 双档 AST 扫描 + `allowedResidues` 白名单（CONV-04 扩口宿主）
- `internal/services/base/cache_functions.go` — `GetOrSetJSON[T]` / `Invalidate` / `InvalidatePattern` 三函数实现与薄委托红线
- `internal/services/base/cache_provider.go` — `CacheProvider` interface 9 方法定义
- `internal/services/base/cache_service_base.go` — `CacheServiceBase` 薄基类（TTLResolver）
- `internal/services/system/user_cache_impl.go` — Phase 92 `CacheServiceBase` 装饰器先例（userCacheService struct 形态参照）
- `.planning/notes/260907-audit-fix-tech-debt-findings.md` §F-10 — 行为变更边界与回归测试纪律依据
- `.planning/milestones/v1.30-phases/99-operations口径统一/99-CONTEXT.md` — D-03-1..4 caller audit 形态先例
- `CLAUDE.md` §Cache Service Convention — base.GetOrSetJSON / Invalidate / InvalidatePattern + CacheProvider 唯一权威

### 代码（迁移目标与参照）

- `internal/services/mac_history_query_service.go` — 4 迁移位点 (`:259-281` vendor + `:309` + `:434` + `:837` GetOrSet) + struct 字段 `dataCache` / `cache`
- `internal/services/mac_history_heatmap_service.go` — 1 迁移位点 (`:118` GetOrSet) + struct 字段 `dataCache` + fallback 直查 + warn 日志
- `internal/services/asset/reconciliation_service.go` — 1 迁移位点 (`:799-820` GetByWorkstation manual cache-aside) + struct 字段 `cache` + `Workstation.ID != ""` 防御 + Marshal/Set 错误语义
- `internal/services/rpa/selector_learner.go` — 2 迁移位点 (`:169-174` GetBestSelector + `:226` SaveSelector) + struct 字段 `cache` + 30min 字面量 TTL + best-effort 静默语义
- `internal/services/system/cache_keys.go` — 既有键族（mac vendor / rpa selector 键 Phase 102 已注册）
- `pkg/constants/cache.go` — CaptchaVerifiedKeyFormat 先例；本相不新增（TTL 三域或保持字面量或走 CacheConfigService）
- `pkg/cache/redis.go` — 低层 Cache interface（迁后三域不再直接引用）
- `internal/services/data_cache_service.go` — 既有 `DataCacheService`（迁后 mac_history 不再引用，作为基础设施保留）
- `internal/services/cache_config_service.go` — `CacheConfigService`（mac_history 仍走 `perfCacheTTL()`）
- `cmd/main.go` — 四个构造函数接线单点（迁移时 grep 全部调用点核对清单）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `base.GetOrSetJSON[T]` 泛型函数族（`internal/services/base/cache_functions.go`）——三域 4+1+1+2 = 8 处闭包位点直接调用
- `base.CacheProvider` interface（9 方法）——替换 `cache.Cache` / `*DataCacheService` 字段类型
- `base.CacheServiceBase` 薄基类（TTLResolver）——可选装饰器模式（Phase 92 system/operations 走法），本相三域因业务逻辑完整独立，**不强制走装饰器**，仅用 `base.GetOrSetJSON` + 字段替换即可
- `base.Invalidate` / `base.InvalidatePattern`（含 nil 防护 + warn 日志内聚）——替换手写 `cache.Delete` / pattern 失效
- `internal/services/system/cache_invariants_92_test.go` AST 扫描框架 + `allowedResidues` 白名单——CONV-04 扩口样式参照
- `pkg/constants.reconciliationHealthCacheTTL` 既有具名常量——reconciliation TTL 保持引用
- `CacheConfigService.perfCacheTTL()` ——mac_history TTL 既有 config 入口
- Phase 92 `user_cache_impl.go` 装饰器先例——本相不复制，三域保持原 struct 名 + 字段替换

### Established Patterns

- 行为等价重构 + 回归测试锁（v1.29/30/31 各 phase 一贯模式）
- 构造函数入参变化时 caller audit 落盘（Phase 99 D-03-1 PAGI-01 + Phase 102 D-102-9 形态）
- 错误语义保留必须显式 wrapper（reconciliation warn-on-set 不阻断 vs selector best-effort 静默 vs heatmap fallback 直查）
- AST 扫描双档（硬失败 + warning）与 `allowedResidues` 白名单 diff 可见（Phase 92 D-10② 惯例）
- 七 gate 实测落盘（后端 coverage ≥78.33% 基线）

### Integration Points

- `cmd/main.go` 单点接线四处构造函数（grep 全部调用点核对清单）
- 三域 `New*Service` 构造方签名变化向上冒泡：路由注册 / 测试构造方 / 任何其他 wiring 文件
- `cache_invariants_92_test.go` 既有扫描面保留（system + operations 硬失败），新文件仅扩口本相三域
- `internal/services/system/cache_keys.go` Phase 102 D-102-1/2 已注册键族直接使用，无需新增
- `internal/services/data_cache_service.go` 迁后 mac_history 不再引用；`DataCacheService` 作为基础设施保留（`base.CacheProvider` 实现基类）

</code_context>

<specifics>
## Specific Ideas

- mac_history vendor lookup TTL `24*time.Hour` 保留字面量（cold path，不入 config TTL 体系）
- reconciliation `Workstation.ID != ""` 防御作为 wrapper 内部首检查（迁 base 后仍需保留，base.GetOrSetJSON 不感知结构）
- selector_learner GetBestSelector 返回 `nil, nil` 时 cache miss 视为"已尝试无最佳 selector"——保留调用方处理路径，不引入额外 error 类型
- mac_history_heatmap fallback 直查函数命名 `queryHeatmapFromMV` 现存，迁 wrapper 内复用
- invariants 新测试文件名暂定 `internal/services/system/cache_invariants_103_test.go`（与 92 同包不同文件）；handwritten cache-aside AST 模式检测与闭包 GetOrSet 分两类断言
- wrapper 私有函数命名建议：`getReconciliationByWorkstationCached` / `getMACHistoryXxxCached` / `getBestSelectorCached` —— `_Cached` 后缀表示带缓存读穿透

</specifics>

<deferred>
## Deferred Ideas

- **`CacheProvider` 全仓持有化**（所有业务 cache 字段统一持有 `CacheProvider`，包括未来新增模块）——本相只动三域，其他模块（network / knowledge / duty / workorder）的 cache 字段待各自 phase 收敛
- **selector_learner TTL 接入 CacheConfigService**（与 mac_history 同架构）——本相 D-103-12 锁定保持字面量；如未来运维需调，单独 phase
- **`DataCacheService` 物理合并/退役**——D-04 范围外（"三层 adapter 物理合并"挂账），本相 mac_history 不再引用但 `DataCacheService` 保留为基础设施
- **`CacheProvider` interface 增加 `GetJSON`/`SetJSON` 便捷方法**（消除 `base.GetOrSetJSON` 内部的 `var result T` 中转）——属 base 包 API 演化，非本相主题
- **统一 `vendor` / `selector_learner` 的 invalidation hook 进 operlog 链路**——属 TODO 治理范畴（Phase 107 顺带）

</deferred>

---

*Phase: 103-cache-closure-base-authority*
*Context gathered: 2026-09-07*
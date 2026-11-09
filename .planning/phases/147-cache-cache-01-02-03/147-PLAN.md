---
phase: 147
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/services/asset/reconciliation_exception.go
  - internal/services/asset/cache_keys.go
  - internal/services/asset/reconciliation_service.go
  - internal/services/asset/asset_gapfill_test.go
  - internal/services/asset/reconciliation_cache_contract_test.go
  - internal/api/v1/asset/reconciliation_exception_router.go
  - internal/api/router.go                     # conditional: false —— 默认不动；ROADMAP.md:145 硬登记 149→151 顺序，仅在 §4.2 触发时才改一行并移交 Lead
  - .planning/phases/147-cache-cache-01-02-03/147-SUMMARY.md
# 文件面说明（W1 并行冲突面以 ROADMAP.md:115 为准）：
#   internal/api/v1/asset/reconciliation_exception_router.go —— 唯一承载 4 个写路径的装配点，
#     Task 2 T2-1 在此追加一行 `system.NewCacheProvider(core.DataCacheService)` 参数（+ 1 行 import）。
#   internal/api/v1/asset/reconciliation_router.go:35 与 internal/api/router.go:894 的 exceptionSvc
#     仅作 matcher 注入（不承载写路径），保持零参构造不动。
#   internal/api/router.go —— ROADMAP.md:145 已硬登记「149 先改、151 后改或拆 commit」；
#     147 默认「尽量不动」，本 plan 定案为**不动**（§4.2 仅在必要条件触发时才允许改一行并显式登记顺序）。
router_go_touched: false
autonomous: true
commit: false
---

# Phase 147 · CACHE 失效契约 — 执行计划（PLAN 01）

> 本文件是 **W1 并行 executor 的可直接执行计划**。W1 = `145 ∥ 146 ∥ 147`，三者写集合两两交集 = ∅（见 ROADMAP.md:115）。
> **禁止任何 git 写操作**（`add / commit / stash / checkout / merge / rebase / tag`）——本 plan `commit:false`，提交由 Lead 统一执行（ROADMAP.md:161）。
> 简体中文书写；技术标识符、路径、命令保留英文。

---

## 1. 必读顺序（开工前按序读，不重新调研根因）

| # | 文件 | 读什么 |
|---|------|--------|
| 1 | `.planning/STATE.md` | v1.55 active、Current Position（Phase 147 属 W1，与 145/146 并行）、提交由 Lead 统一 |
| 2 | `.planning/ROADMAP.md` | § Phase 147 详情（CACHE-01/02/03、4 条成功标准、文件面 `internal/services/asset/**`）、W1 `145∥146∥147`、Constraints、禁区「禁接 Redis 补 Delete」、:115 并行冲突面、:145 router.go 全局唯一归属硬登记 |
| 3 | `.planning/REQUIREMENTS.md` | CACHE-01~03 全文、5 项裁定之「DECISION-例外陈旧窗口 = 立即一致，CACHE-02 必做」、Constraints、Traceability（147=CACHE-01/02/03 唯一） |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` | **§⑤**（:119-132 根因/根治定义/方案对比表 ①②③/「不算根治」边界）、§3 处置排序、§4 未决问题第 6 条；本 phase 的 research 输入即此报告，**跳过领域研究** |
| 5 | `CLAUDE.md` / `AGENTS.md` | Handler-Service、`response.Success()/Error()`、operlog 写操作约定、`0=正常 1=停用`、Cache Key Prefix（`xingran:` 自动加、逻辑键不带前缀）、Go 单测就近 `*_test.go`、构建产物只落 `out/`、`go build ./...` 改后必跑 |
| 6 | 本仓现成范式（已核验，执行时参照其语义，不必再全局搜） | `internal/services/asset/cache_keys.go:112-134` `InvalidateWorkstationHealth`；`internal/services/base/cache_functions.go:50-76` `Invalidate` / `InvalidatePattern`；`internal/services/system/adapter.go:12-17,41-43` `NewCacheProvider` / `DeleteByPattern`；`internal/services/data_cache_service.go:104-147` `GetOrSet` / `DeleteByPattern`（`Keys→MDelete` 真删） |

**已确认事实即权威输入，禁止重复开 subagent 调研 LOCKED ⑤ 根因。** 下方「已确认事实」与「Lead 裁决表」是执行契约，executor 只按它落地、不自作主张改方案。

---

## 2. 目标（Objective）

把 LOCKED ⑤「`invalidateCache` no-op」从**根源**修掉，而非打补丁：

1. 消除「定义缓存键却从不读写 + 失效函数为空」的**悬空契约**（CACHE-01）。
2. 补上**裁定遗漏的真实陈旧点**：`reconciliation_service.go` 的 `GetByWorkstation` 5min TTL 缓存（`computeByWorkstation` 6 步聚合）里 `HealthScore.ExceptionHit` 与 `assets[].AppliedActions` 会随例外规则变更陈旧；实现「例外规则写成功 → 该缓存中的 exception 字段立即一致」，**用户裁定不允许 5min 陈旧**（CACHE-02 必做）。
3. 失效动作**可观测、失败可传播**（返回 error 或 `applogger.Warnf` + metric），禁止空函数体；`Cache.Type=memory`（`*MemoryCache`）与 `redis`（`*MultiLevelCache`）两种配置行为一致（CACHE-03）。
4. 作废错误处方「接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」，注释/文档不留该触发条件（ROADMAP.md:62,176；REQUIREMENTS.md:58,94）。

**保留** `GetByWorkstation` 5min 缓存及其性能价值（`:816` 注释「6 步聚合 CPU/IO 密集，缓存防穿透」），不动其 TTL、不动 `computeByWorkstation` 的 6 步查询口径。

---

## 3. 已确认事实（Lead 交叉核对，禁止重新论证）

### 3.1 死键（CACHE-01 删除对象）
- `internal/services/asset/cache_keys.go:37` 定义 `CacheKeyReconciliationExceptionRuleList = "reconciliation:exceptionRule:list"`；`:70-73` helper `GetReconciliationExceptionRuleListKey()`。
- 全库命中仅 6 处：定义、`cache_keys.go:23` 注释「R3: F4 启用」、helper、`reconciliation_exception.go:337` 注释「失效缓存(CacheKeyReconciliationExceptionRuleList)」、`:579` 裁定注释、`asset_gapfill_test.go:117` 仅断言常量字符串。
- **没有任何 `Set / Get / GetOrSet` 使用该键** → 「写后缓存未失效」这个现象在代码里不存在。
- 同族 `CacheKeyReconciliationExceptionRuleByID`（:40）+ `GetReconciliationExceptionRuleByIDKey()`（:76-78）同样**无读写方**（`asset_gapfill_test.go:118` 仅字符串断言），属同一「R3: F4 启用」悬空契约，**一并删除**（见裁决表 A2）。
- 读路径全都**直查 DB**：`List :257-305`、`GetByID :312-324`、`MatchTest :489-545`、`MatchException :558-572`、`DetectLayer3`（`reconciliation_detection.go:219-230` → `matcher.go:80-96` `preloadActiveRules`）；cron `reconciliation_crons.go:33 = @every 6m`。本就不缓存例外规则。

### 3.2 空函数体（CACHE-01 删除对象）
- `reconciliation_exception.go:574-580` `invalidateCache()` 函数体只有一行注释、零行代码，签名无 `ctx`、无 `error` 返回。
- `:238-241` 服务结构体只有 `db *gorm.DB`；`:243-246` 构造函数 `NewReconciliationExceptionService(db)` 只接 db。
- 三处装配全只传 db：`internal/api/router.go:894`、`internal/api/v1/asset/reconciliation_exception_router.go:34`、`internal/api/v1/asset/reconciliation_router.go:35`。
- `invalidateCache` 旧标识符全库共 6 处命中：**4 个生产写路径顺序正确地调用空函数**（`Create :377-382`、`Update :445-450`、`Delete :465-475`、`ImportFromExcel :602-623`，调点在 :381 / :449 / :474 / :622）**+ 1 处既有测试调用**（`asset_gapfill_test.go:771` 的 `impl.invalidateCache()`；`:768` 另有含 `invalidateCache` 字样的注释 `// invalidateCache no-op(覆盖方法体)`）**+ 1 处函数定义**（`:578`，整段 `:574-580` 含定义段注释）。⇒ 只删 `:574-580` 函数定义不处置 `:771`，该包 `*_test.go` 会 undefined、编译失败。

### 3.3 「无 Redis」是伪命题（错误处方作废）
- `core.go:883` else 分支返回功能完整的 `*MemoryCache`（非 nil）；`pkg/cache/memory.go:103-113` `MemoryCache.Delete` 真删；`pkg/cache/redis.go:697-700` `MultiLevelCache.Delete` 先删 L1 再删 L2；`redis.go:740-743` `MDelete` 删 L1+L2；`memory.go:287-300` / `redis.go:796-800` `Keys(pattern)` 支持 glob。
- `internal/services/base/cache_provider.go:145-146` `NoOpCacheProvider.Delete/DeleteByPattern` 返回 nil 是**自洽**的（其 `GetOrSet :62-72` 直接执行 query、从不写缓存）。
- ⇒ **「接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」是错误处方，应作废**：删的是一个从没人写过的键，且无 Redis 时同样不生效。

### 3.4 真实陈旧点（CACHE-02 必修，用户裁定遗漏点）
- `reconciliation_service.go:193-230` 的 `reconciliationServiceImpl` **有** `cache base.CacheProvider`（:200）+ `matcher ReconciliationExceptionService`（:208），经 `router.go:895` 注入 `system.NewCacheProvider(core.DataCacheService)`。
- `:775-776` `reconciliationHealthCacheTTL = 5 * time.Minute`；`:795-811` `GetByWorkstation` → `:821-842` `getByWorkstationWithCache` 用 `base.GetOrSetJSON[*ByWorkstationResponse]` 缓存 `computeByWorkstation`（键 = `GetReconciliationHealthByWorkstationKey(wsID)` = `reconciliation:health:workstation:%s`）。
- `:202-208` 注释显示 `matcher.MatchException` 参与 `computeByWorkstation` 的 `exceptionHit / AppliedActions` 计算（`:959-967` COUNT + `:993-1061` per-asset `AppliedActions`；`HealthScore.ExceptionHit :746`、`AssetHealthItem.AppliedActions :763`）。
- ⇒ 例外规则变更后，该 5min 缓存里的 `exceptionHit` / `AppliedActions` 会陈旧。

### 3.5 现成正确失效范式（可直接复用）
- `cache_keys.go:112-134` `InvalidateWorkstationHealth(ctx, c cache.Cache, wsID) error`：接 cache 参数、nil-safe（:130）、真调 `c.Delete`（:133）、返回 error（:126）。
- 真实调用点：`reconciliation_handler.go:218`（`h.core.Cache`）、`fix_suggestion_handler.go:399`、`reconciliation_tasks.go:314` —— **按 wsID 反查、只在工单/异常 resolve 时触发**，没有「例外规则变更 → 广播失效所有工位健康度缓存」联动。
- `base.InvalidatePattern(ctx, p CacheProvider, patterns []string, module string)`（`cache_functions.go:66-76`）：循环 `p.DeleteByPattern`、失败 `logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", ...)`、nil-safe。
- `core.go:372` `c.DataCacheService = services.NewDataCacheService(c.Cache)`；`router.go:895` 的 `system.NewCacheProvider(core.DataCacheService)` 与 `reconciliation_handler.go:218` 的 `h.core.Cache` **是同一后端**（`DataCacheService.cache` 即 `c.Cache`）。⇒ 用 `CacheProvider.DeleteByPattern` 广播失效 `reconciliation:health:workstation:*` 会真删到 `getByWorkstationWithCache` 写入的键，memory/redis 两种 `Cache` 实现都走 `Keys→MDelete` 真删（3.3 节）。

### 3.6 方案对比（推荐③ 已由 Lead 定案，executor 不得改选）
| 方案 | 做法 | 判定 |
|------|------|------|
| **③ 最小根治（本 plan 采用）** | 删死键 + 删空函数 + 给 `GetByWorkstation` 补例外变更联动失效（或剥离 exception 字段） | ✅ 保留 5min 缓存性能价值、命中真实陈旧点、零 migration |
| ② 规则版本号 + 读时校验 | 真根治且环境无关，但需 schema/migration | ❌ 本 phase 用③，**禁新增 migration** |
| ① 改全局 `CacheProvider` 接口禁 no-op | 20+ 消费文件/大量 mock 连锁改，未命中真实陈旧点 | ❌ 仅缓解，**禁改全局接口**（ROADMAP.md:62、REQUIREMENTS.md:59） |

---

## 4. Lead 裁决表（执行契约）

| 编号 | 裁决 | 落点 | 依据 |
|------|------|------|------|
| A1 | **立即一致必做**：CACHE-02 非可选，不允许 5min 陈旧 | Task 2 + must_haves #2 | 用户 2026-11-09 DECISION-例外陈旧窗口；REQUIREMENTS.md:15,56；ROADMAP.md:57,171 |
| A2 | **死键/空函数删除**：`invalidateCache` 旧标识符全库共 **6 处命中 = 4 个生产写路径 + 1 处既有测试调用（`asset_gapfill_test.go:771`，`:768` 另有含 `invalidateCache` 字样的注释）+ 1 处函数定义 `:578`**，须全部处置。删 `CacheKeyReconciliationExceptionRuleList` + `GetReconciliationExceptionRuleListKey()` + 空 `invalidateCache()`（含 `:574-580` 函数定义段）+ 4 个生产写路径调点；**连带**删 `CacheKeyReconciliationExceptionRuleByID` + `GetReconciliationExceptionRuleByIDKey()`（同为无读写方悬空契约）；清 `cache_keys.go:23` / `reconciliation_exception.go:337,579` 悬空注释；删 `asset_gapfill_test.go:117,118` 字符串断言；**追加**清理 `asset_gapfill_test.go:768-771`（删 `:768` 注释与 `:771` 调用，或改调 `impl.invalidateExceptionRuleCache(ctx)` 并断言 `err == nil`，定案见 T1-4），否则删旧函数后该测试 undefined、包编译失败 | Task 1 | CACHE-01；ROADMAP.md:59；根因报告 §⑤:122-124；plan-checker R1（Lead 独立核实 6 处命中） |
| A3 | **作废「接 Redis 补 Delete」**：任何注释/文档/SUMMARY 都不得保留该处方；全仓不得出现 `core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)` 或等价表述；死键清零 | Task 1 + must_haves #4 | ROADMAP.md:62,176；REQUIREMENTS.md:58,94 |
| A4 | **方案③ 定案 = 方案 (a) 广播失效**（Task 2 二选一在此定案，executor 不得改选 (b)）：例外规则服务通过 `base.CacheProvider` 注入（沿用 `router.go:895` 同款 `system.NewCacheProvider(core.DataCacheService)` 装配语义），写成功后失效 `reconciliation:health:workstation:*`（直接调 `s.cache.DeleteByPattern` 取 error，语义等价 `base.InvalidatePattern` 但显式返回错误）；**不动 `internal/api/router.go`** | Task 2 | 见 §4.1 定案理由 |
| A5 | **失效可观测 + 失败可传播**：`invalidateExceptionRuleCache(ctx) error` 返回 `DeleteByPattern` 的 error；4 个写路径 `if err := ...; err != nil { applogger.Warnf(...) }`（失败不阻断主写路径，但必有 Warnf 日志/metric 钩子）；禁止空函数体、禁止无返回值的纯注释函数 | Task 3 + must_haves #3 | CACHE-03；ROADMAP.md:61 |
| A6 | **memory/redis 行为一致**：两种 `CacheProvider` 实现（`cacheProviderAdapter` 包 `DataCacheService`；`DataCacheService` 包 `*MemoryCache` 或 `*MultiLevelCache`）都走同一 `DeleteByPattern` 路径；测试用 mock `CacheProvider` 断言 `DeleteByPattern` 调用次数 ≥1 | Task 3 + Task 4 | 3.3 / 3.5 节核验 |
| A7 | **零 git 写操作**；不启动长驻进程；不写生产数据；不改其它 phase 文件面 | 全程 | frontmatter `commit:false`；ROADMAP.md:161；CLAUDE.md Git Workflow |
| A8 | **尽量不动 `internal/api/router.go`**：ROADMAP.md:145 已硬登记「149 先改、151 后改或拆 commit」；147 属 W1 且与 149/151 同文件冲突面。本 plan 定案 (a) 通过 `core.DataCacheService` 已有装配实现，**若 Task 2 确需动 router.go，必须满足 §4.2 登记模板**并在 `147-SUMMARY.md` 移交 Lead 集成复核 | Task 2 / §4.2 | 用户指令 + ROADMAP.md:145 |

### 4.1 方案 (a) 定案理由（为什么选 (a) 不选 (b)）

- **(b) 剥离 exception 字段**（缓存只保留非 exception 字段、读时实时调 `MatchException`）虽然也能立即一致，但要改 `computeByWorkstation` 的 6 步聚合结果装配、`ByWorkstationResponse` 结构、缓存值形状（等价于一次缓存 schema 变更），且每次读都要额外跑 `preloadActiveRules + MatchException` —— **把 6 步聚合的性能价值削掉一部分**，与「保留 5min 缓存性能价值」目标相冲；`All 档/响应形状` 不变式也更难守。
- **(a) 广播失效**：只在**写路径**（低频：例外规则 CRUD/导入）触发一次 `DeleteByPattern("reconciliation:health:workstation:*")`，读路径完全不变、6 步聚合与 5min TTL 原样保留；下一次 `GetByWorkstation` 未命中 → 回源 `computeByWorkstation` 重算含新规则的 `exceptionHit/AppliedActions` → 立即一致。
- **性能影响评估**：成本 = 每次规则写 1 次 `Keys(pattern)` 扫描（memory 遍历 map / redis `SCAN` 合并 L1+L2）+ 1 次 `MDelete`；工位健康度缓存条目数 = 工位数，量级小；6 步聚合只在失效后的首个读回源一次，之后继续 5min 命中。**读路径零额外开销**，符合 `:816` 注释的性能价值。
- **后端一致性已核验**：`core.DataCacheService` 与 `reconciliationSvc` 的 `CacheProvider` 是同一 `c.Cache`（3.5 节）；`DeleteByPattern` 在 memory/redis 两实现都真删（3.3 节）。→ 方案 (a) 真删对键，非「删一个平行宇宙的键」。

### 4.2 若 Task 2 确需动 `internal/api/router.go` 的登记模板（默认不触发）

> 本 plan **默认不动** `internal/api/router.go`。executor 只有在实现 (a) 时发现「必须给 `NewReconciliationExceptionService` 增加 cache 参数且无法在 router 文件外完成装配」才触发本节；触发时 **必须**：
> 1. 在 PLAN 的本节以文字显式登记（复制到 `147-SUMMARY.md`）：
>    「**147 与 149/151 同文件 `internal/api/router.go`，147 先改且只加一行装配（给 `NewReconciliationExceptionService` 传 `systemServices.NewCacheProvider(core.DataCacheService)`），149/151 后改基于最新文件**」；
> 2. 该行**只改 `router.go:894` 附近例外规则 service 的构造**，不得触碰 `:895` reconciliationSvc 装配、不得触碰 149 的 rooms/photos 组（`:1174-1177`）、不得触碰 151 的 ScopedDB 挂载；
> 3. `147-SUMMARY.md` 的「移交 Lead 集成复核」段写明：W1 的 147 先落 `router.go` 一行 → 149（W2）基于含该行的最新文件补挂 rooms/photos 中间件 → 151（W3）基于 149 之后的最新文件挂 ScopedDB；
> 4. 把 frontmatter `router_go_touched: false` 改为 `true`，并在其后追加一行 `# 147 先改：router.go:894 仅加一行 cache 参数；149 后改 rooms/photos 中间件、151 后改 ScopedDB（ROADMAP.md:145）`。
>
> **推荐规避（优先）**：用「可选 cache 注入 + 延迟绑定」把装配留在 `internal/services/asset` 内，见 Task 2 步骤 T2-2，从而完全不触发本节。

---

## 5. 任务（XML）

```xml
<tasks>

  <!-- ============================================================= -->
  <task id="1" name="CACHE-01 删除死键与空函数体，消除悬空契约" req="CACHE-01">
    <files>
      internal/services/asset/cache_keys.go
      internal/services/asset/reconciliation_exception.go
      internal/services/asset/asset_gapfill_test.go
    </files>
    <read_first>
      cache_keys.go:20-51（R1/R2/R3/R4 演进注释块）、:70-78（两个 helper）、:112-134（InvalidateWorkstationHealth 保留不动）
      reconciliation_exception.go:238-246（结构体/构造函数）、:330-341（Create 注释）、:381/:449/:474/:622（4 个调点）、:574-580（空函数）
      asset_gapfill_test.go:113-125（TestCacheKeyBuildersAndStripPrefix）
    </read_first>
    <action>
      T1-1 删除 `cache_keys.go` 中两个死键常量与两个 helper：
        - `CacheKeyReconciliationExceptionRuleList`（:36-37，含其 :36 注释）
        - `CacheKeyReconciliationExceptionRuleByID`（:39-40，含其 :39 注释）—— 同为无读写方悬空契约，裁决表 A2
        - `GetReconciliationExceptionRuleListKey()`（:70-73）
        - `GetReconciliationExceptionRuleByIDKey()`（:75-78）
        保留其余常量/helper 不动（`dashboard / exception:list / exception:byID / view:lastRefresh / health:workstation / health:asset / StripCachePrefix / InvalidateWorkstationHealth`）。
      T1-2 清理悬空注释：
        - `cache_keys.go:23` 注释「R3: F4 启用 reconciliation:exceptionRule:list / reconciliation:exceptionRule:byID」→ 改为「R3: F4 例外规则缓存键已下线（Phase 147 CACHE-01，读路径直查 DB，无缓存契约）」，**不得留「待启用」暗示**；`:20-24` 演进块其余行保持。
        - `reconciliation_exception.go:337` Create 行为注释第 4 步「失效缓存(CacheKeyReconciliationExceptionRuleList)」→ 删除该步或改为「4. 失效工位健康度缓存（见 invalidateExceptionRuleCache，Phase 147 CACHE-02）」。
        - `reconciliation_exception.go:574-580` 整段 `invalidateCache` 注释与函数体删除；**禁止**保留含「若未来接入 Redis 再补 core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)」的裁定注释（裁决表 A3）。
        - `reconciliation_exception.go:391` Update 注释「4. 失效缓存」→ 改为「4. 失效工位健康度缓存（Phase 147）」或同义；`:455-457` Delete 注释不改（语义仍成立）。
      T1-3 删 4 个生产写路径对空函数的调用：删除 `reconciliation_exception.go` 的 `s.invalidateCache()` 四行（:381 / :449 / :474 / :622）。
        口径：`invalidateCache` 旧标识符全库共 6 处命中 = 4 个生产写路径 + 1 处既有测试调用（`asset_gapfill_test.go:771`，`:768` 另有含 `invalidateCache` 字样的注释）+ 1 处函数定义 `:578`；本步只处置 4 个生产写路径，测试调用由 T1-4 处置、函数定义由 T1-2 整段删除。
        ⚠️ 注意：Task 2 会在**同四个位置**插入 `s.invalidateExceptionRuleCache(ctx)`，**不要先删空再留空**——executor 应把 T1-3 与 T2-4 合并为一次编辑（见 Task 2「与 Task 1 的合并点」），避免中间态 4 个写路径既不失效也不报错。
      T1-4 `asset_gapfill_test.go:117,118` 删除两行 `assert.Equal(t, "reconciliation:exceptionRule:list", ...)` 与 `...byID:r-1...`；`TestCacheKeyBuildersAndStripPrefix` 其余断言不动（含 :120 `health:workstation:ws-1` 与 :127 `TestInvalidateWorkstationHealthCacheKey`）。
        **并追加清理 `asset_gapfill_test.go:768-771`**（R1 必修：这是第 5 处 `invalidateCache` 命中，T1-2 删除 `:574-580` 函数定义后 `:771` 会 undefined，导致 `internal/services/asset/` 包 `*_test.go` 编译失败）：
          - 二选一 —— ① 删除 `:768` 注释 `// invalidateCache no-op(覆盖方法体)` 与 `:771` 的 `impl.invalidateCache()` 调用；② 改为调用新的 `impl.invalidateExceptionRuleCache(ctx)` 并断言 `err == nil`。
          - **PLAN 定案：采用 ②（推荐）**。理由：该测试函数 `TestExceptionService_ImportFromExcel` 作用域内已有 `ctx := context.Background()`（`:734`）、文件已 import `context`（`:5`）与 `require`（`:15`），改调 `require.NoError(t, impl.invalidateExceptionRuleCache(ctx))` 可直接编译、零新增 import；且 `svc := NewReconciliationExceptionService(db)` 为零参构造（`cache=nil`），新失效函数 nil-safe 返回 nil，既能顺势覆盖「新失效路径在 nil-cache 下不 panic」语义、又保留原「覆盖方法体」的测试意图，比单纯删两行更有覆盖价值。
          - 硬约束：无论选 ① 还是 ②，删旧函数后该测试必须仍**编译通过**且覆盖新失效语义；`invalidateCache` 旧标识符（含 `:768` 注释字样）在该文件必须清零。
      T1-5 静态自检（不提交）：`grep -rn "CacheKeyReconciliationExceptionRuleList\|GetReconciliationExceptionRuleListKey\|CacheKeyReconciliationExceptionRuleByID\|GetReconciliationExceptionRuleByIDKey\|invalidateCache" internal/ | wc -l` 必须为 0。
        判据：`invalidateCache` **旧标识符全库零命中（含测试与注释）** —— grep 范围**必须覆盖 `*_test.go`**（`internal/` 全目录递归，含 `internal/services/asset/*_test.go`），不得只扫 `*.go` 非测试文件或只扫 `internal/services/asset/`。
        **必须清零清单**：`reconciliation_exception.go:381/:449/:474/:622`（4 个生产写路径）、`:578` 函数定义及 `:574-580` 定义段注释、`:337` 悬空注释、`asset_gapfill_test.go:768` 注释与 `:771` 测试调用、`asset_gapfill_test.go:117,118` 死键断言；除 `147-SUMMARY.md` 作为历史说明可提及外，注释里也应为 0。
    </action>
    <verify>
      go build ./...   # Task 1 单跑时允许因 Task 2 未落而失败；与 Task 2 合并后必须绿
      grep -rn "CacheKeyReconciliationExceptionRuleList\|GetReconciliationExceptionRuleListKey\|CacheKeyReconciliationExceptionRuleByID\|GetReconciliationExceptionRuleByIDKey\|invalidateCache" internal/ || echo "CLEAN"   # 覆盖 *_test.go 与注释；必须清零清单见 T1-5
    </verify>
    <done>
      `invalidateCache` 旧标识符全库共 6 处命中 = 4 个生产写路径 + 1 处既有测试调用（`asset_gapfill_test.go:771`，`:768` 另有含 `invalidateCache` 字样的注释）+ 1 处函数定义 `:578`，全部处置完毕（含测试与注释清零）；4 个死符号全仓零引用；4 个生产写路径不再调用空函数；悬空注释清零；`asset_gapfill_test.go:117,118` 两行死键断言删除且 `:768-771` 已按 T1-4 定案②改调 `invalidateExceptionRuleCache(ctx)` 并断言 `err == nil`、删旧函数后测试仍编译通过；`InvalidateWorkstationHealth` 与其余 key 原样保留。
    </done>
    <forbidden>
      禁止保留「定义缓存键却从不读写」的悬空契约；禁止为死键补 `core.Cache.Delete`；禁止只删常量不删 helper/注释/测试断言造成半悬空。
    </forbidden>
  </task>

  <!-- ============================================================= -->
  <task id="2" name="CACHE-02 例外规则写成功后 GetByWorkstation exception 字段立即一致（用户裁定必做）" req="CACHE-02">
    <files>
      internal/services/asset/reconciliation_exception.go
      internal/services/asset/reconciliation_service.go
      （默认不动）internal/api/router.go
    </files>
    <read_first>
      reconciliation_service.go:193-230（结构体/cache/matcher/构造函数）、:775-842（TTL + GetByWorkstation + getByWorkstationWithCache）、:959-967（exceptionHit COUNT）、:993-1061（AppliedActions）
      cache_keys.go:46-50,85-88（CacheKeyReconciliationHealthByWorkstation 与 GetReconciliationHealthByWorkstationKey，保留不动）
      base/cache_functions.go:62-76（InvalidatePattern 现成范式，本 plan 语义等价但不直接调用，见 T2-2 定案）
      system/adapter.go:12-17,41-43（NewCacheProvider / DeleteByPattern）
      router.go:890-895（现有装配，只读核对，不改）
    </read_first>
    <decision>【定案 = 方案 (a) 广播失效，不采用 (b) 剥离 exception 字段】理由见 §4.1。
      (a)：例外规则服务注入 `base.CacheProvider`，写成功（DB 事务成功后）直接调
           `s.cache.DeleteByPattern(ctx, pattern)` 失效 `reconciliation:health:workstation:*`
           （pattern 由 `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")` 得出；
            **语义等价 `base.InvalidatePattern`**：同 pattern、同 nil-safe、同失败 Warnf，
            但本 plan 定案「直接调以拿到 `error` 返回值」满足 CACHE-03 的「失败可传播」，见 T2-2）；
           下一次 `GetByWorkstation` 未命中 → 回源 `computeByWorkstation` 重算含新规则的
           `HealthScore.ExceptionHit` 与 `assets[].AppliedActions` → 写后立即一致。
      广播 pattern 用常量表达式：因 `CacheKeyReconciliationHealthByWorkstation = "reconciliation:health:workstation:%s"`，
          用 `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")` 得 `reconciliation:health:workstation:*`；
          **定案：不新增常量**（避免制造第二真相源，见 T2-3）；T4-5 断言该 pattern 前缀与 `GetReconciliationHealthByWorkstationKey("ws-1")` 一致。
      不采用 (b)：剥离 `exceptionHit/AppliedActions` 改读时实时 `MatchException` 会改缓存值形状、每次读多跑
           `preloadActiveRules + MatchException`，削掉 6 步聚合 5min 缓存的性能价值，与 ROADMAP.md:57「保留性能价值」相冲。
    </decision>
    <action>
      T2-1 【注入 cache；只在写路径装配点加一行，不动 internal/api/router.go】
        在 `reconciliation_exception.go` 的 `reconciliationExceptionServiceImpl`（:239-241）增加字段 `cache base.CacheProvider`。
        构造函数改为**向后兼容的可变参数**（零参调用方全部零改动、编译通过）：
          func NewReconciliationExceptionService(db *gorm.DB, cacheProviders ...base.CacheProvider) ReconciliationExceptionService {
              svc := new(reconciliationExceptionServiceImpl)
              svc.db = db
              if len(cacheProviders) > 0 {
                  svc.cache = cacheProviders[0]
              }
              return svc
          }
        零参 → `cache=nil` → 失效动作 nil-safe 跳过（单测友好）。
        **唯一生产装配改动**：`internal/api/v1/asset/reconciliation_exception_router.go:34` 是 4 个写路径
        （`/exception-rule/create`、`/:id/update`、`/:id/delete`、`/exception-rule/import`，见该文件 :45-53, :82；
          `/exception-rule/test` 是只读命中测试、不写库）**唯一**的 service 装配点，
        把 `asset.NewReconciliationExceptionService(core.DB.GetDB())` 改为
        `asset.NewReconciliationExceptionService(core.DB.GetDB(), system.NewCacheProvider(core.DataCacheService))`，
        并在该文件 import 块追加一行 `"github.com/xingran-next/xingran-go-backend/internal/services/system"`
        （`core` 已在该文件 import 中，`core.DataCacheService` 可直接取；该文件**不在** ROADMAP.md:145 的 router.go 硬登记表里，改它不触发 §4.2）。
        **保持零参不动的两处**（均只作 `matcher` 注入、不承载任何写路径）：
          - `internal/api/v1/asset/reconciliation_router.go:35` 的 `exceptionSvc`（注入给 `ReconciliationService` 作 per-asset 匹配）
          - `internal/api/router.go:894` 的 `exceptionSvcForWs`（:892 注释：注入 `WorkstationHandler.GetByID` 的 per-asset 匹配）
        ⇒ **本 plan 默认完全不动 `internal/api/router.go`**，也不动 `reconciliation_router.go`。
        ⚠️ 仅当 executor 复核后认为 `router.go:894` 也必须传 cache（例如未来该实例也走写路径）才触发 §4.2 登记模板并移交 Lead；默认不触发。
        导入：`reconciliation_exception.go` 需追加 `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"`
        （`base` 已在 :17、`fmt` 已在 :6、`context` 已在 :4，均无需重复导入）。

      T2-2 【可观测 + 失败可传播的失效函数（CACHE-03 的实体）】
        在 `reconciliation_exception.go` 新增（替代被删的 `invalidateCache`）：
          // invalidateExceptionRuleCache 广播失效工位健康度缓存（Phase 147 CACHE-02）。
          // 例外规则变更后 GetByWorkstation 5min 缓存内的 exceptionHit/AppliedActions 会陈旧，
          // 故写成功后按 pattern 失效全部 reconciliation:health:workstation:*；
          // 返回 error 供调用方 Warnf 观测；cache 为 nil（单测/未配置）时 nil-safe 跳过。
          func (s *reconciliationExceptionServiceImpl) invalidateExceptionRuleCache(ctx context.Context) error {
              if s.cache == nil {
                  return nil
              }
              pattern := fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")
              if err := s.cache.DeleteByPattern(ctx, pattern); err != nil {
                  return fmt.Errorf("失效工位健康度缓存失败: pattern=%s: %w", pattern, err)
              }
              return nil
          }
        【定案：此处**不直接调用** `base.InvalidatePattern`】`InvalidatePattern`（`cache_functions.go:66-76`）是
          void 返回 + 内部 `logger.Warnf`，拿不到 error 返回值；CACHE-03 要「返回 error 或 Warnf + metric」，
          且 Task 4 要断言「失效失败不阻断主写路径但可观测」，故本 plan 定案**直接调 `s.cache.DeleteByPattern` 取 error**，
          由 4 个写路径统一 `applogger.Warnf` 落地可观测（T2-4）。这与 `base.InvalidatePattern` 语义等价
          （同 pattern、同 nil-safe、同失败 Warnf），但把「失败可传播」做成显式返回值；
          `base.InvalidatePattern` / `base.Invalidate` 作为现成范式**保留在代码库、本 phase 不改、T4-4 正向断言其仍存在**。
          ⚠️ executor 不得退化成「void 且无任何日志」的失效动作，也不得改成「调 InvalidatePattern 后再也拿不到 error」。

      T2-3 【广播 pattern 与读缓存键同源（定案：不新增常量，消除二选一）】
        `invalidateExceptionRuleCache` 内直接用 `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")`
        得出 `reconciliation:health:workstation:*`，**不新增** `CacheKeyReconciliationHealthByWorkstationAll` 常量
        （逻辑键模板 `CacheKeyReconciliationHealthByWorkstation` 已是单一真实源，新增字面量常量反而制造第二真相源）。
        `reconciliation_cache_contract_test.go` 必须断言：
          `strings.HasPrefix(GetReconciliationHealthByWorkstationKey("ws-1"), strings.TrimSuffix(fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*"), "*"))`
        防止「删了一个平行宇宙的键」（对应 Task 4 T4-5）。

      T2-4 【4 个写路径接联动失效 + 可观测（与 Task 1 的合并点）】
        在 `Create :377-382` / `Update :445-450` / `Delete :465-475` / `ImportFromExcel :602-623` 的
        **DB 写成功之后、`return` 之前**，把原 `s.invalidateCache()` 替换为：
          if err := s.invalidateExceptionRuleCache(ctx); err != nil {
              // 失效失败不阻断主写路径（DB 已提交），但必须可观测（CACHE-03）
              applogger.Warnf("[reconciliation:exceptionRule] %s 后缓存失效失败（不阻断主流程）: %v", "Create", err)
          }
        （`Create/Update/Delete/ImportFromExcel` 四处的 `"Create"` 分别改为对应动作名，便于日志定位；
          `ImportFromExcel` 的调点放在 `postProcessImportedRules` 之后、`:622` 原位置，保证基础字段与后处理都已落库。）
        **顺序不变**：DB 写 → 失效 → return；handler 侧 `operlog.Record` → `response.Success` 顺序不动（CLAUDE.md 约定）。
        ⚠️ 合并点：Task 1 的 T1-3（删 4 行 `s.invalidateCache()`）与此处 T2-4（插 4 段失效）是**同一 4 个位置的连续编辑**，
           executor 必须一次完成「删空函数 → 插新失效」，禁止中间态出现「写路径既不失效也不报错」。
        ⚠️ 禁止把失效动作放在 `db.Create` 之前（DB 失败却已删缓存 → 无谓回源）；禁止用 `_ =` 静默吞掉 error 且无 Warnf。

      T2-5 【读路径零改动的性能/一致性说明，写进 SUMMARY】
        `reconciliation_service.go` 的 `GetByWorkstation / getByWorkstationWithCache / computeByWorkstation /
         reconciliationHealthCacheTTL` **全部不动**（保留 5min 缓存性能价值）；
        仅在 `reconciliation_service.go` 的 `getByWorkstationWithCache` 注释（:813-820）追加一句：
          「Phase 147 CACHE-02：例外规则写路径经 invalidateExceptionRuleCache 广播失效本键前缀，
            保证规则变更后本缓存内 exceptionHit/AppliedActions 立即一致；本函数读路径不变。」
        这是本 plan 对 `reconciliation_service.go` 的**唯一改动**（注释级），避免碰 6 步聚合逻辑。
    </action>
    <acceptance>
      写后立即一致验收：同一进程内先 `GetByWorkstation(ctx, wsID, ...)` 预热缓存（断言 `HealthScore.ExceptionHit` / `assets[].AppliedActions` 为旧值），
      再 `Create/Update/Delete/ImportFromExcel` 写入新规则，再立刻 `GetByWorkstation(ctx, wsID, ...)`，
      断言返回的是**新规则**计算出的 `ExceptionHit` / `AppliedActions`（不得等 5min、不得等 cron `@every 6m`）。
      Task 4 的 `TestExceptionRuleWrite_ReadYourWrites` 四个子用例（Create/Update/Delete/ImportFromExcel）逐写路径覆盖；
      Task 5 T5-2 的反向验证（临时注释 T2-4 的失效调用 → 该测试**必须红** → 还原后必须绿）证明联动失效是真因。
    </acceptance>
    <performance>
      成本仅落在低频写路径：1 次 `Keys(pattern)`（memory 遍历 map / redis 合并 L1+L2 `Keys`）+ 1 次 `MDelete`；
      工位健康度缓存条目数 = 工位数，量级小；失效后首个读回源一次 6 步聚合，之后继续 5min 命中。
      读路径零额外开销、TTL 不变、`computeByWorkstation` 6 步查询口径不变 → `:816` 注释的性能价值完整保留。
    </performance>
    <done>
      方案 (a) 落地；4 个写路径写成功后广播失效 `reconciliation:health:workstation:*`；
      `GetByWorkstation` 读路径/TTL/6 步聚合零改动；`NewReconciliationExceptionService` 向后兼容（可变参数，3 处装配默认零改动）；
      默认**不动 `internal/api/router.go`**；`reconciliation_service.go` 仅追加一行注释。
    </done>
    <forbidden>
      禁 5min 陈旧；禁采用方案 (b) 剥离 exception 字段；禁改 `computeByWorkstation`/TTL/缓存值形状；
      禁新增 migration（方案②）；禁改全局 `CacheProvider` 接口（方案①）；禁为死键补 `core.Cache.Delete`；
      禁在 `internal/api/router.go` 无 §4.2 登记的情况下改动。
    </forbidden>
  </task>

  <!-- ============================================================= -->
  <task id="3" name="CACHE-03 失效动作可观测、失败可传播、memory/redis 行为一致" req="CACHE-03">
    <files>
      internal/services/asset/reconciliation_exception.go
      internal/services/asset/reconciliation_cache_contract_test.go
    </files>
    <read_first>
      base/cache_provider.go:22-57（CacheProvider 9 方法，DeleteByPattern 在 :38）、:59-72（NoOp.GetOrSet 不写缓存）、:145-146（NoOp.Delete/DeleteByPattern 返回 nil 自洽）
      base/cache_functions.go:50-76（Invalidate / InvalidatePattern 的 Warnf 格式）
      data_cache_service.go:104-147（GetOrSet / Delete / DeleteByPattern = Keys→MDelete）
      pkg/cache/memory.go:103-113,173,287-300；pkg/cache/redis.go:697-700,740-743,796-800
    </read_first>
    <action>
      T3-1 失效函数契约（已在 T2-2 落地，此处复核）：
        - 签名 `func (s *reconciliationExceptionServiceImpl) invalidateExceptionRuleCache(ctx context.Context) error`，**有 ctx、有 error 返回**；
        - 函数体真调 `s.cache.DeleteByPattern(ctx, pattern)`（禁止空函数体、禁止只写注释）；
        - `s.cache == nil` → 返回 nil（nil-safe，单测/未配置场景）；
        - `DeleteByPattern` 失败 → `fmt.Errorf("失效工位健康度缓存失败: pattern=%s: %w", ...)` 包装传播。
      T3-2 4 个写路径统一 `applogger.Warnf(...)`（T2-4 已落地）：失败**不阻断主写路径**（DB 已提交，不返回 error 给 handler），
        但必有 `[reconciliation:exceptionRule] <Action> 后缓存失效失败（不阻断主流程）: <err>` 日志；
        如需 metric 钩子，在同一 `if err != nil` 分支内追加（零新依赖，沿用 `applogger` 即可，**不强制**引入新 metrics 包）。
      T3-3 memory/redis 一致性：两种 `CacheProvider` 实现都经 `cacheProviderAdapter.DeleteByPattern → DataCacheService.DeleteByPattern →
        (Keys(ctx, pattern) → MDelete(ctx, keys...))`；`*MemoryCache` 与 `*MultiLevelCache` 都实现了 `Keys/MDelete` 真删（3.3 节）；
        `NoOpCacheProvider`（`DataCacheService==nil` 时）`DeleteByPattern` 返回 nil 自洽（其 `GetOrSet` 从不写缓存，无陈旧问题）。
        **本 plan 不改 `base.CacheProvider` 接口、不改两种实现、不改 `NoOpCacheProvider`**（禁区方案①）。
      T3-4 `reconciliation_cache_contract_test.go` 中：
        - `TestInvalidate_ReturnsError_OnDeleteByPatternFailure`：mock `CacheProvider.DeleteByPattern` 返回 `errors.New("boom")`，
          直接调 `invalidateExceptionRuleCache(ctx)` 断言返回非 nil error 且 `errors.Is`/`strings.Contains` 含 pattern 与 `boom`；
        - `TestWritePath_InvalidationFailure_DoesNotBlockButObservable`：mock 同上，调 `Create`（DB 用 sqlite 内存 fixture）断言
          **Create 返回成功**、规则已落库，且 mock 记录到 `DeleteByPattern` 被调用（可观测计数 ≥1）；
          若测试基建能捕获 `applogger` 输出，追加断言日志包含「缓存失效失败」（不强依赖，避免脆测）。
        - `TestInvalidate_NilCache_IsNoOp`：零参构造 `NewReconciliationExceptionService(db)` → `invalidateExceptionRuleCache(ctx)` 返回 nil、不 panic。
        - `TestInvalidate_DeleteByPattern_CalledAtLeastOnce`：4 个写路径各断言 `DeleteByPattern` 调用次数 ≥1（见 Task 4）。
    </action>
    <verify>
      go test -count=1 -run 'TestInvalidate|TestWritePath_InvalidationFailure' ./internal/services/asset/
    </verify>
    <done>
      失效动作有 ctx/有 error 返回/真调 Delete/失败 Warnf 可观测/不阻断主写；
      空函数体清零；memory 与 redis 两种 `CacheProvider` 走同一失效路径；`CacheProvider` 接口零改动。
    </done>
    <forbidden>
      禁空函数体；禁 `_ =` 静默吞错且无日志；禁改全局 `CacheProvider` 接口 / 20+ 消费文件 / mock 连锁改（方案①）；
      禁「接 Redis 补 Delete」处方复现。
    </forbidden>
  </task>

  <!-- ============================================================= -->
  <task id="4" name="新增 reconciliation_cache_contract_test.go：写后读必一致 / 双配置 / 可观测 / 死键静态断言" req="CACHE-01, CACHE-02, CACHE-03">
    <files>
      internal/services/asset/reconciliation_cache_contract_test.go  （新增）
      internal/services/asset/asset_gapfill_test.go  （与 Task 1 的合并点：T1-4 删 `:117,118` 两行死键断言 + 清理 `asset_gapfill_test.go:768-771` 的第 5 处 `invalidateCache` 命中；本 task 不再改该文件）
    </files>
    <read_first>
      asset_gapfill_test.go:60-107（newTestDB / sqlite 内存 fixture / 建表，直接复用同款基建，勿另造）
      asset_gapfill_test.go:300-330,680-690（ExceptionHit / AppliedActions 既有断言形态）
      reconciliation_detection_test.go:29-55（DetectLayer3 命中规则 + AppliedActions 断言形态）
      internal/services/operations/cache_invalidator_test.go:20-45（mock CacheProvider 形态，可参照）
      internal/services/duty/duty_cache_impl_test.go:70-90（mock DeleteByPattern 形态）
    </read_first>
    <action>
      T4-0 复用 `asset_gapfill_test.go` 的 sqlite 内存 DB 与 `sys_reconciliation_exception` / `sys_data_reconciliation` /
        `reconciliation_normalized` / `sys_workstation` / `ops_asset` 建表 fixture（同包 `package asset`，就近 `*_test.go`，AGENTS.md）。
        定义 `contractCacheSpy` 实现 `base.CacheProvider` 全部 9 方法（照 `operations/cache_invalidator_test.go` 的 `MockCacheProvider`），
        字段：`deleteByPatternCalls []string`、`deleteByPatternErr error`、`getOrSet` 用可插桩内存 map 模拟读穿透；
        方法 `DeleteByPattern` 记录 pattern、`Delete` 记录 key、`MDelete` 记录 keys，`GetOrSet` 走真实 `computeByWorkstation` 闭包并缓存。
        ⚠️ 同包内若已有同名 mock 类型会编译冲突 → 命名统一加 `contract` 前缀（如 `contractCacheSpy`）。

      T4-1 【写后读必一致，4 个写路径各一次】`TestExceptionRuleWrite_ReadYourWrites`（表驱动 4 子用例：Create / Update / Delete / ImportFromExcel）：
        - 准备：sqlite fixture + 至少一个工位 ws-1 + 资产 + 一条 active 例外规则 R_old（含 `ip_range` 覆盖资产 IP、`exception_actions` 含 `no_alert`）；
          构造 `exceptionSvc := NewReconciliationExceptionService(db, spy)`、`reconSvc := NewReconciliationService(db, spy, exceptionSvc)`；
          `resp0, _ := reconSvc.GetByWorkstation(ctx, "ws-1", "7d")` 预热 5min 缓存，断言 `resp0.HealthScore.ExceptionHit` / `resp0.Assets[].AppliedActions` 为**旧规则**结果；
        - 写：调用对应用例的写方法（Update 改 `ip_range`/`exception_actions`；Delete 删 R_old；ImportFromExcel 用 `t.TempDir()` 自建 xlsx 或 `xlsxMagic` 字节，
          **禁读磁盘模板/禁落盘 `.xlsx` 到仓库**，AGENTS.md；若 ImportFromExcel 的 `operations.ExcelService` 依赖过重，用「等价写路径 = 直接 db.Create 一条新规则 + 调 `invalidateExceptionRuleCache`」的最小替身并在 SUMMARY 登记该替身口径，
          但 4 个写路径的 `DeleteByPattern` 调用次数断言仍必须覆盖 ImportFromExcel 本身）；
        - 读：立刻 `resp1, _ := reconSvc.GetByWorkstation(ctx, "ws-1", "7d")`，断言 `resp1` 反映**新规则**（`ExceptionHit` / `AppliedActions` 与重算一致）；
          并断言 `List / GetByID / MatchTest / MatchException / DetectLayer3` 同进程立即反映新规则（这些本就直查 DB，作为回归守护）；
        - 断言 `len(spy.deleteByPatternCalls) >= 1` 且 pattern = `reconciliation:health:workstation:*`（或 `fmt.Sprintf(...,"*")` 结果）。
        - ⚠️ 若 sqlite 下 `computeByWorkstation` 的某步（如 `reconciliation_normalized` 物化视图探测 / `COUNT FILTER`）不可用，
          用 `asset_gapfill_test.go` 已验证能跑通的 fixture 口径，不新造 SQL；不得为过测而放宽「立即一致」断言。

      T4-2 【memory/redis 双配置】`TestInvalidate_MemoryAndRedis_BehaveIdentically`：
        若测试基建能构造两种真实后端：`pkg/cache.NewMemoryCache(...)`（查 `memory.go` 构造签名）与
        `pkg/cache` 的 `*MultiLevelCache`（需真 Redis，**本机不启动长驻进程/不连生产**）→
        用 `t.Skip` 显式标注 Redis 侧需真机并在 SUMMARY 登记 Known Gap（**不伪造双跑**）；
        **主路径用 mock `CacheProvider` 断言 `DeleteByPattern` 调用次数 ≥1**（用户指令允许），
        并补一个静态断言：遍历 `CacheProvider` 接口的两种生产实现（`cacheProviderAdapter` / `NoOpCacheProvider`）都实现 `DeleteByPattern`，
        且 `DataCacheService.DeleteByPattern` 走 `Keys→MDelete`（用 `go/ast` 或反射读源码字符串，AUDIT 同款静态手法，零运行时依赖）。
        另加 `TestInvalidate_MemoryCache_ReallyDeletes`（若 `NewMemoryCache` 可直接构造）：先 `Set` 一个 `reconciliation:health:workstation:ws-1`，
        再经 `cacheProviderAdapter{DataCacheService: NewDataCacheService(mem)}.DeleteByPattern("reconciliation:health:workstation:*")`，
        断言该键 `Get` 已不存在 —— 证明 memory 配置真删、非 no-op。

      T4-3 【失效失败不阻断主写但可观测】见 Task 3 T3-4（同文件实现，避免重复定义 mock）。

      T4-4 【死键已删除的静态断言】`TestDeadSymbols_AreGone`：
        用 `go/parser` 或 `os.ReadFile` + `strings.Contains` 扫描本包（`internal/services/asset/*.go`，**必须含 `*_test.go`**，含 `cache_keys.go` /
        `reconciliation_exception.go` / `asset_gapfill_test.go` / `reconciliation_cache_contract_test.go`）：
        断言 `invalidateCache` **旧标识符全库零命中（含测试与注释）**，以及 `CacheKeyReconciliationExceptionRuleList`、
        `GetReconciliationExceptionRuleListKey`、`CacheKeyReconciliationExceptionRuleByID`、
        `GetReconciliationExceptionRuleByIDKey` 全部零命中；
        **必须清零清单**须显式列入 `asset_gapfill_test.go:768`（`// invalidateCache no-op...` 注释）与 `:771`（`impl.invalidateCache()` 测试调用）
        两处第 5 处命中，以及 `reconciliation_exception.go:578` 函数定义与 `:574-580` 定义段注释、4 个生产写路径调点 `:381/:449/:474/:622`；
        断言全仓不出现 `core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)` 或「接入 Redis 后补 Delete」语义字符串
        （扫描范围必须覆盖 `*_test.go` 及注释，可限定 `internal/services/asset/` + `internal/api/v1/asset/`；`147-SUMMARY.md` 作为历史说明提及不计入）。
        同时**正向断言**保留：`InvalidateWorkstationHealth`、`GetReconciliationHealthByWorkstationKey`、
        `CacheKeyReconciliationHealthByWorkstation`、`base.InvalidatePattern` 仍存在（防止误删现成范式）。

      T4-5 【广播 pattern 与读键同源】按 T2-3 断言 `fmt.Sprintf(CacheKeyReconciliationHealthByWorkstation, "*")` 的前缀
        与 `GetReconciliationHealthByWorkstationKey("ws-1")` 一致，防止「删了一个平行宇宙的键」。
    </action>
    <verify>
      go test -count=1 ./internal/services/asset/
    </verify>
    <done>
      新测试文件落盘、与 `asset_gapfill_test.go` 同包无命名冲突；
      4 个写路径写后读必一致用例全绿；mock 断言 `DeleteByPattern ≥1`；失效失败不阻断且可观测；死键零命中静态断言全绿。
    </done>
    <forbidden>
      禁启动长驻进程 / 连生产 Redis / 写生产数据；禁伪造 Redis 双跑（skip 必须可观测并登记 Known Gap）；
      禁落盘 `.xlsx` 到仓库；禁把 `*_test.go` 放进非就近目录；禁为过测放宽断言。
    </forbidden>
  </task>

  <!-- ============================================================= -->
  <task id="5" name="自验证：只跑相关包 + 全量编译 + 反向红绿验证" req="CACHE-01/02/03 验收">
    <files>（无源码改动；反向验证临时注释后必须还原）</files>
    <action>
      T5-1 正向验证（只跑相关包，W1 并行期不跑全仓 `go test ./...` 以免与 145/146 互相等待）：
        go test -count=1 ./internal/services/asset/
        go build ./...
        （`go build ./...` 全量编译是 CLAUDE.md 硬约定，必须跑；产物不得落仓库根，本命令无 `-o`、不产文件。）
      T5-2 反向验证（红绿变异，证明联动失效是「立即一致」的真因，非测试自洽）：
        临时把 T2-4 中 4 个写路径的 `s.invalidateExceptionRuleCache(ctx)` 调用**注释掉**（或把 `invalidateExceptionRuleCache` 函数体临时改为 `return nil`），
        重跑 `go test -count=1 -run TestExceptionRuleWrite_ReadYourWrites ./internal/services/asset/`，
        **断言该测试必须红**（写后读拿到陈旧 `ExceptionHit/AppliedActions`）；
        随后**立即还原**注释，`go test -count=1 ./internal/services/asset/` 必须重新全绿。
        ⚠️ 反向验证的临时改动**不得留在最终文件里**、不得提交；在 `147-SUMMARY.md` 记录「反向验证：注释失效调用 → 红；还原 → 绿」及所用命令。
      T5-3 静态终检（判据：`invalidateCache` **旧标识符全库零命中（含测试与注释）**）：
        grep -rn "CacheKeyReconciliationExceptionRuleList\|GetReconciliationExceptionRuleListKey\|CacheKeyReconciliationExceptionRuleByID\|GetReconciliationExceptionRuleByIDKey" internal/ || echo CLEAN
        grep -rn "invalidateCache" internal/ || echo CLEAN          # 范围=全 internal/ 递归，必须覆盖 *_test.go 与注释（不得只扫 internal/services/asset/ 非测试 .go）
        grep -rni "接入 Redis.*补.*Delete\|未来接入 Redis 再补" internal/ || echo CLEAN
        三条都必须 CLEAN（`147-SUMMARY.md` 的历史说明除外）。
        **必须清零清单**：`reconciliation_exception.go:381/:449/:474/:622`（4 个生产写路径）、`:578` 函数定义及 `:574-580` 定义段注释、`:337` 悬空注释、
        `asset_gapfill_test.go:768`（含 `invalidateCache` 字样的注释）与 `:771`（`impl.invalidateCache()` 测试调用，第 5 处命中）、`asset_gapfill_test.go:117,118` 死键断言。
        另须 `go test -count=1 ./internal/services/asset/` 编译且通过（验证删旧函数后 `asset_gapfill_test.go:771` 已按 T1-4 定案②改调新失效函数、包不再 undefined）。
      T5-4 若 `go build ./...` 因 145/146 并行 executor 尚未落盘而短暂失败：只记录、不越界修其它 phase 文件；在 SUMMARY 登记「W1 并行期编译受邻 phase 影响」，交 Lead 集成复核时统一重跑全量矩阵。
    </action>
    <done>
      `go test -count=1 ./internal/services/asset/` 全绿、`go build ./...` 通过；反向验证红→绿闭环；三条 grep CLEAN；临时注释已还原。
    </done>
    <forbidden>
      禁跑 `git` 写操作；禁启动长驻进程；禁改其它 phase 文件面；禁把反向验证的临时注释留在最终代码里。
    </forbidden>
  </task>

  <!-- ============================================================= -->
  <task id="6" name="写 147-SUMMARY.md 并移交 Lead 集成复核（不提交）" req="交付物">
    <files>.planning/phases/147-cache-cache-01-02-03/147-SUMMARY.md（新增）</files>
    <action>
      T6-1 目录不存在则 `mkdir -p .planning/phases/147-cache-cache-01-02-03/`（本 plan 文件名由 SDK 给定，照写）。
      T6-2 SUMMARY 必含 6 段：
        ① **改动清单**（逐文件：删除了什么/新增了什么/`reconciliation_service.go` 仅注释级改动/`router.go` 是否动）；
        ② **3 个 REQ 满足情况**（CACHE-01 死键+空函数清零 / CACHE-02 方案 (a) 广播失效立即一致 / CACHE-03 可观测+失败传播+memory/redis 一致），逐条对应 ROADMAP Phase 147 的 4 条成功标准；
        ③ **验证结果**：`go test -count=1 ./internal/services/asset/`、`go build ./...`、反向红绿验证、3 条 grep CLEAN、T4 各测试名与结果；
        ④ **选定方案与性能评估**：§4.1 的 (a)/(b) 对比结论、写路径 1×Keys+1×MDelete 成本、读路径零开销、5min TTL 与 6 步聚合口径不变；
        ⑤ **Known Gaps**：Redis 侧真机双跑若 `t.Skip` 必须写明「需真 Redis / 不启动长驻进程 / 不连生产」；`ImportFromExcel` 若用了最小替身写明替身口径与未覆盖点；W1 并行期 `go build ./...` 若受邻 phase 影响写明交 Lead 重跑；
        ⑥ **明确「未 git commit」**：本 plan `commit:false`，零 git 写操作，提交由 Lead 统一执行；若触发 §4.2 则**必须**附「147 与 149/151 同文件 router.go 顺序登记 + 移交 Lead 集成复核」段。
      T6-3 SUMMARY 中**不得**出现「接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」作为待办/触发条件（可作历史「已作废」说明，但须明确作废）。
    </action>
    <done>
      `147-SUMMARY.md` 落盘、6 段齐全、明确未提交、必要时含 router.go 顺序移交段；
      本 plan 全部源码改动停在磁盘，等待 Lead 统一提交。
    </done>
    <forbidden>
      禁 `git add / commit / stash / checkout / merge / rebase / tag`；禁在 SUMMARY 里伪造 Redis 双跑 / 伪造全绿；禁把错误处方写成待办。
    </forbidden>
  </task>

</tasks>
```

---

## 6. 验证标准（must_haves ↔ ROADMAP Phase 147 四条成功标准）

| # | ROADMAP 成功标准 | 本 plan 落点 | must_have 判定 |
|---|------------------|--------------|----------------|
| 1 | 静态断言全仓零引用死键 / helper / `invalidateCache()` 旧标识符；`invalidateCache` 全库共 6 处命中 = **4 个生产写路径 + 1 处既有测试调用（`asset_gapfill_test.go:771`；`:768` 另有含 `invalidateCache` 字样的注释）+ 1 处函数定义 `:578`**，须全部处置，悬空契约消除 | Task 1（T1-1~T1-5）+ Task 4 T4-4 | **死键常量（含 `CacheKeyReconciliationExceptionRuleList`/`...ByID` 与两 helper）与空 `invalidateCache()` 函数体清零、6 处旧标识符全部处置（含第 5 处测试调用 `asset_gapfill_test.go:771` 与 `:768` 注释，grep 范围覆盖 `*_test.go` 及注释）、`go test -count=1 ./internal/services/asset/` 编译且通过**；`invalidateExceptionRuleCache` 真调 `DeleteByPattern`；4 个生产写路径调点已替换；`asset_gapfill_test.go:117,118` 两行删除 |
| 2 | 规则写成功后 `GetByWorkstation` 的 `exceptionHit/AppliedActions` 立即反映新规则（写后读必一致，5min TTL 缓存保留但 exception 不再陈旧）；List/GetByID/MatchTest/MatchException/DetectLayer3 同进程立即一致 | Task 2（方案 (a)）+ Task 4 T4-1 | `TestExceptionRuleWrite_ReadYourWrites` 4 子用例全绿；`reconciliationHealthCacheTTL=5min` 与 `getByWorkstationWithCache` / `computeByWorkstation` 未改；反向验证（Task 5 T5-2）注释失效调用后必须红 |
| 3 | `Cache.Type=memory`（`*MemoryCache`）与 `redis`（`*MultiLevelCache`）两种配置行为一致；失效动作真调 `Delete`、nil-safe、失败可传播（返回 error 或日志/metric），**禁止空函数体** | Task 2 T2-2 + Task 3 + Task 4 T4-2/T4-3 | `invalidateExceptionRuleCache(ctx) error` 有 ctx/有 error/真调/nil-safe；4 写路径 `applogger.Warnf`；mock `DeleteByPattern ≥1`；memory 真删断言；`CacheProvider` 接口与两实现零改动 |
| 4 | 明确作废触发条件「项目接入 Redis 后补 `core.Cache.Delete(ctx, 死键)`」（注释/文档不留该处方）；**不改**全局 `CacheProvider` 接口 | Task 1 T1-2/T1-5（A3）+ Task 4 T4-4 + Task 6 T6-3 + 禁区 | 3 条 grep CLEAN；`147-SUMMARY.md` 只作「已作废」历史说明；`base/cache_provider.go` / `NoOpCacheProvider` / 20+ 消费文件零改动 |

**额外硬约束（全程）**：`internal/api/router.go` 默认不动（§4.2 仅在必要时触发且必须登记）；零 git 写操作；不启动长驻进程；不写生产数据；不改其它 phase 文件面；`go build ./...` 改后必跑。

---

## 7. 禁止清单（禁区，executor 逐条自检）

1. **git 写操作**：`add / commit / stash / checkout / merge / rebase / tag / reset` 全禁（frontmatter `commit:false`，Lead 统一提交）。
2. **其它 phase 文件面**：`internal/utils/operlog/**`（145）、`internal/core/security/**`（146）、`internal/services/operations/**`（148/149/150）、`xingran-react-frontend/**`（149/150）、`.golangci.yml`（148）、`.github/workflows/ci.yml`（145）—— 一律不碰。
3. **方案①**：改全局 `base.CacheProvider` 接口 / `NoOpCacheProvider` / 20+ 消费文件 / 大量 mock 连锁改 —— 仅缓解且未命中真实陈旧点。
4. **为死键补 `core.Cache.Delete(ctx, CacheKeyReconciliationExceptionRuleList)`** 或等价「接 Redis 补 Delete」处方 —— 错误处方，作废。
5. **新增 migration**（方案② 规则版本号需 schema）—— 本 phase 用③；`internal/core/db/migrations/**` 不碰。
6. **启动长驻进程**（dev server / Redis 真机 / watcher）—— 需走 DSH 托管终端，本 plan 不启；Redis 双跑只做可观测 `t.Skip` + Known Gap。
7. **写生产数据**；Excel 模板落盘 `.xlsx` 到仓库（AGENTS.md 强制动态生成 / `t.TempDir()`）。
8. **改 `computeByWorkstation` 6 步聚合 / `reconciliationHealthCacheTTL` / 缓存值形状**（会削性能价值、破坏响应形状不变式）。
9. **`internal/api/router.go` 无 §4.2 登记即改动** —— ROADMAP.md:145 已硬登记 149→151 顺序，147 默认绕开。
10. **留「定义缓存键却从不读写」的悬空契约 / 空函数体 / `_ =` 静默吞错且无日志**。

---

## 8. 完成回执（给 Lead）

返回：`## PLANNING COMPLETE (Phase 147)` + 任务数（6）/ 验证命令（`go test -count=1 ./internal/services/asset/` + `go build ./...` + 反向红绿验证 + 3 条 grep CLEAN）/ 文件面（`internal/services/asset/**` 5 个文件 + `internal/api/v1/asset/reconciliation_exception_router.go` 仅一行装配 + `147-SUMMARY.md`）/ 是否动 `internal/api/router.go` 一句话（默认：**否**；`router_go_touched: false`）。
若 Task 2 触发 §4.2，回执须追加「已登记 147 先改 router.go 一行、149/151 后改基于最新文件，移交 Lead 集成复核」。

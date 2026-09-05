# Phase 92: 缓存层三处架构统一 - Research

**Researched:** 2026-09-05
**Domain:** Go 后端缓存抽象层重构（Go 1.24 泛型 + type alias 迁移 + 接口消歧）
**Confidence:** HIGH（全部结论基于本仓实测 grep/read/编译实验，无外部依赖面）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions（D-01..D-10，复制自 92-CONTEXT.md `<decisions>`，逐字）

- **D-01: 迁 base + 依赖倒置** — 新建 `internal/services/base/cache_service_base.go`，基类对 TTL 配置的依赖改为 base 内定义的小接口（TTLResolver 风格：`GetDurationWithDefault(key string, default time.Duration) time.Duration`，`services.CacheConfigService` 隐式满足）。base 包保持零依赖（Go consumer-defined interface 惯例，与 Phase 91 base 包"纯抽象层"定位一致）。ROADMAP CACHE-UNIFY-01 原文达成。
- **D-02: CacheProvider 全家搬 base + alias** — `CacheProvider` 接口（9 方法：GetOrSet/Delete/DeleteByPattern/MGet/MDelete/Exists/SetTTL/GetTTL/GetStats）+ `NoOpCacheProvider` + `CacheStats`/`CacheEntry` 伴生类型一起迁 base。`system/cache_provider.go` 留 `type CacheProvider = base.CacheProvider` 别名，10 个嵌入文件的字段引用零改动（Go type alias 完美兼容，alias 后续可删）。缓存抽象全套归 base，语义完整。
- **D-03: 泛型包级函数族** — base 包提供 `GetOrSetJSON[T any]` / `SetJSON[T]` / `Invalidate` / `InvalidatePattern` 函数族。每处样板 12-15 行 → ~5 行单 return（类型安全：无 interface{} 闭包、无 `var result` 中转、无 `NoOpCacheProvider.setValue` 反射兜底）。**技术约束依据：Go method 不能有自己的类型参数**（`func (b *CacheServiceBase) Get[T]()` 非法），泛型 struct 单 T 也不匹配现实（一个 cacheImpl 有多种缓存值类型，如 userCacheService 同时缓存 `*User`/`[]User`）——泛型包级函数是唯一全类型安全路径。基类退化为薄持有 TTLResolver。与 Phase 91 base 包函数式风格一致。
- **D-04: base 唯一失效底层 + Invalidator 委托** — base 提供唯一失效底层实现（nil 防护 + 统一日志）；`cache_utils.go` 的 `InvalidateCacheByPattern`/`InvalidateCacheByKey` 迁 base 后删除；`operations/CacheInvalidator` **保留** struct（Excel 导入管道的 entityType 分发 + ExcelConfig.CachePatterns 读取语义，excel_service 在用）但底层循环改为委托 base。两套重复逻辑归一，分发语义保留。
- **D-05: 验收锚点 = 定性清零 + LOC ≥200** — 定性底线：33 处 GetOrSet 样板全部走 `base.GetOrSetJSON`，每处 WithCache 方法体 ≤6 行，仓内不再有 interface{} 闭包式 GetOrSet 样板。量化锚点：LOC 净减 ≥200（实测 33 处 × 每处省 6-9 行 ≈ 200-300 诚实区间；Phase 91 D-07 混合标准先例）。
- **D-06: 原地定性为基础设施** — `DataCacheService` 定性为 CacheProvider 实现底座（`system.NewCacheProvider()` 适配器的 Adaptee + `pkg/cache.Cache` 的业务封装层），永久留 services root。本期消除其平行 `GetExpiration`（委托 base 统一 TTL 逻辑；root import base 无循环）。**CACHE-UNIFY-04 措辞同步修订**（REQUIREMENTS + ROADMAP，Phase 90 先例 commit 3a2efe5）：删"迁移到 system/ 下"，改为"原地定性 + 消除平行 TTL 逻辑"。理由：root↔system import cycle 硬约束使字面迁移不可行；12+ API 文件 import 风暴与收益不成比。
- **D-07: 不标 @Deprecated，定位注释** — 文件头注释明确双定位：① `pkg/cache.Cache` 的业务封装层（GetOrSet/JSON 序列化/统计）② `base.CacheProvider` 接口的实现底座（经 `system.NewCacheProvider` 适配）。新增业务缓存代码应走 `base.GetOrSetJSON` + `CacheProvider`，不应直接新增 DataCacheService 调用点。
- **D-08: monitor 同名接口纳入本期，仅 rename 消歧** — `monitor/cache_service.go:49` 的 `CacheProvider`（Get/Set/Keys/FlushDB 原始监控操作型）rename 为不撞名的名字（如 `CacheOperator`，最终名 planner 定）。纯包内 rename + `api/v1/monitor/cache_router.go` adapter 同步，语义零变更。D-02 定名 base.CacheProvider 为唯一权威后，同名异形接口必须消除，否则认知混乱。
- **D-09: SC-5 验证 = miniredis 自动化** — 用 v1.27 INFRA-01 已引入的 miniredis/v2 写自动化集成测试：GetOrSetJSON 读写命中/穿透、TTL 过期、pattern/key 失效、NoOp 透传、CacheStats 读取。可回归、进 CI gate（Phase 91 handler smoke 同款思路）；"端到端"语义落地为 provider 层集成测试。手动冒烟不做强制要求。
- **D-10: CLAUDE.md + invariants 都做** — ① 收口时修订 CLAUDE.md 过时的 "Dual Cache Architecture" 段（root 6 件套描述已失实），新增 Cache Service Convention 段（锁 `base.GetOrSetJSON`/`CacheProvider` 单一权威路径，Phase 90 D-11 先例格式）；② 新增 invariants 扫描锁 "cache_impl 无 interface{} 闭包式 GetOrSet 残留"（warning 不 fail，Phase 89/90 模式）。

### Claude's Discretion（planner/researcher 自决区，复制自 CONTEXT.md）

- TTL 参数在 `GetOrSetJSON` 签名中的形态（预解析 Duration 传入 vs configKey+default+resolver 三参传入）
- `system.CacheServiceBase` 旧名是否同样留 type alias（与 D-02 alias 策略对齐，一次改干净 vs 渐进）
- base 包内文件布局（cache_service_base.go / cache_provider.go / cache_functions.go 拆分粒度）
- 泛型函数的命名最终形态（GetOrSetJSON vs GetOrSetCached vs 其他）
- `monitor.CacheProvider` rename 的目标名（CacheOperator 为建议非锁定）
- 33 处迁移在 3 个 plan 中的分组（ROADMAP 建议 92-01 base 抽取 / 92-02 system 9 文件 / 92-03 operations+root，planner 可按实际微调）
- 错误包装与日志格式的保留/统一方式
- invariants 扫描的具体检测模式（AST 或正则，参考 Phase 89/90 白名单机制）
- `Invalidate`/`InvalidatePattern` 的签名细节（返回 error 还是 void+warn 日志，对齐现状语义）

### Deferred Ideas (OUT OF SCOPE)

- DataCacheService 字面迁移 + mac_history 接口化改造（v1.30+ 候选，本期 D-06 明确不做）
- CacheInvalidator struct 消除 / excel 管道重构（D-04 保留委托）
- monitor 缓存监控接口语义统一（D-08 仅 rename 消歧）
- `cache_config_service.go`（TTL 配置真相源）重构/迁移 — 本期不动，仅作为 TTLResolver 的隐式实现
- `mac_history_cache_decorator.go` / `template_cache.go` 统一评估 — root 另两个缓存文件，无 CacheServiceBase 模式
- config_backup 三处 TODO 闭环（Phase 93）、前端 API 工厂化（Phase 94）
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description（REQUIREMENTS § CACHE-UNIFY :68-76） | Research Support |
|----|-------------|------------------|
| CACHE-UNIFY-01 | 抽 `internal/services/base/cache_service_base.go` 为单一基类（模板方法） | D-01/D-03：base 零依赖已验证可行（root 已 import base 18 文件）；Go method 不能有类型参数已编译实证 → 形态 = TTLResolver 接口 + 薄基类 + 泛型包级函数族；SC-1 措辞按 D-03 修订 |
| CACHE-UNIFY-02 | `system/*_cache_impl.go` (9 文件) 全部继承新基类，删除重复模板代码 | 29 处 GetOrSet 样板已逐一枚举（file:line）；type alias 迁移机制已验证（alias=同一类型，10 嵌入点 + 全部外部消费者零改动）；notice 逃兵归队 diff 已明确 |
| CACHE-UNIFY-03 | `operations/*_cache_impl.go` 全部继承新基类 | floor_cache_impl.go 实测 **3 处**真实调用点（43/60/179；CONTEXT 的"4 处"含 1 行注释 grep 假阳性）；CacheInvalidator 委托改造面已测绘 |
| CACHE-UNIFY-04 | legacy root cache service 迁移（**措辞按 D-06 修订为原地定性 + 消除平行 TTL 逻辑**） | root↔system import cycle 已证实（root 18 文件 import base、0 文件 import system；system import root）；DataCacheService.GetExpiration 生产代码 0 调用者（仅 2 个测试文件锁定）→ 委托 base 即可；12+ API 文件 `services.DataCacheService` 引用零改动 |
| CACHE-UNIFY-05 | base 测试 + system/operations 缓存单测覆盖；`go test ./internal/services/...` 0 失败 | miniredis v2.38.0 已在 go.mod（v1.27 D-02）；参考模板 data_cache_service_79_01_test.go（双装配 + FastForward + 禁 t.Parallel）；AST/regex invariants 先例 pkg/constants/pagination_test.go |
</phase_requirements>

## Summary

本 phase 是纯 Go 源码重构，无新外部依赖。CONTEXT.md 的现实校准经实测全部成立并有三处精确化修正：**(1)** 真实 GetOrSet 样板调用点是 **32 处而非 33**（system 29 + operations 3；"33" 来自 `grep -c` 把 floor_cache_impl.go:168 的注释行也计入了）；**(2)** 失效辅助 `InvalidateCacheByPattern/ByKey` 的调用面远超 system/operations —— **duty(5) + knowledge(4) + network(5) + workorder(4) + core.go(1) 共 19 处外部调用点**也引用 `system.InvalidateCacheByKey/ByPattern`，D-04 的"迁 base 后删除"会由编译器强制这 19 处同步改写（机械 1 行改动，但必须在 plan 里显式列出）；**(3)** 发现 CONTEXT 未记录的第 4 个 CacheProvider 实现 `api/v1/system/cache_adapter.go`（`NewDataCacheAdapter`，生产代码零调用者的死代码 duplicate）。

技术路线上，D-03 的核心前提（Go method 不能有自己的类型参数）已用本机 go1.24.5 工具链编译实证（`syntax error: method must have no type parameters`），泛型包级函数族 + 匿名 struct 类型实参 + 接口 type alias 全部编译验证通过。import cycle 硬约束同样实证：services root 现有 18 个文件 import `internal/services/base`、0 个文件 import `internal/services/system`，root→base 方向安全，D-01/D-02/D-06 全部可行。

**一个 planner 必须处理的隐性陷阱：** D-01 把 `CacheServiceBase.Config` 从具体指针 `*services.CacheConfigService` 改为 `TTLResolver` 接口后，现有测试传入 nil config（如 `NewFloorServiceWithCache(db, adapter, nil)`）会变成 **typed-nil interface**，`if b.Config != nil` 判空失效，而 `CacheConfigService.GetDurationWithDefault` 第一行就 `s.mu.RLock()` —— nil receiver 直接 panic。一行修复：在 `GetDurationWithDefault` 顶部加 nil-receiver 防护（详见 Pitfall 1）。

**Primary recommendation:** 按 ROADMAP 3-plan 结构执行（92-01 base 包 + alias 翻转 + nil-receiver 防护 → 92-02 system 9 文件 29 处 → 92-03 operations + D-04 失效调用面 19 处外部改写 + root 定性 + monitor rename），LOC 锚点采用"样板调用段毛减 + base 投资单列"的双口径报告（Phase 91 OVR-91-01 先例）。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 缓存读/写/失效模板逻辑（GetOrSetJSON/SetJSON） | API/Backend — `internal/services/base` | — | base 是零依赖纯抽象层（Phase 91 定位），只承接管道不做业务解释 |
| TTL 配置解析（TTLResolver） | API/Backend — base 定义接口 | `services.CacheConfigService`（root）隐式实现 | Go consumer-defined interface 惯例；配置真相源留 root 原位（CONTEXT 范围外） |
| CacheProvider 实现（Redis 路径） | API/Backend — root `DataCacheService`（Adaptee）+ system `NewCacheProvider`（Adapter） | `pkg/cache`（L1/L2 多级缓存） | D-06：DataCacheService 永久留 root，import cycle 硬约束 |
| CacheProvider 实现（NoOp / pkg/cache.Cache 直连） | API/Backend — base `NoOpCacheProvider` + system `CacheAdapter` | — | 路由层 DataCacheService==nil 时的 fallback（6 处构造点） |
| Excel 管道缓存失效分发 | API/Backend — `operations.CacheInvalidator` | base 失效底层 | D-04：分发语义（entityType + ExcelConfig.CachePatterns）保留，底层循环委托 base |
| 缓存监控（Cache Monitor 页） | API/Backend — `monitor/cache_service.go` + `api/v1/monitor/cache_router.go` | — | D-08：仅 rename 消歧，语义零变更 |
| 缓存键构造 | API/Backend — `system/cache_keys.go`（367 行单一真相源） | — | P2-A2 已治理，本期不动，键构造在 32 处迁移中逐字保留 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go generics（包级泛型函数 + type alias） | go1.24.0 / toolchain go1.24.5（go.mod 实测） | D-03 泛型函数族 + D-02 alias 迁移 | 语言内建，零依赖；method 类型参数限制已编译实证 |
| `internal/services/base`（既有包） | Phase 91 产物 | 新缓存抽象同居处（GORMRepository 已在） | 零依赖基线已建立（仅 gorm + pkg/errors + pkg/logger） |
| `github.com/alicebob/miniredis/v2` | v2.38.0（go.mod，test-only，v1.27 D-02） | D-09 provider 层集成测试 | 已入库的测试基建，TTL 用 `mr.FastForward` 推进（禁裸 sleep 纪律） |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `pkg/cache` MemoryCache | 既有 | 纯进程内测试装配（cache_infra_test.go 模式） | base 包单测（无需 Redis） |
| glebarez/sqlite + testify | 既有 | 既有 impl 测试回归网 | 既有 `*_cache_impl_test.go` / `*_gapfill_test.go` 保持绿 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 泛型包级函数族（D-03） | 泛型 struct `CacheClient[T]` | 单 T 不匹配现实（一个 impl 缓存多种类型，userCacheService 同缓存 `*User`/`[]User`/`PageResult`）；且基类字段化 T 会让 10 个嵌入点各自实例化，嵌入模式崩坏。D-03 已锁定，不再展开 |
| type alias 一次翻转（D-02） | 逐文件改字段类型 `base.CacheProvider` | 40+ 文件引用面改动 vs alias 零改动；alias 语义 = 同一类型，无适配层风险。D-02 已锁定 |
| 删除 `system.InvalidateCache*`（D-04） | 留薄 wrapper 委托 base | wrapper 可让 duty/knowledge/network/workorder 19 处零改动，但违背 D-04"删除"锁定；且 wrapper 本身就是"两套重复逻辑"残留。建议按 D-04 删除 + 19 处机械改写 |

**Installation:**
```bash
# 无新增依赖 — 本 phase 零安装。全部所需已在 go.mod（go 1.24.5 / miniredis v2.38.0 test-only / go-redis v9.7.0 / testify）
```

**Version verification:** `go version` → go1.24.5（本机实测）；`grep miniredis go.mod` → v2.38.0 test-only（实测）。[VERIFIED: local toolchain + go.mod]

## Package Legitimacy Audit

> 本 phase **不安装任何新外部包**（纯源码重构 + 既有测试依赖）。依协议仍执行核查：

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| （无新增 — miniredis/v2.38.0、go-redis/v9.7.0、testify 均为既有 go.mod 依赖，v1.26/v1.27 已审计入库） | — | — | — | — | — | N/A |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                        ┌──────────────────────────────────────────────┐
                        │              internal/services/base          │
                        │  (零依赖：gorm + pkg/errors + pkg/logger only) │
                        │                                              │
                        │  TTLResolver (interface, D-01)               │
                        │  CacheServiceBase{Config TTLResolver} (薄基类) │
                        │  CacheProvider (9 方法, D-02 迁入)             │
                        │  NoOpCacheProvider / CacheStats / CacheEntry  │
                        │  GetOrSetJSON[T] / SetJSON[T] (D-03)          │
                        │  Invalidate / InvalidatePattern (D-04)        │
                        └──────────────▲───────────────────────────────┘
                                       │ import (root 已有 18 文件走此方向，实证无环)
        ┌──────────────────────────────┴───────────────────────────────┐
        │                internal/services (root)                      │
        │  CacheConfigService.GetDurationWithDefault —隐式满足→ TTLResolver │
        │  DataCacheService (D-06 原地定性: pkg/cache 封装层 + Adaptee)   │
        │    └─ GetExpiration 平行实现 → 委托 base TTL 逻辑 (D-06)        │
        └──────▲───────────────────────────────────────────────────────┘
               │ import (现状已存在: system/cache_utils.go:8)
   ┌───────────┴───────────────────────────────────────────────────┐
   │            internal/services/system                           │
   │  type CacheProvider = base.CacheProvider   (D-02 alias)       │
   │  type NoOpCacheProvider = base.NoOpCacheProvider (建议同法)     │
   │  type CacheServiceBase = base.CacheServiceBase (discretion)   │
   │  NewCacheProvider(DataCacheService) → Adapter (D-06 不动)      │
   │  9 个 *_cache_impl.go: 29 处样板 → base.GetOrSetJSON (D-03)     │
   │  cache_utils.go: InvalidateCache* 迁 base 后删除 (D-04)        │
   │  cache_adapter.go / cache_manager.go / cache_keys.go: 不动     │
   └──────▲───────────────────────────▲────────────────────────────┘
          │ import                    │ import (跨包嵌入, alias 保护零改动)
   ┌──────┴──────────────┐   ┌────────┴──────────────────────────────┐
   │ operations          │   │ duty / knowledge / network / workorder │
   │ floor_cache_impl.go │   │ (范围外但 alias+删除强制 19 处失效调用    │
   │  3 处样板 → base    │   │  改写: duty 5 / knowledge 4 / network 5 │
   │ CacheInvalidator    │   │  / workorder 4 + core.go 1)            │
   │  → 委托 base (D-04) │   └────────────────────────────────────────┘
   └─────────────────────┘
   ┌─────────────────────────┐   ┌────────────────────────────────────┐
   │ monitor (D-08 rename)   │   │ internal/api/v1/* (12+ 文件)        │
   │ CacheProvider           │   │ systemServices.NewCacheProvider /   │
   │  → CacheOperator(建议名) │   │ NoOpCacheProvider / WithCacheProvider│
   │ cache_router.go 同步     │   │ 全部 alias 保护零改动                 │
   └─────────────────────────┘   └────────────────────────────────────┘
```

主用例数据流（迁移后 GetByIDWithCache）：
```
handler → UserService(=userCacheService 装饰器)
  → base.GetOrSetJSON[*models.User](ctx, s.cache, key, s.TTL(cfg, 30m), closure)
    → cacheProvider.GetOrSet(ctx, key, &result, ttl, wrapper)   [结果仍走 JSON 往返，行为不变]
      → 命中: unmarshal → return result
      → 未命中: closure() → sync Set (P0 #9) → return
```

### Recommended Project Structure

```
internal/services/base/
├── service.go                  # Phase 91: GORMRepository[T]（既有，不动）
├── list_request.go             # Phase 91（既有，不动）
├── cache_service_base.go       # 新增: TTLResolver + CacheServiceBase 薄基类
├── cache_provider.go           # 迁入: CacheProvider + NoOp + CacheStats + CacheEntry（D-02，逐字搬迁）
├── cache_functions.go          # 新增: GetOrSetJSON[T] / SetJSON[T] / Invalidate / InvalidatePattern（D-03/D-04）
├── service_test.go             # Phase 91 契约锁（既有）
└── cache_service_base_test.go  # 新增: CACHE-UNIFY-05（命名按 REQUIREMENTS 原文）
```
（文件拆分粒度为 discretion 区，上表为建议；也可合并为 2 文件。）

### Pattern 1: 泛型包级函数族（D-03 核心形态）

**What:** 用包级泛型函数包装 `provider.GetOrSet`，调用点从 10-15 行五段式样板变为单 return。
**When to use:** 所有 system/operations cache_impl 的缓存读取路径（32 处）。
**Example:**
```go
// Source: 本仓实测样板 (user_cache_impl.go:38-52) + D-03 CONTEXT 迁移预览
// ---- base/cache_functions.go ----
// GetOrSetJSON 类型安全的读穿透缓存：命中反序列化，未命中执行 query 并同步写缓存。
// 薄包装：委托 p.GetOrSet（JSON 往返/P0#9 同步写语义逐字保留，零行为变更）。
func GetOrSetJSON[T any](
    ctx context.Context,
    p CacheProvider,
    key string,
    ttl time.Duration,
    query func() (T, error),
) (T, error) {
    var result T
    err := p.GetOrSet(ctx, key, &result, ttl, func() (interface{}, error) {
        return query()
    })
    return result, err
}

// ---- 迁移后调用点（12-15 行 → 3-4 行）----
func (s *userCacheService) GetByIDWithCache(ctx context.Context, id string) (*models.User, error) {
    return base.GetOrSetJSON(ctx, s.cache, GetUserByIDKey(id),
        s.GetExpiration(services.CacheConfigUserByID, 30*time.Minute),
        func() (*models.User, error) { return s.userService.GetByID(ctx, id) })
}
```
注意 `s.GetExpiration` 来自嵌入基类——若 planner 选预解析 Duration 形态（discretion 区），这是最平滑路径：30 个嵌入点调用 `s.GetExpiration(...)` 一行不改，只是结果作为参数传入泛型函数。

### Pattern 2: type alias 翻转（D-02 迁移机制）

**What:** `system/cache_provider.go` 中定义删除、原位留 alias，全部引用点零改动。
**When to use:** CacheProvider / NoOpCacheProvider / CacheStats / CacheEntry（D-02）+ CacheServiceBase（discretion，建议同法一次改干净）。
**Example:**
```go
// Source: Go spec alias 语义（alias = 同一类型的替代名，无适配层）+ 本机编译验证
// ---- system/cache_provider.go（迁移后）----
type CacheProvider = base.CacheProvider
type NoOpCacheProvider = base.NoOpCacheProvider
type CacheStats = base.CacheStats
type CacheEntry = base.CacheEntry
type CacheServiceBase = base.CacheServiceBase // discretion 建议：10 个嵌入点构造字面量 CacheServiceBase{Config: config} 零改动

// 编译期双保险（建议）：
var _ CacheProvider = (*NoOpCacheProvider)(nil)
```
已验证的外部消费者（全部 alias 后零改动）：operations/{cache_invalidator,excel_service,floor_cache_impl}、duty/knowledge/network/workorder 的 cache_impl（字段类型 `systemServices.CacheProvider`）及其测试的 `var _ systemServices.CacheProvider = (*mockCacheProvider)(nil)` 断言、api/router.go、6 处 `&systemServices.NoOpCacheProvider{}` 路由 fallback、`system.WithCacheProvider`（user_sync_service.go:46 UserSyncOption）、core.go 3 处装配。

### Pattern 3: 失效辅助归一（D-04）

**What:** base 提供唯一失效底层（nil 防护 + warn 日志），system 包级函数删除，CacheInvalidator 底层循环改委托。
**Example:**
```go
// Source: cache_utils.go:63-79 + cache_invalidator.go:39-45 现状合并
// ---- base/cache_functions.go ----
// InvalidatePattern 按模式列表失效缓存（唯一底层：nil 防护 + 统一日志）。
// 返回 void + warn 日志（对齐现状 InvalidateCacheByPattern 语义，discretion 可改 error）。
func InvalidatePattern(ctx context.Context, p CacheProvider, patterns []string, module string) {
    if p == nil {
        logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", module)
        return
    }
    for _, pattern := range patterns {
        if err := p.DeleteByPattern(ctx, pattern); err != nil {
            logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", module, pattern, err)
        }
    }
}
func Invalidate(ctx context.Context, p CacheProvider, keys []string, module string) { /* 同构, 用 Delete */ }

// ---- operations/cache_invalidator.go（保留分发器，底层委托）----
func (c *CacheInvalidator) InvalidateByEntityType(ctx context.Context, entityType string, patterns []string) error {
    if len(patterns) == 0 { /* Debugf 保留 */ }
    base.InvalidatePattern(ctx, c.cache, patterns, entityType) // nil 防护已内聚
    return nil
}
```

### Anti-Patterns to Avoid
- **泛型 method：** `func (b *CacheServiceBase) Get[T any]()` 编译直接报错（`method must have no type parameters`，本机实证）。泛型只能放包级函数。
- **在泛型函数里重写 Get/Set/JSON 逻辑：** 应薄委托 `p.GetOrSet`（保留 P0 #9 同步写、`data == ""` 分支、JSON 往返语义），重新实现 = 行为漂移风险。
- **把 `CacheServiceBase.Config` 改接口却不处理 typed-nil：** 见 Pitfall 1。
- **顺手"修复"范围外模块：** duty/knowledge/network/workorder 的 GetOrSet 样板（12 处）与样板同构但**不在 32 处范围内**（v1.30+ 候选）；scope constrainment（CLAUDE.md 纪律）。唯一例外是 D-04 删除强制的那 19 处失效调用改写——那是编译器要求的，不是范围扩张。
- **迁移时改缓存键构造：** CONTEXT 明确"键构造不变"。floor 的 `fmt.Sprintf("floor:building:%s", ...)` 等内联键虽不在 cache_keys.go，本期不改（Redis 数据兼容）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TTL 过期测试推进 | time.Sleep + 真实等待 | miniredis `mr.FastForward(d)` | v1.27 纪律（data_cache_service_79_01_test.go 头注释明令禁裸 sleep）；快且无 flake |
| Redis 集成测试环境 | 真 Redis / docker | miniredis.RunT（go.mod test-only 已入库） | CI 可跑、零外部依赖（D-09 锁定） |
| AST 残留扫描解析器 | 手写正则套 AST 全家桶 | 参照 pkg/constants/pagination_test.go 的 `parser.ParseFile` 模式（Stability + Count 双锁） | 项目已验证的 invariants 模式，Phase 89/90 两期实战 |
| 泛型接口 mock 断言 | 运行时反射检查 | `var _ system.CacheProvider = (*mock)(nil)` 编译期断言 | duty_cache_impl_test.go:48 既有模式，alias 下零改动生效 |
| nil provider 防护 | 每个调用点 if nil | base.Invalidate/InvalidatePattern 内聚 nil 防护（D-04 锁定） | 消除 cache_utils/cache_invalidator 两套重复 nil-check |

**Key insight:** 本 phase 的一切"新逻辑"都应该是**搬迁或薄包装**——GetOrSetJSON 委托既有 provider 方法、失效底层合并两套现有循环、alias 保持类型同一性。任何"重新实现"都会引入行为漂移，违背 v1.29 D-05 零业务行为变更底线。

## Runtime State Inventory

> 重构类 phase，按协议逐类显式回答（无遗漏留空）：

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | **None** — 缓存键构造逐字不变（D-03/CONTEXT 锁定，实测 32 处迁移不改 key 字符串）；Redis 现存键在迁移后继续命中；无 DB schema 改动 | none |
| Live service config | **None** — 无外部服务配置引用被改名；D-08 monitor rename 是包内 Go 类型名，不触 Redis/配置文件 | none |
| OS-registered state | **None** — 纯源码重构，无 OS 注册面 | none |
| Secrets/env vars | **None** — 无密钥/环境变量名变更 | none |
| Build artifacts | **None** — 纯源码改动，无 egg-info/编译产物残留问题；`go build ./...` 即验证 | none |

## Common Pitfalls

### Pitfall 1: TTLResolver 接口化后的 typed-nil panic（D-01 最高风险点）
**What goes wrong:** `CacheServiceBase.Config` 从 `*services.CacheConfigService` 改为 `TTLResolver` 接口后，测试传入 nil config（实测存在：`geocoding_photo_floor_test.go:379`、`notice_service_gapfill_test.go:169` 均传 `nil`）→ composite literal `CacheServiceBase{Config: config}` 把 **typed-nil 指针**装进 interface → `if b.Config != nil` 判定为非 nil → 调用 `GetDurationWithDefault` → 第一行 `s.mu.RLock()` 对 nil receiver **panic**。
**Why it happens:** Go interface 的 nil 判定只看 type+value 双元组；typed-nil ≠ nil interface。
**How to avoid:** 在 `services/cache_config_service.go:392` 的 `GetDurationWithDefault` 顶部加 nil-receiver 防护：`if s == nil { return defaultDuration }`（一行，root 包内，语义与现状 nil-config → default 完全一致）。备选：构造 helper 做判空——但 10 个嵌入点用 composite literal，改动面大，不推荐。
**Warning signs:** 迁移后跑 `go test ./internal/services/...` 出现 nil pointer dereference panic（而非断言失败）即命中此坑。

### Pitfall 2: "33 处"计数与 D-05 锚点的口径
**What goes wrong:** CONTEXT 的 33 = grep -c 把 `floor_cache_impl.go:168` 注释行（"参考 GetTree 的 cache.GetOrSet 模式"）计入；真实调用点 **32**（system 29 + operations 3，已逐 file:line 枚举）。另外"每处省 6-9 行"按实测偏乐观：menu GetTree 类短样板（10 行体）迁移后省 6-7 行，32 处毛减约 **190-290 行**；若把 base 新增生产代码（~100-160 行：TTLResolver + 薄基类 + 4 函数族 + 注释）计入净额，repo 级净减可能 **达不到 ≥200**。
**Why it happens:** grep 计数假阳性 + 锚点未定义 LOC 统计口径。
**How to avoid:** Plan 里显式定义口径——建议沿用 Phase 91 OVR-91-01 先例：**样板调用段毛减单列（达成 ≥200 的口径）+ base 包投资单列**，SUMMARY 双报告；同时注意 D-05 的"净减 ≥200"若严格按 repo 全口径理解，需要在执行中用 compact 形态（单行 return）并在 notice/floor 复杂点之外不损失行数。
**Warning signs:** 92-02 收尾时 numstat 显示毛减 < 190 → 立即复核是否有站点未收敛到单 return 形态。

### Pitfall 3: D-05 定性底线"方法体 ≤6 行"对 3 个复杂站点不可字面达成
**What goes wrong:** `notice.GetUserNotices`（cacheKey if/else 分支 + 匿名 struct 组装 + List/Total 拆包）、`floor.GetFloorsByBuildingID`（闭包内直查 db，非委托 floorService）、`user.List`（buildListCacheKey 已抽出）——迁移后方法体仍 >6 行。
**How to avoid:** 把规则解读为"**GetOrSet 调用段 ≤6 行**"（键构造/结果拆包不算）；notice 的键分支可抽 `buildMyNoticesKey(userID, page, pageSize, status)` helper 顺带减行；floor 的闭包内 db 查询原样保留（搬进泛型函数的 query 闭包，形状仍匹配）。
**Warning signs:** verifier 按"整方法 ≤6 行"字面验收会误报 3 处。

### Pitfall 4: D-04 删除的编译器强制涟漪（范围外 19 处）
**What goes wrong:** `system.InvalidateCacheByPattern/ByKey` 删除后，duty(5)/knowledge(4)/network(5)/workorder(4)/core.go(1) 的 19 处调用**编译失败**——这些模块在"33 样板"叙事之外，plan 若不列会像范围蔓延。
**How to avoid:** Plan 明确列出这 19 处为"D-04 编译器强制同步改写"（每处 1 行 `systemServices.InvalidateCacheByXxx(...)` → `base.InvalidateXxx(...)`，机械替换）；也可在 92-01（迁移发生时）顺手完成，让编译器当 checklist。
**Warning signs:** `go build ./...` 报 undefined: system.InvalidateCacheByKey 即此坑（预期内的编译驱动，不是事故）。

### Pitfall 5: NoOp 路径的潜在行为改善（单向）
**What goes wrong:** 现状 NoOpCacheProvider.setValue 反射在"闭包返回 *T、dest 为 T 值"时静默丢弃（`val.Type().AssignableTo(elem.Type())` false → 跳过）。实测案例：`user_cache_impl.go:186-199` List 的 `var result PageResult` + 闭包返回 `*PageResult` → NoOp 下返回零值 PageResult（Redis 路径经 JSON 往返正常）。迁移到 `GetOrSetJSON[*PageResult]` 后 NoOp 路径**反而变正确**（dest 变 `&result` 即 `**PageResult`，assignable 成立）。
**Why it happens:** 这正是 CONTEXT D-03 说的"setValue 反射兜底"缺陷——泛型包装在类型层面消除了 mismatch 类。
**How to avoid:** 无需动作，但 plan 应在"零业务行为变更"叙事中注明：NoOp fallback 场景（DataCacheService==nil 的路由）下个别站点行为从"返回零值"修正为"返回真实数据"，属缺陷修复方向的单向改善；既有测试用真缓存/mock 不受影响。
**Warning signs:** 无（这是改善）；但要防止有人把它当回归"修回去"。

### Pitfall 6: notice 匿名 struct 缓存值的 T 形态
**What goes wrong:** `notice.GetUserNotices` 缓存 `struct{List []models.Notice; Total int64}` 匿名类型，迁移时若选 `T = 匿名 struct` 语法可行（本机编译实证：匿名 struct 作类型实参 OK）但闭包签名冗长难读。
**How to avoid:** 就地声明具名局部类型（`type noticeListPage struct{...}`）或包级私有类型，T 用具名类型；JSON 序列化形状不变（字段名一致），缓存数据兼容。

## Code Examples

### 现状五段式样板（32 处的统一形状，实测 user_cache_impl.go:38-52）
```go
// Source: internal/services/system/user_cache_impl.go:38-52
cacheKey := GetUserByIDKey(id)
var result models.User

expiration := s.GetExpiration(services.CacheConfigUserByID, 30*time.Minute)

err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
    return s.userService.GetByID(ctx, id)
})

if err != nil {
    return nil, err
}
return &result, nil
```

### TTLResolver 隐式满足（D-01）
```go
// Source: internal/services/cache_config_service.go:392（签名实测）+ system/cache_utils.go:50-61
// ---- base/cache_service_base.go ----
type TTLResolver interface {
    GetDurationWithDefault(configKey string, defaultDuration time.Duration) time.Duration
}

type CacheServiceBase struct {
    Config TTLResolver // 原 *services.CacheConfigService；*CacheConfigService 隐式满足
}

func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration {
    if b.Config != nil {
        return b.Config.GetDurationWithDefault(configKey, defaultVal)
    }
    return defaultVal
}
// ⚠️ 配套：GetDurationWithDefault 需加 nil-receiver 防护（Pitfall 1）
```

### 泛型方法非法性 + alias/匿名 struct 实证（本机 go1.24.5 编译实验）
```go
// Source: 本机编译实验 2026-09-05（VERIFIED: local Go toolchain go1.24.5）
func (b *CacheServiceBase) Get[T any](key string) (T, error) { ... }
// → syntax error: method must have no type parameters   ✗ 实测报错

func GetOrSetJSON[T any](p CacheProvider, key string, ttl time.Duration,
    query func() (T, error)) (T, error) { ... }          // ✓ 包级泛型函数编译通过

GetOrSetJSON(p, "k", ttl, func() (struct {               // ✓ 匿名 struct 作类型实参编译通过
    List  []models.Notice
    Total int64
}, error) { ... })

type CacheProvider = base.CacheProvider                  // ✓ 接口 alias 编译通过
```

### miniredis 双装配测试模板（D-09 直接复用）
```go
// Source: internal/services/data_cache_service_79_01_test.go:58-70（v1.27 已验证模式）
func newBase92Redis(t *testing.T) (base.CacheProvider, *miniredis.Miniredis) {
    t.Helper()
    mr := miniredis.RunT(t)
    host, portStr, err := net.SplitHostPort(mr.Addr())
    require.NoError(t, err)
    port, _ := strconv.Atoi(portStr)
    rc, err := cache.NewRedisCache(&cache.CacheConfig{Host: host, Port: port}, "")
    require.NoError(t, err)
    t.Cleanup(func() { _ = rc.Close() })
    return system.NewCacheProvider(NewDataCacheService(rc)), mr
}
// 断言面：命中/穿透（query 调用计数）、mr.FastForward(ttl) 后穿透、
// Invalidate/InvalidatePattern 后穿透、NoOp 透传（query 必调 + 不写缓存）、GetStats 可读
// 纪律：禁 t.Parallel()（装配含后台 goroutine 与 miniredis 实例）
```

### AST invariants 扫描先例（D-10②）
```go
// Source: pkg/constants/pagination_test.go:22-56（Phase 89 模式，Stability + Count 双锁）
// Phase 92 变体：扫描 system/operations 的 *_cache_impl.go，
// 检测 `.GetOrSet(ctx` 且闭包签名为 `func() (interface{}, error)` 的残留 →
// 期望 0 处；warning 不 fail（t.Logf + 宽松断言或独立 warning test）；
// 白名单机制：显式 map[文件]允许数（初始全 0，未来豁免显式登记）
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| interface{} 闭包 + var 中转 + 反射兜底（Go pre-generics 惯用） | 泛型包级函数 + 类型实参 | Go 1.18 (2022-03) | 本仓 go1.24.5 全面可用；D-03 即此范式落地 |
| 命名接口逐文件改类型 | type alias 翻转（零引用面改动） | Go 1.9+ (2016) | D-02 机制；alias = 同一类型，mock 断言/嵌入字段全部兼容 |
| legacy root 6 件套 cache service | system/ 唯一定义（Phase 79-01 已迁） | v1.27 Phase 79 | CONTEXT 现实校准 1 的由来；CLAUDE.md 文档滞后（D-10 修订对象） |

**Deprecated/outdated:**
- CLAUDE.md § Key Architecture Patterns "Dual Cache Architecture" 段（root 6 件套描述失实）→ D-10① 修订
- CLAUDE.md § Key File Locations "Legacy Services (still used by core)" 列表（dept/role/dict/menu/user/post root 文件已不存在）→ D-10① 一并修订
- CLAUDE.md § Cache System 中 "Legacy" 提法 → 随 D-10① 修订
- ROADMAP Phase 92 SC-1/SC-2/SC-3/SC-5 措辞 → 按 D-03/D-06/D-09 修订（Phase 90 commit 3a2efe5 先例）

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `monitor.CacheProvider` rename 目标名用 `CacheOperator`（D-08 明示为建议非锁定） | Architecture Patterns / D-08 | 低——纯命名，编译器保证完整性 |
| A2 | LOC 锚点口径 = "32 处样板调用段毛减" 单列 + base 投资单列（Phase 91 OVR 先例）；若 planner 坚持 repo 全口径净减 ≥200，需要执行期紧凑形态 + 接受可能 shortfall 诚实记录 | Pitfall 2 | 中——验收判定分歧；建议 discuss/planner 明确 |
| A3 | `DataCacheService.GetExpiration` 的 D-06 处置 = 保留方法签名、内部委托 base TTL 逻辑（而非删除方法）；2 个测试文件（data_cache_service_79_01_test.go:351-359、cache_config_service_79_01_test.go:225-237）锁定现行为，委托实现可直接保持绿 | CACHE-UNIFY-04 | 低——删除亦可但需改 2 个测试；委托更平滑 |
| A4 | nil-receiver 防护加在 `CacheConfigService.GetDurationWithDefault`（root 包一行）是 D-01 的必要配套（Pitfall 1）；属"消除平行 TTL 逻辑"语义内的防御性修改，不违背零行为变更 | Pitfall 1 | 中——若不加，迁移后 nil-config 测试路径 panic |
| A5 | duty/knowledge/network/workorder 12 处 GetOrSet 样板（与 32 处同构）**不迁**，留 v1.30+（CONTEXT 范围锁定）；仅其 19 处失效调用因 D-04 编译器强制改写 | Pitfall 4 / Deferred | 低 |
| A6 | `api/v1/system/cache_adapter.go`（NewDataCacheAdapter，生产 0 调用者）按 scope constrainment 原则本期不动，仅记录 | 发现清单 | 低——死代码遗留，可入 v1.30 清理候选 |
| A7 | `SetJSON[T]` 在 D-03 锁定的函数族内，但实测当前 32 处调用点**零个**直接 Set 使用者（无 s.cache.Set 直调）——实现为薄包装（~10 行）以兑现 D-03，无迁移对象 | Standard Stack / Pattern 1 | 低——YAGNI 但已被 D-03 锁定 |

## Open Questions

1. **LOC 锚点统计口径（建议 planner 在 plan 里显式锁定）**
   - What we know: 32 处毛减 ~190-290 行（实测）；base 新增 ~100-160 行生产代码；repo 全口径净减可能落在 ~30-190 区间。
   - What's unclear: D-05"净减 ≥200"是否把 base 包新增计入净额。
   - Recommendation: 沿 Phase 91 OVR-91-01 双口径（样板毛减 + base 投资分列），SUMMARY 诚实双报告；若坚持全口径，需用户 pre-confirm 接受 shortfall 可能。

2. **D-04 涟漪 19 处的 plan 归属**
   - What we know: duty 5 / knowledge 4 / network 5 / workorder 4 / core.go 1，机械 1 行/处。
   - What's unclear: 放 92-01（与删除同 plan，编译器驱动）还是 92-03（operations+root+外围）。
   - Recommendation: 放 92-01 与"迁 base 后删除"同 commit——编译器保证零遗漏，符合 D-04 原子性。

3. **monitor rename 的最终名**
   - What we know: 候选 CacheOperator（D-08 建议）；接口 8 方法 Get/Set/Delete/Exists/Expire/TTL/Keys/FlushDB，rename 面 3 文件 ~58 处引用（cache_service.go 8 + cache_router.go 27 + cache_router_test.go 23）。
   - Recommendation: `CacheOperator`（动宾语义贴合"原始操作型"）；同文件 CacheConfigProvider/MultiLevelCacheProvider/DirectRedisProvider/StatsProvider 不撞名不动。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 全部 | ✓ | go1.24.5（toolchain） | — |
| miniredis/v2 | D-09 集成测试 | ✓ (go.mod test-only) | v2.38.0 | pkg/cache MemoryCache（cache_infra_test.go 模式） |
| go-redis/v9 | miniredis 装配 | ✓ | v9.7.0 | — |
| glebarez/sqlite | 既有 impl 测试回归网 | ✓ | (go.mod) | — |
| testify | 全部测试 | ✓ | (go.mod) | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test（标准）+ testify + miniredis/v2.38.0 + glebarez/sqlite |
| Config file | none（go.mod 驱动；无独立 test config） |
| Quick run command | `go build ./... && go test ./internal/services/base/... ./internal/services/system/... ./internal/services/operations/...` |
| Full suite command | `go test ./internal/services/...`（SC-4）/ 全量 `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CACHE-UNIFY-01 | base 泛型函数族 + TTLResolver 契约（含 nil-config → default 分支） | unit（fake provider 纯函数） | `go test ./internal/services/base/ -run TestCache -v` | ❌ Wave 0（base/cache_service_base_test.go） |
| CACHE-UNIFY-02 | system 9 文件 29 处迁移后行为不变 | regression（既有测试保持绿，改动最小化） | `go test ./internal/services/system/ -run "Cache" -v` | ✅（config/department/dict/menu/post_cache_impl_test.go + user_cache_impl_test.go + gapfill 系列） |
| CACHE-UNIFY-03 | floor 3 处 + CacheInvalidator 委托后分发语义不变 | regression + unit | `go test ./internal/services/operations/ -run "Cache\|Floor" -v` | ✅（cache_invalidator_test.go + geocoding_photo_floor_test.go:379） |
| CACHE-UNIFY-04 | DataCacheService 原地定性 + GetExpiration 委托后行为不变 | regression | `go test ./internal/services/ -run "Dcs7901\|Ccs7901" -v` | ✅（data_cache_service_79_01_test.go + cache_config_service_79_01_test.go 锁定行为） |
| CACHE-UNIFY-05 | miniredis 集成：命中/穿透/TTL 过期/pattern+key 失效/NoOp 透传/CacheStats | integration（miniredis） | `go test ./internal/services/base/ -v`（双装配：MemoryCache + miniredis） | ❌ Wave 0（同 CACHE-UNIFY-01 文件） |
| D-10② | invariants：system/operations `*_cache_impl.go` 无 interface{} 闭包式 GetOrSet 残留 | AST/regex 扫描（warning 不 fail） | `go test ./internal/services/system/ -run TestNoInterfaceGetOrSetResidue -v` | ❌ Wave 0 |
| SC-4 | 全 services 回归 0 失败（1688+ 测试） | full regression | `go test ./internal/services/...` | ✅ 既有 |

### Sampling Rate
- **Per task commit:** `go build ./...`（CLAUDE.md 强制）+ quick 命令（base + system + operations 三包）
- **Per wave merge:** `go test ./internal/services/...`（SC-4 口径）
- **Phase gate:** 全量 `go test ./...` 绿 + operlog/status AST 锁值防线保持绿 + LOC numstat 双口径报告 + invariants 扫描输出（残留 = 0）

### Wave 0 Gaps
- [ ] `internal/services/base/cache_service_base_test.go` — 覆盖 CACHE-UNIFY-01/05：泛型函数族 hit/miss/TTL(FastForward)/失效/NoOp 透传/GetStats + nil-config 分支（REQUIREMENTS 原文命名的文件）
- [ ] （可选并入上文件）`base` 包 alias 翻转后 `var _` 编译期断言
- [ ] invariants 扫描测试（system 或 pkg 侧，warning-only，Phase 89/90 模式）
- [ ] Framework install: 无需（全部既有）

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本 phase 无认证面改动（JWT/SM3 链路不触碰） |
| V3 Session Management | no | 无 |
| V4 Access Control | no | 权限中间件链不触碰 |
| V5 Input Validation | no | 无新输入路径（泛型函数族为内部重构；缓存键构造逐字保留） |
| V6 Cryptography | no | 国密链路（SM2/SM4）不触碰；缓存值仍为 JSON 明文入 Redis（现状语义，不变更） |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 缓存键注入/串缓存（key 构造变化引入） | Tampering | 本 phase 键构造逐字不变（CONTEXT 锁定）——迁移 PR review 时对 key 拼接做 diff 级零变更检查 |
| 缓存失效遗漏（失效路径重构引入陈旧数据窗口） | Tampering | D-04 统一底层后行为与现状逐字对齐（warn 日志格式保留）；既有 invalidation 测试回归 |
| 无 | — | 本 phase 无新增攻击面 |

## Sources

### Primary (HIGH confidence)
- 本仓实测（2026-09-05）：`grep`/`Read` 逐文件——32 处 GetOrSet 调用点 file:line 枚举、43 处失效辅助调用面、4 个 CacheProvider 实现清单、import 方向实证（root→base 18 文件 / root↛system）
- 本机编译实验（go1.24.5）：泛型 method 非法性（`method must have no type parameters`）、包级泛型函数、匿名 struct 类型实参、接口 type alias
- `internal/services/` 全部关键源文件通读：cache_utils.go / cache_provider.go / adapter.go / cache_adapter.go / cache_keys.go(索引) / cache_manager.go(索引) / user/menu/notice/role/floor cache_impl.go / cache_invalidator.go / data_cache_service.go / cache_config_service.go:385-400 / monitor/cache_service.go:40-185 / api/v1/monitor/cache_router.go / base/service.go
- 既有测试基建通读：data_cache_service_79_01_test.go（miniredis 双装配模板）、cache_infra_test.go（MemoryCache 模式）、pkg/constants/pagination_test.go（AST 锁值先例）
- 规划文档：92-CONTEXT.md（D-01..D-10）、REQUIREMENTS.md § CACHE-UNIFY、ROADMAP.md § Phase 92、STATE.md、PROJECT.md v1.29 段、.planning/config.json（nyquist_validation: true）

### Secondary (MEDIUM confidence)
- go.mod 版本清单（go 1.24.0 / toolchain go1.24.5 / miniredis v2.38.0 / go-redis v9.7.0）——go.mod 直读

### Tertiary (LOW confidence)
- 无（本 phase 全部结论可本仓验证，未依赖外部网络源；go.dev 因网络策略不可达，改用本机编译实证，证据强度更高）

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新依赖；Go 1.24 泛型/alias 语言特性经本机编译实证
- Architecture: HIGH — base 包方向、import 拓扑、alias 消费者清单全部 grep/read 实测；迁移形态有 32 处逐点枚举支撑
- Pitfalls: HIGH — 6 个 pitfall 均有具体 file:line 或编译实验证据（typed-nil panic 路径实测到 `s.mu.RLock()` 源码行）

**Research date:** 2026-09-05
**Valid until:** 2026-10-05（稳定域——纯内部重构，无外部 API 面；代码基线若被 Phase 93/94 并行改动需复核行号）

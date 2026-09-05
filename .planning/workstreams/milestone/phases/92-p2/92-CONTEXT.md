# Phase 92: 缓存层三处架构统一 (🟡 中优 P2) - Context

**Gathered:** 2026-09-05
**Status:** Ready for planning

<domain>
## Phase Boundary

把缓存层统一到 `internal/services/base` 单一权威抽象：`CacheServiceBase` 基类 + `CacheProvider` 接口全家迁入 base 包（依赖倒置保持 base 零依赖），新增**泛型包级函数族**（`GetOrSetJSON[T]` / `SetJSON[T]` / `Invalidate` / `InvalidatePattern`）消灭 system/ + operations/ 下 **33 处** GetOrSet interface{} 闭包式样板（每处 12-15 行 → ~5 行，类型安全无反射兜底）；失效辅助归一到 base 唯一底层（CacheInvalidator 委托）；`DataCacheService` 原地定性为基础设施（消除平行 GetExpiration，不迁移）；monitor 同名 `CacheProvider` rename 消歧；miniredis 自动化验证 + CLAUDE.md 段修订 + invariants 锁值。`go test ./internal/services/...` 0 失败，零业务行为变更。

**关键现实校准（2026-09-05 侦察确认，与 ROADMAP 2026-09-03 审计基线有重大偏差，以本 CONTEXT 为准）：**

1. **"三处独立 CacheServiceBase 定义"已不存在** — v1.27 Phase 79-01 已把 legacy root 6 件套（dept/role/dict/menu/user/post cache service）迁入 `system/`；现状唯一定义在 `system/cache_utils.go`（79 行），system 8/9 个 impl 嵌入它，`operations/floor_cache_impl.go` 也嵌入它（跨包复用）
2. **真正残留的重复是 33 处 GetOrSet 样板**（system 29 处 + operations 4 处，`grep -c GetOrSet` 实测）——这才是 CACHE-UNIFY-01 "模板方法"的真实标的
3. **legacy root 只剩 `data_cache_service.go` 一个文件**（257 行）——CACHE-UNIFY-04 的"迁移到 system/ 下"存在 **root↔system import cycle 硬约束**（mac_history_heatmap/query_service 在 root 同包引用 DataCacheService，而 system 包已 import services root），字面迁移不可行（D-06）
4. **notice_cache_impl.go 是 system 9 个 impl 中唯一未嵌入基类的逃兵**（私有 `getExpiration` 平行实现）——D-03 泛型函数族迁移时自然消灭
5. **CacheProvider interface 存在同名双定义**：`system/cache_provider.go`（GetOrSet 业务型，9 方法）vs `monitor/cache_service.go:49`（Get/Set/Keys/FlushDB 原始监控操作型）——形状不同非重复实现，但 D-02 定名 base.CacheProvider 为唯一权威后必须 rename 消歧（D-08）
6. **失效辅助两套**：`cache_utils.go` 的 `InvalidateCacheByPattern/ByKey` 包级函数 vs `operations/cache_invalidator.go` 的 `CacheInvalidator` struct（含 Excel 管道 entityType 分发语义）——统一底层但保留分发器（D-04）
7. CLAUDE.md "Dual Cache Architecture" 段描述的 root 6 件套已不存在（文档过时，D-10 修订）

**不在本 phase：**
- config_backup 三处 TODO 闭环（Phase 93）
- 前端 API 工厂化（Phase 94）
- monitor CacheProvider 的语义改造/形状统一（仅 rename 消歧，D-08）
- DataCacheService 文件迁移 / mac_history 服务接口化改造（D-06 明确排除；mac_history 改吃 CacheProvider 接口属 v1.30+ 候选）
- CacheInvalidator struct 删除 / excel 管道重构（D-04 保留委托）
- `internal/services/cache_config_service.go`（TTL 配置真相源）的迁移或重构——保持 services root 原位
- `mac_history_cache_decorator.go` / `template_cache.go`（root 的另两个缓存文件，无 CacheServiceBase 模式，不在审计范围）
- Redis 缓存键规范（`system/cache_keys.go` 367 行单一真相源，P2-A2 已治理，不动）

</domain>

<decisions>
## Implementation Decisions

### 基类归属与依赖方向 (Area 1)

- **D-01: 迁 base + 依赖倒置** — 新建 `internal/services/base/cache_service_base.go`，基类对 TTL 配置的依赖改为 base 内定义的小接口（TTLResolver 风格：`GetDurationWithDefault(key string, default time.Duration) time.Duration`，`services.CacheConfigService` 隐式满足）。base 包保持零依赖（Go consumer-defined interface 惯例，与 Phase 91 base 包"纯抽象层"定位一致）。ROADMAP CACHE-UNIFY-01 原文达成。
- **D-02: CacheProvider 全家搬 base + alias** — `CacheProvider` 接口（9 方法：GetOrSet/Delete/DeleteByPattern/MGet/MDelete/Exists/SetTTL/GetTTL/GetStats）+ `NoOpCacheProvider` + `CacheStats`/`CacheEntry` 伴生类型一起迁 base。`system/cache_provider.go` 留 `type CacheProvider = base.CacheProvider` 别名，10 个嵌入文件的字段引用零改动（Go type alias 完美兼容，alias 后续可删）。缓存抽象全套归 base，语义完整。

### GetOrSet 模板方法形状 (Area 2)

- **D-03: 泛型包级函数族** — base 包提供 `GetOrSetJSON[T any]` / `SetJSON[T]` / `Invalidate` / `InvalidatePattern` 函数族。每处样板 12-15 行 → ~5 行单 return（类型安全：无 interface{} 闭包、无 `var result` 中转、无 `NoOpCacheProvider.setValue` 反射兜底）。**技术约束依据：Go method 不能有自己的类型参数**（`func (b *CacheServiceBase) Get[T]()` 非法），泛型 struct 单 T 也不匹配现实（一个 cacheImpl 有多种缓存值类型，如 userCacheService 同时缓存 `*User`/`[]User`）——泛型包级函数是唯一全类型安全路径。基类退化为薄持有 TTLResolver。与 Phase 91 base 包函数式风格一致。
- **D-04: base 唯一失效底层 + Invalidator 委托** — base 提供唯一失效底层实现（nil 防护 + 统一日志）；`cache_utils.go` 的 `InvalidateCacheByPattern`/`InvalidateCacheByKey` 迁 base 后删除；`operations/CacheInvalidator` **保留** struct（Excel 导入管道的 entityType 分发 + ExcelConfig.CachePatterns 读取语义，excel_service 在用）但底层循环改为委托 base。两套重复逻辑归一，分发语义保留。
- **D-05: 验收锚点 = 定性清零 + LOC ≥200** — 定性底线：33 处 GetOrSet 样板全部走 `base.GetOrSetJSON`，每处 WithCache 方法体 ≤6 行，仓内不再有 interface{} 闭包式 GetOrSet 样板。量化锚点：LOC 净减 ≥200（实测 33 处 × 每处省 6-9 行 ≈ 200-300 诚实区间；Phase 91 D-07 混合标准先例）。

### DataCacheService 处置 (Area 3)

- **D-06: 原地定性为基础设施** — `DataCacheService` 定性为 CacheProvider 实现底座（`system.NewCacheProvider()` 适配器的 Adaptee + `pkg/cache.Cache` 的业务封装层），永久留 services root。本期消除其平行 `GetExpiration`（委托 base 统一 TTL 逻辑；root import base 无循环）。**CACHE-UNIFY-04 措辞同步修订**（REQUIREMENTS + ROADMAP，Phase 90 先例 commit 3a2efe5）：删"迁移到 system/ 下"，改为"原地定性 + 消除平行 TTL 逻辑"。理由：root↔system import cycle 硬约束使字面迁移不可行；12+ API 文件 import 风暴与收益不成比。
- **D-07: 不标 @Deprecated，定位注释** — 文件头注释明确双定位：① `pkg/cache.Cache` 的业务封装层（GetOrSet/JSON 序列化/统计）② `base.CacheProvider` 接口的实现底座（经 `system.NewCacheProvider` 适配）。新增业务缓存代码应走 `base.GetOrSetJSON` + `CacheProvider`，不应直接新增 DataCacheService 调用点。

### 范围边界与验证锚点 (Area 4)

- **D-08: monitor 同名接口纳入本期，仅 rename 消歧** — `monitor/cache_service.go:49` 的 `CacheProvider`（Get/Set/Keys/FlushDB 原始监控操作型）rename 为不撞名的名字（如 `CacheOperator`，最终名 planner 定）。纯包内 rename + `api/v1/monitor/cache_router.go` adapter 同步，语义零变更。D-02 定名 base.CacheProvider 为唯一权威后，同名异形接口必须消除，否则认知混乱。
- **D-09: SC-5 验证 = miniredis 自动化** — 用 v1.27 INFRA-01 已引入的 miniredis/v2 写自动化集成测试：GetOrSetJSON 读写命中/穿透、TTL 过期、pattern/key 失效、NoOp 透传、CacheStats 读取。可回归、进 CI gate（Phase 91 handler smoke 同款思路）；"端到端"语义落地为 provider 层集成测试。手动冒烟不做强制要求。
- **D-10: CLAUDE.md + invariants 都做** — ① 收口时修订 CLAUDE.md 过时的 "Dual Cache Architecture" 段（root 6 件套描述已失实），新增 Cache Service Convention 段（锁 `base.GetOrSetJSON`/`CacheProvider` 单一权威路径，Phase 90 D-11 先例格式）；② 新增 invariants 扫描锁 "cache_impl 无 interface{} 闭包式 GetOrSet 残留"（warning 不 fail，Phase 89/90 模式）。

### Claude's Discretion

以下细节由 planner/researcher 决定，无需再问用户：
- TTL 参数在 `GetOrSetJSON` 签名中的形态（预解析 Duration 传入 vs configKey+default+resolver 三参传入）
- `system.CacheServiceBase` 旧名是否同样留 type alias（与 D-02 alias 策略对齐，一次改干净 vs 渐进）
- base 包内文件布局（cache_service_base.go / cache_provider.go / cache_functions.go 拆分粒度）
- 泛型函数的命名最终形态（GetOrSetJSON vs GetOrSetCached vs 其他）
- `monitor.CacheProvider` rename 的目标名（CacheOperator 为建议非锁定）
- 33 处迁移在 3 个 plan 中的分组（ROADMAP 建议 92-01 base 抽取 / 92-02 system 9 文件 / 92-03 operations+root，planner 可按实际微调）
- 错误包装与日志格式的保留/统一方式
- invariants 扫描的具体检测模式（AST 或正则，参考 Phase 89/90 白名单机制）
- `Invalidate`/`InvalidatePattern` 的签名细节（返回 error 还是 void+warn 日志，对齐现状语义）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### v1.29 milestone 全局
- `.planning/REQUIREMENTS.md` § CACHE-UNIFY（:68-76）— CACHE-UNIFY-01..05 需求原文（**04 的"迁移到 system/ 下"措辞按 D-06 修订为"原地定性 + 消除平行 TTL 逻辑"**）
- `.planning/workstreams/milestone/ROADMAP.md` § Phase 92 — Goal / 3 plans 拆分 / 5 Success Criteria（**SC-1 按 D-03 修订为泛型函数族、SC-3 按 D-06 修订**）
- `.planning/PROJECT.md` — v1.29 Current Milestone 段，D-01..D-06 locked decisions（D-03 零回归底线 / D-04 atomic commit + 逐步迁移纪律）
- `.planning/workstreams/milestone/phases/91-crud-base-repository-t-p1/91-CONTEXT.md` — Phase 91 决策先例：D-02 纯 struct 组合、D-07 混合验收标准、base 包"零依赖纯抽象层"定位

### 本期改造核心代码（三处架构现址）
- `internal/services/system/cache_utils.go` — 现有 `CacheServiceBase`（79 行：`{Config}` + `GetExpiration` + `InvalidateCacheByPattern/ByKey`）→ D-01/D-04 迁移源
- `internal/services/system/cache_provider.go` — 现有 `CacheProvider`（9 方法）+ `NoOpCacheProvider`（含反射 setValue 兜底）+ `CacheStats`/`CacheEntry` → D-02 迁移源 + alias 落点
- `internal/services/system/adapter.go` — `NewCacheProvider(dataCache *services.DataCacheService)` 适配器（85 行）→ D-06 Adaptee 模式依据
- `internal/services/base/service.go` — Phase 91 base 包现有内容（`GORMRepository[T]` + `Scope` + `PageResult`）→ D-01/D-02 新文件同居处，零依赖基线参照
- `internal/services/data_cache_service.go` — `DataCacheService`（257 行）→ D-06/D-07 定位注释 + 平行 GetExpiration 消除对象

### 33 处样板迁移目标（GetOrSet 实测计数）
- `internal/services/system/user_cache_impl.go` — 5 处 GetOrSet；嵌入模式样本（`userCacheService{userService, cache, CacheServiceBase}`）
- `internal/services/system/menu_cache_impl.go` — 6 处（最多）+ `internal/services/system/role_cache_impl.go` — 6 处
- `internal/services/system/{config,department,dict,notice,post,settings}_cache_impl.go` — 各 1-3 处；dict 含双 struct（dictTypeCacheService + dictDataCacheService）
- `internal/services/system/notice_cache_impl.go:61` — 逃兵样本：私有 `getExpiration` 未嵌入基类（D-03 迁移时自然消灭）
- `internal/services/operations/floor_cache_impl.go` — 4 处；跨包嵌入 `system.CacheServiceBase` 样本（186 行）
- `internal/services/operations/cache_invalidator.go` — `CacheInvalidator` struct（68 行，D-04 委托改造对象；excel_service 消费其 entityType 分发）

### 消歧与验证
- `internal/services/monitor/cache_service.go:49` — 同名 `CacheProvider`（Get/Set 原始型）→ D-08 rename 对象
- `internal/api/v1/monitor/cache_router.go:42` — `NewCacheProviderAdapter` → D-08 rename 同步点
- `internal/core/core.go:356-366,900,958` + `internal/core/core_services.go:33` — DataCacheService 装配链 + `NewCacheProvider` 适配消费点（D-06 不动，但回归验证范围）
- v1.27 miniredis 基建先例（D-09 依据）— INFRA-01 引入的 miniredis/v2 用法可参考 `pkg/cache` 现有测试

### 项目级约束
- `CLAUDE.md` § Cache System + Dual Cache Architecture — 过时描述（D-10 修订对象）
- `CLAUDE.md` § Compilation & Build Verification — `go build ./...` 0 错误
- `CLAUDE.md` § Testing — `go test ./internal/services/...` 0 失败（SC-4；1688+ 测试不回归，v1.29 D-03）
- `internal/models/status_constants_test.go` / `internal/utils/operlog/regression_test.go` — AST 锁值防线全程保持绿
- Phase 89/90 invariants 扫描先例 — `pkg/constants/*_test.go`（D-10② 模式参考）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `system/cache_utils.go` 的 `CacheServiceBase` — 已是 10 文件嵌入源，迁移而非重写（79 行小体量）
- `system/cache_provider.go` 全套（接口 + NoOp + Stats）— D-02 整体搬迁对象，NoOp 的反射 setValue 缺陷由 D-03 泛型函数族绕开
- `system/cache_keys.go`（367 行）— 缓存键单一真相源（P2-A2 已治理），33 处迁移时键构造不变
- `internal/services/base` 包 — Phase 91 新建，零依赖（gorm + pkg/errors only），D-01/D-02 新文件同居处；`service_test.go` 泛型契约锁定测试模式可复制
- v1.27 miniredis/v2 测试基建 — D-09 自动化验证底座
- Phase 89/90/91 收口惯例 — CLAUDE.md Convention 段 + REQUIREMENTS 措辞同步（3a2efe5）+ invariants 扫描

### Established Patterns
- **cache_impl 装饰器模式**：`type xxxCacheService struct { *xxxService; cache CacheProvider; CacheServiceBase }` — 基础服务 + 缓存包装，`NewXxxServiceWithCache()` 构造。D-03 迁移后装饰器结构不变，仅方法体内部换 `base.GetOrSetJSON` 单 return
- **GetOrSet 样板五段式**（33 处）：cacheKey 构造 → var result 声明 → GetExpiration → interface{} 闭包 GetOrSet → 错误/结果搬运 — D-03 消灭对象
- **Adaptee 适配模式**：`DataCacheService`（root）→ `system.NewCacheProvider()` → `base.CacheProvider`（D-06 定性依据）
- **AST 锁值 + invariants 扫描**（Phase 89/90）— D-10② 复用
- **type alias 渐进迁移**（D-02）— Go 惯例，引用点零改动

### Integration Points
- `core.Core`（core.go:356-366, 900, 958）— DataCacheService 装配 + 缓存预热失效 + menu service 注入 CacheProvider — D-06 不动但须回归
- 12+ API 文件引用 `services.DataCacheService`（router.go / duty / knowledge / monitor / network / system handlers）— D-06 零改动保证
- `excel_service.go` → `CacheInvalidator` → ExcelConfig.CachePatterns — D-04 委托改造的下游
- `internal/api/v1/monitor/cache_router.go` — D-08 rename 同步点

</code_context>

<specifics>
## Specific Ideas

### 迁移后样板形态（D-03 预览，用户确认）
```go
// 迁移后（12-15 行 → 5 行）
func (s *userCacheService) GetByIDWithCache(ctx context.Context, id string) (*models.User, error) {
    return base.GetOrSetJSON(ctx, s.cache,
        GetUserByIDKey(id),
        s.TTL(services.CacheConfigUserByID, 30*time.Minute),
        func() (*models.User, error) {
            return s.userService.GetByID(ctx, id)
        })
}
```

### import cycle 硬约束（D-06 依据，planner 必须知道）
```
data_cache_service.go 迁 system/ 的死路：
  root 包 mac_history_heatmap/query_service（同包引用 DataCacheService）
    → 需 import system → 而 system 已 import services root（cache_utils.go:8）
      → services → system → services 循环导入，编译失败
```

### 现实校准必须体现在 plan 里（用户确认）
- ROADMAP CACHE-UNIFY-01 "完整模板方法（Get/Set/Delete/Invalidate/InvalidatePattern）" → 实际形态是**泛型包级函数族**（Go method 无类型参数约束），SC-1 措辞按 D-03 修订
- ROADMAP CACHE-UNIFY-02 "删除重复的 5 行模板代码" → 实测 33 处 × 每处省 6-9 行，锚点 ≥200（D-05）
- ROADMAP CACHE-UNIFY-04 "迁移到 system/ 下，core.Core 引用路径同步更新" → import cycle 不可行，原地定性 + 措辞修订（D-06）
- ROADMAP Success Criteria 2 "system/ + operations/ 下所有 *_cache_impl.go 继承新基类" → 实际已嵌入 system.CacheServiceBase，本期动作是**嵌入源迁 base + 方法体换泛型函数**（含 notice 逃兵归队）

### 估算依据（供 planner 参考）
- 33 处样板（system 29 + operations 4）× 每处省 6-9 行 ≈ LOC 净减 200-300（D-05 锚点 ≥200）
- 迁移总量：base 新建 3-4 文件（基类/接口/函数族/测试）+ system 10 文件改造 + operations 2 文件 + root 1 文件（定位注释）+ monitor rename 2 文件
- 每处迁移是机械替换（键构造/闭包逻辑不变），适合 92-02/92-03 批量复制模式（Phase 91 pilot→批量节奏先例）

</specifics>

<deferred>
## Deferred Ideas

### 范围外但相关的后续候选（v1.30+）
- **DataCacheService 字面迁移 + mac_history 接口化改造** — mac_history_heatmap/query_service 改吃 `base.CacheProvider` 接口（core 注入适配器解耦具体类型）后 cycle 解除，届时可评估迁移；本期 D-06 明确不做
- **CacheInvalidator struct 消除 / excel 管道重构** — 若未来 ExcelConfig.CachePatterns 机制重构，CacheInvalidator 可彻底并入 base；本期 D-04 保留委托
- **monitor 缓存监控接口语义统一** — D-08 仅 rename 消歧；若未来监控页需要 GetOrSet 型能力再评估
- **`cache_config_service.go`（TTL 配置真相源）重构/迁移** — 本期不动，仅作为 TTLResolver 的隐式实现
- **`mac_history_cache_decorator.go` / `template_cache.go` 统一评估** — root 另两个缓存文件，无 CacheServiceBase 模式，若审计口径扩大再议

None — discussion stayed within phase scope（无 scope creep 提案）

</deferred>

---

*Phase: 92-缓存层三处架构统一*
*Context gathered: 2026-09-05*

# Phase 92: 缓存层三处架构统一 (🟡 中优 P2) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-05
**Phase:** 92-缓存层三处架构统一
**Areas discussed:** 基类归属与依赖方向, GetOrSet 模板方法形状, DataCacheService 处置, 范围边界与验证锚点

---

## 灰区选择

用户选择讨论全部 4 个灰区（multiSelect：基类归属、模板方法形状、DataCacheService 处置、范围边界与验证锚点）。

**开场的现实校准（Claude 侦察发现，用户接受为讨论基础）：**
- ROADMAP 2026-09-03 审计说的"三处独立 CacheServiceBase 定义"已不存在（v1.27 Phase 79-01 已迁 legacy root 6 件套入 system/；operations/floor 嵌入的是 system 的基类）
- 真正残留：33 处 GetOrSet interface{} 闭包样板 / notice 逃兵 / DataCacheService 平行 GetExpiration / CacheProvider 同名双定义 / 失效辅助两套 / CLAUDE.md 过时

## Area 1: 基类归属与依赖方向

### Q1 基类放哪 + 依赖处理

| Option | Description | Selected |
|--------|-------------|----------|
| 迁 base + 依赖倒置 | base/cache_service_base.go，TTL 依赖改为 base 内 TTLResolver 小接口（CacheConfigService 隐式满足），base 保持零依赖 | ✓ |
| 迁 base + 直接依赖 | base import services root 引用 CacheConfigService 具体类型，省事但架构倒挂 | |
| 留 system/ 原地增强 | 承认 cache_utils.go 就是单一基类，修订 CACHE-UNIFY-01 措辞，零迁移成本 | |

**User's choice:** 迁 base + 依赖倒置
**Notes:** consumer-defined interface 是 Go 惯例，与 Phase 91 base 包"纯抽象层"定位一致，ROADMAP CACHE-UNIFY-01 原文达成。

### Q2 CacheProvider 接口是否随迁

| Option | Description | Selected |
|--------|-------------|----------|
| 全家搬 base + alias | CacheProvider + NoOp + CacheStats/CacheEntry 一起迁，system 留 type alias，10 个嵌入文件零改动 | ✓ |
| 留 system，base 用最小接口 | 基类迁走但接口留 system，base 函数用方法子集小接口，引入第二套接口认知成本 | |

**User's choice:** 全家搬 base + alias
**Notes:** 缓存抽象全套归 base 语义最完整；Go type alias 完美兼容，alias 后续可删。

## Area 2: GetOrSet 模板方法形状

**Claude 摆出的关键技术约束（用户接受）：** Go method 不能有自己的类型参数（`func (b *CacheServiceBase) Get[T]()` 非法）；泛型 struct 单 T 不匹配现实（一个 cacheImpl 多种缓存值类型）——泛型包级函数是唯一全类型安全路径。

### Q1 模板方法形态

| Option | Description | Selected |
|--------|-------------|----------|
| 泛型包级函数族 | base.GetOrSetJSON[T]/SetJSON[T]/Invalidate/InvalidatePattern，每处样板 12-15 行 → 5 行单 return | ✓ |
| 基类 interface{} 方法 | SC-1 字面"模板方法"释义，但保留 interface{} 闭包 + var 中转 + 反射兜底 | |
| 函数族 + 基类方法并存 | 两套路径并存，正是本期要消灭的架构病 | |

**User's choice:** 泛型包级函数族
**Notes:** 预览代码确认迁移形态（单 return + 类型安全闭包）。

### Q2 失效辅助统一

| Option | Description | Selected |
|--------|-------------|----------|
| base 唯一底层 + Invalidator 委托 | base 提供唯一失效底层，CacheInvalidator 保留 struct（Excel entityType 分发语义）委托 base | ✓ |
| 彻底删除 CacheInvalidator | 更彻底但改动面扩大到 excel 管道（Phase 91 明确排除的独立管道） | |
| 失效辅助不动 | 只统一读写路径，SC-1 的 Invalidate 部分措辞需修订 | |

**User's choice:** base 唯一底层 + Invalidator 委托

### Q3 验收锚点

| Option | Description | Selected |
|--------|-------------|----------|
| 定性清零 + LOC ≥200 | 33 处全走 GetOrSetJSON + 方法体 ≤6 行 + 无样板残留；LOC 净减 ≥200（诚实区间 200-300） | ✓ |
| 纯定性锚点 | 只要求样板清零，Phase 89/90/91 量化风格不延续 | |
| 高锚点 ≥400 | 超出诚实区间（base 新增代码抵消），有偷空风险 | |

**User's choice:** 定性清零 + LOC ≥200
**Notes:** Phase 91 D-07 混合标准先例（定性底线 + 量化锚点）。

## Area 3: DataCacheService 处置

**Claude 摆出的硬约束（用户接受）：** CACHE-UNIFY-04 字面"迁移到 system/"存在 root↔system import cycle（mac_history_heatmap/query_service 同包引用 DataCacheService，system 已 import services root）——不连带改造 mac_history 则编译失败。

### Q1 DataCacheService 处置

| Option | Description | Selected |
|--------|-------------|----------|
| 原地定性为基础设施 | 定性为 CacheProvider 实现底座永久留 root，消除平行 GetExpiration，CACHE-UNIFY-04 措辞修订 | ✓ |
| 字面迁移 + 连带改造 | 迁 system/ + mac_history 改吃接口 + 12+ API 文件 import 同步，~15 文件改动面 | |
| 迁 base 包 | 与 CacheProvider 接口同居，但 base 从纯抽象层变抽象+实现混居 | |

**User's choice:** 原地定性为基础设施
**Notes:** import cycle 硬约束 + DataCacheService 本质是 NewCacheProvider 的 Adaptee（基础设施非业务缓存服务）；零 import 风暴零行为变更。

### Q2 @Deprecated 标注语义

| Option | Description | Selected |
|--------|-------------|----------|
| 不标 deprecated，定位注释 | 文件头注释双定位（pkg/cache.Cache 封装层 + CacheProvider 底座），新代码走 base 路径 | ✓ |
| 标 @Deprecated 字面满足 | CACHE-UNIFY-04 字面满足，但"基础设施标 deprecated"语义矛盾（core 永远在用） | |

**User's choice:** 不标 deprecated，定位注释

## Area 4: 范围边界与验证锚点

### Q1 monitor 同名 CacheProvider

| Option | Description | Selected |
|--------|-------------|----------|
| 纳入：仅 rename 消歧 | monitor 侧 rename（如 CacheOperator），纯包内 rename + adapter 同步，语义零变更 | ✓ |
| 不纳入，deferred | 严格聚焦 ROADMAP 范围，但 base 定名后同名异形共存认知混乱 | |
| 完全统一到 base | 不可行：监控需要 Keys/FlushDB 等额外能力，强行统一破坏功能 | |

**User's choice:** 纳入：仅 rename 消歧
**Notes:** 最终名 planner 定（CacheOperator 为建议）。

### Q2 SC-5 端到端验证

| Option | Description | Selected |
|--------|-------------|----------|
| miniredis 自动化 | v1.27 INFRA-01 基建写集成测试（读写/TTL/失效/NoOp/Stats），可回归进 CI | ✓ |
| 真环境手动验证 | 最接近字面但不可回归，与 v1.29 "不可无声倒退"精神相悖 | |
| 自动化 + 手动双保险 | 成本最高验证最完整 | |

**User's choice:** miniredis 自动化
**Notes:** 手动冒烟不做强制要求（Phase 91 handler smoke 同款思路）。

### Q3 收口同步

| Option | Description | Selected |
|--------|-------------|----------|
| CLAUDE.md + invariants 都做 | 修订过时 Dual Cache Architecture 段 + 新增 Convention 段；invariants 扫描锁"无样板残留"（warning 不 fail） | ✓ |
| 仅 CLAUDE.md | 无机械防线，与 89/90 AST 锁值风格断裂 | |
| 都不做，留 closeout | 三 phase 惯例断裂，closeout 时上下文已遗失 | |

**User's choice:** CLAUDE.md + invariants 都做

## Claude's Discretion

- TTL 参数在 GetOrSetJSON 签名中的形态（预解析 Duration vs configKey+default+resolver）
- system.CacheServiceBase 旧名是否留 alias（与 D-02 对齐）
- base 包内文件布局与命名（GetOrSetJSON vs GetOrSetCached 等）
- monitor rename 目标名（CacheOperator 为建议）
- 33 处迁移在 3 个 plan 中的分组微调
- 错误包装/日志格式细节
- invariants 扫描检测模式（AST vs 正则）
- Invalidate/InvalidatePattern 签名细节（error vs void+warn）

## Deferred Ideas

- DataCacheService 字面迁移 + mac_history 接口化改造（cycle 解除后可评估，v1.30+）
- CacheInvalidator struct 消除 / excel 管道重构（D-04 保留委托）
- monitor 缓存监控接口语义统一（D-08 仅 rename）
- cache_config_service.go（TTL 配置真相源）重构/迁移
- mac_history_cache_decorator.go / template_cache.go 统一评估

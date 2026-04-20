# Phase 4: Widget 数据获取机制 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-20
**Phase:** 04-widget-data-fetching
**Areas discussed:** API调用方式, 数据转换实现, 批量获取策略, 缓存策略

---

## API 调用方式

| Option | Description | Selected |
|--------|-------------|----------|
| 内部服务调用 | 直接调用 Service 方法，性能最优，但仅限本服务内的 API | ✓ |
| HTTP 客户端调用 | 通过 HTTP 调用任意端点，灵活但性能开销大 | |
| 混合模式 | 内部服务优先，外部服务用 HTTP | |

**User's choice:** 内部服务调用
**Notes:** 性能最优，通过路由注册表找到对应的 Handler/Service，权限验证复用现有机制

---

## 数据转换实现

| Option | Description | Selected |
|--------|-------------|----------|
| JSONata 表达式引擎 | 功能强大，支持复杂表达式、过滤、聚合，但需要引入新依赖 | ✓ |
| 自定义解析器 | 轻量无依赖，但功能有限 | |
| 内置转换函数 | 预定义转换函数，简单可靠 | |

**User's choice:** JSONata 表达式引擎
**Notes:** 引入 `github.com/jsonata-go/jsonata` 库，支持复杂表达式、过滤、聚合

---

## 批量获取策略

| Option | Description | Selected |
|--------|-------------|----------|
| 并行获取 | 使用 goroutine + errgroup 并行获取，性能最优 | ✓ |
| 串行获取 | 逐个获取，简单但性能差 | |
| 自适应模式 | 根据 Widget 数量动态选择 | |

**User's choice:** 并行获取
**Notes:** 使用 `golang.org/x/sync/errgroup` 并行获取，控制并发数（最大 10 个）

---

## 失败处理

| Option | Description | Selected |
|--------|-------------|----------|
| 部分失败继续 | 返回每个 Widget 的数据或错误，前端单独处理失败项 | ✓ |
| 快速失败 | 任一失败则整体失败 | |
| 降级到缓存 | 失败时返回缓存数据 | |

**User's choice:** 部分失败继续
**Notes:** 返回每个 Widget 的数据或错误，前端单独处理失败项（显示错误状态）

---

## 缓存粒度

| Option | Description | Selected |
|--------|-------------|----------|
| Widget 级缓存 | 每个 Widget 数据独立缓存，可设置不同 TTL | ✓ |
| 仪表盘级缓存 | 整个仪表盘数据一起缓存 | |
| 无缓存 | 每次都实时获取 | |

**User's choice:** Widget 级缓存
**Notes:** 每个 Widget 数据独立缓存，TTL 基于 Widget 的 `refreshInterval` 配置

---

## 缓存失效

| Option | Description | Selected |
|--------|-------------|----------|
| TTL 自动过期 | 根据 Widget.refreshInterval 自动过期 | ✓ |
| 事件驱动失效 | 数据变化时主动清除缓存 | |
| 混合模式 | TTL + 手动刷新接口 | |

**User's choice:** TTL 自动过期
**Notes:** 简单可靠，无需复杂的事件订阅

---

## Claude's Discretion

无 — 所有决策均由用户明确选择

## Deferred Ideas

None — discussion stayed within phase scope

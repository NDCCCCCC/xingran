---
phase: 73
phase_name: P1 重要补齐
slug: p1-pending
gathered: 2026-08-21
status: Ready for planning
research_basis: derive-from-CONTEXT (CONTEXT.md 已经过 discuss-phase 完整分析)
---

# Phase 73: P1 重要补齐 - Research

**Gathered:** 2026-08-21
**Status:** Ready for planning
**Basis:** CONTEXT.md 已经过 discuss-phase 完整 4 区域讨论,本 RESEARCH.md 是技术可行性 + 验证架构的派生文档。

<research_summary>

## 一句话总结

P1 重要业务模块(8 个 0% 测试包,共 2259 stmts)的测试补齐。沿用 Phase 72 已 SHIPPED 的 ad_account(handler)+ portwrite(service)双范本 + 真实中间件 + 真实 SM4 加密路径,在不动业务代码前提下将每个包推到 ≥70% 覆盖率。

## Phase 73 与 Phase 72 关键差异

| 维度 | Phase 72 | Phase 73 |
|------|----------|----------|
| 起点 | 11 个 test 文件已存在(services/system 部分子模块) | 8 个 P1 包全部 0%,无基础 |
| 最大包 | services/system 3483 stmts(14 子模块) | api/v1/rpa 612 stmts(7 子 handler) |
| 总规模 | 8204 stmts 跨 14 plan | 2259 stmts 跨 4 plan |
| 复杂度 | 中(子模块多,但单文件不大) | 中高(rpa 公开路由双 router) |
| 测试模式 | 全部已建立 | 沿用 Phase 72,无新设计 |

## 关键技术风险与缓解

### 风险 1: api/v1/rpa 公开路由双 router

- **现象**: `SetupPublicWorkerRouter` 无 auth,3 endpoint;`SetupRPARouter` 有 auth,6 子组
- **缓解**: 测试 setup 提供两套 router,公共 helper 提取(DB / SM4 cipher / 测试数据 factory 共享)
- **预期**: 单 plan 内可完成,无需独立 plan 拆分

### 风险 2: services/monitor 5 文件跨 CRUD/读/统计

- **现象**: cache_impl 部分 + login_log/oper_log/server_service 部分需真实 DB
- **缓解**: 混合策略 — cache_impl 走 portwrite 纯 mock,其余走 ad_account 范本(glebarez sqlite + 真实 service)
- **oper_log 表 DDL**: 复用 Phase 72 `api/v1/monitor/oper_log_handler_test.go` 中 DDL(若存在)

### 风险 3: oper_log_service 不豁免但占比大

- **现象**: oper_log_service.go 占 services/monitor 485 stmts 中约 100-150 stmts,豁免则 SC#6 难达
- **缓解**: 沿用 ad_account 范本(真实 DB + 真实 service 调用),与 Phase 72 D-04 严格均衡原则一致
- **测试层级**: handler 层测 HTTP 包装 + middleware 集成;service 层测 CRUD/Record* 私有方法 + 错误处理 + 批量操作

### 风险 4: 加权平均 ratchet 实际不可达 SC#7 写的 ≥55%

- **现象**: 8 个 P1 包 2259 stmts × 70% = 1581 covered 新增(100% 起点 0%)
- **数学**: 21.5% + 1581/43652 = 25.1% 加权(其他包不动);若其他包继续微调,实际 26-28%
- **缓解**: 沿用 Phase 72 D-07 手动 ratchet 到实际值模式,诚实 bump;ROADMAP SC#7 待 planner 阶段修订

</research_summary>

<validation_architecture>

## Validation Architecture

### 测试金字塔分层

```
        ┌─────────────────────────────┐
        │   Per-package E2E 验证       │  ← Phase 73 不实施
        ├─────────────────────────────┤
        │   Service 单元测试 (portwrite/ │  ← Plan 73-03/04
        │   ad_account 混合范本)        │
        ├─────────────────────────────┤
        │   Handler 集成测试 (glebarez  │  ← Plan 73-01/02
        │   sqlite + 真实 middleware)   │
        ├─────────────────────────────┤
        │   Pure unit (portwrite mock)  │  ← 边界场景
        └─────────────────────────────┘
```

### 验证维度(8 个 Nyquist 维度映射)

| 维度 | 验证策略 | 实施 plan |
|------|---------|-----------|
| D1: 功能正确性 | 表驱动 TC 覆盖正常路径 + 边界 + 异常 | 73-01/02/03/04 |
| D2: 接口契约 | handler 测试断言 `code=0` (success) / `code != 0` (error) | 73-01/02 |
| D3: 错误处理 | service 测试覆盖错误分支(返回 error / 写日志) | 73-03/04 |
| D4: 边界场景 | 空 DB / 空输入 / 重复请求 / 大数据量 | 73-03/04 |
| D5: 安全 | JWT token 工厂 + 真实 SM4 cipher 加密路径 | 73-01/02 |
| D6: 性能 | 不强制(Nyquist 不要求 phase 73 性能测试) | N/A |
| D7: 可观测性 | operlog.Record 被调用断言(handler 层) | 73-01/02 |
| D8: 验证策略存在 | **本文件即是 VALIDATION 来源** | — |

### 失败检测 → 反馈回路

```
go test ./internal/api/v1/duty/...
  ↓
exit code != 0 → executor 立即修复(在 plan 内)
  ↓
go test -coverprofile=coverage.out  →  go tool cover -func
  ↓
per-package % < 70% → executor 在 plan 内补测试
  ↓
final: ./github/scripts/check-coverage.sh exit 0(per-package 全 ≥70%)
  ↓
.ratchet: weighted_avg → 写入 .coverage-threshold + coverage-baseline.md
```

### 已知失败容忍

- **测试框架兼容性**: 沿用 `stretchr/testify v1.x` + `glebarez/sqlite v1.11.0`,已在 go.mod,无新依赖
- **CI timeout**: `go test -timeout 15m`(Phase 72 baseline);若 Phase 73 超过,上调至 20m
- **未覆盖文件容忍**: 单文件覆盖率 < 50% 但不强制(panic recovery / 错误格式化代码难测,优先保证 ≥70% 加权)

</validation_architecture>

<canonical_refs>

## Canonical References (完整列表见 CONTEXT.md §canonical_refs)

### 强制阅读(本阶段直接沿用)
- `.planning/phases/72-p0-core-supplement/72-CONTEXT.md` — D-01..D-12 决策完整记录
- `internal/api/v1/system/ad_account_handler_test.go` — handler 轻量范本
- `internal/services/portwrite/port_write_service_test.go` — service 完整范本
- `.planning/coverage-baseline.md` — 当前 21.5% baseline
- `.github/scripts/check-coverage.sh` — per-package gate

### 规划输入(已读)
- `.planning/REQUIREMENTS.md` — IMP-01..06 详细 stmts 数
- `.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt` — 76 行精确起点
- `.planning/ROADMAP.md` — Phase 73 完整 SC + Notes
- `.planning/PROJECT.md` §Current Milestone:v1.26

### 目标包代码结构(planner 必须先 map)
- `internal/api/v1/duty/` (265 stmts) — 2 文件
- `internal/api/v1/knowledge/` (273 stmts) — 2 文件
- `internal/api/v1/rpa/` (612 stmts) — 7 文件 + rpa_router.go
- `internal/api/v1/vdi/` (298 stmts) — 4 文件
- `internal/services/duty/` (114 stmts) — 1 文件
- `internal/services/knowledge/` (85 stmts) — 1 文件
- `internal/services/monitor/` (485 stmts) — 5 文件
- `internal/services/network/` (127 stmts) — 1 文件

### 测试基础设施(已在 main 上 live)
- `.coverage-threshold` — 当前 `21.5`
- `.github/workflows/ci.yml` — backend job Test step `-coverprofile=coverage.out -covermode=atomic -count=1`
- `.github/scripts/check-coverage.sh` — bash + awk 加权平均 gate(已扩展 per-sub-package grep)

</canonical_refs>

<alternatives_considered>

## 设计替代方案(已 discuss-phase 排除)

### Plan 拆分 3 种方案
1. **8 plans (1 包 1 plan)** — 粒度过细,executor 8 次 commit 节奏碎 → ❌
2. **4 plans (按复杂度横切)** — handlers 简单 / handlers 复杂 / services 简单 / services 中等 → ✓
3. **3 plans (按业务域)** — 跨 service+handler 边界,planner 复杂 → ❌

### oper_log_service 处理 2 种方案
1. **仍按 ≥70% 严格测** — ad_account 范本 + 真实 DB → ✓
2. **跳过 oper_log_service** — handler 层 71.2% 替代 → ❌(违反 D-04 严格均衡)

### rpa 公开路由 2 种方案
1. **与认证路由同套测试** — glebarez sqlite + handler 测试,差异仅 JWT middleware → ✓
2. **公开路由测试跳过** — 仅 ~55-60% 覆盖率 → ❌(SC#3 ≥70% 不可达)

### Ratchet 目标值 3 种方案
1. **Ratchet 到实际值** (~27-30%) — 跑全包测得新加权平均,诚实 bump → ✓
2. **Ratchet 到 ≥55%** (ROADMAP SC#7 写) — 实际数学不可达 → ❌
3. **Ratchet 到 ≥30%** (Phase 72 实际) — 保守但可能虚高 → ❌

</alternatives_considered>

---

*Phase: 73-P1 重要补齐*
*Research completed: 2026-08-21 (derived from CONTEXT.md discuss-phase output)*

---
gsd_state_version: 1.0
milestone: v1.29
milestone_name: closeout + audit
status: executing
last_updated: "2026-09-04T09:43:58.504Z"
last_activity: 2026-09-04 -- Phase 91 planning complete
---

# Project State (v1.29 — milestone workstream)

## Project Reference

See: .planning/PROJECT.md (updated 2026-09-04) — v1.29 Current Milestone 段

**Core value:** 按 2026-09-03 综合审计报告发现的优先级，逐批治理 7 项技术债行动；后端常量/CRUD/缓存/配置备份 + 前端 API 工厂化 + Phase 88 收口，使代码质量基线从此不可无声倒退。

**Current focus:** Phase 91 (CRUD 复用 base.Repository[T]) 待 discuss — v1.29 常量集中化双 phase (89+90) 已 SHIPPED

## Current Position

Phase: 91 (待 discuss)
Plan: —
Status: Ready to execute
Last activity: 2026-09-04 -- Phase 91 planning complete
Resume file: .planning/workstreams/milestone/phases/90-timeouts-port-protocol-concurrency/90-VERIFICATION.md
Next action: `/gsd:discuss-phase 91 --chain`

### Phase 89 Decisions (D-01..D-19) — captured 2026-09-04

- **D-PRINCIPLE**: 行业最佳实践为唯一依据,允许任何形式重构;去除硬编码/处理 todo/消除重复/合理抽象
- **3 常量最终定义** (深度思考后简化): DefaultCurrent=1, DefaultPageSize=10, MaxPageSize=200 (行业惯例对齐 GitHub=100/Stripe=100;200 略宽)
- **D-08/D-09/D-16 删除**: KnowledgeDefaultPageSize=100 / KnowledgeMaxPageSize=500 / AccountPoolDefaultPageSize=20 全部判定为"拍脑袋"决定,无业务依据
- **D-01..D-19**: leaf const pkg pkg/constants/pagination.go + helper 放 pkg/query/pagination.go + NormalizePagination 纯函数 + binding 去 max=100 + Pilot 先行(knowledge_service.go) + 89-02 复制模式迁移剩余 7 个文件
- **业务行为变更** (用户已接受): 知识搜索 default 100→10 max 500→200;AD 账号池 default 20→10,`>200` 反常回退改为标准 clamp;ad_domain `== 0`→`<= 0` bug 修复
- 见 89-CONTEXT.md / 89-DISCUSSION-LOG.md 完整记录

## Accumulated Context (carried from v1.28)

### Decisions to preserve

- D-26-01..05 (v1.26 后端覆盖率 gate): 4 层 CI 防倒退（加权阈值 + per-dir floor + ratchet + PR diff ≥80%）继续生效
- D-27-01..04 (v1.27 后端覆盖率优秀 II): 测试基建 (miniredis/httpmock/ScrapliWrapper/LDAPClientIface/TestHelperProcess/AST守护) 沿用
- D-28-01..04 (v1.28 前端覆盖率): 阶段性收口 45.13%，不再 push 到 70% 目标
- operlog regression_test.go: 11 强制敏感关键词、25 OperType 常量全程保持绿
- status_constants_test.go: 状态 0/1 命名常量全程 AST 锁值

### Blockers (active)

- 无新增 blocker（v1.29 主要是技术债治理，业务风险低）

### Pending Todos (carry forward, not in v1.29 scope)

- `.planning/todos/pending/operlog-exclude-paths.md` — operlog 白名单配置驱动（RPA heartbeat 日志污染），独立 deferred 到后续 milestone（不在 v1.29 7 项治理范围内）

### Audit Baseline (2026-09-03)

| 维度 | 基线 | 数据 |
|------|------|------|
| 后端覆盖率 | 78.12% | v1.27 SHIPPED |
| 前端覆盖率 | 45.13% | v1.28 SHIPPED (45.87% at batch47, final = 阶段性收口) |
| 测试通过 | 1688/1688 | 45/45 dirs Gate PASS |
| 真实 TODO | 18 项 | 后端 13 + 前端 7 |
| 中高度硬编码 | ~20 处 | 分页 12+ / 超时 6 / URL/端口 2 |
| CRUD 重复 | 8 services | 60-70% 重复；base.Repository[T] 已存在未用 |
| 缓存层重复 | 3 路径 | legacy root + system/ + operations/ 同一模式 |
| 配置文件 TODO | 3 函数 | config_backup_service.go:158/206/543 |
| 前端 API 文件 | ~15 | 散落在 src/lib/ |

### Phase 88 收口状态

- Phase 88 已完成 18+ batches (R24-R41, R42-R47)
- GLOBAL: 45.13% → 45.87% (batch47 末态)
- 1688 tests passing, Gate 45/45 dirs PASS
- Blocker: 边效益递减，每批仅 ~0.27pp，距 70% 目标差 24.13pp
- Decision: 接受 45.13% 阶段性收口（用户 2026-09-04 确认）

## Next Step

`/gsd:plan-phase 89` (待执行) — Phase 89: 常量集中化 (pkg/constants/pagination.go + timeouts.go)

或 `/gsd:discuss-phase 89` 先讨论实施方案

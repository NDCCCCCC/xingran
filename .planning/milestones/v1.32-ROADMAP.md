---
last_updated: 2026-09-09
milestone: v1.32
update_trigger: v1.32 defined — 审计驱动的安全与可靠性收尾（5 phases / 22 requirements）
previous_update: 2026-09-08 v1.31 SHIPPED — V131 技术债清偿 archive
---

# Roadmap: XingRan-Next 运维管理系统 — v1.32

> **v1.31 milestone 历史已归档**: `.planning/milestones/v1.31-ROADMAP.md` / `.planning/milestones/v1.31-REQUIREMENTS.md`
> 本文件自 2026-09-09 起追踪 **v1.32 V132 审计驱动的安全与可靠性收尾**。

## Current Milestone: v1.32 V132 审计驱动的安全与可靠性收尾 (Audit-Driven Security & Reliability)

**Goal**: 基于 2026-09-09 全量后端审计报告（`.planning/reviews/20260909-backend-audit.md`）的 18 项 P0/P1 风险 + 4 项 Phase 104/107/108 MUST-FIX + 6 项新发现并发风险，按用户决策（范围=P0+P1+回归守护；TLS 选项=环境变量；回归守护=Phase 109 前置）分 5 个 phase 收尾。

**Source planning data**:

- `.planning/REQUIREMENTS.md` v1.32 范围定义（22 requirements / 5 类别）
- `.planning/PROJECT.md` Current Milestone 段（D-01~D-06 锁定决策）
- `.planning/reviews/20260909-backend-audit.md`（审计基线）

**Phase 编号**: 从 Phase 109 续编（v1.31 用 102-108，v1.30 用 96-101，v1.29 用 89-95）

### Phase Dependency Graph

```
Phase 109 (回归守护前置 — 8 项 invariants)
   ├─→ Phase 110 (P0 安全 TLS 环境变量化 — Redis/AD authenticator/WebSocket CORS)
   ├─→ Phase 111 (P0 并发裸 goroutine 守护 — OperLog/AD dept sync/reconnect/login log + Captcha fail-closed)
   └─→ Phase 112 (P1 handler 收敛补丁 — operations 3 端点 + login_log nil + HandleGetByID)
                                                          └─→ Phase 113 (部署文档同步 — secret-management.md 新增 TLS/Origin env 清单)
```

### Phase 状态

| Phase | 标题 | Requirements | 预计 Plan 数 | Status |
|-------|------|--------------|--------------|--------|
| 109 | 回归守护前置 | GUARD-01..08 | 8 | ✅ SHIPPED |
| 110 | P0 安全 TLS 环境变量化 | TLS-01..06 | 5 | ✅ SHIPPED |
| 111 | P0 并发裸 goroutine 守护 + Captcha | GOR-01..04 + CAP-01 | 4 | ⏳ PLANNED |
| 112 | P1 handler 收敛补丁 | HANDLER-01..05 | 2 | ⏳ PENDING |
| 113 | 部署文档同步 | DOC-01 | 1 | ⏳ PENDING |

Plans:
- [x] `.planning/phases/109-regression-guards/PLAN.md` — 8 plans (109-01 .. 109-08)
- [x] `.planning/phases/110-tls-env-vars/PLAN.md` — 5 plans (110-01 .. 110-05)
- [x] `.planning/phases/111-concurrency-guards/PLAN.md` — 4 plans (111-01 .. 111-04)

---
## Phase Details

*(empty — execute phase to populate)*

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|----------|
| Phase 102 机械常量化（缓存键/状态/分页） | ✅ SHIPPED 2026-09-07 | 5/5 | CACHE-01..02 + STATUS-01 + PAGI-01 | 2026-09-07 | 2026-09-07 |
| Phase 103 缓存闭包收敛 base 单一权威 | ✅ SHIPPED 2026-09-08 | 4/4 | CONV-01..04 | 2026-09-08 | 2026-09-08 |
| Phase 104 handler 层架构收敛（wire+handler） | ✅ SHIPPED 2026-09-08 | 4/4 | WIRE-01 + HANDLER-01..02 | 2026-09-08 | 2026-09-08 |
| Phase 105 前端 CRUD 收敛 apiFactory | ✅ SHIPPED 2026-09-08 | 4/4 | FEAPI-01..04 | 2026-09-08 | 2026-09-08 |
| Phase 106 前端映射统一与类型卫生 | ✅ SHIPPED 2026-09-08 | 4/4 | FEMAP-01..03 + TS-01..02 | 2026-09-08 | 2026-09-08 |
| Phase 107 TODO 清零 + nilness 排查 | ✅ SHIPPED 2026-09-08 | 4/4 | TODO-01..06 + NIL-01 | 2026-09-08 | 2026-09-08 |
| Phase 108 skip 测试恢复 | ✅ SHIPPED 2026-09-08 | 4/4 | SKIP-01..02 | 2026-09-08 | 2026-09-08 |
| Phase 109 回归守护前置 | ✅ SHIPPED 2026-09-09 | 8/8 | GUARD-01..08 | 2026-09-09 | 2026-09-09 |
| Phase 110 P0 安全 TLS 环境变量化 | ✅ SHIPPED 2026-09-09 | 5/5 | TLS-01..06 | 2026-09-09 | 2026-09-09 |
| Phase 111 P0 并发裸 goroutine 守护 + Captcha | ⏳ PLANNED | 4/4 | GOR-01..04 + CAP-01 | — | — |

**Total:** 10 phases / 37 plans / 47 requirements — 9 delivered, 1 planned.

---

## Out of Scope (locked)

- **captcha-background 1=启用语义** — QUIRK-80-03-D 就地锁定，非 bug
- **operlog exclude_paths / LDAP InsecureSkipVerify / agent 裸 c.JSON / 三层 adapter 物理合并 / 协议默认值常量化** — Future Requirements

---

*Last updated: 2026-09-09 — Phase 111 planned; v1.32 scope defined. Previous milestone: v1.31 V131 技术债清偿 SHIPPED 2026-09-08（7 phases / 25 plans / 29 requirements）.*

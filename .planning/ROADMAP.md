---
last_updated: 2026-08-19
update_trigger: v1.23 启动块 Phase 68 EXECUTED (1/1 plan, 5 commits a21dcec..25ded8f, DEPLOY-01~05 PASS, SUMMARY `68-01-SUMMARY.md` 落地);Phase 70-07 EXECUTED 2026-08-19 (visual gate passed, 收口 commit `e138411`)
last_plan_update: 2026-08-19 — Phase 68 SUMMARY 落地 + ROADMAP Phase 68 标记 EXECUTED + Progress 表更新为 7/7 phases (64-70) 全部落地
previous_update: 2026-08-19 after Phase 70-02 EXECUTED
---

# Roadmap: XingRan-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联 + 部门物理位置映射** — Phases 28 + 39 (shipped 2026-06-10 / 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 + Phase 47 root-cause (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 + Phase 49 gap closure (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08) — see [milestones/v1.19-ROADMAP.md](milestones/v1.19-ROADMAP.md)
- ✅ **v1.20 网络设备 VLAN + 端口绑定** — Phase 56 (shipped 2026-07-10) — see [milestones/v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)
- ✅ **v1.21 API Key 认证链修复 + 能力补全 (API Key Auth Chain Repair + Feature Completion)** — Phases 57-62 (shipped 2026-08-18) — see [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)
- ✅ **v1.22 + v1.23 + v1.24 + v1.25 启动块 (Combined Launch Blocks)** — Phases 64-70 (shipped 2026-08-18 → 2026-08-19; 7 phases, 20 plans, audit `passed`) — see [milestones/v1.22-v1.25-ROADMAP.md](milestones/v1.22-v1.25-ROADMAP.md) + [milestones/v1.22-v1.25-REQUIREMENTS.md](milestones/v1.22-v1.25-REQUIREMENTS.md)
  - v1.22 前端品牌化改造 (Phases 64-67)
  - v1.23 部署稳健性 & 文档一致性 (Phase 68)
  - v1.24 字典与状态值治理 (Phase 69)
  - v1.25 系统设置页面布局重构 (Phase 70)

<details>
<summary>✅ v1.22 + v1.23 + v1.24 + v1.25 Combined (Phases 64-70) — SHIPPED 2026-08-18 → 2026-08-19</summary>

**Milestone Goal:** v1.22 把 `brand-spec.md` 像素实测品牌令牌固化进 design-system;v1.23 闭环 SM2 密钥配置部署稳健性;v1.24 建立状态语义单一真相源(94 常量 + 11 字典);v1.25 重构系统设置页对齐 v1.22 品牌理念。共 7 phases / 20 plans / 36 items (15 v1.22 REQ + 5 v1.23 DEPLOY-XX + 4 v1.24 DICT-XX + 12 v1.25 D-XX)。

- [x] Phase 64: 品牌令牌层落地 + 对比度验证 — Phases 64-67 完整细节见 [milestones/v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md)
- [x] Phase 65: 主题系统收敛 — 同上
- [x] Phase 66: 通用组件样式 + 硬编码色扫描 — 同上
- [x] Phase 67: 构建回归 + 视觉确认 — 同上
- [x] Phase 68: 部署稳健性 & 文档一致性 — 完整细节见 [milestones/v1.23-ROADMAP.md](milestones/v1.23-ROADMAP.md)
- [x] Phase 69: 字典与状态值治理 — 完整细节见 [milestones/v1.24-ROADMAP.md](milestones/v1.24-ROADMAP.md)
- [x] Phase 70: 系统设置页面布局重构 — 完整细节见 [milestones/v1.25-ROADMAP.md](milestones/v1.25-ROADMAP.md)

**Audit**: [v1.22-v1.25-MILESTONE-AUDIT.md](v1.22-v1.25-MILESTONE-AUDIT.md) — passed (20/20 reqs, 7/7 phases, 5/5 integration, 4/4 E2E flows)

</details>

---

## Phases (Next Milestone — TBD by /gsd:new-milestone)

**Milestone Goal:** TBD — 见下一里程碑启动块(v1.26+)

启动新里程碑:`/clear` 然后 `/gsd:new-milestone`

---

## Progress

**Execution Order:**
Phases execute in numeric order: 63 → 64 → 65 → 66 → 67 → 68 → 69 → 70 (Phase 63 独立无依赖,2026-08-20 SHIPPED)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 63. 前端工具链自动化 | independent | 1/1 | ✅ SHIPPED | 2026-08-20 |
| 64. 品牌令牌层落地 + 对比度验证 | v1.22 | 1/1 | ✅ SHIPPED | 2026-08-18 |
| 65. 主题系统收敛 | v1.22 | 1/1 | ✅ SHIPPED | 2026-08-18 |
| 66. 通用组件样式 + 硬编码色扫描 | v1.22 | 1/1 | ✅ SHIPPED | 2026-08-18 |
| 67. 构建回归 + 视觉确认 | v1.22 | 1/1 | ✅ SHIPPED | 2026-08-18 |
| 68. 部署稳健性 & 文档一致性 | v1.23 | 1/1 | ✅ EXECUTED | 2026-08-19 |
| 69. 字典与状态值治理 | v1.24 | 8/8 | ✅ EXECUTED | 2026-08-19 |
| 70. 系统设置页面布局重构 | v1.25 | 7/7 | ✅ EXECUTED | 2026-08-19 |

---

## Coverage Map (v1.22 + v1.23 + v1.24 + v1.25)

完整 v1.22 (15) + v1.23 (5) + v1.24 (4) + v1.25 (12) requirement coverage 见 [milestones/v1.22-v1.25-REQUIREMENTS.md](milestones/v1.22-v1.25-REQUIREMENTS.md) — 36/36 items mapped to exactly one phase (0 orphans, 0 duplicates)。

**Inter-phase dependency graph (combined):**

```
Phase 64 (品牌令牌 + 对比度验证)
   │
   └─→ Phase 65 (主题系统收敛) [depends on 64]
          │
          └─→ Phase 66 (通用组件样式 + 硬编码扫描) [depends on 65]
                 │
                 └─→ Phase 67 (构建回归 + 视觉确认) [depends on 66]
                                                │
Phase 68 (SM2 部署稳健性)         Phase 69 (字典 + 状态治理)
   [独立 phase]                          [独立 phase]
                                                │
                                          Phase 70 (settings 重构) [depends on 67]
```

Phase 63 (frontend-toolchain-automation) 独立 IN PROGRESS,提供 CI / lint / 测试基建。

---

## Archive: v1.21 Milestone History

<details>
<summary>✅ v1.21 API Key 认证链修复 + 能力补全 (Phases 57-62, shipped 2026-08-18) — 详见 [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)</summary>

- ✓ **Phase 57**: 认证链核心修复 + 回归测试 — 修复 `setUserContextForAPIKey` 类型断言(P0-2),消除 MultiAuth 死代码(P0-1),集成测试锁住三路径链路
- ✓ **Phase 58**: 前后端路由契约对齐 — 前端 `getAPIKey` / `updateAPIKey` / `deleteAPIKey` 改 POST 对齐后端;`CONTRACT-02` 字段命名 camelCase 对齐
- ✓ **Phase 59**: 可观测性 / 使用日志修复 — 使用日志记录时机后移,异步 goroutine detached context,`successRate` 可信
- ✓ **Phase 60**: 安全加固与启用决策 — MultiAuth 路由挂载启用决策 + API Key 哈希存储决策 + 限流响应头编码修复 + 重复索引移除
- ✓ **Phase 61**: 资源级权限矩阵 + 限流生产调优 — `RequireAPIKeyResourcePermission` resource 参数真实生效 + `RateLimitByScope` 生产接入与调优(conditional on Phase 60 AUTH-03=启用)
- ✓ **Phase 62**: 数据库核心安全加固(跨 AI 评审修复) — internal/core/db 跨 AI 评审(codex + opencode)共识 C1-C7 + 单方 HIGH/MEDIUM 全部清零,迁移安全 / 种子凭据 / 并发保护

**v1.21 shipped 14/14 v1.21 requirements + Phase 62 跨 AI 评审 14 项 review items。** 唯一遗留:Phase 58 SC#1-SC#4 端到端验证因 dev DB 性能延期(代码契约修复已提交且自动化门全绿),见 `58-01-SUMMARY` §Deferred。

</details>

---

## Archive: Pre-v1.21 Milestone History

<details>
<summary>✅ Earlier milestone phase history (v1.0–v1.20) preserved for reference</summary>

- ✓ Phases 1-2 — v1.0 工位导入部门/用户关联 (7 plans)
- ✓ Phase 3 — v1.1 信息点导入设备端口关联 (1 plan)
- ✓ Phases 4-7 — v1.2 可配置仪表盘 (11 plans)
- ✓ Phases 8-10 — v1.3 技术债清理 (9 plans)
- ✓ Phase 11 — v1.4 MAC地址采集优化 (4 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据 (26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10 plans)
- ✓ Phases 17 — v1.7 加密配置同步 (6 plans)
- ✓ Phase 18 — v1.8 登录端加密 (4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11 plans)
- ✓ Phase 21 — v1.10 网络设备权限修复 (1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (18 plans)
- ✓ Phase 26 — v1.13 资产管理 (6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4 plans)
- ✓ Phases 30-34 — 前端性能/P0收尾/P1P2/React best practices/操作日志全模块集成 (~40 plans)
- ✓ Phases 37-39 — 前端部门选择统一/AD账号池统一/部门物理位置映射
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-47 — v1.17 资产对账 + 根因修复 (16 plans)
- ✓ Phases 48-49 — v1.18 网络设备硬件清单 + gap closure (5 plans)
- ✓ Phases 50-55 — v1.19 网络设备写命令 (9 plans, 5 build + 1 cleanup)
- ✓ Phase 56 — v1.20 网络设备 VLAN + 端口绑定 (5 plans)

**Total shipped through v1.21**: 173 plans across 62 phases (Phases 1-56 + 57-62).
**Total shipped through v1.22-v1.25 (combined)**: +20 plans across 7 phases (64-70).

**Cumulative shipped through 2026-08-20**: 194 plans across 70 phases (Phases 1-63 + 64-70).

</details>

*Last updated: 2026-08-20 — Combined v1.22 + v1.23 + v1.24 + v1.25 milestone complete (Phases 63-70 / 21 plans / 36 items). 8/8 phases passed verification; audit [v1.22-v1.25-MILESTONE-AUDIT.md](v1.22-v1.25-MILESTONE-AUDIT.md) `passed` (20/20 reqs + 5/5 integration + 4/4 E2E flows). Combined archive [milestones/v1.22-v1.25-ROADMAP.md](milestones/v1.22-v1.25-ROADMAP.md) + [milestones/v1.22-v1.25-REQUIREMENTS.md](milestones/v1.22-v1.25-REQUIREMENTS.md) created; per-block archives preserved for full traceability. **Phase 63 frontend-toolchain-automation SHIPPED 2026-08-20** (1/1 plan, 5/5 SC, 7-commit chain `760606a..d2184fd`).*

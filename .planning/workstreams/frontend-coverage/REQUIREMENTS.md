# Requirements: v1.28 前端测试覆盖率优秀 (Frontend Test Coverage Excellence)

**Defined:** 2026-08-23
**Core Value:** 前端全量口径测试覆盖率达到优秀（≥70%），CI 防倒退 gate 对称后端 v1.26 治理，使前端覆盖率从此不可无声倒退。
**Workstream:** `frontend-coverage`（与 `milestone` workstream 的 v1.27 后端覆盖率 II 并行；Phase 82 起，75-81 留给 v1.27）

**基线（2026-08-23 实测）:**

- 旧口径（Phase 63，只算被 import 文件）: statements 24.58% / branches 15.04% / functions 18.94% / lines 24.75%（71 文件 3376 stmts）
- **全量口径（本里程碑基准，coverage.include 全 src）: statements 3.67% (830/22602) / branches 1.98% / functions 2.38% / lines 3.70%（584 文件）**

## v1 Requirements

### 治理与口径 (GOV)

- [x] **GOV-01**: vitest coverage 切换全量口径——`coverage.include` 显式圈定 `src/**/*.{ts,tsx}`（Vitest 4 已移除 `coverage.all`，未配置 include 时只报告被 import 文件），白名单目录同步 exclude，未测试文件计入报告
- [x] **GOV-02**: 前端覆盖率基线落盘（起点 3.67% / 22602 stmts + per-dir 快照 + ratchet 记录），与后端 `.planning/coverage-baseline.md` 模式对称
- [x] **GOV-03**: CI 前端全局阈值 gate 切到全量口径并 ratchet 只升不降（失败即阻断）
- [x] **GOV-04**: PR diff coverage ≥80% gate（复用后端 74-10 自实现 bash+awk 模式，前端版）
- [x] **GOV-05**: per-directory floor gate（白名单外目录 ≥70%，未达标目录走白名单 ratchet 过渡，对称后端 `p1_package_check` / `p2_package_check`）

### P0 基建层 ≥70% (INFRA) — ~3,900 stmts

- [x] **INFRA-01**: lib 通信层 ≥70%（api.ts 加密客户端 / opsApi / networkApi / menuApi / profileApi / 列配置 api 等 20 文件 1042 stmts）
- [x] **INFRA-02**: utils 层 ≥70%（sm2 / sm4 / encoding 国密、token 三件套、dualLevelCache、errorHandler、authHelpers、deptUtils、datetime 等 23 文件 950 stmts）
- [x] **INFRA-03**: hooks 层 ≥70%（27 文件 1050 stmts，含 usePagination / useServerSort / usePersistedState 续推）
- [x] **INFRA-04**: store 层 ≥70%（auth / menu / settings / tabs / dashboard / layout / notice / theme 9 文件 589 stmts）
- [x] **INFRA-05**: services + router + constants + types 收尾 ≥70%（~626 stmts）

### P1 组件层 ≥70% (COMP) — ~5,000 stmts

- [x] **COMP-01**: components/shared ≥70%（21 文件 892 stmts，最高价值共享组件）
- [x] **COMP-02**: components/dashboard ≥70%（29 文件 1068 stmts，仪表盘 Widget 体系）
- [x] **COMP-03**: components/layout ≥70%（HybridLayout / Sidebar / Header，16 文件 507 stmts）
- [x] **COMP-04**: components/CronSelector 316 + components/captcha 154 + components/operations 149 ≥70%
- [x] **COMP-05**: components/network 324（现 50.6%）+ components/reconciliation 144（现 18.1%）+ 零散小组件续推 ≥70%

### P2 页面层 ≥70% (PAGES) — ~13,100 stmts

- [x] **PAGES-01**: pages/operations ≥70%（76 文件 3611 stmts，最大单一目录）
- [x] **PAGES-02**: pages/system ≥70%（56 文件 2203 stmts）
- [x] **PAGES-03**: pages/network ≥70%（61 文件 1962 stmts）
- [x] **PAGES-04**: pages/duty 1190 + pages/ad-domain 1082 ≥70%（48 文件）
- [x] **PAGES-05**: monitor 627 / vdi 567 / workorder 551 / asset 475 / knowledge 262 / dashboard-system 203 / my-notices 127 / login 62 续推 + 零散页 ≥70%（~70 文件）

### 质量守护 (QUAL)

- [ ] **QUAL-01**: 全量 vitest 0 失败 0 flaky（现有 19 文件 159 测试不回归）
- [x] **QUAL-02**: 白名单治理——cad-editor 804 + cad-elements 224 等重画布 UI 排除登记（理由 / 面积 / 复审条件），白名单面积 ≤5% 总语句
- [x] **QUAL-03**: 测试公共 harness 沉淀（mock api / antd message / router wrapper 等高频样板），供 P1/P2 复用提效

## v2 Requirements

（无——本里程碑为覆盖率专项；后续候选：E2E 测试层、视觉回归）

## Out of Scope

| Feature | Reason |
|---------|--------|
| E2E 测试（Playwright / Cypress） | 本里程碑只做 vitest 单元/组件测试 |
| 视觉回归测试 | 与覆盖率正交 |
| CAD 编辑器可测性重构 | 白名单排除而非改造（对称后端"辅助包不强制覆盖"） |
| 后端覆盖率 | v1.27 `milestone` workstream 范围 |
| bundle size / knip 治理 | Phase 63 advisory 遗留，另行处理 |
| 业务逻辑修改 | 仅补测试；测试暴露的 bug 修复除外 |

## Traceability

Roadmap created 2026-08-23 — `.planning/workstreams/frontend-coverage/ROADMAP.md`（Phases 82-88，7 phases）。
映射备注：顶层 `design-system` 194 stmts 归入 Phase 84（COMP-05 零散桶），保证白名单外无无主面积、全局 ≥70% 由 per-dir 全 ≥70% 数学保证。

| Requirement | Phase | Status |
|-------------|-------|--------|
| GOV-01 | Phase 82 | Complete |
| GOV-02 | Phase 82 | Complete |
| GOV-03 | Phase 82 | Complete |
| GOV-04 | Phase 82 | Complete |
| GOV-05 | Phase 82 | Complete |
| INFRA-01 | Phase 83 | Complete |
| INFRA-02 | Phase 83 | Complete |
| INFRA-03 | Phase 83 | Complete |
| INFRA-04 | Phase 83 | Complete |
| INFRA-05 | Phase 83 | Complete |
| COMP-01 | Phase 84 | Complete |
| COMP-02 | Phase 84 | Complete |
| COMP-03 | Phase 84 | Complete |
| COMP-04 | Phase 84 | Complete |
| COMP-05 | Phase 84 | Complete |
| PAGES-01 | Phase 85 | Complete |
| PAGES-02 | Phase 86 | Complete |
| PAGES-03 | Phase 86 | Complete |
| PAGES-04 | Phase 87 | Complete |
| PAGES-05 | Phase 87 | Complete |
| QUAL-01 | Phase 88 | Pending |
| QUAL-02 | Phase 82 | Complete |
| QUAL-03 | Phase 83 | Complete |

**Coverage:**

- v1 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-23*
*Last updated: 2026-08-23 — roadmap created (Phases 82-88), 23/23 mapped, design-system 194 stmts 归入 COMP-05 零散桶*

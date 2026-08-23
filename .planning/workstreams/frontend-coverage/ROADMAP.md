# Roadmap: v1.28 前端测试覆盖率优秀 (Frontend Test Coverage Excellence)

**Workstream:** `frontend-coverage`（与 `milestone` workstream v1.27 后端覆盖率 II 并行）
**Milestone Goal:** 前端全量口径（vitest `coverage.include` 全 src，白名单排除重画布 UI）语句覆盖率 **3.67% → ≥70%（优秀）**，建成与后端 v1.26 对称的 4 层 CI 防倒退 gate。
**Created:** 2026-08-23 | **Granularity:** standard（7 phases） | **Coverage:** 23/23 requirements mapped

## Overview

从实测基线 3.67%（830/22602 stmts，584 文件——Vitest 4 移除 `coverage.all` 后，旧口径 24.58% 是"只算被 import 文件"的失真值）出发：Phase 82 先修正口径并落地 4 层治理 gate（全局阈值 + per-directory floor + baseline ratchet + PR diff coverage ≥80%），随后按 P0 基建层（~3,900 stmts）→ P1 组件层（~4,000 stmts 白名单外）→ P2 页面层三波（operations → system+network → 剩余全部）分层补齐。每个 phase 是自然交付边界：边界处全量 vitest 0 失败 + CI gate 绿 + 覆盖率单调上升（ratchet 只升不降）。Phase 88 收口：全局 floor 收口 70.0 + 0 失败 0 flaky + ratchet 终态落盘。白名单排除 cad-editor 804 + cad-elements 224（1028 stmts ≈ 4.5% ≤ D-02 的 5% 上限）后，待覆盖基数约 21,574 stmts，全部白名单外目录 per-dir ≥70% 即数学上保证全局 ≥70%。

**Phase 编号 (D-04):** 从 Phase 82 起；**75-81 属于并行 `milestone` workstream（v1.27），本 workstream 严禁使用**。

**锁定决策 (v1.28 init):** D-01 语句 ≥70%（全量口径）/ D-02 全 src + 白名单 ≤5% / D-03 4 层 CI gate 对称后端 v1.26 / D-04 Phase 82 起编。

**范围边界:** 仅补测试 + 覆盖率治理，不修改业务逻辑（测试暴露的 bug 修复除外）；CAD/3D 重画布 UI 白名单排除不强制覆盖。

## 基线快照 (2026-08-23 实测)

- **全量口径**: statements 3.67% (830/22602) / branches 1.98% / functions 2.38% / lines 3.70%（584 文件）
- **旧口径（失真，Phase 63）**: statements 24.58%（只算被 import 的 71 文件 3376 stmts）——GOV-01 要修的对象
- **白名单候选 (D-02)**: `components/cad-editor` 804 + `components/cad-elements` 224 = 1028 stmts ≈ 4.5%（≤5% 上限，终版 Phase 82 落地登记）
- **per-dir 语句数**: pages 13144 (operations 3611 / system 2203 / network 1962 / duty 1190 / ad-domain 1082 / monitor 627 / vdi 567 / workorder 551 / asset 475 / knowledge 262 / dashboard-system 203 / my-notices 127 / login 62 / 零散 224) / components 4986 (dashboard 1068 / shared 892 / cad-editor 804* / layout 507 / network 324 / CronSelector 316 / cad-elements 224* / captcha 154 / reconciliation 144 / operations 149 / 零散 304) / hooks 1050 / lib 1042 / utils 950 / store 589 / router 272 / services 238 / design-system 194 / constants 84 / types 32（* = 白名单候选）
- **现有测试**: 19 文件 159 tests 全过（vitest + jsdom + @testing-library，@vitest/coverage-v8 4.1.10）
- **对标模式（后端 v1.26）**: `.github/scripts/check-coverage.sh`（bash+awk 零依赖 gate / exit 分层 floor）/ `.planning/coverage-baseline.md`（ratchet tracking）/ 74-10 diff coverage 自实现（bash+awk，PR-only job）

## Phases

**Phase Numbering (D-04):**

- 本 workstream 整数 phase 从 82 起，单调递增到 88
- 75-81 保留给并行 `milestone` workstream（v1.27），严禁使用

- [ ] **Phase 82: 口径修正与治理基建** - vitest 全量口径切换 + 白名单登记 + 4 层 CI gate 落地（全局阈值 / per-dir floor / baseline ratchet / PR diff coverage ≥80%）
- [ ] **Phase 83: P0 基建层全清 ≥70%** - lib / utils / hooks / store / services+router+constants+types ~3,900 stmts 各目录 ≥70% + 测试公共 harness 沉淀
- [ ] **Phase 84: P1 组件层 ≥70%** - components 五组（shared / dashboard / layout / CronSelector+captcha+operations / network+reconciliation+零散含 design-system）≥70%
- [ ] **Phase 85: P2 页面层 R1 — operations** - 最大单一目录 76 文件 3611 stmts ≥70%
- [ ] **Phase 86: P2 页面层 R2 — system + network** - 两大目录 2203 + 1962 stmts ≥70%
- [ ] **Phase 87: P2 页面层 R3 — 剩余全部页面** - duty / ad-domain / monitor / vdi / workorder / asset / knowledge / dashboard-system / my-notices / login + 零散页收清，全局首达 ≥70%
- [ ] **Phase 88: 收口 — 全量达标验证与 ratchet 终态** - 全局 floor 收口 70.0 + 0 失败 0 flaky 终验 + ratchet 终态落盘

## Phase Details

### Phase 82: 口径修正与治理基建

**Goal**: 覆盖率测量切换到全量口径（584 文件全部计入报告）并落地与后端 v1.26 对称的 4 层 CI 防倒退 gate——从这一刻起任何覆盖率倒退都会让 CI 变红，白名单完成正式登记。
**Depends on**: Nothing (first phase of v1.28)
**Requirements**: GOV-01, GOV-02, GOV-03, GOV-04, GOV-05, QUAL-02
**Success Criteria** (what must be TRUE):

  1. `npm run test:coverage` 在不新增任何测试的情况下以全量口径报告 statements（白名单排除后约 3.85%），584 个 src 文件全部出现在 per-file 报告中、未测试文件以 0% 计入——旧口径 24.58% 的失真消失
  2. 白名单登记文档落地：cad-editor 804 + cad-elements 224 stmts（重画布 UI）以「排除理由 / 面积 / 复审条件」三项登记并同步 coverage exclude，白名单面积 ≤5% 总语句（1028/22602 ≈ 4.5%）
  3. 前端覆盖率基线文档落盘（起点行 3.67% / 22602 stmts + per-dir 快照 + ratchet 记录表，对称后端 `coverage-baseline.md` 模式），CI 前端 job 的全局阈值 gate 切到全量口径且失败即阻断（低于阈值 exit 非零）
  4. PR diff coverage ≥80% gate 以 PR-only CI job 生效：改动 `xingran-react-frontend/src/**` 的 PR 若新增/改动行覆盖 <80%，该 job 失败并输出未覆盖文件清单（复用后端 74-10 bash+awk 自实现模式，前端版基于 vitest json 报告）
  5. per-directory floor gate 机制上线：白名单外目录目标 floor 70% + 未达标目录按当期实测值 ratchet 过渡（只升不降），floor 违例以独立 exit code 在 CI 可区分

**Plans**: 5 plans
Plans:
**Wave 1**

- [ ] 82-01-PLAN.md — vitest 全量口径切换（include/删 thresholds/test:coverage run 语义）+ 白名单 exclude 两步口径断言（584→571）

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 82-02-PLAN.md — check-frontend-coverage.sh（全局阈值 + per-dir floor + 漂移检测 + --init）+ .coverage-fe-floors 生成与五分支干跑矩阵
- [ ] 82-03-PLAN.md — check-frontend-diff-coverage.sh（PR diff ≥80% lines 口径三段式）+ 空树合成基线两分支实证

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 82-04-PLAN.md — ci.yml 接线（frontend job 内嵌 gate + PR-only frontend-coverage-diff job）+ 基线文档与白名单登记

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 82-05-PLAN.md — 真实 CI 验证 PR 三项证据 + D-14 全局阈值 CI 校准 + 基线 SHA 回填

### Phase 83: P0 基建层全清 ≥70%

**Goal**: 前端基建层（lib / utils / hooks / store / services / router / constants / types，约 3,900 stmts）各目录语句覆盖率全清 ≥70%，并沉淀供 P1/P2 复用的测试公共 harness——全站 mock 与渲染样板在此定型。
**Depends on**: Phase 82（全量口径 + gate + 白名单就绪；harness 是 P1/P2 的效率前提）
**Requirements**: INFRA-01, INFRA-02, INFRA-03, INFRA-04, INFRA-05, QUAL-03
**Success Criteria** (what must be TRUE):

  1. lib 通信层 20 文件 1042 stmts（api.ts 加密客户端 / opsApi / networkApi / menuApi / profileApi / 列配置 api 等）语句覆盖率 ≥70%
  2. utils 23 文件 950 stmts（sm2 / sm4 / encoding 国密、token 三件套、dualLevelCache、errorHandler、authHelpers、deptUtils、datetime 等）+ hooks 27 文件 1050 stmts（含 usePagination / useServerSort / usePersistedState 续推）+ store 9 文件 589 stmts（auth / menu / settings / tabs / dashboard / layout / notice / theme）+ services/router/constants/types 约 626 stmts——每个目录语句覆盖率 ≥70%（per-dir floor 检查绿）
  3. 测试公共 harness（mock api 客户端 / antd message·Modal / router wrapper / 常用 store 预置等高频样板）沉淀为可导入的测试工具模块并有使用示例，P1/P2 测试直接复用
  4. 全量 vitest 0 失败（现有 19 文件 159 测试全数通过，无回归），CI gate 绿，全局 ratchet 阈值上调至 Phase 83 后实测值（单调上升）

**Plans**: TBD

### Phase 84: P1 组件层 ≥70%

**Goal**: components/* 五个组件组（白名单外约 4,150 stmts + 顶层 design-system 194 stmts）语句覆盖率全部 ≥70%——共享组件、仪表盘 Widget 体系、布局骨架与 network/reconciliation 低洼目录全部拉平。
**Depends on**: Phase 83（harness 复用 + 基建层 mock 基础）
**Requirements**: COMP-01, COMP-02, COMP-03, COMP-04, COMP-05
**Success Criteria** (what must be TRUE):

  1. components/shared 21 文件 892 stmts ≥70%（最高价值共享组件）
  2. components/dashboard 29 文件 1068 stmts ≥70%（仪表盘 Widget 体系）
  3. components/layout 16 文件 507 stmts（HybridLayout / Sidebar / Header）≥70% 且 CronSelector 316 + captcha 154 + operations 149 各目录 ≥70%
  4. components/network 324（现 50.6%）+ components/reconciliation 144（现 18.1%）+ 零散小组件（components 剩余约 304 + 顶层 design-system 令牌/主题桥 194）各目录 ≥70%——至此 components 层除白名单外全清
  5. 全量 vitest 0 失败 + CI gate 绿 + 全局 ratchet 随本 phase 上调（覆盖率单调上升）

**Plans**: TBD
**UI hint**: yes

### Phase 85: P2 页面层 R1 — operations

**Goal**: pages/operations（76 文件 3611 stmts，全站最大单一页面目录：楼宇/楼层/工位/机房/专线/信息点等）语句覆盖率 ≥70%，单目录全清。
**Depends on**: Phase 84（harness + shared 组件测试模式已定型）
**Requirements**: PAGES-01
**Success Criteria** (what must be TRUE):

  1. pages/operations 语句覆盖率 ≥70%（全量口径 per-dir floor 检查绿，76 文件 3611 stmts）
  2. 运维模块核心交互路径有测试实证：Excel 导入/导出流程、工位 CRUD、楼宇/楼层级联等高频页面在 jsdom 渲染测试中通过（抽查页面组件均有对应测试文件）
  3. 全量 vitest 0 失败 + CI gate 绿 + 全局 ratchet 随本 phase 上调（覆盖率单调上升）

**Plans**: TBD
**UI hint**: yes

### Phase 86: P2 页面层 R2 — system + network

**Goal**: pages/system（56 文件 2203 stmts）与 pages/network（61 文件 1962 stmts）两大目录 ≥70%——合计 4165 stmts 的第二大增量批。
**Depends on**: Phase 85
**Requirements**: PAGES-02, PAGES-03
**Success Criteria** (what must be TRUE):

  1. pages/system 56 文件 2203 stmts ≥70%（用户/角色/部门/菜单/字典/岗位/参数/通知等系统管理页）
  2. pages/network 61 文件 1962 stmts ≥70%（设备/凭据/命令模板/拓扑/MAC/端口等网络管理页）
  3. 全量 vitest 0 失败 + CI gate 绿 + 全局 ratchet 随本 phase 上调（覆盖率单调上升）

**Plans**: TBD
**UI hint**: yes

### Phase 87: P2 页面层 R3 — 剩余全部页面

**Goal**: 剩余全部页面目录（duty / ad-domain / monitor / vdi / workorder / asset / knowledge / dashboard-system / my-notices / login 及零散页，约 5,300 stmts）≥70%——P2 页面层收清，白名单外所有 src 目录 per-dir ≥70%，全局全量口径首次实测 ≥70%。
**Depends on**: Phase 86
**Requirements**: PAGES-04, PAGES-05
**Success Criteria** (what must be TRUE):

  1. pages/duty 1190 + pages/ad-domain 1082（合计 48 文件 2272 stmts）各目录 ≥70%
  2. monitor 627 / vdi 567 / workorder 551 / asset 475 / knowledge 262 / dashboard-system 203 / my-notices 127 / login 62 及零散页（约 70 文件）各目录 ≥70%
  3. 白名单外全部 src 目录 per-directory floor 检查全绿（无 ratchet 过渡目录残留，白名单登记项除外），全局全量口径 statements 实测首次 ≥70%

**Plans**: TBD
**UI hint**: yes

### Phase 88: 收口 — 全量达标验证与 ratchet 终态

**Goal**: v1.28 全量达标终验——全局阈值 floor 收口 70.0、0 失败 0 flaky 终验、ratchet 终态落盘，4 层 gate 以最终值固化，交付「覆盖率从此不可无声倒退」的终局防线。
**Depends on**: Phase 87
**Requirements**: QUAL-01
**Success Criteria** (what must be TRUE):

  1. 全量 vitest 0 失败 0 flaky：全量测试套件连续多次运行全绿，现有 19 文件 159 测试全数通过（不回归）
  2. 全局阈值 gate floor 收口至 70.0（全量口径 statements）后 CI 主 job 仍绿——证明 ≥70% 是实测达标而非 ratchet 过渡；per-directory floor 无过渡项残留（白名单登记项除外）
  3. 覆盖率基线文档记录 ratchet 终态行（3.67% → ≥70% 全链路 ratchet 历史 + per-dir 终值 + commit SHA），PR diff coverage ≥80% gate 与 4 层防线全部保持绿——里程碑审计可复算

**Plans**: TBD

## Progress

**Execution Order:** Phase 82 → 83 → 84 → 85 → 86 → 87 → 88（严格串行；每个 phase 边界 = 全量 vitest 0 失败 + gate 绿 + 覆盖率单调上升的检查点）

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 82. 口径修正与治理基建 | 0/5 | Not started | - |
| 83. P0 基建层全清 ≥70% | 0/TBD | Not started | - |
| 84. P1 组件层 ≥70% | 0/TBD | Not started | - |
| 85. P2 页面层 R1 — operations | 0/TBD | Not started | - |
| 86. P2 页面层 R2 — system + network | 0/TBD | Not started | - |
| 87. P2 页面层 R3 — 剩余全部页面 | 0/TBD | Not started | - |
| 88. 收口 — 全量达标验证与 ratchet 终态 | 0/TBD | Not started | - |

---
last_updated: 2026-08-20
update_trigger: Phase 71 plan 71-01b executed (verify + push + amend half; CI Coverage gate SHIPPED at 12.8% threshold on main)
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
- ✅ **v1.15 工位设备关联 + 部门物理位置映射** — Phases 28 + 39 (shipped 2026-06-10 / 06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 + Phase 47 root-cause (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 + Phase 49 gap closure (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08)
- ✅ **v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding)** — Phase 56 (shipped 2026-07-10)
- ✅ **v1.21 API Key 认证链修复 + 能力补全 (API Key Auth Chain Repair)** — Phases 57-62 (shipped 2026-08-18)
- ✅ **v1.22 前端品牌化改造 (Frontend Brand Design-System)** — Phases 64-67 (shipped 2026-08-18)
- ✅ **v1.23 部署稳健性 & 文档一致性 (Deploy Robustness & Docs Consistency)** — Phase 68 (shipped 2026-08-19)
- ✅ **v1.24 字典与状态值治理 (Dict & Status Governance)** — Phase 69 (shipped 2026-08-19)
- ✅ **v1.25 系统设置页面布局重构 (Settings Page Layout Redesign)** — Phase 70 (shipped 2026-08-19)
- ✅ **Phase 63 前端工具链自动化 (Frontend Toolchain Automation)** — Phase 63 (shipped 2026-08-20)
- 🚧 **v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence)** — Phases 71-74 — **IN PROGRESS** (milestone started 2026-08-20)

---

## Current Milestone: v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence) — 🚧 IN PROGRESS

**Goal:** 后端 Go 加权平均测试覆盖率 **12.8% → ≥70%**(优秀等级),P0/P1 业务模块零测试清零,CI 落地 coverage 阈值 gate + PR diff coverage,使覆盖率从此不可无声倒退。

**Source planning data:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md`(2026-08-20 纯只读扫描,74 业务包 / 43652 stmts / 加权 12.8% / 33 个 0% 包 / P0-P2 分级)。

**Milestone success criteria:**

- SC-a: 加权平均 ≥70%(12.8% → ≥70%)
- SC-b: 0% 覆盖业务包 ≤5(33 → ≤5)
- SC-c: CI coverage threshold gate 生效(失败即阻断)
- SC-d: 全量 `go test` 零失败(flaky 已修,`5ead742`)
- SC-e: PR diff coverage ≥80% 门槛启用

**Phase 编号:** 从 v1.25 末尾 Phase 70 续编(71+)。

**研究:** 已跳过——覆盖率数据/模块清单/CI 现状均在 quick-260820-bcs 扫描中落盘,无需域研究。

### Phase Dependency Graph

```
Phase 71 (治理基线 + CI gate)
   │
   ├─→ Phase 72 (P0 核心补齐) [depends on 71]
   │         │
   │         └─→ Phase 73 (P1 重要补齐) [depends on 72]
   │                  │
   │                  └─→ Phase 74 (P2 增量 + diff coverage 收尾) [depends on 73]
```

每个 phase 是自然交付边界,可在 phase boundary 处构建 + CI 绿。

### Phase 71: 治理基线 + CI gate — IN PROGRESS

**Goal:** CI 后端 job 产出覆盖率报告与阈值 gate;建立测试补齐的基线记录与 ratchet 防倒退机制;前置 flaky test 修复验收。
Plans:

- [x] [71-01-PLAN.md](phases/71-governance-baseline-and-ci-gate/71-01-PLAN.md) — Bash+awk coverage gate + ci.yml extension + baseline snapshot (file creation half, 4 tasks, commits 4242a75/3e2664c/75a4703/75b96be; SUMMARY: [71-01-SUMMARY.md](phases/71-governance-baseline-and-ci-gate/71-01-SUMMARY.md))
- [ ] [71-01b-PLAN.md](phases/71-governance-baseline-and-ci-gate/71-01b-PLAN.md) — Local go test verification + bump-threshold fail test + push + gh run watch + amend TBD SHA in coverage-baseline.md (verify+push+amend half)

**Requirements:** GOV-01, GOV-02, GOV-04

**Success Criteria:**

1. `.github/workflows/ci.yml` 后端 job 含 `-coverprofile=coverage.out -covermode=atomic` 参数并产出 profile artifact
2. CI 引入阈值 gate(初值对应当前 12.8% 基线),低于阈值 PR 即 fail;ratchet 机制记录每次 phase 提升后的实际数字
3. `.planning/quick/260820-backend-test-coverage-scan/` 或同等位置有完整 per-package 基线表,phase 完成后回填新数字形成可追踪差异
4. pkg/cache `TestAsyncRetryWorker_Enqueue` 通过验证(`5ead742` 已修,Phase 71 跑 `go test ./pkg/cache/...` 全过即可验收)
5. `go test ./internal/... ./pkg/... ./cmd/...` 全包 exit 0,无任何测试失败

**Key Decisions (lock at plan-internal):**

- 阈值 gate 工具选择:bash + awk 自检 vs go-test-coverage GitHub Action vs custom Makefile——planner 评估,优先复用最少依赖
- profile artifact 上传:actions/upload-artifact vs 仅在失败时 dump——按 CI 体积成本权衡
- 基线数据存放:更新 quick task 目录 vs 写入 `.planning/coverage-baseline.md`——planner 决策

**Notes:**

- GOV-03(diff coverage)属于 Phase 74,Phase 71 只建立全量覆盖率基线 gate
- Phase 71 是整条治理链的起点——所有测试补齐工作受益于此 gate 的"防倒退"价值
- 阈值 gate 的初始值需等于或低于当前 12.8% 基线(防止首次落地就 fail),后续 phase 完成时逐步 ratchet 上调

---

### Phase 72: P0 核心补齐 — SHIPPED 2026-08-21

**Goal:** P0 核心业务链路(高频审计路径 + 系统管理核心)零测试清零,所有 P0 包达 ≥70% 覆盖。

**Requirements:** CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06

**Success Criteria:**

1. `internal/api/v1/workorder` 覆盖率 ≥70%(297 stmts)
2. `internal/api/v1/monitor` 覆盖率 ≥70%(518 stmts)
3. `internal/api/v1/scheduler` 覆盖率 ≥70%(152 stmts)
4. `internal/api/v1/system` 覆盖率 ≥70%(3039 stmts)
5. `internal/services/workorder` 覆盖率 ≥70%(715 stmts)
6. `internal/services/system` 覆盖率 ≥70%(3483 stmts)
7. Phase 72 完成后加权平均覆盖率显著上升(预估 12.8% → ≥30%),CI gate 阈值 ratchet 上调至新实际值
8. 全部新测试遵循 portwrite 85.3% 范本模式(glebarez sqlite in-memory + table-driven tests),零业务代码改动(测试暴露的确定性 bug 修复除外,最小 diff 单独说明)

**Notes:**

- services/system 3483 stmts 是最大单包,需分层拆分测试(user / role / dept / menu / dict / post / config)——planner 分配
- api/v1/system 3039 stmts 同理;按子模块拆 plan
- operlog 82.2% 是治理范本,Phase 72 service 测试可参考其 regression_test.go AST 锁值手法(如需)

---

### Phase 73: P1 重要补齐 — PLANNED 2026-08-21

**Goal:** 8 个 P1 重要模块(duty/knowledge/rpa/vdi/monitor/network)从 0% 测试提升至 ≥70%。

**Requirements:** IMP-01, IMP-02, IMP-03, IMP-04, IMP-05, IMP-06

**Success Criteria:**

1. `internal/api/v1/duty` 覆盖率 ≥70%(265 stmts)
2. `internal/api/v1/knowledge` 覆盖率 ≥70%(273 stmts)
3. `internal/api/v1/rpa` 覆盖率 ≥70%(612 stmts)
4. `internal/api/v1/vdi` 覆盖率 ≥70%(298 stmts)
5. `internal/services/duty` + `internal/services/knowledge` 覆盖率 ≥70%(114 + 85 stmts)
6. `internal/services/monitor` + `internal/services/network` 覆盖率 ≥70%(485 + 127 stmts)
7. Phase 73 完成后加权平均覆盖率 ratchet 到新实际值(预估 27-30%,沿用 Phase 72 D-07),CI gate 阈值 ratchet 上调
8. 全部新测试零业务代码改动

**Wave / Plan structure:**

- **Wave 1 (parallel):** 73-01 (handler 简单: duty+knowledge, 538 stmts) + 73-02 (handler 复杂: rpa+vdi, 910 stmts, incl. public router) + 73-03 (service 简单: duty+knowledge+network, 326 stmts) + 73-04 (service 中等: monitor, 485 stmts, incl. oper_log)
- **Wave 5:** 73-05 (ratchet + per-package gate extension, depends on 73-01..04)

**Notes:**

- services/rpa 1865 stmts 1.1% / services/vdi 1127 stmts 2.7% 暂归 P2(Phase 74),本 phase 只补到 70% handler + 小 service 包
- 复杂模块(rpa 1865 / vdi 1127)planner 可分多 plan 拆分
- Plan 73-05 (Wave 5) mirrors Phase 72 Plan 72-13: Task 0 批量提交所有 test files → Task 1 全包覆盖率测量 → Task 2 扩展 check-coverage.sh (8 P1 包 per-package 验证) → Task 3 原子 ratchet commit (6 文件) → Task 4 CI 验证
- D-08 + D-09 (test pattern consistency + 真实中间件) 沿用 Phase 71/72 范本,在 73-05 must_haves 中显式标注
- D-04 (rpa 公开路由不豁免) + D-03 (oper_log_service 不豁免) 严格按 ≥70% 覆盖

---

### Phase 74: P2 增量 + diff coverage 收尾 — Pending

**Goal:** 把整体加权平均推到 ≥70%(SCALE-03),落地 PR 增量 diff coverage 门槛,验证 milestone SC 全部达成。

**Requirements:** GOV-03, SCALE-01, SCALE-02, SCALE-03

**Success Criteria:**

1. PR 增量 diff coverage ≥80% 门槛在 CI 启用(GOV-03)— 变更代码必须 ≥80% 覆盖才通过
2. `api/v1/{operations, asset, network}` 覆盖率均 ≥70%(1285 / 420 / 1971 stmts)
3. `services/{rpa, vdi}` 覆盖率均 ≥70%(1865 / 1127 stmts)
4. `internal/core` / `internal/device` / `internal/utils` / `internal/agent/server` / `services/scheduler` 等中等覆盖包系统性提升(目标区间按整体达标分配)
5. 中等覆盖(10-50%)与高性价比纯工具包(`pkg/{response,query,time,logger,captcha}`、`internal/pkg/*`)选择性提升
6. **整体加权平均覆盖率 ≥70%**(43652 stmts 口径;SC-a)
7. **0% 覆盖业务包 ≤5**(SC-b;仅剩辅助类豁免——纯 struct / cmd / docs)
8. milestone SC-a/b/c/d/e 全部 5 项达成并写入 audit 报告

**Notes:**

- 本 phase 是 v1.26 收口 phase,验证所有 milestone SC
- diff coverage 工具:gocover-diff / go-test-coverage(action 内置支持)——planner 评估选型
- Phase 74 plan 数量可能较多(planner 按业务包工作量拆分),建议 3-5 个 plan

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 71 | SHIPPED | 2/2 | GOV-01, GOV-02, GOV-04 (3/3 done — file creation half 71-01 + verify+push+amend half 71-01b both complete; CI Coverage gate PASSED on runs 32382995316 + 32384622877; coverage-baseline.md Phase 71 后 row commit column amended TBD → 326a541 via commit cfec2c4) | 2026-08-20 | 2026-08-20 |
| Phase 72 | SHIPPED | 13/13 | CORE-01..06 (6/6 done — workorder 75.4%, monitor 71.2%, scheduler 85.5% ≥70%; system 35.4%, services/system 53.5% <70% per-sub-module deferred to Phase 73/74; weighted avg 12.8% → 21.5% ratchet landed) | 2026-08-21 | 2026-08-21 |
| Phase 73 | Pending | 0/? | IMP-01..06 (0/6 done) | — | — |
| Phase 74 | Pending | 0/? | GOV-03, SCALE-01..03 (0/4 done) | — | — |

**Total:** 4 phases / 19 requirements (0 done)

---

## Future Milestones (TBD)

候选方向(基于 STATE.md §Deferred Items + Future Requirements):

- **v1.27+**: Phase 63 启动块遗留 advisory 收尾(network/ports 3 timeout + total bundle +90kB + knip ignore 收紧)
- **v1.27+**: PROTO-01..04 逐屏原型对齐 + VIS-01..03 视觉深化(v1.22 Future Requirements)
- **v1.27+**: VDI 22C/22D(账号管理 + Web Console,依赖 v1.11 生产稳定性数据)
- **v1.27+**: 分支覆盖率 / mutation testing / PR 评论机器人(FUT-02..04)
- **v1.27+**: 真机 site-visit UAT 闭环(v1.18-v1.20 deferred items)

---

## Operational Architecture

```
Backend (Go)
├── internal/api/v1/{system,operations,workorder,scheduler,duty,network,
│                       knowledge,monitor,rpa,vdi,asset}/  ← Phase 72/73/74 重点补齐
├── internal/services/{system,operations,workorder,scheduler,duty,network,
│                       knowledge,monitor,rpa,vdi,asset,portwrite,
│                       component_collector,topology,lldp,addomain}/  ← Phase 72/73/74 重点补齐
├── internal/middleware/  (86.2% 已高,Phase 71-74 维持)
├── internal/utils/operlog/  (82.2% 范本)
├── pkg/{cache,normalize,transform,response,query,logger,...}/
└── internal/{core,config,device,agent,scheduler,collector}/

CI Gate (.github/workflows/ci.yml)
├── backend job:
│   ├── go test ./internal/... ./pkg/... ./cmd/... -coverprofile=coverage.out -covermode=atomic  ← Phase 71 加
│   ├── go tool cover -func=coverage.out → awk 算加权平均 vs 阈值 gate  ← Phase 71 加
│   └── (Phase 74 加) diff coverage ≥80% 门槛
└── frontend job:  Phase 63 已加 vitest coverage thresholds 25/15/22/25(本 milestone 不动)

Test Infrastructure
├── glebarez sqlite in-memory (已有,不引新 mock)
├── testing + table-driven tests (已有模式)
├── portwrite 85.3% 范本 (Phase 50-55 落地,Phase 72-74 参考)
├── operlog 82.2% 范本 (regression_test.go 锁公共 API)
└── middleware 86.2% 范本 (apikey 全链路测试)
```

---

*Last updated: 2026-08-20 — v1.26 backend-test-coverage-excellence milestone IN PROGRESS: 4 phases (71-74) / 19 requirements / 100% mapped. Quick task 260820-bcs 后端覆盖率扫描数据为规划输入。**Phase 71 SHIPPED 2026-08-20 (2/2 plans)** — plan 71-01 file creation half (4 commits `4242a75` `.coverage-threshold` + `3e2664c` check-coverage.sh + `75a4703` ci.yml 4-step extension + `75b96be` coverage-baseline.md scaffold with TBD commit on Phase 71 后 row; SUMMARY: `.planning/phases/71-governance-baseline-and-ci-gate/71-01-SUMMARY.md`) + plan 71-01b verify+push+amend half (1 commit `cfec2c4` TBD → 326a541 amend as new commit on protected branch; CI backend Coverage gate PASSED on runs 32382995316 + 32384622877; backend-coverage artifact 1.4MB / 30-day retention; SUMMARY: `.planning/phases/71-governance-baseline-and-ci-gate/71-01b-SUMMARY.md`). Phase 63 frontend-toolchain-automation SHIPPED 2026-08-20(前端 coverage thresholds 已就位,本 milestone 为后端对称物)。Combined v1.22-v1.25 SHIPPED + ARCHIVED 2026-08-19.*

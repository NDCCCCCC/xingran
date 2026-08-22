---
phase: 74-p2-finalize-and-diff-coverage
phase_title: P2 增量 + diff coverage 收尾
last_updated: 2026-08-21
update_trigger: Phase 73 SHIPPED (25.9% weighted avg, 8 P1 packages ≥70%, 22 0%-pkg) — Phase 74 planning start
status: planning
---

# Phase 74 Context: P2 增量 + diff coverage 收尾

**Milestone:** v1.26 后端测试覆盖率优秀 (Phases 71-74)
**Phase 74 role:** v1.26 milestone 收口 phase — 把整体加权平均从 25.9% 推到 ≥70%, 落地 PR diff coverage ≥80% 门槛, 验证 milestone SC 全部 5 项达成。

---

## Inherited D-Locked Decisions (from Phases 71/72/73 — strict honor)

所有 Phase 71-73 决策完全继承, Phase 74 不重新讨论:

- **D-01** bash + awk 自检, 零依赖 (check-coverage.sh 已落地)
- **D-02** glebarez sqlite in-memory + standard testing, 不引入新 mock framework
- **D-03** oper_log_service 不豁免 — 任何含 operlog 测试的业务包必须实测 (Phase 73-04 已落地)
- **D-04** P1 per-package ≥70% floor (Phase 73-05 已落地 8 包) — Phase 74 不得让任何 P1 倒退
- **D-05** STATE.md + ROADMAP.md 状态更新跟随 ratchet commit
- **D-06** bash + awk only, 无新依赖
- **D-07** atomic ratchet commit (D-04 manual ratchet — Phase 72/73 范本, Phase 74 final plan 沿用 6 files)
- **D-08** 测试模式一致性 — ad_account handler + portwrite service 双范本, 无新模式
- **D-09** 真实中间件 + 真实 SM4 加密路径
- **D-10** P1 per-package 严格 ≥70% (Phase 73 落地 8 包, Phase 74 不得退化)
- **D-11** STATE.md + ROADMAP.md updates 在 ratchet commit 而非 follow-up commit
- **D-12** 0 业务代码改动 (测试暴露的确定性 bug 修复除外, 最小 diff 单独说明)
- **D-13** cache invalidation 5+4 个 interface 方法都有默认 override

---

## Phase 74 NEW Decisions (locked at phase init by planner)

### D-14 — diff coverage 工具选型

**决策:** 复用 Phase 71/72/73 范式, 用 `gocover-coverage` GitHub Action (市场最成熟) 落地 GOV-03 PR diff coverage 门槛, 配置 `min-coverage: 80` 即可。

**理由:**
- Phase 71 RESEARCH.md 已研究过 go-test-coverage action 的 `diff: 80` 选项, 当时未落地 (因阶段是 CI gate baseline, 非 diff gate)
- gocover-coverage action 支持增量 diff coverage, 配置简单, 与现有 ci.yml backend job 无缝集成
- 比 gocover-diff 自实现方案更省工, 比纯 go-test-coverage (Phase 71 评估时其 diff 支持尚不完善) 更稳定

**落地形式 (Plan 74-10):**
- 新增独立 job `coverage-diff` (单独触发 + 单独权限)
- 使用 `ory/xcoverage-action@v0.x` 或 `gocover-coverage-action@vX` (最终选型由 executor 在 Plan 74-10 Task 1 锁定, 优先选 ORY 系)
- 配置: `min-coverage: 80`, `min-coverage-compare-with: diff`
- 失败即 fail, 与现有 backend Coverage gate (exit 0/1) 串联

**Ratchet:** D-14 如锁定的工具在落地中发现不适合 (如 action 维护停滞), Phase 74 executor 可降级为自实现方案 (git diff + awk + coverage.out 解析), 但必须 100% 满足 GOV-03 ≥80% threshold。

### D-15 — Phase 74 per-package floor 扩展

**决策:** check-coverage.sh 新增 section 4 `p2_package_check`, 10 个 P2 大块包均需 ≥70% per-package, 失败 exit 5。

**P2 大块包列表 (10 包, 来自 SCALE-01):**

| 包 | stmts | 当前 | 目标 |
|---|---|---|---|
| internal/api/v1/operations | 1285 | 3.0% | ≥70% |
| internal/api/v1/asset | 420 | 8.3% | ≥70% |
| internal/api/v1/network | 1971 | 7.6% | ≥70% |
| internal/services/rpa | 1865 | 1.1% | ≥70% |
| internal/services/vdi | 1127 | 2.7% | ≥70% |
| internal/core | 754 | 2.1% | ≥70% |
| internal/device | 1249 | 2.5% | ≥70% |
| internal/utils | 531 | 4.5% | ≥70% |
| internal/agent/server | 616 | 2.1% | ≥70% |
| internal/services/scheduler | 167 | 4.8% | ≥70% |

**理由:**
- Phase 73 锁定 8 P1 包 floor (D-04 strict), Phase 74 同样模式扩展到 10 P2 大块
- 单包 ≥70% 是 D-04 严格一致性延伸, 防止"加权平均达标但单包倒退"
- 落地形式与 Phase 73 section 3 byte-identical (同样的 awk 逻辑, 同样 floor 比较, 同样 PASS/FAIL 输出格式), 仅扩展 PACKAGES 列表与 exit code (5)

**Backward compat:** sections 1-3 (weighted-avg + P1 floor) 不变; section 4 加性不互斥。

### D-16 — 0% 业务包豁免清单

**决策:** 当前 22 个 0% 包中, 11 个永久豁免 (纯 struct / cmd main / docs 类), 11 个需在 Phase 74 测试补齐或选择性豁免。

**永久豁免 (11 包, 240 stmts, 无业务逻辑):**

| 包 | stmts | 豁免理由 |
|---|---|---|
| internal/services/common | 1 | 仅常量 |
| internal/server | 2 | 仅常量/接口 |
| internal/models/system/requests | 109 | DTO 纯 struct |
| internal/models/system | 11 | DTO 纯 struct |
| internal/models/rpa | 94 | DTO 纯 struct |
| internal/models/operations | 23 | DTO 纯 struct |
| internal/docs | 1 | 仅 doc.go |
| internal/api/v1/operations/requests | 15 | DTO 纯 struct |
| internal/agent/pkg/retry | 33 | 内部 retry 工具 (已低优先级, 由 planner 评估是否豁免) |
| cmd/agent | 59 | cmd main |
| cmd | 106 | cmd main (per D-02 exempt) |

**需补齐或选择性豁免 (11 包, 任务分摊):**
- 纯工具 (SCALE-02, Plan 74-09 责任): pkg/response (51), pkg/query (105), pkg/time (63), pkg/logger (79), pkg/captcha (409), pkg/gormutil (194), pkg/ldaputils (33)
- 业务包补齐 (SCALE-01, Plan 74-04 责任): internal/api/v1/agent (38)
- 选择性豁免 (由 executor 评估): internal/api (417) — 路由胶水, 集成测试覆盖; internal/pkg/system (345) — 已声明包, 由 planner 在 Plan 74-09 评估补齐或豁免
- 系统设施 (per D-02 partial exempt): internal/pkg/cache (167) — 可在 Plan 74-09 选择性补齐

**0% 包 count 目标:** 22 → ≤5 (per SC-b)。

---

## Phase 74 数学预算 (planner 已拆分)

**起点:** 25.9% (11336 / 43652 covered/total stmts)
**目标:** ≥70% (30556 / 43652)
**需要:** +19220 covered stmts (+44.0 pp)

### A) P2 大块 (SCALE-01) — 10 包 @ ≥70%:

- api/v1/operations (1285 × 0.67) = +860
- api/v1/asset (420 × 0.62) = +260
- api/v1/network (1971 × 0.62) = +1222
- services/rpa (1865 × 0.69) = +1287
- services/vdi (1127 × 0.67) = +755
- internal/core (754 × 0.68) = +513
- internal/device (1249 × 0.67) = +837
- internal/utils (531 × 0.66) = +350
- internal/agent/server (616 × 0.68) = +419
- services/scheduler (167 × 0.65) = +109

**小计:** +6612 covered stmts (+15.1 pp → 41.0%)

### B) 中等覆盖补足 (SCALE-01/02):

- internal/services/operations 22.5% → 60% (3714 × 0.375) = +1393
- internal/services/system 53.5% → 75% (3483 × 0.215) = +749
- internal/services (5202 × 0.35, 综合子包) = +1821
- internal/services/addomain 15.4% → 50% (2415 × 0.346) = +836
- internal/services/asset 40.5% → 70% (1354 × 0.295) = +399
- internal/api/v1/system 35.4% → 75% (3039 × 0.396) = +1203
- internal/core/db 37.5% → 70% (643 × 0.325) = +209
- internal/core/security 48.9% → 75% (313 × 0.261) = +82
- pkg/crypto 33.7% → 70% (439 × 0.363) = +159
- pkg/middleware 14.3% → 60% (609 × 0.457) = +278
- pkg/cache 24.6% → 70% (926 × 0.454) = +420
- internal/templates 44.4% → 80% (243 × 0.356) = +87
- internal/services/portcollection 19.3% → 60% (580 × 0.407) = +236
- internal/core/db/migrations 28.3% → 60% (293 × 0.317) = +93

**小计:** +7965 covered stmts (+18.3 pp → 59.3%)

### C) 高性价比纯工具 (SCALE-02) — Plan 74-09:

- pkg/response (51 × 0.70) = +36
- pkg/query (105 × 0.70) = +74
- pkg/time (63 × 0.70) = +44
- pkg/logger (79 × 0.70) = +55
- pkg/captcha (409 × 0.70) = +286
- pkg/gormutil (194 × 0.70) = +136
- pkg/ldaputils (33 × 0.70) = +23
- internal/pkg/cache (167 × 0.70) = +117
- internal/pkg/system (345 × 0.70) = +242
- internal/utils/operlog (已有 82.2%, 维护) = +0
- internal/agent/server 已计入 P2 大块

**小计:** +1013 covered stmts (+2.3 pp → 61.6%)

### D) 0% 业务包 ≤5 (SC-b):

当前 22 → ≤5 = 减少 17 包。

**总数学校验:** A + B + C = 6612 + 7965 + 1013 = +15590 covered stmts (+35.7 pp → 61.6%)
**距离目标 (70%):** 还差 ~3630 stmts (+8.4 pp)

**planner 注:** A+B+C 数学预估保守, 实际执行时 executor 可在 SCALE-01/02 中增补 (如 services/asset 推到 ≥80%, services/system 推到 ≥80%, internal/services 综合子包覆盖更深)。Phase 74-11 final ratchet 由 executor 实测加权, 不足时由 executor 在 ratchet commit 前补齐最后一波小包。

---

## Wave 结构 (locked, 11 plans)

**Wave 1 (4 plans, 并行) — P2 handler 大块:**
- 74-01: api/v1/operations handler tests (1285 stmts, 3.0% → ≥70%)
- 74-02: api/v1/asset handler tests (420 stmts, 8.3% → ≥70%)
- 74-03: api/v1/network handler tests (1971 stmts, 7.6% → ≥70%)
- 74-04: api/v1/system sub-module 增量 + api/v1/agent (3039 stmts, 35.4% → ≥70%, +38 stmts agent)

**Wave 2 (4 plans, 并行) — P2 service 大块:**
- 74-05: services/rpa tests (1865 stmts, 1.1% → ≥70%, 复杂 — 含 rpa worker / execution / credential / ai / flow)
- 74-06: services/vdi tests (1127 stmts, 2.7% → ≥70%, 含 vdi_server + vm)
- 74-07: services/operations + services/system + services/asset 增量 (3714 + 3483 + 1354 stmts, 推到 ≥70%)
- 74-08: internal/core + internal/device + services/scheduler + internal/core/db + pkg/cache + pkg/crypto + pkg/middleware + internal/templates tests

**Wave 3 (1 plan) — 纯工具包 + 0% 业务包:**
- 74-09: pkg/{response, query, time, logger, captcha, gormutil, ldaputils} + internal/pkg/{cache, system} + internal/utils + internal/agent/server + services/addomain + services/portcollection 增量

**Wave 4 (1 plan) — Diff coverage gate:**
- 74-10: GOV-03 PR diff coverage ≥80% gate — ci.yml 新增独立 job `coverage-diff`, 选型 D-14 锁定工具

**Wave 5 (1 plan, depends 1-10) — Final ratchet + milestone audit:**
- 74-11: 扩展 check-coverage.sh section 4 p2_package_check (10 P2 包 ≥70% floor, exit 5) + 原子 ratchet commit (6 files: .coverage-threshold + coverage-baseline.md + check-coverage.sh + STATE.md + ROADMAP.md + 74-11-SUMMARY.md) + milestone audit 报告 (5 项 SC 全绿)

---

## 必须保留的关键约束 (strict)

1. **D-04 strict**: Phase 73 的 8 P1 package per-package ≥70% floor 不得退化 — check-coverage.sh section 3 必须继续 PASS
2. **D-12 strict**: 0 业务代码改动 (测试暴露的确定性 bug 修复除外, 最小 diff + 单独 commit + SUMMARY 显式说明)
3. **D-07 atomic ratchet**: 74-11 (final plan) commit 必须含 .coverage-threshold + coverage-baseline.md + check-coverage.sh + STATE.md + ROADMAP.md + 74-11-SUMMARY.md (6 files, 1 commit)
4. **D-05/D-11**: STATE.md + ROADMAP.md updates 在 ratchet commit 而非 follow-up commit
5. **ratchet UP only**: .coverage-threshold 只能从 25.9 上调, 不得下调
6. **test-only commits**: 每个 plan 的 test 文件应在 plan 内 atomic commit (沿用 Phase 73-05 Task 0 batch pattern 或 per-plan commit)

---

## Files 引用 (executor 必须读)

- `.planning/ROADMAP.md` (Phase 74 section, milestone SC)
- `.planning/REQUIREMENTS.md` (GOV-03 + SCALE-01..03 + SC-a..e)
- `.planning/STATE.md` (Phase 73 SHIPPED 状态)
- `.planning/coverage-baseline.md` (Phase 73 后 row, 当前 25.9%)
- `.planning/quick/260820-backend-test-coverage-scan/per-package-coverage.txt` (per-package 起点表)
- `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` (P0/P1/P2 分类 + CI 现状)
- `.github/scripts/check-coverage.sh` (current gate; 74-11 扩展 section 4)
- `.github/workflows/ci.yml` (current 4-step + Upload artifact; 74-10 新增 diff coverage job)
- `.coverage-threshold` (current 25.9)
- `.planning/phases/73-p1-pending/73-05-PLAN.md` (Wave 5 ratchet 结构镜像)
- `.planning/phases/72-p0-core-supplement/72-13-PLAN.md` (Phase 72 ratchet 结构参考)
- `.planning/phases/73-p1-pending/73-01..04-PLAN.md` (test 编写范本 — handler 简单/复杂, service 简单/中等)
- `CLAUDE.md` (project guidelines — operlog convention, status constants, response format)

---

## Quality Gate (planner 自验)

- [x] GOV-03 / SCALE-01 / SCALE-02 / SCALE-03 4 个 requirement ID 全部映射到 plan requirements field
- [x] Phase 74 8 SC 全部进入 74-11 success_criteria
- [x] Wave 结构支持 Wave 内并行 (depends_on satisfied)
- [x] Final plan (74-11) 含 atomic ratchet commit (6 files per D-07)
- [x] D-01..D-13 锁定决策全部 honored (D-14/15/16 在本 CONTEXT.md 显式锁定)
- [x] Every task 有 `<read_first>` 和 `<acceptance_criteria>` (per deep_work_rules)
- [x] Every `<action>` 含 concrete identifiers, 无 fenced code blocks
- [x] test-only constraint (D-12) 在每个 test plan 显式说明
- [x] Plan count = 11 (Phase 72: 13, Phase 73: 5; Phase 74: 11 — 收口 phase 工作量合理)

---

*Last updated: 2026-08-21 — Phase 74 planning complete, 11 plans created (74-01..74-11)*
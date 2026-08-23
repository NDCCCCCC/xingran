---
phase: 82-coverage-caliber-and-governance
plan: 05
subsystem: testing
tags: [coverage, ci-gate, ci-verification, pr-diff-coverage, ratchet, d14-calibration, validation-evidence]

# Dependency graph
requires:
  - ci.yml frontend job 内嵌 Coverage gate + Upload coverage artifact（82-04 提供）
  - ci.yml frontend-coverage-diff PR-only job + download-artifact with.path LCA 还原（82-04 提供，T-82-04-05 mitigation）
  - .coverage-fe-floors GLOBAL 3.8 + 28 目录 floor（82-02 提供）
  - check-frontend-coverage.sh / check-frontend-diff-coverage.sh CLI 契约（82-02/82-03 提供）
  - .planning/frontend-coverage-baseline.md ratchet 表 + TBD 起点 commit 列（82-04 提供，本 plan 回填定稿）
provides:
  - Phase 82 真实 CI 验证证据链（PR 双绿 + diff gate 实读 profile 防御断言 + main push diff job skipped）
  - 基线文档 ratchet 起点 commit 列定稿（bddb2fc）+ 「Phase 82 CI 实测读数」记录行（8c7b69f）
  - D-14 校准判定记录：CI 3.85% == 本地 3.85% 零漂移，无需校准
affects: [/gsd:verify-work（VALIDATION Phase gate 达成的证据基础）]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PR-only job 的 push run 断言口径：job 级 if 为假时 GitHub 仍把 job 列在 run 的 job 列表并标记 conclusion=skipped——断言必须用 skipped 而非「不在列表」（frontend-coverage-diff 在 main push run 32643452003 即此表现，与 backend coverage-diff 同款）"
    - "diff gate 实读 profile 的日志级防御断言：grep 日志无 json 缺失软跳过提示（`was Test step skipped` 0 次）且含空 diff 软通过行（`no testable .ts/.tsx lines changed — PASS`）——job 绿不能证明 gate 真跑，日志语义才能（T-82-04-05 真实 CI 复核）"
    - "docs-only 验证 PR 走 diff gate 空软通过分支：只动 .planning/ 不触 xingran-react-frontend/src，FAIL 分支已由 82-03 空树合成基线证毕，勿为测 FAIL 制造红 CI"

key-files:
  created:
    - .planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-05-SUMMARY.md
  modified:
    - .planning/frontend-coverage-baseline.md

key-decisions:
  - "D-14 校准未触发：CI 实测 3.85% == 本地 3.85% 零漂移（Ubuntu runner 与本地 vitest 同读数），≥ 阈值 3.8，GLOBAL 维持 3.8，.coverage-fe-floors 零改动（起点 commit bddb2fc 携带其最终态）"
  - "ratchet 起点行回填 bddb2fc（82-04 落盘 commit）而非 8c7b69f：起点是基线数据首次落盘的 commit；8c7b69f 记录在同一文档新增的「Phase 82 CI 实测读数」行，职责分明"
  - "D-04 观察项不调整：Test (coverage) CI 实际耗时 41 秒（13:23:20→13:24:01）vs 15 分钟 timeout，余量充足"

patterns-established:
  - "验证 PR 三步采集模板：① PR run 日志防御断言（无软跳过 + 有 diff gate 输出行）→ ② main push run conclusion==skipped 断言 → ③ gate PASS 读数回填基线文档同 commit 落盘"

requirements-completed: [GOV-03, GOV-04]

# Metrics
duration: 66min（跨 3 段执行：PR #5 作废 + PR #6 正式验证 + 回填续接；含用户确认与 merge 等待）
completed: 2026-08-23
---

# Phase 82 Plan 05: 真实 CI 验证 PR 三项证据 + D-14 校准 + SHA 回填 Summary

**验证 PR #6（docs-only，squash merge 8c7b69f）三项 CI 证据齐备——PR run 32642143749 frontend/frontend-coverage-diff 双绿且 diff gate 日志实证实读 artifact 还原后的 profile（软跳过提示 0 次 + 空 diff 软通过 1 次）；main push run 32643452003 frontend=success 且 frontend-coverage-diff=skipped（PR-only 证明）；gate 读数 PASS: weighted avg 3.85% >= threshold 3.80%（28/28 目录）；D-14 校准未触发（CI 与本地零漂移），基线文档起点 SHA 回填 bddb2fc + CI 实测读数行 8c7b69f 定稿**

## Performance

- **Duration:** 66 min（含 3 次用户确认等待与 PR merge 等待；实际执行 ~35 min）
- **时间线:** PR #5 作废弃用（closed 13:20:38Z）→ PR #6 创建并跑绿 → merge 8c7b69f（13:48:02Z，用户本人完成）→ main push run 32643452003 确认 → 回填 commit 2a53443 + 簿记收尾（14:21Z 起）
- **Tasks:** 2（Task 1 auto 完成；Task 2 checkpoint:human-verify 待用户终检）
- **Files modified:** 1（.planning/frontend-coverage-baseline.md）+ SUMMARY 1

## 三项 CI 证据（acceptance criteria 逐项）

### 证据一：验证 PR 双绿 + diff gate 实读 profile（防御断言）

- **PR**: https://github.com/NDCCCCCC/xingran/pull/6 — `chore(82): CI gate verification`，docs-only 单 commit `4d1361b`（基线文档 +2 行占位，不触 `xingran-react-frontend/src`），**squash merge = 8c7b69f（2026-08-23T13:48:02Z，用户本人 merge）**
- **PR run 32642143749**（分支 chore/82-ci-gate-verify-v2——PR #5 污染作废后从 main 重建的新分支名；计划 verify 块字面查旧分支名，续接执行时按实际正式分支替换执行，断言原样）：
  - frontend job **pass**，`Coverage gate` 步骤输出：`TOTAL 21574 830 3.85%` / `PASS: weighted avg 3.85% >= threshold 3.80%` / `PASS: per-dir floor gate — 28/28 directories >= floor`
  - frontend-coverage-diff job **pass**，日志防御断言（T-82-05-04 / T-82-04-05 真实 CI 复核）：
    - json 缺失软跳过提示 **0 次**（`Test step skipped` / `skipping gate` 均未出现）——download-artifact `with.path: xingran-react-frontend/coverage` 的 LCA 还原生效
    - 空 diff 软通过 **1 次**：`diff-coverage: no testable .ts/.tsx lines changed vs origin/main — PASS (nothing to gate)`——docs-only PR 的预期分支，证明 gate 真实读到了 artifact 还原后的 profile 而非静默空转

### 证据二：main push run 上 PR-only job 未执行

- **main push run 32643452003**（head=8c7b69f，PR merge 触发）：
  - frontend 结论 **success**（gate 在 main 上同样绿）
  - frontend-coverage-diff 结论 **skipped**——job 级 `if: github.event_name == 'pull_request'` 为假时 GitHub 仍把 job 列在 run 下并标记 skipped（「未执行」而非「从列表消失」，backend coverage-diff 同款表现）
  - backend = success / coverage-diff = skipped（main 常态，两 workstream 共享 CI 均健康）

### 证据三：gate 读数 PASS 留档

- `PASS: weighted avg 3.85% >= threshold 3.80%`（28/28 目录 PASS）——CI 实测读数已写入基线文档 ratchet 表新增行「Phase 82 CI 实测读数」（commit 列 = 8c7b69f），全文无 `TBD (atomic ratchet)` 残留

## D-14 校准判定（未触发）

CI 实测 3.85%（run 32642143749）== 本地实测 3.85% **零漂移**（Ubuntu runner 与本地 vitest 同读数），≥ 阈值 3.8 → 无需校准。`.coverage-fe-floors` GLOBAL 3.8 维持、**零改动**；起点行 commit 回填 **bddb2fc**（82-04 落盘 commit——该 commit 携带 .coverage-fe-floors 与基线文档的起点最终态），与 CI 读数行（8c7b69f）在同一文档内可复算审计（T-82-05-01/02 mitigation：只降不升约束未被触碰）。

## D-04 观察项

`Test (coverage)` 步骤 CI 实际耗时 **41 秒**（13:23:20 → 13:24:01），对照 15 分钟 timeout 余量充足——**不调整**（按计划纪律：仅超时才在 phase 末尾调整）。

## v1 先行佐证（作废 run 32630491947，PR #5）

首次验证 PR #5 的分支 head 被并行 workstream 共享工作树事故污染（同一工作树上并行会话的 commit 混入验证分支，docs-only 单 commit 约束被破坏——T-82-05-03 防线触发），PR #5 于 13:20:38Z 关闭作废，从干净的 main 重新创建分支开 PR #6（正式证据载体）。作废 run 32630491947 的读数与正式证据**完全一致**（3.85% / 28/28 目录 / 无软跳过提示）——佐证 gate 行为可复现，非偶然绿。

## Task Commits

1. **Task 1: 验证 PR 三项证据 + D-14 校准判定 + SHA 回填** - 证据采集跨 `8c7b69f`（PR #6 squash merge，占位行）与 `2a53443`（回填：起点 SHA bddb2fc + CI 实测读数行 + 验证记录段 + note blockquote TBD 清理）
2. **Task 2: checkpoint:human-verify** - 无实现 commit（人工终检，见下方 Checkpoint）

**Plan metadata:** 簿记收尾 docs commit（本 SUMMARY + workstream STATE/ROADMAP 更新 + config.json 入库）。

## 用户确认点清单（Git 确认纪律全程遵守）

| # | 确认点 | 内容 | 结果 |
|---|--------|------|------|
| 1 | orchestrator 层 | 本轮续接执行选择（回填两笔 commit 的执行方式） | 用户选择续接执行 |
| 2 | merge 验证 PR | PR #6 diff 范围（仅基线文档）+ checks 双绿 | 用户本人完成 merge（squash，13:48:02Z） |
| 3 | 回填 commit ×2 + push | 基线文档回填（2a53443）与 GSD 簿记收尾（本 commit）直接 push main | 用户回复 approved |

## Files Created/Modified

- `.planning/frontend-coverage-baseline.md` - ratchet 表起点行 commit `TBD (atomic ratchet)` → `bddb2fc`；新增「Phase 82 CI 实测读数」行（3.85 / 21574 / 830 / 15 / 8c7b69f）；note blockquote TBD 表述改写为已回填声明；占位行改写为完整「CI 验证记录」段（三项证据 + run URL + D-14 判定 + D-04 耗时 + 作废 run 佐证）
- `.coverage-fe-floors` - **零改动**（D-14 未触发；verify 断言 GLOBAL 行仍为 `GLOBAL 3.8`）
- 本 SUMMARY（+ workstream STATE.md / ROADMAP.md / config.json 簿记）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] verify 块 PRRUN 查询的分支名替换**
- **Found during:** 续接会话执行计划 automated verify 块
- **Issue:** 计划 verify 块字面查询 `--branch chore/82-ci-gate-verify`，但 PR #5 污染作废后正式验证 PR #6 重建在 `chore/82-ci-gate-verify-v2`（head 4d1361b）——按字面查询取到的是旧污染分支上的 PR #5 时代 run（32639775886，head 3924d0d），断言对象错位
- **Fix:** 同一断言逻辑改查 `--branch chore/82-ci-gate-verify-v2` 取得正式 run 32642143749，四项断言（DIFF_GATE_RAN_ON_PROFILE / MAIN_DIFF_JOB_SKIPPED / FRONTEND_GREEN_ON_MAIN / SHA_BACKFILLED）原样全过
- **Files modified:** 无（执行口径修正，计划文件保持历史原样）
- **Commit:** 无（续接会话批准范围仅限两笔 commit）

**2. [Rule 1 - Bug] 本 SUMMARY 证据一 run 分支标注笔误**
- **Found during:** 同上（排查 verify 失败时核实 run 归属）
- **Issue:** 证据一节初版将 run 32642143749 标注为「分支 chore/82-ci-gate-verify」——实为 `chore/82-ci-gate-verify-v2`
- **Fix:** 标注改为 -v2 并注明分支重建缘由（本处修正；基线文档无此问题，未提及分支名）
- **Files modified:** 82-05-SUMMARY.md（待随 checkpoint 后续收尾 commit 入库）

PR #5 分支污染作废重开属计划外事故但按 T-82-05-03 防线处置——关闭重开干净分支，非计划偏离。

## Issues Encountered

- PR #5（run 32630491947）验证分支 head 被并行 workstream 共享工作树事故污染，docs-only 单 commit 约束被破坏——按 T-82-05-03 关闭作废（13:20:38Z），从 main 重新创建分支开 PR #6。作废 run 读数与正式证据一致，已在基线文档与本 SUMMARY 留档。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Phase 82 全部 5 个 plan 完成**——GOV-01~05 + QUAL-02 六项 requirement 全部闭环，VALIDATION Phase gate 证据链完整
- **可进入 /gsd:verify-work**：三项 CI 证据 + 基线文档定稿（无 TBD 残留、SHA 可复算）+ 本地三件套干跑基线（82-04 SUMMARY 记录的 FE_GATE / FE_DIFF / BE_GATE 全 exit 0）
- **Task 2 人工终检 checkpoint 待用户完成**（how-to-verify 四步）后 phase 收口

## Known Stubs

None.

## Self-Check: PASSED

- .planning/frontend-coverage-baseline.md 存在且无 `TBD (atomic ratchet)` 残留、含「Phase 82 CI 实测读数」行
- 8c7b69f / 2a53443 在 git log 可见
- 计划 verify 四断言（DIFF_GATE_RAN_ON_PROFILE / MAIN_DIFF_JOB_SKIPPED / FRONTEND_GREEN_ON_MAIN / SHA_BACKFILLED）全通过（见执行日志）

---
*Phase: 82-coverage-caliber-and-governance*
*Completed: 2026-08-23*

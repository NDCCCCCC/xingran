---
phase: 82-coverage-caliber-and-governance
plan: 01
subsystem: testing
tags: [vitest, coverage, coverage-caliber, whitelist, ci-gate]

# Dependency graph
requires: []
provides:
  - 全量口径 coverage 配置（include src/**/*.{ts,tsx}，584/571 文件全部入报告，未测文件 0% 计入）
  - 白名单 exclude 配置侧真相源（cad-editor + cad-elements，D-10）
  - coverage-final.json 571 文件口径产物（Wave 2 两 gate 脚本只读复用的输入）
  - test:coverage 单次运行语义（vitest run --coverage）
affects: [82-02 check-frontend-coverage.sh, 82-03 check-frontend-diff-coverage.sh, 82-04 ci.yml 接线与基线文档]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "coverage.include 是 Vitest 4 全量口径唯一开关（coverage.all 已移除）"
    - "白名单排除的单一真相源在 vitest.config.ts coverage.exclude（D-10），gate 脚本做漂移检测"
    - "gate 阈值唯一真相源在外部脚本 + 数据文件，vitest 原生 thresholds 删除（D-16）"

key-files:
  created: []
  modified:
    - xingran-react-frontend/vitest.config.ts
    - xingran-react-frontend/package.json

key-decisions:
  - "584→571 两步口径断言按计划字面通过，实测与研究基点零偏差（22602/21574/830/3.85 全部吻合）"
  - "QUAL-02 仅标记配置侧完成——排除登记文档侧（理由/面积/复审条件三项）由 82-04 落地后才整体 complete"

patterns-established:
  - "include 圈定后 root 级 exclude 模式（node_modules/、**/*.config.*、dist/）已失效可删，src 内仅保留 src/test/ 与 **/*.d.ts"

requirements-completed: [GOV-01]

# Metrics
duration: 6min
completed: 2026-08-23
---

# Phase 82 Plan 01: vitest 全量口径切换 + 白名单 exclude Summary

**Vitest 4 coverage 切换到全量口径：include 圈定 584 个 src 文件（未测文件 0% 计入）、旧 thresholds 双轨移除（D-16）、cad 白名单 exclude 落地（D-10，571 文件 / 3.85% 收口）、test:coverage 修复为单次运行语义，159 测试零回归**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-23T08:07:06Z
- **Completed:** 2026-08-23T08:13:19Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- 全量口径生效（GOV-01）：`coverage.include: ["src/**/*.{ts,tsx}"]` 让 584 个 src 文件全部进入 coverage-final.json，statements 报告 3.67%（830/22602），旧口径 24.58% 的失真消失
- thresholds 整段删除（D-16）：连同 Phase 63 历史注释一并移除，gate 唯一真相源移交外部 bash 脚本 + .coverage-fe-floors，口径切换瞬间 test:coverage 不再因 3.85% < 24 直接红
- 白名单 exclude 落地（D-10/QUAL-02 配置侧）：cad-editor（804 stmts/8 文件）+ cad-elements（224 stmts/5 文件）写入 coverage.exclude 并配注释（真相源声明 + 面积 4.55% ≤ 5% + D-11 复审条件）
- test:coverage 改为 `vitest run --coverage`（单次运行语义，本地干跑不再挂 watch；CI=true 行为不变）
- 两步口径断言闭环：Task 1 files=584 → Task 2 files=571 / cad=0 / stmts=21574 / cov=830 / pct=3.85，四个断言值与研究基点逐位一致
- 现有 19 文件 159 测试全绿，本 plan 零新增测试（按计划）

## Task Commits

Each task was committed atomically:

1. **Task 1: 全量口径切换——include 落地 + thresholds 删除 + run 语义修复** - `51d9de0` (chore)
2. **Task 2: 白名单 exclude 落地——571 文件口径收口** - `aea0404` (chore)

**Plan metadata:** (见下方最终 docs commit)

## Files Created/Modified
- `xingran-react-frontend/vitest.config.ts` - coverage 块：+include（GOV-01/D-10 注释）、exclude 重写为 src/test/ + **/*.d.ts + 白名单两项（QUAL-02/D-10/D-11 注释）、thresholds 连注释删除（D-16 注释替代，不含 `thresholds:` 字样）
- `xingran-react-frontend/package.json` - scripts.test:coverage: `vitest --coverage` → `vitest run --coverage`（仅此一行，test/test:ui 未动）
- `xingran-react-frontend/coverage/coverage-final.json` - 本地产物（5.6MB，571 文件口径，.gitignore 已忽略不入库），供 Wave 2 gate 脚本只读复用

## Decisions Made
- **QUAL-02 不在本 plan 标记 complete**：本 plan 交付其配置侧（exclude 同步 + D-10 真相源），登记文档侧（理由/面积/复审条件三项的正式登记文档）由 82-04 落地——82-04 的 requirements 同样含 QUAL-02，届时整体勾选
- **新注释措辞规避回归 grep 误报**：D-16 替代注释写"原生 thresholds 配置已整段删除"（thresholds 后无冒号），`grep -q "thresholds:" vitest.config.ts` 无命中，满足 82-02 gate 脚本的漂移检测前提
- **零偏差记录**：584/571/22602/21574/830/3.85 全部实测命中，无需按计划的偏差记录分支处理（1f3a8f0 之后 src 无文件增删）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.（并行 workstream 的 worktree merge commit a7e9c78 在两次任务 commit 之间落入 main，属预期的并行协作行为，与本 plan 文件无交集）

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Wave 2（82-02/82-03）输入就绪：`xingran-react-frontend/coverage/coverage-final.json`（571 文件口径）已留存工作树，两 gate 脚本可直接只读干跑
- vitest.config.ts 中 include/exclude 的字面形态已锁定（grep -F 断言通过），82-02 漂移检测与 --init 可依赖
- 82-04 接线时注意：package.json 的 test:coverage 已是单次运行语义，ci.yml Test 步骤直接 `npm run test:coverage` 即可
- 遗留关注（非本 plan 范围）：D-14 全局阈值 3.8% 余量仅 0.047pp，82-05 的 CI 首跑读数校准点是已计划的应对

## Self-Check: PASSED

- vitest.config.ts / package.json 均已提交（51d9de0 / aea0404 在 git log 中可见）
- 两步断言输出：files=584（Task 1）、files=571 cad=0 stmts=21574 cov=830 pct=3.85（Task 2）
- coverage/coverage-final.json 存在于工作树（571 口径）

---
*Phase: 82-coverage-caliber-and-governance*
*Completed: 2026-08-23*

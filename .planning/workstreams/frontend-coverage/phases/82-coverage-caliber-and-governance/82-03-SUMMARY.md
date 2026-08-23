---
phase: 82-coverage-caliber-and-governance
plan: 03
subsystem: testing
tags: [coverage, diff-coverage, pr-gate, ci-gate, supply-chain, d15-lines-caliber]

# Dependency graph
requires:
  - coverage-final.json 571 文件全量口径产物（82-01 提供，本 plan 只读复用，零写入 coverage/）
provides:
  - check-frontend-diff-coverage.sh PR diff coverage ≥80% gate（三段式：diff 解析 → statementMap 区间 join → gate 判定，exit 0/1/2）
  - CLI 契约 `<coverage-final.json> <base-ref> <threshold>`（82-04 接线 ci.yml PR-only job 时消费）
  - DIFF 实测参照基线 `7376 77481 9.52`（空树合成基线全量 join 读数，82-04/82-05 join 正确性对照用）
affects: [82-04 ci.yml frontend-coverage-diff job 接线, 82-05 真实 CI diff gate 证据链]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "三段式 diff gate 逐段复刻后端 check-diff-coverage.sh：unified=0 hunk 解析（+c 偏移递增/removed 不推进/\\ No newline 跳过）→ 区间 join → PASS/FAIL +0 数值比较，仅 pathspec 与注释正则两处前端化"
    - "merge-base 回退模式：git merge-base base HEAD 为空（无共同祖先，git 三点 diff 会 fatal 'no merge base'）时回退两树 diff——只放宽不收紧（fail-safe），CI PR 场景恒走三点主路径"
    - "D-15 lines 口径 span 包含 join：行 L covered ⟺ ∃hit=1 区间 start ≤ L ≤ end（对称后端块粒度）；istanbul 原生 Lines 指标是 start-line 制，两者是刻意不同的口径选择（truth 锁定前者）"

key-files:
  created:
    - .github/scripts/check-frontend-diff-coverage.sh
  modified: []

key-decisions:
  - "merge-base 回退（Rule 1）：plan Task 2 的空树合成基线与 HEAD 无共同祖先，git diff BASE...HEAD 实测 fatal 'no merge base'（plan 技术假设盲点）；脚本在 merge-base 为空时回退两树 diff——语义只放宽不收紧（diff 只会更大、gate 只会更严），CI PR 主路径行为不变"
  - "DIFF 参照基线 7376/77481/9.52%（空树全量 join）：与全局语句覆盖 3.85% 的差额完整归因——(a) span 包含把 hit 多行语句的内部行全部计 covered（129 个 span>10 的 hit 语句贡献 9871 行，最大单语句 776 行）；(b) diff 分母剔除空行/注释行，而空行/注释在未覆盖 span 中更密（分母 −57%、分子仅 −36%）。三口径对账：statements 3.85% / istanbul-Lines(start-line 制) 3.88%（vitest 文本 3.70% 同族）/ D-15 span 口径 9.52%"
  - "Task 2 为纯实证任务（零文件改动），不产生独立代码 commit；实证结果（两分支 exit code + DIFF 三数值 + 三口径对账）由本 SUMMARY 落盘"

patterns-established:
  - "文件不在 json 的防御分支与「0 区间文件」区分：in_tsv 标记文件存在性（present-but-zero-range 的文件全部改动行不计分母，absent 文件全部计已测量未覆盖）——istanbul json 存在 75 个 0 语句文件，两者语义必须分开"
  - "供应链决策链头注释逐字引用：74-10 marketplace action 否决结论作为前端版 gate 的留档依据，可审计"

requirements-completed: [GOV-04]

# Metrics
duration: 11min
completed: 2026-08-23
---

# Phase 82 Plan 03: 前端 PR diff coverage gate 脚本 Summary

**check-frontend-diff-coverage.sh 三段式 gate（unified=0 diff 解析 + pathspec 四件套 + TS 注释三态排除 → istanbul statementMap 区间 join → ≥阈值判定，exit 0/1/2 四分支 + 空树合成基线 pass/fail 两分支全部本地确定性实证，DIFF 参照基线 7376/77481/9.52% 三口径对账落盘）**

## Performance

- **Duration:** 11 min
- **Started:** 2026-08-23T08:32:09Z
- **Completed:** 2026-08-23T08:42:34Z
- **Tasks:** 2
- **Files created:** 1

## Accomplishments
- 三段式 gate 脚本落地（GOV-04，300 行 ≥ min_lines 120）：第 1 段近乎逐字复刻后端 L87-106（仅 pathspec 四件套 + 注释三态两处替换），第 2 段 node 扁平化（与 82-02 同款 P-node，各自内嵌不互相 import）+ awk 区间 join，第 3 段 gate 判定（FILE/DIFF/UNCOVERED/PASS/FAIL 输出格式逐字保留后端 L170-184）
- D-15 lines 口径 join：git diff 新增/改动行落入 hit=1 的 statementMap 语句区间（start.line ≤ L ≤ end.line）计 covered——与后端「块粒度」语义对称；文件不在 json = 全部改动可执行行按已测量未覆盖计（新文件不给免费通行，T-82-03-03）；不落入任何区间的行不计分母
- 四分支 exit code 实证：`bash -n` 通过 / 无参 exit 2 / 坏 ref（refs/heads/nonexistent-82 经 rev-parse --verify 拒绝）exit 2 / base=HEAD 空 diff 软过 exit 0（输出 "no testable .ts/.tsx lines changed — PASS (nothing to gate)"）
- 空树合成基线两分支实证（Task 2）：`git hash-object -t tree /dev/null` + `git commit-tree` 悬空基准 → 全部 571 src 文件所有行均为新增行——threshold 80 → exit 1 + `FAIL: diff coverage` + UNCOVERED 清单 70105 行非空；threshold 0 → exit 0 + `PASS: diff coverage` + `DIFF 7376 77481 9.52`（三数值均正、pct ∈ (0,100)）
- join 正确性三口径对账（超出 plan 预期的诚实归因，见 Decisions）：statements 830/21574=3.85% / istanbul-Lines（start-line 制）799/20610=3.88% / D-15 span 口径（diff 分母）7376/77481=9.52%——9.52% 的量级差由 span 包含语义 + 空行/注释分布不均两个机制完整解释，非 join bug
- 供应链决策链留档：74-10 marketplace diff-coverage action 否决结论（gocover-coverage 无可验证 marketplace 存在、ory/xcoverage-action 不存在）逐字引用进头注释，前端沿用 in-repo bash+awk 零第三方自实现（T-82-03-01 mitigation）
- 注入面防御（T-82-03-02）：base-ref 先 `git rev-parse --verify --quiet` 校验再使用、全部 shell 展开双引号、中间文件 CHANGED/FLAT 双 mktemp + trap EXIT 清理
- 纪律遵守：全程零写入 xingran-react-frontend/coverage/（只读复用 82-01 产物）、悬空 commit 不建分支不 push、工作树仅本 plan 文件变动

## Task Commits

Each task was committed atomically:

1. **Task 1: 实现 check-frontend-diff-coverage.sh 三段式** - `38fe4c6` (feat)
2. **Task 2: 空树合成基线两分支实证** - 纯实证任务（零文件改动），无独立 commit；实证数据由本 SUMMARY 落盘

**Plan metadata:** (见下方最终 docs commit)

## Files Created/Modified
- `.github/scripts/check-frontend-diff-coverage.sh` - PR diff coverage ≥80% gate 脚本（三段式 / exit 0 过或空 diff 软过 / 1 阈值未达或解析失败 / 2 用法或坏 ref；头注释含供应链决策链、join 语义四条、CI hookup 示例供 82-04 接线）

## Decisions Made
- **merge-base 回退而非改用两防 dot**：`git diff A...B` 在无共同祖先时直接 fatal（实测），plan Task 2 技术假设的盲点。修复选回退到 `git diff A HEAD` 两树 diff 而非全改两点语义——三点 merge-base 仍是 PR 主路径（truths 锁定），回退仅在 merge-base 为空时触发且只放宽 diff 范围（gate 更严不会更松），fail-safe
- **DIFF 参照基线记录 7376 77481 9.52（供 82-04/82-05 对照）**：plan 预期「约 3-4%」基于 lines≈statements 的假设；实测 D-15 span 口径 9.52%。差异机制：(a) 129 个 span>10 的 hit 语句（最大 776 行，v8 remap 的函数级大语句）贡献 9871 covered 行；(b) diff 分母剔除空行/注释后分母降 57% 而分子仅降 36%（未覆盖 span 中空行/注释更密）。istanbul 原生 Lines 指标是 start-line 制（3.88%），与 span 制是口径选择差异，truths 锁定的是后者（对称后端块粒度）
- **Task 2 无独立 commit**：该 task 的产出是实证结论（exit code 矩阵 + DIFF 数值 + 对账），文件层面零改动，按原子 commit 纪律无物可提交；SUMMARY 即其交付物

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] 空树合成基线触发 git "no merge base" fatal**
- **Found during:** Task 1 实现前的前置实测（probe）
- **Issue:** plan Task 2 假定 `git diff BASE...HEAD`（BASE 为 `git commit-tree $(空树)` 悬空 commit）输出全部文件为新增行；实测 git 对无共同祖先的三点 diff 直接 `fatal: ... no merge base`（exit 1），脚本若逐字照抄后端将在 Task 2 全部分支 exit 2 而非预期 exit 1/0
- **Fix:** 脚本第 1 段前加 merge-base 探测——`git merge-base "$BASE_REF" HEAD` 非空走 `"${BASE_REF}...HEAD"` 三点主路径，为空回退 `"$BASE_REF" HEAD` 两树 diff；头注释留档该回退的存在理由与 fail-safe 论证
- **Files modified:** .github/scripts/check-frontend-diff-coverage.sh（Task 1 实现内建，非事后修补 commit）
- **Commit:** 38fe4c6（随 Task 1 一体落地）
- **验证:** 空树基线 threshold 80 → exit 1（FAIL + UNCOVERED 非空）、threshold 0 → exit 0（PASS + DIFF 数值行）双分支命中；base=HEAD（merge-base 存在）仍走三点路径且空 diff 软过 exit 0

（除此之外 plan 逐字执行；DIFF pct 9.52% 与 plan 预期 3-4% 的差异是口径机制的诚实实测而非实现偏差，完整对账见 Decisions。）

## Issues Encountered
None.（并行 milestone workstream 的 commit 98a4984 在本 plan 执行窗口落入 main，无文件冲突，未 rebase 未 amend。）

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 82-04 接线就绪：脚本头注释含 PR-only job 四件套模板（needs: frontend / if: pull_request / fetch-depth: 0 / download-artifact frontend-coverage），run 行三参数 `xingran-react-frontend/coverage/coverage-final.json origin/${{ github.base_ref }} 80` 与 key_link 契约一致
- 82-05 对照基线就绪：真实 PR 的 DIFF 行 covered/total 应远小于合成基线（77481 为全量 src 口径），但 pct 量级与 join 语义可用本 SUMMARY 的 7376/77481/9.52 与三口径对账表复核
- 软跳过语义已对齐：diff gate 的 profile 缺失软跳过 exit 0 + stderr 提示与 82-02 主 gate 同款，配合 PR job 的 needs 依赖（上游 Test 失败才是真阻断）
- 遗留关注（非本 plan 范围）：CI 真实 PR 上 merge-base 路径的首次实证归 82-05（本地已覆盖回退路径与空 diff 路径，三点主路径在 HEAD...HEAD 空 diff 分支间接覆盖）

## Self-Check: PASSED

- .github/scripts/check-frontend-diff-coverage.sh 存在（300 行，bash -n 通过）
- 38fe4c6 在 git log 可见（feat(82-03)）
- Task 1 verify：BASH_N_OK / EXIT2_OK / EXIT2_BADREF_OK / EXIT0_OK
- Task 2 verify：EXIT1_FAIL_OK + FAIL_LINE_OK + UNCOVERED_OK（70105 行）/ EXIT0_PASS_OK + PASS_LINE_OK + DIFF_LINE_OK（DIFF 7376 77481 9.52）

---
*Phase: 82-coverage-caliber-and-governance*
*Completed: 2026-08-23*

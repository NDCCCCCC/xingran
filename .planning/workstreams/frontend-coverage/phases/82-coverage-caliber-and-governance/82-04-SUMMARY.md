---
phase: 82-coverage-caliber-and-governance
plan: 04
subsystem: testing
tags: [coverage, ci-gate, ci-orchestration, pr-diff-coverage, coverage-baseline, whitelist-registry, ratchet]

# Dependency graph
requires:
  - check-frontend-coverage.sh 主 gate（82-02 提供，exit 0/1/2/4/6 CLI 契约）
  - check-frontend-diff-coverage.sh PR diff gate（82-03 提供，`<json> <base-ref> <threshold>` CLI 契约）
  - .coverage-fe-floors 阈值数据文件（82-02 提供，GLOBAL 3.8 + 28 目录 floor）
  - coverage-final.json 571 文件全量口径产物（82-01 提供，Task 2 复算数据源，零写入 coverage/）
provides:
  - ci.yml frontend job 内嵌 Coverage gate + Upload coverage artifact 步骤（D-01/D-03，GOV-03/GOV-05 的 CI 编排）
  - ci.yml frontend-coverage-diff PR-only job（D-02，GOV-04 的 CI 编排；download 带 with.path 还原 upload v4 LCA 剥离）
  - .planning/frontend-coverage-baseline.md（D-09，GOV-02 ratchet 审计载体 + QUAL-02 白名单登记段）
affects: [82-05 真实 CI 验证（PR 三项证据 + D-14 GLOBAL 校准 + 起点 commit 列 TBD 回填本基线文档）]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "upload-artifact@v4 目录上传的 LCA 语义与 download path 还原配对：上传目录时以路径最小公共祖先为 artifact 根（xingran-react-frontend/ 前缀被剥离，json 与 html 报告位于 artifact 根），download 必须以 with.path: xingran-react-frontend/coverage 还原到脚本第一参数路径——无 path 输入则 json 落 workspace 根、gate 命中脚本内 json 缺失软跳过、全 PR 静默 exit 0（T-82-04-05 mitigation）"
    - "gate 步骤双防御：working-directory: . 显式覆盖 job 级 xingran-react-frontend（Pitfall 4——.github/scripts 与 .coverage-fe-floors 相对解析）+ 不挂 if: always()（gate 弱化防御 T-82-04-02，软跳过仅在脚本内且仅限 json 缺失）"
    - "共享 ci.yml 只增不改纪律的机械验证：sed 区间（backend: 到 frontend: 之间，含 coverage-diff job）与 git show main: 逐字节 diff 为空即 SHARED_JOBS_UNCHANGED"
    - "基线文档双口径并存：文档头白名单前口径（3.67%/22602/584）与 ratchet 表/per-dir 快照 gate 口径（3.85%/21574/571）是刻意的口径映射而非矛盾"

key-files:
  created:
    - .planning/frontend-coverage-baseline.md
  modified:
    - .github/workflows/ci.yml

key-decisions:
  - "components 细分参考 block 取 json 复算而非 RESEARCH 研究期数字：RESEARCH「其余 91」与其自身具名合计 3981 ≠ gate 口径 3958，按计划「两者有出入以脚本输出为准」纪律改为 coverage-final.json 复算（12 具名子目录 + 1 行零散聚合 = 13 行，合计恰为 3958/215/118），行内标注非 gate 粒度供 Phase 84 用"
  - "0pct_pkg_count = 15：per-dir 快照中 pct 为 0.00% 的目录实测数（json 复算），起点行如实登记"
  - "ratchet 表起点行 commit 写 TBD (atomic ratchet)：82-05 CI 校准定稿后回填实际短 SHA（后端 D-04 同款纪律；Ratchet note blockquote 与表格两处 TBD 字样 82-05 会一并清理）"

patterns-established:
  - "PR-only diff job 的 artifact 复用模板（frontend-coverage-diff）：needs: frontend + if: pull_request + fetch-depth: 0 + download-artifact@v4 带 path 还原——后端 coverage-diff 的可复制模板再加 LCA path 差异注释"

requirements-completed: [GOV-02, QUAL-02]

# Metrics
duration: 6min
completed: 2026-08-23
---

# Phase 82 Plan 04: ci.yml 接线 + 前端覆盖率基线文档 Summary

**ci.yml frontend job 内嵌三合一 Coverage gate（working-directory: . 覆盖 job 级目录 + 无 if: always()）+ PR-only frontend-coverage-diff job（download-artifact 带 with.path 还原 upload v4 LCA 剥离的目录前缀，零重复测试），.planning/frontend-coverage-baseline.md 落盘（ratchet 表 schema 逐字对称后端 + 28 目录快照与 json 复算逐值一致 + QUAL-02 白名单三项登记 4.55% ≤ 5%），backend/coverage-diff 两 job 区间与 main 逐字节一致**

## Performance

- **Duration:** 6 min
- **Started:** 2026-08-23T08:48:04Z
- **Completed:** 2026-08-23T08:54:00Z
- **Tasks:** 2
- **Files created:** 1（.planning/frontend-coverage-baseline.md）
- **Files modified:** 1（.github/workflows/ci.yml）

## Accomplishments
- **Task 1（ci.yml 接线，GOV-03/GOV-04/GOV-05 的 CI 编排）**：frontend job 四步改造——Test → `Test (coverage)`（npm run test:coverage，82-01 已改 vitest run --coverage）；其后内嵌 `Coverage gate`（`bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors`，exit 1/4/6 任一即 job 红，**working-directory: .** 覆盖 job 级目录避免 Pitfall 4，无 if: always()）与 `Upload coverage artifact`（if: always()，name frontend-coverage 与 backend-coverage 命名对齐，path 相对 workspace root，retention 30 天，D-03 json+html 供本地 debug）；Build 保持最后；timeout-minutes 15 不动（D-04）
- **新增 frontend-coverage-diff job（D-02）**：逐字段对称后端 coverage-diff（runs-on / timeout 10 / needs: frontend / if: github.event_name == 'pull_request' / checkout@v7 fetch-depth: 0），唯一刻意偏差 = download-artifact@v4 带 `with.path: xingran-react-frontend/coverage`——upload v4 目录上传以最小公共祖先为 artifact 根（前缀剥离），无 path 输入则 json 解压到 workspace 根命中 82-03 软跳过、全 PR 静默 exit 0（GOV-04 失效）；注释留档 LCA 语义与后端单文件上传的差异原因
- **共享编辑区零改动实证（T-82-04-01）**：backend + coverage-diff 区间（sed `/^  backend:/,/^  frontend:/`）与 `git show main:` 逐字节 diff 为空（SHARED_JOBS_UNCHANGED）；concurrency 与 permissions: contents: read 未触碰（T-82-04-03 accept 前提保持）
- **Task 2（基线文档，GOV-02 + QUAL-02）**：`.planning/frontend-coverage-baseline.md`（103 行 ≥ min_lines 80）落盘于仓库 .planning/ 根（D-09 字面路径，与后端 coverage-baseline.md 对称并列）；ratchet 记录表列名与后端 schema **逐字一致**（diff 校验 SCHEMA_IDENTICAL：date / phase_label / weighted_avg / total_stmts / total_covered / 0pct_pkg_count / commit / phase_executor / ratchet_from / ratchet_to）；起点行 2026-08-23 / 3.85 / 21574 / 830 / 15 / TBD (atomic ratchet) / n/a / n/a / n/a
- **per-dir 快照 28 行**（D-05 粒度，含 `(src root)` 13 stmts 与 `api` 8 stmts 显式条目，合计 21574/830/571）与 coverage-final.json node 复算**逐值一致**（T-82-04-04）；components 细分 13 行参考 block 附后（json 复算，注明非 gate 粒度供 Phase 84 用）
- **白名单登记三项齐全（QUAL-02）**：cad-editor（804/8，3.56%）与 cad-elements（224/5，0.99%）各含排除理由/面积/复审条件，合计 1028/13 = **4.55% ≤ 5%**；表后 D-12 锁死（仅此两项，新增须 milestone 级显式决策 + 面积重核算）与 D-10 同源（真相源 vitest.config.ts coverage.exclude，gate exit 6 漂移检测守护）两条声明；复审条件为定量触发（自身 ≥70% 可启动移除 + milestone 收口强制重审）
- **倒退检查清单 + Ratchet note blockquote**：本行三项全勾（无新增白名单 / 面积 ≤5% / 数字可由 gate 脚本复算）；note 声明 commit 列 TBD 由 82-05 回填、.coverage-fe-floors 变更与本文件追加必须同一 commit

## 双口径对照（防 SC-3 字面误读，两组数字并存是有意的口径映射）

| 口径 | 覆盖率 | stmts | 文件数 | 含义 |
|------|-------:|------:|-------:|------|
| 白名单前（全量含 cad 两目录） | 3.67% | 22602 | 584 | 文档头「测量口径」行记载；QUAL-02 白名单面积分母（1028/22602 = 4.55%） |
| gate 口径（白名单排除后） | 3.85% | 21574 | 571 | ratchet 记录表、per-dir 快照、.coverage-fe-floors、CI gate 全部使用此口径 |

## Task Commits

Each task was committed atomically:

1. **Task 1: ci.yml 接线——frontend job 四步改造 + 新增 frontend-coverage-diff job** - `3992824` (feat)
2. **Task 2: 基线文档 .planning/frontend-coverage-baseline.md + 白名单登记** - `bddb2fc` (docs)

**Plan metadata:** (见下方最终 docs commit)

## Files Created/Modified
- `.github/workflows/ci.yml` - frontend job Test→Test (coverage) + Coverage gate + Upload coverage artifact 三步改造、顶部注释 frontend 步骤序更新、文件末尾新增 frontend-coverage-diff job（backend/coverage-diff/concurrency/permissions 零改动）
- `.planning/frontend-coverage-baseline.md` - 前端覆盖率基线文档（文档头双口径 + ratchet 记录表 + 28 目录 per-dir 快照 + components 细分参考 + QUAL-02 白名单登记 + 倒退检查清单 + Ratchet note）

## Decisions Made
- **components 细分取 json 复算**：RESEARCH 的「其余 91」与其具名行合计 3981 ≠ gate 口径 components 3958（研究期 CLI 覆盖运行计数差异，同 82-02 的 152 vs 118 文件数结论）；按计划「两者有出入以脚本输出为准并在行尾标注」纪律，13 行参考 block 全部用 coverage-final.json 复算（12 具名子目录 + 1 行零散聚合 table/three/charts/NoticeDetail/markdown/modal = 68 stmts/7 文件），合计恰为 3958/215/118，文档 Notes 段标注 components 文件数 118 与 RESEARCH 152 的差异原因
- **0pct_pkg_count = 15**：起点行如实登记 json 复算的 0.00% 目录数（pages 12 个零覆盖二级目录 + router + (src root) + api），非后端 33 个 0% 包的同义概念——前端以目录为单位
- **起点 commit 列 TBD (atomic ratchet)**：Phase 82 只建基线不 bump（ratchet_from/ratchet_to 均 n/a），真实 SHA 由 82-05 在 CI 校准定稿后回填——后端 Phase 71 起点（5ead742）是事后已知的特例，前端起点在 82-04 执行时点无法预知 82-05 校准后是否需要调整 GLOBAL，TBD 是诚实选择

## Deviations from Plan

None - plan executed exactly as written.（ci.yml 改造与新增 job 一次通过全部 verify；基线文档数字与 json 复算零偏差，无 Rule 1-3 触发。）

## Issues Encountered
None.（并行 milestone workstream 在本 plan 执行窗口未落入 ci.yml 相关 commit；提交前重新确认 ci.yml 工作树与 HEAD 一致。）

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- **82-05 真实 CI 验证就绪**：PR 三项证据（frontend job gate 日志 PASS + frontend-coverage-diff job 日志**无 json 缺失软跳过提示**（T-82-04-05 的真实 CI 复核点，证明 download path 还原生效）+ artifact 含 json/html）；D-14 GLOBAL 校准（CI 首跑 <3.8 时重跑 `--init` 即得校准值，幂等）；起点 commit 列 TBD 回填 + Ratchet note 两处清理
- **本地三 gate 干跑全绿基线**：FE_GATE_EXIT=0（GLOBAL PASS + 28/28 目录）/ FE_DIFF_EXIT=0（vs HEAD 空 diff 软过）/ BE_GATE_EXIT=0——接线未破坏 82-02/82-03 产物
- **82-03 对照基线可用**：DIFF 参照 7376/77481/9.52%（空树全量 join）与三口径对账表在 82-03-SUMMARY，真实 PR 读数远小于该基线属预期

## Self-Check: PASSED

- .github/workflows/ci.yml 存在（YAML OK / SHARED_JOBS_UNCHANGED / STEP_ORDER_OK / DIFF_ARTIFACT_PATH_RESTORED / PR_ONLY_OK / WORKDIR_DOT_OK / GATE_NO_ALWAYS_OK / DIFF_ARGS_OK）
- .planning/frontend-coverage-baseline.md 存在（BASELINE_OK stmts=21574 cov=830 / 103 行 / SCHEMA_IDENTICAL）
- 3992824 / bddb2fc 在 git log 可见
- 本地三 gate 干跑 exit 0（FE_GATE / FE_DIFF / BE_GATE）

---
*Phase: 82-coverage-caliber-and-governance*
*Completed: 2026-08-23*

---
phase: 82-coverage-caliber-and-governance
plan: 02
subsystem: testing
tags: [coverage, coverage-gate, ratchet, per-dir-floor, ci-gate, whitelist-drift]

# Dependency graph
requires:
  - coverage-final.json 571 文件全量口径产物（82-01 提供，本 plan 只读复用）
provides:
  - check-frontend-coverage.sh 主 gate（全局阈值 + per-dir floor + 白名单漂移三合一，exit 0/1/2/4/6）
  - .coverage-fe-floors 阈值数据文件（GLOBAL 3.8 + 28 目录 ratchet 初值表，D-07）
  - --init 生成器（D-08：floor 初值 = 实测 −0.5pp 下限 0，GLOBAL = min(实测, 3.8) 向下截断）
  - CLI 契约（82-04 接线 ci.yml、82-05 校准 GLOBAL 行时消费）
affects: [82-04 ci.yml frontend job Coverage gate 步骤, 82-05 CI 首跑读数校准与基线文档]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "istanbul json 扁平化三要件（反斜杠→正斜杠 + toLowerCase 锚点匹配 + xingran-react-frontend/src/ 截取）——bash 包装内联 node -e，Windows/CI 双兼容"
    - "floors 数据文件双段式解析：GLOBAL 行空格分隔按原始记录正则排除、目录行 TAB 分隔 -F'\\t' 提取（两种分隔符并存是契约）"
    - "per-dir 双向比对：floors 条目缺失于 profile 与 profile 目录未登记 floors 均判违例——新增目录不得免检（无无主面积不变式）"

key-files:
  created:
    - .github/scripts/check-frontend-coverage.sh
    - .coverage-fe-floors
  modified: []

key-decisions:
  - "pages/login floor 61.6 与 RESEARCH 表 61.7 差 0.1：62.11−0.5=61.61 按一位小数应为 61.6，以 gate 可复算的脚本输出为真相源（Open Question 3 既定约定），RESEARCH 表 61.7 属研究期舍入笔误"
  - "--init 的 GLOBAL 向下截断（int(v*10+ε)/10）而非四舍五入：实测落在 (3.7,3.8) 区间时四舍五入会生成高于实测的阈值让 gate 自锁，截断保证生成值永不高于实测（82-05 CI 校准同样受益）"
  - "components 文件数 118 与 RESEARCH 表 152 不一致但 stmts/covered 逐位一致（3958/215）：gate 只消费 stmts/covered，文件数不参与判定，差异源于研究期 CLI 覆盖运行的计数口径，无需处理"

patterns-established:
  - "漂移检测 fail-fast 置于聚合之前（exit 6 先于 exit 1/4 判定）：污染数据不得进入 gate 数学"
  - "malformed floors 行（非注释非空非 GLOBAL 且无 TAB）与非法 floor 值（非数字）独立 exit 2：阈值数据文件被写坏时 gate 拒绝解释而非静默取 0"

requirements-completed: [GOV-03, GOV-05]

# Metrics
duration: 12min
completed: 2026-08-23
---

# Phase 82 Plan 02: 前端主 gate 脚本 + floors 数据文件 Summary

**check-frontend-coverage.sh 三合一 gate（全局加权 statements ≥ GLOBAL 3.8 + 28 目录 per-dir floor 双向比对 + cad 白名单漂移 fail-fast，exit 0/1/2/4/6 五分支干跑矩阵全命中）+ --init 自动生成 .coverage-fe-floors ratchet 初值表（实测 −0.5pp，幂等可复算）**

## Performance

- **Duration:** 12 min
- **Started:** 2026-08-23T08:17:22Z
- **Completed:** 2026-08-23T08:29:18Z
- **Tasks:** 2
- **Files created:** 2

## Accomplishments
- 主 gate 脚本落地（GOV-03/GOV-05）：bash+awk+内联 node 扁平化（5.6MB 单行 istanbul json → TSV），结构对称后端 check-coverage.sh（软跳过/防御收尾/数值 +0 强制/`%-50s %8d %8d %6.2f%%` 表格式全部照抄），417 行
- 全局阈值 gate：加权 statements 平均 3.85% ≥ GLOBAL 3.8%（D-13 statements 单维度、D-14 阈值），空 profile（0 statements）防御性 FAIL，输出无 ^PASS: 行按解析失败 exit 1
- per-dir floor gate（D-05 粒度：src 一级 + pages 二级 + `(src root)`/`api` 显式条目）：28/28 目录全过；双向比对成立——floors 条目缺失于 profile 或 profile 目录未登记 floors 均计违例 exit 4（无无主面积不变式，T-82-02-04 mitigation）
- 白名单漂移检测（D-10）：扁平化后 fail-fast grep cad-editor/cad-elements → exit 6（先于聚合，污染数据不进 gate 数学，T-82-02-03 mitigation）；合成注入假键干跑实证
- --init 生成器（D-08）：floor 初值 = max(0, 一位小数(实测% − 0.5))（D-06 测量噪声余量，头注释摘录后端 P2_RATCHET 方法论三条噪声来源）；GLOBAL = min(实测, 3.8) 向下截断（D-14）；重跑与已存文件 diff 为空（幂等）
- .coverage-fe-floors 入库仓库根（与 .coverage-threshold 同级，未入 .gitignore）：`GLOBAL 3.8` + 28 行 TAB 分隔目录表 + 注释头（bump 本文件即可、与基线文档追加同 commit 的纪律）
- 五分支干跑矩阵全部命中：真实数据 exit 0 / 无参 exit 2 / GLOBAL 5.0 → exit 1 / components 90.0 → exit 4（FAIL 行含 5.43% < 90.0% 明细）/ cad 假键注入 → exit 6（stderr 报漂移并列出路径）；矩阵只写 mktemp 副本，正式文件与 coverage/ 目录零污染
- 28 个目录条目与 RESEARCH 实测全表逐值核对：除 pages/login 一处 0.1 级差异（见 Decisions）外全部一致；`(src root)` 0.0 与 `api` 0.0 在列，21 stmts 无主面积消除（Pitfall 8）

## Task Commits

Each task was committed atomically:

1. **Task 1: 实现 check-frontend-coverage.sh（扁平化 + 全局 gate + per-dir floor + 漂移检测 + --init）** - `f8d3d44` (feat)
   - 附带 Rule 1 修复 commit `c88f729` (fix)：GLOBAL 行解析 bug，见 Deviations
2. **Task 2: --init 生成 .coverage-fe-floors + 四失败分支干跑矩阵** - `9c561b3` (chore)

**Plan metadata:** (见下方最终 docs commit)

## Files Created/Modified
- `.github/scripts/check-frontend-coverage.sh` - 主 gate 脚本（gate 模式 + --init 模式；exit 0/1/2/4/6；头注释含 CI hookup 示例与 working-directory Pitfall 4 提示、ratchet 方法论摘录）
- `.coverage-fe-floors` - 阈值数据文件（GLOBAL 3.8 + 28 目录 floor 表；ratchet 状态唯一存储，bump 即纯数据变更）

## Decisions Made
- **pages/login 61.6 vs RESEARCH 61.7**：59/95=62.11%，−0.5=61.61，一位小数=61.6；RESEARCH 表的 61.7 属研究期舍入笔误。按 Task 2 步骤 2 的既定约定（Open Question 3），gate 可复算的脚本聚合输出是真相源，已按计划要求记录差异
- **--init GLOBAL 向下截断而非四舍五入**：min(实测, 3.8) 落在两位小数时（如 3.76），%.1f 四舍五入会得 3.8 > 实测而让 gate 永久红；int(v×10+ε)/10 截断保证生成阈值 ≤ 实测值。本期实测 3.85 > 3.8 取 min 后恰为 3.8，截断分支未触发但已为 82-05 CI 校准（可能得 3.7x 读数）预埋
- **零新增依赖确认**：node/awk/grep/sed 均为 runner 与前端工具链既有运行时（T-82-SC accept 处置成立），无任何包安装

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] GLOBAL 行被误判 malformed（exit 2）**
- **Found during:** Task 2（五分支矩阵第一分支 exit 0 失败）
- **Issue:** floors 文件契约是混合分隔符——GLOBAL 行空格分隔、目录行 TAB 分隔。原实现的 malformed 检查用 `-F'\t'` 后 `$1 != "GLOBAL"` 排除 GLOBAL 行，但 TAB 分割下 `GLOBAL 3.8` 整行是一个字段（$1="GLOBAL 3.8"），永远不等于 "GLOBAL"，落入 NF<2 判定为 malformed，gate 对真实 floors 文件直接 exit 2
- **Fix:** GLOBAL 行改为按原始记录正则 `^GLOBAL([ \t]|$)` 排除（同时兼容空格与 TAB 写法），malformed 检查与目录表提取两条 awk 统一该排除规则
- **Files modified:** .github/scripts/check-frontend-coverage.sh
- **Commit:** c88f729
- **验证:** 修复后五分支矩阵全命中 + Task 1 verify 链（bash -n / 29 行 / GLOBAL=3.8 / 关键条目）重跑通过 + --init 幂等不回归

（除此之外 plan 逐字执行；pages/login 0.1 级差异为计划内预期分支，记录于 Decisions 而非 deviation。）

## Issues Encountered
None.（并行 workstream 的 merge commit 未在本 plan 执行窗口落入 main；工作树自始至终仅本 plan 文件变动。）

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 82-04 接线就绪：ci.yml frontend job 的 Coverage gate 步骤可直接 `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors`（脚本头注释含 working-directory: . 的 Pitfall 4 提示；key_link：.coverage-fe-floors → ci.yml 经脚本名引用已成立）
- 82-03 共享前提确认：扁平化 TSV 契约（relpath\tstmts\tcovered\tranges）已在本脚本实证，diff gate 脚本可复用同一 node 段与 ranges 列（start-end:hit）
- 82-05 校准就绪：CI 首跑若 <3.8，GLOBAL 行 := min(CI 实测, 3.8) 向下取一位小数——--init 已内置该数学，重跑 `--init` 即得校准值（幂等性保证无漂移）
- 遗留关注（非本 plan 范围）：D-14 余量 0.047pp 的 CI 读数风险由 82-05 显式检查点承接

## Self-Check: PASSED

- .github/scripts/check-frontend-coverage.sh 存在（417+ 行，bash -n 通过）
- .coverage-fe-floors 存在（40 行：12 注释 + GLOBAL + 28 目录）
- f8d3d44 / c88f729 / 9c561b3 全部在 git log 可见
- 五分支矩阵输出：EXIT0_OK / EXIT2_OK / EXIT1_OK / EXIT4_OK / EXIT6_OK / INIT_IDEMPOTENT

---
*Phase: 82-coverage-caliber-and-governance*
*Completed: 2026-08-23*

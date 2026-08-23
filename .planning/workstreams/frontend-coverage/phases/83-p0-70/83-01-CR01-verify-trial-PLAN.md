---
phase: 83-p0-70
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .github/workflows/ci.yml
  - .planning/frontend-coverage-baseline.md
autonomous: false
requirements:
  - GOV-04
user_setup: []
must_haves:
  truths:
    - "[D-01][D-02] CR-01/WR-01/WR-02/WR-03 修复已以提交存在于 main，无需重复实现"
    - "ci.yml 中关于 diff gate \"json 缺失软跳过\" 的注释已修正为与脚本 fail-closed 行为一致（含第 219 行 stale 描述）[D-02]"
    - "[D-03] 试验 PR 变更覆盖 src/test/、.d.ts 与白名单目录三类路径，CI frontend 与 frontend-coverage-diff job 均 success 后关闭不 merge"
  artifacts:
    - path: .github/workflows/ci.yml
      provides: 修正后的 diff gate 注释
    - path: .planning/frontend-coverage-baseline.md
      provides: 记录本次验证的 CI 验证记录追加行
  key_links:
    - from: .github/scripts/check-frontend-diff-coverage.sh
      to: .github/workflows/ci.yml
      via: "注释语义一致性：脚本 WR-01 已改为 exit 2，ci.yml 注释不得再称 exit 0"
      pattern: "exit 2.*configuration drift"
---

<objective>
验证 Phase 82 review-fix 中 CR-01/WR-01~03 四项 gate 修复已正确落库，清理遗留注释措辞（含第 219 行 stale 描述），并发起一次真实 CI 试验 PR 以实证 GOV-04 diff coverage gate 在 src/test/、.d.ts、白名单目录变更场景下不会误报失败。

Purpose: 消除 82 遗留 WARNING，确保后续 Phase 83 测试/harness PR 不会因 gate 缺陷而踩红。
Output: 修正后的 ci.yml 注释、试验 PR 关闭记录、基线文档追加的验证记录行。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/workstreams/frontend-coverage/ROADMAP.md
@.planning/workstreams/frontend-coverage/REQUIREMENTS.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-CONTEXT.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-RESEARCH.md
@.planning/workstreams/frontend-coverage/phases/83-p0-70/83-PATTERNS.md
@.github/scripts/check-frontend-diff-coverage.sh
@.github/scripts/check-frontend-coverage.sh
@.github/workflows/ci.yml
@.coverage-fe-floors
@.planning/frontend-coverage-baseline.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 验证四项修复已落库并修正 ci.yml 注释</name>
  <files>.github/workflows/ci.yml</files>
  <read_first>
    - .github/scripts/check-frontend-diff-coverage.sh（确认 WR-01 exit 2 与 CR-01 pathspec 镜像）
    - .github/scripts/check-frontend-coverage.sh（确认 WR-02 锚定前缀漂移检测与 WR-03 floors 数值校验）
    - .github/workflows/ci.yml（确认 frontend-coverage-diff job 注释当前措辞，重点看第 213-220 行与第 226-231 行）
    - .planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW-FIX.md（如存在，确认已落库修复记录）
  </read_first>
  <action>
    1. 使用 git log --oneline 核对提交 60f712c、27f275e、94d3a16、aa3bf0c 存在于 main 分支，且对应修改了 check-frontend-diff-coverage.sh / check-frontend-coverage.sh。
    2. 阅读两个 gate 脚本的当前 HEAD 内容，确认：diff 脚本缺 profile 分支为 exit 2（WR-01）；pathspec 完整镜像 vitest.config.ts 的 coverage.exclude，包含 src/test/、**/*.d.ts、cad-editor/**、cad-elements/**（CR-01）；coverage 脚本白名单漂移检测锚定于 ^xingran-react-frontend/src/components/cad-(editor|elements)/（WR-02）；GLOBAL 与 per-dir floor 均经过 /^[0-9]+([.][0-9]+)?$/ 数值结构校验（WR-03）。
    3. 修正 .github/workflows/ci.yml 中三处 stale/misleading 注释：第 172-179 行 Coverage gate、第 213-220 行 download-artifact 注释块（含第 219 行 "json 缺失软跳过而静默 exit 0"）以及第 226-231 行 Diff coverage gate 的注释。统一改为准确描述——diff 脚本在 profile 缺失时 fail-closed exit 2（WR-01），coverage 脚本在 profile 缺失时软跳过 exit 0（对称 backend），避免开发者产生 diff gate 也会软跳过的误解。仅改注释，不改 job 行为或命令。
  </action>
  <verify>
    <automated>
      git log --oneline --all | grep -E "(60f712c|27f275e|94d3a16|aa3bf0c)" 返回 4 条记录；
      grep -n "exit 2" .github/scripts/check-frontend-diff-coverage.sh | grep -c "configuration drift" >= 1；
      grep -n "src/test/" .github/scripts/check-frontend-diff-coverage.sh | wc -l >= 1；
      grep -n "cad-(editor|elements)" .github/scripts/check-frontend-coverage.sh | wc -l >= 1；
      grep -nE "^[0-9]+([.][0-9]+)?$" .github/scripts/check-frontend-coverage.sh | wc -l >= 2；
      grep -nE "软跳过|exit 0" .github/workflows/ci.yml | grep -E "Diff coverage|frontend-coverage-diff|Download coverage report" | wc -l == 0。
    </automated>
  </verify>
  <done>
    - 四项修复落库证据已收集，ci.yml 注释（含第 219 行）与脚本实际行为一致。
    - 无功能代码变更，仅注释修正。
  </done>
  <acceptance_criteria>
    - ci.yml 中 frontend-coverage-diff job 相关注释（第 213-231 行）不再包含 "json 缺失软跳过" 或 "exit 0" 等暗示软跳过的措辞。
    - git diff 仅包含 ci.yml 的注释行变更，无命令变更。
    - 本地执行 bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json origin/main 80 时，若 profile 存在则正常输出 PASS 或 FAIL；若将 profile 路径改为不存在文件，脚本 exit 2 并输出 "configuration drift"。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: 本地空树合成基线复现 CR-01 修复前后行为</name>
  <files>无新增文件</files>
  <read_first>
    - .github/scripts/check-frontend-diff-coverage.sh（确认 pathspec 与软跳过逻辑）
    - xingran-react-frontend/vitest.config.ts（coverage.exclude 真相源）
  </read_first>
  <action>
    1. 在本地工作树构造一个临时分支，创建一个空树提交作为 diff 基线（git hash-object -t tree /dev/null），再创建第二个提交：添加一个 src/test/utils/trial-test.ts（返回 0 的占位测试）、修改 xingran-react-frontend/src/types/global.d.ts（追加一行无害注释）、在 src/components/cad-editor/ 下任一文件追加一行无害注释（例如 README 式注释，不引入运行时变更）。
    2. 运行 bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json <空树提交> 80，确认输出 "no testable .ts/.tsx lines changed ... PASS" 或 diff coverage 仅统计非排除路径，无 src/test/、.d.ts、cad-editor 行被计入 denominator。
    3. 删除临时分支与临时文件，恢复工作树干净状态。
  </action>
  <verify>
    <automated>
      脚本返回 exit 0；输出包含 "PASS" 或 "no testable .ts/.tsx lines changed"；输出中不出现 xingran-react-frontend/src/test/、*.d.ts 或 cad-editor/cad-elements 相关未覆盖行。
    </automated>
  </verify>
  <done>
    - 本地复现证明 CR-01 pathspec 镜像生效，三类路径不再误触 diff gate。
  </done>
  <acceptance_criteria>
    - 临时分支上的 diff gate 运行 exit 0。
    - 输出中三类排除路径（src/test/、.d.ts、cad-editor/cad-elements）无任何行被标记为 UNCOVERED 或计入 denominator。
    - 临时分支与文件已清理，不影响后续 plan。
  </acceptance_criteria>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: 发起真实 CI 试验 PR 并关闭不 merge</name>
  <what-built>
    已准备本地验证与注释清理；需要用户在 GitHub 上发起一个真实 PR（或授权使用 gh CLI 发起），PR 包含三类路径的合法变更：src/test/utils/ 新增一个占位测试文件、任一 .d.ts 文件追加一行注释、任一 cad-editor 或 cad-elements 下文件追加一行注释。CI 跑通后关闭 PR 不合并。
  </what-built>
  <how-to-verify>
    1. 创建分支 phase-83-trial-cr01，提交三类路径变更（每类至少一个文件）。
    2. 推送并创建 PR 到 main，标题注明 "[DO NOT MERGE] Phase 83 CR-01 trial PR"。
    3. 等待 CI 完成，检查 frontend job 与 frontend-coverage-diff job 均 success（或 frontend-coverage-diff 正确 skip / pass）。
    4. 收集 CI run URL 与关键日志片段（diff gate 输出 PASS、无 json 缺失软跳过提示）。
    5. 关闭 PR（Close without merging）。
  </how-to-verify>
  <resume-signal>提供 CI run URL 与结论（frontend job success + frontend-coverage-diff job success/skip）后，Claude 继续追加基线文档记录。</resume-signal>
  <read_first>
    - .github/workflows/ci.yml（确认 frontend job 与 frontend-coverage-diff job 步骤名称与命令）
    - .github/scripts/check-frontend-diff-coverage.sh（确认 diff gate 行为与期望日志）
    - .planning/frontend-coverage-baseline.md（确认 CI 验证记录段格式）
  </read_first>
  <acceptance_criteria>
    - 用户提供试验 PR 链接与 CI run URL。
    - CI run 中 frontend job 状态为 success。
    - CI run 中 frontend-coverage-diff job 状态为 success 或 skipped（PR-only job 在 push run 显示 skipped 为预期行为）。
    - PR 状态为 closed（未 merge）。
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 4: 追加基线文档 CI 验证记录</name>
  <files>.planning/frontend-coverage-baseline.md</files>
  <read_first>
    - .planning/frontend-coverage-baseline.md（确认 CI 验证记录段格式）
    - .github/workflows/ci.yml（确认 job 名称与命令）
  </read_first>
  <action>
    在 .planning/frontend-coverage-baseline.md 的 "CI 验证记录 (Phase 82 · 82-05, 2026-08-23)" 段下方追加 "Phase 83 · 83-01 CR-01 试验 PR 验证" 小节，记录：试验 PR 链接、分支名、head commit、CI run URL、结论（frontend job success / frontend-coverage-diff job success 或 skip）、三类路径清单。不修改 floors 文件（本次无覆盖率变化，不触发 ratchet）。
  </action>
  <verify>
    <automated>
      .planning/frontend-coverage-baseline.md 包含 "Phase 83 · 83-01" 小节；小节中包含 PR 链接、CI run URL、"关闭不 merge" 字样；.coverage-fe-floors 在本次 plan 中未被修改。
    </automated>
  </verify>
  <done>
    - 基线文档记录了 83-01 的 CI 验证证据，CR-01 修复得到真实 CI 确认。
  </done>
  <acceptance_criteria>
    - .planning/frontend-coverage-baseline.md 新增段包含 phase-83-trial-cr01 分支名或 PR 链接、CI run URL、结论。
    - git diff 中 .coverage-fe-floors 无变更。
    - 文档格式与前一段 "CI 验证记录" 保持一致。
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| CI workflow → gate scripts | ci.yml 注释若误导开发者，可能让人误以为 diff gate 会软跳过，降低对 fail-closed 行为的警惕 |
| 试验 PR → main | 试验 PR 必须关闭不 merge，避免占位测试/注释进入主分支 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-83-01-01 | Information Disclosure | 试验 PR | mitigate | PR 标题明确 "[DO NOT MERGE]"，变更仅限无害注释与 src/test/ 占位文件，不触碰业务逻辑或 secrets |
| T-83-01-02 | Denial of Service | diff gate 注释误导 | mitigate | 修正 ci.yml 注释，明确 coverage 脚本软跳过 exit 0 与 diff 脚本 fail-closed exit 2 的区别 |
| T-83-01-SC | Tampering | npm/pip/cargo installs | accept | 本 plan 不引入新包；无安装步骤 |
</threat_model>

<verification>
1. 本地运行 bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json origin/main 80，确认 profile 存在时正常输出且行为符合预期。
2. 本地运行 bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors，确认 GLOBAL PASS + 28/28 目录 PASS（本次未改覆盖率）。
3. npm run test:coverage 全量通过，159 存量测试不回归。
4. 提交前 git diff 检查：仅 ci.yml 注释与 .planning/frontend-coverage-baseline.md 有变更。
</verification>

<success_criteria>
- ci.yml 注释准确反映 diff gate fail-closed 行为（含第 219 行 stale 描述已清理）。
- 四项 gate 修复落库证据完整，无需重复实现代码。
- 试验 PR 的 CI 双 job（frontend / frontend-coverage-diff）均 success 或按预期 skip，PR 已关闭不 merge。
- 基线文档追加验证记录，.coverage-fe-floors 未变更。
</success_criteria>

<output>
Create `.planning/workstreams/frontend-coverage/phases/83-p0-70/83-01-CR01-verify-trial-SUMMARY.md` when done
</output>

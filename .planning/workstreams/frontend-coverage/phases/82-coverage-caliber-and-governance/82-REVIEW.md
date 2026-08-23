---
phase: 82-coverage-caliber-and-governance
reviewed: 2026-08-23T17:05:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - .github/scripts/check-frontend-coverage.sh
  - .github/scripts/check-frontend-diff-coverage.sh
  - .github/workflows/ci.yml
  - xingran-react-frontend/package.json
  - xingran-react-frontend/vitest.config.ts
findings:
  critical: 1
  warning: 3
  info: 6
  total: 10
status: issues_found
---

# Phase 82: Code Review Report

**Reviewed:** 2026-08-23T17:05:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

评审对象为 Phase 82 四层前端覆盖率 CI gate（全量口径测量、全局阈值 + per-dir floor + 白名单漂移三合一 gate、PR diff ≥80% gate、ci.yml 接线）。评审方式：五个文件全量精读 + 与已验证的后端孪生脚本（check-coverage.sh / check-diff-coverage.sh）逐段比对 + 本地端到端行为验证（真实 coverage-final.json 复跑两个脚本、用历史基点触发 diff join 路径、git pathspec 语义实证、vitest 源码确认 exclude 合并语义）。

**验证中确认的正面结论（简述）**：
- coverage gate 在本地真实 json 上 28/28 目录通过，floors 解析、双向登记检查、GLOBAL 加权口径均按设计工作。
- 本地 coverage-final.json 实证 571 文件 + 13 个 cad 白名单文件 = 584，与 vitest.config.ts 注释的全量口径自洽；json 中 0 个测试文件、0 个 `src/test` 条目、0 个 cad 条目——"vitest 已把测试文件挡在 coverage json 外"的注释主张成立（vitest 4 `coverageConfigDefaults.exclude` 为空、用户 exclude 整体替换默认，实测结果干净）。
- ci.yml 接线的 working-directory / artifact 上传下载路径还原逻辑正确（upload v4 LCA 剥前缀 + download `path:` 还原）。

**核心问题**：diff gate 的 git pathspec 未镜像 vitest coverage.exclude 的豁免范围（白名单 cad 目录、`*.d.ts`、`src/test/`），这些文件的合法改动会被按 0% 覆盖计罚并必然打红 PR——已在本地用真实数据端到端复现（见 CR-01）。此外，json 缺失软跳过（exit 0）构成 gate 静默失效通道（WR-01）、白名单漂移检测为无锚定子串匹配（WR-02）。

另一个值得记录的验证局限：本阶段 PR 自身只改了脚本/配置/文档，无 `src/**` 变更，因此 diff gate 在真实 CI（run 32642143749）上只走过 "nothing to gate" 跳过分支，≥80% 的 join+阈值路径从未在 CI 环境被真实 src 变更触发过（该路径本次在本地验证：行为符合设计，但同时暴露了 CR-01）。

## Narrative Findings (AI reviewer)

（本评审无 structural pre-pass，以下全部为 AI 直接评审发现。）

## Critical Issues

### CR-01: diff gate pathspec 未镜像 vitest coverage.exclude——白名单/豁免文件的合法 PR 必然被打红

**File:** `.github/scripts/check-frontend-diff-coverage.sh:135-138`（pathspec）、`215` 与 `251-253`（absent→uncovered 逻辑）；关联 `xingran-react-frontend/vitest.config.ts:24-35`

**Issue:** diff gate 的 git pathspec 只排除了 `*.test.*` 与 `__tests__/**`，但 vitest coverage.exclude 还额外排除了三类文件，它们会进入 diff 却**不在 coverage-final.json 中**：

1. `src/components/cad-editor/**`、`src/components/cad-elements/**`——D-10 白名单（13 个文件、1028 stmts）；
2. `**/*.d.ts`——现存 `src/types/externals.d.ts`、`src/types/global.d.ts`、`src/utils/sm-crypto.d.ts`；
3. `src/test/`——现存 `src/test/setup.ts`（测试基础设施）。

按脚本 L215/L251-253 的防御语义，json 中缺席文件的**所有**变更可执行行都按"已测量 + 未覆盖"计罚。因此任何触及上述文件的 PR，其 diff coverage 几乎必然 <80% 而 exit 1：
- 编辑 cad 白名单组件（白名单的豁免契约被彻底反转：全局 gate 豁免它们，PR gate 却把它们变成必败文件）；
- 新增/修改 `.d.ts` 类型声明（纯类型文件，正是脚本 L39-41 自述"pure type / declaration lines 不入分母、不惩罚 gate"所要保护的对象——该保护只对 json 内文件生效，被 exclude 的声明文件 100% 受罚）；
- 编辑 `src/test/setup.ts`（无产品风险的测试基建改动）。

**已端到端复现**：以 `156a68d^`（b4eb9da）为基点运行本脚本（本地真实 json），该"PR"仅含 `src/types/global.d.ts` 一行类型声明变更，输出：

```
UNCOVERED xingran-react-frontend/src/types/global.d.ts:22
FILE xingran-react-frontend/src/types/global.d.ts  0/1 lines 0.00%
DIFF 0 1 0.00
FAIL: diff coverage 0.00% < threshold 80.00%   (exit 1)
```

**旁证**：`git ls-files -- 'xingran-react-frontend/src/*.ts'` 实证 git pathspec 的 `*` 跨目录匹配且命中 `.d.ts`、`src/test/setup.ts` 与 cad 文件（297 个 .ts）；本地 json 实证三类文件 0 条目。本阶段验证 PR 未触及这些文件，故 CI 全绿掩盖了该缺陷。

**Fix:** 在 diff pathspec 中镜像 vitest exclude（并加同步注释，与 D-10"单一真相源"纪律对齐——两处必须同步维护）：

```bash
if ! git diff --unified=0 "${DIFF_ARGS[@]}" -- \
  'xingran-react-frontend/src/*.ts' 'xingran-react-frontend/src/*.tsx' \
  ':(exclude)xingran-react-frontend/src/*.test.*' \
  ':(exclude)xingran-react-frontend/src/**/__tests__/**' \
  ':(exclude)xingran-react-frontend/src/**/*.d.ts' \
  ':(exclude)xingran-react-frontend/src/test/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-editor/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-elements/**' | awk '
```

注：若认为"白名单文件改动也必须被 gate"是有意设计，那也应在文档中显式声明并与 D-10 白名单语义对账——当前脚本注释（L26-27 只提 test 文件、L39-41 声明类型行不受罚）与实际行为自相矛盾。

## Warnings

### WR-01: json 缺失软跳过（exit 0）把配置漂移变成 GOV-03/GOV-04 的静默失效通道

**File:** `.github/scripts/check-frontend-diff-coverage.sh:99-103`；`.github/scripts/check-frontend-coverage.sh:123-127`；关联 `.github/workflows/ci.yml:217-220`

**Issue:** 两个脚本在 coverage profile 缺失时均软跳过（exit 0）。在 CI 接线下：
- frontend job 的 gate 步骤无 `if: always()`，Test 失败时 gate 根本不会执行——软跳过唯一能触发的场景是 **Test 成功但 json 未生成**（vitest reporter 配置漂移），此时全局 gate 静默放行；
- frontend-coverage-diff job 带 `needs: frontend`，能运行到这里意味着 frontend job 已绿、json 必然存在——此时 json 缺失**只可能是 misconfiguration**（artifact 名称/路径漂移等），软跳过把这种配置错误转成 GOV-04 的静默 PASS。

ci.yml L217-220 的注释自己就承认了这一 hazard class（"gate 会在所有 PR 上命中脚本内 json 缺失软跳过而静默 exit 0 (GOV-04 失效)"），脚本却在同一场景保留了软跳过。**后端孪生脚本同位置是 `exit 2`（fail-closed）**——这是对已验证后端模式的偏离，且偏离方向是 fail-open。脚本注释给出的理由（"pairing with needs:"）恰好推出相反结论：上游失败已由 `needs:` 显性暴露，此处缺失没有软跳过的必要。

**Fix:** diff 脚本缺 profile 改为 `exit 2`（镜像后端）；coverage 脚本因需配合 `if: always()` 的 Upload 可保留软跳过，但应把提示升级为不可忽视的形式，并在 ci.yml gate 步骤后追加显式存在性断言：

```yaml
      - name: Coverage gate
        working-directory: .
        run: |
          test -s xingran-react-frontend/coverage/coverage-final.json \
            || { echo "coverage-final.json missing after successful Test step"; exit 2; }
          bash .github/scripts/check-frontend-coverage.sh \
            xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors
```

### WR-02: 白名单漂移检测为无锚定子串匹配（且大小写不敏感），误伤合法新文件

**File:** `.github/scripts/check-frontend-coverage.sh:179-185`

**Issue:** `grep -Eqi 'cad-editor|cad-elements'` 对整行 TSV 做子串匹配。真相源（vitest.config.ts）中的白名单是锚定路径 `src/components/cad-editor/**`、`src/components/cad-elements/**`，但漂移检测会命中任何位置出现的子串：合法新增的 `src/utils/cad-editor-helpers.ts`、`src/pages/cad-editor/index.tsx`、甚至大小写变体 `Cad-Editor.tsx` 都会触发 exit 6，且报错文案"restore the exclusion"指向错误的方向（新文件本应走 per-dir"未登记 floor"路径，而不是被当成 exclude 配置漂移）。检测范围宽于它声称守护的真相源，即产生假阳性 gate 失败。

**Fix:** 与 vitest exclude 的锚定范围对齐：

```bash
if printf '%s\n' "$FLAT" | grep -Eq '^xingran-react-frontend/src/components/cad-(editor|elements)/'; then
```

（如需保留 `-i`，需同时说明理由；vitest 的 glob 本身是大小写敏感的。）

### WR-03: floors 数值校验放过结构畸形值，awk 静默截断为错误阈值

**File:** `.github/scripts/check-frontend-coverage.sh:256-261`（GLOBAL）、`378-383`（per-dir，同一模式）

**Issue:** `case "$x" in ''|*[!0-9.]*)` 只排除非 `[0-9.]` 字符，不验证数字结构。`GLOBAL 3..8`、`GLOBAL 1.2.3`、`components	4..9`、`.` 之类的手误都能通过校验，随后 `awk '... threshold + 0'` 静默截断（`"3..8"+0 = 3`、`"1.2.3"+0 = 1.2`、`"."+0 = 0`）——阈值被无声改写（可能变严导致莫名失败，`0` 则让 gate 恒过变松）。

**Fix:** 用正则做结构校验（两处同改）：

```bash
if ! awk -v v="$GLOBAL_FLOOR" 'BEGIN { exit (v !~ /^[0-9]+([.][0-9]+)?$/) }'; then
  echo "check-frontend-coverage.sh: floors file $FLOORS_FILE has no numeric GLOBAL line" >&2
  exit 2
fi
```

## Info

### IN-01: node 大小写不敏感锚定与 awk 大小写敏感前缀剥离不一致

**File:** `.github/scripts/check-frontend-coverage.sh:154`（vs `293`/`358`）；`.github/scripts/check-frontend-diff-coverage.sh:187`（vs `218` join）

**Issue:** node 侧用 `toLowerCase().indexOf("xingran-react-frontend/src/")` 命中锚点但保留原始大小写切片；awk 侧 `sub(/^xingran-react-frontend\/src\//, ...)` 是大小写敏感的。若本地 checkout 目录被 rename 成不同大小写（如 `XingRan-React-Frontend`），rel 路径保留新大小写、awk 前缀剥离失败，目录键退化为整个路径前缀 → 误报"未登记 floor"（exit 4）；diff 脚本 join 键失配 → 全部按 uncovered 计。fail-closed（不会放过坏代码）但报错信息完全误导。两脚本注释均宣称 "Windows/CI dual compatible"，此不一致削弱该承诺。

**Fix:** awk 侧对 tolower 副本做前缀剥离（coverage 脚本目录键本可与 floors 同步小写），或 node 侧统一输出小写 rel（diff 脚本的 join 对端来自 git pathspec，repo 路径全小写，实证安全）。

### IN-02: diff hunk 解析的 `^\+\+\+ b/` 规则无 in_hunk 守卫

**File:** `.github/scripts/check-frontend-diff-coverage.sh:139`

**Issue:** hunk 内一条新增行若恰好以 `++ b/` 开头（如内嵌字符串字面量），会被当作新文件头处理：重置 `file` 与 `in_hunk`，该 hunk 后续新增行被静默丢弃（分母变小，gate 偏松）。后端 check-diff-coverage.sh 同款复制，属继承缺陷。

**Fix:** 给该规则加 `!in_hunk &&` 守卫并让 in_hunk 内的匹配走 `/^\+/` 分支。

### IN-03: `*` 前缀注释排除规则误伤可执行行

**File:** `.github/scripts/check-frontend-diff-coverage.sh:149-150`

**Issue:** `line !~ /^[[:space:]]*\*/` 把所有以 `*` 开头的行当 JSDoc 续行排除——generator 方法（`*handleRequest() {`）与 `/* 注释 */ code` 同行写法都是可执行行，会被排除出分母（gate 偏松）。当前代码库以 hooks 为主、罕见此写法，影响低。

**Fix:** 排除条件收紧为 `^[[:space:]]*\*([^/]|$)` 之类的形式，或接受现状并在注释中记录已知漏计。

### IN-04: `--init` 的 GLOBAL cap 3.8 为脚本内魔数，与 floors 文件形成双真相源

**File:** `.github/scripts/check-frontend-coverage.sh:225`

**Issue:** `gv = (gp < 3.8) ? gp : 3.8` 中的 3.8 是 D-14 决策值的硬编码副本。floors 的 GLOBAL 后续 ratchet 到 4.0+ 后，若有人再跑 `--init` "刷新"，GLOBAL 会被降回 `min(measured, 3.8)`——脚手架使用的真实 footgun（虽有 D-14/D-08 文档背书）。

**Fix:** `--init` 时读取现有 floors 的 GLOBAL 并取 `min(measured, 现值)`，或至少在 --init 输出头部加显著警告"会重置 GLOBAL 为 min(measured,3.8)，勿在 ratchet 后使用"。

### IN-05: `origin/${{ github.base_ref }}` 直接内插 run 命令

**File:** `.github/workflows/ci.yml:120`、`ci.yml:231`

**Issue:** GitHub 表达式直接内插进 `run:` 是模板注入的标准告警面。此处 `base_ref` 取值为 base 仓库既有分支名（制造恶意分支名需仓库写权限），实际风险低；且为镜像后端既有模式。按 GitHub 官方建议改为经 env 中转更稳妥。

**Fix:**
```yaml
        env:
          BASE_REF: ${{ github.base_ref }}
        run: bash .github/scripts/check-frontend-diff-coverage.sh \
               xingran-react-frontend/coverage/coverage-final.json "origin/$BASE_REF" 80
```

### IN-06: 验证证据局限——CI 上 join 路径未被真实 src 变更触发

**File:** `.github/workflows/ci.yml:198-231`（说明性条目）

**Issue:** 本阶段 PR（run 32642143749）只改动了脚本/工作流/配置/文档，无 `src/**` 变更，diff gate 在 CI 上实际走的是 "no testable lines changed" 跳过分支；≥80% 的 interval-join + 阈值判定路径未在真实 CI 环境被 src 变更触发过（本次评审在本地用历史基点 b4eb9da 触发验证：路径行为符合设计，同时暴露 CR-01）。"All gates verified green on real CI" 的结论对 GOV-04 主判定路径证据不足。

**Fix:** 修 CR-01 后，用一个含 src 变更（含白名单/.d.ts 文件）的试验 PR 在 CI 上回归一次 join 路径。

---

_Reviewed: 2026-08-23T17:05:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

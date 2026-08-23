---
phase: 82-coverage-caliber-and-governance
reviewed: 2026-08-23T15:22:33Z
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
  info: 7
  total: 11
status: issues_found
---

# Phase 82: Code Review Report

**Reviewed:** 2026-08-23T15:22:33Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

评审对象为 Phase 82 前端覆盖率治理管线（Vitest 4 全量口径切换、全局阈值 + per-dir floor + 白名单漂移三合一 gate 脚本、PR diff ≥80% gate 脚本、ci.yml 接线与 artifact 路径还原）。评审方式：五个文件全量精读，与已上线的后端孪生脚本（check-coverage.sh / check-diff-coverage.sh）逐段比对，并在本地用真实 `coverage-final.json`（571 文件 / 21574 stmts / 3.85%）对两个脚本做了端到端行为验证，包括全部文档化 exit code 路径（drift=6、空 profile=1、畸形 floors=2、缺失 profile 软跳过=0、坏 base ref=2、--init 多余参数=2）与 `--init` 再生成稳定性。

**实测确认无问题的部分**：
- coverage gate 在真实 json 上 28/28 目录通过；floors 解析（GLOBAL 空格分隔 + TAB 目录表 + CRLF 容错）、双向登记检查、加权口径均按设计工作；`--init` 输出与已入库的 `.coverage-fe-floors` 数据区**逐字节一致**（21574 + 1028 = 22602 stmts，与基线文档自洽）。
- json 中 0 个测试文件、0 个 `.d.ts`、0 个 cad 条目、0 个 `src/test/` 条目——vitest 4 用户 `exclude` 整体替换默认值的语义下实测干净。
- ci.yml 接线正确：gate 步骤 `working-directory: .` 覆盖、upload v4 LCA 剥前缀 + download `path:` 还原（相对后端 job 的唯一刻意偏差已正确处理并注释）、`needs:` + PR-only 条件、gate 在 upload 之前且 upload 带 `if: always()`。
- lockfile 中 vitest / @vitest/coverage-v8 / @vitest/ui 三者锁定同版本 4.1.10。

**核心缺陷（CR-01，已本地端到端复现）**：diff gate 的 git pathspec 未镜像 vitest `coverage.exclude` 的豁免范围（cad 白名单目录、`**/*.d.ts`、`src/test/`），这些文件的合法改动按"缺席 = 全部未覆盖"计罚，必然打红 PR——用一个真实的纯 prettier 格式化 commit 作基点复现了 `global.d.ts` 单行变更 → 0/1 = 0% < 80% → exit 1。

本评审为独立第二轮，结论与上一轮 REVIEW（17:05Z）在 CR-01 / 软跳过 / 漂移匹配 / 数值校验四处收敛，并新增 IN-06（版本 range 失同步隐患）；WR-03 的三种畸形值行为本次均补充了实证。

## Narrative Findings (AI reviewer)

（本评审无 structural pre-pass，以下全部为 AI 直接评审发现。）

## Critical Issues

### CR-01: diff gate pathspec 未镜像 vitest coverage.exclude——白名单/.d.ts/测试基建文件的合法 PR 必然被打红

**File:** `.github/scripts/check-frontend-diff-coverage.sh:135-138`（pathspec）、`217-254`（join + "缺席=未覆盖"语义）；关联 `xingran-react-frontend/vitest.config.ts:24-35`

**Issue:** diff gate 的 pathspec 只排除 `*.test.*` 与 `__tests__/**`，但 vitest `coverage.exclude` 还排除三类文件。它们匹配 `src/*.ts(x)` 进入 diff，却**永远不在 coverage-final.json 中**，落入 L251-254 的防御分支——"json 缺席文件的所有变更可执行行按已测量 + 未覆盖计罚"：

1. `src/components/cad-editor/**`、`src/components/cad-elements/**`（D-10 白名单，13 文件 / 1028 stmts）；
2. `**/*.d.ts`（现存 `src/types/externals.d.ts`、`src/types/global.d.ts`、`src/utils/sm-crypto.d.ts`）；
3. `src/test/`（现存 `src/test/setup.ts`）。

后果：任何触及上述文件的 PR，diff coverage 几乎必然 <80% 而 exit 1。cad 白名单的豁免契约被彻底反转（全局 gate 豁免它们，PR gate 却把它们变成必败文件）；`.d.ts` 是纯类型声明，正是脚本 L39-41 自述"pure type / declaration lines 不入分母、不惩罚 gate"要保护的对象——该保护只对 json 内文件生效，被 exclude 的声明文件 100% 受罚；改 `setup.ts` 这类零产品风险的测试基建同样受罚。

**已端到端复现**（本地真实 json，基点 `156a68d^`，该 commit 是纯 prettier 格式化）：

```
UNCOVERED xingran-react-frontend/src/types/global.d.ts:22
FILE xingran-react-frontend/src/types/global.d.ts  0/1 lines 0.00%
DIFF 0 1 0.00
FAIL: diff coverage 0.00% < threshold 80.00%   → exit 1
```

变更内容仅为 `}// 注释` → `} // 注释`（行首是 `}`，非空非注释，被计为可执行行）。旁证：json 中 `global.d.ts` 出现 0 次（grep 实证）；`git ls-files` 实证 pathspec `*.ts` 跨目录匹配且命中三类文件。本阶段验证 PR 自身无 `src/**` 变更（走 "nothing to gate" 分支），CI 全绿掩盖了该缺陷。

**Fix:** pathspec 镜像 vitest exclude（两处必须同步维护，加注释与 D-10"单一真相源"纪律对齐）：

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

若"白名单文件改动也必须被 gate"是有意设计，须在脚本头与 D-10 白名单语义显式对账——当前注释（L26-27 只列 test 排除、L39-41 声明类型行不受罚）与实际行为自相矛盾。

## Warnings

### WR-01: json 缺失软跳过（exit 0）是 GOV-04 的静默失效通道，偏离后端孪生的 fail-closed 语义

**File:** `.github/scripts/check-frontend-diff-coverage.sh:99-103`；关联 `.github/scripts/check-frontend-coverage.sh:123-127`、`.github/workflows/ci.yml:213-224`

**Issue:** 两个前端脚本在 profile 缺失时软跳过（exit 0）。对 diff 脚本而言这是 fail-open：该 job 带 `needs: frontend`，能运行到这一步意味着 frontend job 已绿、json 理应必然存在——此时 json 缺失**只可能是配置漂移**（artifact 名/路径、reporter 文件名变更等），软跳过把配置错误转成 GOV-04 的静默 PASS。后端孪生 check-diff-coverage.sh 同位置是 `exit 2`（fail-closed，L62-65）。ci.yml L217-220 的注释自己承认了这一 hazard class（"gate 会在所有 PR 上命中脚本内 json 缺失软跳过而静默 exit 0 (GOV-04 失效)"）。coverage 脚本的软跳过有 backend check-coverage.sh 对等语义背书（配合 `if: always()` Upload），可保留，但 diff 脚本的偏离方向是放松而非收紧，其注释理由（"pairing with needs:"）恰好推出相反结论。

**Fix:** diff 脚本缺 profile 改 `exit 2` 镜像后端；frontend job 的 gate 步骤前追加显式存在性断言，防止 reporter 配置漂移静默掏空全局 gate：

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

**Issue:** `grep -Eqi 'cad-editor|cad-elements'` 对整行 TSV 做子串匹配。真相源（vitest.config.ts）的白名单是锚定路径 `src/components/cad-editor/**`、`src/components/cad-elements/**`，但漂移检测命中任何位置出现的子串：合法新增的 `src/utils/cad-editor-helpers.ts`、`src/pages/cad-editor/index.tsx`、大小写变体 `Cad-Editor.tsx` 都会触发 exit 6，且报错文案"restore the exclusion"指向错误方向（新文件本应走 per-dir"未登记 floor"路径）。当前树内无此类路径（`src/utils/cad` 不匹配），属潜伏假阳性。检测范围宽于它声称守护的真相源。

**Fix:** 锚定到真相源的路径范围：

```bash
if printf '%s\n' "$FLAT" | grep -Eq '^xingran-react-frontend/src/components/cad-(editor|elements)/'; then
```

（vitest glob 本身大小写敏感，`-i` 一并去掉。）

### WR-03: floors 数值校验放过结构畸形值，awk 静默截断为错误阈值（含"gate 恒过"）

**File:** `.github/scripts/check-frontend-coverage.sh:256-261`（GLOBAL）、`378-383`（per-dir，同一模式）

**Issue:** `case "$x" in ''|*[!0-9.]*)` 只挡非 `[0-9.]` 字符，不验证数字结构。`.coverage-fe-floors` 是 ratchet 工作流指定**手工编辑**的数据文件（D-07），手误概率真实存在。三种畸形值已实测：

- `GLOBAL .` → 通过校验，`"."+0 = 0` → 输出 `PASS: weighted avg 3.85% >= threshold 0.00%`，**全局 gate 静默失效**（任意覆盖率恒过）；
- `GLOBAL 3..8` → 阈值静默变 3.00；
- `components<TAB>4..9` → floor 静默变 4，`PASS: components 5.43% >= 4..9%`（本应 4.9 拦截）。

**Fix:** 两处改结构校验（同时拒绝空整数部）：

```bash
if ! awk -v v="$GLOBAL_FLOOR" 'BEGIN { exit (v !~ /^[0-9]+([.][0-9]+)?$/) }'; then
  echo "check-frontend-coverage.sh: floors file $FLOORS_FILE has no numeric GLOBAL line" >&2
  exit 2
fi
```

## Info

### IN-01: node 大小写不敏感锚定与 awk 大小写敏感前缀剥离不一致

**File:** `.github/scripts/check-frontend-coverage.sh:154`（vs `293`/`358`）；`.github/scripts/check-frontend-diff-coverage.sh:187`（vs `218` join）

**Issue:** node 侧 `toLowerCase().indexOf("xingran-react-frontend/src/")` 命中锚点但保留原始大小写切片；awk 侧 `sub(/^xingran-react-frontend\/src\//, ...)` 大小写敏感。本地 checkout 目录改大小写（如 `XingRan-React-Frontend`）时前缀剥离失败 → 目录键退化为整段路径 → 误报"未登记 floor"（exit 4）；diff join 键失配 → 全按 uncovered 计。fail-closed 但报错误导，与两脚本"Windows/CI dual compatible"的注释承诺不符。

**Fix:** awk 对 `tolower($1)` 副本做前缀剥离，或 node 统一输出小写 rel（diff join 对端来自 git pathspec，repo 路径全小写，实证安全）。

### IN-02: diff hunk 解析的 `^\+\+\+ b/` 规则无 in_hunk 守卫（继承自后端）

**File:** `.github/scripts/check-frontend-diff-coverage.sh:139`（后端 `check-diff-coverage.sh:88` 同款）

**Issue:** hunk 内新增行内容恰以 `+ b/` 开头（拼成 `+++ b/...`）会被误判为文件头，重置 `file`/`in_hunk`，该 hunk 后续新增行被静默丢弃（分母变小，gate 偏松）。概率低，属后端脚本的继承缺陷。

**Fix:** 该规则加 `!in_hunk &&` 守卫。

### IN-03: `*` 前缀注释排除规则误伤可执行行

**File:** `.github/scripts/check-frontend-diff-coverage.sh:149-150`

**Issue:** `line !~ /^[[:space:]]*\*/` 把所有 `*` 开头的行当 JSDoc 续行排除——generator 方法（`*handleRequest() {`）与 `/* */ code` 同行写法都是可执行行，会被排出分母（gate 偏松）。当前代码库罕见此写法。

**Fix:** 收紧为 `^[[:space:]]*\*([^/]|$)` 或注释记录已知漏计。

### IN-04: `--init` 的 GLOBAL cap 3.8 为脚本内魔数，与 floors 文件形成双真相源

**File:** `.github/scripts/check-frontend-coverage.sh:225`

**Issue:** `gv = (gp < 3.8) ? gp : 3.8` 是 D-14 决策值的硬编码副本。GLOBAL 后续 ratchet 到 4.0+ 后再跑 `--init`，GLOBAL 会被重置回 `min(measured, 3.8)`——脚手架误用的真实 footgun。

**Fix:** `--init` 读取现有 floors 的 GLOBAL 取 `min(measured, 现值)`，或输出头部加显著警告。

### IN-05: `origin/${{ github.base_ref }}` 直接内插 run 命令

**File:** `.github/workflows/ci.yml:120`、`ci.yml:231`

**Issue:** GitHub 表达式直接内插进 `run:` 属模板注入标准告警面。`base_ref` 取值需要 base 仓库写权限才能投毒，实际风险低，且为后端既有模式镜像。

**Fix:** 经 `env:` 中转：`env: { BASE_REF: "${{ github.base_ref }}" }` + `"origin/$BASE_REF"`。

### IN-06: vitest 与 @vitest/coverage-v8 的 caret range 允许失同步（当前 lock 对齐）

**File:** `xingran-react-frontend/package.json:76,98`

**Issue:** `vitest: ^4.0.18` 与 `@vitest/coverage-v8: ^4.1.10` 声明基线不同。lockfile 已把两者锁到同版本 4.1.10（含 @vitest/ui），CI 用 `npm ci` 确定性安装，当前无实际故障；但 vitest 要求 coverage provider 与核心版本匹配，未来一次 `npm update` 可能把 vitest 升到 4.2.x 而 provider 留在 4.1.x，`--coverage` 直接报错。

**Fix:** 三个 `@vitest/*` 包声明同一 range（如统一 `^4.1.10`），与后端"CI 缺包但本地正常先查依赖声明"的教训对齐。

### IN-07: 验证证据局限——GOV-04 join 路径未在真实 CI 上被 src 变更触发过

**File:** `.github/workflows/ci.yml:198-231`（说明性条目）

**Issue:** 本阶段 PR 只改脚本/工作流/配置/文档，无 `src/**` 变更，diff gate 在 CI 上实际只走过 "nothing to gate" 跳过分支；≥80% 的 interval-join + 阈值主判定路径从未在 CI 环境被真实 src 变更触发（本次评审在本地用历史基点触发验证：行为符合设计，同时暴露 CR-01）。"CI 全绿"对 GOV-04 主路径证据不足。

**Fix:** 修完 CR-01 后，用一个含 src 变更（含白名单 / .d.ts 文件）的试验 PR 回归一次 join 路径。

---

_Reviewed: 2026-08-23T15:22:33Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

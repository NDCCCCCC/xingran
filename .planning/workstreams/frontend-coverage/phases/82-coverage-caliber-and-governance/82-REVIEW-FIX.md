---
phase: 82-coverage-caliber-and-governance
fixed_at: 2026-08-23T16:30:36Z
review_path: .planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 82: Code Review Fix Report

**Fixed at:** 2026-08-23T16:30:36Z
**Source review:** .planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4（CR-01 + WR-01/02/03，fix_scope=critical_warning，Info 不在范围内）
- Fixed: 4
- Skipped: 0

修复均在隔离 git worktree（临时分支 `gsd-reviewfix/82-revfix002223`，基点 = main @ d6cd0bb）内完成，
每个 finding 一个原子 commit，经 husky 钩子（lint-staged + commitlint）校验后落库。清理阶段发现
main 在修复期间前进了两个纯 docs commit（`7c4e240`、`4885010`，仅动 `.planning/`，与修复路径零交集），
4 个 fix commit 已 rebase 到新 main 之上（脚本树与 rebase 前逐字节一致，`git diff` 零输出）后
fast-forward 回 main，历史保持线性，未触碰用户提交。commit subject 中的 finding id 为小写
（`cr-01` 等），是 commitlint `subject-case` 规则的要求，与本报告中的规范 ID 一一对应。

## Fixed Issues

### CR-01: diff gate pathspec 未镜像 vitest coverage.exclude——白名单/.d.ts/测试基建文件的合法 PR 必然被打红

**Files modified:** `.github/scripts/check-frontend-diff-coverage.sh`
**Commit:** 60f712c（`fix(82): cr-01 mirror vitest coverage.exclude in diff gate pathspec`）
**Applied fix:** git diff pathspec 追加四个 `:(exclude)`，与 `vitest.config.ts` 的 `coverage.exclude`
数组逐项镜像：`src/**/*.d.ts`、`src/test/**`、`src/components/cad-editor/**`、
`src/components/cad-elements/**`（原有 `*.test.*` 与 `__tests__/**` 保留）。同时更新脚本头注释，
写明"pathspec 与 vitest exclude 数组必须在同一 commit 内同步维护"（D-10 单一真相源纪律）。

**验证（全部在修复后实测）：**
- 评审复现重跑：`check-frontend-diff-coverage.sh <json> 156a68d^ 80` →
  `no testable .ts/.tsx lines changed — PASS (nothing to gate)`，exit 0（修复前同命令 exit 1，
  `global.d.ts:22` 计为 0/1 未覆盖——pre-fix 基线已先行固化）。
- 真实 src 文件仍正常进入 diff：detached worktree @ `d2184fd`（prettier commit，含 5 个真实
  .tsx + 1 个 `__tests__/*.test.tsx`）叠加修复后脚本端到端运行：sidebar.tsx（0/8）、
  email-config.tsx（18/18）、api-config.tsx（8/8）等全部进入 FILE/DIFF 统计（DIFF 28/36 = 77.78%
  → exit 1，gate 正常咬合），`__tests__/SettingsShell.test.tsx` 与 `login.css` 正确缺席。
- cad 白名单排除实证：`9beadd4`（实际触及 cad-editor 文件的 commit）在修复后 pathspec 下，
  95 个真实 src 文件照常进入 diff，cad-editor / cad-elements / *.d.ts / src/test 四类零出现。

### WR-01: json 缺失软跳过（exit 0）是 GOV-04 的静默失效通道

**Files modified:** `.github/scripts/check-frontend-diff-coverage.sh`
**Commit:** 27f275e（`fix(82): wr-01 fail-closed exit 2 on missing profile in diff gate`）
**Applied fix:** diff 脚本的缺失 profile 分支从软跳过 `exit 0` 改为 fail-closed `exit 2`（镜像后端
孪生 `check-diff-coverage.sh` L62-65），诊断信息保留在 stderr；头注释与 exit-code 文档同步更新。
注释同时说明 coverage gate 脚本（`check-frontend-coverage.sh`）的软跳过**有意保留**——它与
frontend job 自身的 `if: always()` Upload 步骤配对，有后端 `check-coverage.sh` 对等语义背书（评审结论）。
按 orchestrator 划定范围，ci.yml 未改动：其 download `path:` 还原 + L213-224 的漂移防护注释与
fail-closed 语义一致。

**验证：** 缺失 profile → exit 2 + stderr 诊断（pre-fix 基线为 exit 0 软跳过，已固化）；坏 base ref →
exit 2；usage error → exit 2；CR-01 复现场景回归 → 仍 exit 0（profile 存在时 fail-closed 不触发）。

### WR-02: 白名单漂移检测为无锚定子串匹配（且大小写不敏感），误伤合法新文件

**Files modified:** `.github/scripts/check-frontend-coverage.sh`
**Commit:** 94d3a16（`fix(82): wr-02 anchor whitelist drift grep to cad dir path prefix`）
**Applied fix:** 漂移检测两处 grep（判定 + 明细列举）从 `grep -Eqi 'cad-editor|cad-elements'` 改为
锚定路径前缀 `grep -Eq '^xingran-react-frontend/src/components/cad-(editor|elements)/'`，并去掉
`-i`（vitest glob 本身大小写敏感）；区块注释写明锚定理由与假阳性示例。

**验证（合成 fixture 实测）：** `utils/cad-editor-helpers.ts` → exit 0（旧逻辑必误报 exit 6）；
`components/cad-editor/Foo.tsx` → exit 6 且明细正确列出该文件（真漂移仍拦截）；
`components/Cad-Editor/x.ts`（大小写变体目录，vitest glob 不会排除它）→ 不再误报，走正常
per-dir 路径 exit 0；真实 json 回归 → PASS 28/28。

### WR-03: floors 数值校验放过结构畸形值，awk 静默截断为错误阈值（含"gate 恒过"）

**Files modified:** `.github/scripts/check-frontend-coverage.sh`
**Commit:** aa3bf0c（`fix(82): wr-03 structural numeric validation for floors values`）
**Applied fix:** GLOBAL 行与 per-dir 行两处校验从字符集 case（`''|*[!0-9.]*`）改为结构化正则
`awk -v v=... 'BEGIN { exit (v !~ /^[0-9]+([.][0-9]+)?$/) }'`，畸形值按文档 exit 2 拒绝；
合法格式不变（`GLOBAL <float>` 空格分隔 + `<dir><TAB><float>` 行）。

**验证（实测矩阵）：** `GLOBAL .`（旧：阈值 0 恒过）→ exit 2；`GLOBAL 3..8`（旧：静默 3.00）→
exit 2；`GLOBAL -1` / `GLOBAL abc` / 缺 GLOBAL 行 → exit 2；`components 4..9`（旧：floor 静默
变 4）→ exit 2 且报 `got '4..9'`；`components -1` / `abc` → exit 2；入库 `.coverage-fe-floors`
全部 29 个值（GLOBAL 3.8 + 28 目录）均通过新校验。

## 收尾全量验证（四修复合入后一次性重跑）

- `check-frontend-coverage.sh <真实 json> .coverage-fe-floors` → `PASS: weighted avg 3.85% >=
  threshold 3.80%`（TOTAL 21574 stmts）+ `PASS: per-dir floor gate — 28/28 directories`，exit 0。
- `check-frontend-diff-coverage.sh <真实 json> HEAD 80` 与 `origin/main 80`（空 src diff 基点）→
  均走 nothing-to-gate，exit 0。
- exit-code 矩阵复核：drift=6、malformed floors=2、diff 脚本 missing profile=2（本次修复后的
  新语义）、bad ref=2——与两脚本头部文档一致。
- `--init` 再生成与入库 `.coverage-fe-floors` **逐字节一致**（`diff` 零输出）；`.coverage-fe-floors`
  内容零改动（GLOBAL 3.8 + 28 dirs 原样）。
- 变更范围审计：`git diff --stat 4885010..aa3bf0c`（用户 docs 提交之后 → fix 顶端）仅两个 gate
  脚本，无 Info 修复、无 ci.yml / floors / 文档外溢。

## 备注（供 orchestrator / 后续参考）

- ci.yml L213-224 download 步骤注释中"gate 会在所有 PR 上命中脚本内 json 缺失软跳过而静默
  exit 0"一句描述的是修复前行为；WR-01 修复后该场景表现为显性 exit 2（job 红、可见），注释
  依据的 `path:` 还原必要性不变。按范围约定未改动 ci.yml，如需注释措辞对齐可随后续 docs
  commit 处理。
- WR-02/WR-03 属逻辑行为变更，Tier-1/2 之外已按上述 fixture 矩阵做了语义级实测（非仅语法
  验证），矩阵全部命中预期。

---

_Fixed: 2026-08-23T16:30:36Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_

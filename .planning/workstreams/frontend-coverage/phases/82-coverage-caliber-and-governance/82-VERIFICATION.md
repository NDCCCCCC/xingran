---
phase: 82-coverage-caliber-and-governance
verified: 2026-08-24T00:42:00Z
status: passed
score: 14/14 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 14/14
  gaps_closed:
    - "CR-01: diff gate pathspec 已镜像 vitest coverage.exclude（cad 白名单 / **/*.d.ts / src/test/）"
    - "WR-01: diff gate 缺失 coverage profile 已改为 fail-closed exit 2"
    - "WR-02: 白名单漂移检测已锚定到 src/components/cad-(editor|elements)/ 路径前缀"
    - "WR-03: floors 数值已加结构化正则校验（拒绝 3..8 / . / -1 等畸形值）"
  gaps_remaining: []
  regressions: []
---

# Phase 82: 口径修正与治理基建 — Verification Report（复审后验证）

**Phase Goal:** vitest 全量口径切换（GOV-01：修正 Vitest 4 移除 coverage.all 后的失真旧口径 24.58%，全量真基线 3.67%）+ 白名单登记（cad-editor 804 + cad-elements 224 = 1028 stmts ≈ 4.5% ≤ 5% 上限，GOV-02/QUAL-02）+ 4 层 CI 防倒退 gate 落地（全局阈值 / per-dir floor / baseline ratchet / PR diff coverage ≥80%，GOV-03/GOV-04/GOV-05）
**Verified:** 2026-08-24T00:42:00Z
**Status:** passed
**Re-verification:** Yes — 针对 82-REVIEW.md / 82-REVIEW-FIX.md 关闭的 CR-01、WR-01、WR-02、WR-03 进行复审后验证

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | SC-1/GOV-01：vitest 全量口径生效——`include: ["src/**/*.{ts,tsx}"]` 让 584→571 文件进入报告，gate 口径 3.85%（830/21574），未测文件 0% 计入 | ✓ VERIFIED | `xingran-react-frontend/vitest.config.ts:22` 含字面 `include`；本地 `coverage-final.json` 复算 `files=571 cad=0 stmts=21574 cov=830 pct=3.85`；CI run 32652540378 `PASS: weighted avg 3.85% >= threshold 3.80%` |
| 2 | GOV-01/D-16：coverage.thresholds 整段删除，gate 真相源移交外部脚本 | ✓ VERIFIED | `grep -c "thresholds:" xingran-react-frontend/vitest.config.ts` = 0；D-16 替代注释在 L36-38 |
| 3 | D-03：text/json/html 三 reporter 保留；`test:coverage` 单次运行语义 | ✓ VERIFIED | `vitest.config.ts:23` reporter 三项；`package.json:19` 字面 `"test:coverage": "vitest run --coverage"` |
| 4 | 159 测试零回归（19 文件全绿） | ✓ VERIFIED | CI run 32652540378 frontend job log：`Test Files 19 passed (19)` / `Tests 159 passed (159)` |
| 5 | SC-2/QUAL-02：白名单三项登记（理由/面积/复审条件）+ coverage exclude 同步，合计 1028/13 = 4.55% ≤ 5% | ✓ VERIFIED | `vitest.config.ts:33-34` exclude 两项；`.planning/frontend-coverage-baseline.md` 白名单表三列齐全，含 D-12 锁死 + D-10 同源声明与 D-11 定量复审条件 |
| 6 | SC-3/GOV-02：基线文档落盘（ratchet 表对称后端 schema + 28 目录 per-dir 快照 + SHA 回填 + CI 读数行，无 TBD 残留） | ✓ VERIFIED | 文档存在 115 行；ratchet 表列名含 `ratchet_from`/`ratchet_to`/`0pct_pkg_count`；起点 commit=`bddb2fc`；CI 读数行 commit=`8c7b69f`；`grep -c "TBD (atomic ratchet)"` = 0 |
| 7 | GOV-03：全局阈值 gate statements 维度加权判定，≥GLOBAL exit 0 / 不足 exit 1 | ✓ VERIFIED | 本地副本 `GLOBAL 5.0` → exit 1；真实数据 → exit 0 并输出 `PASS: weighted avg 3.85% >= threshold 3.80%` |
| 8 | GOV-05：per-dir floor 28 目录双向比对，违例独立 exit 4 | ✓ VERIFIED | 本地副本 `components 90.0` → exit 4（FAIL 明细）；真实数据 → `PASS: per-dir floor gate — 28/28 directories >= floor` |
| 9 | GOV-05/D-10：白名单漂移 fail-fast exit 6（先于聚合） | ✓ VERIFIED | 注入 `src/components/cad-editor/Fake.tsx` → exit 6；合法文件名 `src/utils/cad-editor-helpers.ts` → exit 0（WR-02 修复后不误报） |
| 10 | GOV-04：diff gate <80% exit 1 + UNCOVERED 清单；空 diff 软过 exit 0；缺参/坏 ref exit 2 | ✓ VERIFIED | 空树合成基线 threshold 80 → exit 1 + FAIL/UNCOVERED；threshold 0 → exit 0 + `DIFF 7376 77481 9.52`；无参/坏 ref → exit 2；base=HEAD → exit 0 |
| 11 | GOV-04 CI 编排：frontend-coverage-diff PR-only job + download path 还原（gate 实读 profile 非软跳过） | ✓ VERIFIED | `ci.yml:198-231` 含 `needs: frontend` / `if: github.event_name == 'pull_request'` / `fetch-depth: 0` / `download-artifact@v4 with.path: xingran-react-frontend/coverage` |
| 12 | GOV-03 CI 编排：frontend job 内嵌 Coverage gate（working-directory: `.` 且无 `if: always()`），exit 1/4/6 任一即 job 红 | ✓ VERIFIED | `ci.yml:166-179` 步骤序 Test (coverage) → Coverage gate → Upload artifact → Build；gate 步骤 `working-directory: .`，无 `if: always()` |
| 13 | 共享编辑区纪律：backend 与 coverage-diff 两 job 零字节改动 | ✓ VERIFIED | `diff <(git show main:.github/workflows/ci.yml 区间) <(工作树区间)` = empty；`timeout-minutes: 15` 在 ci.yml 中仅出现 1 次（frontend job） |
| 14 | 代码评审修复闭环：CR-01 / WR-01 / WR-02 / WR-03 已在 main（origin/main HEAD `9f9a72e`）修复，且 main push run 32652540378 全绿、frontend-coverage-diff 为 skipped | ✓ VERIFIED | 源码确认：diff gate pathspec 镜像 exclude（CR-01）、缺失 profile exit 2（WR-01）、漂移 grep 锚定路径前缀（WR-02）、floors 数值正则校验（WR-03）；`gh run view 32652540378` 显示 frontend=success、backend=success、frontend-coverage-diff=skipped、headSha=9f9a72e |

**Score:** 14/14 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `xingran-react-frontend/vitest.config.ts` | 全量口径 coverage 配置（include + 白名单 exclude，无 thresholds） | ✓ VERIFIED | 47 行实质内容，include/exclude/reporter 全部字面命中 |
| `xingran-react-frontend/package.json` | test:coverage 单次运行语义 | ✓ VERIFIED | L19 字面 `vitest run --coverage` |
| `.github/scripts/check-frontend-coverage.sh` | 全局阈值 + per-dir floor + 漂移检测 + --init（≥150 行） | ✓ VERIFIED | 444 行（≥ min_lines 150）；五分支 exit 矩阵本地重跑全命中 |
| `.coverage-fe-floors` | GLOBAL 3.8 + 28 目录 floor 表 | ✓ VERIFIED | 29 非注释行（1 GLOBAL + 28 目录），`(src root)`/`api` 在列；git 追踪 |
| `.github/scripts/check-frontend-diff-coverage.sh` | PR diff ≥80% gate 三段式（≥120 行） | ✓ VERIFIED | 314 行（≥ min_lines 120）；CR-01 修复后 pathspec 与 vitest exclude 逐项镜像 |
| `.github/workflows/ci.yml` | frontend 内嵌 gate + PR-only diff job 编排 | ✓ VERIFIED | YAML 解析通过；backend/coverage-diff 区间与 main 逐字节一致 |
| `.planning/frontend-coverage-baseline.md` | 基线文档（ratchet + per-dir 快照 + 白名单登记，≥80 行） | ✓ VERIFIED | 115 行（≥ min_lines 80）；数字与 json 复算一致 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `vitest.config.ts` coverage.include | `coverage-final.json` 文件集合 | vitest v8 全量计入 | ✓ WIRED | 本地 json 571 文件 + cad 0 条目实证 |
| `coverage.exclude` 白名单 | json 无 cad 条目 | exclude 过滤 | ✓ WIRED | `cad=0` 复算命中 |
| `ci.yml` frontend Coverage gate 步骤 | `check-frontend-coverage.sh` + `.coverage-fe-floors` | `working-directory: .` 双参数调用 | ✓ WIRED | `ci.yml:179` 字面调用；CI log PASS |
| `ci.yml` frontend-coverage-diff job | `check-frontend-diff-coverage.sh` + frontend-coverage artifact | download path 还原 + origin/base_ref 三参数 | ✓ WIRED | `ci.yml:231`；CR-01 修复后 pathspec 与 exclude 镜像 |
| Upload coverage artifact | frontend-coverage-diff job | artifact 名 `frontend-coverage` | ✓ WIRED | PR diff job 依赖 `needs: frontend` 消费 artifact |
| `vitest.config.ts` coverage.exclude | `check-frontend-diff-coverage.sh` pathspec | 同一 commit 同步维护（D-10 单一真相源） | ✓ WIRED | CR-01 fix 后 diff gate `:(exclude)` 与 exclude 数组逐项对应 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `check-frontend-coverage.sh` | FLAT (TSV) | `xingran-react-frontend/coverage/coverage-final.json` | 是（571 文件真实聚合 21574/830） | ✓ FLOWING |
| `check-frontend-diff-coverage.sh` | CHANGED / FLAT | git diff 真实输出 + coverage-final.json | 是（空 diff / 合成基线 / CR-01 复现均按预期 join） | ✓ FLOWING |
| `.coverage-fe-floors` | floor 表 | `--init` 从真实 json 生成 | 是（与 json 复算逐值对应） | ✓ FLOWING |
| `.planning/frontend-coverage-baseline.md` | ratchet/per-dir 数字 | json 复算 + CI 读数 | 是（21574/830/3.85 双侧一致） | ✓ FLOWING |

无 HOLLOW / ORPHANED / STATIC 产物——全部 gate 数学消费真实测量数据。

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| 主 gate 真实数据判定 | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` | exit 0；`PASS: weighted avg 3.85% >= threshold 3.80%` + 28/28 目录 PASS | ✓ PASS |
| 主 gate 全局阈值失败分支 | 副本 floors `GLOBAL 5.0` | exit 1 | ✓ PASS |
| 主 gate floor 违例分支 | 副本 floors `components 90.0` | exit 4（FAIL 明细） | ✓ PASS |
| 主 gate 白名单漂移真阳性 | json 注入 `src/components/cad-editor/Fake.tsx` | exit 6 | ✓ PASS |
| 主 gate 白名单漂移无假阳性（WR-02） | json 注入 `src/utils/cad-editor-helpers.ts` | exit 0（不触发漂移） | ✓ PASS |
| 主 gate 用法错误 | 无参调用 | exit 2 | ✓ PASS |
| floors 数值结构校验（WR-03） | 副本 floors `GLOBAL .` / `components 4..9` | 均 exit 2 | ✓ PASS |
| diff gate 空 diff 软过 | base=HEAD threshold=80 | exit 0 + `no testable .ts/.tsx lines changed` | ✓ PASS |
| diff gate 缺失 profile fail-closed（WR-01） | `/nonexistent.json HEAD 80` | exit 2 + 配置漂移诊断 | ✓ PASS |
| diff gate CR-01 复现 | base=`156a68d^` threshold=80 | exit 0 + `no testable .ts/.tsx lines changed`（`.d.ts` 已被排除） | ✓ PASS |
| diff gate FAIL/PASS 主路径 | 空树合成基线 threshold 80 / 0 | exit 1 + FAIL/UNCOVERED / exit 0 + PASS；`DIFF 7376 77481 9.52` | ✓ PASS |
| CI：main push run 全绿 | `gh run view 32652540378 --json jobs` | frontend=success / backend=success / frontend-coverage-diff=skipped / headSha=9f9a72e | ✓ PASS |
| CI：前端测试回归 | run 32652540378 log grep | `Test Files 19 passed (19)` / `Tests 159 passed (159)` | ✓ PASS |
| CI：gate PASS 读数 | run 32652540378 log grep | `PASS: weighted avg 3.85% >= threshold 3.80%` / `PASS: per-dir floor gate — 28/28 directories >= floor` | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| （本 phase 无 `scripts/*/tests/probe-*.sh` 约定探针；验证以 spot-check 形式在本进程独立重跑） | — | — | — |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| GOV-01 | 82-01 | vitest 全量口径切换（include 圈定 + 白名单 exclude + 未测文件计入） | ✓ SATISFIED | Truth 1/2/3；`vitest.config.ts` L18-38 + 本地 571 口径复算 |
| GOV-02 | 82-04 | 前端覆盖率基线落盘（起点 + per-dir 快照 + ratchet 记录，对称后端） | ✓ SATISFIED | Truth 6；`frontend-coverage-baseline.md` 全结构核对 |
| GOV-03 | 82-02, 82-04, 82-05 | CI 全局阈值 gate 全量口径 + ratchet 只升不降 + 失败即阻断 | ✓ SATISFIED | Truth 7/12/14；本地 exit 1 分支 + CI PASS 日志 |
| GOV-04 | 82-03, 82-04, 82-05 | PR diff coverage ≥80% gate（bash+awk 自实现，前端版） | ✓ SATISFIED | Truth 10/11/14；CR-01 修复后 pathspec 镜像 exclude；CI PR-only job 生效 |
| GOV-05 | 82-02 | per-directory floor gate（独立 exit code 可区分） | ✓ SATISFIED | Truth 8/9；exit 4/6 独立分支实证 |
| QUAL-02 | 82-01, 82-04 | 白名单治理（理由/面积/复审条件登记 + ≤5%） | ✓ SATISFIED | Truth 5；登记表三列 + D-11/D-12/D-10 声明 |

Orphaned requirements：无——`REQUIREMENTS.md` Traceability 表映射到 Phase 82 的恰为上述 6 项，且全部在 PLAN frontmatter 中声明并闭环。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact / 处置 |
| ---- | ---- | ------- | -------- | ------------- |
| 全部 7 个交付文件 | — | TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER | 无 | 扫描 0 命中；`TBD (atomic ratchet)` 已清理（grep=0） |

### Human Verification Required

无。82-05 Task 2 的 `checkpoint:human-verify` 已由用户 approved（记录于 commit `468a624`），且本次复审后的可编程检查（CR-01/WR-01/WR-02/WR-03 修复验证、CI run 32652540378、本地 gate 矩阵）已全部在本进程内独立重跑通过；无视觉/实时/外部服务类残留项。

### Gaps Summary

无 must-have 缺口。14/14 truths 全部 VERIFIED，6/6 requirements 全部 SATISFIED，7 个交付物全部实质且接线且数据流真实，关键链接与真实 CI 日志级证据完整。

代码评审修复（CR-01 / WR-01 / WR-02 / WR-03）已在 origin/main HEAD `9f9a72e` 闭环：

- **CR-01**：`check-frontend-diff-coverage.sh:144-151` 的 pathspec 已追加 `:(exclude) **/*.d.ts`、`src/test/**`、`cad-editor/**`、`cad-elements/**`，与 `vitest.config.ts:24-35` 的 `coverage.exclude` 逐项镜像；本地复现命令 `bash ... 156a68d^ 80` 从修复前 exit 1 变为 exit 0。
- **WR-01**：diff gate 缺失 profile 分支从 exit 0 软跳过改为 exit 2 fail-closed（`check-frontend-diff-coverage.sh:109-111`），与后端 `check-diff-coverage.sh` 语义一致。
- **WR-02**：主 gate 漂移检测从 `grep -Eqi 'cad-editor|cad-elements'` 改为锚定路径前缀 `^xingran-react-frontend/src/components/cad-(editor|elements)/`（`check-frontend-coverage.sh:183`），合法新文件 `utils/cad-editor-helpers.ts` 不再误报。
- **WR-03**：GLOBAL 与 per-dir floor 数值从字符集校验改为结构化正则 `/^[0-9]+([.][0-9]+)?$/`（`check-frontend-coverage.sh:264`、`286-294`、`401`），`GLOBAL .`、`GLOBAL 3..8`、`components 4..9` 均被 exit 2 拒绝。

main push run **32652540378**（push 到 main，headSha=9f9a72e）结论：frontend=success、backend=success、frontend-coverage-diff=skipped、coverage-diff=skipped；frontend job log 显示 `Tests 159 passed (159)`、`PASS: weighted avg 3.85% >= threshold 3.80%`、`PASS: per-dir floor gate — 28/28 directories >= floor`。

Phase 82 目标已达成，可继续下一 phase。

---

_Verified: 2026-08-24T00:42:00Z_
_Verifier: Claude (gsd-verifier)_

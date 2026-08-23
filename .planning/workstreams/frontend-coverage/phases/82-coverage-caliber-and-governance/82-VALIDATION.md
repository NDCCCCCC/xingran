---
phase: 82
slug: coverage-caliber-and-governance
status: ready
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-23
---

# Phase 82 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.10（jsdom + @testing-library；现有 19 文件 159 测试） |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npm run test` |
| **Full suite command** | `cd xingran-react-frontend && npm run test:coverage`（全量口径，本地实测约 4-7 分钟含 transform） |
| **Estimated runtime** | ~300 秒 |

> 本 phase **不新增 vitest 测试文件**（SC 明确「不新增任何测试」）。gate 脚本按后端范式不配单测（CI 即验收场），但每个 gate 脚本必须支持本地对既有 coverage-final.json 干跑验证。

---

## Sampling Rate

- **After every task commit:** Run `cd xingran-react-frontend && npm run test:coverage`（19 文件 159 测试全绿 + json 正常产出）+ 对应 gate 脚本本地干跑 exit 0
- **After every plan wave:** 全套 gate 干跑（全局/floor/diff 三脚本全 exit 0）+ gate 失败分支注入演练（每脚本至少一次非零 exit 验证）
- **Before `/gsd:verify-work`:** 开一个 PR 触发真实 CI——frontend job 绿（含 gate 步骤）+ frontend-coverage-diff job 出现在 PR checks + push main 无 diff job（PR-only 生效证明）
- **Max feedback latency:** ~300 秒

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 82-01-T1 | 82-01 | 1 | GOV-01 | T-82-01-01/02 | N/A | smoke（产物断言） | `npm run test:coverage && node -e` 断言 files=584 + `! grep -q "thresholds:" vitest.config.ts` + package.json run 语义 grep | ⬜ pending |
| 82-01-T2 | 82-01 | 1 | GOV-01, QUAL-02 | T-82-01-01 | N/A | smoke（产物断言） | `npm run test:coverage && node -e` 断言 files=571 / cad=0 / stmts=21574 / cov=830 / pct=3.85 + exclude 两项 grep -cF =1 | ⬜ pending |
| 82-02-T1 | 82-02 | 2 | GOV-03, GOV-05 | T-82-02-01/03/04 | gate 不静默通过 | integration（--init 干跑） | `bash -n` + `--init` 输出 29 行（1 GLOBAL + 28 目录，含 (src root)/api） | ⬜ pending |
| 82-02-T2 | 82-02 | 2 | GOV-03, GOV-05 | T-82-02-02/03 | exit 1/4/6 独立可区分 | integration（gate 干跑矩阵） | 五分支矩阵：真实 0 / 无参 2 / GLOBAL 5.0→1 / components 90.0→4 / cad 注入→6 + --init 幂等 diff | ⬜ pending |
| 82-03-T1 | 82-03 | 2 | GOV-04 | T-82-03-01/02 | base-ref 校验 + 引号 | integration（软过干跑） | `bash -n` + 无参→2 + 坏 ref→2 + base=HEAD 空 diff→0 | ⬜ pending |
| 82-03-T2 | 82-03 | 2 | GOV-04 | T-82-03-03 | 新文件不免费通行 | integration（合成基线） | 空树 commit-tree 基线：threshold 80→exit 1 + UNCOVERED 清单；threshold 0→exit 0 + DIFF 数值行 | ⬜ pending |
| 82-04-T1 | 82-04 | 3 | GOV-03, GOV-04 | T-82-04-01/02 | backend job 零改动 | integration（YAML/结构断言） | pyyaml 解析 + `git show main:` 区间 diff 断言共享 job 未变 + frontend 步骤序 + diff job 四件套 grep | ⬜ pending |
| 82-04-T2 | 82-04 | 3 | GOV-02, QUAL-02 | T-82-04-04 | 数字可复算 | manual-only（文档审查）+ 自动核算 | node -e 复算 json stmts/cov 与文档记载一致 + 白名单三列/4.55%/ratchet schema grep | ⬜ pending |
| 82-05-T1 | 82-05 | 4 | GOV-03, GOV-04 | T-82-05-01/02 | 校准只降不升留档 | integration（真实 CI） | gh 证据链：PR checks 含 diff job / main push 无 diff job + frontend success / SHA 回填无 TBD 残留 | ⬜ pending |
| 82-05-T2 | 82-05 | 4 | 全 phase | — | N/A | checkpoint:human-verify | 人工核验 PR 双绿 + main job 列表 + 本地三件套 + 基线文档审阅 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- 无 vitest 测试缺口——本 phase 零新增测试，现有基础设施（19 文件 159 测试）覆盖回归验证。
- gate 脚本（`check-frontend-coverage.sh` / `check-frontend-diff-coverage.sh`）本地干跑清单属执行期任务——脚本本身是本 phase 产出物（82-02/82-03 的 Task 2 即干跑矩阵）。

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 基线文档落盘且数字可复算 | GOV-02 | 文档任务 | `node -e` 复算 coverage-final.json 比对文档记载 21574 stmts / 830 covered / 3.85% |
| 白名单登记三项齐全（理由/面积/复审条件）+ 面积 ≤5% | QUAL-02 | 文档审查 | 人工审查登记表完整性；自动核算 1028/22602=4.55%（已实测成立） |
| 4 层 gate 上线终检 | 全 phase | 视觉/流程核验 | 82-05 Task 2 checkpoint：PR 双绿 + main job 列表 + 本地三件套复跑 + 基线文档审阅 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags（package.json test:coverage 已改 `vitest run --coverage`，82-01-T1）
- [x] Feedback latency < 300s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planner（2026-08-23，plans 82-01~82-05 落盘）

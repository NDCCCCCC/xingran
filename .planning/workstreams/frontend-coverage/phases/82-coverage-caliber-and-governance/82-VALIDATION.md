---
phase: 82
slug: coverage-caliber-and-governance
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| *(planner 落 PLAN.md 后按任务填写)* | | | GOV-01 | — | N/A | smoke（产物断言） | `npm run test:coverage && node -e "const d=require('./coverage/coverage-final.json');const k=Object.keys(d);console.log('files='+k.length,'cad='+k.filter(p=>p.includes('cad-')).length)"` → 期望 files=571, cad=0 | ❌ 执行期即时命令 | ⬜ pending |
| | | | GOV-03 | — | N/A | integration（gate 干跑） | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` → exit 0（3.85≥3.8）；临时改全局行 5.0 → exit 1 | ❌ W0（脚本本 phase 新建） | ⬜ pending |
| | | | GOV-04 | — | N/A | integration（gate 干跑） | `bash .github/scripts/check-frontend-diff-coverage.sh <json> HEAD~1 80`（构造已知覆盖/未覆盖 diff 合成基线验证两分支 exit code） | ❌ W0（脚本本 phase 新建） | ⬜ pending |
| | | | GOV-05 | — | N/A | integration（gate 干跑） | 同 GOV-03 脚本：临时把某目录 floor 改 90.0 → exit 4；白名单漂移注入 → exit 6 | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- 无 vitest 测试缺口——本 phase 零新增测试，现有基础设施（19 文件 159 测试）覆盖回归验证。
- gate 脚本（`check-frontend-coverage.sh` / `check-frontend-diff-coverage.sh`）本地干跑清单属执行期任务——脚本本身是本 phase 产出物。

*Existing infrastructure covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 基线文档落盘且数字可复算 | GOV-02 | 文档任务 | `node -e` 复算 coverage-final.json 比对文档记载 21574 stmts / 830 covered / 3.85% |
| 白名单登记三项齐全（理由/面积/复审条件）+ 面积 ≤5% | QUAL-02 | 文档审查 | 人工审查登记表完整性；自动核算 1028/22602=4.55%（已实测成立） |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 300s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

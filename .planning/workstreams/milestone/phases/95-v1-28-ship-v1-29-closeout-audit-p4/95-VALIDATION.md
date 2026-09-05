---
phase: 95
slug: v1-28-ship-v1-29-closeout-audit-p4
status: ready
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-06
---

# Phase 95 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test（后端）+ vitest 4.x（前端）+ bash gate 脚本（本相主体为文档/审计，验证 = gate 跑批 exit code + 核对清单 grep） |
| **Config file** | `go.mod` / `xingran-react-frontend/vitest.config.ts` / `.coverage-threshold` / `.coverage-fe-floors` |
| **Quick run command** | `cd xingran-react-frontend && npm run type-check`（D-03 修复后为真检查）+ `grep` 核对项 |
| **Full suite command** | go build + go test ./... + npm type-check/lint/test + 双 coverage gate（D-06 七项清单，~25-30min） |
| **Estimated runtime** | 快验 ~30s；全量 ~25-30min（go ~7-9min + npm coverage ~17min） |

---

## Sampling Rate

- **After every task commit:** Quick run（涉改核对项 grep + type-check）
- **After every plan wave:** 全量 gate（95-02 主体即 gate 跑批）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30min（全量 gate 为交付物本身）

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 95-01-T1 | 95-01 | 1 | CLOSEOUT-01 | — | — | grep 核对 | `grep -c "SHIPPED + 阶段性收口" .planning/MILESTONES.md` ≥1 + 措辞校准项逐条 | ✅ | ⬜ pending |
| 95-01-T2 | 95-01 | 1 | CLOSEOUT-02 | — | — | grep 核对 | `grep -c "SHIPPED + ARCHIVED" .planning/PROJECT.md` ≥1 + frontend-coverage 目录存在性 ls | ✅ | ⬜ pending |
| 95-02-T1 | 95-02 | 1 | D-03 | — | — | type-check gate | `cd xingran-react-frontend && npm run type-check`（修后 3701 文件 0 错误）+ `npm run build` 救活 | ✅ | ⬜ pending |
| 95-02-T2 | 95-02 | 1 | CLOSEOUT-03(D-06) | — | — | gate 跑批 | go build/test + npm 四件套 + 双 coverage gate，逐项 exit code 记录 | ✅ | ⬜ pending |
| 95-02-T3 | 95-02 | 1 | CLOSEOUT-03(D-07/08) | — | — | audit 报告存在性 + 结构 | `test -f .planning/milestones/v1.29-MILESTONE-AUDIT.md` + 六段结构 grep + 7 项行动确认表 | ❌ → W0（本任务建） | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `.planning/milestones/v1.29-MILESTONE-AUDIT.md` — D-07 audit 报告（由 95-02 T3 创建，模板 = v1.27 同款）

*其余验证基建全部既有（gate 脚本/阈值文件/测试套件）。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| （无 — 全部为可自动化 gate/核对；外观级变化产品确认已在 94 D-04 裁决接受） | — | — | — |

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30min
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** 2026-09-06

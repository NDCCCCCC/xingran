---
phase: 95
slug: v1-28-ship-v1-29-closeout-audit-p4
status: ready
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-06
revised: 2026-09-06
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
- **After every plan wave:** 全量 gate（95-02 T3 主体即 gate 跑批）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30min（全量 gate 为交付物本身）

---

## Per-Task Verification Map

> 2026-09-06 修订：与 plan 实际任务结构对齐——95-01 T1/T2、95-02 T1..T5（checker 修订采纳 T4/T5 拆分：T4 audit 生成、T5 标记流转 + 记账闭环）。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 95-01-T1 | 95-01 | 1 | CLOSEOUT-01 | — | — | grep 核对 | `grep -c "45\.13" .planning/MILESTONES.md` ≥2 + `grep -c "24\.87"` ≥1 + `grep -c "核对确认" .planning/REQUIREMENTS.md .planning/ROADMAP.md .planning/workstreams/milestone/ROADMAP.md` | ✅ | ⬜ pending |
| 95-01-T2 | 95-01 | 1 | CLOSEOUT-02 | — | — | grep + ls 核对 | `sed -n '35p' .planning/PROJECT.md \| grep -c "SHIPPED + ARCHIVED"` + frontend-coverage 目录 ls + `grep -c "45 requirements"` 三文件 | ✅ | ⬜ pending |
| 95-02-T1 | 95-02 | 2 | CLOSEOUT-03③（D-03） | T-95-03 | type-check 从恒真 gate 变真 gate | type-check gate | `cd xingran-react-frontend && npm run type-check && npx tsc -b`（修后 3701 文件 0 错误 + build 链救活） | ✅ | ⬜ pending |
| 95-02-T2 | 95-02 | 2 | CLOSEOUT-03② 前置 | — | — | go test flaky 复验 | `go test -count=10 -run "TestBackupHandler_Restore" ./internal/api/v1/network/` 全绿 + `go test -count=1 -run "TestJbu8003" ./internal/api/v1/` 绿 | ✅ | ⬜ pending |
| 95-02-T3 | 95-02 | 2 | CLOSEOUT-03①②④⑤⑥⑦（D-06） | — | — | gate 跑批 | go build/test + npm 四件套 + 双 coverage gate，逐项 exit code 记录（② 含 `./...` 补充留证） | ✅ | ⬜ pending |
| 95-02-T4 | 95-02 | 2 | CLOSEOUT-03⑧（D-07/D-08） | T-95-06 | audit 证据实名口径（89-01..03 / 90-01..04-SUMMARY.md，无幻影引用） | audit 报告存在性 + 结构 | `test -f .planning/milestones/v1.29-MILESTONE-AUDIT.md` + `grep -c "^## "` ≥8 + `grep -c "45/45"` + `grep -c "89-01-SUMMARY"` + `grep -c "90-04-SUMMARY"`（实名口径） | ❌ → W0（本任务建） | ⬜ pending |
| 95-02-T5 | 95-02 | 2 | CLOSEOUT-03⑨ + 记账闭环（D-04/D-05/D-09） | — | — | grep 核对 | `grep -c "JOBSTAT-01" .planning/REQUIREMENTS.md` + `grep -c "^\- \[x\] \*\*CLOSEOUT-03\*\*" .planning/REQUIREMENTS.md` + `head -3 94-HUMAN-UAT.md \| grep -c resolved` + `grep -c "✅ SHIPPED 2026-09-06" .planning/MILESTONES.md` + 两 ROADMAP `grep -c "45/45"` | ❌ → W0（UAT/记账落点由本任务改写） | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `.planning/milestones/v1.29-MILESTONE-AUDIT.md` — D-07 audit 报告（由 95-02 T4 创建，模板 = v1.27 同款；89/90 证据用各 plan SUMMARY 实名口径）
- [ ] 95-02 T5 记账落点改写（REQUIREMENTS CLOSEOUT-03 复选框 + 两 ROADMAP Progress 表 Phase 95 行）— 对象文件既有，T5 落账（前置校准由 95-01 落地）

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

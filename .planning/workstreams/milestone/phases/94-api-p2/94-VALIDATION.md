---
phase: 94
slug: api-p2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-05
---

# Phase 94 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.x（xingran-react-frontend） |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run src/lib/apiFactory.test.ts src/lib/download.test.ts` |
| **Full suite command** | `cd xingran-react-frontend && npm run test` |
| **Estimated runtime** | ~30-60 seconds（lib 域单文件 ~5s，全量 ~60s） |

---

## Sampling Rate

- **After every task commit:** Run quick run command（涉改文件对应 .test.ts 一并跑）
- **After every plan wave:** Run full suite command + `npm run type-check` + `npm run lint`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| （planner 生成 plan 后填充：每任务一行，映射 API-FACTORY-01..05 与 D-11/D-12 契约/扫描测试） | | | | | | | | | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `src/lib/apiFactory.test.ts` — D-11 工厂契约测试（路径拼接七端点、CreatePayload 排除、statistics/searchOptions 解包）
- [ ] `src/lib/download.test.ts` — D-11 下载基建测试（GET+POST 双变体 blob 链）
- [ ] 扫描测试文件（D-12，文件名 planner 定）— 手写 CRUD 模板检测（TS compiler API AST，硬档+warning 档）

*测试基建已存在（vitest 4 + 13 个配套 .test.ts），Wave 0 仅新增上述三件。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| （无 — 全部行为可自动化验证；消费方零改动由 type-check + 现有测试保证） | API-FACTORY-05 | — | — |

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

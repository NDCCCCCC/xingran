---
phase: 84
slug: p1-70
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-27
---

# Phase 84 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 来源:`84-RESEARCH.md` ## Validation Architecture + Wave 0 Gaps + Success Criteria Observable Signals。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.10 + @vitest/coverage-v8 4.1.10 + jsdom 27.4.0 + @testing-library/react 16.3.2 |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run src/components/<subdir>/__tests__/<file>.test.tsx` |
| **Full suite command** | `cd xingran-react-frontend && npm run test:coverage` |
| **Gate script command** | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` |
| **Diff gate command** | `bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json <base-ref> 80` |
| **Estimated runtime** | 全量 ~60-90 秒(83 终点 + 84 增量测试) |

---

## Sampling Rate

- **After every task commit:** `cd xingran-react-frontend && npx vitest run <修改测试文件>` 单文件验证
- **After every plan wave:** `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh ...` + `bash .github/scripts/check-frontend-diff-coverage.sh ... HEAD 80`
- **Before `/gsd:verify-work`:** 全量 vitest 退出 0 + gate 全 PASS + diff gate ≥80%
- **Max feedback latency:** ~90 秒(全量 vitest)+ ~5 秒(单文件)+ ~3 秒(gate)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 84-0-01 | 0 | 0 | harness 沉淀 | — | N/A(测试基建) | unit | `npx vitest run src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` | ❌ W0 | ⬜ pending |
| 84-0-02 | 0 | 0 | setup.ts polyfill | — | ResizeObserver polyfill 集中 | unit | 同上 | ❌ W0 | ⬜ pending |
| 84-0-03 | 0 | 0 | gate 扩展 + floors 新行 | — | L219/L316/L381 awk 镜像 | unit | `bash .github/scripts/check-frontend-coverage.sh --init ...` | ❌ W0 | ⬜ pending |
| 84-1a-01 | 1a | 1 | COMP-01 | — | components/shared ≥70% | coverage | `bash check-frontend-coverage.sh ... \| grep '^components/shared '` | ❌ W1 | ⬜ pending |
| 84-1b-01 | 1b | 1 | COMP-02 | — | components/dashboard ≥70% | coverage | `grep '^components/dashboard '` | ❌ W1 | ⬜ pending |
| 84-2a-01 | 2a | 2 | COMP-03 | — | components/layout ≥70% | coverage | `grep '^components/layout '` | ❌ W2 | ⬜ pending |
| 84-2b-01 | 2b | 2 | COMP-04 | — | CronSelector+captcha+operations 各 ≥70% | coverage | 三个 subdir grep | ❌ W2 | ⬜ pending |
| 84-3a-01 | 3a | 3 | COMP-05a/05b/05c | — | network+reconciliation+零散 各 ≥70% | coverage | 三个 subdir grep | ❌ W3 | ⬜ pending |
| 84-3b-01 | 3b | 3 | COMP-05d | — | design-system ≥70% | coverage | `grep '^design-system '` | ❌ W3 | ⬜ pending |
| 84-XX-NN | each | each | QUAL-01 | — | 159 存量测试不回归 | unit | `npm run test:coverage` + `Tests N >= 159` | ✅ 现有 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `src/test/utils/renderWithProviders.tsx` — Router + antd App + 按需 stores reset(默认 MemoryRouter + App + zustand 自动 reset)
- [ ] `src/test/utils/createApiMock.ts` — 端点工厂 `vi.fn()` + 批量注册 `mockApiBatch`
- [ ] `src/test/setup.ts` 增补 `ResizeObserver` polyfill(对齐 BulkWriteDrawer L27-36 inline 形态);`IntersectionObserver` / `PointerEvent` / canvas getContext 按需(执行阶段实证失败再加)
- [ ] `.coverage-fe-floors` 新增 9 个 components subdir 行 + 1 个 design-system 行 + 1 个 components 聚合行(初值 = 0,D-14 后逐 plan bump)
- [ ] `.github/scripts/check-frontend-coverage.sh` L219/L316/L381 三处扩展 `components/<subdir>` 二级聚合分支(镜像 `pages/<subdir>`)
- [ ] `src/components/network/port-write/__tests__/BulkWriteDrawer.test.tsx` plan 0 末尾验证 5 用例仍 PASS(不强制改 Wrapper,仅 setup.ts 沉淀后允许)

*若全部完成:`Existing infrastructure covers all phase requirements.` 行替换 wave_0_complete = true。*

---

## Per-Plan Verification Map (SC-driven)

| SC | Plan | Requirement | Automated Command | Pass Criteria |
|----|------|-------------|-------------------|---------------|
| SC-1 | 1a | components/shared ≥70% | `grep '^PASS: components/shared '` | 输出 ≥1 行 PASS |
| SC-2 | 1b | components/dashboard ≥70% | `grep '^PASS: components/dashboard '` | 输出 ≥1 行 PASS |
| SC-3 | 2a + 2b | layout + CronSelector + captcha + operations 各 ≥70% | 四个 subdir grep | 四行均 PASS |
| SC-4 | 3a + 3b | network + reconciliation + 零散 + design-system 各 ≥70% | 四个 subdir + 顶层 grep | 五行均 PASS |
| SC-5 | 3b 末尾 | components 聚合行 ≥69.5% | `grep '^PASS: components '` | 输出 PASS |
| SC-6 | 0+各 wave | harness 沉淀 | `grep -r 'renderWithProviders\|createApiMock' src/components/ \| wc -l >= 3` | wave 1/2/3 各有 import |
| SC-7 | 0 | setup.ts polyfill 集中 | `grep 'ResizeObserver' src/test/setup.ts` + `grep 'ResizeObserverStub' BulkWriteDrawer.test.tsx` 不存在 | 两条件同时成立 |
| SC-8 | each | QUAL-01 159 存量不回归 | `npm run test:coverage` + `Tests N >= 159` | 全量 vitest 退出 0 |
| SC-9 | each | CI gate 绿 + ratchet 单调 | `git diff .coverage-fe-floors frontend-coverage-baseline.md` 含本 plan 新行 | 同 PR 双变更 |

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实 CI 端到端验证 | QUAL-01 | gh CLI 在 push 后才跑 | push 后 `gh run watch <run-id> --exit-status` 盯 ci.yml |

*若真实 CI 通过:在 84-XX-SUMMARY.md 引用 run-id 与 log URL。*

---

## Nyquist 维度映射

| Nyquist 维度 | 84 验证映射 | 真信号 | 假信号(需警惕) |
|--------------|------------|--------|----------------|
| **Correctness(行为正确)** | 组件测试中 `user event` + `props 渲染断言` 双向覆盖(D-11) | fireEvent.click 后状态变化断言 + DOM 文本/role 查询命中 | 只断言 `expect(container).toBeTruthy()` 这种无意义 wrapper |
| **Coverage(覆盖率)统计** | `npm run test:coverage` + `bash .github/scripts/check-frontend-coverage.sh` 二者均绿 | gate 输出 PASS 行数 = 28 既有 + 9 subdir 行 + components 聚合行 = 38 行 PASS | gate 绿但 profile 中含未测试文件 0% 计入;全局 weighted avg 提升但各 subdir 仍 0% |
| **Robustness(回归守护)** | QUAL-01 159 存量测试 + BulkWriteDrawer 5 用例 + HealthCard 5 用例不回归 | 全量 vitest 退出 0;`Tests N passed (N >= 159)` | 单测过、一起跑部分失败(状态泄漏 / 未 reset store) |
| **Reuse(模式沉淀)** | harness `renderWithProviders` + `createApiMock` 在 wave 1/2/3 各有 ≥1 个 import | `grep -r 'renderWithProviders' src/components/ \| wc -l >= 3` | harness 存在但无任何测试 import(死代码) |

---

## Validation Sign-Off

- [ ] Wave 0 全部完成(setup.ts + renderWithProviders + createApiMock + gate 扩展 + floors 新行)
- [ ] Sampling continuity:每个 plan 都有 automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
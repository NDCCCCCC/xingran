---
phase: 83
slug: p0-70
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-24
---

# Phase 83 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4.1.10 + @vitest/coverage-v8 4.1.10 + jsdom + @testing-library/react 16.3.2 |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run <path/to/test.ts>` |
| **Full suite command** | `cd xingran-react-frontend && npm run test:coverage` |
| **Gate script command** | `bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors` |
| **Diff gate command** | `bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json <base-ref> 80` |
| **Estimated runtime** | ~60–90 秒 |

---

## Sampling Rate

- **After every task commit:** Run `cd xingran-react-frontend && npx vitest run <新增/修改的测试文件>` 确保单文件通过；如涉及覆盖率，加 `--coverage` 局部验证。
- **After every plan wave:** Run `cd xingran-react-frontend && npm run test:coverage` + gate script + diff gate；按 D-11 bump 对应目录 floor 并追加基线文档。
- **Before `/gsd:verify-work`:** Full suite must be green，且 gate 输出中 P0 目录全部 ≥70%。
- **Max feedback latency:** 90 秒（完整 suite + gate）

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 83-01-01 | 01 | 1 | INFRA-04 / QUAL-03 | — | 验证 CR-01/WR-01~03 修复在 main 已生效；发起试验 PR 触发 CI 双绿 | script + CI | `bash .github/scripts/check-frontend-diff-coverage.sh ...` | ✅ main 已存在 | ⬜ pending |
| 83-02-xx | 02 | 2 | INFRA-02 | V6 / V8 | utils（国密/token/cache）statements ≥70% | unit + coverage | `npx vitest run src/utils/...` / gate | ❌ W0 后创建 | ⬜ pending |
| 83-03-xx | 03 | 2 | INFRA-01 | V2 / V3 / V5 | lib（api.ts 双轨）statements ≥70% | unit + coverage | `npx vitest run src/lib/...` / gate | ❌ W0 后创建 | ⬜ pending |
| 83-04-xx | 04 | 3 | INFRA-03 | V3 / V5 | hooks（usePagination/useServerSort/usePersistedState 等）≥70% | unit + coverage | `npx vitest run src/hooks/...` / gate | ❌ W0 后创建 | ⬜ pending |
| 83-04-xx | 04 | 3 | INFRA-04 | V3 | store（auth/menu/tabs/dashboard 等）≥70% | unit + coverage | `npx vitest run src/store/...` / gate | ❌ W0 后创建 | ⬜ pending |
| 83-05-xx | 05 | 4 | INFRA-05 | — | services/router/constants/types ≥70% + harness 定稿 | unit + coverage | `npx vitest run src/{services,router,constants,types}/...` / gate | ❌ W0 后创建 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `xingran-react-frontend/src/test/utils/renderWithProviders.tsx` — Router + AntD ConfigProvider 包装器（coverage.exclude 已排除 `src/test/`）
- [ ] `xingran-react-frontend/src/test/utils/createApiMock.ts` — 端点工厂形态的 `@/lib/api` mock
- [ ] `xingran-react-frontend/src/test/utils/mockAntdMessage.ts` — antd message / Modal 统一 mock
- [ ] `xingran-react-frontend/src/test/utils/*.test.ts` — harness 自身使用示例（不计入 coverage）
- [ ] `xingran-react-frontend/src/lib/api.test.ts` — api.ts 双轨直测（INFRA-01 关键）
- [ ] `xingran-react-frontend/src/utils/sm4.test.ts`、`src/utils/encoding.test.ts` — 国密向量直测
- [ ] `xingran-react-frontend/src/utils/token/TokenManager.test.ts`、`src/utils/token/SecureTokenStorageImpl.test.ts` — TokenManager fake timers
- [ ] `xingran-react-frontend/src/store/*.test.ts` — 各 Zustand store 按需注入 + reset 测试
- [ ] `xingran-react-frontend/src/router/*.test.ts` — routeConfigManager / routeGenerator 等低成本覆盖

*Wave 0 在 Plan 01（CR-01 验证/清理/试验 PR）与 Plan 02（utils）之间完成，作为 harness 可用前提。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 试验 PR CI 双绿 | CR-01 修复验收 | 真实 GitHub Actions 环境才能验证 diff gate 行为 | 创建含 `src/test/` + `.d.ts` + 白名单文件变更的试验 PR；确认 `ci.yml` frontend job 与 `frontend-coverage-diff` job 均 success；关闭该 PR 不 merge。 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

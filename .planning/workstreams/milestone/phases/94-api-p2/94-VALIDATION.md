---
phase: 94
slug: api-p2
status: ready
nyquist_compliant: true
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
| 94-01-T1 | 94-01 | 1 | API-FACTORY-01/02 | — | — | type-check gate | `cd xingran-react-frontend && npm run type-check && npm run lint` | ✅（npm scripts） | ⬜ pending |
| 94-01-T2 | 94-01 | 1 | D-04（API-FACTORY-02 配套） | T-94-01/T-94-02/T-94-03 | blobAxios 异步 getAccessToken 注入 Bearer（禁同步拼头）+ 沿用既有 content-disposition 解析 + env baseURL | type-check gate | `cd xingran-react-frontend && npm run type-check && npm run lint` | ✅（npm scripts） | ⬜ pending |
| 94-01-T3 | 94-01 | 1 | API-FACTORY-01/02 + D-11 | T-94-01（Test 1 锁 Bearer 注入）/ T-94-02（文件名解析契约锁定） | 拦截器注入断言 + timeout 300000 锁值 + URL 编码文件名提取 | unit（契约） | `cd xingran-react-frontend && npx vitest run src/lib/apiFactory.test.ts src/lib/download.test.ts && npm run type-check && npm run lint && npx vitest run` | ❌ → Wave 0（apiFactory.test.ts / download.test.ts 本任务建） | ⬜ pending |
| 94-02-T1 | 94-02 | 2 | API-FACTORY-03 | T-94-06 | —（等价替换：既有 URL/动词/签名零变化，既有测试锁定） | unit（既有回归）+ type-check | `cd xingran-react-frontend && npx vitest run src/lib/opsApi.test.ts && npm run type-check && npm run lint && npx vitest run` | ✅（opsApi.test.ts 既有，仅小幅适配） | ⬜ pending |
| 94-02-T2 | 94-02 | 2 | API-FACTORY-03/04 + D-08 | T-94-04/T-94-05 | downloadReport 归一 blobAxios 链（Bearer 注入 + 300000ms 超时防护，消除无超时裸 fetch） | unit（既有回归 + 新增用例） | `cd xingran-react-frontend && npx vitest run src/lib/rpaApi.test.ts src/lib/__tests__/rpaApi.batch56.unit.test.ts src/lib/download.test.ts && npm run type-check && npm run lint && npx vitest run` | ✅（rpa 两测试既有；download.test.ts 94-01 建、本任务追加组 6 用例） | ⬜ pending |
| 94-02-T3 | 94-02 | 2 | API-FACTORY-04 | T-94-06 | —（vmApi.list/create 原样覆盖保住 interface 直传消费点类型契约） | unit（既有回归）+ type-check | `cd xingran-react-frontend && npx vitest run src/lib/vdiApi.test.ts && npm run type-check && npm run lint && npx vitest run` | ✅（vdiApi.test.ts 既有） | ⬜ pending |
| 94-03-T1 | 94-03 | 3 | API-FACTORY-04 + D-05/D-14 | T-94-08 | —（同 URL 同动词等价委托，KEEP 区零触碰） | unit（既有回归）+ type-check | `cd xingran-react-frontend && npx vitest run src/lib/workorderApi.test.ts src/lib/knowledgeApi.test.ts src/lib/dutyApi.test.ts && npm run type-check && npm run lint` | ✅（三个 .test.ts 既有且零改动） | ⬜ pending |
| 94-03-T2 | 94-03 | 3 | API-FACTORY-04 + D-05/D-08 | T-94-07 | —（:501 潜伏 bug 修复为唯一行为变更，独立 atomic commit 登记可审计） | unit（既有回归）+ type-check | `cd xingran-react-frontend && npx vitest run src/lib/noticeApi.test.ts src/lib/adDomainApi.test.ts && npm run type-check && npm run lint` | ✅（adDomainApi.test.ts 或含 :501 断言最小修正） | ⬜ pending |
| 94-03-T3 | 94-03 | 3 | API-FACTORY-05 + D-12/D-13 | T-94-SC | —（零新依赖，无供应链新面） | unit（invariants 扫描）+ coverage gate + 三件套 | `cd xingran-react-frontend && npx vitest run src/lib/apiFactory.invariants.test.ts && npm run test:coverage && cd .. && bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors && cd xingran-react-frontend && npm run type-check && npm run lint && npx vitest run` | ❌ → Wave 0（apiFactory.invariants.test.ts 本任务建）；gate 脚本 ✅ 既有 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> 路径基准注记（94-03-T3）：coverage gate 与 CI 相同以仓库根为工作目录调用（ci.yml:183 `working-directory: .`），脚本/json/floors 三路径按仓库根解析——命令链内已用 `cd ..` 切回仓库根、gate 后再 `cd xingran-react-frontend` 回到前端目录跑三件套。

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

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** 2026-09-05

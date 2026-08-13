---
phase: 58
slug: route-contract-alignment
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 58 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 本 phase 是纯前端契约对齐（路由方法 + 字段命名），后端零改动。验证以 `npm run type-check`（天然契约门）+ curl SC 断言 + 既有 Go 单测回归为主。来源：`58-RESEARCH.md` §Validation Architecture。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework (前端)** | Vitest（jsdom + setupFiles `./src/test/setup.ts` + `@` alias） |
| **Framework (后端)** | Go testing + testify v1.11.1（既有，本 phase 不改后端） |
| **Config file** | `xingran-react-frontend/vitest.config.ts` / Go 标准 `_test.go` 同包 |
| **Quick run command** | `cd xingran-react-frontend && npm run lint && npm run type-check` |
| **Full suite command (前端)** | `cd xingran-react-frontend && npm run test` |
| **Backend regression** | `go test ./internal/services/system/ -run "TestGetAPIKey|TestDeleteAPIKey|TestUpdateAPIKey" -v` |
| **Estimated runtime** | type-check <30s；后端单测 <5s |

---

## Sampling Rate

- **After every task commit:** Run `cd xingran-react-frontend && npm run lint && npm run type-check`（类型检查是契约对齐最关键的回归门——`APIKey` 接口字段改 camel 后，漏改的 `record.ip_whitelist` 会立刻触发 TS 编译错误）
- **After every plan wave:** Run `npm run test`（前端）+ `go test ./internal/services/system/`（后端回归）+ 手动 curl SC#1-SC#3
- **Before `/gsd:verify-work`:** type-check + lint 绿 + 后端回归绿 + SC#1-SC#4 手动验证全部成立
- **Max feedback latency:** ~30s（type-check）

---

## Per-Task Verification Map

> Task ID 由 PLAN.md 分配后填入下表。每个改字段名/路由方法的 task 完成后必须跑 `npm run type-check`。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-01 SC#1 | — | N/A | 集成 (curl) | `curl -X POST .../system/apikeys/<id>` → 断言 code:0 非 404 | ❌ W0 | ⬜ pending |
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-01 SC#2 | — | N/A | 集成 (curl+DB) | curl 更新 + `SELECT updated_at` 刷新 | ❌ W0 | ⬜ pending |
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-01 SC#3 | — | N/A | 集成 (curl) | 二次 delete 断言 code:1001 + "密钥不存在" 非 404 | ❌ W0 | ⬜ pending |
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-02 SC#1 | T-58-01 | inheritPerms/ipWhitelist 真正写入 DB | 单测/手动 | vitest 断言 record→form 映射 / 编辑 Modal 回填 | ❌ W0 (可选) | ⬜ pending |
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-02 SC#2 | T-58-01 | 复合字段非零值持久化 | 集成 (curl+DB) | curl create 带 ipWhitelist + `SELECT ip_whitelist` 非空 | ❌ W0 | ⬜ pending |
| *(待 PLAN 分配)* | 01 | 1 | CONTRACT-02 排序 | — | N/A | 手动 (UI) | 点列头排序，网络请求 orderByColumn=isActive | ❌ W0 | ⬜ pending |
| *(回归)* | — | — | 后端行为不变 | — | N/A | 单元 (Go) | `go test ./internal/services/system/ -run "TestGetAPIKey|TestDeleteAPIKey|TestUpdateAPIKey"` | ✅ 现有 apikey_service_test.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] 手动 curl 验证脚本（SC#1-SC#3 的 HTTP 断言）—— 需可运行 backend + 已知 API Key ID + JWT token
- [ ] 可选：`xingran-react-frontend/src/api/apikey.test.ts` —— vitest mock `post` 断言三函数路径含 `/update`、`/delete` 且用 POST（planner 按 TESTING.md 决定）
- [x] `npm run type-check` 编译期契约门 —— 现有基础设施已覆盖（无需 Wave 0 安装）

*若 planner 决定不补 vitest 单测：Existing infrastructure（type-check + curl + Go 单测）covers all phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 编辑 Modal 复合字段完整回填（ipWhitelist/inheritPerms/expiresAt） | CONTRACT-02 SC#1 | antd Form 渲染 + setFieldsValue 交互 | 1. 打开 API Key 管理页 2. 点某条记录「编辑」3. 确认 IP 白名单/继承权限/过期时间字段值正确回填（非空/非 undefined） |
| 列排序生效（状态/过期/最后使用） | CONTRACT-02 排序 | antd Table 排序交互 + 网络请求观察 | 1. 点列头「状态/过期时间/最后使用」排序 2. DevTools 网络面板确认 `orderByColumn` 为 camelCase（isActive/expiresAt/lastUsedAt）3. 列表顺序随之变化 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

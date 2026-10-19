---
phase: "59"
slug: "ad-login-guard-reconciliation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-19"
---

# Phase 59 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.24.5, stdlib testing + sqlite in-memory) |
| **Config file** | none — existing `_test.go` files in package dirs |
| **Quick run command** | `go test ./internal/services/addomain/ ./internal/services/system/ -count=1` |
| **Full suite command** | `go build ./... && go vet ./... && go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/addomain/ ./internal/services/system/ -count=1`
- **After every plan wave:** Run `go build ./... && go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 59-01-01 | 01 | 1 | GUARD-01 | T-59-01 | 守卫只读队列，不改写谓词 | unit | `go test ./internal/services/system/ -run TestPhase59_SyncADUser_ActivePendingQueueSkipsDept -count=1` | ❌ W0 | ⬜ pending |
| 59-01-02 | 01 | 1 | GUARD-01 | T-59-01 | 查询失败保守跳过，不上抛 | unit | `go test ./internal/services/system/ -run 'TestPhase59_SyncADUser_' -count=1` | ❌ W0 | ⬜ pending |
| 59-02-01 | 02 | 1 | GUARD-02 | T-59-04 | 视图 DDL 静态、无输入插值 | migration | `go test ./internal/core/db/migrations/ -run TestPhase59_Migration217 -count=1` | ❌ W0 | ⬜ pending |
| 59-02-02 | 02 | 1 | GUARD-02 | T-59-03 | 四类口径 + 双方言 CAST/布尔/后缀 | migration | `go test ./internal/core/db/migrations/ -run 'TestPhase59_Migration217' -count=1` | ❌ W0 | ⬜ pending |
| 59-03-01 | 03 | 2 | GUARD-02 | T-59-06 | 参数绑定 + LIKE 转义 | unit | `go test ./internal/services/addomain/ -run 'TestPhase59_DeptOUMismatch\|TestPhase59_MatchConsistency' -count=1` | ❌ W0 | ⬜ pending |
| 59-03-02 | 03 | 2 | GUARD-02 | T-59-05 | RequirePermissions 独立鉴权 | unit（httptest） | `go test ./internal/api/v1/system/ -run 'TestPhase59_ReconHandler' -count=1` | ❌ W0 | ⬜ pending |
| 59-03-03 | 03 | 2 | GUARD-02 | T-59-05 | F 型权限载体 + 精准授权 | migration | `go test ./internal/core/db/migrations/ -run TestPhase59_Migration218 -count=1` | ❌ W0 | ⬜ pending |
| 59-04-01 | 04 | 3 | GUARD-02 | T-59-09 | 无权限零请求 + 后端鉴权 | component | `cd ruoyi-react-frontend && npx vitest run src/lib/adDomainApi.test.ts src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ❌ W0 | ⬜ pending |
| 59-04-02 | 04 | 3 | GUARD-02 | T-59-10 | 不展示 UUID、无写入口 | component | `cd ruoyi-react-frontend && npx vitest run src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ❌ W0 | ⬜ pending |
| 59-04-03 | 04 | 3 | GUARD-02 | — | 无 hex / 无原生 axios | component + source | `cd ruoyi-react-frontend && npx vitest run src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Planner refines this map to actual task IDs after planning.*

---

## Wave 0 Requirements

- [ ] `internal/services/system/user_sync_service_phase59_test.go` — GUARD-01 守卫用例（pending/failed 拦截 + WARN / pushed/dead/disable 放行 / 无行照常更新 / 查询失败保守跳过）
- [ ] `internal/core/db/migrations/migration_217_dept_ou_mismatch_view_test.go` — 视图建表、幂等、列顺序、四类口径
- [ ] `internal/services/addomain/dept_ou_mismatch_service_test.go` — 分页/筛选 + Go/SQL 双份实现一致性
- [ ] `internal/api/v1/system/ad_reconciliation_handler_test.go` — 绑定错误 / 白名单 / 默认分页
- [ ] `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go` — 权限载体幂等与精准授权
- [ ] `ruoyi-react-frontend/src/pages/ad-domain/configs/__tests__/ReconciliationTab.test.tsx` — 权限门/空态/错误态/部分行

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实 AD 登录回写 dept 被 WARN 拦截 | GUARD-01 | 需真实 AD 环境与 pending 队列行 | 编辑用户 dept → 等队列入队 → 用该用户 AD 登录 → 查日志 WARN + sys_user.dept_id 未变 |
| 对账 Tab 权限授予后内容渲染 | GUARD-02 | 需浏览器 + 角色权限配置 | 管理员为角色授予 `ops:ad:config:reconciliation:list` → 登录该角色 → AD 配置页对账 Tab 出数据 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

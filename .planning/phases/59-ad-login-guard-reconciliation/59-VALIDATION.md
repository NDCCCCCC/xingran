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
| 59-01-01 | 01 | 1 | GUARD-01 | — | N/A | unit | `go test ./internal/services/system/ -run TestPhase59 -count=1` | ❌ W0 | ⬜ pending |
| 59-02-01 | 02 | 1 | GUARD-02 | — | 端点需登录鉴权 | unit | `go test ./internal/services/addomain/ -run TestPhase59 -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Planner refines this map to actual task IDs after planning.*

---

## Wave 0 Requirements

- [ ] `internal/services/system/user_sync_guard_test.go` — GUARD-01 守卫用例（有队列行跳过+WARN / 无队列行照常更新）
- [ ] `internal/services/addomain/reconciliation_*_test.go` — GUARD-02 视图口径 + Go/SQL 双份实现一致性用例

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

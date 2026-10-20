---
phase: "60"
slug: "soft-delete-recovery-safety"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-20"
---

# Phase 60 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — existing per-package `_test.go` conventions |
| **Quick run command** | `go build ./... && go test ./internal/services/system/ ./internal/core/security/ -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/services/system/ ./internal/core/security/ -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 60-01-01 | 01 | 1 | SAFE-01 / SAFE-02 | T-60-* | DN 不一致拒绝恢复 + 审计 WARN + 专用原因码 | unit | Wave 0 后由 planner 落 `<automated>` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/system/user_sync_service_phase60_test.go` — SAFE-01/SAFE-02 测试桩（复用 `setupTestDBForSync` / `captureLogs`）
- [ ] `internal/core/security/` sentinel 原因码测试

*Planner 须在 PLAN.md 中按实际任务拆分补全本表。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 登录路径软删恢复拒绝的用户可见文案 | SAFE-01 | 需真实登录流 + 国密加密请求 | 手动触发 DN 漂移场景登录，观察响应文案与日志 WARN |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

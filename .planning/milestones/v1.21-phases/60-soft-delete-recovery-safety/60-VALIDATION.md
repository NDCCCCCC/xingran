---
phase: "60"
slug: "soft-delete-recovery-safety"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
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
| 60-01-01 | 01 | 1 | SAFE-01 / SAFE-02 | T-60-01, T-60-02 | 恢复前 `NormalizeDN` 身份校验；拒绝分支只读、零写入、不签发 JWT | unit + build | `go build ./... && go test ./internal/core/security/ ./internal/api/v1/ ./internal/services/system/ -count=1` | ✅ | ✅ green |
| 60-01-02 | 01 | 1 | SAFE-01 / SAFE-02 | T-60-03, T-60-04 | 用户文案不含 DN；策略拒绝 WARN 与基础设施故障 ERROR 分级 | unit | `go test ./internal/services/system/ -run 'TestPhase60_CheckSoftDeletedRestore_Predicate|TestPhase60_Login_' -count=1 && go test ./internal/core/security/ -run TestPhase60_SyncErrorReasonFor -count=1 && go test ./internal/api/v1/ -run TestAuth8003_SyncErrorReasonMessage -count=1` | ✅ | ✅ green |
| 60-02-01 | 02 | 2 | SAFE-01 / SAFE-02 | T-60-05, T-60-06, T-60-07 | 批量拒项计入 `Skipped`、短路绕开降级管道、响应错误文本不含 DN | unit | `go test ./internal/services/system/ -run 'TestPhase60_Batch_|TestPhase60_PathConsistency_' -count=1` | ✅ | ✅ green |
| 60-02-02 | 02 | 2 | SAFE-01 / SAFE-02 | T-60-05, T-60-06, T-60-07 | 全量回归（build + 三个受影响包）+ 验证映射/需求/路线图/状态回填 | build + integration | `go build ./... && go test ./internal/services/system/ ./internal/core/security/ ./internal/api/v1/ -count=1` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/services/system/user_sync_service_phase60_test.go` — SAFE-01/SAFE-02 测试桩（复用 `setupTestDBForSync` / `captureLogs`；60-01 建立判定/登录用例，60-02 追加批量与双路径一致性用例）
- [x] `internal/core/security/ad_authenticator_phase60_test.go` — sentinel 原因码测试（`TestPhase60_SyncErrorReasonFor`）

*Planner 须在 PLAN.md 中按实际任务拆分补全本表。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 登录路径软删恢复拒绝的用户可见文案 | SAFE-01 | 需真实登录流 + 国密加密请求 | 手动触发 DN 漂移场景登录，观察响应文案与日志 WARN |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-10-20

## Validation Audit 2026-10-20

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

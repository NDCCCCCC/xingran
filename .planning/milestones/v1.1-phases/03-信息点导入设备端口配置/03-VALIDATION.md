---
phase: 03
slug: 信息点导入设备端口配置
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-16
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go build ./...` |
| **Full suite command** | `go test ./internal/services/operations/... -v` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./...`
- **After every plan wave:** Run `go test ./internal/services/operations/... -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | IMPORT-05, IMPORT-06, IMPORT-07 | — | N/A | unit | `go test ./internal/services/operations/... -v -run TestInfoPoint` | ✅ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | VAL-03, VAL-04 | — | N/A | unit | `go test ./internal/services/operations/... -v -run TestInfoPointExport` | ✅ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Import Excel with device/port names, verify ops_info_points has correct device_id/port_id UUIDs | IMPORT-05, IMPORT-06 | Requires running server + database | Import test Excel, query DB |
| Import Excel without device/port columns, verify no error and fields are null | IMPORT-07 | Requires running server + database | Import minimal Excel, verify success |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

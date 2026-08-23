---
phase: 76
slug: test-doubles
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-23
---

# Phase 76 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Wave 0 installs test-only deps (miniredis/v2, httpmock) |
| **Quick run command** | `go test ./pkg/cache/ ./internal/device/` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/cache/ ./internal/device/`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 76-01-01 | 01 | 1 | INFRA-01 | — | N/A | unit | `go test ./pkg/cache/` | ❌ W0 | ⬜ pending |
| 76-02-01 | 02 | 1 | INFRA-02 | — | N/A | unit | `go test ./internal/device/` | ❌ W0 | ⬜ pending |
| 76-03-01 | 03 | 1 | INFRA-03 | — | N/A | unit | `go test ./internal/services/addomain/` | ❌ W0 | ⬜ pending |
| 76-04-01 | 04 | 1 | INFRA-04 | — | N/A | unit | `go test ./internal/services/addomain/` | ❌ W0 | ⬜ pending |
| 76-05-01 | 05 | 1 | INFRA-05 | — | N/A | unit | `go test ./internal/models/` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go get github.com/alicebob/miniredis/v2@v2.38+` — test-only dep (INFRA-01)
- [ ] `go get github.com/jarcoal/httpmock@v1.4.x` — test-only dep (INFRA-01)
- [ ] geocoding httpmock PoC test — anchors httpmock in go.mod so `go mod tidy` keeps it

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ubuntu CI double-green, no Docker | INFRA-01 | Local Windows cannot verify CI runner | Push branch, confirm `.github/workflows/ci.yml` go test job passes on ubuntu without Docker |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

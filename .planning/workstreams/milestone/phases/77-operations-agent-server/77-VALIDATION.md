---
phase: 77
slug: operations-agent-server
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-24
---

# Phase 77 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib + testify v1.11.1) |
| **Config file** | none — existing infrastructure (Phase 76 已交付全部 test doubles/注入缝) |
| **Quick run command** | `go test ./internal/services/operations/ ./internal/agent/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/operations/ ./internal/agent/...`
- **After every plan wave:** Run `go test ./...` + `.github/scripts/check-coverage.sh`（gate 不倒退）
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (待 planner 产出 77-01..05 后填充) | | | | | | | | | |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.（sqlite/httptest/excelize/re-exec stub 全部就绪 — 见 76-VERIFICATION.md INFRA-01..05）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Windows 本地 vs ubuntu CI 覆盖率差 <2pp（SC#3，D-04/D-05） | BLOCK-02 | CI 数字需 push 后观测，本地无法程序化获取 | 收口时本地 `go test -cover ./internal/services/operations/ ./internal/agent/...` 与 CI backend-coverage artifact 的 per-package 数字人工对比，差值记录进 77-VERIFICATION.md |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending

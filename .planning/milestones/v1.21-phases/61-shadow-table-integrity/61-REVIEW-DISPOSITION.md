---
phase: 61
review: 61-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: open
    title: "updateSyncLog writes nonexistent column `error_message` — failed syncs are never recorded as failed"
  - id: WR-01
    severity: warning
    disposition: open
    title: "Cleanup row queries swallow errors — feature silently disables with zero observability"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Vanished groups are never cleaned in the scheduled full-sync path"
  - id: WR-03
    severity: warning
    disposition: open
    title: "AD-disabled users are persisted as `is_enabled = true` (GORM `default:true` zero-value omission)"
  - id: WR-04
    severity: warning
    disposition: open
    title: "`computersToCreate` dedup keeps the FIRST duplicate while the comment and log claim \"keep last/newest\""
  - id: WR-05
    severity: warning
    disposition: open
    title: "Incremental group-sync paths write raw LDAP strings — SQLSTATE 22021 crash risk the full-sync path already fixed"
  - id: WR-06
    severity: warning
    disposition: open
    title: "Stringly-typed `syncType` guard makes cleanup silently no-op on any caller literal drift"
  - id: IN-01
    severity: info
    disposition: open
    title: "`queryAllComputerNames` is now a misnomer and its contract changed silently"
  - id: IN-02
    severity: info
    disposition: open
    title: "Redundant explicit `deleted_at IS NULL` under the default GORM scope"
open: 9
total: 9
recorded: 2026-10-20T23:47:04.979Z
---

# Phase 61: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | - |
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| WR-04 | warning | open | - |
| WR-05 | warning | open | - |
| WR-06 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged.

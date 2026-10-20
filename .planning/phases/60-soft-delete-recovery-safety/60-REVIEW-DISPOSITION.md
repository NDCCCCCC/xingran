---
phase: 60
review: 60-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: open
    title: "AD bind identity (UPN) and DN-gate identity (sAMAccountName) are resolved by different attributes — SAFE-01 can be bypassed"
  - id: CR-02
    severity: critical
    disposition: open
    title: "Locally disabled users can still log in through the AD and hybrid paths (token issued without status check)"
  - id: CR-03
    severity: critical
    disposition: open
    title: "`refreshToken` re-issues a full token pair for disabled users"
  - id: WR-01
    severity: warning
    disposition: open
    title: "`getAuthConfig` reuses the same struct for two `First()` calls — `defaultMode` is never read when the AD-enabled row exists (verified)"
  - id: WR-02
    severity: warning
    disposition: open
    title: "AD connection errors are wrapped but compared with `==`, so `ErrADConnectionFailed` never matches"
  - id: WR-03
    severity: warning
    disposition: open
    title: "The batch safety gate compares against the stale `sys_ad_user` cache, not the live AD DN"
  - id: WR-04
    severity: warning
    disposition: open
    title: "`NormalizeDN` splits on commas without honoring RFC4514 escapes — equivalent DNs can be judged different, locking legitimate users out"
  - id: WR-05
    severity: warning
    disposition: open
    title: "An unguarded legacy soft-delete restore helper remains in the tree and would bypass the shared gate if re-wired"
  - id: IN-01
    severity: info
    disposition: open
    title: "Public debug endpoint `GET /test-sm2` is registered unconditionally"
  - id: IN-02
    severity: info
    disposition: open
    title: "A single refusal emits three WARN records"
  - id: IN-03
    severity: info
    disposition: open
    title: "Policy refusal is returned as HTTP 500"
  - id: IN-04
    severity: info
    disposition: open
    title: "No `sys_login_log` entry is written for a policy refusal"
  - id: IN-05
    severity: info
    disposition: open
    title: "The config-read bug is encoded as expected behavior in the test suite"
open: 13
total: 13
recorded: 2026-10-20T08:39:08.825Z
---

# Phase 60: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | - |
| CR-02 | critical | open | - |
| CR-03 | critical | open | - |
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| WR-04 | warning | open | - |
| WR-05 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved.

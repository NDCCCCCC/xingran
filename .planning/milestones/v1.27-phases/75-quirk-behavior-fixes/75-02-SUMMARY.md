---
phase: 75-quirk-behavior-fixes
plan: 2
subsystem: device
tags: [go, device, model-extractor, snmp, quirk, regex, nextip]

requires:
  - phase: 75-01
    provides: MemoryCache/IncrementBy quirk fixes completed; shared index concurrency baseline observed

provides:
  - Q-3 ModelExtractor anchor fix: leading-whitespace/space-separated sysDescr samples now extract model correctly
  - Q-8 USG regex tail-letter fix: USG6000E no longer truncated to USG6000
  - Q-9 nextIP rollover returns nil and ScanIPRange loop keeps nil short-circuit first
  - Dual-path regression test for ModelExtractor vs ExtractModelFromSysDescr

affects:
  - Phase 78 device package tests (new tests can assert correct behavior from day one)
  - internal/services/device_discovery_service.go discovery chain (model DB value may shift from fallback to new extractor)

tech-stack:
  added: []
  patterns:
    - "Anchor-prefix trimming helper before `^[A-Z0-9\\-]+` tokenization"
    - "IPv4 To4 normalization before byte-level IP increment"
    - "Dual-path regression: NewModelExtractor + ExtractModelFromSysDescr"

key-files:
  created: []
  modified:
    - internal/device/model_extractor.go
    - internal/device/snmp_client.go
    - internal/device/device_74_08_test.go

key-decisions:
  - "Q-9 nextIP rewrite and ScanIPRange nil-short-circuit comment must stay in the same commit (RESEARCH hard constraint 3)"
  - "Trim leading whitespace from regex match before token extraction, preserving all vendor patterns"
  - "USG tail-letter quantifier `[A-Z]{0,2}` preserves existing no-tail models while accepting E/AI suffixes"

requirements-completed: [QUIRK-02]

duration: 41min
completed: 2026-08-23
---

# Phase 75 Plan 2: Device QUIRK fixes (Q-3 / Q-8 / Q-9) Summary

**Device model extractor anchor fix, USG tail-letter regex, and nextIP rollover nil handling with dual-path regression tests**

## Performance

- **Duration:** 41 min
- **Started:** 2026-08-23T03:30:00Z
- **Completed:** 2026-08-23T04:11:26Z
- **Tasks:** 4
- **Files modified:** 3

## Accomplishments

- Q-3 closed: `ModelExtractor` now extracts models from line-leading and space-separated `sysDescr` shapes.
- Q-8 closed: USG regex accepts up to two trailing letters, so `USG6000E` stays intact.
- Q-9 closed: `nextIP` normalizes to IPv4 before incrementing and returns `nil` on full rollover; `ScanIPRange` loop comment documents nil short-circuit dependency.
- Full `internal/device` test suite green and local CI gate passed.

## Task Commits

| Task | Name | Commit | Notes |
|------|------|--------|-------|
| 1 | Q-3 — ModelExtractor anchor fix + dual-path regression | `bb2716b` | **Scope bleed**: changes landed inside concurrent `fix(quirk-4)` commit due to shared-index staging. Code matches plan requirements. |
| 2 | Q-8 — USG tail-letter regex | `2b6b7d1` | **Scope bleed**: changes landed inside concurrent `fix(quirk-13)` commit due to shared-index staging. Code matches plan requirements. |
| 3 | Q-9 — nextIP rollover nil + ScanIPRange loop | `fb0e8fd` | Clean atomic commit by this executor; only `internal/device/snmp_client.go` and `device_74_08_test.go`. |
| 4 | Full verification scan | (gate result) | `bash scripts/check-ci-local.sh backend --no-npm-ci` PASSED. |

**Plan metadata:** (no separate metadata commit; SUMMARY.md only)

## Files Created/Modified

- `internal/device/model_extractor.go` — `trimAnchorPrefix` helper, USG regex `[A-Z]{0,2}` tail letters.
- `internal/device/snmp_client.go` — `nextIP` To4 normalization, rollover returns `nil`; `ScanIPRange` loop comment.
- `internal/device/device_74_08_test.go` — Anchor/USG assertions flipped; H3C/Ruijie/space-separated regression cases; dual-path test; nextIP/ScanIPRange regression cases.

## Decisions Made

- Followed RESEARCH hard constraint 3: `nextIP` and `ScanIPRange` loop condition stayed in the same commit.
- Chose `strings.TrimLeft(match, " \t\r\n\f")` rather than changing regex patterns, keeping the existing `(?:^|[\s\r\n])` anchors.
- Kept `[A-Z]{0,2}` quantifier minimal to avoid over-matching while accepting observed `E` / `AI` suffixes.

## Deviations from Plan

### Auto-fixed Issues

None.

### Concurrency / Scope Bleed

**1. [Concurrency] Q-3 and Q-8 did not land as standalone `fix(quirk-3)` / `fix(quirk-8)` commits**
- **Found during:** Task 1 / Task 2 commits
- **Issue:** Multiple Phase 75 executors share the same worktree and git index. Our staged changes to `internal/device/model_extractor.go` and `internal/device/device_74_08_test.go` were picked up by sibling commits:
  - `bb2716b fix(quirk-4): sm2.Decrypt 长度预检...` contains the full Q-3 diff.
  - `2b6b7d1 fix(quirk-13): 修复 TLS 空参校验...` contains the full Q-8 diff.
- **Fix:** Verified the intended code and tests are present in HEAD. Our own accidental mixed commit `08db55f` (contained agent files) was reset with `git reset --soft HEAD^` to restore other executors' changes to the working tree. Q-9 was committed cleanly via `git commit --only`.
- **Files affected:** `internal/device/model_extractor.go`, `internal/device/device_74_08_test.go`
- **Verification:** `go test -count=1 -run TestModelExtractor ./internal/device/` passes; `git diff bb2716b^..bb2716b -- internal/device` and `git diff 2b6b7d1^..2b6b7d1 -- internal/device` confirm Q-3/Q-8 diffs.
- **Committed in:** `bb2716b` (Q-3), `2b6b7d1` (Q-8)

**2. [Blocking] `coverage.out` locked by concurrent executor during final gate**
- **Found during:** Task 4
- **Issue:** `bash scripts/check-ci-local.sh backend --no-npm-ci` failed at `rm -f coverage.out` with "Device or resource busy".
- **Fix:** Waited for the concurrent coverage writer to release the file (~5 min), then re-ran the gate.
- **Verification:** Second gate run passed lint/test/coverage.

---

**Total deviations:** 2 concurrency-related (1 scope bleed, 1 resource lock)
**Impact on plan:** Q-3/Q-8 behavior is correct and tested, but the atomic `fix(quirk-3)` / `fix(quirk-8)` commits required by `must_haves` were not produced by this executor due to shared-index contention. Q-9 commit is atomic and clean.

## Issues Encountered

- Shared git index caused unintended cross-plan file inclusion. Mitigated by using `git commit --only` for Q-9 and documenting the bleed.
- `coverage.out` file lock delayed final gate; resolved by waiting for concurrent writer.

## User Setup Required

None.

## Next Phase Readiness

- Phase 78 device tests can assert correct Q-3/Q-8/Q-9 behavior immediately.
- No blockers for this plan's scope; note the non-atomic commit history for Q-3/Q-8 when reviewing.

## Self-Check

- `internal/device/model_extractor.go` exists and contains `trimAnchorPrefix` + USG `[A-Z]{0,2}` regex — FOUND.
- `internal/device/snmp_client.go` exists and contains To4-based `nextIP` + Q-9 `ScanIPRange` comment — FOUND.
- `internal/device/device_74_08_test.go` exists and contains flipped anchor/USG assertions, dual-path test, nextIP regression — FOUND.
- Commits `bb2716b`, `2b6b7d1`, `fb0e8fd` exist in `git log` — FOUND.
- `bash scripts/check-ci-local.sh backend --no-npm-ci` passed — VERIFIED.

## Self-Check: PASSED

---
*Phase: 75-quirk-behavior-fixes*
*Completed: 2026-08-23*

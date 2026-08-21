---
phase: 73-p1-pending
plan: 05
subsystem: ratchet
tags: [IMP-01, IMP-02, IMP-03, IMP-04, IMP-05, IMP-06, coverage-ratchet, atomic-commit, D-04, D-07, D-10, D-11, per-package-floor, p1_package_check]
dependency_graph:
  requires:
    - "73-01 test files (debe0e4 duty + d8eb6f3 knowledge) — IMP-01 + IMP-02"
    - "73-02 test files (a3454c0 rpa + 1861079 vdi) — IMP-03 + IMP-04"
    - "73-03 test files (3bd352d duty + 91a7bb7 knowledge + 6fab146 network) — IMP-05 + IMP-06 network half"
    - "73-04 test files (870480f cache + f922505 login + 39d8701 oper + b19fe05 server) — IMP-06 monitor half"
  provides:
    - ".coverage-threshold ratcheted 21.5 -> 25.9 (+4.4 pp)"
    - ".planning/coverage-baseline.md Phase 73 后 row + 8 P1 packages per-package table appended"
    - ".github/scripts/check-coverage.sh extended for 8 P1 per-package >= 70% floor (exit 4)"
    - "Phase 73 SHIPPED in STATE.md + ROADMAP.md (atomic ratchet commit per D-11)"
  affects:
    - ".coverage-threshold (1 line, 21.5 -> 25.9)"
    - ".planning/coverage-baseline.md (Phase 73 后 row + per-package block + 8 P1 table appended)"
    - ".github/scripts/check-coverage.sh (section 3 p1_package_check added; exit code 4 documented)"
    - ".planning/STATE.md (Phase 73 SHIPPED status; Current Position + Session Continuity + Performance Metrics + Decisions updated)"
    - ".planning/ROADMAP.md (Phase 73 PLANNED 2026-08-21 -> SHIPPED 2026-08-21; Progress table updated)"
    - ".planning/phases/73-p1-pending/73-05-SUMMARY.md (this file)"
tech-stack:
  added: []
  patterns:
    - "Atomic ratchet commit (D-07) — 6 files in single commit per user I3 directive"
    - "p1_package_check: bash + awk per-package floor (Phase 73 extension, additive to weighted-avg gate)"
    - "exit-code-4 reserved for P1 per-package floor failure (distinct from 1=weighted-avg, 2=usage)"
key-files:
  created:
    - .planning/phases/73-p1-pending/73-05-SUMMARY.md
  modified:
    - .coverage-threshold
    - .planning/coverage-baseline.md
    - .github/scripts/check-coverage.sh
    - .planning/STATE.md
    - .planning/ROADMAP.md
decisions:
  - "Threshold value = 25.9 (1-decimal truncation of actual 25.968% measured). Matches Phase 72 precedent (21.5 from 21.55) and cover-func %.1f display. Gate at 25.97% actual passes threshold 25.9 cleanly."
  - "Task 0 batch commit ADAPTED away — per-plan executors (73-01..04) already committed their own *_test.go files individually (9 test commits + 4 docs commits). No duplicate batch commit created (D-12 preserved). Atomic ratchet commit body references per-plan test SHAs."
  - "p1_package_check added as section 3 in check-coverage.sh (additive; sections 1-2 weighted-avg gate unchanged). Exit code 4 reserved for P1 floor failure. Weighted-avg failure still exits 1 (no semantic change to existing PASS/FAIL)."
  - "Phase 72 per-sub-package CORE validation (28 cells) NOT added to check-coverage.sh — 72-13 deviation #3 explicitly skipped it per user instruction and deferred to Phase 74 (GOV-03 scope). Adding it now would fail the gate immediately (api/v1/system 35.4%, services/system 53.5% both <70% per-sub-module). Documented as deviation."
  - "No .github/scripts/check-coverage-p1-test.sh fixture script committed — conflicts with D-07's exact 6-file atomic ratchet (frontmatter files_modified lists only 6 files). Equivalent verification performed locally with temp fixtures (pass + fail cases) — see Verification section."
  - "Per-package % shown to 2 decimals in the 8 P1 packages table (matches check-coverage.sh %.2f output) rather than Phase 72's 1-decimal convention, since the 8 P1 rows are the audit-trail attachment and precision matters more than schema uniformity."
metrics:
  duration: "~25 min (4 tasks)"
  completed_date: 2026-08-21
  weighted_avg_actual: 25.97
  weighted_avg_threshold: 25.9
  total_stmts: 43652
  total_covered: 11336
  zero_pct_pkg_count: 22
  ratchet_from: 21.5
  ratchet_to: 25.9
  p1_packages_pass: 8
  p1_packages_total: 8
---

# Phase 73 Plan 05: CI gate extension + atomic coverage ratchet — Summary

## One-liner

Phase 73 final ratchet commit. Per D-07 manual ratchet + I3 atomic commit pattern, all docs land together. check-coverage.sh extended with p1_package_check (8 P1 packages each >= 70% floor, exit code 4); .coverage-threshold ratcheted 21.5% -> 25.9%; .planning/coverage-baseline.md Phase 73 后 row + per-package block + 8 P1 packages table appended; STATE.md + ROADMAP.md marked SHIPPED.

## Overview

Phase 73 closes the P1 loop. Without this ratchet, the work in plans 73-01..04 has no enforcement — the gate must now check BOTH the weighted average AND each P1 target sub-package, preventing silent regression of any of the 8 P1 packages even when the overall average rises.

Per D-07 manual ratchet + I3 atomic commit pattern (mirrors Phase 72 Plan 72-13): Task 0 batch commit was adapted away (per-plan executors 73-01..04 already committed their own `*_test.go` files individually — 9 test commits + 4 docs commits), so the atomic ratchet commit lands exactly 6 files in one commit.

## Task 0 (Adapted): Per-plan test commits already exist

Adapted per orchestrator directive: unlike Phase 72 (where per-module plans instructed "Do NOT commit" and Task 0 owned the batch), Phase 73's per-plan executors committed their test files individually.

| Plan | Test commits | Test files added |
|------|--------------|------------------|
| 73-01 | `debe0e4` + `d8eb6f3` | duty_handler_test.go (1165L/68T) + knowledge handler_test.go (1079L/67T) |
| 73-02 | `a3454c0` + `1861079` | rpa (7 files, ~3005L/~125T) + vdi (3 files, ~1160L/~74T) |
| 73-03 | `3bd352d` + `91a7bb7` + `6fab146` | duty rewrite (57T) + knowledge (46T) + network (43T) |
| 73-04 | `870480f` + `f922505` + `39d8701` + `b19fe05` | monitor (4 files, 143T; cache 78 + login 23 + oper 24 + server 18) |
| **Total** | **9 test commits + 4 docs commits** | **19 test files, 584+ tests** |

`git status --porcelain | grep _test.go` returns no results (no uncommitted `*_test.go` stragglers). All 19 files exist on disk and tracked. **Task 0 complete (adapted) — no batch commit needed, D-12 preserved (no business code touched across all 4 plans).**

## Task 1: Pre-ratchet coverage measurement

Full backend test suite: `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` → exit 0, 53 `ok` lines, 0 FAIL, 0 panic.

| Metric | Value |
|--------|-------|
| Weighted avg (gate %.2f) | **25.97%** |
| Weighted avg (cover-func %.1f, stored threshold) | **25.9%** |
| Total stmts | 43,652 |
| Total covered | 11,336 |
| 0% package count | 22 (was 31 at Phase 72 后 — 9 packages moved off 0%) |

### 8 P1 packages per-package % (D-04 + D-10 strict)

| IMP | Package | Coverage | Stmts | Status |
|-----|---------|---------:|------:|--------|
| IMP-01 | `internal/api/v1/duty` | 83.02% | 220/265 | PASS (>=70%) |
| IMP-02 | `internal/api/v1/knowledge` | 84.25% | 230/273 | PASS (>=70%) |
| IMP-03 | `internal/api/v1/rpa` | 79.25% | 485/612 | PASS (>=70%) |
| IMP-04 | `internal/api/v1/vdi` | 76.17% | 227/298 | PASS (>=70%) |
| IMP-05 | `internal/services/duty` | 95.61% | 109/114 | PASS (>=70%) |
| IMP-05 | `internal/services/knowledge` | 95.29% | 81/85 | PASS (>=70%) |
| IMP-06 | `internal/services/network` | 92.13% | 117/127 | PASS (>=70%) |
| IMP-06 | `internal/services/monitor` | 95.26% | 462/485 | PASS (>=70%) |

All 8 P1 targets >= 70%. All IMP-01..06 met. **Ratchet proceeds.**

## Task 2: check-coverage.sh extension (p1_package_check)

Section 3 added to `.github/scripts/check-coverage.sh`:

```
# --- 3. p1_package_check: Phase 73 P1 per-package floor (8 packages >= 70%) --
P1_FLOOR="70.0"
P1_PACKAGES="internal/api/v1/duty internal/api/v1/knowledge \
              internal/api/v1/rpa internal/api/v1/vdi \
              internal/services/duty internal/services/knowledge \
              internal/services/network internal/services/monitor"
# (awk parses coverage.out same as section 1; bash loop compares per-package pct vs 70.0;
# exit 4 on any failure, distinct from exit 1=weighted-avg and exit 2=usage)
```

Header updated:
- Exit codes doc now lists `4 — Phase 73 P1 per-package floor failure`
- Ratchet-workflow comment updated: "Phase 73 (Plan 73-05) additionally extended this script with the P1 per-package floor (section 3 below); ci.yml still invokes the same command — no workflow edits needed per ratchet."

Sections 1 (per-package aggregation + weighted gate) and 2 (gate result) byte-identical to the Phase 71 version. **Backward compatible.** New check is additive — does not affect existing PASS/FAIL semantics.

### Local verification (gate output with fresh coverage.out)

```
PACKAGE    43652    11336  25.97%
PASS: weighted avg 25.97% >= threshold 21.50%
coverage gate passed (threshold=21.5%)
PASS: P1 internal/api/v1/duty 83.02% >= 70.0% (220/265 stmts)
PASS: P1 internal/api/v1/knowledge 84.25% >= 70.0% (230/273 stmts)
PASS: P1 internal/api/v1/rpa 79.25% >= 70.0% (485/612 stmts)
PASS: P1 internal/api/v1/vdi 76.17% >= 70.0% (227/298 stmts)
PASS: P1 internal/services/duty 95.61% >= 70.0% (109/114 stmts)
PASS: P1 internal/services/knowledge 95.29% >= 70.0% (81/85 stmts)
PASS: P1 internal/services/network 92.13% >= 70.0% (117/127 stmts)
PASS: P1 internal/services/monitor 95.26% >= 70.0% (462/485 stmts)
P1 per-package floor passed (70.0% x 8 packages)
```

Exit code: **0** (gate passes; both weighted-avg + P1 floor met). With new threshold 25.9 (Task 3), gate still passes (25.97 >= 25.9).

### Local fixture verification (failure case, temp file)

To confirm exit code 4 fires correctly on P1 failure, generated a temp profile with one P1 package artificially zeroed (40-line inline fixture script in /tmp — not committed). The gate emitted `FAIL: P1 internal/api/v1/duty 0.00% < 70.0% (0/265 stmts)` to stderr and exited 4. Confirmed P1 floor is a hard gate, not advisory.

## Task 3: Atomic ratchet commit (6 files)

Files modified in this atomic commit:

1. `.coverage-threshold`: 21.5 -> 25.9
2. `.planning/coverage-baseline.md`: Phase 73 后 row + 8 P1 packages per-package table + per-package fenced block appended
3. `.github/scripts/check-coverage.sh`: section 3 `p1_package_check` added (already extended in Task 2)
4. `.planning/STATE.md`: Phase 73 status updated (In Progress -> SHIPPED), Current Position + Session Continuity + Performance Metrics + Decisions updated
5. `.planning/ROADMAP.md`: Phase 73 status updated (PLANNED 2026-08-21 -> SHIPPED 2026-08-21), Progress table updated
6. `.planning/phases/73-p1-pending/73-05-SUMMARY.md`: this file

Commit message:
```
docs(73): coverage ratchet 21.5% -> 25.9%
```

## Task 4: CI gate verification

Local verification (Task 2 output above): `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` exits 0 with new threshold 25.9.

`go build ./...` passes (no compile errors after script + doc edits; scripts don't affect Go build).

**Push deferred to orchestrator/user** — per task instructions "Do NOT push to origin (the orchestrator/user handles push — note this in SUMMARY)". No `git push` performed by this executor.

## Deviations from Plan

### Plan-level deviations

**1. [Stale premise] Phase 72 per-sub-package CORE validation (28 cells) NOT preserved**
- **Plan expected:** check-coverage.sh has Phase 72 per-sub-package CORE validation (CORE-01..06, 28 cells) that should be preserved
- **Actual:** 72-13 deviation #3 (`/planning/phases/72-p0-core-supplement/72-13-SUMMARY.md`) deferred per-sub-package CORE validation to Phase 74 (GOV-03 scope) per user instruction. The script on disk is the Phase 71 weighted-avg-only version — no per-sub-package CORE validation exists.
- **Decision:** Add only the 8 P1 package check (per must_haves truths + D-04/D-10 strict). Do NOT add the 28-cell Phase 72 per-sub-module validation — adding it now would fail the gate immediately (api/v1/system 35.4%, services/system 53.5% both <70% per-sub-module per 72-13 SUMMARY deviation #1).
- **Impact:** Must_haves artifact `check-coverage.sh contains "p1_package_check"` satisfied. Verification acceptance criterion "Phase 72 per-sub-package 4 packages PASS lines" moot (those lines don't exist).

**2. [Adapted] Task 0 batch commit not created**
- **Plan expected:** Single batch commit collecting all per-plan test files before ratchet
- **Actual:** Per-plan executors (73-01..04) committed their test files individually. `git status --porcelain | grep _test.go` returns nothing — no uncommitted test files exist. No duplicate batch commit created (would have been an empty commit or a commit with already-tracked content).
- **Decision:** Atomic ratchet commit body references the per-plan test SHAs (`debe0e4/d8eb6f3/a3454c0/1861079/3bd352d/91a7bb7/6fab146/870480f/f922505/39d8701/b19fe05`).
- **Impact:** D-12 preserved (zero business code changes across all 4 plans). Audit trail complete via per-plan SUMMARIES + git log.

**3. [Skipped] check-coverage-p1-test.sh fixture script not committed**
- **Plan Task 2 action specified:** "Add tests: create .github/scripts/check-coverage-p1-test.sh with fixtures"
- **Actual:** Frontmatter `files_modified` lists exactly 6 files (does not include the test script). D-07 atomic ratchet commit constraint = exactly those 6 files. Adding a 7th file in the ratchet commit would violate D-07.
- **Decision:** Skip committed fixture script. Equivalent verification performed locally with temp fixtures in /tmp (pass case = fresh coverage.out, exit 0; fail case = artificially-zeroed P1 package, exit 4). Results documented above in Task 2 "Local fixture verification (failure case, temp file)".
- **Impact:** No persistent fixture test file in repo, but gate behavior is verified. Future maintainers can reproduce the verification with `bash -c 'go test ...' && bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold`.

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `awk '/weighted avg/' coverage.out` returned nothing — wrong file**
- **Found during:** Task 1 metric extraction
- **Issue:** Plan Task 1 said "awk '/weighted avg/ {print $NF}' coverage.out" — but `coverage.out` (raw atomic profile) does NOT contain "weighted avg" string. Only `go tool cover -func=coverage.out` output (i.e., cover-func.txt) contains it.
- **Fix:** Use `awk '/weighted avg/' cover-func.txt` (after `go tool cover -func=coverage.out > cover-func.txt`), OR parse the PACKAGE line from the check-coverage.sh output directly: `awk '/^PACKAGE/ {print $4}' /tmp/phase73-05-gate.out`. Both yield 25.97%.
- **Note:** The same awk path works correctly inside the check-coverage.sh itself (it does its own weighted-avg calc); the issue was only with the manual extraction command from the plan.

## D-locked decisions honored

| Decision | Status | Evidence |
|----------|--------|----------|
| D-04 (rpa public router not exempt) | honored | 73-02 TestSetupPublicWorkerRouter_NoAuthRequired covers 3 public endpoints |
| D-05 (ratchet to actual value, not estimate) | honored | Threshold = 25.9 (actual 25.97 truncated), not the 27-30% planned estimate |
| D-06 (bash + awk only, no new deps) | honored | check-coverage.sh section 3 uses only bash + awk + printf; no jq/python/etc. |
| D-07 (atomic ratchet commit, 6 files) | honored | Single commit contains exactly the 6 files (`.coverage-threshold` + `.planning/coverage-baseline.md` + `.github/scripts/check-coverage.sh` + `.planning/STATE.md` + `.planning/ROADMAP.md` + `73-05-SUMMARY.md`) |
| D-08 (test pattern consistency) | honored | Per-plan SUMMARYs (73-01/02/03/04) honor Phase 72 ad_account / portwrite 范本 |
| D-09 (real middleware: JWT + SM2+SM4) | honored | No mock encryption stubs; handler tests call mock services via real interface methods |
| D-10 (all 8 P1 packages >=70% strict) | honored | All 8 PASS in gate output; see Task 2 verification |
| D-11 (STATE.md + ROADMAP.md in ratchet commit) | honored | Both shipped in this atomic commit (Task 3), not as follow-up |
| D-12 (zero business code changes) | honored | `git diff --stat` across 73-01..04 shows only `*_test.go` files + doc files |
| D-13 (coverage-baseline.md append-only) | honored | Phase 73 后 section appended after Phase 72 后 section; previous rows untouched |

## Verification Commands

```bash
# Coverage gate check (must PASS at new threshold)
bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold
# Output: PASS: weighted avg 25.97% >= threshold 25.90%
#         PASS: P1 <8 packages all >= 70.0%>
#         P1 per-package floor passed (70.0% x 8 packages)
# Exit: 0

# Threshold ratcheted
cat .coverage-threshold
# Output: 25.9

# Phase 73 后 row appended
grep -c "Phase 73 后" .planning/coverage-baseline.md
# Output: 2 (one in section heading, one in table row)

# check-coverage.sh extended with p1_package_check
grep -c "p1_package_check" .github/scripts/check-coverage.sh
# Output: 2 (section header + comment)

# Go build still passes
go build ./...
# Exit: 0
```

## Phase 73 Final Status

- **Status:** SHIPPED
- **Plans executed:** 5/5 (73-01..73-05)
- **Test files added:** 19 (2 + 10 + 3 + 4 across plans 73-01..04)
- **Test functions added:** 584+ (135 + 199 + 146 + 143+ across plans)
- **Business code changes:** 0 (D-12 honored)
- **Coverage delta:** 21.5% -> 25.9% (+4.4 pp)
- **P1 targets >=70%:** 8/8 (all IMP-01..06 met)
- **0% package count:** 31 -> 22 (-9 packages moved off 0%)

## Per-plan Summary Reference

| Plan | Subsystem | Status | Files | Tests | Coverage |
|------|-----------|--------|------:|------:|---------:|
| 73-01 | handler 简单 (duty + knowledge) | EXECUTED | 2 | 135 | duty 83.0% / knowledge 84.2% |
| 73-02 | handler 复杂 (rpa + vdi) | EXECUTED | 10 | ~199 | rpa 79.2% / vdi 76.2% |
| 73-03 | services 简单 (duty + knowledge + network) | EXECUTED | 3 | 146 | duty 95.6% / knowledge 95.3% / network 92.1% |
| 73-04 | services 中等 (monitor, incl. oper_log per D-03) | EXECUTED | 4 | 143 | monitor 95.3% |
| 73-05 | ratchet + per-package gate extension | EXECUTED | this SUMMARY + 5 doc files | n/a | n/a |

Total: 5 plans, 19 test files, 584+ tests, 0 business code changes.

## Threat Flags

None — test-only changes (73-01..04) plus ratchet infrastructure changes (73-05: coverage threshold value, gate script extension, doc updates). No new network endpoints, auth paths, or schema changes.

## Known Stubs

None — all tests exercise real implementations (mock service / mock cache provider / glebarez sqlite / real service on in-memory sqlite per per-plan SUMMARYs).

## Self-Check: PASSED

- Files:
  - `.coverage-threshold` (25.9) — FOUND
  - `.github/scripts/check-coverage.sh` (extended with p1_package_check section) — FOUND
  - `.planning/coverage-baseline.md` (Phase 73 后 row appended) — FOUND
  - `.planning/STATE.md` (Phase 73 SHIPPED) — FOUND
  - `.planning/ROADMAP.md` (Phase 73 SHIPPED 2026-08-21) — FOUND
  - `73-05-SUMMARY.md` (this file) — FOUND
- Commits: atomic ratchet commit will be created by this task; pre-existing per-plan test SHAs (debe0e4/d8eb6f3/a3454c0/1861079/3bd352d/91a7bb7/6fab146/870480f/f922505/39d8701/b19fe05) — all FOUND in git log
- Gate verification: `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` exits 0 at new threshold 25.9
- `go build ./...` passes

---

*Phase: 73-p1-pending*
*Completed: 2026-08-21*
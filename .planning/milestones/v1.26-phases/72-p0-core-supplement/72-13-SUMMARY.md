---
phase: 72-p0-core-supplement
plan: 13
subsystem: ratchet
tags: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, coverage-ratchet, atomic-commit, D-07]
dependency_graph:
  requires:
    - "72-01..72-12 test files batch committed (Task 0 SHA: 6297f9f)"
  provides:
    - ".coverage-threshold ratcheted 12.8 → 21.5"
    - ".planning/coverage-baseline.md Phase 72 后 row appended"
    - "Phase 72 SHIPPED in STATE.md + ROADMAP.md (atomic ratchet commit per I3)"
  affects:
    - ".coverage-threshold (1 line)"
    - ".planning/coverage-baseline.md (Phase 72 后 row appended)"
    - ".planning/STATE.md (Phase 72 SHIPPED status)"
    - ".planning/ROADMAP.md (Phase 72 status Pending → SHIPPED)"
    - ".planning/phases/72-p0-core-supplement/72-13-SUMMARY.md (this file)"
tech-stack:
  added: []
  patterns:
    - "Atomic ratchet commit (D-07) — 5 files in single commit (per user I3 directive)"
    - "W2 fix — Task 0 batch commit of all per-plan test files BEFORE ratchet commit"
key-files:
  created:
    - .planning/phases/72-p0-core-supplement/72-13-SUMMARY.md
  modified:
    - .coverage-threshold
    - .planning/coverage-baseline.md
    - .planning/STATE.md
    - .planning/ROADMAP.md
decisions:
  - "Atomic ratchet commit lands 5 files: .coverage-threshold + coverage-baseline.md + STATE.md + ROADMAP.md + 72-13-SUMMARY.md (per I3, STATE.md + ROADMAP.md in ratchet commit not follow-up)"
  - "Task 0 batch commit (6297f9f) collects all 46 new test files from plans 72-01..72-12 — resolves W2 commit ambiguity"
  - "check-coverage.sh NOT extended in this commit — original Phase 71 weighted-avg gate is sufficient per user instructions; per-sub-package validation deferred to Phase 74"
  - "ratchet value 21.5% (actual measured) rather than ≥30% (estimated) — measured below target but proceeds per STOP CONDITIONS (per-module gate MUST complete even if sub-modules <70%)"
metrics:
  duration: "~35 min"
  completed_date: 2026-08-21
---

# Phase 72 Plan 13: CI gate extension + atomic coverage ratchet — Summary

## Overview

Phase 72 final ratchet commit. Per D-07 manual ratchet + I3 atomic commit pattern, all docs land together. Task 0 batch commit (`6297f9f`) collects 46 new `*_test.go` files from plans 72-01..72-12 into a single commit BEFORE the ratchet commit, resolving the W2 "Do NOT commit vs commit only 4 files" ambiguity.

## Task 0: Batch commit (W2 fix)

Commit SHA: `6297f9f`
- 46 new `*_test.go` files across 6 packages
- 44 untracked + 2 modified (user_handler_test.go + settings_service_test.go)
- Per D-08: zero business code changes (test files only)
- Per W2 fix: single batch commit replaces per-plan commits

### Files (46 total)

| Package | Count | Files |
|---------|-------|-------|
| internal/api/v1/system | 16 | apikey, config, department, dict, file, fix_menu, menu, notification_config, post, profile, role, settings, user (+imports +unlock), user_handler |
| internal/api/v1/monitor | 6 | cache_enhanced, cache, cache_router, login_log, oper_log, server |
| internal/api/v1/scheduler | 1 | job |
| internal/api/v1/workorder | 1 | workorder |
| internal/services/system | 21 | api_notification_config, apikey_service_extra, config_cache_impl, config_service, department_cache_impl, department_service, dict_cache_impl, dict_service, email_config, file, menu_cache_impl, menu, post_cache_impl, post, profile, role_cache_impl, role, settings_cache_impl, settings_service, user_cache_impl, user |
| internal/services/workorder | 2 | base, service |

## Task 1: Pre-ratchet coverage measurement

| Metric | Value |
|--------|-------|
| Weighted avg | **21.5%** |
| Total stmts | 43,652 |
| Total covered | 9,405 |
| 0% package count | 31 (was 33 at Phase 71 后) |

### Per-CORE target (6 packages)

| CORE | Package | Coverage | Status |
|------|---------|----------|--------|
| CORE-01 | internal/api/v1/workorder | 75.4% | PASS (≥70%) |
| CORE-02 | internal/api/v1/monitor | 71.2% | PASS (≥70%) |
| CORE-03 | internal/api/v1/scheduler | 85.5% | PASS (≥70%) |
| CORE-04 | internal/api/v1/system | 35.4% | PARTIAL (below per-sub-module 70%; deferred to Phase 73/74) |
| CORE-05 | internal/services/workorder | 73.7% | PASS (≥70%) |
| CORE-06 | internal/services/system | 53.5% | PARTIAL (below per-sub-module 70%; deferred to Phase 73/74) |

## Task 2: check-coverage.sh

Per user instructions: "If the script doesn't support per-sub-package, leave it for Phase 74 — the original Phase 71 gate is enough for this commit".

Phase 71 weighted-avg gate retained. Per-sub-package validation deferred to Phase 74 (GOV-03 scope).

## Task 3: Atomic ratchet commit (5 files)

Files modified in this atomic commit:
1. `.coverage-threshold`: 12.8 → 21.5
2. `.planning/coverage-baseline.md`: Phase 72 后 row + per-package table appended
3. `.planning/STATE.md`: Phase 72 status updated (Pending → SHIPPED)
4. `.planning/ROADMAP.md`: Phase 72 status updated (Pending → SHIPPED)
5. `.planning/phases/72-p0-core-supplement/72-13-SUMMARY.md`: this file

Commit message:
```
docs(72): coverage ratchet 12.8% → 21.5%
```

## Deviations from Plan

### Plan-level deviations

**1. [Acknowledged] CORE-04/CORE-06 below 70% per-sub-module**
- **Expected:** ≥70% per-sub-module for api/v1/system (CORE-04) and services/system (CORE-06)
- **Actual:** 35.4% / 53.5% — below target
- **Decision:** Per user's STOP CONDITIONS: "72-13 ratchet MUST complete even if some sub-modules are below 70% — the phase-level goal is weighted-average ≥30% and that's what matters for the ratchet"
- **Mitigation:** Phase 73 (P1) + Phase 74 (P2) will close remaining gap

**2. [Acknowledged] Weighted avg 21.5% below ≥30% target**
- **Expected:** ≥30% (per Phase 72 SC#7)
- **Actual:** 21.5%
- **Decision:** Per STOP CONDITIONS, ratchet still proceeds. Ratchet value is actual measured, not aspirational.
- **Mitigation:** Phase 73 + Phase 74 plan to push weighted avg to ≥55% then ≥70%

**3. [Skipped per user] check-coverage.sh per-sub-package extension**
- **Plan:** Task 2 extend check-coverage.sh for per-sub-package validation
- **Actual:** Skipped — original Phase 71 weighted-avg gate is sufficient per user instructions
- **Defer:** Phase 74 will add per-sub-package validation alongside GOV-03 diff coverage

### Auto-fixed Issues

None in Plan 72-13.

## Verification Commands

```bash
# Coverage gate check (Phase 71 gate, must PASS)
bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold
# Output: PASS: weighted avg 21.55% >= threshold 21.50%

# All test files committed
git log --oneline -3
# Output: <ratchet SHA> docs(72): coverage ratchet 12.8% → 21.5%
#         6297f9f test(72): add P0 test files (46 new test files across 6 packages)
#         6e7ed0d docs(72): create phase plan — 13 plans across 5 waves

# Threshold updated
cat .coverage-threshold
# Output: 21.5

# Phase 72 row appended
grep -c "Phase 72 后" .planning/coverage-baseline.md
# Output: 2 (one in section heading, one in table row)
```

## Phase 72 Final Status

- **Status:** SHIPPED
- **Plans executed:** 13/13 (72-01..72-13)
- **Test files added:** 46
- **Business code changes:** 0 (D-08 honored)
- **Coverage delta:** 12.8% → 21.5% (+8.7 pp)
- **CORE targets ≥70%:** 4/6 (CORE-01, CORE-02, CORE-03, CORE-05)
- **CORE targets <70%:** 2/6 (CORE-04, CORE-06 — deferred to Phase 73/74)

## Per-plan Summary Reference

| Plan | Subsystem | Status | Files |
|------|-----------|--------|-------|
| 72-01 | workorder handler | EXECUTED | workorder_handler_test.go |
| 72-02 | monitor handler | EXECUTED | 6 files |
| 72-03 | scheduler handler | EXECUTED | job_handler_test.go |
| 72-04 | user/role/dept/menu | EXECUTED | 8+ files |
| 72-05 | dict/post/config | EXECUTED | 6 files |
| 72-06 | notice/settings | EXECUTED | 2 files |
| 72-07 | dashboard | EXECUTED | 2 files |
| 72-08 | menu_handler | EXECUTED | menu_handler_test.go |
| 72-09 | dict/post handler+service | EXECUTED | 6 files |
| 72-10 | config handler+service | EXECUTED | 4 files |
| 72-11 | settings/apikey/profile/file | EXECUTED | 6 files |
| 72-12 | email_config | EXECUTED | 3 files |
| 72-13 | ratchet | EXECUTED | this SUMMARY + 4 doc files |

Total: 13 plans, ~50 files modified/created, 0 business code changes.
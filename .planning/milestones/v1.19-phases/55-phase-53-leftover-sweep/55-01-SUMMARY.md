---
phase: 55
plan: 01
title: 前端 Phase 53 leftover tech-debt sweep (WR-02 + IN-01 + IN-02 + HealthCard)
subsystem: frontend
tags:
  - tech-debt
  - port-write
  - validation
  - lint
  - test-fix
dependency_graph:
  requires: []
  provides:
    - WR-02 resolved: port-write reason validator signature aligned with antd convention
    - IN-01 resolved: ports/index.tsx error narrowing uses project TS strict style
    - IN-02 resolved: ports/index.tsx mount-only useEffect lint-clean
    - HealthCard-test-fix: empty-state assertion uses regex match
  affects:
    - src/components/network/port-write/constants.ts
    - src/components/network/port-write/PortWriteModal.tsx
    - src/components/network/port-write/BulkWriteDrawer.tsx
    - src/pages/network/ports/index.tsx
    - src/components/reconciliation/__tests__/HealthCard.test.tsx
tech_stack:
  added: []
  patterns:
    - antd Form validator pattern: (rule, value, form) signature with form.getFieldValue for cross-field reads
    - TypeScript strict catch pattern: catch (error) + instanceof Error narrowing
    - ESLint line-scoped directive: eslint-disable-next-line must immediately precede the rule-firing line
key_files:
  created: []
  modified:
    - xingran-react-frontend/src/components/network/port-write/constants.ts
    - xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx
    - xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx
    - xingran-react-frontend/src/pages/network/ports/index.tsx
    - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx
decisions: []
metrics:
  duration: ~14 min
  completed_date: 2026-07-08
---

# Phase 55 Plan 01: 前端 Phase 53 leftover tech-debt sweep

## One-liner

Cleaned 4 frontend tech-debt items from Phase 53 code review: WR-02 reason validator signature fix (extracted shared helpers to constants.ts and fixed antd cross-field validator convention across PortWriteModal + BulkWriteDrawer), IN-01 `instanceof Error` narrowing in ports/index.tsx, IN-02 eslint-disable placement on mount-only useEffect, and HealthCard.test.tsx empty-state assertion regex match.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | WR-02 extract reason validator helpers to constants.ts | 640e2cf0 | port-write/constants.ts |
| 2 | WR-02 PortWriteModal imports shared helpers | 23b3ddae | port-write/PortWriteModal.tsx |
| 3 | WR-02 BulkWriteDrawer aligns reason validation | 879fc768 | port-write/BulkWriteDrawer.tsx |
| 4 | IN-01 ports/index.tsx instanceof Error narrowing | b6b95c0d | pages/network/ports/index.tsx |
| 5 | IN-02 ports/index.tsx eslint-disable (placement fix in follow-up commit adce5799) | c17da02e + adce5799 | pages/network/ports/index.tsx |
| 6 | HealthCard.test.tsx empty-state regex match | c1f1991d | reconciliation/__tests__/HealthCard.test.tsx |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] IN-02 eslint-disable directive initial placement was wrong**

- **Found during:** Task 5 verification (after commit c17da02e)
- **Issue:** The plan placed `// eslint-disable-next-line react-hooks/exhaustive-deps` immediately above the `useEffect(() => {` opening line. However, the `react-hooks/exhaustive-deps` rule fires on the line containing the dependency array `}, []);` — not the opening line. The directive placement was a no-op: ESLint reported BOTH "Unused eslint-disable directive" warning AND the original `react-hooks/exhaustive-deps` error.
- **Fix:** Moved the directive to immediately precede `}, []);` (matching the codebase precedent in App.tsx:36, MatchTestPanel.tsx:73, templates/index.tsx:262, exceptions/index.tsx:171). Verified by re-running eslint on the file: no more `exhaustive-deps` warnings on ports/index.tsx.
- **Files modified:** xingran-react-frontend/src/pages/network/ports/index.tsx
- **Commits:** c17da02e (initial, with wrong placement) + adce5799 (corrected placement)

## Verification Results

### type-check
- `npm run type-check` exit 0 (no TS errors)

### test (full suite)
- `npx vitest run`: 64/65 tests pass, 9/10 test files pass
- HealthCard.test.tsx: my targeted fix works (`renders empty state when total=0` now passes)
- 1 remaining failure is PRE-EXISTING and OUT OF SCOPE:
  - Test "renders 5 KPIs + score + trend when data is loaded" at line 101 expects `对账健康度` Card title text
  - HealthCard.tsx removed the Card title in a /gsd-fast refactor on 2026-06-30 (single-row compact version no longer renders a Card title)
  - Per CLAUDE.md Scope Constrainment, this pre-existing test bug is NOT modified by this plan

### lint (only modified files)
- `npx eslint` on the 5 modified files: no new `react-hooks/exhaustive-deps` warnings introduced
- 15 pre-existing lint issues remain (lines 40/116/117/523 in ports/index.tsx; lines 42/115/179/328/371/475 in BulkWriteDrawer.tsx; etc.) — these existed before this plan and are out of scope

## Known Stubs

None introduced by this plan.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries introduced.

## Deferred Items

- **HealthCard.test.tsx line 101 pre-existing failure** (`对账健康度` Card title) — independent of this plan's WR-02/IN-01/IN-02/HealthCard-empty-state work. Caused by /gsd-fast 2026-06-30 single-row compact refactor that removed the Card title without updating this test assertion. Not in this phase's scope; recommend follow-up phase or quick task.
- **HealthCard.test.tsx import 80s performance issue** — per CONTEXT.md D-06, deferred to future performance/dependency-graph phase.
- **WR-03 / WR-04** (53-REVIEW recorded but not in this phase's locked 5-item scope) — deferred.
- **CR-02 backend fallback defense** — covered by 55-02 plan (separate Go file scope).
- **Other pre-existing `react-hooks/exhaustive-deps` errors** in ports/index.tsx lines 40/116/117 etc. — out of scope per CLAUDE.md Scope Constrainment.

## Auth Gates

None — frontend changes only, no auth or backend dependencies.

## Self-Check

All 6 task commits verified in git log:
- 640e2cf0 refactor(55-01): extract reason validator helpers to constants.ts (WR-02)
- 23b3ddae refactor(55-01): PortWriteModal imports shared reason helpers (WR-02)
- 879fc768 refactor(55-01): BulkWriteDrawer aligns reason validation with PortWriteModal (WR-02)
- b6b95c0d fix(55-01): ports/index.tsx handleBatchExport uses instanceof Error (IN-01)
- c17da02e fix(55-01): ports/index.tsx mount-only useEffect gets eslint-disable (IN-02) — initial placement
- adce5799 fix(55-01): ports/index.tsx mount-only useEffect gets eslint-disable (IN-02) — corrected placement
- c1f1991d fix(55-01): HealthCard.test.tsx empty-state assertion uses regex match

All 5 modified files exist on disk and contain the expected changes.

## Self-Check: PASSED
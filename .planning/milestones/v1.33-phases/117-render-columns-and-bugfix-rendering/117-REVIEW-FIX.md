---
phase: 117
fixed_at: 2026-09-14T15:05:00+08:00
review_path: .planning/phases/117-render-columns-and-bugfix-rendering/117-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 2
skipped: 1
status: partial
---

# Phase 117: Code Review Fix Report

**Fixed at:** 2026-09-14T15:05:00+08:00
**Source review:** .planning/phases/117-render-columns-and-bugfix-rendering/117-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 3 (WR-01, WR-02, + orchestrator-directed IN-02/TS2304 Phase 116 carry-over)
- Fixed: 2
- Skipped: 1 (WR-01 — verified NOT to reproduce against current code)

## Fixed Issues

### WR-02: dedicated-lines `monthlyFee` ternary `v ?` hides `0` value

**Files modified:** `xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx`
**Commit:** 07763f1
**Applied fix:** Table column render at line 377 changed from `render: (v) => (v ? `¥${v}` : "-")` to `render: (v) => (v != null ? `¥${v}` : "-")`, matching the card-view guard already present at line 494. Confirmed real bug: Go model `internal/models/operations/dedicated_line.go:42` declares `MonthlyFee float64` with `json:"monthlyFee"` (no omitempty), so `monthlyFee: 0` is always serialized and previously rendered `-` instead of `¥0`.

### IN-02 (Phase 116 carry-over): DashboardStore TS2304 `clearWidgetCache`

**Files modified:** `xingran-react-frontend/src/store/dashboardStore.ts`
**Commit:** 29b611c
**Applied fix:** `updateDashboard` (line 214) called a module-scope `clearWidgetCache()` that did not exist — only the store action did. Added the missing module-level helper `export function clearWidgetCache(widgetId?: string): void` (next to `widgetDataCache` / `getWidgetDataCache`) and delegated the store action to it (single source of truth, action API unchanged). This was worse than a type error: it was a runtime `ReferenceError` crashing `updateDashboard`.

## Skipped Issues

### WR-01: FloorCardView `area` falsy check hides `0` value

**File:** `xingran-react-frontend/src/pages/operations/floors/components/FloorCardView.tsx:82`
**Reason:** Verified NOT to reproduce — code context differs from the review's description. The reviewer described a "subsequent `&& floor.area` short-circuit" that hides `area=0`, but no such expression exists in the current code: the guard is `{floor.area != null && (<div>...{floor.area}m²</div>)}`, and `0 != null` is `true`, so the div renders and shows `0m²`. Empirical proof: the BUGFIX-01 regression test (`floors/__tests__/components.render.test.tsx:109`, a real DOM assertion via `findByText("0m²")` with `area: 0` — not a source-pattern test as WR-01 claims) **passes pre-fix, unchanged**. Additionally, `floor.go:23` declares `Area *float64` with `json:"area,omitempty"`, so `null` never reaches the frontend (nil is omitted → `undefined`); the suggested `floor.area !== undefined` would be behaviorally identical for `0`/`undefined` but strictly worse if an explicit `null` ever appeared (it would render `面积：m²` instead of hiding the row). `!= null` is retained as the safer, already-correct guard. Applying a behavior-neutral edit based on a disproven premise would add no value, so the finding is skipped rather than blindly applied.

**Original issue:** "The condition `floor.area != null &&` evaluates to `true` for `area=0` ... BUT the subsequent `&& floor.area` short-circuits to falsy when `area=0`, so the div is never rendered."

## Verification

- `npx tsc --noEmit -p tsconfig.app.json`: 0 errors project-wide, 0 TS2304 (was failing on `dashboardStore.ts`)
- `npx vitest run` on `dashboardStore.test.ts` + `floors/__tests__/components.render.test.tsx` + `dedicated-lines/__tests__/index.render.test.tsx`: **42/42 passed** (dashboardStore was 36/38 pre-fix — the 2 `updateDashboard`/`saveCurrentDashboard` tests crashing with `ReferenceError: clearWidgetCache is not defined` now pass; floors `area=0` renders `0m²` green)
- `npx eslint` on both modified files: 0 errors (1 pre-existing unrelated warning at `dedicated-lines/index.tsx:194`, `'_' unused`, untouched by this fix)
- lint-staged pre-commit hooks (eslint --fix / prettier / type-check) passed on both commits

## Notes

- Commit message adjusted from the suggested `WR-01/02` form to `WR-02`-only since WR-01 was skipped as a non-bug.
- IN-01 (test-quality) and IN-03/04/05 are Info-tier, out of the critical_warning fix scope; note the floors test IN-01 criticizes as "source-pattern" is in fact already a DOM assertion.
- dashboardStore suite is 38 tests (35 was the Phase 116 count; suite grew), all passing.

---

_Fixed: 2026-09-14T15:05:00+08:00_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_

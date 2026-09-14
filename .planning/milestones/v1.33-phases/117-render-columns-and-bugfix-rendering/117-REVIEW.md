# Phase 117: Code Review Report

**Reviewed:** 2026-09-14T10:30:00Z
**Depth:** standard
**Files Reviewed:** 16
**Status:** issues_found

## Summary

Phase 117 applies `useMemo` to table columns (RENDER-01, 7 files), adds `virtual` + `scroll.y` to Tables (RENDER-02, 5 files), optimizes MACHeatmapChart with hoisted useMemo (RENDER-03), and fixes three rendering bugs (BUGFIX-01/02/03).

**RENDER correctness:** All columns useMemo deps are correct. Table virtual + scroll.y is correctly applied (pagination/sort/filter preserved via `onChange`).

**BUGFIX-03 (stale closure) is correctly fixed** in all 3 locations (new workstation creation, text drawing, batch drag) -- `prev.snapToGrid` / `prev.gridSize` used inside `setFloorPlanData` updater.

**Two active rendering bugs remain** (neither BUGFIX-01 regression test actually tests DOM output), plus one test-quality issue.

---

## Critical Issues

*None.*

---

## Warnings

### WR-01: FloorCardView `area` falsy check hides `0` value (BUGFIX-01 regression test is a false positive)

**File:** `xingran-react-frontend/src/pages/operations/floors/components/FloorCardView.tsx:82`
**Issue:** The condition `floor.area != null &&` evaluates to `true` for `area=0` (since `0 != null` is `true` in JS), BUT the subsequent `&& floor.area` short-circuits to falsy when `area=0`, so the div is never rendered. The BUGFIX-01 regression test (`index.render.test.tsx:109`) passes only because it tests the source code pattern (`!= null`) rather than the actual DOM output -- a source-pattern test cannot distinguish between "condition passes but value is falsy" and "condition passes and value is truthy".

**Failure scenario:** A floor record with `area=0` (legitimate -- zero-area floor or import default) renders nothing instead of `0m²`.

**Fix:**
```tsx
// BEFORE (line 82):
{floor.area != null && (
  <div>
    <strong>面积：</strong>
    {floor.area}m²
  </div>
)}

// AFTER:
{floor.area !== undefined && (
  <div>
    <strong>面积：</strong>
    {floor.area}m²
  </div>
)}
```

**Regression test quality issue:** The test at `components/FloorCardView.test.tsx:109` should assert the DOM (`findByText("0m²")`) rather than relying on source-pattern matching. However, since the test already uses `findByText` and the bug is in the component (not the test logic), the component fix is the primary action.

---

### WR-02: dedicated-lines `monthlyFee` ternary `v ?` hides `0` value (BUGFIX-01 regression test covers a different line)

**File:** `xingran-react-frontend/src/pages/operations/dedicated-lines/index.tsx:377`
**Issue:** The `monthlyFee` render uses `v ?` which is falsy for `0`. Line 494 correctly uses `!= null` guard for the card view, but the table column at line 377 uses `v ? "¥${v}" : "-"` which renders `0` as `"-"`.

The BUGFIX-01 regression test (`index.render.test.tsx:79-85`) asserts `src).toMatch(/monthlyFee != null/)` which matches the card-view line 494 (correct guard) but does NOT cover the broken table-column line 377. The test passes because line 494 has the pattern, even though line 377 still has the bug.

**Failure scenario:** A dedicated line with `monthlyFee=0` (free line, promotional rate) shows `-` in the table instead of `¥0`.

**Fix:**
```tsx
// BEFORE (line 377):
render: (v) => (v ? `¥${v}` : "-"),

// AFTER:
render: (v) => (v != null ? `¥${v}` : "-"),
```

**Regression test quality issue:** The test should assert on the table column's render function specifically (e.g., by checking the column definition's `render` function output for `v=0`) or add a separate test for the table column path. The current test only verifies one of the two render sites has the correct pattern.

---

## Info

### IN-01: Regression tests are source-pattern assertions, not DOM assertions (both BUGFIX-01 tests)

**Files:**
- `xingran-react-frontend/src/pages/operations/dedicated-lines/__tests__/index.render.test.tsx:79-85`
- `xingran-react-frontend/src/pages/operations/floors/__tests__/components.render.test.tsx:109-119`

**Issue:** Both tests verify the source code contains `!= null` rather than asserting the DOM renders the expected output. Source-pattern tests cannot distinguish:
- "condition passes but value is falsy" vs "condition passes and value is truthy"
- "one render site is correct while another is broken"

**Fix:** The dedicated-lines test should add a render assertion:
```tsx
it("monthlyFee=0 renders ¥0 in table", async () => {
  // render with monthlyFee: 0 and verify DOM output
  // ...
});
```
The floors test already uses `findByText("0m²")` correctly but the component bug prevents it from finding the text. Once WR-01 is fixed the test should green-pass naturally.

---

### IN-02: DashboardStore TS2304 `clearWidgetCache` pre-existing from Phase 116

**File:** `xingran-react-frontend/src/store/dashboardStore.ts`
**Issue:** TS2304 error on `clearWidgetCache` was identified in Phase 116 review fix (WR-02). This is a Phase 116 scope issue that was not resolved before Phase 117 started. Not a Phase 117 bug.

**Verdict:** Out of scope for Phase 117. Noted as pre-existing.

---

### IN-03: RENDER-01/RENDER-02 correctness confirmed

All 7 RENDER-01 files correctly use `useMemo` for column definitions with complete dependency arrays:

| File | useMemo deps |
|------|-------------|
| `monitor/job/index.tsx:95-105` | `[handleToggleStatus, handleExecute, handleViewLogs, openModal, handleDelete, form]` |
| `monitor/logs/index.tsx:203-218` | `[handleViewDetail, operLogManager.getColumnSortOrder]` / `[handleViewDetail, loginLogManager.getColumnSortOrder]` |
| `system/dict/index.tsx:463-487` | `[openTypeModal, handleDeleteType, getTypeColumnSortOrder, loadDictData, setSelectedType, setActiveTab, setCurrent, typeForm]` (complete) |
| `network/executions/modals/DetailDrawer.tsx:26-29` | `[handleViewOutput]` |
| `network/templates/modals/VariablesModal.tsx:32-42` | `[]` (static columns) |
| `operations/workstations/LocationAliasDrawer.tsx:158-206` | `[canDelete, handleDelete]` |
| `monitor/logs/index.tsx:203-218` | oper/log column deps correct |

All 5 RENDER-02 files correctly add `virtual` + `scroll={{ x, y }}` to Table without breaking pagination/sort/filter (all use `onChange={handleTableChange}` or equivalent).

---

### IN-04: BUGFIX-03 (stale closure) correctly fixed in all 3 locations

All three locations in `CADFloorPlanEditor.tsx` now read `prev.snapToGrid` / `prev.gridSize` inside `setFloorPlanData` updater:

1. **New workstation creation** (line 721-752): `setFloorPlanData((prev) => { ... prev.snapToGrid ... prev.gridSize ... })`
2. **Text drawing snap** (line 769-778): `setFloorPlanData((prev) => { ... prev.snapToGrid ... prev.gridSize ... })`
3. **Batch drag** (line 964-981): `newData.workstations = prev.workstations.map(...) { if (prev.snapToGrid) { ... prev.gridSize ... } }`

The `handleCanvasMouseMove` deps correctly exclude `floorPlanData` (line 1018-1032).

---

### IN-05: MACHeatmapChart rules-of-hooks verified correct

**File:** `xingran-react-frontend/src/components/network/MACHeatmapChart.tsx`

Both `useMemo` calls (`desktopOption` at line 41 and `topPorts` at line 105) are placed before the early `return` statements (lines 114 and 122), satisfying React's rules-of-hooks. The `isMobile` branch (line 126) correctly renders the mobile Top-20 list without calling `desktopOption`.

---

## Pre-Existing (Not Phase 117)

- `dashboardStore.ts`: TS2304 `clearWidgetCache` -- Phase 116 WR-02 carry-over, not in Phase 117 scope

---

_Reviewed: 2026-09-14T10:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

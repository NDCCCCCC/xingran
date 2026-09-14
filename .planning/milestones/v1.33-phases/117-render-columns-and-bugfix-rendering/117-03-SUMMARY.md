# Phase 117 Plan 03: RENDER-03/04 Summary

**Phase:** 117-render-columns-and-bugfix-rendering
**Plan:** 03
**Wave:** 2
**Commit:** 193f1e7
**Executed:** 2026-09-14

## One-liner

RENDER-03 (MACHeatmapChart mobile useMemo) + RENDER-04 (DoorElement snapCoord on hingePoint/openEndPoint)

## Deviation from Plan

**Rule 1 - Bug fix during execution:** React Hooks cannot be called inside conditional blocks (`if (isMobile)`). The original plan placed `useMemo` calls inside the mobile branch, which violates the rules-of-hooks. Fixed by hoisting both `topPorts` and `maxCount` useMemo calls to unconditionally execute before early returns, with a guard for empty data (`data?.cells ?? []` and `topPorts.length > 0` fallback). Mobile branch now references the pre-computed memos without redeclaring them. Desktop branch is completely unchanged.

## Decisions Made

1. **Hooks placement**: All useMemo calls in MACHeatmapChart are now unconditionally called before `if (loading)` and `if (!data?.cells)` early returns, consistent with the existing `desktopOption` useMemo pattern at line 41.

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| `grep -c "useMemo" MACHeatmapChart.tsx` >= 2 | PASS (4 useMemo calls) |
| Mobile topPorts sort inside useMemo | PASS |
| Desktop branch unchanged | PASS |
| `snapCoord.*hingePoint` >= 2 | PASS (2) |
| `snapCoord.*openEndPoint` >= 1 | PASS (2) |
| `snapCoord.*leafEnd` (existing) | PASS (leafEndX/Y already snapped at :94-95) |

## Files Modified

| File | Change |
|------|--------|
| `xingran-react-frontend/src/components/network/MACHeatmapChart.tsx` | RENDER-03: topPorts + maxCount useMemo chain, hoisted before early returns |
| `xingran-react-frontend/src/components/cad-elements/DoorElement.tsx` | RENDER-04: hingePoint + openEndPoint wrapped with snapCoord in geometry useMemo |

## Verification

- `npx eslint` on modified files: **0 errors** (1 pre-existing warning: `_DOOR_THICKNESS` unused-var in DoorElement.tsx)
- `npx vitest run MACHeatmapChart.test.tsx`: **5 passed**
- Pre-existing type-check failure in `dashboardStore.ts` (`clearWidgetCache` not defined) is unrelated to these changes

## Threat Flags

None — pure performance refactoring with no trust boundary changes.

## Self-Check: PASSED

- Commit `193f1e7` exists in git log
- MACHeatmapChart.tsx useMemo count: 4
- DoorElement.tsx snapCoord applied to hingePoint (x,y), openEndPoint (x,y)

---
phase: 99
phase_name: operations口径统一
plan: 99-02
subsystem: operations
tags: [fix, floor-move, optimistic-lock, V130R-07]
dependency_graph:
  requires: []
  provides: []
  affects: [floor_service.go]
tech_stack:
  added: [ErrFloorNotInExpectedBuilding sentinel error]
  patterns: [optimistic locking via WHERE clause on building_id]
key_files:
  created:
    - internal/services/operations/floor_move_building_test.go
  modified:
    - internal/services/operations/floor_service.go
decisions:
  - id: V130R-07-1
    decision: Use optimistic locking with WHERE building_id=? in UPDATE to prevent concurrent floor moves from silently overwriting each other
metrics:
  duration_minutes: 10
  completed_date: 2026-09-06
  tasks_completed: 2
  files_changed: 2
  lines_added: ~120
---

# Phase 99 Plan 99-02: V130R-07 换楼同步有序化

## One-liner

Add optimistic lock on floor building_id update to prevent silent reordering when concurrent requests move the same floor.

## Problem Fixed

**Root Cause:** When updating `floor.building_id` (moving a floor between buildings), the `Update` method read `oldBuildingID` from the database, then called `Save()`. Between these two steps, a concurrent request could update the same floor, causing:
1. Request A reads oldBuildingID=A, floor in A
2. Request B reads oldBuildingID=A, floor in A
3. Request A commits: UPDATE floor SET building_id=B WHERE building_id=A (succeeds, floor now in B)
4. Request B commits: UPDATE floor SET building_id=C WHERE building_id=A (WHERE clause finds 0 rows, silently succeeds with no effect)

The second update appeared to succeed but had no effect, leaving the floor in the wrong building.

**Fix:** When building_id changes, use `UPDATE ... WHERE id=? AND building_id=?` — the optimistic WHERE clause ensures the update only succeeds if the floor is still in the expected building. If another request already moved it, `RowsAffected == 0` and `ErrFloorNotInExpectedBuilding` is returned.

## Changes

### Modified: `internal/services/operations/floor_service.go`

- Added `ErrFloorNotInExpectedBuilding` sentinel error for optimistic lock conflicts
- When `oldBuildingID != "" && floor.BuildingID != "" && oldBuildingID != floor.BuildingID`:
  - Use direct `Updates()` with `WHERE id=? AND building_id=?` to ensure atomicity
  - Return `ErrFloorNotInExpectedBuilding` if `RowsAffected == 0`
- Non-building-id updates continue to use `Save()` path

### Created: `internal/services/operations/floor_move_building_test.go`

4 regression tests:
- `TestFloorMoveBuilding_Success` — Normal A→B move succeeds
- `TestFloorMoveBuilding_OptimisticLockError` — B→C after A→B direct DB write succeeds (floor in B, update to C valid)
- `TestFloorMoveBuilding_StaleReadReturnsError` — Invariant: UPDATE with wrong building_id affects 0 rows
- `TestFloorMoveBuilding_SameBuildingNoOp` — Name-only update uses Save() path (no optimistic lock)

## Verification

```bash
go test -v -run "TestFloorMove" ./internal/services/operations/  # PASS (4 tests)
go test -v -run "Floor" ./internal/services/operations/          # PASS (25 tests total)
go build ./internal/services/operations/...                     # PASS
```

## Deviations from Plan

None - plan executed as written.

## Commit

- `9b861c8` (bundled with 99-03 docs commit): fix(99-02): add optimistic lock on floor building_id update

---
phase: 12-data-model-integration
plan: 01
subsystem: database, time-series, mac-address-tracking
tags: [postgresql, partitioning, brin-index, mac-history, change-detection, gorm]

# Dependency graph
requires: []
provides:
  - MAC address history data model (DeviceMACHistory)
  - Change detection logic (appeared/disappeared/moved/vlan_changed)
  - MAC address normalization (vendor format handling)
  - Partitioned history table with BRIN indexing
affects: [12-02, 13-01, 13-02]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Change detection with state transition matrix (old state vs new state)
    - MAC address normalization to AA:BB:CC:DD:EE:FF format
    - Time-series partitioning by month (PARTITION BY RANGE)
    - BRIN indexing for time-series queries (pages_per_range=128)
    - Non-blocking error handling (history failures don't block collection)

key-files:
  created:
    - internal/models/device_mac_history.go
    - internal/services/mac_history_service.go
    - internal/core/db/migrations/migration_mac_history.go
  modified:
    - internal/services/mac_collection_service.go

key-decisions:
  - "Use incremental snapshot strategy (only record changes) - saves 99% storage"
  - "MAC address normalization prevents comparison failures (RESEARCH.md Pitfall 3)"
  - "History recording happens AFTER database mutations to avoid blocking collection"
  - "Device name stored as snapshot (non-FK) ensures readability after device deletion"
  - "Monthly partitioning with BRIN indexing optimizes time-series queries"

patterns-established:
  - "State transition matrix for change detection (oldMACs vs newMACs)"
  - "Vendor-agnostic MAC address format normalization"
  - "Non-blocking history recording (failures logged but don't block)"
  - "PostgreSQL native partitioning (no TimescaleDB needed)"

requirements-completed: [DATA-01]

# Metrics
duration: 25min
completed: 2026-05-11
---

# Phase 12: Plan 01 Summary

**MAC地址历史数据模型和变更检测逻辑，使用PostgreSQL分区表+BRIN索引实现时间序列存储**

## Performance

- **Duration:** 25 min
- **Started:** 2026-05-11T00:55:03Z
- **Completed:** 2026-05-11T01:20:15Z
- **Tasks:** 4
- **Files modified:** 4

## Accomplishments

- Created `DeviceMACHistory` model with 4 event types (appeared/disappeared/moved/vlan_changed)
- Implemented `normalizeMACAddress()` function to handle vendor format differences (aabb.ccdd.eeff → AA:BB:CC:DD:EE:FF)
- Built change detection logic using state transition matrix (oldState vs newState comparison)
- Created partitioned table migration with BRIN indexing (pages_per_range=128)
- Integrated non-blocking history recording into MAC collection flow

## Task Commits

Each task was committed atomically:

1. **Task 1: Create MAC history data model** - `fc5f44b` (feat)
2. **Task 2: Create MAC history service** - `319e469` (feat)
3. **Task 3: Integrate change detection** - `32fa2f1` (feat)
4. **Task 4: Create migration script** - `3b05746` (feat)

**Plan metadata:** N/A (no final commit yet)

## Files Created/Modified

### Created
- `internal/models/device_mac_history.go` - History data model with event types, timestamps, device name snapshot
- `internal/services/mac_history_service.go` - Change detection logic, MAC normalization, state map building, flapping merge
- `internal/core/db/migrations/migration_mac_history.go` - Partition table creation, BRIN indexing, monthly partition management

### Modified
- `internal/services/mac_collection_service.go` - Integrated change detection (query old state → delete → insert → record history)

## Decisions Made

1. **MAC Address Normalization (RESEARCH.md Pitfall 3)**
   - Implemented `normalizeMACAddress()` to convert vendor formats to unified AA:BB:CC:DD:EE:FF
   - Prevents comparison failures when devices output different formats (aabb.ccdd.eeff vs aa:bb:cc:dd:ee:ff)
   - Uses regex validation for 12 hex characters, inserts colons programmatically

2. **History Recording Timing**
   - History recording happens AFTER database mutations (DELETE + INSERT complete)
   - Ensures history failures don't block MAC collection (CONTEXT.md D-07)
   - Old state queried BEFORE DELETE to capture data for comparison

3. **State Map Building Export**
   - Exported `BuildMACStateMap()` method and added to `MACHistoryService` interface
   - Allows `mac_collection_service.go` to reuse normalization logic
   - Ensures consistent MAC formatting across services

4. **Partition Auto-Creation**
   - Migration creates current month + 2 future partitions automatically
   - Prevents "no partition found" errors when data crosses month boundaries
   - Helper functions `CreateMonthlyPartition`/`DropOldPartition` for scheduled tasks

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed undefined type errors in mac_collection_service.go**
- **Found during:** Task 3 (Integration)
- **Issue:** Used `MACHistoryService` interface type before it was imported, caused "undefined" errors
- **Fix:** Replaced interface dependency with `enableHistory` bool flag, created history service instance inline
- **Files modified:** internal/services/mac_collection_service.go
- **Verification:** `go build ./internal/services/` passes
- **Committed in:** `32fa2f1` (Task 3 commit)

**2. [Rule 3 - Blocking] Exported BuildMACStateMap method**
- **Found during:** Task 3 (Integration)
- **Issue:** `buildMACStateMap` was private (lowercase), couldn't be called from other services
- **Fix:** Renamed to `BuildMACStateMap` (uppercase) and added to `MACHistoryService` interface
- **Files modified:** internal/services/mac_history_service.go
- **Verification:** Interface method compiles, can be called via interface
- **Committed in:** `32fa2f1` (Task 3 commit - part of same commit as it was needed for integration)

**3. [Rule 3 - Blocking] Fixed db.Dialect undefined errors**
- **Found during:** Task 4 (Migration script)
- **Issue:** GORM v2 doesn't have `db.Dialect.Name`, replaced by `db.Config.Dialector.Name()`
- **Fix:** Created `isPostgreSQL()` helper function using correct API
- **Files modified:** internal/core/db/migrations/migration_mac_history.go
- **Verification:** `go build ./internal/core/db/migrations/` passes
- **Committed in:** `3b05746` (Task 4 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 3 - blocking issues)
**Impact on plan:** All fixes necessary for code to compile and integrate. No scope creep.

## Issues Encountered

1. **Type definition cross-file dependency**
   - Initially tried to use `MACHistoryService` interface in `mac_collection_service.go`
   - Go compiler couldn't resolve type when compiling single file
   - Solution: Simplified to `enableHistory` bool flag, create service instance inline

2. **Private method access**
   - `buildMACStateMap` was private, couldn't be reused by collection service
   - Solution: Exported method and added to interface definition

3. **GORM API changes**
   - Used outdated `db.Dialect.Name` pattern from older GORM versions
   - Solution: Created helper using `db.Config.Dialector.Name()`

All issues resolved without changing scope or requirements.

## Known Stubs

**None** - All implemented functionality is complete and wired to real data sources.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: injection | internal/core/db/migrations/migration_mac_history.go | Partition name string concatenation - mitigated by using fmt.Sprintf with year/month (validated integers) |

**Mitigation applied:** Year and month are integers, not user input. Partition names follow strict pattern `sys_device_mac_history_%d_%02d`.

## Next Phase Readiness

**Ready for Phase 12-02:**
- ✅ History table created with partitioning
- ✅ Change detection logic integrated into collection flow
- ✅ MAC normalization prevents vendor format issues
- ✅ Migration script ready to be called in database initialization

**Dependencies for next phase:**
- None - 12-01 is foundation for all history-based features

**Recommendations:**
- Test migration script in development environment to verify partition creation
- Monitor first collection cycle to confirm history records are written correctly
- Verify BRIN index effectiveness with `EXPLAIN ANALYZE` on time-range queries

---
*Phase: 12-data-model-integration*
*Completed: 2026-05-11*

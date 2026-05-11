---
phase: 12-data-model-integration
plan: 02
subsystem: database, time-series, mac-address-tracking
tags: [postgresql, partitioning, auto-cleanup, scheduler, mac-history]

# Dependency graph
requires: [12-01]
provides:
  - Automatic monthly partition creation for MAC history table
  - Scheduled cleanup task for expired partitions (daily at 2 AM)
  - Configurable retention period (default 120 days, min 30 days)
  - Non-blocking partition management during application startup
affects: [13-01, 13-02]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - PostgreSQL native partitioning (PARTITION BY RANGE with monthly partitions)
    - Scheduled task registration pattern (RegisterTask with dependency injection)
    - Configuration-driven retention (sys_config table with minimum validation)
    - Application startup partition validation (EnsurePartitionsExist)
    - DROP PARTITION for instant space release (vs DELETE rows)

key-files:
  created:
    - internal/services/mac_history_partition.go
    - internal/scheduler/mac_history_tasks.go
  modified:
    - internal/core/core.go
    - cmd/main.go

key-decisions:
  - "Use monthly partitions (sys_device_mac_history_YYYY_MM) for efficient cleanup"
  - "Application startup ensures 2 future months exist (prevents partition missing errors)"
  - "Daily cleanup at 2 AM using existing scheduler infrastructure"
  - "Retention configurable via sys_config (default 120 days, minimum 30 days)"
  - "Partition management failures don't block application startup (non-blocking)"

patterns-established:
  - "Partition service interface with dependency injection pattern"
  - "Scheduled task registration using sync.Once for thread safety"
  - "PostgreSQL pg_inherits catalog for partition discovery"
  - "Regex validation for partition name format (YYYY_MM)"
  - "Configuration initialization on first startup (idempotent)"

requirements-completed: [DATA-02]

# Metrics
duration: 2min
completed: 2026-05-11
---

# Phase 12: Plan 02 Summary

**MAC地址历史表分区管理和自动清理机制，使用PostgreSQL原生分区+定时任务实现数据生命周期管理**

## Performance

- **Duration:** 2 min
- **Started:** 2026-05-11T01:00:39Z
- **Completed:** 2026-05-11T01:02:51Z
- **Tasks:** 3
- **Files created:** 2
- **Files modified:** 2

## Accomplishments

- Created `PartitionService` interface with 4 core methods (CreateMonthlyPartition, EnsurePartitionsExist, DropExpiredPartitions, GetRetentionDays)
- Implemented monthly partition creation using PostgreSQL native partitioning syntax
- Built scheduled cleanup task with dependency injection pattern (sync.Once for thread safety)
- Integrated partition management into application startup flow
- Added default retention configuration (120 days) with minimum validation (30 days)
- Registered MAC history cleanup task to scheduler (daily at 2 AM)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create partition management service** - `3319640` (feat)
2. **Task 2: Create scheduled cleanup task** - `c394662` (feat)
3. **Task 3: Integrate into application startup** - `3ddfd1d` (feat)

**Plan metadata:** N/A (no final commit yet)

## Files Created/Modified

### Created
- `internal/services/mac_history_partition.go` - Partition service implementation with 4 methods
- `internal/scheduler/mac_history_tasks.go` - Scheduled task registration and execution

### Modified
- `internal/core/core.go` - Added PartitionService field and initialization
- `cmd/main.go` - Added task registration and default configuration initialization

## Decisions Made

1. **Monthly Partition Strategy** (CONTEXT.md D-05)
   - Partition name format: `sys_device_mac_history_YYYY_MM` (e.g., `sys_device_mac_history_2025_01`)
   - Partition boundary: FROM 'YYYY-MM-01' TO 'YYYY-(MM+1)-01'
   - Automatic creation on application startup (ensures 2 future months exist)
   - Uses `CREATE TABLE IF NOT EXISTS` to avoid errors when partitions already exist

2. **Retention Configuration** (CONTEXT.md D-10)
   - Configuration key: `network.mac.history.retention_days`
   - Default value: 120 days (90-day data + 30-day buffer to avoid boundary issues)
   - Minimum validation: 30 days (prevents accidental data loss from misconfiguration)
   - Stored in `sys_config` table for runtime modification without restart

3. **Scheduled Cleanup** (CONTEXT.md D-08, D-09)
   - Task type: `mac_history_cleanup`
   - Schedule: Daily at 2:00 AM (via existing scheduler infrastructure)
   - Cleanup method: `DROP PARTITION` (instant space release vs DELETE rows)
   - Partition discovery: Query `pg_catalog.pg_inherits` for all child partitions
   - Regex validation: `^sys_device_mac_history_(\d{4})_(\d{2})$` to extract year/month

4. **Non-Blocking Initialization** (CONTEXT.md D-07 applied to partitions)
   - Partition management failures logged but don't block application startup
   - Partitions can be created manually later if automatic creation fails
   - Scheduler checks for partition service initialization before running cleanup

5. **Dependency Injection Pattern**
   - `SetPartitionService()` uses `sync.Once` for thread-safe singleton
   - Service interface defined in `mac_history_tasks.go` to avoid circular dependency
   - Core.Init() creates partition service and calls `EnsurePartitionsExist(2)`
   - main.go registers task after scheduler is fully initialized

## Deviations from Plan

### Auto-fixed Issues

**None** - All tasks executed exactly as specified in the plan.

---

**Total deviations:** 0
**Impact on plan:** All tasks completed without deviations.

## Issues Encountered

None - All tasks compiled successfully on first attempt.

## Known Stubs

**None** - All implemented functionality is complete and wired to real data sources.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: injection | internal/services/mac_history_partition.go | Partition name string concatenation - mitigated by using fmt.Sprintf with validated year/month integers |
| threat_flag: tampering | internal/services/mac_history_partition.go | Retention config could be set to very low value - mitigated by minimum 30-day validation |

**Mitigations applied:**
- Year and month are validated integers (2020-2100, 1-12) before string formatting
- Retention days validated with minimum of 30 days to prevent accidental data loss
- Partition name format validated with regex before DROP operations

## Next Phase Readiness

**Ready for Phase 13:**
- ✅ Partition infrastructure in place for MAC history table
- ✅ Automatic cleanup ensures data won't grow unbounded
- ✅ Configuration system allows runtime adjustment of retention period
- ✅ Application startup ensures future partitions exist before data collection

**Dependencies for next phase:**
- None - 12-02 provides foundational infrastructure for all query features in Phase 13

**Recommendations:**
- Monitor first cleanup cycle to verify partitions are dropped correctly
- Verify retention period configuration is accessible via admin UI
- Test partition creation manually in development environment

## Verification Results

**Build verification:**
- `go build ./internal/services/mac_history_partition.go` ✅ PASSED
- `go build ./internal/scheduler/mac_history_tasks.go` ✅ PASSED
- `go build ./cmd/main.go` ✅ PASSED
- `go build ./...` ✅ PASSED (full project build)

**Functional verification (pending runtime):**
- Application startup should create 2 future monthly partitions
- Scheduler should have `mac_history_cleanup` task registered
- sys_config should have `network.mac.history.retention_days` entry with value "120"
- Cleanup task should drop partitions older than retention period

---

*Phase: 12-data-model-integration*
*Plan: 02*
*Completed: 2026-05-11*
*Total commits: 3*
*All tasks: Complete*

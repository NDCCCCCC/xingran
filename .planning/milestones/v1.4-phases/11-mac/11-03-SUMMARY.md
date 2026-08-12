---
phase: 11-mac
plan: 03
subsystem: network
tags: [mac-filter, topology, gorm, postgres, handler-service, crud-api]

# Dependency graph
requires:
  - phase: 11-mac
    plan: 01
    provides: [LLDP discovery service]
  - phase: 11-mac
    plan: 02
    provides: [MAC filtering enhancement with LLDP]
provides:
  - MAC filter rule data model with device type/vendor support
  - Database table sys_mac_filter_rules with default rules
  - Filter rule service with CRUD and priority-based resolution
  - REST API for managing filter rules
  - GetEffectiveRule endpoint for debugging rule resolution
affects: [mac-collection, network-monitoring, device-management]

# Tech tracking
tech-stack:
  added: []
  patterns: [Handler-Service-Router pattern, Priority-based rule resolution, System rule protection]

key-files:
  created:
    - internal/models/mac_filter_rule.go
    - internal/core/db/migrations/117_create_mac_filter_rules.sql
    - internal/services/topology/filter_rules.go
    - internal/api/v1/network/topology_handler.go
    - internal/api/v1/network/topology_router.go
  modified:
    - internal/api/router.go

key-decisions:
  - "Vendor empty string (not NULL) for device-type-only rules"
  - "System rules protected from deletion/update via is_system flag"
  - "Priority-based resolution: vendor+type > type > default"
  - "UNIQUE constraint on (device_type, vendor) prevents duplicate rules"
  - "Hardcoded default fallback when no DB rules match"

patterns-established:
  - "Pattern: Filter rule service with interface-based design"
  - "Pattern: Priority-based rule resolution with fallback hierarchy"
  - "Pattern: System rule protection in CRUD operations"

requirements-completed: ["MAC-05"]

# Metrics
duration: 3min
completed: 2026-05-09T02:23:00Z
---

# Phase 11-03: Configurable MAC Filter Rules Summary

**Database-backed MAC filter rule management with priority-based resolution (vendor+type → type → default), system rule protection, and REST API for CRUD operations**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-09T02:20:29Z
- **Completed:** 2026-05-09T02:23:00Z
- **Tasks:** 5
- **Files modified:** 5

## Accomplishments

- **MAC filter rule data model** with device type, vendor, threshold, LLDP filter, and priority fields
- **Database migration** creating sys_mac_filter_rules table with 5 default rules (switch, router, firewall, loadbalancer, ap)
- **Filter rule service** with full CRUD operations, duplicate detection, and system rule protection
- **Priority-based rule resolution** (GetEffectiveRule) supporting vendor-specific → device-type → default fallback
- **REST API** with 6 endpoints for managing filter rules and debugging effective rule per device

## Task Commits

Each task was committed atomically:

1. **Task 1: Create MAC filter rule data model** - `23f83ac` (feat)
2. **Task 2: Create database migration for filter rules** - `9e92e32` (feat)
3. **Task 3: Create filter rule service with CRUD operations** - `4d51dd0` (feat)
4. **Task 4: Create topology API handlers for filter rule management** - `7b325c9` (feat)
5. **Task 5: Create topology API router and register routes** - `7d9697f` (feat)

**Plan metadata:** Pending final commit

## Files Created/Modified

- `internal/models/mac_filter_rule.go` - MACFilterRule model with validation and UUID generation
- `internal/core/db/migrations/117_create_mac_filter_rules.sql` - Table creation with 5 default system rules
- `internal/services/topology/filter_rules.go` - FilterRuleService interface and implementation
- `internal/api/v1/network/topology_handler.go` - TopologyHandler with 6 REST endpoints
- `internal/api/v1/network/topology_router.go` - Route registration under /api/v1/network/topology
- `internal/api/router.go` - Added topology router registration

## Decisions Made

- **Vendor representation**: Empty string (not NULL) for device-type-only rules to simplify query logic
- **System rule protection**: is_system flag prevents deletion/modification of default rules via API
- **Rule priority**: Integer field for ordering multiple matches; higher priority checked first
- **Unique constraint**: (device_type, vendor) prevents duplicate rules at same specificity level
- **Default fallback**: Hardcoded defaults (threshold=10, lldp=true) when no DB rules match

## Deviations from Plan

None - plan executed exactly as written

## Issues Encountered

None - all tasks completed without issues

## Next Phase Readiness

- Filter rule infrastructure complete and ready for integration with MAC collection service
- Next phase (11-04) should integrate GetEffectiveRule into MAC collection service
- API endpoints available for frontend rule management UI

## API Endpoints

**Base Path:** `/api/v1/network/topology/rules`

- `POST /list` - List rules with filtering (device_type, vendor) and pagination
- `POST /` - Create new rule
- `POST /:id` - Get rule by ID
- `POST /:id/update` - Update rule
- `POST /:id/delete` - Delete rule (system rules protected)
- `GET /effective?deviceId=xxx` - Get effective rule for specific device (debug endpoint)

## Default Rules

| Rule Name | Device Type | Vendor | MAC Threshold | LLDP Filter | Priority | System |
|-----------|-------------|--------|---------------|-------------|----------|--------|
| 默认交换机规则 | switch | NULL | 10 | TRUE | 0 | TRUE |
| 默认路由器规则 | router | NULL | 500 | TRUE | 0 | TRUE |
| 默认防火墙规则 | firewall | NULL | 100 | TRUE | 0 | TRUE |
| 默认负载均衡器规则 | loadbalancer | NULL | 50 | TRUE | 0 | TRUE |
| 默认无线接入点规则 | ap | NULL | 100 | TRUE | 0 | TRUE |

## Rule Resolution Logic

```
1. Most specific: vendor + device type (e.g., huawei + switch)
2. Less specific: device type only (e.g., switch + NULL vendor)
3. Default fallback: hardcoded (threshold=10, lldp=true)
```

Within each level, rules with higher priority are checked first.

---
*Phase: 11-mac*
*Completed: 2026-05-09*

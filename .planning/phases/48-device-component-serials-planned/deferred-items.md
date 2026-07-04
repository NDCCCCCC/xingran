# Phase 48 — Deferred Items (Out-of-Scope Discoveries)

Pre-existing issues found during execution that are NOT caused by Phase 48 work and are out of scope per executor Rule scope boundary. Not fixed; logged for follow-up.

## Discovered during 48-01 Task 2 (2026-07-04)

### Pre-existing operations package test failures

**Verified via `git stash` baseline test run** — these tests fail on commit `b8fd2f45` (the base of Task 2) before any Task 2 changes were applied. They are NOT regressions caused by Phase 48-01.

| Test | File | Symptom |
|------|------|---------|
| `TestValidator_ValidateFloor/存在的楼层` | `internal/services/operations/validation_helper_test.go:73` | `error code = 1500, wantErrorCode 0` (likely sqlite schema mismatch in `sys_floor` lookup) |
| `TestValidator_ValidateWall/存在的墙体` | `internal/services/operations/validation_helper_test.go:124` | same pattern |
| `TestValidator_ValidateDoor/存在的门` | `internal/services/operations/validation_helper_test.go:173` | same pattern |
| `TestReferenceResolver_ResolveSingle` | `internal/services/operations/reference_resolver_test.go:144` | expected "1" actual "" — `sys_dept` lookup returns empty (sqlite `deleted_at` schema-related) |

Suspected root cause category: sqlite in-memory fixtures missing columns that production code references via `WHERE deleted_at IS NULL` style clauses (visible in logged queries). Independent of Phase 48 schema additions.

**Action:** Out of scope. Owner: someone with context on `validation_helper_test.go` / `reference_resolver_test.go` sqlite fixture maintenance.

### Note (not a failure)

`asset_statistics_test.go` fixture was missing the new `component_type` column **after** Task 2 added the `WHERE component_type IS NULL` clause to `Statistics()`. That was an in-scope Task 2 fix (the column-add broke the existing test directly because of Task 2's change) and has been fixed inline in the Task 2 commit — it is NOT a deferred item.

## Discovered during 48-02 final verification (2026-07-04)

### Pre-existing operations + system package test failures

Logged during the plan-level verification step (`go test ./internal/services/...`). These tests do not import or reference any Phase 48-02 file (component_collector package, snmp_entity_mib.go, the 6 new TextFSM templates). The failures pre-date this plan and are tracked here for follow-up.

**Operations package:**

| Test | File | Symptom |
|------|------|---------|
| `TestBatchUpsert_Update` | `internal/services/operations/batch_upserter_test.go:132` | "Not equal: expected 0 actual 1" — sqlite fixture semantics |
| `TestBatchUpsert_Mixed` | `internal/services/operations/batch_upserter_test.go` | same family |
| `TestBatchUpsertWithCamelCaseFields` | `internal/services/operations/batch_upserter_test.go` | same family |
| `TestExtractPagination` | `internal/services/operations/pagination_helper_test.go` | pagination expectation mismatch |
| `TestClampPageSize` | `internal/services/operations/pagination_helper_test.go:172` | "clampPageSize() = 200, want 100" — clamp boundary changed elsewhere |
| `TestClampPageSizeMath` | `internal/services/operations/pagination_helper_test.go` | same family |
| `TestPageSizeConstants` | `internal/services/operations/pagination_helper_test.go` | constants drifted from test expectations |
| `TestReferenceResolver_ResolveBatch` | `internal/services/operations/reference_resolver_test.go` | sqlite `sys_dept` lookup family (already documented above) |

**System package:**

| Test | File | Symptom |
|------|------|---------|
| `TestRoleService_Create_RoleNameExists` | `internal/services/system/role_service_apperrors_test.go:27` | "Received unexpected error" — fixture/seed state |

**Verification of non-regression:** the Phase 48-02 plan only ADDS new files (`internal/services/component_collector/*`, `internal/device/snmp_entity_mib.go`) and modifies exactly one existing template (`templates/ruijie_os_show_interfaces_status.textfsm` — adds STATUS captured variable; previously discarded). None of the failing tests import component_collector or device/snmp_entity_mib, and none exercise the ruijie status template.

**Action:** Out of scope for Phase 48. Owner: maintainers of `operations` pagination/batch-upserter fixtures and `system/role_service_apperrors_test.go`.


# Root Cause Report — Reconciliation Workorder: role_id Column + "SYSTEM" Submitter Bugs

**Date:** 2026-06-30
**Bug ID:** reconciliation-workorder-role-id-not-exist
**Investigator:** GSD Debugger (autonomous)
**Mode:** Diagnose-only (no fixes applied)

---

## Symptom

The scheduled task `reconciliation:refreshView` → `createWorkorderHigh` ran on 2026-06-30 09:53:05.
All 9 attempted workorder creations failed with two cascading SQL errors:

1. **Error 1:** `column "role_id" does not exist (SQLSTATE 42703)` — fires 9 times, once per attempt.
2. **Error 2:** `invalid input syntax for type uuid: "SYSTEM" (SQLSTATE 22P02)` — the workorder INSERT writes the string literal `"SYSTEM"` into the UUID-typed `submitter_id` column.

Final outcome: `success=0 failure=9 total=9`.

---

## Root Cause

**Two independent bugs in `internal/services/asset/reconciliation_workorder.go:CreateWorkorderFromException`, both from incorrect assumptions about the schema.**

### Bug A — Query against a non-existent column

`reconciliation_workorder.go:202` issues:

```go
s.db.WithContext(ctx).
    Where("role_id = ? AND status = ?", roleID, 0).
    Order("created_at ASC").Limit(1).
    First(&user).Error
```

`sys_user` has **no `role_id` column** (verified at `internal/models/user.go:8-43`). The user→role relationship is via the join table `sys_user_role` (`user_id uuid`, `role_id uuid`), defined at `internal/models/user.go:46-50`.

This produces `SQLSTATE 42703 column "role_id" does not exist`, fired once per workorder attempt.

### Bug B — "SYSTEM" literal stuffed into UUID column

`reconciliation_workorder.go:255` passes the literal string `"SYSTEM"` as the submitter:

```go
baseSvc.Create(ctx, &workorder.CreateRequest{...}, "SYSTEM")
```

`internal/services/workorder/base.go:251` (BaseService.Create) sets `SubmitterID: submitterID` directly onto `models.WorkOrder`, where `internal/models/workorder.go:84` defines:

```go
SubmitterID string `gorm:"type:uuid;not null;index:idx_wo_submitter,priority:1"`
```

PostgreSQL rejects `"SYSTEM"` with `SQLSTATE 22P02 invalid input syntax for type uuid`.

### Why both bugs exist — wrong assumptions

- **Bug A**: code assumes `sys_user.role_id` exists (it does not); should join `sys_user_role`.
- **Bug A**: code also declares `AssigneeRoleMap map[string]int64` (line 119) — but `sys_role.id` is UUID (`models/role.go:4` + `base.go:12 type:uuid;primary_key`). The migration_171 seed (`migration_171_reconciliation_workorder_assignee_role.go:43`) writes `{"asset_owner":1,"ops_owner":2,"responsible_owner":3}` — placeholder ints, not real role IDs (acknowledged in the migration's remark that "1/2/3 是占位"). The query with `roleID = 1` would have failed even if the column existed (PG would say `invalid input syntax for type uuid: 1`).
- **Bug B**: code assumes `submitter_id` accepts free-form strings; it does not — it is `uuid;not null`. The `"SYSTEM"` literal was a "T-43-01 mitigation" for "submitter越权" but never validated against the schema. The docstring on line 148-149 even says submitterID="SYSTEM" but does not justify the type mismatch.

### Precedent: how `workorder_tasks.go` solves the same problem

`internal/scheduler/workorder_tasks.go:56-62` shows the correct pattern for cron-created workorders:

```go
// 获取 system 用户的 ID
var systemUser struct { ID string }
if err := db.Table("sys_user").Select("id").
    Where("username = ?", "system").First(&systemUser).Error; err != nil {
    return fmt.Errorf("查询系统用户失败，请确保存在 username='system' 的用户: %w", err)
}
// ... later ...
SubmitterID: systemUser.ID, // 系统创建用户ID
```

There is a dedicated `username='system'` user whose UUID is used as the submitter.

---

## Evidence

| # | File:Line | Observation |
|---|-----------|-------------|
| 1 | `internal/services/asset/reconciliation_workorder.go:202` | Bad query: `Where("role_id = ?", roleID, 0)` against `sys_user` |
| 2 | `internal/services/asset/reconciliation_workorder.go:255` | Bad submitter literal: `}, "SYSTEM")` passed to BaseService.Create |
| 3 | `internal/models/user.go:8-43` | `User` struct — no `role_id` field, only `Roles []string gorm:"-"` |
| 4 | `internal/models/user.go:46-50` | `UserRole{UserID, RoleID}` both `type:uuid;not null` (join table) |
| 5 | `internal/models/workorder.go:84` | `SubmitterID string gorm:"type:uuid;not null;index:..."` |
| 6 | `internal/services/workorder/base.go:251-265` | `Create(ctx, req, submitterID)` directly assigns `SubmitterID: submitterID` to GORM |
| 7 | `internal/services/asset/reconciliation_workorder.go:119` | `AssigneeRoleMap map[string]int64` — wrong type (should be string/UUID) |
| 8 | `internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go:43` | Seed JSON `{"asset_owner":1,...}` — placeholders, not UUIDs |
| 9 | `internal/models/role.go:4` + `internal/models/base.go:12` | `Role.BaseModel.ID` → `type:uuid;primary_key` |
| 10 | `internal/scheduler/workorder_tasks.go:60-81` | Precedent: queries `sys_user WHERE username='system'` for system submitter UUID |

---

## Proposed Fix Shape (DO NOT APPLY — diagnose only)

Two independent fixes are required.

### Fix A — Replace role→user query with join

In `reconciliation_workorder.go:202`, change the user lookup to JOIN `sys_user_role`:

```go
// Find an enabled user in the given role via sys_user_role join
var user models.User
joinErr := s.db.WithContext(ctx).
    Table("sys_user u").
    Joins("INNER JOIN sys_user_role ur ON ur.user_id = u.id").
    Where("ur.role_id = ? AND u.status = ?", roleID, 0).
    Order("u.created_at ASC").
    Limit(1).
    First(&user).Error
```

### Fix A.1 — Change AssigneeRoleMap type to UUID

`reconciliation_workorder.go:119` should be:

```go
type AssigneeRoleMap map[string]string  // was int64; sys_role.id is UUID
```

Update `migration_171` seed JSON accordingly: `{"asset_owner":"<uuid-of-asset-owner-role>",...}`. The current seed's int placeholders would still error even after Fix A (because PG would reject `roleID = 1` against a UUID column). Both fixes are required.

### Fix B — Use real system user UUID as submitter

Either of two approaches:

**Option B1 (recommended — matches workorder_tasks.go precedent):**

```go
// In CreateWorkorderFromException, before Create:
var systemUser struct{ ID string }
if err := s.db.WithContext(ctx).Table("sys_user").
    Select("id").Where("username = ?", "system").First(&systemUser).Error; err != nil {
    logrus.Errorf("[reconciliation:workorder] 查询 system 用户失败(将不带 submitter): %v", err)
    // continue with empty submitterID? — NO: submitter_id is NOT NULL. Better: skip.
    return nil, fmt.Errorf("查询 system 用户失败: %w", err)
}
// then:
baseSvc.Create(ctx, &workorder.CreateRequest{...}, systemUser.ID)
```

**Option B2 (alternative — requires schema change):**

Make `sys_workorder.submitter_id` nullable; pass empty string. More invasive — needs a migration + frontend read-path changes. Not recommended.

### Fix B.1 — Remove or update the misleading docstring

`reconciliation_workorder.go:148-149` and line 142 should be amended to reflect the real mechanism (UUID of `username='system'` user, not a literal `"SYSTEM"`).

---

## Files Affected (Diagnosis Scope)

| File | Change |
|------|--------|
| `internal/services/asset/reconciliation_workorder.go` | Fix A (query), Fix B (submitter), type change to `AssigneeRoleMap`, update docstrings (lines 105-119, 148-149, 188-214, 246-255) |
| `internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go` | Update seed JSON to real UUIDs (or add a follow-up migration to fix the seed) |
| `sys_config` row `asset.reconciliation.workorder.assignee_role_map` | Needs operator action (or migration) to replace int placeholders with real role UUIDs |

**Scope spans 2 files** (one Go service + one migration). The sys_config row is data, not code — but without operator action the seeded int values will keep producing no-match even after Fix A.

---

## Verification Plan

After fixes are applied:

1. **Unit test:** add `reconciliation_workorder_test.go` covering `CreateWorkorderFromException`:
   - Insert fixture: 1 `sys_role`, 1 `sys_user` joined via `sys_user_role`, 1 `sys_data_reconciliation` row, 1 `sys_config` row with the real role UUID.
   - Assert: workorder INSERT succeeds; `submitter_id == systemUser.ID`; `assignee_id == fixtureUser.ID`; final log shows `success=1`.
2. **Build check:** `go build ./...` must succeed.
3. **Manual smoke:** Trigger `reconciliation:createWorkorderHigh` cron manually; observe in `sys_workorder` table that 9 rows now exist with `submitter_id` = a real UUID and `assignee_id` populated for the role whose UUID is in the config.
4. **Regression:** re-run `reconciliation_workorder.go` related tests under `internal/services/asset/*_test.go`.
5. **Migration verification:** confirm `sys_config.asset.reconciliation.workorder.assignee_role_map` contains valid UUID strings after operator update.

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Existing `sys_config` JSON has int placeholders → Fix A still finds no user (different failure mode) | Update seed migration AND provide ops-runbook; or add operator alert when `roleMap` JSON unmarshals to non-UUID values |
| `username='system'` user may not exist in production | Add migration to seed it (or surface clear error at startup) — same pattern as migration_171 for other seed users |
| Fix B.1 docstring change touches public-facing comments only — low risk | Cosmetic only |
| Submitter audit: any historical workorders with `submitter_id = ''` or NULL after this change | No — current state is that the INSERT fails entirely; no orphan rows are created. Once fixed, all new rows have valid submitter_id |
| Concurrency: parallel cron calls may race to find the same role-user | Not a regression — same code path already exists for periodic workorder tasks (line 56-62) |
| Test fixtures: existing `reconciliation_*_test.go` may use mock sys_user without role_id | Verify all `reconciliation_*_test.go` files; the schema mismatch is in production code, not tests |

**Likely scope:** single fix pull request touching 2 files (service + migration). Safe to land.

---

## Open Questions

1. Is there a `username='system'` user in production `sys_user`? If not, Fix B will surface that as a hard error — need either a migration to seed it, or accept `submitter_id` to be optional (Option B2, more invasive).
2. Should the operator-driven config update be bundled into the same fix release, or handled by a follow-up ops task?
3. Should we add a startup-time validation that warns when `assignee_role_map` values are non-UUID strings? (Defensive — not strictly required for the fix.)

---

## Notes

- All assertions in this report are read-only and verified by reading the listed files. No code modifications were made.
- The bug was introduced in Phase 43 R2 (commits in `internal/services/asset/reconciliation_workorder.go`).
- The same `internal/services/asset/reconciliation_workorder.go` file also contains the WS/SysNotice publish path that uses `"SYSTEM"` strings at line 457 — this may be acceptable since NoticeService uses `CreatedByName string` (text), not UUID. Worth confirming in a separate diagnostic if NoticeService begins to expect a UUID submitter.
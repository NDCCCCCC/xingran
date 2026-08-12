---
phase: quick
plan: 260527-gra
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/models/user.go
  - internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql
autonomous: true
requirements: []
must_haves:
  truths:
    - "AD user login updates dept_id, ad_user_dn, ad_ou_dn, and ad_synced_at without SQL errors"
    - "GORM AutoMigrate creates the new columns automatically from the model"
  artifacts:
    - path: "internal/models/user.go"
      provides: "AdOuDn, AdUserDn, AdSyncedAt fields on User struct"
    - path: "internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql"
      provides: "Explicit migration for new columns with index"
  key_links:
    - from: "internal/services/addomain/user_ou_service.go"
      to: "sys_user table"
      via: "GORM Updates map with ad_ou_dn, ad_user_dn, ad_synced_at keys"
      pattern: "ad_ou_dn|ad_user_dn|ad_synced_at"
---

<objective>
Add AdOuDn, AdUserDn, and AdSyncedAt fields to the User model so that AD login department sync works without SQL "column does not exist" errors.

Purpose: The AD user login flow writes to `ad_ou_dn`, `ad_user_dn`, and `ad_synced_at` columns in sys_user, but these columns do not exist in the database because the User model never declared them. This causes every AD user login to silently fail department synchronization.

Output: Updated User model with three new fields, plus a SQL migration script.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/models/user.go
@internal/services/addomain/user_ou_service.go
@internal/services/addomain/user_ad_sync_service.go
@internal/core/db/database.go (AutoMigrate section)
@internal/core/db/migrations/138_fix_addn_root_cause.sql (prior migration pattern)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add AD fields to User model and create migration</name>
  <files>internal/models/user.go, internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql</files>
  <action>
Add three new fields to the User struct in `internal/models/user.go`, in the "AD认证相关字段" section, after the existing `AdDn` field:

```go
AdOuDn     *string    `gorm:"type:text;column:ad_ou_dn" json:"adOuDn,omitempty"`
AdUserDn   *string    `gorm:"type:text;column:ad_user_dn" json:"adUserDn,omitempty"`
AdSyncedAt *time.Time `gorm:"column:ad_synced_at" json:"adSyncedAt,omitempty"`
```

These map exactly to the column names used by `user_ou_service.go` (line 69-71: `"ad_user_dn"`, `"ad_ou_dn"`, `"ad_synced_at"`) and `user_ad_sync_service.go` (line 115: `Update("ad_ou_dn", ouDN)`, line 175: `Update("ad_synced_at", ...)`).

The `time` import already exists in the file.

Then create migration `internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql`:
```sql
-- Migration 139: Add ad_ou_dn, ad_user_dn, ad_synced_at columns to sys_user
-- Description: AD login department sync writes to these columns
-- Created: 2026-05-27

-- Add columns if they don't exist (GORM AutoMigrate will also handle this, but explicit is safer)
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_ou_dn TEXT;
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_user_dn TEXT;
ALTER TABLE sys_user ADD COLUMN IF NOT EXISTS ad_synced_at TIMESTAMPTZ;

-- Create index for OU DN lookups (used in AD sync queries)
CREATE INDEX IF NOT EXISTS idx_sys_user_ad_ou_dn ON sys_user (ad_ou_dn) WHERE ad_ou_dn IS NOT NULL;
```

IMPORTANT: `AdDn` (user's full DN) already exists in the model as `*string` with `column:ad_dn`. The new `AdUserDn` field stores the same user DN but is updated separately by `user_ou_service.go`. Both fields serve the same logical purpose but are written at different points in the flow. Do NOT merge them -- keep them separate to match the existing service code exactly.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./...</automated>
  </verify>
  <done>
    - User struct has AdOuDn, AdUserDn, AdSyncedAt fields with correct GORM column tags
    - Migration 139 SQL file exists with ALTER TABLE and CREATE INDEX
    - `go build ./...` passes with zero errors
  </done>
</task>

<task type="auto">
  <name>Task 2: Verify column name consistency across service code</name>
  <files>none (verification only)</files>
  <action>
Verify that the column names in the model GORM tags match exactly what the service code uses. Run these checks:

1. `user_ou_service.go` line 67-72 uses map keys: `"ad_user_dn"`, `"ad_ou_dn"`, `"ad_synced_at"` -- must match `column:ad_user_dn`, `column:ad_ou_dn`, `column:ad_synced_at` in model tags. CONFIRMED MATCH.

2. `user_ad_sync_service.go` line 115 uses `Update("ad_ou_dn", ouDN)` -- must match `column:ad_ou_dn`. CONFIRMED MATCH.

3. `user_ad_sync_service.go` line 175 uses `Update("ad_synced_at", ...)` -- must match `column:ad_synced_at`. CONFIRMED MATCH.

4. `user_ad_sync_service.go` line 254 uses `Update("ad_ou_dn", ouDN)` -- must match `column:ad_ou_dn`. CONFIRMED MATCH.

5. The `ad_user_dn` column in the model should NOT conflict with the existing `ad_dn` column from `AdDn`. They are separate columns: `ad_dn` stores the user's AD distinguished name set during auth, `ad_user_dn` stores the same value but updated by the OU service during login dept sync. Both columns coexist.

6. Run `go test ./internal/services/addomain/...` to verify the service tests still compile (they reference these columns in their SQLite table DDL).

No file changes needed for this task -- it is a verification pass.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./... && go vet ./internal/services/addomain/...</automated>
  </verify>
  <done>
    - All column name references in service code match the model GORM tags
    - Build passes, vet passes, no compilation errors
  </done>
</task>

</tasks>

<verification>
1. `go build ./...` -- compiles without errors
2. `go vet ./internal/services/addomain/...` -- no issues
3. After deployment, AD user login should update dept_id, ad_user_dn, ad_ou_dn, ad_synced_at without SQL errors
</verification>

<success_criteria>
- User model has three new fields: AdOuDn, AdUserDn, AdSyncedAt with correct GORM column mappings
- Migration 139 SQL file exists
- `go build ./...` passes
- All existing service code column references match the new model tags
</success_criteria>

<output>
After completion, create `.planning/quick/260527-gra-user-adoudn/260527-gra-SUMMARY.md`
</output>

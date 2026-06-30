# reconciliation_normalized.workstation_id 列创建失败

## Symptom (User-reported)

```
ERRO[2026-06-30 14:42:48] Phase 45+ reconciliation_normalized MV workstation_id 列修复失败: reconciliation_normalized.workstation_id 列创建失败(information_schema 仍缺失)
2026/06/30 14:42:48 Running migration 117: Create sys_mac_filter_rules table
2026/06/30 14:42:48 Table sys_mac_filter_rules already exists, skipping migration 117...
2026/06/30 14:42:48 Running migration 033: Create sys_mac_oui_vendor table
2026/06/30 14:42:48 Table sys_mac_oui_vendor already exists, skipping migration 033...
```

**Date reported**: 2026-06-30 14:42:48
**Module**: Phase 45+ reconciliation physical chain
**Target**: PostgreSQL MV `reconciliation_normalized` → column `workstation_id`

## Initial Context

- Migration file likely at `internal/core/db/migrations/migration_181_reconciliation_normalized_workstation_id.go`
- Earlier Phase 45 added reconciliation physical chain (per STATE.md, R5 work)
- Reconciliation_normalized is a materialized view (MV)
- MV references `sys_workstation` (likely) and `sys_device` etc.
- PostgreSQL won't allow `ALTER TABLE` on a column referenced by a MV without DROP/CREATE
- Reference: `reconciliation-mv-refresh-zombie-context-cancel.md` for related MV zobmie issues
- Reference: `gorm-automigrate-blocked-by-matview.md` for "GORM AutoMigrate blocked by materialized view"

## Hypotheses to investigate

1. Migration attempts `ALTER TABLE reconciliation_normalized ADD COLUMN workstation_id` but PG errors with "cannot alter type ... used by view"
2. Migration tries to DROP+RECREATE MV but the REPLACE loses a grant/permission
3. Migration runs AFTER GORM AutoMigrate but the MV column never gets added because AutoMigrate strips/ignores MV
4. Migration depends on a prior column that also doesn't exist in information_schema
5. Information_schema lag - MV was dropped but its schema entries not cleared
6. Migration SQL syntax correct but uses `IF NOT EXISTS` clause that doesn't exist for ADD COLUMN in older PG

## Investigation Steps

- [x] Read migration_181_reconciliation_normalized_workstation_id.go to understand the repair attempt
- [x] Check what migrations ran in Phase 45+ to build reconciliation_normalized
- [x] Check if MV exists in DB (DROP/EXISTS pattern)
- [x] Check if other columns of reconciliation_normalized are present
- [x] Check DB log for the specific PG error
- [x] Determine correct fix: DROP MV + ALTER base tables + REFRESH CONCURRENTLY MV (or just DROP/CREATE MV)

## Key Evidence Found

### Evidence 1: migration_173 explicitly documents the bug

`internal/core/db/migrations/migration_173_reconciliation_silence_mv.go:177-198`:

```go
// R5 修正(2026-06-29): PG materialized view 的列**不**在 information_schema.columns
// 里(0 行返回是 PG 标准行为)。改用 pg_attribute 查 attnum > 0 排除系统列,
// attisdropped 过滤已删列,真实反映 MV 列存在性。
//
// 旧代码用 information_schema.columns,MV 一直返回 0 行被误判为"列缺失",
// 导致 173 永远报 "MV 缺少 last_resolved_at"。本次 R5 调整后,
// 验证通过的判定基于 pg_attribute 是否找到 last_resolved_at 这条 attname。
var lastResolvedAttname string
if err := db.Raw(`
    SELECT a.attname FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid
    WHERE c.relname = 'reconciliation_normalized'
      AND a.attname = 'last_resolved_at'
      AND a.attnum > 0 AND NOT a.attisdropped
    LIMIT 1
`).Scan(&lastResolvedAttname).Error; err != nil {
    ...
}
```

This is the exact pattern that migration 181 should use but doesn't. The author already knew this for migration 173 but did NOT apply the lesson to migration 181.

### Evidence 2: migration_181 step 3 (CREATE INDEX) proves the column DOES exist

`migration_181:118`:
```go
`CREATE INDEX IF NOT EXISTS idx_recon_norm_workstation_id
   ON reconciliation_normalized (workstation_id)
   WHERE workstation_id IS NOT NULL;`,
```

If `workstation_id` did not exist, the CREATE INDEX would have failed with SQLSTATE 42703. But step 3 has no error path triggered (no log line "[迁移] 181 索引重建失败" appears in the log; only the workstation_id-missing error appears later).

### Evidence 3: migration_181 step 2 (CREATE MV) logs success but step 4 verification fails

The flow:
1. `db.Exec(createMV).Error` returns nil (MV created successfully) → logs `[迁移] 181 reconciliation_normalized 已重建(含 workstation_id 列)`
2. Step 3 indexes all created successfully (no error log)
3. Step 4 verification returns 0 because `information_schema.columns` does NOT contain materialized view columns

The MV was created with the workstation_id column correctly. The verification is using the wrong catalog.

### Evidence 4: GORM Scan(int64) of count(*) IS working — same pattern works for migration_176

`migration_176:180`:
```go
if err := db.Raw("SELECT COUNT(*) FROM reconciliation_normalized").Scan(&mvCount).Error; err != nil {
```

This pattern works fine. The issue is NOT with GORM Scan + int64. The issue is purely that `information_schema.columns` does not list materialized view columns.

### Evidence 5: `reconciliation_physical_chain` view exposes workstation_id (no schema issue)

`migration_180:101`: `ws.id AS workstation_id`
`migration_175:86`: `ws.id AS workstation_id`
`migration_178:81`: `ws.id AS workstation_id`

The view column is consistently named `workstation_id` across all migrations. Migration 181's `pc.workstation_id AS workstation_id` reference is valid.

### Evidence 6: All migration DDL uses unqualified names; schema is `public`

DSN: `host=... dbname=...` — no search_path override. PostgreSQL default search_path is `$user, public`. The MV is created in `public` schema. `information_schema.columns` does list columns from `public` tables — but NOT from MVs.

## Root Cause

**HIGH CONFIDENCE**: The verification query in migration_181 line 132-136 uses `information_schema.columns` which does NOT contain materialized view columns in PostgreSQL. This is documented PostgreSQL behavior — only base tables and regular views are reflected in `information_schema.columns`. Materialized views store their columns in `pg_attribute`/`pg_class` only.

The MV was created successfully (step 2) and `workstation_id` exists in pg_attribute. The CREATE INDEX on `workstation_id` (step 3) succeeded, proving the column exists. But step 4's verification query against `information_schema.columns` returns 0 rows (standard PG behavior for MVs).

This is a self-inflicted false negative. Migration 173 already hit this exact bug and was patched to use `pg_attribute`. Migration 181 was written without applying that lesson.

The user's downstream code (`WorkstationIDForException`) WILL work correctly because the MV column was created.

## Fix

Replace the verification query in `migration_181_reconciliation_normalized_workstation_id.go` lines 131-141 with the pg_attribute pattern already proven in migration 173:

```go
var wsIDAttname string
if err := db.Raw(`
    SELECT a.attname FROM pg_attribute a
    JOIN pg_class c ON c.oid = a.attrelid
    WHERE c.relname = 'reconciliation_normalized'
      AND a.attname = 'workstation_id'
      AND a.attnum > 0 AND NOT a.attisdropped
    LIMIT 1
`).Scan(&wsIDAttname).Error; err != nil {
    return fmt.Errorf("验证 reconciliation_normalized.workstation_id 列存在失败: %w", err)
}
if wsIDAttname != "workstation_id" {
    return fmt.Errorf("reconciliation_normalized.workstation_id 列创建失败(pg_attribute 查无)")
}
```

This is the same fix already applied to migration_173. Update the error message text on line 140 accordingly.

## Status

- diagnosis: complete
- fix: identified (do NOT apply per user request — diagnose only)
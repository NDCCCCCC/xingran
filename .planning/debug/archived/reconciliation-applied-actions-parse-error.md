# AppliedActions pq.StringArray Scan 失败

## Symptom (User-reported)

```
INFO[2026-06-30 14:52:04] Request processed  client_ip=10.62.10.33 latency=18 method=POST path=/api/v1/ops/workstation-device/5300cc2d-66e7-4985-9c66-e88ac6242706/ad request_body="{}" request_id=mr0ah167lvce67p6gri status_code=200 user_agent="Mozilla/5.0 ... Chrome/120.0.0.0 Safari/537.36"

2026/06/30 14:52:04 D:/CODE/ClaudeCode/xingran-go-backend/internal/services/asset/reconciliation_service.go:899
[error] failed to parse field: AppliedActions, error: unsupported data type: &[]
```

**Date reported**: 2026-06-30 14:52:04
**Module**: reconciliation_service.go GetByWorkstation (badge data feed)
**Endpoint**: POST /api/v1/ops/workstation-device/{wsID}/ad
**API response**: 200 OK (request succeeded, but Scan warning fired)

## Initial Context

- `internal/services/asset/reconciliation_service.go:880-903` — healthRow struct with `AppliedActions pq.StringArray`
- `internal/models/reconciliation.go:50` — `AppliedActions pq.StringArray gorm:"type:text[]"`
- Driver stack: gorm.io/driver/postgres v1.5.9 → jackc/pgx/v5 v5.5.5
- Also: github.com/lib/pq v1.10.9 (where pq.StringArray comes from)

## Likely Root Cause

GORM Scan into `pq.StringArray` fails when pgx returns `&[]` (pointer to empty slice) for a NULL or empty text[] column. This is a known incompatibility between pgx's array handling and lib/pq's StringArray type.

The error message "unsupported data type: &[]" comes from pgx (or the postgres driver) trying to scan the result into the target type — pq.StringArray doesn't accept a `&[]` because it expects a different binary representation.

## Hypotheses to investigate

1. **NULL column → &[]**: The applied_actions column is NULL for all rows in this query, pgx returns `&[]`, pq.StringArray Scan fails.
2. **Empty array literal → &[]**: pgx returns empty array as `&[]`, pq.StringArray doesn't recognize.
3. **pgx/pq driver mismatch**: pgx v5 + lib/pq StringArray type incompatibility (GORM driver wraps pgx, lib/pq type used in model).
4. **Column default vs Scan target**: Column default in DB is something that pgx encodes as `&[]`.
5. **Scan target is wrong type**: `pq.StringArray` should be `*pq.StringArray` to handle NULL.

## Investigation Steps

- [ ] Check what pgx v5 returns for NULL text[] columns
- [ ] Check what pgx v5 returns for empty text[] columns
- [ ] Check the actual column data in DB for one of the affected workstations
- [ ] Test if changing to `*pq.StringArray` fixes the issue
- [ ] Test if changing to `pq.StringArray` + COALESCE in SQL fixes the issue
- [ ] Test if changing to `datatypes.PostgreSQLStringArray` (gorm.io/datatypes) fixes the issue
- [ ] Check other places in the codebase that use pq.StringArray Scan successfully

## Fix Candidates

1. **Change to pointer**: `AppliedActions *pq.StringArray` — handles NULL gracefully
2. **Add SQL COALESCE**: `COALESCE(r.applied_actions, '{}') AS applied_actions` — never returns NULL
3. **Use custom type**: Replace pq.StringArray with a custom scan type that handles pgx's encoding
4. **Drop the field from SELECT** if it's not used downstream — simplest if AppliedActions isn't consumed

## Status

- diagnosis: pending (gsd-debugger dispatched)
- fix: pending
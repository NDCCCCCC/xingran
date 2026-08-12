---
status: investigating
trigger: "POST /api/v1/ops/workstation returns 500 - GORM INSERT into sys_workstation with column primary_device_serial (SQLSTATE 42703)"
created: 2026-06-26
updated: 2026-06-26
---

## Current Focus

hypothesis: GORM tag `gorm:"-:migration"` on PrimaryDeviceSerial field does NOT skip the field during Create(); it only skips migration. GORM still treats the field as a column to insert.
test: Investigate GORM v1.30.5 source for how `"-:"` prefix is parsed vs the suffix portion.
expecting: Confirm that GORM treats the tag value (after splitting on `;`) and does not interpret `-:` as a complete "ignore" token when followed by other directives.
next_action: Read go.mod for GORM version, then search GORM source for tag parsing of `"-:"` prefix.

## Symptoms

expected: POST creates workstation successfully.
actual: 500 - GORM INSERT includes `primary_device_serial` column which doesn't exist in DB.
errors: ERROR: column "primary_device_serial" of relation "sys_workstation" does not exist (SQLSTATE 42703)
reproduction: POST /api/v1/ops/workstation → triggers service.Create → db.Create(workstation)
started: Recent (after adding PrimaryDeviceSerial derived field)

## Eliminated

- hypothesis: `gorm:"-:migration"` causes GORM to skip the field entirely during Create().
  evidence: GORM v1.30.5 schema/field.go:344-361 — the switch on `field.TagSettings["-"]` branches on the suffix; `case "migration"` only sets `field.IgnoreMigration = true`. `Creatable`/`Updatable`/`Readable` keep their default value of `true` from line 123-125. Schema/schema.go:220-231 then registers the field's derived DBName `primary_device_serial` into `schema.DBNames` so it is included in INSERT.
  timestamp: 2026-06-26

- hypothesis: `gorm:"-"` would solve the Create error without breaking the JOIN read.
  evidence: schema/field.go:347-351 — the `case "-"` branch sets `Creatable=false`, `Updatable=false`, `Readable=false`. `Readable=false` prevents the Scan step from populating the field from the JOIN subquery alias, breaking List/GetByID. Need `gorm:"->"` (read-only) instead.
  timestamp: 2026-06-26

- hypothesis: `PrimaryDeviceSerial` could be pre-populated from request body.
  evidence: handler binds JSON via json tag; the request body in the failing log (`{"orgId":...,"floorId":...,"name":"4FF","type":0,"width":160,"depth":70,"status":0}`) does NOT include `primaryDeviceSerial`. Field is nil. GORM still adds the column to INSERT because the field's Creatable=true and DBName is registered — nil pointer doesn't suppress schema-registered columns.
  timestamp: 2026-06-26

- hypothesis: a missing migration created the column at some point and the error would disappear.
  evidence: full Grep of `internal/core/db/migrations/` for `primary_device_serial` returned zero matches; only `workstation_service.go:16` references the alias as a SELECT subquery alias on `ops_workstation_device`. The column intentionally never existed on `sys_workstation`.
  timestamp: 2026-06-26

## Evidence

- timestamp: 2026-06-26
  checked: GORM v1.30.5 module cache: `schema/utils.go:16-45` (ParseTagSetting)
  found: Tag string `"-:migration"` is split on `;` yielding one piece, then split on `:` yielding `["-", "migration"]`. The map key is upper-cased `"-"`, value `"migration"`.
  implication: `field.TagSettings["-"] = "migration"` (NOT `"-"`).
- timestamp: 2026-06-26
  checked: GORM v1.30.5 `schema/field.go:344-361`
  found: switch on `val` (the value of `field.TagSettings["-"]`):
    case "-":        → Creatable/Updatable/Readable=false, DataType=""
    case "all":      → same as "-" plus IgnoreMigration=true
    case "migration": → ONLY IgnoreMigration=true (Creatable/Updatable/Readable untouched, remain default true)
  implication: `gorm:"-:migration"` matches the `migration` case → only stops AutoMigrate from creating the column. It does NOT prevent Create/Update/Read.
- timestamp: 2026-06-26
  checked: GORM v1.30.5 `schema/field.go:107-133` (field struct init)
  found: Default `Creatable=true`, `Updatable=true`, `Readable=true`. Only the switch above changes them.
  implication: Without `case "-"` or `case "all"`, the field is fully writable.
- timestamp: 2026-06-26
  checked: GORM v1.30.5 `schema/schema.go:220-231`
  found: If `field.DBName == ""` and `field.DataType != ""`, then `field.DBName = namer.ColumnName(table, field.Name)` — for `PrimaryDeviceSerial` on `sys_workstation`, this produces `primary_device_serial`. Then field is added to `schema.DBNames` (used by INSERT builder) and `schema.FieldsByDBName` (used by Scan mapping).
  implication: The column is registered. Verified with: `naming.ColumnName("sys_workstation", "PrimaryDeviceSerial")` → `"primary_device_serial"`.
- timestamp: 2026-06-26
  checked: GORM v1.30.5 `schema/field.go:363-368` (read-only -> tag)
  found: `if v, ok := field.TagSettings["->"]; ok { Creatable=false; Updatable=false; if !strings.ToLower(v)=="false" { Readable=true } }`. The `->` tag makes a field read-only.
  implication: This is the correct tag for a derived/computed field populated only by SELECT subqueries.
- timestamp: 2026-06-26
  checked: codebase pattern search — `internal/models/` for `gorm:"-` tags
  found: 13 other fields using `gorm:"-"` (no suffix): `LeaderName`, `LeaderUsername`, `CredentialName`, `Description` (floor), `DeptFullName`, `Roles`, `RoleIds`, `Progress`, plus 5 in `rpa/credentials.go`. None use `gorm:"-:migration"` and none are JOIN-derived fields.
  implication: The convention for transient/transient-computed fields is `gorm:"-"`, but those are for fields NEVER read back from the DB. `PrimaryDeviceSerial` IS read back (via JOIN subquery) so it needs `gorm:"->"`.
- timestamp: 2026-06-26
  checked: handler `internal/api/v1/operations/workstation_handler.go:80-92`
  found: Create binds JSON into `models.Workstation` (Gin uses json tag), then calls `service.Create(ctx, &workstation)` → `s.db.Create(workstation)`.
  implication: Gin JSON binding is decoupled from GORM tag. Even if field is nil in struct, GORM still adds the registered column to INSERT (because schema registered the column based on field declaration, not on nil check).
- timestamp: 2026-06-26
  checked: failing request body in debug session log
  found: `{"orgId":"...","floorId":"...","name":"4FF","type":0,"width":160,"depth":70,"status":0}` — `primaryDeviceSerial` is NOT in body. Field would be nil pointer after binding.
  implication: nil-pointer argument. GORM's Create() builds INSERT from schema fields, not from non-nil struct values (for column inclusion — values are handled separately). Confirms: even empty submissions hit the bug.

## Resolution

root_cause: The `gorm:"-:migration"` tag on `PrimaryDeviceSerial` (internal/models/workstation.go:62) matches the `case "migration"` branch in GORM v1.30.5 schema/field.go:358-360, which only sets `IgnoreMigration=true`. The field's `Creatable`, `Updatable`, `Readable` flags remain `true` (defaults at field.go:123-125), so GORM's `db.Create(workstation)` includes this column in the INSERT. Because `PrimaryDeviceSerial` has no `column:` override, GORM derives DBName from the field name → `primary_device_serial` (verified with naming.ColumnName). That column does not exist on the `sys_workstation` table (no migration ever created it; it lives only as a SELECT subquery alias on `ops_workstation_device`). PostgreSQL rejects the INSERT with SQLSTATE 42703.
fix: (diagnosis only — no edit applied)
verification: (pending human verification post-fix)
files_changed: []
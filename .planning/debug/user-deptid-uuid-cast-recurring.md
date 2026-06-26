---
slug: user-deptid-uuid-cast-recurring
status: resolved
trigger: "ERRO[2026-06-15 12:38:58] invalid input syntax for type uuid: \"\" (SQLSTATE 22P02) — CASE WHEN regex guard in JOIN ON clause is still failing"
created: 2026-06-15T00:00:00.000Z
updated: 2026-06-26
---

# Debug Session: sys_user.dept_id UUID Cast — DEEP ROOT CAUSE

## Final Diagnosis

**ROOT CAUSE: All previous fixes have been predicated on a FALSE ASSUMPTION.**

The repeated assumption has been: `sys_user.dept_id` is a VARCHAR(64) string column that may contain non-UUID garbage like `''`, `' '`, `'null'`. The "fixes" added increasingly elaborate CASE WHEN guards with `IS NOT NULL AND dept_id != '' AND dept_id ~ '^[UUID regex]$'` to defensively cast to `::uuid`.

**The truth:** `sys_user.dept_id` has been a native `uuid` column in PostgreSQL for a long time. It cannot hold empty strings, whitespace, or `'null'`. It can only hold NULL or valid UUIDs. The Go model claims `*string gorm:"size:64"`, but GORM's postgres driver auto-converts UUID columns to `*string` on the Go side. **The CAST `sys_user.dept_id::uuid` is operating on a value that is ALREADY uuid, and `dept_id ~ 'regex'` is an operator that does not exist for the uuid type.**

This is why each "fix" continues to fail with the same misleading `invalid input syntax for type uuid: ""` error — the error is a **secondary/false-positive** triggered by PostgreSQL's planner when it tries to coerce the uuid column for the non-existent `uuid ~ unknown` operator and falls back through invalid cast paths.

## Symptoms

- **expected**: User list query executes successfully, returning users with their department names
- **actual**: PostgreSQL reports `invalid input syntax for type uuid: ""` (SQLSTATE 22P02) regardless of the CASE WHEN guards added
- **errors**: `SQLSTATE 22P02`
- **started**: 2026-06-15 (recurring after multiple fixes attempted since 2026-05-25)
- **reproduction**: Call `GET /system/user/list` (or any code path using `userJoinClause`)

## Current Focus

- **hypothesis**: CONFIRMED — `sys_user.dept_id` column type is `uuid`, not varchar. The `dept_id::uuid` cast is redundant, and the `dept_id ~ regex` operator does not exist for uuid type. PostgreSQL's planner generates a misleading error when it tries to coerce uuid to text for the regex operator.
- **test**: Direct PostgreSQL queries via probe (D:\tmp\deptid_probe\main.go)
- **expecting**: Drop the CASE WHEN entirely; replace with simple `sys_dept.id = sys_user.dept_id`
- **next_action**: Return ROOT CAUSE FOUND to orchestrator

## Evidence

### E1: Column type — sys_user.dept_id is UUID
- **source**: PostgreSQL `information_schema.columns` and `pg_attribute`
- **found**:
  - `data_type = "uuid"`, `udt_name = "uuid"`
  - `pg_attribute.atttypid -> pg_type.typname = "uuid"`
- **implication**: The column CANNOT store empty strings, whitespace, or `'null'` — only NULL or valid UUIDs. All previous "data quality" fixes targeting non-UUID strings were predicated on a false assumption.

### E2: All sys_user rows have valid UUID dept_id
- **source**: `SELECT dept_id, COUNT(*) FROM sys_user WHERE deleted_at IS NULL GROUP BY dept_id`
- **found**: 2249 rows, ALL with valid UUID-format dept_id. ZERO NULL. ZERO empty strings.
- **implication**: The "fix the bad data" approach is moot — there is no bad data to fix. The bad-data theory was wrong.

### E3: Direct JOIN works perfectly without CASE WHEN
- **source**: probe — `SELECT COUNT(*) FROM sys_user LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id`
- **found**: Returns 2249 rows, no error
- **implication**: The simplest fix works. Both columns are already UUID; no cast needed.

### E4: Even `dept_id != ''` fails — UUID can't compare to text literal
- **source**: probe — production SQL stripped to `IS NOT NULL AND != ''`
- **found**: `ERROR: invalid input syntax for type uuid: "" (SQLSTATE 22P02)`
- **implication**: The error is reproducible with the SIMPLEST guard, no regex. This is not a data problem.

### E5: The `dept_id ~ 'regex'` operator does not exist for UUID
- **source**: probe — `SELECT COUNT(*) FROM sys_user WHERE dept_id ~ '^[0-9a-fA-F]{8}-...'`
- **found**: `ERROR: operator does not exist: uuid ~ unknown (SQLSTATE 42883)`
- **implication**: The regex guard in the production CASE WHEN is actually an undefined operator. PostgreSQL's planner attempts coercion and generates the misleading secondary error `invalid input syntax for type uuid: ""`.

### E6: Using `dept_id::text ~ regex` works
- **source**: probe — variant with explicit text cast
- **found**: Returns 2249 rows successfully
- **implication**: Confirms the root cause — the regex was being applied to uuid type instead of text. With explicit `::text` cast, the regex works.

### E7: 0 orphan users (every dept_id matches a real sys_dept)
- **source**: `SELECT COUNT(*) FROM sys_user u LEFT JOIN sys_dept d ON d.id = u.dept_id WHERE d.id IS NULL`
- **found**: 0 orphans
- **implication**: Simple equality JOIN gives correct results — no need for defensive null filtering.

### E8: Final fix verified end-to-end
- **source**: probe — full query `SELECT sys_user.*, sys_dept.dept_name, sys_dept.ancestors FROM sys_user LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1000`
- **found**: Returns correct rows with non-null `dept_name` and `ancestors`
- **implication**: The fix is the simplest possible: drop the entire CASE WHEN.

### E9: Same bug pattern appears in 7 worktrees
- **source**: `.claude/worktrees/*/internal/services/system/user_service.go`
- **found**: Worktrees have variants:
  - `agent-a50027555adb03978`: `NULLIF(sys_user.dept_id, '')::uuid` (P1 attempt — broken because NULLIF returns text)
  - `agent-a53c852672762fc53`, `agent-a8e6bef26c2fc83bc`, `agent-ac3d253da1dd84419`, `agent-ad7cd1330a9804166`, `agent-adb55657df6c019b7`: bare `sys_user.dept_id::uuid` (original — broken)
  - main branch: regex-guarded CASE WHEN (current — still broken)
- **implication**: All these attempts share the false assumption that `dept_id` is text and needs casting.

### E10: Prior resolved debug session set precedent
- **source**: `.planning/debug/resolved/dept-uuid-null-fix.md` (2026-05-25)
- **found**: "UUID类型字段不能与空字符串比较，只能与NULL或有效UUID值比较" — same root cause for `sys_dept.parent_id` UUID column with `parent_id = ''`
- **implication**: The team has hit this exact pattern before. The lesson: when seeing `invalid input syntax for type uuid: ""`, **first check if the column is already UUID** before adding defensive string comparisons.

## Eliminated

### ELIM1: Bad data (whitespace, 'null' literals, malformed UUIDs) in dept_id
- **evidence**: E2 — all 2249 rows have valid UUIDs; E1 — column type is uuid and physically cannot store non-UUID strings.
- **status**: Confirmed not the cause.

### ELIM2: GORM reordering CASE WHEN conditions
- **evidence**: The error reproduces with stripped-down guards (E4), and with no guard at all in some variants. Not GORM-specific.
- **status**: Confirmed not the cause.

### ELIM3: GORM `Joins(string)` parsing/rewriting the ON clause
- **evidence**: Direct `db.Raw(sql)` with the same JOIN ON expression produces the same error. The SQL GORM generates is byte-identical to what we wrote.
- **status**: Confirmed not the cause.

### ELIM4: SysUser `gorm:"foreignKey:DeptID"` relation auto-adding a JOIN
- **evidence**: The error reproduces when the JOIN is sent via `db.Raw()` with no GORM relation walking. The model relation is not exercised.
- **status**: Confirmed not the cause.

### ELIM5: `*string` Go type causing NULL conversion to empty string
- **evidence**: E2 — column has 0 NULL values, and even if it did, the SQL JOIN ON expression evaluates server-side, not Go-side.
- **status**: Confirmed not the cause.

### ELIM6: Stale binary running old SQL
- **evidence**: The probe connects to the same database and reproduces the error using the CURRENT production SQL. The user's runtime may have updated since, but the SQL itself is fundamentally broken.
- **status**: Confirmed not the cause (current code is also broken).

## Resolution

- **root_cause**: `sys_user.dept_id` is a native PostgreSQL `uuid` column (NOT varchar as the Go model tag suggests). All defensive CASE WHEN / NULLIF / regex guards in `userJoinClause` are predicated on a false assumption that `dept_id` may contain non-UUID strings. The guards fail because:
  1. `dept_id != ''` — UUID can't compare to empty string literal
  2. `dept_id ~ 'regex'` — operator `uuid ~ unknown` does not exist
  3. PostgreSQL's planner, when trying to resolve these, generates a secondary/misleading error: `invalid input syntax for type uuid: ""`

  The simplest correct JOIN is `sys_dept.id = sys_user.dept_id` with no cast, no CASE WHEN, no regex.

- **fix**: Replace the entire `userJoinClause` string with a simple equality JOIN:
  ```go
  userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id"
  ```
  No `Select(...)` change needed — `sys_user.*, sys_dept.dept_name, sys_dept.ancestors` is correct.

- **files_changed**: `internal/services/system/user_service.go` (line 336)

- **verification**:
  - Probe confirms: `SELECT COUNT(*) FROM sys_user LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id WHERE deleted_at IS NULL` returns 2249
  - Probe confirms: full SELECT with all columns returns rows with non-null `dept_name` and `ancestors`
  - All 2249 dept_id values are valid UUIDs and 0 orphans

- **additional fixes recommended (out of scope unless requested)**:
  - The Go model tag `gorm:"size:64"` on `DeptID *string` is misleading — should be removed or set to `gorm:"type:uuid"` for clarity (does not change behavior because the postgres driver handles UUID natively)
  - Worktree copies of user_service.go should also be fixed to prevent recurrence

## Suggested Fix Direction (DO NOT APPLY per user instruction)

**Single-line change at `internal/services/system/user_service.go:336`:**

```diff
- userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = CASE WHEN sys_user.dept_id IS NOT NULL AND sys_user.dept_id != '' AND sys_user.dept_id ~ '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN sys_user.dept_id::uuid END"
+ userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id"
```

**Specialist Hint:** `go` (GORM Go backend; Go code change)

## Phase 41 Closure (2026-06-26)

**复测:** `userJoinClause` 简化修复已落地,本 plan 不重复实修。
- `internal/services/system/user_service.go:389` `userJoinClause := "LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id"` — 直接等值连接,**无 CASE WHEN / NULLIF / regex guard**。
- `internal/services/system/user_service.go:385` 注释明确说明"之前的 NULLIF/CASE WHEN 防御性写法是错误的:基于'dept_id 是 VARCHAR'的错误假设"。
- 文件中其余 4 处 `CASE WHEN` 出现在统计聚合查询(行 470/480/481),用于 `SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END)` 类聚合,与本 session JOIN 问题无关,**保留**。

**根因复述(沿用 .md 诊断):** `sys_user.dept_id` 列原生是 PostgreSQL `uuid` 类型,所有"防御性"写法(`dept_id != ''`、`dept_id ~ 'regex'`、`dept_id::uuid`)都基于错误假设;PostgreSQL planner 在尝试对 uuid 列做 text 比较/正则时产生误导性二次错误 `invalid input syntax for type uuid: ""`。直接 `sys_dept.id = sys_user.dept_id` 等值连接即可,无需任何 cast 或 guard。

**Phase 41 验证:** `go build ./...` 退出 0(本 plan 未触发任何 .go 改动)。

### won't_fix_reason (D-02)
复测确认 userJoinClause 已简化为 `LEFT JOIN sys_dept ON sys_dept.id = sys_user.dept_id`(user_service.go:389),代码层修复完整;本 plan 复测证据即翻 resolved。
action: wontfix (D-02,复测发现已落地型)
verification: 复测 `internal/services/system/user_service.go:389` userJoinClause 已简化,go build ./... 退出 0
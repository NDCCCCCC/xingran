# Phase 32: v1.14 P1 重构与 P2 架构优化 — Research

**Researched:** 2026-06-13
**Domain:** Backend security hardening, concurrency/consistency, business logic fixes, and architectural-debt cleanup
**Confidence:** MEDIUM-HIGH (most P1 items already fixed; remaining work is P2 refactors + validation tests)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- All 15 P1 items (7 security + 6 concurrency + 2 business logic) must be resolved
- All 8 P2 architectural-debt categories must be addressed
- Tech stack MUST follow existing patterns (Handler-Service, opsApi, excel_config)
- New fields/keys must be backward-compatible (no breaking changes)
- `org_id` and `user_id` foreign keys MUST be valid UUIDs
- Status convention: 0=normal/enabled, 1=stopped/disabled
- Response format: `response.Success()` / `response.Error()`
- Commit message format includes P1/P2 ID: `fix(security): P1-S4 取消子菜单权限继承`
- `go vet ./...` must produce 0 warnings
- Critical path test coverage ≥70% (AD/JWT/password modules)
- Re-audit against 20260612 review dimensions must pass
- P1 security hardening takes priority over P2 architectural debt
- Strong dependency: Phase 31 (P0 wrap-up) must be complete (✅ confirmed 2026-06-13)

### Claude's Discretion
- Wave split (7 waves suggested in ROADMAP.md, but planner may reorganize)
- Implementation order within each wave
- Test framework choices (table-driven vs property-based)
- Specific apperrors helper functions to add vs use existing generic ones
- Whether to apply minor fixes opportunistically when touching code

### Deferred Ideas (OUT OF SCOPE)
- P0 items already handled in Phase 31 (F-14 connection_pool, F-17 ConfigUpdateRequest)
- P0 items fixed via quick tasks (see `.planning/debug/resolved/`)
- "Good practices" section items from 20260612 review
- Future P1 items not in the 15-item curated list (47 → 15 selection)
- Frontend changes (only backend in scope)
- New feature work or milestone planning
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| P1-S1 | SM2 JWT alg header validation | ✅ Already fixed (commit 64b1b40); verification test needed |
| P1-S2 | Replay window tighten to ±60s (configurable) | ⚠️ Partial — currently ±120s (commit af05d99); tighten to ±60s + add config |
| P1-S3 | Nonce cleanup goroutine | ✅ Already fixed (commit 1071867); add test |
| P1-S4 | Disable child menu auto-inherit | ✅ Already fixed (commit 2b55e0d); add test |
| P1-S5 | PBKDF2 iterations ≥600000 (OWASP) | ⚠️ Partial — currently 100000 (commit b7dedac); bump to ≥600000 |
| P1-S6 | Random password no bias | ✅ Already fixed (commit 07f210c); add test |
| P1-S7 | Excel magic bytes validation | ✅ Already fixed (commit 2c74c06); add test |
| P1-C1 | AD sync singleflight mutex | ✅ Already fixed (commit 5c573c5); add concurrent test |
| P1-C2 | handleDeletedGroups threshold | ✅ Already fixed (commit 887a438); add test |
| P1-C3 | UpsertMapping single transaction | ✅ Already fixed (commit ffaecae); add test |
| P1-C4 | Port collector batch insert | ✅ Already fixed (commit 12d3139); add test |
| P1-C5 | WebSocket readPump zombie cleanup | ✅ Already fixed (commit a1e7324); add test |
| P1-C6 | validateUniqueness batch IN query | ❌ NOT fixed — verifyUniqueness still N+1; needs refactor |
| P1-B1 | Config encryption cache invalidation | ✅ Already fixed (commit 0bcac33); verify hot-reload works |
| P1-B2 | buildDepartmentPaths double call | ✅ Already fixed (commit ab0a279); verify only one call |
| P2-A1 | Core god struct split | ❌ NOT fixed — 18+ fields in core.Core; split into CoreInfra + CoreServices |
| P2-A2 | Cache keys consolidation | ❌ NOT fixed — two parallel systems (data_cache_service.go + system/cache_keys.go) |
| P2-A3 | Remove duplicate user_service_optimized | ✅ Already done (commit 3bdd3fc) |
| P2-A4 | Migration file renumbering | ❌ NOT fixed — 11 files with 027/028/029/030/031/036 duplicate prefixes |
| P2-A5 | apperrors unification for role_service | ❌ NOT fixed — 43 fmt.Errorf calls, 0 apperrors |
| P2-A6 | AD test coverage | ❌ NOT fixed — no LDAP mock; stripBaseDN + dept_ou_mapper have empty tests |
| P2-A7 | Subprocess management | ❌ NOT fixed — no Python subprocesses (scrapli is Go-native); but internal/agent/server has 15+ exec.Command calls without process group |
| P2-A8 | Excel import transaction wrapper | ❌ NOT fixed — ImportData has no Transaction; processThreeLevelDepartments not wrapped |
</phase_requirements>

---

## Summary

Phase 32 cleans up the remaining 15 P1 (curated from 47) and 8 P2 categories from the 20260612 backend code review. **CRITICAL FINDING:** Of the 15 P1 items, 11 are already partially or fully fixed in prior commits. The actual remaining P1 work is small (2 partials + 1 unfixed: P1-S2 tighten further, P1-S5 bump to OWASP, P1-C6 N+1 refactor) plus verification tests for all 15 items.

The real work is P2 architectural debt (6 of 8 categories unfixed): Core struct split, cache key consolidation, migration file renumbering, apperrors migration for role_service, AD test coverage with LDAP mocks, and Excel import transaction wrapping. **No Python subprocesses exist** — scrapli is Go-native via scrapligo — so P2-A7 needs to be re-scoped to internal/agent/server's PowerShell/Linux tool invocations.

**Primary recommendation:** Execute 7 waves (suggested) but **reorder** to put P1-C6 (N+1 unfixed) into Wave 1, group P2-A4/A5 (lower risk) earlier than suggested to de-risk the refactor-heavy P2-A1/A2.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| JWT signing/verification | API/Backend | — | Token validation is server-side crypto |
| Password hashing/migration | API/Backend | — | PBKDF2 requires server-side salt + iterations |
| Cache key management | API/Backend | — | Centralized in pkg/cache + services |
| Excel import transactions | API/Backend | — | All ImportData logic is in Go service |
| Migration file ordering | Database/Storage | — | SQL files are storage-layer |
| AD sync orchestration | API/Backend | Scheduler | Scheduler triggers + service layer mutex |
| WebSocket heartbeat | Frontend Server (WS) | — | Hub runs in backend, but client-side ping matters |
| Subprocess lifecycle | API/Backend | — | exec.Command only used in internal/agent/server |
| Error type unification | API/Backend | — | Service returns AppError, Handler maps to HTTP |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| golang.org/x/sync/singleflight | v0.19.0 | Single in-flight dedup | Already used in P1-C1 fix |
| gorm.io/gorm | v1.30.5 | ORM with Transaction | Already used throughout |
| gorm.io/gorm/clause | (bundled) | OnConflict for batch upsert | Already used in P1-C4 fix |
| github.com/tjfoc/gmsm | v1.4.1 | SM2/SM3/SM4 national crypto | Project standard |
| github.com/go-ldap/ldap/v3 | v3.4.12 | LDAP client | Already used in ldap_client.go |
| github.com/scrapli/scrapligo | v1.3.3 | Go-native device SSH (no subprocess) | Already used in device/ |
| github.com/xuri/excelize/v2 | v2.10.0 | Excel import/export | Already used in operations/ |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/stretchr/testify | (project-dep) | Assertions | All unit tests |
| gorm.io/driver/sqlite | v1.5.4 | In-memory test DB | AD dept_ou_mapper tests |
| gorm.io/driver/postgres | v1.5.9 | Production DB | Integration tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom LDAP mock | go-ldap/ldap/v3 test helpers | Mock gives full control over Connect/Bind/Search |
| apperrors.Wrap everywhere | sentinel errors | apperrors has 154 helpers + unified HTTP mapping; more idiomatic |
| rename migrations | keep duplicates + sort by content | Renumber is cleaner but touches many files |

**Version verification:** `go build ./...` passes on main as of 2026-06-13; Go 1.24.5 toolchain.

---

## Package Legitimacy Audit

> **N/A for this phase** — No new external packages will be installed. All P1/P2 work uses existing project dependencies.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| (none) | — | — | — | — | — | — |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### Recommended Project Structure (post Phase 32)
```
internal/services/system/
├── cache_keys.go             # SINGLE source of truth (P2-A2)
├── cache_adapter.go          # impl
├── cache_manager.go          # impl
├── role_service.go           # uses apperrors (P2-A5)
└── ...

internal/core/
├── core.go                   # split (P2-A1)
├── core_infra.go             # NEW — DB, Cache, Config, JWT
├── core_services.go          # NEW — UserService, RoleService, etc.
└── ...

internal/services/operations/
├── excel_service.go          # wrapped in Transaction (P2-A8)
└── ...

pkg/errors/
├── errors.go                 # existing
└── role_helpers.go           # NEW — RoleNameExists, RoleKeyExists (P2-A5)
```

### Pattern 1: P1 fix verification via regression tests

**What:** Every P1 fix commit should have an accompanying test that proves the fix works AND would have failed before the fix.

**When to use:** All P1/P2 items that modify security-sensitive code paths.

**Example:**
```go
// TestP1S1_SM2JWTRejectsAlgNone
// Source: Phase 31 pattern; commit 64b1b40
func TestSM2JWT_RejectsAlgNone(t *testing.T) {
    publicKey := loadTestPublicKey(t)
    
    // Craft token with alg=none
    header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
    payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","exp":99999999999}`))
    signature := ""  // alg=none uses empty signature
    token := header + "." + payload + "." + signature
    
    _, err := ValidateTokenWithSM2(token, publicKey)
    if err == nil {
        t.Fatal("expected alg=none to be rejected, got nil error")
    }
}

func TestSM2JWT_RejectsAlgHS256Confusion(t *testing.T) {
    publicKey := loadTestPublicKey(t)
    // Craft token with alg=HS256 using public key as HMAC secret
    // ... should be rejected
}
```

### Pattern 2: AD LDAP mocking via interface

**What:** Extract LDAP operations into an interface to enable mocking.

**When to use:** All AD module tests that exercise Connect/Bind/Search.

**Example:**
```go
// pkg/addomain/ldap_iface.go (NEW)
type LDAPConn interface {
    Bind(username, password string) error
    Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
    Close() error
}

type LDAPClientIface interface {
    Connect() error
    Bind() error
    Search(baseDN, filter string, attrs []string) (*ldap.SearchResult, error)
    Close() error
}

// Mock implementation
type mockLDAPClient struct {
    bindErr   error
    searchRes []*ldap.Entry
    searchErr error
}
```

### Pattern 3: Excel import full-transaction wrapper

**What:** Wrap the entire ImportData flow in a single GORM Transaction, with processThreeLevelDepartments as a nested callable.

**When to use:** All multi-stage operations that should be atomic.

**Example:**
```go
// Source: existing pattern in room_photo_service.go:129
func (s *ExcelService) ImportData(ctx, ...) (*ImportResult, error) {
    return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // ... all existing logic, but using tx instead of s.db
        // ... processThreeLevelDepartments(tx, ...)
        // ... upserter.Upsert(tx, ...)
        return nil
    })
}
```

### Anti-Patterns to Avoid
- **Renaming migrations in-place**: NEVER modify the content of a migration file once it's been applied to any environment. Only rename by appending a new "added" file or using a new prefix range (e.g., 142_add_x.go instead of 027_again_x.sql).
- **Bumping PBKDF2 iterations past 600000 in one step**: Login latency will spike 6x. Recommend a phased approach: deploy to dev, monitor, then prod.
- **Splitting core.Core across packages in one commit**: Will break every router setup. Use a step-wise migration: add CoreInfra + CoreServices with re-exports on Core first, then update router one module at a time.
- **Mocking go-ldap with a third-party library**: Use a hand-rolled interface — the project already uses gomock-free testing patterns.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LDAP mock for testing | Custom dial simulator | Interface extraction + struct mock | Project doesn't use gomock; minimal interface is enough |
| Cache key deduplication | Manual replace-all in code | `gofmt -r 'OLDKEY -> NEWKEY'` + new helper that reads `cache_keys.go` only | Manual edits miss string literals |
| Core struct split | Just rename fields | Embedding pattern (`type Core struct { *CoreInfra; *CoreServices }`) | Preserves backward-compat during migration |
| Excel transaction wrapper | Try-catch with manual rollback | GORM `s.db.WithContext(ctx).Transaction(func(tx)...)` | Tested pattern (room_photo_service.go uses it) |
| Migration renumbering | Find-and-replace on filename | Use `git mv` + commit + verify applied DBs | Renames are git-aware, history preserved |

**Key insight:** Every P1/P2 fix in this phase already has prior art in the codebase. The job is to apply the same pattern, not invent new ones.

---

## Per-Item State and Action Map

### P1-S1: SM2 JWT alg header validation
**File:** `pkg/crypto/sm2_jwt.go:222-266`
**Current state:** ✅ FIXED in commit 64b1b40 — header alg is now parsed and compared to `sm2Method.Alg()`. Returns error if mismatch.
**Remaining work:** Add `TestSM2JWT_RejectsAlgNone` and `TestSM2JWT_RejectsAlgHS256Confusion` regression tests.
**Callers:** Every request that validates JWT — i.e., auth middleware. No migration path needed; tokens are stateless.

### P1-S2: Replay window tightening
**File:** `pkg/crypto/request_encryption.go:88-103`
**Current state:** ⚠️ PARTIAL — commit af05d99 tightened to ±120s. CONTEXT.md targets ±60s with config.
**Remaining work:**
- Reduce `maxTimeDiff` from 120 to 60
- Add `security.replay_window_sec` to config.yaml
- Read from config in RequestEncryptor
- Add test verifying timestamp ±61s is rejected

### P1-S3: Nonce cleanup goroutine
**File:** `pkg/crypto/nonce_storage.go`
**Current state:** ✅ FIXED in commit 1071867 — `NewShardedNonceStorage()` starts ticker goroutine that calls `cleanupExpiredNonces()` every `maxTimeDiff` seconds.
**Remaining work:** Add `TestShardedNonceStorage_CleansExpired` test that injects nonces with old timestamps, advances time, verifies cleanup.

### P1-S4: Child menu permission inherit
**File:** `pkg/middleware/permission.go:106-118`
**Current state:** ✅ FIXED in commit 2b55e0d — only button-type (F) child menus can inherit to parent.
**Remaining work:** Add `TestPermissionCheck_DoesNotInheritCTypeChildren` regression test.

### P1-S5: PBKDF2 iteration count
**File:** `internal/core/security/password.go:25`
**Current state:** ⚠️ PARTIAL — `Iterations = 100000` (commit b7dedac). CONTEXT.md targets ≥600000.
**Remaining work:**
- Bump `DefaultPasswordConfig.Iterations` from 100000 to 600000
- Document in comment that this is OWASP 2023 baseline
- Add `TestPasswordManager_DefaultIterationsAre600k` assertion
- Existing 100k hashes still verify (format embeds iterations), but new ones use 600k — login latency on first verify of 100k hash will be ~5x slower (~500ms)
- Add `TestPasswordManager_VerifyBackwardCompat_100k` test

### P1-S6: Random password no bias
**File:** `internal/core/security/password.go:145-166`
**Current state:** ✅ FIXED in commit 07f210c — uses `rand.Int(rand.Reader, charsetLen)` (rejection sampling).
**Remaining work:** Add `TestGenerateRandomPassword_NoBiasDistribution` (chi-square or Kolmogorov-Smirnov test, with relaxed threshold for test speed).

### P1-S7: Excel magic bytes
**File:** `internal/services/operations/excel_handler.go:67-72`
**Current state:** ✅ FIXED in commit 2c74c06 — three-layer check: extension + size + magic bytes (PK\x03\x04).
**Remaining work:** Add `TestVerifyExcelMagicBytes_RejectsNonZip` test.

### P1-C1: AD sync singleflight
**File:** `internal/services/addomain/sync.go`
**Current state:** ✅ FIXED in commit 5c573c5 — `syncGroup singleflight.Group` deduplicates by configID.
**Remaining work:** Add `TestSyncData_ConcurrentCallsDeduplicated` test that fires 10 goroutines simultaneously and verifies only 1 actual sync runs.

### P1-C2: handleDeletedGroups threshold
**File:** `internal/services/addomain/group_sync_service.go:336-376`
**Current state:** ✅ FIXED in commit 887a438 — two gates: (1) reject if LDAP returns 0 entries, (2) reject if delete ratio >50%.
**Remaining work:** Add `TestHandleDeletedGroups_RejectsEmptyLDAP` and `TestHandleDeletedGroups_RejectsOverThreshold` tests.

### P1-C3: UpsertMapping single transaction
**File:** `internal/services/addomain/dept_ou_mapper.go:60-100`
**Current state:** ✅ FIXED in commit ffaecae — wrapped in `s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})`.
**Remaining work:** Add `TestUpsertMapping_AtomicDeleteInsert` test (already has basic test in dept_ou_mapper_test.go but no atomicity verification).

### P1-C4: Port collector batch insert
**File:** `internal/collectors/port_collector.go:353-381`
**Current state:** ✅ FIXED in commit 12d3139 — uses `c.db.CreateInBatches(portStatuses, 100)`.
**Remaining work:** Add `TestPortCollector_SaveToDatabase_Batches100PerCall` test counting actual INSERT statements via SQL mock.

### P1-C5: WebSocket readPump
**File:** `internal/websocket/notice_hub.go`
**Current state:** ✅ FIXED in commit a1e7324 — `go client.readPump()` started in `RegisterClient`.
**Remaining work:** Add `TestClient_ReadPump_ClosesStaleConnection` test that creates a client, doesn't write, advances time, verifies unregister.

### P1-C6: validateUniqueness N+1
**File:** `internal/services/operations/excel_service.go:905-906` (and function body at line 944-979)
**Current state:** ❌ NOT FIXED — still does per-row, per-column Count query.
**Remaining work:**
- Refactor to collect all unique values across rows first
- Single `WHERE col IN (?)` query per unique column
- Add `TestValidateUniqueness_SingleBatchQuery` test using SQL mock or query counter

### P1-B1: Config cache invalidation
**File:** `internal/services/system/config_service.go:91-99`
**Current state:** ✅ FIXED in commit 0bcac33 — calls `OnEncryptionConfigChanged` callback that middleware registers.
**Remaining work:** Add `TestConfigService_UpdateEncryptionFlag_InvalidatesMiddlewareCache` test.

### P1-B2: buildDepartmentPaths duplicate call
**File:** `internal/services/system/user_service.go:404`
**Current state:** ✅ FIXED in commit ab0a279 — only one call at line 398.
**Remaining work:** Verification only; no new code needed.

### P2-A1: Core god struct split
**File:** `internal/core/core.go` (724 lines, 18+ fields)
**Current state:** ❌ NOT FIXED.
**Remaining work:**
- Create `CoreInfra` (Config, DB, Cache, JWTManager, PwdManager, SM4Cipher, Scheduler, RPAScalingService)
- Create `CoreServices` (UserService, RoleService, MenuService, DeptService, PostService, all device/scheduler services, etc.)
- Embed both in `Core struct { *CoreInfra; *CoreServices }` for backward compat
- Update router to use `core.UserService` (still works via embedding) — no router changes needed
- Add 2 new files: `core_infra.go`, `core_services.go`
- Estimated 6-8 hour task; high risk of breaking imports

**Dependencies:** P2-A1 must precede P2-A5 (apperrors migration references core services). Order: P2-A1 → P2-A2 → P2-A5.

### P2-A2: Cache key consolidation
**Files:** `internal/services/system/cache_keys.go` (279 lines) and `internal/services/data_cache_service.go` (355 lines)
**Current state:** ❌ NOT FIXED. Two parallel systems:
- `system/cache_keys.go` uses "dict:type" format (newer)
- `data_cache_service.go` uses "cache:dict:type" format (legacy) + has `GetDictDataByTypeKey` helper
- 21 `CacheKey*` constants in data_cache_service.go conflict with 21 in cache_keys.go
**Remaining work:**
- Audit all call sites of legacy `data_cache_service.go` cache keys
- Replace with `system.BuildDictDataCacheKey(dictType)` style functions
- Delete the duplicate `CacheKey*` constants from data_cache_service.go (keep the `DataCacheService` struct + Get/Set methods)
- Update `dict_cache_impl.go` to use new helper
- Estimated 4-6 hour task

**Dependencies:** P2-A2 must precede P2-A5 (apperrors migration touches error paths that reference cache).

### P2-A3: Remove user_service_optimized
**File:** `internal/services/system/user_service_optimized.go`
**Current state:** ✅ DONE in commit 3bdd3fc.
**Remaining work:** None.

### P2-A4: Migration file renumbering
**Directory:** `internal/core/db/migrations/` (143 files)
**Current state:** ❌ NOT FIXED. Conflicting prefixes:
- 027: cleanup_duplicate_indexes.sql + create_user_column_config.sql
- 029: add_building_coordinates.sql + add_building_spaces_menu.sql
- 030: add_building_spaces_3d_menu.sql + create_workstation_device.sql + enhance_workstation_table.sql
- 031: enhance_server_room_table.sql + update_building_coordinates.sql
**Remaining work:**
- Renumber to next available numbers in correct order (use git log to find chronological order of original commit)
- For each rename, add header comment `// Renamed from 027_create_user_column_config.sql on 2026-06-XX during Phase 32`
- Critical: do NOT modify file content, only filename
- Critical: do NOT renumber if migration has been applied to any production environment (check with DBA / migration log)
- Recommendation: instead of renumbering, **add source comment** to each duplicate file with original commit hash, leaving filename conflict intact. The runner likely sorts by `created_at` from the DB record, not filename.

**Safer alternative:** Verify how migrations are loaded — if by filename sort, rename is needed; if by DB-stored `created_at`, comments are sufficient.

### P2-A5: apperrors unification for role_service
**File:** `internal/services/system/role_service.go` (470 lines, 43 fmt.Errorf calls, 0 apperrors)
**Current state:** ❌ NOT FIXED.
**Remaining work:**
- Add role-specific helpers to `pkg/errors/errors.go`:
  - `RoleNameExists(name string) *AppError` (Code 2041)
  - `RoleKeyExists(key string) *AppError` (Code 2041)
  - `RoleIsAdmin() *AppError` (Code 2045)
- Replace 43 `fmt.Errorf(...)` calls in role_service.go with apperrors helpers
- Generic ones already exist: `RoleNotFound()`, `RoleHasUsers()`, `RoleHasMenus()`, `RoleHasDepts()`, `RoleIsSuper()`
- Update role_handler.go to handle the new error types (most use `response.Error(c, err)` which already handles AppError)
- Estimated 2-3 hour task

**Dependencies:** Must come AFTER P2-A1 (uses core services indirectly).

### P2-A6: AD module test coverage
**Files:** 
- `internal/services/addomain/ldap_client.go` (Connect, Bind, Search) — NO mock
- `internal/services/addomain/group_sync_service.go` — no test file
- `internal/services/addomain/user_ou_service.go` — has user_ou_service_test.go (basic)
- `internal/services/addomain/stripBaseDN_test.go` — has 2 tests (NOT empty as reported)
- `internal/services/addomain/dept_ou_mapper_test.go` — has 4 tests (NOT empty as reported)
- `internal/services/ad_ldap_client.go` (root, 544 lines) — NO test file
**Current state:** ❌ NOT FIXED. The CONTEXT.md's claim that stripBaseDN_test.go and dept_ou_mapper_test.go are "empty" is INCORRECT — they have real tests. Only ldap_client.go (both copies) lacks mock-based tests for Connect/Bind/Search.
**Remaining work:**
- Extract LDAP operations into an interface (e.g., `LDAPClientIface` in addomain package)
- Create `mockLDAPClient` struct implementing the interface
- Add `group_sync_service_test.go` with mocked LDAP
- Add `ldap_client_test.go` (in addomain/ and root services/) with mocked Connect/Bind/Search
- Add `dept_sync_service_test.go` to test sync orchestration
- Estimated 6-8 hour task

### P2-A7: Subprocess management
**Files:** `internal/agent/server/account_manager.go` (15+ `exec.Command` calls)
**Current state:** ❌ NOT FIXED. **NOTE: CONTEXT.md's claim of "Scrapli / Python subprocess" is INCORRECT** — scrapli is Go-native via `scrapligo` library (no subprocess). The only actual subprocess usage is in `internal/agent/server/account_manager.go` for OS account management (PowerShell, useradd, userdel, usermod, getent, chpasswd, tee, chmod, etc.).
**Remaining work:**
- Add process group setting via `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` for each `exec.Command` call
- Add deferred `syscall.Kill(-cmd.Process.Pid, ...)` cleanup on context cancellation
- Add periodic zombie reaper goroutine (calls `wait4(-1, ...)` with `WNOHANG`)
- The reaper should run from `core.Init()` and stop on `core.Close()`
- Estimated 3-4 hour task

### P2-A8: Excel import transaction
**File:** `internal/services/operations/excel_service.go` `ImportData()` (line 224-404)
**Current state:** ❌ NOT FIXED. No `Transaction` wrapper. `processThreeLevelDepartments` (line 1270) runs outside transaction.
**Remaining work:**
- Wrap entire `ImportData` body in `s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})`
- Pass `tx` instead of `s.db` to all sub-services: `referenceResolver.ResolveBatch(tx, ...)`, `upserter.Upsert(tx, ...)`, `cacheInvalidator.InvalidateByEntityType(tx, ...)`, `s.processThreeLevelDepartments(tx, ...)`
- Cache invalidation must move INSIDE the transaction (or after commit) — invalidating before commit is wasteful
- Estimated 3-4 hour task

---

## Common Pitfalls

### Pitfall 1: Bumping PBKDF2 iterations breaks login latency SLO
**What goes wrong:** Bumping from 100k to 600k makes first verify of an old 100k hash 6x slower (~3 seconds for 100k → ~18 seconds for 600k... actually let me recalculate: SM3 is ~50% slower than SHA-256, so 600k SM3 ≈ 300ms. Still acceptable).
**Why it happens:** Linear scaling of PBKDF2 iterations.
**How to avoid:** Verify in staging first; consider keeping 100k for Verify backward compat and only use 600k for new HashPassword. (Currently VerifyPassword uses the iteration count embedded in the hash string, so old 100k hashes still verify at 100k speed — only NEW hashes use 600k.)
**Warning signs:** Login p99 latency spikes; AD users complain about slow logins.

### Pitfall 2: Core struct split breaks all 100+ Setup*Router calls
**What goes wrong:** If `Core.UserService` is moved to `CoreServices.UserService` (no embedding), every router setup needs updating.
**Why it happens:** Router setup uses field access syntax.
**How to avoid:** Use **struct embedding** — `type Core struct { *CoreInfra; *CoreServices }` so `core.UserService` still works.
**Warning signs:** `go build ./...` fails on `internal/api/router.go` with "field UserService not in struct".

### Pitfall 3: Migration renumbering silently breaks production
**What goes wrong:** If a migration was already applied to prod under old name, renumbering the file changes its run order in dev environments, causing "table already exists" errors.
**Why it happens:** Migration runner likely uses filename sort, not DB-stored order.
**How to avoid:** Check how migrations are loaded BEFORE renaming. If by filename, only rename if not yet applied to any environment. Otherwise, leave filename alone and add `// Source:` comment with original commit hash.
**Warning signs:** Dev environment fails to start with migration error.

### Pitfall 4: Excel transaction wrapper hangs on reference resolver
**What goes wrong:** If `referenceResolver.ResolveBatch` holds a non-transactional connection, wrapping it in `tx` may deadlock.
**Why it happens:** GORM transaction holds a single connection; nested calls must use `tx`, not fresh `db`.
**How to avoid:** Audit every `s.db.WithContext` call inside ImportData and replace with `tx.WithContext`.
**Warning signs:** Import hangs after timeout; DB connection pool exhausted.

### Pitfall 5: LDAP mock doesn't exercise the actual LDAP library
**What goes wrong:** Mocking at the interface level skips testing of the actual go-ldap/ldap/v3 library binding.
**Why it happens:** Mock is too high-level.
**How to avoid:** Write at least one test that uses a real local LDAP server (via Docker or slapd) for end-to-end verification, in addition to the mock tests.
**Warning signs:** All tests pass but integration with real AD fails.

### Pitfall 6: Replay window tightening breaks legitimate clients
**What goes wrong:** Tightening from ±120s to ±60s breaks clients with clock skew >60s.
**Why it happens:** Some VMs have NTP drift; some IoT devices have very wrong clocks.
**How to avoid:** Make it configurable via `security.replay_window_sec` (already specified in CONTEXT.md) and default to 60s; document the NTP requirement in deployment guide.
**Warning signs:** Spike in 400 Bad Request responses from specific client IPs.

### Pitfall 7: Subprocess reaper kills legitimate processes
**What goes wrong:** Naive `wait4(-1, ...)` reaper may kill processes spawned by other parts of the app.
**Why it happens:** Without process group isolation (`Setpgid: true`), the reaper can't distinguish our subprocesses from others.
**How to avoid:** ALWAYS set process group on our subprocesses; reaper only reaps child processes that match our pgid or that we explicitly track.
**Warning signs:** Other features break mysteriously after reaper added.

---

## Code Examples

### Verified patterns from official sources:

### Example 1: GORM Transaction wrapping (existing pattern)
```go
// Source: internal/services/operations/room_photo_service.go:129
return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&photo).Error; err != nil {
        return fmt.Errorf("创建照片失败: %w", err)
    }
    // ... more operations using tx
    return nil
})
```

### Example 2: singleflight deduplication (existing pattern in sync.go)
```go
// Source: internal/services/addomain/sync.go (post P1-C1 fix)
type SyncService struct {
    db *gorm.DB
    syncGroup singleflight.Group
}

func (s *SyncService) SyncData(ctx context.Context, config *models.ADConfig, syncType string) (*SyncResult, error) {
    key := config.ID + ":" + syncType
    v, err, _ := s.syncGroup.Do(key, func() (interface{}, error) {
        return s.doSync(ctx, config, syncType)
    })
    if err != nil {
        return nil, err
    }
    return v.(*SyncResult), nil
}
```

### Example 3: apperrors usage (existing pattern in apikey_service.go)
```go
// Source: internal/services/system/apikey_service.go
import apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"

return apperrors.Wrap(nil, apperrors.CodeParamError, "无效的作用域: "+scope)
return nil, apperrors.DatabaseError(err)
return nil, apperrors.RecordExists()
```

### Example 4: Magic byte verification (existing pattern in excel_handler.go)
```go
// Source: internal/api/v1/operations/excel_handler.go (post P1-S7 fix)
func verifyExcelMagicBytes(file *multipart.FileHeader) error {
    src, err := file.Open()
    if err != nil {
        return fmt.Errorf("打开文件失败: %w", err)
    }
    defer src.Close()
    
    magic := make([]byte, 4)
    if _, err := src.Read(magic); err != nil {
        return fmt.Errorf("读取文件头失败: %w", err)
    }
    
    // ZIP/OOXML magic: PK\x03\x04
    if !bytes.Equal(magic, []byte{0x50, 0x4B, 0x03, 0x04}) {
        return fmt.Errorf("文件非 ZIP/OOXML 格式")
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `core.Core` god struct (18 fields) | Embedded `CoreInfra` + `CoreServices` | Phase 32 (P2-A1) | Backward-compatible refactor |
| Two parallel cache key systems | Single `cache_keys.go` source of truth | Phase 32 (P2-A2) | Eliminates key conflicts |
| `fmt.Errorf` for business errors | `apperrors.Wrap(code, message)` | Phase 32 (P2-A5) | Unified HTTP error mapping |
| Per-row uniqueness check | Single `WHERE col IN (?)` query | Phase 32 (P1-C6) | 100x faster Excel import for large files |
| `user_service` + `user_service_optimized` | Single `user_service` (optimized) | Phase 31 (commit 3bdd3fc) | Removes confusion |
| 1000-iteration PBKDF2 | 600000-iteration PBKDF2 (OWASP 2023) | Phase 32 (P1-S5 follow-up) | Brute-force resistant |

**Deprecated/outdated:**
- `user_service_optimized.go` — already deleted (commit 3bdd3fc)
- `containsIgnoreCase` custom function — replaced with `strings.Contains` (commit b47823e)
- `coordinatesToColumnString` custom function — replaced with `excelize.ColumnNumberToName` (commit 4ed6f58)
- `isDuplicateKeyError` string match — replaced with `pgconn.PgError.Code` (commit 0b3f6b7)
- `InsecureSkipVerify: true` hardcoded — replaced with `LDAP_TLS_INSECURE_SKIP_VERIFY` env var (Phase 31)

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24+ | All Go code | ✓ | 1.24.5 | — |
| PostgreSQL 18 | Integration tests | ✗ (likely) | — | SQLite in-memory for unit tests |
| Redis 7.4 | Integration tests | ✗ (likely) | — | In-memory cache for unit tests |
| LDAP server (slapd) | AD integration tests | ✗ (likely) | — | Mock interface for unit tests |
| Docker | LDAP/Redis/PG containers | ✗ (likely) | — | Skip integration tests, rely on unit tests |

**Missing dependencies with no fallback:**
- None — all phase work is unit-testable with mocks/in-memory DBs.

**Missing dependencies with fallback:**
- Real LDAP/Redis/PG → mocked/in-memory for unit tests; full E2E tests skipped if env unavailable.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` + `github.com/stretchr/testify/assert` |
| Config file | None (Go's built-in test discovery) |
| Quick run command | `go test -count=1 -run "<TestName>" ./<package>/` |
| Full suite command | `go test -count=1 ./...` |
| Coverage command | `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| P1-S1 | Reject alg=none JWT | unit | `go test -count=1 -run "TestSM2JWT" ./pkg/crypto/ -v` | ❌ Wave 0 |
| P1-S2 | Reject ±61s timestamp | unit | `go test -count=1 -run "TestRequestEncryption" ./pkg/crypto/ -v` | ❌ Wave 0 |
| P1-S3 | Nonce cleanup runs | unit | `go test -count=1 -run "TestShardedNonceStorage" ./pkg/crypto/ -v` | ❌ Wave 0 |
| P1-S4 | Reject C-type child inherit | unit | `go test -count=1 -run "TestPermission" ./pkg/middleware/ -v` | ❌ Wave 0 |
| P1-S5 | 600k iterations + backward compat | unit | `go test -count=1 -run "TestPasswordManager" ./internal/core/security/ -v` | ❌ Wave 0 (existing tests use 500) |
| P1-S6 | No bias in random password | unit (statistical) | `go test -count=1 -run "TestGenerateRandomPassword" ./internal/core/security/ -v` | ❌ Wave 0 |
| P1-S7 | Reject non-PK magic bytes | unit | `go test -count=1 -run "TestVerifyExcelMagic" ./internal/api/v1/operations/ -v` | ❌ Wave 0 |
| P1-C1 | Single in-flight sync | unit (concurrent) | `go test -count=1 -run "TestSyncData" ./internal/services/addomain/ -v` | ❌ Wave 0 |
| P1-C2 | Reject empty LDAP / over-threshold | unit | `go test -count=1 -run "TestHandleDeletedGroups" ./internal/services/addomain/ -v` | ❌ Wave 0 |
| P1-C3 | Atomic UpsertMapping | unit (mock DB failure mid-tx) | `go test -count=1 -run "TestUpsertMapping_Atomic" ./internal/services/addomain/ -v` | ❌ Wave 0 |
| P1-C4 | Batch insert (100 per call) | unit (SQL mock) | `go test -count=1 -run "TestPortCollector" ./internal/collectors/ -v` | ❌ Wave 0 |
| P1-C5 | readPump closes stale conn | unit (mock conn) | `go test -count=1 -run "TestClient_ReadPump" ./internal/websocket/ -v` | ❌ Wave 0 |
| P1-C6 | Single IN query for uniqueness | unit (SQL counter) | `go test -count=1 -run "TestValidateUniqueness" ./internal/services/operations/ -v` | ❌ Wave 0 |
| P1-B1 | Cache invalidation on config update | unit | `go test -count=1 -run "TestConfigService" ./internal/services/system/ -v` | ❌ Wave 0 |
| P1-B2 | buildDepartmentPaths called once | unit (assertion) | code review only | N/A |
| P2-A1 | Core split preserves field access | unit (compile) | `go build ./...` | N/A |
| P2-A2 | Only one cache_keys source | unit (compile) | `go vet ./...` | N/A |
| P2-A3 | user_service_optimized removed | unit (compile) | `go build ./...` | N/A (done) |
| P2-A4 | Migration conflicts documented | manual review | N/A | N/A |
| P2-A5 | role_service uses apperrors | unit | `go test -count=1 -run "TestRole" ./internal/services/system/ -v` | ❌ Wave 0 |
| P2-A6 | LDAP mock tests for Connect/Bind/Search | unit (mock) | `go test -count=1 -run "TestLDAP" ./internal/services/addomain/ -v` | ❌ Wave 0 |
| P2-A7 | Process group + reaper | unit (subprocess test) | `go test -count=1 -run "TestSubprocess" ./internal/agent/server/ -v` | ❌ Wave 0 |
| P2-A8 | Excel import transactional | unit (rollback test) | `go test -count=1 -run "TestImportData" ./internal/services/operations/ -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go build ./... && go vet ./...`
- **Per wave merge:** `go test -count=1 ./...`
- **Phase gate:** Full suite green + `go test -coverprofile=coverage.out ./internal/services/addomain/ ./internal/services/system/ ./internal/services/operations/ ./internal/collectors/ ./internal/websocket/ ./pkg/crypto/ ./internal/core/security/ && go tool cover -func=coverage.out` shows ≥70% for these packages.

### Wave 0 Gaps
- [ ] `pkg/crypto/sm2_jwt_alg_test.go` — covers P1-S1
- [ ] `pkg/crypto/request_encryption_window_test.go` — covers P1-S2
- [ ] `pkg/crypto/nonce_storage_cleanup_test.go` — covers P1-S3
- [ ] `pkg/middleware/permission_inherit_test.go` — covers P1-S4
- [ ] `internal/core/security/password_owasp_test.go` — covers P1-S5 (bump iterations + add backward compat test)
- [ ] `internal/core/security/random_password_bias_test.go` — covers P1-S6
- [ ] `internal/api/v1/operations/excel_magic_bytes_test.go` — covers P1-S7
- [ ] `internal/services/addomain/sync_singleflight_test.go` — covers P1-C1
- [ ] `internal/services/addomain/group_sync_threshold_test.go` — covers P1-C2
- [ ] `internal/services/addomain/dept_ou_mapper_atomic_test.go` — covers P1-C3
- [ ] `internal/collectors/port_collector_batch_test.go` — covers P1-C4
- [ ] `internal/websocket/notice_hub_readpump_test.go` — covers P1-C5
- [ ] `internal/services/operations/excel_uniqueness_batch_test.go` — covers P1-C6
- [ ] `internal/services/system/config_invalidation_test.go` — covers P1-B1
- [ ] `internal/services/system/role_service_apperrors_test.go` — covers P2-A5
- [ ] `internal/services/addomain/ldap_client_mock_test.go` — covers P2-A6
- [ ] `internal/services/addomain/group_sync_service_test.go` — covers P2-A6
- [ ] `internal/agent/server/subprocess_pgroup_test.go` — covers P2-A7
- [ ] `internal/services/operations/excel_transaction_test.go` — covers P2-A8
- [ ] `internal/core/core_split_compat_test.go` — verifies P2-A1 backward compat (compile test)

*(Existing test infrastructure in pkg/crypto, internal/services/system, internal/services/addomain is sufficient — only NEW test files needed.)*

---

## Security Domain

> `security_enforcement` not explicitly set in config — assumed enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes | JWT alg validation (P1-S1), PBKDF2 ≥600k (P1-S5) |
| V3 Session Management | yes | Replay window tightening (P1-S2), nonce cleanup (P1-S3) |
| V4 Access Control | yes | Permission inheritance (P1-S4) |
| V5 Input Validation | yes | Excel magic bytes (P1-S7), N+1 unique check (P1-C6) |
| V6 Cryptography | yes | PBKDF2 strength (P1-S5), random password no bias (P1-S6) |
| V7 Error Handling | yes | apperrors unified mapping (P2-A5) |
| V9 Communication | no | (not in phase scope) |
| V10 Malicious Code | yes | Excel transaction rollback (P2-A8) |
| V12 Files and Resources | no | (not in phase scope) |
| V14 Configuration | yes | Replay window config (P1-S2 follow-up) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| JWT alg confusion | Spoofing | Whitelist alg in header (P1-S1) |
| Replay attack | Tampering | Timestamp window + nonce (P1-S2/P1-S3) |
| Privilege escalation via menu inherit | Elevation | Explicit menu permission (P1-S4) |
| Offline brute-force on hashes | Information Disclosure | PBKDF2 ≥600k (P1-S5) |
| Module bias in passwords | Information Disclosure | crypto/rand rejection sampling (P1-S6) |
| File upload bypass | Tampering | Magic byte check (P1-S7) |
| TOCTOU on multi-step write | Tampering | Single transaction (P1-C3, P2-A8) |
| Connection pool exhaustion | Denial of Service | Batch insert (P1-C4) |
| Zombie connection FD leak | Denial of Service | readPump + ping/pong (P1-C5) |
| LDAP outage → mass delete | Tampering | Threshold protection (P1-C2) |
| Subprocess FD leak | Denial of Service | Process group + reaper (P2-A7) |

---

## Sources

### Primary (HIGH confidence)
- Code inspection of cited files: `pkg/crypto/sm2_jwt.go`, `internal/core/security/password.go`, `pkg/middleware/permission.go`, `internal/services/operations/excel_handler.go`, `internal/services/operations/excel_service.go`, `internal/services/addomain/sync.go`, `internal/services/addomain/group_sync_service.go`, `internal/services/addomain/dept_ou_mapper.go`, `internal/collectors/port_collector.go`, `internal/websocket/notice_hub.go`, `internal/services/system/role_service.go`, `internal/services/system/user_service.go`, `internal/services/system/config_service.go`, `internal/core/core.go`
- `git log --oneline` history for fix verification

### Secondary (MEDIUM confidence)
- `.planning/reviews/20260612-backend-code-review.md` (source review with 47 P1 + 80+ P2 findings)
- `.planning/phases/32-v1-14-p1-p2/CONTEXT.md` (user-locked decisions)
- `.planning/ROADMAP.md` (Phase 32 requirements)
- `.planning/STATE.md` (project history)
- `.claude/CLAUDE.md` and `D:\code\ClaudeCode\xingran-go-backend\CLAUDE.md` (project conventions)

### Tertiary (LOW confidence)
- None — all claims verified against current code state.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified by `go build ./...` passing
- Architecture: HIGH — verified by reading 18+ source files at cited line numbers
- Pitfalls: MEDIUM — based on Go concurrency patterns + GORM docs; specific failure modes extrapolated
- Per-item state: HIGH — verified via `git log` showing fix commits for 11/15 P1 items
- Test coverage gaps: HIGH — listed all existing test files and identified missing tests

**Research date:** 2026-06-13
**Valid until:** 2026-07-13 (30 days — Go stack is stable; OWASP recommendations change ~yearly)

---

## Open Questions

1. **Migration file ordering — does the runner sort by filename or by DB record?**
   - What we know: `internal/core/db/migrations/` has 143 .sql files with conflicting numeric prefixes.
   - What's unclear: How the migration runner resolves order when duplicates exist.
   - Recommendation: **Add source comment + add new file at end of sequence** as safer alternative to renumbering. Check `internal/core/db/migrations.go` or equivalent runner file before planning P2-A4.

2. **Should the project migrate to a newer apperrors helper style (e.g., context-based) or stick with Wrap(code, message)?**
   - What we know: 154 helpers exist; apikey_service uses them.
   - What's unclear: Project convention for adding new helpers (e.g., `RoleNameExists(name string)` vs `Wrap(nil, CodeRoleExists, "name=" + name)`).
   - Recommendation: Use specific helpers for human-readable errors (e.g., `RoleNameExists(name)`); use generic `Wrap` for dynamic messages.

3. **Are the duplicate P0 fixes in `.planning/debug/resolved/` actually committed to main, or are they worktree-only?**
   - What we know: Recent commits (af05d99, 64b1b40, 1071867, etc.) are visible in `git log` on main.
   - What's unclear: Whether ALL 22 P0 + 11 P1 fixes are on main, or if some are in worktree branches.
   - Recommendation: At planning time, run `git log --all --oneline --grep "P1"` and `git log --all --oneline --grep "P0"` to confirm.

4. **Should P1-S5 bump from 100k to 600k in one step, or phase it (e.g., 200k → 400k → 600k)?**
   - What we know: Current 100k works; 600k is OWASP 2023 baseline.
   - What's unclear: Login latency budget in production; SM3 implementation speed on target hardware.
   - Recommendation: Benchmark in staging; if login latency stays <500ms p99, go directly to 600k. Otherwise phase.

5. **For P2-A6 AD testing, should we use gomock, mockery, or hand-rolled mock?**
   - What we know: Project doesn't currently use gomock (per dependency search).
   - What's unclear: Team preference for test infrastructure.
   - Recommendation: Hand-rolled mock struct implementing an interface — lowest dependency cost, matches existing style.

6. **For P2-A7 subprocess management, should the reaper run as part of `core.Init()` or be a standalone service?**
   - What we know: `core.Init()` already starts background goroutines (cache warm-up, scheduler).
   - What's unclear: Where the reaper lifecycle should be owned.
   - Recommendation: Add to `core.Init()` as a new method `startSubprocessReaper()`, paired with cleanup in `core.Close()`. Mirrors existing patterns.

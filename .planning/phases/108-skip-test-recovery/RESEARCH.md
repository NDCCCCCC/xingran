# Phase 108 Research: SKIP Test Recovery

## SKIP-01: HybridAuthenticator Interface Refactor

### Current HybridAuthenticator Struct

```go
// internal/core/security/hybrid_authenticator.go
type HybridAuthenticator struct {
    localAuth *LocalAuthenticator  // concrete type
    adAuth    *ADAuthenticator     // concrete type
}

func NewHybridAuthenticator(local *LocalAuthenticator, ad *ADAuthenticator) *HybridAuthenticator
func (h *HybridAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)
func (h *HybridAuthenticator) Name() string
```

**The 5 Skipped Test Cases** (in `hybrid_authenticator_test.go`):

| Line | Test Function | Skip Reason |
|------|---------------|-------------|
| 12 | `TestHybridAuthenticator_Name` | "TODO: WIP - 等待 HybridAuthenticator 支持 interface 参数后恢复测试" |
| 24 | `TestHybridAuthenticator_Authenticate_LocalSuccess` | "TODO: WIP - mockAuthenticator 无法在具体类型 LocalAuthenticator/ADAuthenticator 场景下工作；等待 refactor" |
| 74 | `TestHybridAuthenticator_Authenticate_FallbackToAD` | Same as above |
| 120 | `TestHybridAuthenticator_Authenticate_BothFailed` | Same as above |
| 153 | `TestHybridAuthenticator_TableDrivenTests` | Same as above |

### Root Cause

`HybridAuthenticator` holds **concrete types** (`*LocalAuthenticator`, `*ADAuthenticator`), not interfaces. The tests have a `mockAuthenticator` type (defined in `authenticator_test.go`) that implements the `Authenticator` interface, but `NewHybridAuthenticator` requires specific concrete types.

The `Authenticator` interface already exists:

```go
// internal/core/security/authenticator.go:12
type Authenticator interface {
    Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)
    Name() string
}
```

### Why Current Design is Hard to Test

The test wants to inject a mock authenticator to control `Authenticate` behavior without hitting real DB/LDAP. But because `HybridAuthenticator` embeds concrete types, not interfaces, the test cannot substitute mocks.

### Recommended Interface Design

**Option A (Recommended):** Change `HybridAuthenticator` to accept interfaces:

```go
type HybridAuthenticator struct {
    localAuth Authenticator  // interface instead of *LocalAuthenticator
    adAuth    Authenticator  // interface instead of *ADAuthenticator
}

func NewHybridAuthenticator(local, ad Authenticator) *HybridAuthenticator
```

This requires:
1. Changing `localAuth` and `adAuth` fields from concrete to interface type `Authenticator`
2. Updating `Authenticate` method to call via interface
3. Updating `AuthStrategyFactory.GetAuthenticator("hybrid")` to pass concrete `*LocalAuthenticator` / `*ADAuthenticator` where they satisfy the interface
4. Restoring the 5 test cases by injecting `mockAuthenticator` or equivalent mocks

**Option B:** Add a constructor that accepts interfaces (dual constructor pattern):

```go
func NewHybridAuthenticatorWithInterfaces(local, ad Authenticator) *HybridAuthenticator
```

Keep existing `NewHybridAuthenticator` for backward compatibility, add new interface-based constructor.

### Estimated Effort

- Interface refactor: **Medium** — changes to `hybrid_authenticator.go` constructor + fields, `AuthStrategyFactory`, and test restoration
- No changes needed to `LocalAuthenticator` or `ADAuthenticator` themselves (they already implement `Authenticator` interface)
- Verification: `go test ./internal/core/security/ -run "TestHybrid"` should pass without Skip

---

## SKIP-02: 10+ Skip Test Restoration

### Complete Table of Skipped Tests

| File | Line | Test Function | Skip Reason | Can Restore with Embedded? | Needs Real Env |
|------|------|---------------|-------------|---------------------------|----------------|
| `internal/core/security/local_authenticator_test.go` | 17 | `TestLocalAuthenticator_Authenticate_Success` | "测试数据库未配置" | **YES** (sqlite :memory:) | No |
| `internal/core/security/local_authenticator_test.go` | 45 | `TestLocalAuthenticator_Authenticate_UserNotFound` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/local_authenticator_test.go` | 65 | `TestLocalAuthenticator_Authenticate_InvalidPassword` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/local_authenticator_test.go` | 88 | `TestLocalAuthenticator_Authenticate_UserDisabled` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/local_authenticator_test.go` | 111 | `TestLocalAuthenticator_Authenticate_SM3PasswordVerification` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/local_authenticator_test.go` | 151 | `TestLocalAuthenticator_Name` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/local_authenticator_test.go` | 164 | `TestLocalAuthenticator_TableDrivenTests` | "测试数据库未配置" | **YES** | No |
| `internal/core/security/authenticator_test.go` | 36 | `setupTestDB` (helper) | "测试数据库配置未实现" | N/A (helper) | — |
| `internal/core/security/ad_authenticator_test.go` | 67 | `TestADAuthenticator_Authenticate_Success` | "需要真实 DB + LDAP 测试环境" | **PARTIAL** (LDAP fake server available) | LDAP |
| `internal/core/security/ad_authenticator_test.go` | 108 | `TestADAuthenticator_Authenticate_ConfigNotFound` | "需要真实 DB + LDAP 测试环境" | **YES** (no LDAP needed, just nil config) | No |
| `internal/core/security/ad_authenticator_test.go` | 133 | `TestADAuthenticator_Authenticate_TableDrivenTests` | "需要真实 DB + LDAP 测试环境" | **PARTIAL** (needs fake LDAP) | LDAP |
| `internal/core/security/ad_authenticator_test.go` | 214 | `TestADAuthenticator_NeedsSyncFlag` | "需要真实 DB + LDAP 测试环境" | **PARTIAL** | LDAP |
| `internal/core/security/ad_authenticator_test.go` | 244 | `TestADAuthenticator_IntegrationTest` | "集成测试需要真实AD环境配置" | **NO** | Real AD |
| `internal/core/security/ad_authenticator_test.go` | 253 | `TestADAuthenticator_IntegrationTest` | "集成测试需要真实AD环境配置" | **NO** | Real AD |
| `internal/services/system/user_sync_service_test.go` | 15 | `setupTestDBForSync` (helper) | "测试数据库配置未实现" | N/A (helper) | — |
| `internal/services/system/user_sync_service_test.go` | 23 | `TestUserSyncService_SyncUserFromAD_FirstTime` | "测试数据库未配置" | **YES** (sqlite :memory:) | No |
| `internal/services/system/user_sync_service_test.go` | 55 | `TestUserSyncService_SyncUserFromAD_UpdateExisting` | "测试数据库未配置" | **YES** | No |
| `internal/services/system/user_sync_service_test.go` | 101 | `TestUserSyncService_SyncUserFromAD_TransactionRollback` | "测试数据库未配置" | **YES** | No |
| `internal/services/system/user_sync_service_test.go` | 131 | `TestUserSyncService_SyncUserFromAD_RoleAssignment` | "测试数据库未配置" | **YES** | No |
| `internal/services/system/user_sync_service_test.go` | 171 | `TestUserSyncService_SyncUserFromAD_DepartmentAssignment` | "测试数据库未配置" | **YES** | No |
| `internal/services/system/user_sync_service_test.go` | 214 | `TestUserSyncService_SyncUserFromAD_TableDrivenTests` | "测试数据库未配置" | **YES** | No |

### Analysis by Category

#### Category A: Can Restore with sqlite :memory: (8 tests)

**`local_authenticator_test.go` (7 tests):** All skipped due to `setupTestDB(t)` returning nil. This is a simple pattern — replace `setupTestDB(t)` with a real sqlite `:memory:` database setup. The test helper at line 36 of `authenticator_test.go` also needs implementation.

**`user_sync_service_test.go` (6 tests):** All skipped due to `setupTestDBForSync(t)` returning nil. Same pattern — implement with sqlite `:memory:`.

#### Category B: Can Restore with Fake LDAP Server (3 tests)

**`ad_authenticator_test.go` — partial restore:** v1.27 established a precedent with `ldap_fake_server_78_07_test.go` which provides a `fakeLDAPServer78` — an in-process BER-encoded LDAP responder. This can be reused for `ad_authenticator_test.go`.

- `TestADAuthenticator_Authenticate_Success` (line 67): Could use fake LDAP server
- `TestADAuthenticator_Authenticate_TableDrivenTests` (line 133): Could use fake LDAP server
- `TestADAuthenticator_NeedsSyncFlag` (line 214): Could use fake LDAP server

**Limitation:** `TestADAuthenticator_Authenticate_ConfigNotFound` (line 108) doesn't need LDAP — only needs `NewADAuthenticator(nil, "nonexistent")` to test the config-not-found path. This one is **restorable without any fake LDAP**.

#### Category C: Needs Real Environment (2 tests)

- `TestADAuthenticator_IntegrationTest` (lines 244, 253): Explicitly requires real AD environment. These should go to **HUMAN-UAT**.

### v1.27 Embedded LDAP Precedent

Found in `internal/services/addomain/ldap_fake_server_78_07_test.go`:

```go
// fakeLDAPServer78 is a minimal in-process LDAP responder using raw BER encoding
type fakeLDAPServer78 struct {
    ln         net.Listener
    port       int
    addr       string
    bindCount  int
    bindResult int
    entries    []*ldapSearchEntry
    mu         sync.Mutex
}

func newFakeLDAPServer(t *testing.T) *fakeLDAPServer78 {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    // ... binds to random available port
}
```

This was used for Phase 78 to test LDAP client code coverage without a real LDAP server. The same pattern can be applied to `ad_authenticator_test.go` — spawn a fake LDAP server on a random port and configure `ADAuthenticator` to use it.

---

## Summary & Recommendations

### SKIP-01: HybridAuthenticator Interface Refactor

- **Goal:** Restore 5 skipped tests in `hybrid_authenticator_test.go`
- **Approach:** Change `HybridAuthenticator` struct fields from concrete types to `Authenticator` interface
- **Impact:** Minimal — `LocalAuthenticator` and `ADAuthenticator` already implement the interface
- **Estimated effort:** Medium

### SKIP-02: 10+ Skip Tests Restoration

**Restorable with sqlite :memory: (13 tests):**
- 7 `local_authenticator_test.go` tests
- 6 `user_sync_service_test.go` tests

**Restorable with fake LDAP (3 tests) + no LDAP needed (1 test):**
- `ad_authenticator_test.go` line 108 (ConfigNotFound) — no LDAP needed
- `ad_authenticator_test.go` lines 67, 133, 214 — need fake LDAP server (reuse Phase 78 pattern)

**Needs HUMAN-UAT (2 tests):**
- `ad_authenticator_test.go` lines 244, 253 — IntegrationTest requiring real AD

### Key Files to Modify

1. `internal/core/security/hybrid_authenticator.go` — interface refactor
2. `internal/core/security/auth_strategy_factory.go` — pass interfaces
3. `internal/core/security/hybrid_authenticator_test.go` — restore 5 tests
4. `internal/core/security/authenticator_test.go` — implement `setupTestDB`
5. `internal/core/security/local_authenticator_test.go` — use real sqlite DB
6. `internal/core/security/ad_authenticator_test.go` — add fake LDAP setup
7. `internal/services/system/user_sync_service_test.go` — implement `setupTestDBForSync`

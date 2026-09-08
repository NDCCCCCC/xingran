---
phase: 108
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/core/security/hybrid_authenticator.go
  - internal/core/security/hybrid_authenticator_test.go
  - internal/core/security/auth_strategy_factory.go
autonomous: true
requirements_addressed: [SKIP-01]
---

<objective>
Refactor HybridAuthenticator struct fields from concrete types (`*LocalAuthenticator`, `*ADAuthenticator`) to the `Authenticator` interface type, enabling mock injection in tests. This restores the 5 skipped tests in `hybrid_authenticator_test.go`.
</objective>

<tasks>

## Task 1: Update HybridAuthenticator struct and constructor

**File:** `internal/core/security/hybrid_authenticator.go`

Change the `HybridAuthenticator` struct fields from concrete types to the `Authenticator` interface:

```go
// OLD
type HybridAuthenticator struct {
    localAuth *LocalAuthenticator
    adAuth    *ADAuthenticator
}

func NewHybridAuthenticator(local *LocalAuthenticator, ad *ADAuthenticator) *HybridAuthenticator

// NEW
type HybridAuthenticator struct {
    localAuth Authenticator  // interface
    adAuth    Authenticator // interface
}

func NewHybridAuthenticator(local, ad Authenticator) *HybridAuthenticator
```

The `Authenticate` method calls `h.localAuth.Authenticate(...)` and `h.adAuth.Authenticate(...)` — these work identically through the interface. No logic changes needed.

## Task 2: Verify LocalAuthenticator and ADAuthenticator implement Authenticator

**Files:** `internal/core/security/local_authenticator.go`, `internal/core/security/ad_authenticator.go`

Confirm both types already implement `Authenticator` (interface `{ Authenticate(ctx, req) (*AuthResult, error); Name() string }`):
- `LocalAuthenticator` has `Authenticate` and `Name` methods — YES, implements interface
- `ADAuthenticator` has `Authenticate` and `Name` methods — YES, implements interface

No code changes needed here; this is a verification task.

## Task 3: Update AuthStrategyFactory.GetAuthenticator("hybrid")

**File:** `internal/core/security/auth_strategy_factory.go`

The `case "hybrid":` branch already passes `local` (`*LocalAuthenticator`) and `ad` (`*ADAuthenticator`) to `NewHybridAuthenticator`. Since both concrete types satisfy the `Authenticator` interface, the call site requires **no changes**:

```go
case "hybrid":
    local := NewLocalAuthenticator(f.db, f.pwdManager)
    // ... ad setup ...
    return NewHybridAuthenticator(local, ad), nil  // already compatible with interface params
```

The existing code is already compatible. The refactor in Task 1 makes the constructor accept interfaces, and the factory passes types that implement it.

## Task 4: Restore skipped tests in hybrid_authenticator_test.go

**File:** `internal/core/security/hybrid_authenticator_test.go`

Remove `t.Skip(...)` from 5 tests and implement the actual test logic using `mockAuthenticator`. The `mockAuthenticator` type is already defined in `authenticator_test.go` in the same package.

### Test 1: `TestHybridAuthenticator_Name` (line 12)

Remove Skip, use `mockAuthenticator` for both fields:

```go
func TestHybridAuthenticator_Name(t *testing.T) {
    mockLocal := &mockAuthenticator{nameFunc: func() string { return "local" }}
    mockAD := &mockAuthenticator{nameFunc: func() string { return "ad" }}

    auth := NewHybridAuthenticator(mockLocal, mockAD)
    assert.Equal(t, "hybrid", auth.Name())
}
```

### Test 2: `TestHybridAuthenticator_Authenticate_LocalSuccess` (line 24)

Remove Skip, implement:
- `mockLocal` returns success result
- `mockAD` returns error (unreachable since local succeeds)
- Assert local result, `NeedsSync=false`, AD not called

```go
func TestHybridAuthenticator_Authenticate_LocalSuccess(t *testing.T) {
    mockLocal := &mockAuthenticator{
        authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
            return &AuthResult{
                User:       &UserResult{Username: "localuser"},
                AuthSource: "local",
                NeedsSync:   false,
            }, nil
        },
    }
    mockAD := &mockAuthenticator{
        authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
            t.Error("AD认证器不应该被调用")
            return nil, ErrUserNotFound
        },
    }

    auth := NewHybridAuthenticator(mockLocal, mockAD)
    req := MockAuthRequest("localuser", "password")

    result, err := auth.Authenticate(context.Background(), req)

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "local", result.AuthSource)
    assert.False(t, result.NeedsSync)
    assert.False(t, mockAD.called, "AD认证器不应该被调用")
}
```

### Test 3: `TestHybridAuthenticator_Authenticate_FallbackToAD` (line 74)

Remove Skip:
- `mockLocal` returns `ErrUserNotFound`
- `mockAD` returns success result with `NeedsSync=true`
- Assert AD result, `NeedsSync=true`, both called

### Test 4: `TestHybridAuthenticator_Authenticate_BothFailed` (line 120)

Remove Skip:
- `mockLocal` returns `ErrUserNotFound`
- `mockAD` returns `ErrInvalidCredentials`
- Assert error returned, both called

### Test 5: `TestHybridAuthenticator_TableDrivenTests` (line 153)

Remove Skip, use table-driven approach with 3 cases:
1. Local success, no AD call
2. Local fail, AD success
3. Both fail

## Task 5: Verify

Run the hybrid authenticator tests:

```bash
go test ./internal/core/security/ -run "TestHybrid" -v
```

All 5 tests should pass without Skip.

</tasks>

<success_criteria>
- `go test ./internal/core/security/ -run "TestHybrid" -v` passes without Skip
- `go test ./internal/core/security/ -run "TestHybridAuthenticator_Name" -v` passes
- `go test ./internal/core/security/ -run "TestHybridAuthenticator_Authenticate" -v` passes
- `go test ./internal/core/security/ -run "TestHybridAuthenticator_TableDrivenTests" -v` passes
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>

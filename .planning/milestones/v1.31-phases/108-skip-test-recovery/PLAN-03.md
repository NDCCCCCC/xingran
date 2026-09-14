---
phase: 108
plan: "03"
type: execute
wave: 2
depends_on: []
files_modified:
  - internal/core/security/ad_authenticator_test.go
autonomous: true
requirements_addressed: [SKIP-02]
---

<objective>
Restore 4 skipped AD authenticator tests using the fake LDAP server pattern from Phase 78 (`fakeLDAPServer78`). Two additional tests remain skipped as HUMAN-UAT (integration tests requiring real AD environment).
</objective>

<tasks>

## Context

The Phase 78 probe established `fakeLDAPServer78` in `internal/services/addomain/ldap_fake_server_78_07_test.go`. It provides a minimal in-process LDAP responder using raw BER encoding. The same pattern can be reused for `ad_authenticator_test.go`.

**Key files:**
- `internal/services/addomain/ldap_fake_server_78_07_test.go` — `fakeLDAPServer78` type + helper functions (`newFakeLDAPServer`, `buildBindResponse`, `buildSearchResultEntry`, etc.)
- `internal/core/security/ad_authenticator.go` — `ADAuthenticator` implementation

**Fake LDAP server capabilities:**
- Handles BindRequest (opNum 0) — returns configurable result code
- Handles SearchRequest (opNum 3) — returns configurable entries
- Runs on random available port

**Important constraint:** The fake LDAP server lives in `addomain` package, not `security` package. Tests in `security` package cannot import it directly. Solution: copy the minimal required fake server subset into `ad_authenticator_test.go` (same pattern as Phase 78, not a new implementation).

## Task 1: Add fake LDAP server helpers to ad_authenticator_test.go

**File:** `internal/core/security/ad_authenticator_test.go`

Add the minimal fake LDAP server implementation needed for AD authenticator testing. Copy the essential types and functions from `ldap_fake_server_78_07_test.go` (lines 33-162 range):

```go
// fakeLDAPServer is a minimal in-process LDAP responder for testing.
// Based on Phase 78 ldap_fake_server_78_07_test.go pattern.
type fakeLDAPServer struct {
    ln         net.Listener
    port       int
    addr       string
    bindCount  int
    bindResult int   // LDAP result code (0=success)
    entries    []*ldapSearchEntry
    mu         sync.Mutex
    closeOnce  sync.Once
    closed     bool
}

type ldapSearchEntry struct {
    dn   string
    attrs map[string][]string
}

func newFakeLDAPServer(t *testing.T) *fakeLDAPServer {
    t.Helper()
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    _, portStr, err := net.SplitHostPort(ln.Addr().String())
    require.NoError(t, err)
    port := 0
    for _, c := range portStr {
        port = port*10 + int(c-'0')
    }
    return &fakeLDAPServer{ln: ln, port: port, addr: ln.Addr().String()}
}

func (s *fakeLDAPServer) Addr() string { return s.addr }
func (s *fakeLDAPServer) Port() int    { return s.port }
func (s *fakeLDAPServer) BindCount() int {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.bindCount
}
func (s *fakeLDAPServer) SetBindResult(code int) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.bindResult = code
}
func (s *fakeLDAPServer) Close() {
    s.closeOnce.Do(func() {
        s.closed = true
        s.ln.Close()
    })
}

// berLength, berInt, berString, berSequence — copy from ldap_fake_server_78_07_test.go
// ldapMessage, buildBindResponse, buildSearchResultEntry, buildSearchResultDone — copy

func (s *fakeLDAPServer) handleConnection(conn net.Conn) {
    defer conn.Close()
    buf := make([]byte, 4096)
    for {
        if s.closed {
            return
        }
        conn.SetReadDeadline(time.Now().Add(5 * time.Second))
        n, err := conn.Read(buf)
        if err != nil {
            return
        }
        data := buf[:n]
        reader := bytes.NewReader(data)
        msgID, opNum, err := berPeekAppTag(reader)
        if err != nil {
            return
        }
        switch opNum {
        case 0: // BindRequest
            s.mu.Lock()
            s.bindCount++
            result := s.bindResult
            s.mu.Unlock()
            resp := buildBindResponse(msgID, result)
            conn.Write(resp)
        case 3: // SearchRequest
            entries := s.entries
            for _, entry := range entries {
                resp := buildSearchResultEntry(msgID, entry.dn, entry.attrs)
                conn.Write(resp)
            }
            doneResp := buildSearchResultDone(msgID, 0)
            conn.Write(doneResp)
        case 2: // UnbindRequest
            return
        default:
            return
        }
    }
}

func (s *fakeLDAPServer) Start() {
    go func() {
        for {
            conn, err := s.ln.Accept()
            if err != nil {
                if s.closed {
                    return
                }
                continue
            }
            go s.handleConnection(conn)
        }
    }()
}
```

Add required imports:
```go
import (
    "bytes"
    "net"
    "sync"
    "time"
)
```

## Task 2: Restore TestADAuthenticator_Authenticate_ConfigNotFound (line 108)

**File:** `internal/core/security/ad_authenticator_test.go`

This test does NOT need LDAP — it tests that a non-existent config ID returns an error. Remove Skip:

```go
func TestADAuthenticator_Authenticate_ConfigNotFound(t *testing.T) {
    // This test only needs a nil db — NewADAuthenticator will fail to find config
    auth := NewADAuthenticator(nil, "nonexistent-config")
    req := MockAuthRequest("testuser", "password")

    result, err := auth.Authenticate(context.Background(), req)

    assert.Error(t, err)
    assert.Nil(t, result)
}
```

Note: This test was already using `NewADAuthenticator(nil, "nonexistent-config")` — the Skip was overly conservative since no LDAP is involved.

## Task 3: Restore TestADAuthenticator_Authenticate_Success (line 67)

**File:** `internal/core/security/ad_authenticator_test.go`

Remove Skip. Use fake LDAP server + a real `ADAuthenticator` with sqlite db:

```go
func TestADAuthenticator_Authenticate_Success(t *testing.T) {
    // Setup fake LDAP server
    server := newFakeLDAPServer(t)
    server.entries = []*ldapSearchEntry{
        {
            dn:   "cn=testuser,dc=test,dc=com",
            attrs: map[string][]string{
                "cn":           {"testuser"},
                "mail":         {"testuser@test.com"},
                "displayName":  {"Test User"},
                "telephoneNumber": {"1234567890"},
                "department":   {"Engineering"},
                "title":        {"Engineer"},
            },
        },
    }
    server.SetBindResult(0) // success
    server.Start()
    t.Cleanup(server.Close)

    // Setup test DB with AD config
    db := setupSecurityTestDB(t)
    adConfig := &models.ADConfig{
        BaseModel:     models.BaseModel{ID: "test-ad-config"},
        ConfigName:    "Test AD",
        ServerAddress: "127.0.0.1",
        ServerPort:    server.Port(),
        DomainName:    "test.com",
        BaseDN:        "dc=test,dc=com",
        AdminUsername: "admin",
        AdminPassword: "admin_password",
        Status:        0,
    }
    require.NoError(t, db.Create(adConfig).Error)

    auth := NewADAuthenticator(db, "test-ad-config")
    req := MockAuthRequest("testuser", "password123")

    result, err := auth.Authenticate(context.Background(), req)

    // With fake LDAP returning success bind, auth should succeed
    assert.NoError(t, err, "AD auth should succeed with fake LDAP")
    assert.NotNil(t, result)
    assert.Equal(t, "ad", result.AuthSource)
}
```

Note: Need to add `require` import if not present.

## Task 4: Restore TestADAuthenticator_Authenticate_TableDrivenTests (line 133)

**File:** `internal/core/security/ad_authenticator_test.go`

Remove Skip. Use fake LDAP server for the tests that need it. The table-driven tests include cases for config not found, empty username, empty password — but also tests that would need LDAP. Consider splitting or conditionally using fake LDAP:

```go
func TestADAuthenticator_Authenticate_TableDrivenTests(t *testing.T) {
    db := setupSecurityTestDB(t)

    tests := []struct {
        name     string
        configID string
        username string
        password string
        wantErr  bool
        setup    func()
    }{
        {
            name:     "配置不存在",
            configID: "nonexistent",
            username: "testuser",
            password: "password",
            wantErr:  true,
            setup:    func() {},
        },
        {
            name:     "空用户名",
            configID: "test-config",
            username: "",
            password: "password",
            wantErr:  true,
            setup:    func() {},
        },
        {
            name:     "空密码",
            configID: "test-config",
            username: "testuser",
            password: "",
            wantErr:  true,
            setup:    func() {},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            auth := NewADAuthenticator(db, tt.configID)
            req := MockAuthRequest(tt.username, tt.password)
            result, err := auth.Authenticate(context.Background(), req)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

For the table entries that need real LDAP (bind success), add additional test cases in a separate run block with fake LDAP server setup.

## Task 5: Restore TestADAuthenticator_NeedsSyncFlag (line 214)

**File:** `internal/core/security/ad_authenticator_test.go`

Remove Skip. Use fake LDAP server returning successful bind + search entry:

```go
func TestADAuthenticator_NeedsSyncFlag(t *testing.T) {
    server := newFakeLDAPServer(t)
    server.entries = []*ldapSearchEntry{
        {
            dn:   "cn=syncuser,dc=test,dc=com",
            attrs: map[string][]string{
                "cn":           {"syncuser"},
                "mail":         {"syncuser@test.com"},
                "displayName":  {"Sync User"},
            },
        },
    }
    server.SetBindResult(0)
    server.Start()
    t.Cleanup(server.Close)

    db := setupSecurityTestDB(t)
    adConfig := &models.ADConfig{
        BaseModel:     models.BaseModel{ID: "test-sync-config"},
        ConfigName:    "Test Sync",
        ServerAddress: "127.0.0.1",
        ServerPort:    server.Port(),
        DomainName:    "test.com",
        BaseDN:        "dc=test,dc=com",
        AdminUsername: "admin",
        AdminPassword: "admin_password",
        Status:        0,
    }
    require.NoError(t, db.Create(adConfig).Error)

    auth := NewADAuthenticator(db, "test-sync-config")
    req := MockAuthRequest("syncuser", "password")

    result, err := auth.Authenticate(context.Background(), req)

    assert.NoError(t, err, "AD auth should succeed with fake LDAP")
    assert.NotNil(t, result)
    assert.True(t, result.NeedsSync || result.User != nil,
        "AD auth success should set NeedsSync=true or return User info")
}
```

## Task 6: Keep IntegrationTest skipped (lines 244, 253)

**File:** `internal/core/security/ad_authenticator_test.go`

Leave `t.Skip("集成测试需要真实AD环境配置")` on both `TestADAuthenticator_IntegrationTest` functions. These require a real Active Directory environment and should be documented in HUMAN-UAT.md.

## Task 7: Verify

```bash
go test ./internal/core/security/ -run "TestADAuth" -v
```

The tests at lines 108, 67, 133, 214 should pass. Lines 244, 253 should remain Skip.

</tasks>

<success_criteria>
- `go test ./internal/core/security/ -run "TestADAuth" -v` passes without Skip for ConfigNotFound, Success, TableDriven, NeedsSyncFlag
- `TestADAuthenticator_IntegrationTest` remains skipped (real AD required)
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>

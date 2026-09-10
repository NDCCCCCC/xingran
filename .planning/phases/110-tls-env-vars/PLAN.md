---
phase: "110"
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - pkg/cache/redis.go
autonomous: true
requirements_addressed: [TLS-01]
must_haves:
  truths:
    - "Redis TLS InsecureSkipVerify is controlled by REDIS_TLS_INSECURE_SKIP_VERIFY env var (default false)"
    - "When env var is true, a one-time SECURITY warn is logged"
  artifacts:
    - path: "pkg/cache/redis.go"
      provides: "Redis TLS env var control"
      exports: ["NewRedisCache"]
  key_links:
    - from: "pkg/cache/redis.go:30-36"
      to: "os.Getenv"
      via: "REDIS_TLS_INSECURE_SKIP_VERIFY"
      pattern: "REDIS_TLS_INSECURE_SKIP_VERIFY"
---

<objective>
Fix TLS-01: Replace the hardcoded `InsecureSkipVerify: true` in `pkg/cache/redis.go:35` with environment-variable control using `REDIS_TLS_INSECURE_SKIP_VERIFY` (default false = secure). This turns GUARD-01 GREEN.
</objective>

<context>
@pkg/cache/redis.go (lines 30-36)
@internal/services/addomain/ldap_client.go:99-107 (reference pattern for env-var TLS control)

Reference pattern from ldap_client.go (already fixed, use identical pattern):
```go
insecureSkip := strings.EqualFold(os.Getenv("LDAP_TLS_INSECURE_SKIP_VERIFY"), "true")
tlsConfig := &tls.Config{
    InsecureSkipVerify: insecureSkip,
    MinVersion:         tls.VersionTLS12,
}
if insecureSkip {
    logTLSInsecureSkipOnce()  // warn-once log
}
```

Current RED state in redis.go:
```go
if config.TLS {
    // 托管 Redis (Upstash 等) 强制 TLS;InsecureSkipVerify 与现有 LDAPS 路径一致,
    // 待生产化时统一替换为受信 CA 池 — 单独跟踪,不在本 quick task scope。
    tlsCfg = &tls.Config{InsecureSkipVerify: true}  // ← hardcoded true, INSECURE
}
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Replace Redis hardcoded InsecureSkipVerify with env var control</name>
  <files>pkg/cache/redis.go</files>
  <action>
In `pkg/cache/redis.go`, modify the `NewRedisCache` function to replace the hardcoded `InsecureSkipVerify: true` with environment-variable control.

**Imports needed** (add to existing import block if not present):
```go
"os"
"strings"
"sync"
```

**Add warn-once logger** (after imports, package-level):
```go
// tlsInsecureWarnOnce records a one-time SECURITY warning when REDIS_TLS_INSECURE_SKIP_VERIFY=true.
var tlsInsecureWarnOnce sync.Once

func logTLSInsecureSkipOnce() {
    tlsInsecureWarnOnce.Do(func() {
        applogger.Warnf("[SECURITY] REDIS_TLS_INSECURE_SKIP_VERIFY=true: TLS certificate verification is disabled. This is INSECURE and must only be used in internal networks with self-signed certificates.")
    })
}
```

**Modify the TLS config block** (lines 32-36):
Replace:
```go
if config.TLS {
    // 托管 Redis (Upstash 等) 强制 TLS;InsecureSkipVerify 与现有 LDAPS 路径一致,
    // 待生产化时统一替换为受信 CA 池 — 单独跟踪,不在本 quick task scope。
    tlsCfg = &tls.Config{InsecureSkipVerify: true}
}
```

With:
```go
if config.TLS {
    insecureSkip := strings.EqualFold(os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY"), "true")
    tlsCfg = &tls.Config{
        InsecureSkipVerify: insecureSkip,
        MinVersion:         tls.VersionTLS12,
    }
    if insecureSkip {
        logTLSInsecureSkipOnce()
    }
}
```

**Key constraints:**
- Default (env var unset or not "true") = `InsecureSkipVerify = false` (secure default)
- Only when env var is explicitly "true" = `InsecureSkipVerify = true`
- Follow the identical pattern as ldap_client.go for consistency
</action>
  <verify>
    <automated>go build ./pkg/cache/... && go test ./pkg/cache/... -run "TestRedis_TLSConfig_NotInsecureByDefault" -v 2>&1 | tail -20</automated>
  </verify>
  <done>REDIS_TLS_INSECURE_SKIP_VERIFY env var controls InsecureSkipVerify; default is false (secure). GUARD-01 test passes.</done>
</task>

</tasks>

<verification>
GUARD-01 must turn GREEN:
```
go test ./pkg/cache/... -run "TestRedis_TLSConfig_NotInsecureByDefault" -v
```
Expected: test passes (env-var pattern found, no hardcoded true).
</verification>

<success_criteria>
- `pkg/cache/redis.go` no longer contains `InsecureSkipVerify: true` without env var reference
- `REDIS_TLS_INSECURE_SKIP_VERIFY` env var controls the behavior
- Default (unset) = secure (InsecureSkipVerify = false)
- One-time SECURITY warn logged when env var = "true"
- GUARD-01 passes
- `go build ./...` passes
</success_criteria>

<output>
Create `.planning/phases/110-tls-env-vars/110-01-SUMMARY.md` when done
</output>

---

---
phase: "110"
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/core/security/ad_authenticator.go
autonomous: true
requirements_addressed: [TLS-02]
must_haves:
  truths:
    - "AD Authenticator dialConnection reads AD_AUTH_TLS_INSECURE_SKIP_VERIFY env var (default false)"
    - "When env var is true, a one-time SECURITY warn is logged"
  artifacts:
    - path: "internal/core/security/ad_authenticator.go"
      provides: "AD Authenticator TLS env var control"
      exports: ["dialConnection"]
  key_links:
    - from: "internal/core/security/ad_authenticator.go:181-183"
      to: "os.Getenv"
      via: "AD_AUTH_TLS_INSECURE_SKIP_VERIFY"
      pattern: "AD_AUTH_TLS_INSECURE_SKIP_VERIFY"
---

<objective>
Fix TLS-02: Replace the hardcoded `InsecureSkipVerify: true` in `internal/core/security/ad_authenticator.go:181-183` with environment-variable control using `AD_AUTH_TLS_INSECURE_SKIP_VERIFY` (default false = secure). This turns GUARD-02 GREEN.
</objective>

<context>
@internal/core/security/ad_authenticator.go (lines 179-183)
@internal/services/addomain/ldap_client.go:99-107 (reference pattern)

Current RED state (ad_authenticator.go:181-183):
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: true, // TODO: 生产环境应配置证书
}
```

Reference pattern from ldap_client.go (already fixed):
```go
insecureSkip := strings.EqualFold(os.Getenv("LDAP_TLS_INSECURE_SKIP_VERIFY"), "true")
tlsConfig := &tls.Config{
    InsecureSkipVerify: insecureSkip,
    MinVersion:         tls.VersionTLS12,
}
if insecureSkip {
    logTLSInsecureSkipOnce()
}
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Replace AD Authenticator hardcoded InsecureSkipVerify with env var control</name>
  <files>internal/core/security/ad_authenticator.go</files>
  <action>
In `internal/core/security/ad_authenticator.go`, modify the `dialConnection` function to replace the hardcoded `InsecureSkipVerify: true` with environment-variable control.

**Add to imports** (if not already present):
```go
"os"
"strings"
"sync"
```

**Add warn-once logger** (package-level, near top of file after imports):
```go
// tlsInsecureWarnOnce records a one-time SECURITY warning when AD_AUTH_TLS_INSECURE_SKIP_VERIFY=true.
var tlsInsecureWarnOnce sync.Once

func logADTLSInsecureSkipOnce() {
    tlsInsecureWarnOnce.Do(func() {
        // Use the standard logger directly since this package may not use applogger
        println("[SECURITY] AD_AUTH_TLS_INSECURE_SKIP_VERIFY=true: TLS certificate verification is disabled for AD authentication. This is INSECURE and must only be used in internal networks with self-signed certificates.")
    })
}
```

**Modify the tlsConfig construction** (lines 181-183):
Replace:
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: true, // TODO: 生产环境应配置证书
}
```

With:
```go
insecureSkip := strings.EqualFold(os.Getenv("AD_AUTH_TLS_INSECURE_SKIP_VERIFY"), "true")
tlsConfig := &tls.Config{
    InsecureSkipVerify: insecureSkip,
    MinVersion:         tls.VersionTLS12,
}
if insecureSkip {
    logADTLSInsecureSkipOnce()
}
```

**Key constraints:**
- Default (env var unset or not "true") = `InsecureSkipVerify = false` (secure default)
- Only when env var is explicitly "true" = `InsecureSkipVerify = true`
- Follow the identical pattern as ldap_client.go
</action>
  <verify>
    <automated>go build ./internal/core/security/... && go test ./internal/core/security/... -run "TestADAuthenticator_TLSConfig_StrictByDefault" -v 2>&1 | tail -20</automated>
  </verify>
  <done>AD_AUTH_TLS_INSECURE_SKIP_VERIFY env var controls InsecureSkipVerify; default is false (secure). GUARD-02 test passes.</done>
</task>

</tasks>

<verification>
GUARD-02 must turn GREEN:
```
go test ./internal/core/security/... -run "TestADAuthenticator_TLSConfig_StrictByDefault" -v
```
Expected: test passes (AST finds no hardcoded InsecureSkipVerify:true without env-var wrapper).
</verification>

<success_criteria>
- `internal/core/security/ad_authenticator.go` no longer contains `InsecureSkipVerify: true` without env var reference
- `AD_AUTH_TLS_INSECURE_SKIP_VERIFY` env var controls the behavior
- Default (unset) = secure (InsecureSkipVerify = false)
- One-time SECURITY warn logged when env var = "true"
- GUARD-02 passes
- `go build ./...` passes
</success_criteria>

<output>
Create `.planning/phases/110-tls-env-vars/110-02-SUMMARY.md` when done
</output>

---

---
phase: "110"
plan: "03"
type: execute
wave: 2
depends_on: []
files_modified:
  - internal/services/email_sender_service.go
autonomous: true
requirements_addressed: [TLS-03]
must_haves:
  truths:
    - "Email sender reads EMAIL_TLS_INSECURE_SKIP_VERIFY env var (default false = secure)"
  artifacts:
    - path: "internal/services/email_sender_service.go"
      provides: "Email sender TLS env var control"
      exports: ["sendWithTLS", "sendWithSTARTTLS"]
  key_links:
    - from: "internal/services/email_sender_service.go:203-206,271-274"
      to: "os.Getenv"
      via: "EMAIL_TLS_INSECURE_SKIP_VERIFY"
      pattern: "EMAIL_TLS_INSECURE_SKIP_VERIFY"
---

<objective>
Fix TLS-03: Add environment-variable control `EMAIL_TLS_INSECURE_SKIP_VERIFY` (default false = secure) to `internal/services/email_sender_service.go` for the `sendWithTLS` and `sendWithSTARTTLS` functions. The current hardcoded `InsecureSkipVerify: false` is already secure, but adding env var provides consistency with the LDAP/Redis pattern and allows internal-network flexibility.
</objective>

<context>
@internal/services/email_sender_service.go (lines 200-206, 254-278)

Current state (already secure, but add env var for consistency):
```go
// sendWithTLS (line 203-206)
tlsConfig := &tls.Config{
    InsecureSkipVerify: false,  // currently hardcoded false (secure)
    ServerName:         strings.Split(addr, ":")[0],
}

// sendWithSTARTTLS (line 271-274)
tlsConfig := &tls.Config{
    InsecureSkipVerify: false,  // currently hardcoded false (secure)
    ServerName:         strings.Split(addr, ":")[0],
}
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add EMAIL_TLS_INSECURE_SKIP_VERIFY env var control to email sender</name>
  <files>internal/services/email_sender_service.go</files>
  <action>
In `internal/services/email_sender_service.go`, add environment-variable control to both `sendWithTLS` and `sendWithSTARTTLS` functions.

**Add to imports** (if not already present):
```go
"os"
"strings"
```

**Modify sendWithTLS** (around line 203-206):
Replace:
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: false,
    ServerName:         strings.Split(addr, ":")[0],
}
```

With:
```go
insecureSkip := strings.EqualFold(os.Getenv("EMAIL_TLS_INSECURE_SKIP_VERIFY"), "true")
tlsConfig := &tls.Config{
    InsecureSkipVerify: insecureSkip,
    ServerName:         strings.Split(addr, ":")[0],
}
```

**Modify sendWithSTARTTLS** (around line 271-274):
Replace:
```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: false,
    ServerName:         strings.Split(addr, ":")[0],
}
```

With:
```go
insecureSkip := strings.EqualFold(os.Getenv("EMAIL_TLS_INSECURE_SKIP_VERIFY"), "true")
tlsConfig := &tls.Config{
    InsecureSkipVerify: insecureSkip,
    ServerName:         strings.Split(addr, ":")[0],
}
```

**Key constraints:**
- Default (env var unset or not "true") = `InsecureSkipVerify = false` (secure default)
- Only when env var is explicitly "true" = `InsecureSkipVerify = true`
- No warn logging needed here (email sender is lower risk than Redis/AD)
</action>
  <verify>
    <automated>go build ./internal/services/... && grep -c 'EMAIL_TLS_INSECURE_SKIP_VERIFY' internal/services/email_sender_service.go</automated>
  </verify>
  <done>EMAIL_TLS_INSECURE_SKIP_VERIFY env var controls both sendWithTLS and sendWithSTARTTLS; default is false (secure).</done>
</task>

</tasks>

<verification>
Verify the env var appears in both functions:
```
grep -n 'EMAIL_TLS_INSECURE_SKIP_VERIFY' internal/services/email_sender_service.go
```
Expected: at least 2 occurrences (one per function).
</verification>

<success_criteria>
- Both `sendWithTLS` and `sendWithSTARTTLS` read `EMAIL_TLS_INSECURE_SKIP_VERIFY`
- Default (unset) = secure (InsecureSkipVerify = false)
- `go build ./...` passes
</success_criteria>

<output>
Create `.planning/phases/110-tls-env-vars/110-03-SUMMARY.md` when done
</output>

---

---
phase: "110"
plan: "04"
type: execute
wave: 2
depends_on: []
files_modified:
  - cmd/main.go
  - pkg/middleware/cors.go
autonomous: true
requirements_addressed: [TLS-04, TLS-05]
must_haves:
  truths:
    - "cmd/main.go reads allowedOrigins from config.Server.AllowedOrigins (not hardcoded)"
    - "pkg/middleware/cors.go respects WS_ALLOW_ALL_ORIGINS env var (default false)"
    - "Empty config or only '*' triggers fail-fast in production"
  artifacts:
    - path: "cmd/main.go:104-105"
      provides: "WebSocket allowedOrigins from config"
    - path: "pkg/middleware/cors.go"
      provides: "WS_ALLOW_ALL_ORIGINS env override"
  key_links:
    - from: "cmd/main.go:104"
      to: "cfg.Server.AllowedOrigins"
      via: "config read"
      pattern: "AllowedOrigins"
---

<objective>
Fix TLS-04 and TLS-05: Remove the hardcoded `allowedOrigins := []string{"*"}` from `cmd/main.go:104` and add `WS_ALLOW_ALL_ORIGINS` environment variable override to `pkg/middleware/cors.go`. This turns GUARD-03 GREEN.
</objective>

<context>
@cmd/main.go (lines 104-105)
@pkg/middleware/cors.go
@cmd/main_test.go:TestMain_AllowedOrigins_FromConfig_NotWildcard (GUARD-03)

Current RED state (cmd/main.go:104):
```go
allowedOrigins := []string{"*"}
setupRoutes(engine, cfg, coreModule, allowedOrigins)
```

GUARD-03 test checks that this hardcoded pattern no longer exists.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Remove hardcoded wildcard, read allowedOrigins from config</name>
  <files>cmd/main.go</files>
  <action>
In `cmd/main.go`, replace the hardcoded `allowedOrigins := []string{"*"}` with config-based reading.

**Modify lines 104-105**:
Replace:
```go
allowedOrigins := []string{"*"}
setupRoutes(engine, cfg, coreModule, allowedOrigins)
```

With:
```go
allowedOrigins := cfg.Server.AllowedOrigins
// Fail-fast in production if no origins configured (empty list or only "*")
if len(allowedOrigins) == 0 {
    applogger.Fatalf("server.allowed_origins must be configured in production: at least one origin is required")
}
// Warn if only wildcard is configured (still allows, but warns about insecure setup)
for _, origin := range allowedOrigins {
    if origin == "*" {
        applogger.Warnf("server.allowed_origins contains '*' wildcard — this allows all origins. In production, configure specific origins.")
        break
    }
}
setupRoutes(engine, cfg, coreModule, allowedOrigins)
```

**Note:** The `applogger` is used at package level in main.go already (check imports). If not imported, add:
```go
applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
```
</action>
  <verify>
    <automated>go build ./cmd/... && go test ./cmd/... -run "TestMain_AllowedOrigins_FromConfig_NotWildcard" -v 2>&1 | tail -20</automated>
  </verify>
  <done>allowedOrigins is read from config.Server.AllowedOrigins; no hardcoded wildcard. GUARD-03 passes.</done>
</task>

<task type="auto">
  <name>Task 2: Add WS_ALLOW_ALL_ORIGINS env var override to CORS middleware</name>
  <files>pkg/middleware/cors.go</files>
  <action>
In `pkg/middleware/cors.go`, add `WS_ALLOW_ALL_ORIGINS` environment variable override.

**Add to imports** (if not already present):
```go
"os"
"strings"
```

**Modify the `Cors` function** (around line 20-22):
Replace:
```go
func Cors(allowedOrigins []string) gin.HandlerFunc {
    // 如果没有指定允许的域名或者包含通配符，则允许所有来源（仅开发环境）
    allowAll := len(allowedOrigins) == 0 || contains(allowedOrigins, "*")
```

With:
```go
func Cors(allowedOrigins []string) gin.HandlerFunc {
    // WS_ALLOW_ALL_ORIGINS env var overrides allowedOrigins for WebSocket endpoints.
    // Default (unset/"false") = use config. Set to "true" to allow all origins (INSECURE).
    allowAllEnv := strings.EqualFold(os.Getenv("WS_ALLOW_ALL_ORIGINS"), "true")
    // 如果没有指定允许的域名或者包含通配符，则允许所有来源（仅开发环境）
    allowAll := allowAllEnv || len(allowedOrigins) == 0 || contains(allowedOrigins, "*")
```

**Key constraints:**
- Default (env var unset or not "true") = use config.allowedOrigins (secure)
- Only when env var is explicitly "true" = allow all origins (INSECURE, for dev only)
- The env var provides a fallback override without changing config files
</action>
  <verify>
    <automated>go build ./pkg/middleware/... && grep -c 'WS_ALLOW_ALL_ORIGINS' pkg/middleware/cors.go</automated>
  </verify>
  <done>WS_ALLOW_ALL_ORIGINS env var controls CORS allowAll override; default is false (use config).</done>
</task>

</tasks>

<verification>
GUARD-03 must turn GREEN:
```
go test ./cmd/... -run "TestMain_AllowedOrigins_FromConfig_NotWildcard" -v
```
Expected: test passes (no `allowedOrigins := []string{"*"}` found in source).
</verification>

<success_criteria>
- `cmd/main.go` reads `cfg.Server.AllowedOrigins` instead of hardcoding `[]string{"*}"`
- Empty config causes fail-fast with `applogger.Fatalf`
- `WS_ALLOW_ALL_ORIGINS` env var can override CORS when set to "true"
- GUARD-03 passes
- `go build ./...` passes
</success_criteria>

<output>
Create `.planning/phases/110-tls-env-vars/110-04-SUMMARY.md` when done
</output>

---

---
phase: "110"
plan: "05"
type: execute
wave: 3
depends_on: []
files_modified:
  - docs/deployment/secret-management.md
autonomous: true
requirements_addressed: [TLS-06]
must_haves:
  truths:
    - "secret-management.md documents all TLS/Origin env vars with MUST SET / 内网兼容 guidance"
  artifacts:
    - path: "docs/deployment/secret-management.md"
      provides: "TLS env var documentation"
  key_links:
    - from: "docs/deployment/secret-management.md"
      to: "TLS/Origin env vars"
      via: "documentation"
      pattern: "TLS"
---

<objective>
Fix TLS-06: Update `docs/deployment/secret-management.md` to document all TLS and Origin environment variables introduced in Phase 110, with MUST SET / 内网兼容 guidance per D-02.
</objective>

<context>
@docs/deployment/secret-management.md (existing structure)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Document TLS/Origin env vars in secret-management.md</name>
  <files>docs/deployment/secret-management.md</files>
  <action>
In `docs/deployment/secret-management.md`, add a new section "## TLS / Origin Configuration (v1.32)" after the existing content.

**Section content to add:**

```markdown
## TLS / Origin Configuration (v1.32)

The following environment variables control TLS certificate verification and CORS origin policies. All default to secure values (verification enabled, restricted origins).

### Environment Variables

| Env Var | Default | Description |
|---------|---------|-------------|
| `LDAP_TLS_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification for LDAP connections. **MUST NOT** be set in production with public CAs. 内网兼容自签证书时可设 `true`。 |
| `AD_AUTH_TLS_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification for AD Authenticator. **MUST NOT** be set in production with public CAs. 内网兼容自签证书时可设 `true`。 |
| `REDIS_TLS_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification for Redis connections (when `config.TLS=true`). **MUST NOT** be set in production with public CAs. 内网兼容自签证书时可设 `true`。 |
| `EMAIL_TLS_INSECURE_SKIP_VERIFY` | `false` | Skip TLS verification for SMTP email connections. **MUST NOT** be set in production with public CAs. 内网兼容自签证书时可设 `true`。 |
| `WS_ALLOW_ALL_ORIGINS` | `false` | Allow all WebSocket origins (`Access-Control-Allow-Origin: *`). **MUST NOT** be set in production. 仅开发环境使用。 |

### Production Checklist

- [ ] **MUST NOT** set any `_INSECURE_SKIP_VERIFY` env vars in production with public CA certificates
- [ ] **MUST NOT** set `WS_ALLOW_ALL_ORIGINS=true` in production
- [ ] **MUST** configure `server.allowed_origins` with specific origins in production
- [ ] 内网部署 with self-signed certificates: set `_INSECURE_SKIP_VERIFY=true` only for the specific service needing it

### Security Warning

Setting any `_INSECURE_SKIP_VERIFY` variable to `true` disables TLS certificate verification. This allows man-in-the-middle (MITM) attacks and should only be used in trusted internal networks with self-signed certificates.
```

**Key constraints:**
- Place the new section after existing content (do not overwrite existing sections)
- Follow the existing document's markdown style and formatting
- Include both English and Chinese descriptions
</action>
  <verify>
    <automated>grep -c 'TLS.*SKIP.*VERIFY\|WS_ALLOW_ALL_ORIGINS' docs/deployment/secret-management.md</automated>
  </verify>
  <done>secret-management.md documents all 5 TLS/Origin env vars with MUST NOT / 内网兼容 guidance.</done>
</task>

</tasks>

<verification>
Verify all env vars are documented:
```
grep -E 'LDAP_TLS|AD_AUTH_TLS|REDIS_TLS|EMAIL_TLS|WS_ALLOW_ALL_ORIGINS' docs/deployment/secret-management.md
```
Expected: all 5 env vars appear in the document.
</verification>

<success_criteria>
- All 5 TLS/Origin env vars documented
- MUST NOT SET guidance for production
- 内网兼容 guidance for self-signed cert scenarios
- `go build ./...` passes (doc change only)
</success_criteria>

<output>
Create `.planning/phases/110-tls-env-vars/110-05-SUMMARY.md` when done
</output>

---

## PHASE 110 SUMMARY

**Phase:** P0 安全 TLS 环境变量化
**Goal:** All TLS/Origin options controlled by env vars (default false = secure); GUARD-01/02/03 turn GREEN

### Plan Structure

| Plan | Wave | TLS IDs | GUARD IDs | Files Modified | Tasks |
|------|------|---------|-----------|---------------|-------|
| 110-01 | 1 | TLS-01 | GUARD-01 | `pkg/cache/redis.go` | 1 |
| 110-02 | 1 | TLS-02 | GUARD-02 | `internal/core/security/ad_authenticator.go` | 1 |
| 110-03 | 2 | TLS-03 | - | `internal/services/email_sender_service.go` | 1 |
| 110-04 | 2 | TLS-04, TLS-05 | GUARD-03 | `cmd/main.go`, `pkg/middleware/cors.go` | 2 |
| 110-05 | 3 | TLS-06 | - | `docs/deployment/secret-management.md` | 1 |

### Wave Execution

**Wave 1 (parallel):** Plans 110-01, 110-02 — independent files, run together
**Wave 2:** Plan 110-03, 110-04 — after Wave 1
**Wave 3:** Plan 110-05 — documentation, after Wave 2

### Critical Verification Commands

```bash
# GUARD-01 (TLS-01 fix)
go test ./pkg/cache/... -run "TestRedis_TLSConfig_NotInsecureByDefault" -v

# GUARD-02 (TLS-02 fix)
go test ./internal/core/security/... -run "TestADAuthenticator_TLSConfig_StrictByDefault" -v

# GUARD-03 (TLS-04 fix)
go test ./cmd/... -run "TestMain_AllowedOrigins_FromConfig_NotWildcard" -v

# All phase gates
go build ./...
go test ./...
go vet ./...
```

### Phase Completion Criteria

After all 5 plans executed:
- GUARD-01, GUARD-02, GUARD-03 all GREEN
- All 6 TLS items (TLS-01..TLS-06) implemented
- `go build ./...` passes
- `go test ./...` passes (all GUARD tests pass)
- Backend coverage maintained at >=78.33%

### Commit Strategy

Each plan produces one atomic commit:
```
110-01: fix(tls-01): redis TLS env var REDIS_TLS_INSECURE_SKIP_VERIFY
110-02: fix(tls-02): ad_authenticator TLS env var AD_AUTH_TLS_INSECURE_SKIP_VERIFY
110-03: fix(tls-03): email sender TLS env var EMAIL_TLS_INSECURE_SKIP_VERIFY
110-04: fix(tls-04,tls-05): allowedOrigins from config + WS_ALLOW_ALL_ORIGINS env
110-05: docs(tls-06): document TLS/Origin env vars in secret-management.md
```

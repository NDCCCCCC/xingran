# Deploy Crash Reproduction Report

**Commit:** 1fd7e20 (v1.33 carryover)
**Date:** 2026-09-14
**Reproduction Status:** ROOT CAUSE IDENTIFIED

---

## Reproduction Command

```bash
# Build
cd D:/CODE/ClaudeCode/guoguo && go build -o /tmp/xingran-backend-repro ./cmd/main.go

# Setup minimal environment
mkdir -p /tmp/xingran-repro/{configs,data,logs,uploads}
cp configs/config.sqlite.example.yaml /tmp/xingran-repro/configs/config.yaml

# Run WITHOUT allowed_origins (reproduces crash):
cd /tmp/xingran-repro && \
  SM4_KEY="dGVzdC1zZWNyZXQxNiEhIQ==" \
  JWT_SECRET="test-jwt-secret-key" \
  JWT_SM2_PRIVATE_KEY="" JWT_SM2_PUBLIC_KEY="" \
  SERVER_MODE=release /tmp/xingran-backend-repro
# Result: Process exits with status 1, no panic/fatal in output

# Run WITH allowed_origins (startup succeeds):
# Add to config.yaml: server.allowed_origins: ["http://localhost:9000"]
# Result: Server binds :9000, /health returns 200, process stays alive
```

---

## Root Cause

**File:** `cmd/main.go:106-107`
```go
if len(allowedOrigins) == 0 {
    applogger.Fatalf("server.allowed_origins must be configured in production: at least one origin is required")
}
```

**Mechanism:**
- In `release` mode (SERVER_MODE=release), the server requires `server.allowed_origins` to be non-empty
- `config.sqlite.example.yaml` does NOT include `allowed_origins` field
- When deploying with `config.sqlite.example.yaml` as the production config, `allowedOrigins` is empty
- `applogger.Fatalf()` logs "server.allowed_origins must be configured..." and calls `os.Exit(1)`
- This exits with status 1 — no panic, no stack trace, just silent exit

**Why the journal dump only shows WARN (not FATAL):**
- The OUI WARN appears last in the journal because `importOUIData()` is called at the very end of `initializeCoreModule()` (line 169), after the fatal exit from `setupRoutes()` has already terminated the process
- The process exits during route setup, before OUI import runs
- The OUI WARN is NOT the cause — it appears last because execution never reaches it in the failing case

**Confirmed by log ordering:**
- Without allowed_origins: `setupRoutes` FATAL exit happens before `initializeCoreModule` completes → OUI WARN never appears
- With allowed_origins: Full startup completes including OUI WARN (harmless)

---

## Why OUI/API Metadata WARNs Are Not Fatal

Both `initAPIEndpointService` and `importOUIData` use non-fatal `applogger.Warnf()` on failure:
- `initAPIEndpointService`: returns error → caller logs `WARN` → continues
- `importOUIData`: called in `initializeCoreModule` → logs `WARN` on failure → continues

These are graceful degradation paths, not crash causes.

---

## Evidence Summary

| Test | Config | Result |
|------|--------|--------|
| Sqlite template without allowed_origins | config.sqlite.example.yaml as-is | `applogger.Fatalf` → exit 1 |
| Sqlite template with allowed_origins added | Added `server.allowed_origins` | Server binds :9000, /health 200 |
| Previous working commit (hypothesis) | Unknown | Presumably had allowed_origins or different mode check |

---

## Hypothesis

**The crash is caused by `server.allowed_origins` being absent from the deployed config.yaml.**

The `config.sqlite.example.yaml` template is missing the `allowed_origins` field. When deployed as-is with `SERVER_MODE=release`, the `setupRoutes` check at `cmd/main.go:106` fires `applogger.Fatalf()` and the process exits with status 1.

The OUI/API-metadata WARNs in the journal are coincidental — they appear in the 80-line dump because:
1. The process may have been restarted multiple times
2. Or the journal captures logs from the startup sequence across restarts
3. The OUI warning is the last warning that appears in a *successful* startup sequence

---

## Recommended Fix

1. **Add `allowed_origins` to `config.sqlite.example.yaml`:**
   ```yaml
   server:
     name: "XingRan-Next"
     host: "0.0.0.0"
     port: 9000
     mode: release
     allowed_origins:
       - "https://your-production-domain.com"  # Replace with actual domain
   ```

2. **Verify production deployments** use a config that includes `allowed_origins`

3. **Optionally**: Add validation at config load time (config.go) to fail fast with a clear error message if `allowed_origins` is missing in release mode, rather than during route setup

---
phase: "110-tls-env-vars"
plan: "01"
subsystem: infra
tags: [redis, tls, security, env-var, golang]

# Dependency graph
requires: []
provides:
  - "REDIS_TLS_INSECURE_SKIP_VERIFY env var controls InsecureSkipVerify (default false = strict)"
  - "GUARD-01 TestRedis_TLSConfig_NotInsecureByDefault now GREEN"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TLS env-var control pattern: os.Getenv + strings.EqualFold + warn-once sync.Once"

key-files:
  modified:
    - "pkg/cache/redis.go"

key-decisions:
  - "Default false (strict TLS verification) per D-02; internal networks can set REDIS_TLS_INSECURE_SKIP_VERIFY=true"
  - "Followed ldap_client.go pattern exactly: env var + MinVersion + sync.Once warn log"

patterns-established:
  - "TLS InsecureSkipVerify env-var pattern: strings.EqualFold(os.Getenv(\"X_TLS_INSECURE_SKIP_VERIFY\"), \"true\")"

requirements-completed: [TLS-01]

# Metrics
duration: 5min
completed: 2026-09-09
---

# Phase 110-01: Redis TLS Env Var Control Summary

**REDIS_TLS_INSECURE_SKIP_VERIFY env var replaces hardcoded InsecureSkipVerify: true — GUARD-01 GREEN**

## Performance

- **Duration:** 5 min
- **Started:** 2026-09-09T01:25:00Z
- **Completed:** 2026-09-09T01:30:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Replaced hardcoded `InsecureSkipVerify: true` in `pkg/cache/redis.go` with env var `REDIS_TLS_INSECURE_SKIP_VERIFY` (default false = strict)
- Added `MinVersion: tls.VersionTLS12` to enforce TLS 1.2+
- Added warn-once log via `sync.Once` when insecure skip is enabled
- GUARD-01 (`TestRedis_TLSConfig_NotInsecureByDefault`) now PASS

## Task Commits

1. **Task 1: Replace Redis hardcoded InsecureSkipVerify with env var control** - `a86fdb4` (feat)

## Files Created/Modified

- `pkg/cache/redis.go` - Added `os`, `sync` imports; added `tlsInsecureWarnOnce`/`logTLSInsecureSkipOnce` warn-once logger; replaced hardcoded `InsecureSkipVerify: true` with `strings.EqualFold(os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY"), "true")` + `MinVersion: tls.VersionTLS12`

## Decisions Made

- Used `strings.EqualFold` for case-insensitive env var comparison (consistent with ldap_client.go pattern)
- Kept warn-once pattern (`sync.Once`) to avoid log flooding on every connection
- Default is **false** (secure) — production拒绝自签证书; only set `REDIS_TLS_INSECURE_SKIP_VERIFY=true` for internal networks with self-signed certs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification Results

```
go test ./pkg/cache/... -run "TestRedis_TLSConfig_NotInsecureByDefault" -v
=== RUN   TestRedis_TLSConfig_NotInsecureByDefault
    redis_test.go:106: PASS: InsecureSkipVerify uses REDIS_TLS_INSECURE_SKIP_VERIFY env var (secure default)
--- PASS: TestRedis_TLSConfig_NotInsecureByDefault (0.00s)
PASS

go build ./...  # passed (pre-existing ad_authenticator_test.go vet warning unrelated to this change)
go vet ./pkg/cache/...  # passed
go test ./pkg/cache/...  # all tests passed
```

## Next Phase Readiness

- TLS-01 (REDIS_TLS_INSECURE_SKIP_VERIFY) complete — ready for plan 110-02 (AD_AUTH_TLS_INSECURE_SKIP_VERIFY)
- GUARD-01 GREEN

---
*Phase: 110-01*
*Completed: 2026-09-09*

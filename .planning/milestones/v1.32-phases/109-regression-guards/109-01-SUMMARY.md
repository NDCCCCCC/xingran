# Phase 109 Plan 01: GUARD-01 Redis TLS Regression Test — Summary

## Execution Summary

**Plan:** 109-01
**GUARD ID:** GUARD-01
**Test:** `TestRedis_TLSConfig_NotInsecureByDefault`
**Location:** `pkg/cache/redis_test.go`
**Commit:** `8335d65`

## Test Behavior

| State | Condition | Result |
|-------|-----------|--------|
| **RED (Phase 109)** | `InsecureSkipVerify: true` hardcoded in `redis.go:35` | **FAIL** (expected) |
| **GREEN (Phase 110)** | `InsecureSkipVerify` controlled by `REDIS_TLS_INSECURE_SKIP_VERIFY` env var | **PASS** |

## Test Strategy

Source code pattern inspection — the test reads `redis.go`, extracts the TLS config block within `NewRedisCache` via regex, and asserts:

1. `InsecureSkipVerify` is NOT hardcoded to `true`
2. If present, it uses `os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY")` pattern

**Why not live Redis connection:** miniredis (used by all existing Redis tests) does not support TLS mode, so a live TLS connection test is not feasible. The source inspection approach is the same strategy used by other regression guards (Plan 109-07).

## Current Test Output (RED)

```
=== RUN   TestRedis_TLSConfig_NotInsecureByDefault
    redis_test.go:76: InsecureSkipVerify is hardcoded to true in TLS config — this is INSECURE.
        Expected: env-var control (e.g. os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY") == "true")
        Found TLS config body: InsecureSkipVerify: true
        FIX: Phase 110 TLS-01 must replace hardcoded true with env-var controlled value
--- FAIL: TestRedis_TLSConfig_NotInsecureByDefault (0.00s)
```

## Phase 110 Fix Required

In `pkg/cache/redis.go`, replace:
```go
tlsCfg = &tls.Config{InsecureSkipVerify: true}
```

With env-var controlled pattern:
```go
insecureSkip := os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY") == "true"
tlsCfg = &tls.Config{InsecureSkipVerify: insecureSkip}
```

This gives:
- Default (env unset/empty): `InsecureSkipVerify = false` (secure)
- `REDIS_TLS_INSECURE_SKIP_VERIFY=true`: `InsecureSkipVerify = true` (dev/debug only)

## Truths Captured

- Redis TLS config defaults to secure (`InsecureSkipVerify=false` when TLS enabled)

## Artifacts

| File | Lines | Purpose |
|------|-------|---------|
| `pkg/cache/redis_test.go` | 111 | GUARD-01 test implementation |

## Deviations

None — plan executed as written.

---
phase: 98
plan: 98-01
title: V130R-04 列表缓存键防碰撞修复
subsystem: system
tags: [cache, security, bugfix]
dependency_graph:
  requires: []
  provides: []
  affects: [internal/services/system/user_cache_impl.go, internal/services/system/role_cache_impl.go]
tech_stack:
  added: []
  patterns: [EscapeCacheKeyValue helper in cache_keys.go]
key_files:
  created:
    - internal/services/system/user_cache_key_collision_test.go
  modified:
    - internal/services/system/user_cache_impl.go
    - internal/services/system/role_cache_impl.go
decisions:
  - "URL-escape colons in parameter values (replace ':' with '%3A') - chosen over hashing for debuggability and simplicity"
metrics:
  duration: "~10 minutes"
  completed_date: 2026-09-06
---

# Phase 98 Plan 98-01: V130R-04 Cache Key Collision Fix Summary

## One-liner

Fixed cache key collision bug where `Username="bob:status:1"` and `Username="bob"+Status=1` produced identical cache keys.

## What Was Fixed

`buildListCacheKey` in both `user_cache_impl.go` and `role_cache_impl.go` concatenated parameter values directly with `:` separators:

```
key = "user:list:username:bob:status:1:page:1:size:10"
```

If `Username = "bob:status:1"`, it would collide with `Username="bob"+Status=1` since both produce `"user:list:username:bob:status:1:..."`.

## How

Applied `EscapeCacheKeyValue` (existing helper in `cache_keys.go`) to all user-input parameter values that are concatenated into cache keys:

**user_cache_impl.go** - escaped fields: `username`, `nickname`, `phone`, `deptId`, `recursiveDeptId`, `beginTime`, `endTime`

**role_cache_impl.go** - escaped fields: `roleName`, `roleKey`

Sorting params (`orderBy`, `isAsc`, `page`, `size`) and status are code-controlled integers/enums and do not need escaping.

## Verification

- `go build ./...` - PASS
- `go test -v -run TestCacheKeyCollision ./internal/services/system/` - PASS (collision cases verified)
- `go test -v -run "Test.*Cache.*" ./internal/services/system/` - PASS (all 16 cache tests)

## Commits

- `5fd062e` fix(98-01): escape colon in buildListCacheKey to prevent key collision (V130R-04)

## Deviations from Plan

None - plan executed exactly as written. `EscapeCacheKeyValue` helper already existed in `cache_keys.go` (was added in a prior phase), so the implementation was simpler than anticipated.

## Self-Check: PASSED

- [x] `internal/services/system/user_cache_impl.go` - modified, `strings` import added, `EscapeCacheKeyValue` applied to 7 fields
- [x] `internal/services/system/role_cache_impl.go` - modified, `EscapeCacheKeyValue` applied to 2 fields
- [x] `internal/services/system/user_cache_key_collision_test.go` - created, collision test passes
- [x] Commit `5fd062e` exists in git log
- [x] `go build ./...` passes
- [x] All cache tests pass

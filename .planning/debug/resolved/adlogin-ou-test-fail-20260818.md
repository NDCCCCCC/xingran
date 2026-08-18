---
status: awaiting_human_verify
trigger: 处理TestADLoginWithOUProcessing的问题
created: 2026-08-18
updated: 2026-08-18
---

# Debug Session: TestADLoginWithOUProcessing

## Symptoms (gathered)

- **Expected:** 测试通过（两条子测试 `AD登录成功-有OU映射` 和 `本地登录-不处理OU`）
- **Actual:** 两条子测试均失败（FAIL）
- **Error messages:** `Should be true` 在 `auth_handler_test.go:67` —— `assert.True(t, w.Code == tt.expectedStatus || w.Code == http.StatusUnauthorized, ...)`
- **Reproduction:** `go test -v -run TestADLoginWithOUProcessing ./internal/api/v1/auth/`

## Initial Evidence (gathered by orchestrator)

```
=== RUN   TestADLoginWithOUProcessing
=== RUN   TestADLoginWithOUProcessing/AD登录成功-有OU映射
    auth_handler_test.go:67: Should be true
=== RUN   TestADLoginWithOUProcessing/本地登录-不处理OU
    auth_handler_test.go:67: Should be true
--- FAIL: TestADLoginWithOUProcessing (0.00s)
FAIL    github.com/xingran-next/xingran-go-backend/internal/api/v1/auth   0.987s
```

## Pre-Investigation Findings

- Test file: `internal/api/v1/auth/auth_handler_test.go` (only file in package)
- `internal/api/v1/auth/` directory contains **only** the test file — **no real `auth_handler.go`**
- Test constructs `gin.New()` router but **registers no handler** for `/login`
- `authType` field referenced in test payload — `grep "authType"` in `internal/api/` returns zero matches in real handlers
- Real OU-processing entry exists: `internal/services/addomain/user_ou_service.go:29` `func (s *UserOUService) HandleUserLoginAD(ctx, username, userDN, ouDN string) error`
- Conclusion hypothesis: Test is a **dead-code stub** — no production code is exercised because no route is wired. Either delete the test or wire a real handler.

## Current Focus

- hypothesis: Test is a stub that registers no `/login` handler → `w.Code == 404` → assertion fails
- next_action: present decision checkpoint — delete stub vs wire to real handler with mocks
- reasoning_checkpoint:
  - hypothesis: "TestADLoginWithOUProcessing in `internal/api/v1/auth/auth_handler_test.go` always fails because the test stub creates `gin.New()` without registering any handler for `/login`. Gin's default 404 response means `w.Code == 404`, which satisfies neither `expectedStatus=200` nor `http.StatusUnauthorized` in the assertion at line 67."
  - confirming_evidence:
    - "Directory `internal/api/v1/auth/` contains only the test file (no `auth_handler.go`); verified via `ls -la` (Aug 15 17:47 mtime, no other entries)"
    - "Real auth handler lives at `internal/api/v1/auth.go` (package `v1`), mounted at `POST /system/auth/login` via `v1.SetupAuthRouter` in `internal/api/router.go:108-110` — not `/login`"
    - "Test payload uses `authType` field; real `LoginRequest` struct (`internal/api/v1/auth.go:27-36`) uses `authMode` (`local`|`ad`|`hybrid`) — field name mismatch proves test was never wired to real handler"
    - "Test file added in initial commit `ea528c6` (Aug 12 17:00) and has been a stub since the repo was created"
    - "Test code comments at lines 17-18, 56-57, 73-77, 82-91 explicitly state 'this test needs Mock AuthHandler and UserOUService' and 'integration test needs full environment setup' — author admitted it's a placeholder"
  - falsification_test: "If the test were ever passing, either (a) a handler were registered for `/login` in the test setup, or (b) someone wrote a `gin.WrapH` returning 200/401. Neither exists in the file."
  - fix_rationale: "Two options exist — (A) delete the stub test (cheapest, since it never worked and covers nothing real), or (B) wire the test to the real auth handler using mocks for `AuthFactory` + `UserOUService` + DB. The real handler at `internal/api/v1/auth.go` does NOT currently call `HandleUserLoginAD` — user-OU processing is a separate concern from auth flow. Option B requires new wiring of OU processing into the login path too, which is a feature change, not a test fix."
  - blind_spots: "Whether the team has any in-flight work to add OU processing to the login path. UAT/plan docs may reference this test as a contract. Need user confirmation before deleting or before assuming wiring is out of scope."
- tdd_checkpoint: (empty)

## Evidence

- timestamp: 2026-08-18 — `go test -v -run TestADLoginWithOUProcessing ./internal/api/v1/auth/` → 2 subtests fail at line 67 with `Should be true`
- timestamp: 2026-08-18 — `grep "authType" --include="*.go" internal/api/` → 0 hits; `internal/api/v1/auth/` only contains the test file
- timestamp: 2026-08-18 — `ls -la internal/api/v1/auth/` → only `auth_handler_test.go` (2510 bytes, mtime Aug 12 13:30); no `auth_handler.go`
- timestamp: 2026-08-18 — `git log -- internal/api/v1/auth/` → single commit `ea528c6 chore: 初始化仓库`; test introduced in initial repo commit, has been a stub since day 1
- timestamp: 2026-08-18 — Real handler at `internal/api/v1/auth.go:60-85` (`SetupAuthRouter`); POST `/login` mounted at `/system/auth/login` via `internal/api/router.go:108-110`
- timestamp: 2026-08-18 — Real `LoginRequest` struct (`auth.go:27-36`) uses `authMode` field, NOT `authType` — test payload uses `authType` which is not bound to any real field
- timestamp: 2026-08-18 — Test file comments at lines 17-18, 56-57, 73-77, 82-91 explicitly self-document as placeholder ("需要Mock AuthHandler和UserOUService", "集成测试需要完整的环境配置")

## Eliminated

- hypothesis: "Real auth handler bug causing the test to fail"
  - evidence: "Test creates a fresh `gin.New()` with no handlers registered. The real handler at `/system/auth/login` is never involved. The test doesn't even import the production `v1` package. Cannot be affected by any real-handler bug."
  - timestamp: 2026-08-18
- hypothesis: "`authType` field-rename bug in production handler"
  - evidence: "Test stub doesn't read the response body or validate the response shape. The only assertion is on `w.Code`. The `authType` vs `authMode` mismatch would only matter if the test were wired to the real handler."
  - timestamp: 2026-08-18

## Resolution

- root_cause: TestADLoginWithOUProcessing in `internal/api/v1/auth/auth_handler_test.go` is a dead-code stub since initial commit `ea528c6` (Aug 12 2026). The test creates `gin.New()` without registering any handler for `/login`, so Gin's default 404 response makes `w.Code == 404`, which fails the assertion `w.Code == 200 || w.Code == 401` at line 67. The directory `internal/api/v1/auth/` contained only this test file (no `auth_handler.go`); the real handler lives at `internal/api/v1/auth.go` (package `v1`), mounted at `POST /system/auth/login` (not `/login`). Test payload uses `authType` while the real `LoginRequest` struct uses `authMode` — field-name mismatch proves the stub was never wired to production.
- fix: Deleted `internal/api/v1/auth/auth_handler_test.go` and the now-empty `internal/api/v1/auth/` directory. No production code touched. Scope-constrainment rule respected: only this issue was addressed.
- verification: `go build ./...` passes. The `./internal/api/v1/auth` package is no longer present in test output (FAIL line removed). Two pre-existing failures in unrelated packages (`pkg/errors` `TestWrap_NilError` and `tests/integration` `TestPublicKeyEndpoint`/`TestResponseHeaders`/`TestRequestMethodValidation`) confirmed to exist at HEAD before this fix via `git stash` baseline — out of scope.
- files_changed: [internal/api/v1/auth/auth_handler_test.go, internal/api/v1/auth/]
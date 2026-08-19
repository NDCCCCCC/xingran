---
status: reference
type: knowledge-base
note: 不是 debug session —— 已解决会话的模式参考库，供 gsd-debugger 检索。audit-open 应跳过。
---

# GSD Debug Knowledge Base

Resolved debug sessions. Used by `gsd-debugger` to surface known-pattern hypotheses at the start of new investigations.

---

## adlogin-ou-test-fail-20260818 — Dead test stub asserting on un-wired gin.New() produces 404
- **Date:** 2026-08-18
- **Error patterns:** TestADLoginWithOUProcessing, Should be true, 404, gin.New, authType, expectedStatus, w.Code
- **Root cause:** Test file at `internal/api/v1/auth/auth_handler_test.go` is a dead-code stub since initial commit. It constructs `gin.New()` without registering any handler for `/login`, so Gin returns default 404, which fails the assertion `w.Code == 200 || w.Code == 401`. The directory contains only the test file (no `auth_handler.go`); the real handler lives at `internal/api/v1/auth.go` (package `v1`) at route `POST /system/auth/login` (not `/login`). Test payload uses `authType` field while real `LoginRequest` uses `authMode` — confirms stub was never wired to production.
- **Fix:** Delete the dead test file and the now-empty `internal/api/v1/auth/` directory. No production code touched.
- **Files changed:** internal/api/v1/auth/auth_handler_test.go, internal/api/v1/auth/
---

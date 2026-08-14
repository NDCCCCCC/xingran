# Deferred Items — quick-260814-h0e

Discovered during execution but out of scope (per executor SCOPE BOUNDARY:
pre-existing failures in unrelated files are not auto-fixed).

## Pre-existing flaky test: TestValidateAPIKey/密钥已禁用

- **Discovered during:** Task 2 verification (`go test ./internal/services/system/`)
- **Reproduces on:** Clean baseline at HEAD~ (Task 1 only, before Task 2 changes
  applied) — confirmed via stash + re-run
- **Symptom:**
  `apikey_service_test.go:484` fails with
  `"[1014] 数据库操作失败: database table is locked: sys_user" does not contain "密钥不存在或已禁用"`
- **Root cause (suspected):** SQLite in-memory write-lock contention when
  TestValidateAPIKey runs alongside other tests in the same package. Passes in
  isolation (`go test -run TestValidateAPIKey`) and passes when run with a
  narrower filter (`-run 'TestMenu|TestCache|TestValidateAPIKey'`).
- **Scope relevance:** None. This is an apikey test; Task 1/2 only touched
  `cache_keys.go` and `menu_cache_impl.go` (menu cache layer).
- **Suggested fix (for future task):** Serialize apikey test writes via
  `t.Cleanup` barriers or use per-test sqlite file DB instead of shared
  in-memory DB. Or migrate apikey tests to use the same test isolation pattern
  used in quick-260814-h0e (parallel menu-cache tests).
- **Owner action:** Log as tech-debt. Do NOT block this quick task on it.

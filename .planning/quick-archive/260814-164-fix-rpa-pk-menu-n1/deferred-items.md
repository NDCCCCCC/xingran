# Deferred Items — quick-260814-164

Out-of-scope discoveries logged per executor SCOPE BOUNDARY rule. Not caused by this task's changes; not fixed here.

## 1. Flaky test: TestListUsageLogs/时间范围筛选 (pre-existing)

- **File:** `internal/services/system/apikey_service_test.go:948`
- **Symptom:** `assert.Equal(t, int64(1), result.Total)` fails with `expected 1, actual 0`, followed by `panic: runtime error: index out of range [0] with length 0` at line 949 (`result.List[0].Path`).
- **Flakiness confirmed:** 5x run without this task's menu change → 1 FAIL (run 3). 3x run with this task's menu change → 1 FAIL (run 2). Failure rate ~20% either way — independent of this task.
- **Scope:** `apikey_service_test.go` / `ListUsageLogs` is API Key usage log time-range filtering — **unrelated** to this task's `menu_service.go` ancestor-collection change.
- **Suspected cause:** Test data isolation between subtests sharing the sqlite DB; usage-log rows leak across subtests or `cleanupTestData` order interacts with `time.Now()`-relative timestamps under load.
- **Action required:** Separate investigation in apikey module. Do NOT bundle into a menu/rpa bugfix quick task.

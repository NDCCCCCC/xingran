---
phase: 72-p0-core-supplement
plan: 03
subsystem: scheduler
tags: [CORE-03, scheduler, handler-tests, coverage, function-field-mock]
dependency_graph:
  requires: []
  provides:
    - "internal/api/v1/scheduler handler test suite (85.5% coverage)"
  affects:
    - "internal/api/v1/scheduler (no business code changes per D-08)"
tech-stack:
  added: []
  patterns:
    - "function-field mock (mockJobService + mockJobLogService)"
    - "httptest.NewRecorder + gin.CreateTestContext"
    - "compile-time interface assertion at file top"
key-files:
  created:
    - internal/api/v1/scheduler/job_handler_test.go
  modified: []
decisions:
  - "function-field mock 模式 (D-01 lightweight) 覆盖 JobService + JobLogService 全部 11 方法"
  - "表驱动 TC1..TC33 命名,每个 handler 方法至少 2 个场景 (happy + error)"
  - "operlog nil-safe 通过 h.core = nil 即可 (operlog.Record 内部 nil-guard)"
metrics:
  duration: "~10 min"
  completed_date: 2026-08-21
---

# Phase 72 Plan 03: scheduler handler tests 0% -> 85.5%

## One-liner
Add 33 table-driven test cases covering all 11 handler methods on `JobHandler` (CRUD + lifecycle + logs) using function-field mocks for `JobService` + `JobLogService`, bringing `internal/api/v1/scheduler` from 0% to 85.5% statement coverage.

## Coverage
- **Target package:** `internal/api/v1/scheduler`
- **Stmts:** 152 (D-04 baseline)
- **Achieved coverage:** 85.5% (>= 70% target)
- **Pre-plan coverage:** 0.0%
- **Delta:** +85.5pp

## Files Modified

| File | Status | Test Cases |
|------|--------|------------|
| `internal/api/v1/scheduler/job_handler_test.go` | created | 33 (TC1..TC33) |

## Test Coverage Map

| Category | Methods | Test Cases |
|----------|---------|------------|
| 1. CRUD | Create, List, GetByID, Update, Delete | TC1-TC16 (16) |
| 2. Lifecycle | UpdateStatus, Execute | TC17-TC23 (7) |
| 3. Logs | Statistics, ListLogs, CleanLogs | TC24-TC31 (8) |
| 4. WithCore + nil-safe | WithCore, WithCore_NilCore | TC32-TC33 (2) |

## Key Design Decisions

### D-01 lightweight pattern adherence
- **Function-field mocks** for both `JobService` (7 methods) and `JobLogService` (4 methods)
- **Compile-time interface assertions** at top of file:
  ```go
  var _ schedulerServices.JobService = (*mockJobService)(nil)
  var _ schedulerServices.JobLogService = (*mockJobLogService)(nil)
  ```
- **httptest.NewRecorder + gin.CreateTestContext** pattern from `ad_account_handler_test.go`

### operlog nil-safety
Handler tests inject `&core.Core{CoreInfra: {}, CoreServices: {}}` (no DB). operlog.Record's built-in nil-guard handles the nil DB correctly.

### Edge case coverage
- **UpdateStatus status text branch**: TC18 (status=0 → "启用成功") vs TC19 (status=1 → "暂停成功") — covers the `if req.Status == int(models.JobStatusPause)` branch
- **List with int vs float64 pagination**: TC5 uses `current: 2.0` (float64) to cover the type switch in handler

### Test invocation idiom
- JSON body via `bytes.NewBuffer(json.Marshal(body))` with `Content-Type: application/json` header
- Path params via `c.Params = gin.Params{{Key: "id", Value: id}}` for `:id` URL params
- Invalid JSON via `bytes.NewBufferString("{invalid")` to trigger binding error

## Verification

```bash
$ go test -count=1 -cover ./internal/api/v1/scheduler/...
ok  github.com/xingran-next/xingran-go-backend/internal/api/v1/scheduler  0.310s  coverage: 85.5% of statements
```

## Deviations from Plan

- **None** — plan executed as specified. All 11 handler methods covered with at least 2 test cases each.

## Self-Check

```
go test -count=1 -cover ./internal/api/v1/scheduler/...
ok  internal/api/v1/scheduler  coverage: 85.5% of statements
```

PASSED.

## Next Plan
72-04: `internal/services/workorder` service tests (715 stmts, 0.6% -> >= 70%)

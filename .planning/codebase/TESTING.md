# Testing

## Testing Framework

### Backend (Go)
- **testing** (stdlib) + **testify v1.11.1** (assertions)
- Test files: `*_test.go` alongside source files
- Run: `go test ./...` or `go test -v ./internal/services/operations/`

### Frontend (React)
- **Vitest** — unit testing
- Run: `npm run test`, `npm run test:ui`, `npm run test:coverage`

## Test Coverage

### Current State
Test coverage is **sparse**. Most business logic lacks tests.

**Existing test files (17 total):**
- `internal/constants/example_test.go`
- `internal/core/security/password_test.go`
- `internal/scheduler/ad_sync_tasks_test.go`
- `internal/services/data_cache_service_test.go`
- `internal/services/operations/batch_upserter_test.go`
- `internal/services/operations/building_service_test.go`
- `internal/services/operations/cache_invalidator_test.go`
- `internal/services/operations/cache_stats_test.go`
- `internal/services/operations/pagination_helper_test.go`
- `internal/services/operations/rate_limiter_test.go`
- `internal/services/operations/reference_resolver_test.go`
- `internal/services/operations/validation_helper_test.go`
- `pkg/cache/l2_writer_test.go`
- `pkg/cache/retry_test.go`
- `pkg/crypto/nonce_storage_bench_test.go`
- `pkg/errors/errors_test.go`

**Best-covered areas:**
- `internal/services/operations/` — 8 test files (building service, batch upserter, cache)
- `pkg/cache/` — 2 test files (L2 writer, retry)
- `pkg/crypto/` — 1 bench test

**No test coverage for:**
- API handlers (all modules)
- Most service modules (system, workorder, network, duty, knowledge, etc.)
- Middleware
- Router configuration
- WebSocket
- Scheduler engine

## Testing Patterns

### Go Test Structure
```go
func TestXxx(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    svc := NewXxxService(db, nil)

    // Execute
    result, err := svc.DoSomething(context.Background(), params)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Test Database
- Some tests use real DB connections (requires running PostgreSQL)
- No mock database layer — tests that hit DB need environment setup
- No Docker-based test infrastructure detected

### Assertions
Uses testify/assert:
```go
assert.NoError(t, err)
assert.Equal(t, expected, actual)
assert.NotNil(t, result)
assert.Contains(t, slice, element)
```

## Frontend Testing

- **Vitest** configured in `xingran-react-frontend/`
- Very limited test coverage
- No component tests detected
- No integration/e2e tests detected

## Gaps & Recommendations

1. **No test infrastructure**: No Makefile, no CI test pipeline, no coverage enforcement
2. **Handler tests missing**: All HTTP handlers untested
3. **Service coverage sparse**: Only operations module has meaningful tests
4. **No mocks**: No gomock or testify/mock usage; tests that need DB use real connections
5. **No frontend component tests**: React components untested
6. **No e2e tests**: No Playwright/Cypress/ similar framework

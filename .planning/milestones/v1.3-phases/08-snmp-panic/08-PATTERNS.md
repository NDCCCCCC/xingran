# Phase 8: SNMP Panic 修复 - Pattern Map

**Mapped:** 2026-04-27
**Files analyzed:** 2
**Analogs found:** 2 / 2

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/device/snmp_client.go` | client | request-response | `internal/device/scrapli_wrapper.go` | exact (panic recovery) |
| `internal/device/snmp_client_test.go` | test | unit-test | `internal/services/operations/rate_limiter_test.go` | role-match |

## Pattern Assignments

### `internal/device/snmp_client.go` (client, request-response)

**Analog:** `internal/device/scrapli_wrapper.go`

**Panic recovery pattern** (lines 296-304):
```go
// 捕获 panic
defer func() {
    if r := recover(); r != nil {
        w.setState(StateClosed)
        w.closeOnce.Do(func() {
            close(w.initDone)
        })
        resultCh <- result{err: fmt.Errorf("连接 panic: %v", r)}
    }
}()
```

**Panic recovery with logging** (from `connection_pool.go` lines 89-97):
```go
// 捕获可能的 panic（特别是 scrapligo 库的空指针问题）
defer func() {
    if r := recover(); r != nil {
        err = fmt.Errorf("执行命令时发生 panic: %v, deviceID=%s", r, pc.deviceID)
        // 标记连接为已关闭，防止后续使用
        if pc.wrapper != nil {
            pc.wrapper.setState(StateClosed)
        }
    }
}()
```

**Logging pattern** (from `pkg/logger/logger.go`):
```go
// Error logging with context
applogger.Errorf("[设备] SNMP操作panic: device=%s, method=%s, error=%v, stack=%s", 
    deviceID, methodName, r, string(debug.Stack()))

// Warning logging
applogger.Warnf("[设备] SNMP操作恢复: device=%s, method=%s", deviceID, methodName)
```

**Error handling pattern** (existing in `snmp_client.go` lines 88-91):
```go
result, err := c.client.Get([]string{oid})
if err != nil {
    return nil, fmt.Errorf("SNMP GET失败: %w", err)
}
```

**Connection management pattern** (existing in `snmp_client.go` lines 82-86):
```go
func (c *SNMPClient) Get(oid string) (interface{}, error) {
    if err := c.Connect(); err != nil {
        return nil, err
    }
    defer c.Close()
    // ... SNMP operation
}
```

**Device context pattern** (from `connection_pool.go` lines 91-92):
```go
// Include device identification in all error messages
err = fmt.Errorf("执行命令时发生 panic: %v, deviceID=%s", r, pc.deviceID)
```

---

### `internal/device/snmp_client_test.go` (test, unit-test)

**Analog:** `internal/services/operations/rate_limiter_test.go`

**Test structure pattern** (lines 9-24):
```go
func TestNewRateLimiter(t *testing.T) {
    maxTokens := 50
    refillInterval := 500 * time.Millisecond

    limiter := NewRateLimiter(maxTokens, refillInterval)

    if limiter.maxTokens != maxTokens {
        t.Errorf("maxTokens = %v, want %v", limiter.maxTokens, maxTokens)
    }
    if limiter.refillRate != refillInterval {
        t.Errorf("refillRate = %v, want %v", limiter.refillRate, refillInterval)
    }
    if limiter.currentTokens != maxTokens {
        t.Errorf("currentTokens = %v, want %v", limiter.currentTokens, maxTokens)
    }
}
```

**Concurrent testing pattern** (lines 110-143):
```go
func TestRateLimiter_Concurrent(t *testing.T) {
    maxTokens := 100
    refillInterval := 10 * time.Millisecond
    limiter := NewRateLimiter(maxTokens, refillInterval)

    var wg sync.WaitGroup
    allowed := 0
    var mu sync.Mutex

    // 并发请求
    for i := 0; i < maxTokens; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if limiter.Allow() {
                mu.Lock()
                allowed++
                mu.Unlock()
            }
        }()
    }

    wg.Wait()

    // 所有请求都应该被允许（因为令牌足够）
    if allowed != maxTokens {
        t.Errorf("允许的请求数 = %v, want %v", allowed, maxTokens)
    }
}
```

**Table-driven test pattern** (lines 184-206):
```go
func TestMin(t *testing.T) {
    tests := []struct {
        name     string
        a        int
        b        int
        expected int
    }{
        {"a 小于 b", 3, 5, 3},
        {"a 大于 b", 7, 4, 4},
        {"a 等于 b", 6, 6, 6},
        {"负数", -5, -3, -5},
        {"零", 0, 5, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := min(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

**Skip pattern for unimplemented tests** (from `building_service_test.go` lines 7-11):
```go
func TestBuildingService_Create(t *testing.T) {
    // 这里需要 mock DB 或者使用测试数据库
    // 暂时跳过，实际实现时需要添加
    t.Skip("需要 mock DB 或测试数据库")
}
```

---

## Shared Patterns

### Panic Recovery Wrapper
**Source:** `internal/device/scrapli_wrapper.go` lines 296-304
**Apply to:** All SNMP operation methods (`Get`, `GetNext`, `Walk`, `GetBulk`)
```go
// 捕获 panic 的通用模式
defer func() {
    if r := recover(); r != nil {
        // Log with full context
        applogger.Errorf("[SNMP] Panic: device=%s, method=%s, error=%v, stack=%s",
            deviceID, methodName, r, string(debug.Stack()))
        // Return error instead of crashing
        err = fmt.Errorf("SNMP操作panic: %v", r)
    }
}()
```

### Error Wrapping with Context
**Source:** `internal/device/snmp_client.go` lines 88-91
**Apply to:** All SNMP methods that return errors
```go
// Standard error wrapping pattern
if err != nil {
    return nil, fmt.Errorf("SNMP GET失败: %w", err)
}
```

### Logging with Device Context
**Source:** `pkg/logger/logger.go` and `internal/device/connection_pool.go`
**Apply to:** All panic recovery logging
```go
// Structured logging with device identification
applogger.Errorf("[SNMP] Panic: ip=%s, method=%s, error=%v", 
    ip, methodName, r)
```

### Connection Cleanup
**Source:** `internal/device/scrapli_wrapper.go` lines 299-302
**Apply to:** Panic recovery in connection-dependent operations
```go
// Mark connection as closed on panic to prevent reuse
w.setState(StateClosed)
w.closeOnce.Do(func() {
    close(w.initDone)
})
```

### Test Naming Convention
**Source:** `internal/services/operations/rate_limiter_test.go`
**Apply to:** All test functions
```go
// Test function naming: Test<FunctionName>_<Scenario>
func TestSNMPClient_Get_PanicRecovery(t *testing.T)
func TestSNMPClient_Walk_PanicRecovery(t *testing.T)
```

## No Analog Found

Files with no close match in the codebase (planner should use CONTEXT.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (None) | - | - | All files have appropriate analogs |

## Metadata

**Analog search scope:**
- `internal/device/` (device management patterns)
- `internal/services/operations/` (testing patterns)
- `pkg/logger/` (logging patterns)

**Files scanned:** 3
- `internal/device/scrapli_wrapper.go` (panic recovery pattern)
- `internal/device/connection_pool.go` (panic recovery with logging)
- `internal/services/operations/rate_limiter_test.go` (testing patterns)
- `pkg/logger/logger.go` (logging API)

**Pattern extraction date:** 2026-04-27

## Key Implementation Notes

1. **Panic Recovery Scope**: Apply to 4 SNMP methods: `Get()`, `GetNext()`, `Walk()`, `GetBulk()`
2. **Logging Requirements**: Must include device IP, method name, panic value, and stack trace
3. **Error Return**: Convert panic to error return, don't crash the application
4. **Connection Management**: Mark connection as failed after panic to prevent reuse
5. **Testing Strategy**: Use table-driven tests for multiple panic scenarios, concurrent tests for thread safety
6. **Debug Stack**: Use `debug.Stack()` to capture full call stack in panic logs

## Architecture Alignment

This phase maintains the existing architecture patterns:
- **Handler-Service pattern**: SNMP client is a service layer component
- **Error handling**: Follows the `fmt.Errorf("context: %w", err)` wrapping convention
- **Logging**: Uses the centralized `applogger` package
- **Testing**: Standard Go testing package with table-driven and concurrent test patterns
- **Panic recovery**: Consistent with existing `scrapli_wrapper.go` and `connection_pool.go` patterns

# Domain Pitfalls

**Domain:** Go Backend Network Management & SNMP Integration
**Researched:** 2026-04-27
**Overall confidence:** HIGH

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

### Pitfall 1: SNMP Transport Layer Panic During Device Connection

**What goes wrong:** Intermittent goroutine panic in scrapligo transport layer during Channel.GetPrompt when trying to get system name OID (1.3.6.1.2.1.1.5.0). The panic occurs at `transport.(*Standard).Write` with exit status 2, crashing the entire application during scheduled SNMP monitoring tasks.

**Why it happens:**
- scrapligo's transport layer spawns internal goroutines during connection initialization
- `GetPrompt()` method creates a Write goroutine that may panic if the transport is not fully ready
- Timing-dependent: occurs intermittently during concurrent SNMP operations
- No proper synchronization between connection state and transport layer goroutines

**Consequences:**
- Application crash (exit status 2) during scheduled monitoring tasks
- Loss of real-time monitoring data
- Potential data corruption if panic occurs during device configuration backup
- Unreliable network device management functionality

**Prevention:**
- **ALWAYS wrap scrapligo operations with panic recovery** in defer functions
- **Use connection state machine pattern** with proper mutex protection (opMu.Lock/RLock)
- **Implement readiness verification** before calling transport methods
- **Add sleep delays before Close()** to allow internal goroutines to finish (100ms minimum)
- **Use context-based cancellation** for connection initialization with timeout

**Detection:**
- Look for goroutine stack traces mentioning `transport.(*Standard).Write` or `Channel.GetPrompt`
- Monitor for unexpected exit status 2 during scheduled tasks
- Check logs for "panic during SNMP Ping operation" patterns

**Phase Assignment:** Phase 1 - SNMP Panic Fix
- This must be addressed first as it affects system stability
- All subsequent phases depend on stable SNMP operations

### Pitfall 2: Race Conditions in Connection Pool Management

**What goes wrong:** Data races between connection.Close() and concurrent operations (SendCommand, SendConfig) accessing the same driver instance, causing "use of closed network connection" errors or nil pointer dereferences.

**Why it happens:**
- scrapligo driver is not concurrency-safe for simultaneous operations
- Connection pool may return the same connection to multiple goroutines
- No proper synchronization between Close() and active operations
- State checking (TOCTOU - Time-Of-Check-Time-Of-Use) race conditions

**Consequences:**
- Connection pool corruption
- Device commands sent to closed connections
- Memory leaks from connections that never close properly
- Intermittent "connection reset by peer" errors

**Prevention:**
- **Use device-level mutex locks** in connection pool (not connection-level)
- **Implement RWMutex pattern**: operations use RLock, Close uses Lock
- **Double-check state after acquiring locks** to prevent TOCTOU races
- **Set connection state to Closing before acquiring exclusive lock** for Close()
- **Add grace period before actual Close()** to allow in-flight operations to complete

**Detection:**
- Race detector output: `WARNING: DATA RACE`
- Stack traces showing concurrent Close() and SendCommand/SendConfig
- "use of closed network connection" errors in device operations

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - No synchronization
func (p *PooledConnection) SendCommand(cmd string) error {
    return p.driver.SendCommand(cmd) // Race with Close()
}

func (p *PooledConnection) Close() error {
    return p.driver.Close() // Closes while operations active
}
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - Proper synchronization
func (w *ScrapliWrapper) SendCommand(cmd string) (*Response, error) {
    if err := w.acquireOp(); err != nil { // RLock + state check
        return nil, err
    }
    defer w.releaseOp() // RUnlock
    
    // Safe to use driver now
    r, err := w.driver.SendCommand(cmd)
    // ... error handling
}

func (w *ScrapliWrapper) Close() error {
    w.setState(StateClosing) // Block new operations
    w.opMu.Lock()           // Wait for all operations
    defer w.opMu.Unlock()
    
    time.Sleep(100 * time.Millisecond) // Grace period
    return w.driver.Close()
}
```

**Phase Assignment:** Phase 1 - SNMP Panic Fix
- This is the root cause of the SNMP panic issue
- Must be fixed before implementing export functionality

### Pitfall 3: SQL Injection in Dynamic Table Names

**What goes wrong:** Direct string concatenation of user-provided table names in SQL queries, allowing attackers to execute arbitrary SQL commands.

**Why it happens:**
- Dynamic table name generation for multi-tenant scenarios
- Lack of input validation on table name parameters
- String concatenation instead of parameterized queries for table identifiers

**Consequences:**
- Data leakage from unauthorized tables
- Data corruption through malicious UPDATE/DELETE commands
- Complete database compromise
- Security audit failures

**Prevention:**
- **ALWAYS use whitelist validation** for table names
- **Never concatenate user input directly into SQL queries**
- **Use a predefined map of allowed table names**
- **Reject any table name not in the whitelist**

**Detection:**
- Code patterns: `+tableName+` or `fmt.Sprintf("...%s...", tableName)`
- Missing validation before table name usage
- Direct string concatenation in query building

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - Direct concatenation
floorTable := "ops_floors" // Could come from user input
query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE...)")
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - Whitelist validation
func validateTableName(tableName string) bool {
    allowedTables := map[string]bool{
        "ops_floors": true,
        "sys_floors": true,
    }
    return allowedTables[tableName]
}

if floorCode != "" {
    if !validateTableName(floorTable) {
        return nil, fmt.Errorf("invalid table name: %s", floorTable)
    }
    // Safe to use floorTable in query
}
```

**Phase Assignment:** Phase 2 - Backend Code Optimization
- Critical security issue that must be addressed
- Review all dynamic SQL construction in the codebase

## Moderate Pitfalls

### Pitfall 4: Improper Error Handling in Export Operations

**What goes wrong:** Export operations fail silently or return partial results when errors occur during Excel generation or database queries, leading to incomplete or corrupted export files.

**Why it happens:**
- Missing error checks on excelize operations
- No rollback mechanism for partial exports
- Lack of transaction management in export queries
- Assuming all operations will succeed

**Consequences:**
- Users receive incomplete export files
- Silent data loss in exported data
- Corrupted Excel files that cannot be opened
- User frustration and loss of trust in export functionality

**Prevention:**
- **Check all excelize method return values**
- **Use database transactions for export queries** to ensure consistency
- **Implement row-by-row error handling** with logging but continue processing
- **Validate export size limits** before starting to prevent memory exhaustion
- **Add progress tracking** for large exports

**Detection:**
- Missing `if err != nil` checks after excelize calls
- No `defer tx.Rollback()` pattern in database operations
- Hardcoded limits without validation (e.g., maxExportRows)

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - No error handling
for i, device := range devices {
    row := i + 2
    cell, _ := excelize.CoordinatesToCellName(i+1, row)
    file.SetCellValue(sheet, cell, device.Name) // Error ignored
}
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - Proper error handling
for i, device := range devices {
    row := i + 2
    cell, err := excelize.CoordinatesToCellName(i+1, row)
    if err != nil {
        logger.Errorf("Invalid cell coordinates: %v", err)
        continue // Skip this row but continue
    }
    if err := file.SetCellValue(sheet, cell, device.Name); err != nil {
        logger.Errorf("Failed to set cell value: %v", err)
        continue
    }
}
```

**Phase Assignment:** Phase 3 - Network Device Export
- Export functionality must be robust and handle errors gracefully
- Implement comprehensive error handling in all export operations

### Pitfall 5: Memory Exhaustion in Large Exports

**What goes wrong:** Export operations attempt to load entire datasets into memory before writing to Excel, causing out-of-memory errors when exporting large tables (e.g., 100K+ devices or MAC addresses).

**Why it happens:**
- Loading all records into memory before Excel generation
- No pagination or streaming in export operations
- Inefficient data structures that duplicate data
- Missing limits on export size

**Consequences:**
- Application crash (OOM killer) during export
- Server performance degradation
- Failed exports for large datasets
- Poor user experience

**Prevention:**
- **Implement pagination with reasonable page sizes** (e.g., 1000 rows per batch)
- **Use streaming patterns** for large Excel files (write row-by-row)
- **Set hard limits on export size** (e.g., 100K rows max)
- **Provide warning feedback** when approaching limits
- **Use cursor-based pagination** for large datasets

**Detection:**
- Patterns like `var devices []models.NetworkDevice` without pagination
- `Find(&devices)` without Limit() clause
- Export operations that load all data before processing

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - Loads all into memory
var devices []models.NetworkDevice
db.Find(&devices) // Could be 100K+ records
for _, device := range devices {
    // Process
}
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - Pagination with streaming
const pageSize = 1000
for page := 0; ; page++ {
    var devices []models.NetworkDevice
    offset := page * pageSize
    if err := db.Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
        return err
    }
    if len(devices) == 0 {
        break // No more records
    }
    // Process this batch
    writeBatchToExcel(file, devices)
}
```

**Phase Assignment:** Phase 3 - Network Device Export
- Must handle large exports efficiently
- Implement pagination and streaming patterns

### Pitfall 6: Resource Leaks in Connection Management

**What goes wrong:** Network connections, file handles, or database connections are not properly closed, leading to resource exhaustion over time. Common with SNMP operations and file exports.

**Why it happens:**
- Missing defer statements for cleanup
- Early returns that bypass cleanup code
- Panic conditions that skip defer execution
- Incorrect error handling that prevents cleanup

**Consequences:**
- File descriptor leaks (too many open files)
- Database connection pool exhaustion
- Memory leaks from unclosed connections
- Performance degradation over time
- Application crashes after running for extended periods

**Prevention:**
- **ALWAYS use defer for cleanup** (Close, Rollback, etc.)
- **Use named return values** to allow deferred cleanup to modify return values
- **Wrap panic-prone operations** in deferred recovery functions
- **Implement connection timeouts** to prevent hanging connections
- **Use context with cancellation** for long-running operations

**Detection:**
- Functions that open resources without matching defer Close()
- Early returns that skip cleanup code
- Missing error handling that prevents cleanup code from running

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - Resource leak on error
func ExportDevices() error {
    file := excelize.NewFile()
    if err := loadData(); err != nil {
        return err // file never closed!
    }
    file.WriteTo(buffer)
    file.Close()
    return nil
}
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - Proper cleanup with defer
func ExportDevices() (err error) {
    file := excelize.NewFile()
    defer func() {
        if cerr := file.Close(); cerr != nil {
            // Log but don't hide original error
            logger.Errorf("Failed to close file: %v", cerr)
        }
    }()
    
    if err := loadData(); err != nil {
        return err // defer will run
    }
    return file.WriteTo(buffer)
}
```

**Phase Assignment:** Phase 2 - Backend Code Optimization
- Review and fix all resource management patterns
- Implement proper cleanup across the codebase

## Minor Pitfalls

### Pitfall 7: Inconsistent Status Value Conventions

**What goes wrong:** Mixing up status value semantics (0 = enabled vs 0 = disabled) across different modules, causing confusion and bugs in business logic.

**Why it happens:**
- Inconsistent adoption of project conventions
- Different developers using different patterns
- Copy-paste from external codebases
- Lack of clear documentation

**Consequences:**
- Incorrect filtering in queries
- UI showing wrong status indicators
- Business logic errors (e.g., enabling disabled devices)
- Migration errors when combining data from different sources

**Prevention:**
- **Document the convention**: 0 = enabled/normal/visible, 1 = disabled/stopped/hidden
- **Exception documentation**: Menu visibility uses 1 = visible (boolean semantics)
- **Use constants or enums** instead of magic numbers
- **Add code reviews** to catch violations
- **Create helper functions** like `IsEnabled(status int) bool`

**Detection:**
- Magic numbers (0, 1) in business logic
- Inconsistent WHERE clauses (`status = 0` vs `status = 1`)
- Comments that contradict the code

**Phase Assignment:** Phase 2 - Backend Code Optimization
- Standardize status conventions across the codebase
- Add documentation and helper functions

### Pitfall 8: Hardcoded Configuration Values

**What goes wrong:** Configuration values (timeouts, limits, ports) are hardcoded in business logic, making it difficult to adapt to different environments without code changes.

**Why it happens:**
- Quick prototyping without proper configuration
- Lack of planning for deployment variability
- Copying values from external examples
- Not using the existing config system

**Consequences:**
- Cannot adapt to different environments without code changes
- Difficult to tune performance parameters
- Deployment complexity increases
- Configuration drift between environments

**Prevention:**
- **Move all timeouts to config files** (config.yaml)
- **Use environment variables** for deployment-specific values
- **Define constants in internal/constants/** for global values
- **Add validation** for configuration values on startup
- **Document configuration options** in example files

**Detection:**
- Hardcoded values like `time.Sleep(100 * time.Millisecond)`
- Magic numbers like `maxExportRows = 100000` in code
- Ports and IP addresses embedded in business logic

**Code Example - Anti-Pattern:**
```go
// ❌ WRONG - Hardcoded
timeout := 5 * time.Second
maxDevices := 100000
port := 161
```

**Code Example - Correct Pattern:**
```go
// ✅ CORRECT - From config
timeout := time.Duration(cfg.Device.Timeout) * time.Second
maxDevices := cfg.Export.MaxRows
port := cfg.SNMP.DefaultPort
```

**Phase Assignment:** Phase 2 - Backend Code Optimization
- Extract hardcoded values to configuration
- Centralize timeout and limit constants

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| **Phase 1: SNMP Panic Fix** | Race conditions in connection pool | Implement RWMutex pattern with state machine |
| **Phase 1: SNMP Panic Fix** | Panic in scrapligo transport layer | Add panic recovery in all scrapligo operations |
| **Phase 1: SNMP Panic Fix** | TOCTOU races in state checking | Double-check state after acquiring locks |
| **Phase 2: Code Optimization** | SQL injection in dynamic queries | Implement whitelist validation for table names |
| **Phase 2: Code Optimization** | Resource leaks (connections, files) | Add defer cleanup for all resources |
| **Phase 2: Code Optimization** | Hardcoded configuration values | Move to config.yaml or constants |
| **Phase 2: Code Optimization** | Inconsistent status conventions | Document and standardize |
| **Phase 3: Network Export** | Memory exhaustion on large exports | Implement pagination and streaming |
| **Phase 3: Network Export** | Silent failures in export operations | Check all excelize return values |
| **Phase 3: Network Export** | Missing transaction management | Use DB transactions for export queries |

## Integration Pitfalls

### Fixing One Issue Without Breaking Others

**Warning:** The SNMP panic fix involves modifying connection pool synchronization, which affects all code that uses the `ScrapliWrapper`. Be extremely careful to:

1. **Maintain backward compatibility** with existing connection pool usage
2. **Test all device operations** (SSH, Telnet, SNMP, config backup/restore)
3. **Verify concurrent access patterns** work correctly with new synchronization
4. **Ensure cleanup code runs** even when panics occur
5. **Monitor for performance degradation** from additional locking

**Specific Risks:**
- Adding opMu.Lock() in Close() could cause deadlocks if not all code paths release RLock properly
- Grace period sleep in Close() (100ms) might delay connection reuse in high-throughput scenarios
- State machine transitions must be atomic to prevent race conditions
- Panic recovery must not hide legitimate errors

**Testing Strategy:**
- Run load tests with concurrent device operations
- Verify connection pool doesn't leak connections
- Test graceful degradation when individual devices fail
- Monitor for deadlocks with the race detector
- Validate that scheduled tasks don't accumulate panics

## Sources

- **Debug Session:** `.planning/debug/scrapligo-snmp-panic.md` (MEDIUM confidence)
- **Code Review Report:** `docs/code-review-report.md` (HIGH confidence) 
- **Code Fix Guide:** `docs/code-fix-guide.md` (HIGH confidence)
- **Source Code Analysis:** `internal/device/scrapli_wrapper.go` (HIGH confidence)
- **Source Code Analysis:** `internal/api/v1/network/network_export_handler.go` (HIGH confidence)
- **Project Documentation:** `CLAUDE.md` (HIGH confidence)
- **Go Concurrency Patterns:** Standard Go sync.RWMutex usage (HIGH confidence)

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| SNMP Transport Panics | HIGH | Debug session provides detailed stack trace evidence |
| Race Conditions | HIGH | Source code shows inadequate synchronization in connection pool |
| SQL Injection Risks | HIGH | Code review report documents specific vulnerable patterns |
| Export Implementation | HIGH | Existing export handler code shows the patterns to follow/avoid |
| Resource Management | MEDIUM | General Go patterns, needs verification across all code paths |
| Configuration Issues | MEDIUM | Inferred from common patterns, needs systematic review |

## Gaps to Address

- **Performance Impact**: The 100ms grace period in Close() needs performance testing under load
- **Memory Profiling**: Large export operations need memory profiling to identify bottlenecks
- **Deadlock Analysis**: Complex interaction between opMu, stateMu, and closing channel needs formal analysis
- **Connection Pool Sizing**: Current pool size limits may not be optimal for the new synchronization patterns
- **Error Recovery**: Panic recovery strategy should be tested systematically across all failure modes

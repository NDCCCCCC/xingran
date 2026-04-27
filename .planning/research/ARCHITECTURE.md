# Architecture Patterns: Technical Debt Cleanup Milestone

**Domain:** Technical Debt Cleanup - SNMP Panic Fix, Backend Optimization, Network Device Export
**Researched:** 2025-04-27
**Overall confidence:** HIGH

## Executive Summary

The technical debt cleanup milestone addresses three critical architectural concerns:

1. **SNMP Panic Fix**: Resolves scrapligo transport layer crashes during device connectivity checks through enhanced connection state management and panic recovery mechanisms
2. **Backend Code Optimization**: Simplifies the monolithic Core structure, removes dead code, and fixes security issues to improve maintainability
3. **Network Device Export**: Integrates export functionality into existing architecture using the established Handler-Service pattern with Excel generation via excelize

All three areas leverage existing architectural patterns without introducing new dependencies or breaking changes. The work primarily strengthens existing infrastructure rather than adding new capabilities.

## Recommended Architecture

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| **SNMP Panic Fix** | | |
| `internal/device/scrapli_wrapper.go` | Connection state management, panic recovery | connection_pool, task_scheduler |
| `internal/device/connection_pool.go` | Connection lifecycle, readiness validation | scrapli_wrapper, device_executor |
| `internal/device/executor.go` | Command execution with retry logic | task_scheduler, scrapli_wrapper |
| **Backend Optimization** | | |
| `internal/core/core.go` | Simplified dependency injection | All service modules |
| `internal/services/network/` | Network device business logic | api/v1/network, models |
| **Network Device Export** | | |
| `internal/api/v1/network/network_export_handler.go` | Export HTTP endpoints | services/network, excelize |
| `internal/services/network/` | Device data retrieval | models, cache_provider |

### Data Flow

#### SNMP Panic Fix Flow
```
Device Connection Request
    ↓
DeviceConnectionPool.GetConnection()
    ↓
checkDeviceReachable() - TCP connectivity test
    ↓
ScrapliWrapper.OpenContext() - Async connection with panic recovery
    ↓
WaitForReady() - Polling loop with GetPrompt() validation
    ↓
StateReady ← Connection fully initialized
    ↓
Return PooledConnection to caller
```

#### Network Device Export Flow
```
POST /network/devices/export
    ↓
NetworkExportHandler.ExportDevices()
    ↓
networkService.List() - Fetch device data
    ↓
excelize.NewFile() - Create Excel workbook
    ↓
Write headers + data rows
    ↓
Set HTTP headers + stream response
    ↓
Client downloads .xlsx file
```

## Patterns to Follow

### Pattern 1: Connection State Machine with Panic Recovery

**What:** State-based connection lifecycle with explicit panic recovery at each transition point

**When:** Network device connections using scrapligo transport layer

**Example:**
```go
// ConnectionState enum defines valid states
type ConnectionState int
const (
    StateInitializing ConnectionState = iota
    StateReady
    StateClosing
    StateClosed
)

// Panic recovery in OpenContext
defer func() {
    if r := recover(); r != nil {
        w.setState(StateClosed)
        resultCh <- result{err: fmt.Errorf("连接 panic: %v", r)}
    }
}()

// Validate readiness before returning
if err := wrapper.WaitForReady(30 * time.Second); err != nil {
    return nil, fmt.Errorf("等待连接就绪失败: %w", err)
}
```

**Key principles:**
- State transitions are atomic and protected by mutex
- All public methods validate state before operations
- Panic recovery at every goroutine boundary
- Explicit readiness validation before returning connections

### Pattern 2: Handler-Service with Cache Provider

**What:** Service layer with optional caching via CacheProvider interface

**When:** Business logic that benefits from caching (high-frequency queries)

**Example:**
```go
// Service interface
type CacheService interface {
    List(ctx context.Context, req *ListDeviceRequest) ([]models.NetworkDevice, int64, error)
    GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error)
    InvalidateDeviceCache(ctx context.Context, deviceID string) error
}

// Implementation with cache
type cacheServiceImpl struct {
    db     *gorm.DB
    base   *NetworkDeviceService
    cache  systemServices.CacheProvider
    config *CacheConfigService
}

// Cached method
func (s *cacheServiceImpl) GetDeviceStatistics(ctx context.Context) (map[string]interface{}, error) {
    cacheKey := "network_device:statistics"
    var result map[string]interface{}
    expiration := s.getExpiration("cache.network_device.statistics", 3*time.Minute)
    err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
        return s.base.GetDeviceStatistics(ctx)
    })
    return result, err
}
```

### Pattern 3: Excel Export via excelize

**What:** Server-side Excel generation using excelize library

**When:** Exporting large datasets that don't fit in browser memory

**Example:**
```go
// Create workbook
file := excelize.NewFile()
sheet := "网络设备"
file.SetSheetName("Sheet1", sheet)

// Write headers
headers := []string{"设备名称", "IP地址", "状态"}
for i, header := range headers {
    cell, _ := excelize.CoordinatesToCellName(i+1, 1)
    file.SetCellValue(sheet, cell, header)
}

// Write data rows
for i, device := range devices {
    row := i + 2
    writeRow(file, sheet, row, []interface{}{
        device.DeviceName, device.IPAddress, device.Status,
    })
}

// Stream response
var buf bytes.Buffer
file.WriteTo(&buf)
c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
c.Data(http.StatusOK, "...", buf.Bytes())
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Ignoring Transport Initialization Race

**What:** Assuming scrapligo connection is ready after `Open()` returns

**Why bad:** Transport layer goroutines may still be initializing, causing panics on subsequent operations

**Instead:**
```go
// ❌ WRONG
if err := wrapper.Open(); err != nil {
    return err
}
// Assume ready - may panic!

// ✅ CORRECT
if err := wrapper.OpenContext(ctx); err != nil {
    return err
}
if err := wrapper.WaitForReady(30 * time.Second); err != nil {
    return err
}
// Explicitly validated readiness
```

### Anti-Pattern 2: Core as God Object

**What:** Core struct with 20+ fields, all services initialized in `Init()`

**Why bad:** Hard to test, circular dependencies, unclear initialization order

**Instead:**
```go
// ❌ WRONG
type Core struct {
    DB *gorm.DB
    Cache Cache
    UserService *UserService
    RoleService *RoleService
    // ... 20 more services
}

// ✅ CORRECT - Lazy initialization or service locator
type Core struct {
    db     *gorm.DB
    cache  Cache
    // Only essential infrastructure
}

func (c *Core) UserService() *UserService {
    return NewUserService(c.db, c.cache)
}
```

### Anti-Pattern 3: Export Without Pagination Limits

**What:** Export endpoints that fetch all rows without limits

**Why bad:** Memory exhaustion on large datasets

**Instead:**
```go
// ❌ WRONG
devices, _, err := service.List(ctx, &ListRequest{
    PageSize: 0, // No limit!
})

// ✅ CORRECT
const maxExportRows = 100000
devices, _, err := service.List(ctx, &ListRequest{
    Current: 1,
    PageSize: maxExportRows,
})
```

## Scalability Considerations

| Concern | At 100 devices | At 10K devices | At 100K devices |
|---------|---------------|---------------|----------------|
| **SNMP connectivity checks** | Sequential OK | Concurrent pool (20 workers) | Batch processing with queue |
| **Connection pool size** | 5 connections | 20 connections | 50 connections + partitioning |
| **Export generation** | On-demand sync | On-demand with caching | Async job + download link |
| **Core initialization** | <100ms | <500ms | Lazy service loading |

## Integration Points

### 1. SNMP Panic Fix → Existing Architecture

**New components:** None (enhancement only)

**Modified components:**
- `internal/device/scrapli_wrapper.go` - Add state machine, panic recovery
- `internal/device/connection_pool.go` - Add readiness validation
- `internal/device/executor.go` - Already has panic recovery

**Data flow changes:**
- Connection creation: Add `WaitForReady()` validation before returning
- Connection reuse: Validate `IsReady()` state before acquiring
- Error handling: Convert panics to errors gracefully

**Backward compatibility:** Full - API unchanged, internal behavior enhanced

### 2. Backend Optimization → Existing Architecture

**New components:** None (removal/simplification)

**Modified components:**
- `internal/core/core.go` - Remove dead code, simplify Init()
- `internal/services/network/` - Consolidate redundant services

**Data flow changes:**
- Service initialization: Lazy load instead of eager init
- Cache access: Use CacheProvider interface consistently

**Backward compatibility:** Full - public APIs unchanged

### 3. Network Device Export → Existing Architecture

**New components:**
- `internal/api/v1/network/network_export_handler.go` - Already exists!
- Uses excelize v2.10.0 (already in dependencies)

**Modified components:**
- `internal/api/v1/network/network_router.go` - Register export routes

**Data flow changes:**
- New endpoint: `POST /network/devices/export`
- Leverages existing `networkService.List()` for data
- Stream Excel response directly to client

**Backward compatibility:** Full - additive feature only

## Build Order

### Phase 1: SNMP Panic Fix (Critical Path)

1. **Enhance scrapli_wrapper.go**
   - Add ConnectionState enum and stateMu mutex
   - Implement `OpenContext()` with panic recovery
   - Add `WaitForReady()` with polling validation
   - Update `Close()` to use opMu.Lock() for safety

2. **Update connection_pool.go**
   - Add readiness check in `GetConnection()`
   - Call `WaitForReady(30*time.Second)` before returning
   - Add `IsReady()` validation in `Acquire()`

3. **Test under load**
   - Concurrent connection attempts (20+ workers)
   - Verify no panics in logs
   - Measure connection success rate

**Dependencies:** None (self-contained)

**Testing strategy:**
- Unit tests for state transitions
- Integration tests with real network devices
- Load tests with 100+ concurrent connections

### Phase 2: Backend Optimization (Can parallelize with Phase 3)

1. **Audit Core struct**
   - Identify unused fields (grep for references)
   - Map initialization order dependencies

2. **Simplify Core.Init()**
   - Remove commented code
   - Extract service initialization to separate functions
   - Add error recovery for non-critical services

3. **Remove dead code**
   - Delete unused legacy services
   - Consolidate duplicate service implementations

4. **Fix security issues**
   - LDAP: Remove `InsecureSkipVerify: true`
   - Add proper TLS config with certificate validation

**Dependencies:** None (clean-up work)

**Testing strategy:**
- Verify all services still initialize correctly
- Check for circular dependency errors
- Integration tests for LDAP connections

### Phase 3: Network Device Export (Can parallelize with Phase 2)

1. **Register export routes**
   - Add routes in `network_router.go`
   - Map export endpoints to handler methods

2. **Implement export methods** (already exists!)
   - `ExportDevices` - Network device list
   - `ExportCredentials` - Auth credentials
   - `ExportBackups` - Configuration backups

3. **Add frontend integration**
   - Export button in device list UI
   - Download progress indicator
   - Error handling for large datasets

**Dependencies:** Phase 1 complete (stable device service)

**Testing strategy:**
- Export with 10, 100, 1000 devices
- Verify Excel file format and data accuracy
- Performance test with max export rows (100K)

## Sources

### SNMP Panic Fix Research
- **Source Code Analysis:** `internal/device/scrapli_wrapper.go` - Lines 268-390 (OpenContext panic recovery)
- **Source Code Analysis:** `internal/device/connection_pool.go` - Lines 192-281 (GetConnection flow)
- **Source Code Analysis:** `internal/device/executor.go` - Lines 222-237 (Panic recovery pattern)
- **Confidence:** HIGH - Root cause and solution pattern verified in existing code

### Backend Optimization Research
- **Source Code Analysis:** `internal/core/core.go` - Lines 42-75 (Core struct definition)
- **Source Code Analysis:** `docs/code-fix-guide.md` - Known issues and patterns
- **Documentation:** `docs/代码简化优化总结.md` - Previous optimization efforts
- **Confidence:** MEDIUM - Simplification straightforward, security issues need validation

### Network Device Export Research
- **Source Code Analysis:** `internal/api/v1/network/network_export_handler.go` - Complete implementation exists
- **Pattern Reference:** `internal/services/operations/excel_service.go` - Excel operations pattern
- **Library Documentation:** excelize v2.10.0 (already in go.mod)
- **Confidence:** HIGH - Export handler already implemented, just needs route registration

### Architecture Patterns
- **Project Documentation:** `docs/项目概述和架构设计.md` - Overall architecture
- **Development Standards:** `docs/开发规范.md` - Coding conventions
- **API Standards:** `docs/API响应规范.md` - Response format requirements
- **Confidence:** HIGH - Established patterns in existing codebase

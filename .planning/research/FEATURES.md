# Feature Landscape

**Domain:** 技术债清理里程碑 (v1.3)
**Researched:** 2026-04-27
**Overall confidence:** MEDIUM

## Executive Summary

The v1.3 milestone focuses on technical debt cleanup across three critical areas: SNMP panic fixes, backend code optimization, and network device export functionality. Research reveals these are interconnected issues - the SNMP panics stem from scrapligo transport layer concurrency issues during SNMP Ping operations, code optimization involves removing accumulated dead code from legacy iterations, and network device export builds upon existing Excel export infrastructure. The complexity ranges from LOW (code cleanup) to HIGH (SNMP panic resolution due to third-party library limitations).

## Key Findings

**Stack**: Go 1.24 + scrapligo v1.3.3 + gosnmp + excelize v2.10.0
**Architecture**: Handler-Service pattern with dependency injection, dual cache architecture, panic recovery patterns
**Critical pitfall**: scrapligo transport layer panics during concurrent SNMP operations due to goroutine synchronization issues in Channel.GetPrompt

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **SNMP Panic Recovery** | System crashes during scheduled SNMP monitoring tasks | HIGH | scrapligo v1.3.3 transport.(*Standard).Write panic in Channel.GetPrompt, intermittent but critical |
| **Dead Code Removal** | Codebase accumulated unused services from previous iterations | LOW | 2 confirmed dead files: `dashboard_service.go`, `settings_service.go` (zero external refs) |
| **Core Struct Cleanup** | Core struct carries 12 dead fields increasing memory footprint | MEDIUM | Fields only used internally or created elsewhere (DeviceManager, NetworkDeviceService, etc.) |
| **Network Device Export** | Users need to export device configurations for audits/backups | MEDIUM | Existing Excel infrastructure supports this, gaps in configuration content export |
| **Security Hardening** | WebSocket CheckOrigin bypass, race conditions in scheduler | MEDIUM | Identified but not yet addressed in production |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Panic-Resilient SNMP** | Automatic recovery from scrapligo panics without service interruption | HIGH | Requires goroutine-level isolation and recovery patterns |
| **Configuration Diff Export** | Export device configurations with change tracking (Git-like) | HIGH | Beyond basic export, requires version tracking and diff generation |
| **Vendor-Specific Export Formats** | Native format support (Cisco .cfg, Huawei .zip, H3C compatibility) | MEDIUM | Vendors have different config structures and compression preferences |
| **Scheduled Backup Automation** | Automated configuration exports with retention policies | MEDIUM | Builds on existing scheduler infrastructure |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Upgrade scrapligo** | Breaking changes, uncertain fix for panic, extensive testing required | Implement defensive recovery patterns around current version |
| **Real-time config streaming** | Over-engineered for current use case, adds complexity | Batch export with scheduling is sufficient |
| **Multi-vendor config normalization** | Vendors have incompatible config syntax, loss of fidelity | Preserve native format, let vendor tools handle normalization |
| **Full config analysis in export** | Export should be dumb (data dump), analysis is separate concern | Separate analysis service if needed |

## Feature Dependencies

```
Core Struct Cleanup → Dead Code Removal (simplifies Core first)
Network Device Export → Excel Export Infrastructure (already exists)
SNMP Panic Fix → Panic Recovery Patterns (requires implementation)
Configuration Backup → Network Device Export (backup is specialized export)
```

**Dependency Chain:**
1. Code cleanup (independent, can start immediately)
2. Network device export (independent, uses existing patterns)
3. SNMP panic fix (independent, highest priority due to production impact)

## MVP Recommendation

**Prioritize for v1.3:**
1. **SNMP Panic Recovery** (HIGH priority, production impact)
   - Implement goroutine isolation for SNMP operations
   - Add panic recovery middleware around scrapligo calls
   - Add circuit breaker pattern for failing SNMP connections

2. **Dead Code Removal** (LOW hanging fruit, quick win)
   - Delete 2 confirmed dead service files
   - Remove 12 dead fields from Core struct
   - Verify `go build ./...` passes after each removal

3. **Network Device Configuration Export** (MEDIUM complexity, high value)
   - Extend existing Excel export infrastructure
   - Support device configuration dump (running-config)
   - Add backup metadata (timestamp, version, device info)

**Defer to v1.4:**
- **Configuration Diff Analysis**: Requires version control infrastructure
- **Vendor-Specific Config Validation**: Requires deep vendor knowledge base
- **Real-time Config Monitoring**: Different use case, separate architecture

## Detailed Feature Analysis

### 1. SNMP Panic Recovery

**Current State:**
- Intermittent panic in `transport.(*Standard).Write` during `Channel.GetPrompt`
- Occurs during scheduled SNMP Ping operations (cron-triggered)
- Stack trace shows goroutine 72399 panic in scrapligo transport layer
- Error location: `scrapligo/transport/standard.go:249`

**Root Cause Hypothesis:**
- scrapligo v1.3.3 spawns internal goroutines for async operations
- `GetPrompt` creates child goroutine that calls `Write` on transport
- Race condition: transport closed while child goroutine still writing
- SNMP Ping operations run concurrently via scheduler, exacerbating the issue

**Expected Behavior:**
1. **Prevention**: Avoid triggering conditions that cause panic
   - Limit concurrent SNMP operations per device
   - Add connection state validation before SNMP operations
   - Implement connection pooling with health checks

2. **Recovery**: Graceful degradation when panic occurs
   - Catch panic at operation boundary (not process level)
   - Log panic details for debugging
   - Retry with exponential backoff
   - Mark device as "unknown" status vs crashing service

3. **Isolation**: Contain blast radius
   - Run SNMP operations in isolated goroutines
   - Use timeout contexts to prevent hanging
   - Implement circuit breaker per device

**Implementation Patterns:**
```go
// Pattern 1: Goroutine-level panic recovery
func safeSNMPOperation(deviceID string) (result bool, err error) {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[SNMP Panic] Recovered for device %s: %v", deviceID, r)
            err = fmt.Errorf("SNMP operation panic: %v", r)
        }
    }()
    // SNMP operation here
    return true, nil
}

// Pattern 2: Connection state validation
func (c *SNMPClient) IsReady() bool {
    if c.client == nil || c.client.Conn == nil {
        return false
    }
    // Quick check without triggering GetPrompt
    return true
}

// Pattern 3: Circuit breaker per device
type DeviceCircuitBreaker struct {
    failures    map[string]int
    lastFailure map[string]time.Time
    mutex       sync.RWMutex
}
```

**Anti-Patterns to Avoid:**
- ❌ Global panic recovery in `main()` (too coarse, hides issues)
- ❌ Upgrading scrapligo without testing (breaking changes risk)
- ❌ Removing SNMP functionality altogether (degrades monitoring)
- ❌ Adding mutexes around all SNMP calls (performance killer)

### 2. Backend Code Optimization

**Current State:**
- Core struct has 12 dead fields (only created/closed internally)
- 2 service files with zero external references
- WebSocket CheckOrigin allows any origin (security issue)
- Race condition in GlobalDeviceMonitorService

**Expected Behavior:**

**Phase 1: Delete Dead Code**
1. Remove unused service files:
   - `internal/services/dashboard_service.go` (router uses `system/` version)
   - `internal/services/settings_service.go` (Core uses `system/` version)

2. Remove Core struct dead fields:
   - `DeviceManager` (only created/closed internally in Core)
   - `NetworkDeviceService` (network router creates its own instance)
   - `DeviceMonitorService` (accessed via scheduler global)
   - `UserService` through `SettingsService` (7 fields, only for cache warm-up)
   - `APIMetadata` (only used internally)

3. Extract `RPAScalingService` from `interface{}` to proper interface

**Phase 2: Fix Security Issues**
1. WebSocket CheckOrigin validation:
   ```go
   // Current: allows any origin
   upgrader.CheckOrigin = func(r *http.Request) bool { return true }

   // Fixed: validate against allowed origins
   upgrader.CheckOrigin = func(r *http.Request) bool {
       origin := r.Header.Get("Origin")
       return isValidOrigin(origin)
   }
   ```

2. Fix GlobalDeviceMonitorService race condition:
   - Add `sync.RWMutex` around service state access
   - Protect concurrent reads/writes to device status map

3. Add error logging for silent failures:
   - Login log creation failures currently silent
   - Add structured logging with context

**Phase 3: Migrate Flat Service Files**
- Move `internal/services/duty_*.go` (6 files) → `internal/services/duty/`
- Move `internal/services/notice_*.go` (5 files) → `internal/services/notice/`
- Move `internal/services/device_*.go` (4 files) → `internal/services/device/`
- Move `internal/services/knowledge_service.go` → `internal/services/knowledge/`
- Update all import paths across codebase

**Expected Outcomes:**
- Reduced memory footprint (12 fewer fields in Core)
- Improved code organization (services grouped by domain)
- Eliminated security vulnerabilities (CORS, race conditions)
- Better maintainability (clearer module boundaries)

### 3. Network Device Export

**Current State:**
- Existing Excel export infrastructure in `internal/services/operations/excel_export_service.go`
- Network device export handler exists but incomplete
- Configuration backup service exists but separate from export

**Expected Behavior:**

**Export Types:**
1. **Device List Export** (basic metadata)
   - Device name, IP, vendor, model, status
   - Department, location, credentials
   - Last seen timestamp
   - Already implemented in `network_export_handler.go`

2. **Configuration Export** (NEW)
   - Running configuration text (vendor-specific format)
   - Configuration metadata (version, timestamp, size)
   - Change tracking (diff from previous version)
   - Compression option for large configs

3. **Batch Export** (NEW)
   - Export multiple devices to separate files
   - ZIP archive with device-name structure
   - Summary manifest with device metadata

**Export Formats:**
- **Excel (.xlsx)**: Device lists, metadata (human-readable)
- **Text (.txt/.cfg)**: Raw configuration dump (vendor-native)
- **ZIP Archive**: Batch exports with compression
- **JSON** (future): Structured data for automation

**Configuration Content:**
```go
type ConfigExportData struct {
    DeviceID       string
    DeviceName     string
    IPAddress      string
    Vendor         models.DeviceVendor
    ConfigContent  string    // running-config output
    ConfigSize     int       // bytes
    BackupType     models.BackupType
    Version        int       // backup version number
    Compressed     bool
    ExportedAt     time.Time
    ExportedBy     string
}
```

**Export Workflow:**
1. User initiates export (UI or API)
2. Service fetches device configuration via `DeviceExecutor.GetConfig()`
3. Store configuration in database or file system
4. Generate Excel file with device list + configuration references
5. For direct config export: offer text file download
6. For batch export: create ZIP archive with multiple configs

**Integration Points:**
- Reuse `excel_export_service.go` infrastructure
- Extend `config_backup_service.go` for export functionality
- Add export endpoint to network router
- Frontend: add "Export Configuration" button to device list

**Best Practices:**
- **Configuration Retrieval**: Use vendor-specific commands
  - Cisco: `show running-config`
  - Huawei/H3C: `display current-configuration`
  - Juniper: `show configuration`

- **Handling Large Configs**:
  - Compress configs > 100KB (configurable threshold)
  - Stream large files instead of loading in memory
  - Provide progress feedback for long-running exports

- **Security Considerations**:
  - Redact sensitive data (passwords, keys) before export
  - Encrypt exported files if containing credentials
  - Log all export operations with user attribution

## Complexity Assessment

| Feature | Technical Complexity | Risk Level | Dependencies | Time Estimate |
|---------|---------------------|------------|--------------|---------------|
| **SNMP Panic Fix** | HIGH | HIGH | scrapligo internals, scheduler | 2-3 days |
| **Dead Code Removal** | LOW | LOW | None | 0.5 day |
| **Core Struct Cleanup** | MEDIUM | MEDIUM | Multiple modules | 1 day |
| **Security Fixes** | MEDIUM | MEDIUM | WebSocket, scheduler | 1 day |
| **Service Migration** | MEDIUM | LOW | Import paths | 1 day |
| **Device List Export** | LOW | LOW | Existing infrastructure | 0.5 day |
| **Config Export** | MEDIUM | MEDIUM | DeviceExecutor, backup service | 1-2 days |
| **Batch Export** | MEDIUM | LOW | Single device export | 0.5 day |

## Implementation Risks

### SNMP Panic Fix
**Risk**: scrapligo internals not well understood, fix may be incomplete
**Mitigation**:
- Implement defense-in-depth (multiple recovery layers)
- Add comprehensive logging for debugging
- Test with various network conditions (timeout, congestion)
- Consider timeout-based isolation as fallback

### Code Cleanup
**Risk**: Removing code that appears unused but has runtime dependencies
**Mitigation**:
- Use `go build ./...` after each removal
- Check for reflection-based access patterns
- Verify import paths are updated correctly
- Keep git history for potential revert

### Network Export
**Risk**: Large configurations causing memory issues
**Mitigation**:
- Implement streaming for large files
- Add file size limits and validation
- Use compression for configs > threshold
- Provide progress feedback for long operations

## Testing Strategy

### SNMP Panic Recovery
```go
func TestSNMPPanicRecovery(t *testing.T) {
    // Simulate panic conditions
    // Verify recovery catches panic
    // Verify retry mechanism works
    // Verify device marked as unknown on failure
}
```

### Code Cleanup
```bash
# After each removal phase
go build ./...
go test ./...
```

### Export Functionality
```go
func TestDeviceConfigExport(t *testing.T) {
    // Test single device export
    // Test batch export
    // Test compression
    // Test redaction of sensitive data
}
```

## Success Criteria

### SNMP Panic Fix
- [ ] No service crashes during SNMP operations
- [ ] Panics logged and recovered gracefully
- [ ] Device monitoring continues after panic
- [ ] Circuit breaker prevents repeated failures

### Code Cleanup
- [ ] All dead code removed (verified by `go build`)
- [ ] Core struct reduced by 12 fields
- [ ] Security vulnerabilities fixed
- [ ] All tests passing after cleanup

### Network Export
- [ ] Device list export functional
- [ ] Configuration export retrieves running-config
- [ ] Batch export creates ZIP archive
- [ ] Export works for all supported vendors (Huawei, H3C, Ruijie, Maipu)

## Sources

### SNMP Panic Issue
- **Debug Session**: `.planning/debug/scrapligo-snmp-panic.md` (MEDIUM confidence - internal analysis)
- **Code Analysis**: `internal/device/scrapli_wrapper.go` panic recovery patterns (HIGH confidence)
- **Stack Trace**: `transport.(*Standard).Write` panic in `Channel.GetPrompt` (HIGH confidence)

### Code Optimization
- **Optimization Plan**: `.planning/quick/20260416-backend-optimize/PLAN.md` (HIGH confidence - approved plan)
- **Go Best Practices**: Internal codebase patterns (MEDIUM confidence)

### Network Export
- **Existing Infrastructure**: `internal/services/operations/excel_export_service.go` (HIGH confidence)
- **Network Export Handler**: `internal/api/v1/network/network_export_handler.go` (HIGH confidence)
- **Config Backup Service**: `internal/services/config_backup_service.go` (HIGH confidence)
- **Vendor Patterns**: `internal/device/snmp_client.go` vendor detection logic (HIGH confidence)

### External References
- **scrapligo Documentation**: https://github.com/scrapli/scrapligo (LOW confidence - unable to access due to rate limits)
- **Go Refactoring Patterns**: Standard Go community practices (MEDIUM confidence)
- **Network Config Management**: Industry best practices (MEDIUM confidence - general knowledge)

## Gaps Requiring Phase-Specific Research

1. **scrapligo Internals**: Need deeper understanding of transport layer goroutine lifecycle for effective panic prevention
2. **Vendor Config Formats**: Specific format differences between Huawei/H3C/Ruijie/Maipu configurations
3. **Compression Strategies**: Optimal compression algorithms for network configurations
4. **Large File Handling**: Streaming patterns for multi-MB configuration files

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| SNMP Panic Analysis | MEDIUM | Root cause hypothesized but not confirmed, fix patterns identified |
| Code Cleanup | HIGH | Dead code confirmed via static analysis, low risk |
| Security Fixes | HIGH | Issues clearly identified, fixes straightforward |
| Network Export | MEDIUM | Infrastructure exists, configuration export needs vendor-specific testing |
| Vendor Support | LOW | Need research into vendor-specific config retrieval commands |
| Testing Strategy | MEDIUM | Unit tests clear, integration testing needs device access |
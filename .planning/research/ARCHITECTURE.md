# Architecture Patterns: MAC地址历史数据管理

**Domain:** Network Device MAC Address Time-Series Data Management
**Researched:** 2026-05-09
**Overall confidence:** HIGH

## Executive Summary

MAC地址历史数据管理需要在现有MAC采集服务基础上，增加时间序列数据存储、变更检测和历史查询能力。基于代码分析，推荐采用**增量快照模式**，仅在MAC地址变化时记录历史，平衡存储效率与查询性能。

**Key Architecture Decisions:**
- **Database**: PostgreSQL原生时间序列优化（BRIN索引 + 分区表）
- **Storage Strategy**: 增量快照（仅记录变化）
- **Query Pattern**: 时态查询（AS OF SYSTEM TIME）+ 聚合统计
- **Integration**: 最小侵入式扩展现有MACCollectionService

---

## Database Schema Design

### Current State

**Existing Table:** `sys_device_mac_address`

```sql
CREATE TABLE sys_device_mac_address (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL,
    mac_address VARCHAR(30) NOT NULL,
    interface_name VARCHAR(100) NOT NULL,
    vlan_id INT,
    mac_type VARCHAR(20),  -- 'dynamic', 'static', 'secure'
    collected_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP
);

-- Indexes (Migration 019)
CREATE INDEX idx_sys_device_mac_address_composite 
  ON sys_device_mac_address(device_id, mac_address, interface_name);
CREATE INDEX idx_sys_device_mac_address_mac 
  ON sys_device_mac_address(mac_address);
```

**Current Limitations:**
- Migration 020删除了唯一索引，允许重复记录
- **无变更检测**: 每次采集全量删除旧数据（`mac_collection_service.go:201-206`）
- **无历史保留**: 仅保留最新快照
- **查询性能**: 无时态查询优化

---

### Proposed Schema: Time-Series History

**Option A: Dedicated History Table (Recommended)**

```sql
-- Migration: 12X_create_mac_history_table.sql

CREATE TABLE sys_device_mac_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    mac_address VARCHAR(30) NOT NULL,
    interface_name VARCHAR(100) NOT NULL,
    vlan_id INT,
    mac_type VARCHAR(20),
    
    -- Time-series fields
    collected_at TIMESTAMP NOT NULL,
    first_seen TIMESTAMP NOT NULL,     -- 首次发现时间
    last_seen TIMESTAMP NOT NULL,      -- 最后发现时间
    change_type VARCHAR(20) NOT NULL,  -- 'appeared', 'disappeared', 'moved', 'vlan_changed'
    
    -- Change tracking
    prev_interface VARCHAR(100),       -- Previous interface (if moved)
    prev_vlan INT,                     -- Previous VLAN (if changed)
    
    created_at TIMESTAMP DEFAULT NOW()
) PARTITION BY RANGE (collected_at);

-- BRIN index for time-series (efficient for large tables)
CREATE INDEX idx_mac_history_collected_at_brin 
  ON sys_device_mac_history USING BRIN (collected_at);

-- B-tree indexes for common queries
CREATE INDEX idx_mac_history_device_mac 
  ON sys_device_mac_history(device_id, mac_address);
CREATE INDEX idx_mac_history_mac 
  ON sys_device_mac_history(mac_address);
CREATE INDEX idx_mac_history_change_type 
  ON sys_device_mac_history(change_type, collected_at DESC);

-- Partitioning (monthly partitions)
-- Automatically created via trigger or pg_partman
CREATE TABLE sys_device_mac_history_2026_05 
  PARTITION OF sys_device_mac_history 
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
```

**Why This Design:**

1. **Partitioning by `collected_at`**: 
   - Time-series data naturally partitions by time
   - Enables efficient partition pruning (查询最近数据时扫描更少分区)
   - 支持定期归档/删除旧分区

2. **BRIN Index on `collected_at`**:
   - BRIN (Block Range INdex) 比 B-tree 空间效率高100倍
   - 适合按时间插入且按时间范围查询的数据
   - PostgreSQL官方推荐用于时间序列

3. **Change Detection Fields**:
   - `change_type`: 记录事件类型（新增/消失/移动/VLAN变更）
   - `prev_interface/prev_vlan`: 追踪变化前的状态
   - `first_seen/last_seen`: 计算MAC存活时间

4. **Separate from Current Table**:
   - `sys_device_mac_address` 保留为"当前状态"表（最新快照）
   - `sys_device_mac_history` 保存所有历史变更
   - 查询"当前状态"时无需扫描历史表

---

**Option B: Hybrid Table (Not Recommended)**

```sql
-- 在现有表添加历史字段
ALTER TABLE sys_device_mac_address 
  ADD COLUMN change_type VARCHAR(20),
  ADD COLUMN prev_interface VARCHAR(100),
  ADD COLUMN is_history BOOLEAN DEFAULT FALSE;
```

**Why Not:**
- 混合当前状态和历史数据，查询复杂度高
- 无法利用分区表优势
- 与现有Migration 020冲突（已删除唯一索引）

---

### Integration Points with Existing MAC Collection

#### Current MAC Collection Flow

```
mac_collection_service.go:collectDeviceMAC()
  ├── Execute SNMP command → parseMACAddressTable()
  ├── Apply LLDP filter (Phase 11)
  ├── Apply MAC threshold filter (Phase 11)
  ├── Delete old records (Line 201-206) ⚠️ OVERWRITES HISTORY
  └── Insert new records (Line 237-243)
```

**Critical Change Required:**

**Location:** `internal/services/mac_collection_service.go:196-246`

**Current Code (Destructive):**
```go
// 1. 先删除该设备的旧MAC地址记录
if err := s.db.WithContext(ctx).
    Where("device_id = ?", device.ID).
    Delete(&models.DeviceMACAddress{}).Error; err != nil {
    // ...
}

// 2. 批量插入新的MAC地址记录
// ... (Line 209-243)
```

**Proposed Change (Incremental):**
```go
// 1. 获取当前快照（用于变更检测）
var currentMACs []models.DeviceMACAddress
s.db.WithContext(ctx).
    Where("device_id = ?", device.ID).
    Find(&currentMACs)

// 2. 计算差异（新增、消失、移动、VLAN变更）
changes := s.calculateMACChanges(currentMACs, filteredMACAddresses)

// 3. 记录历史（仅记录变化）
if len(changes) > 0 {
    s.recordMACHistory(ctx, device.ID, changes, collectionTime)
}

// 4. 更新当前状态表（全量替换，但保留历史）
if err := s.db.WithContext(ctx).
    Where("device_id = ?", device.ID).
    Delete(&models.DeviceMACAddress{}).Error; err != nil {
    return result
}
// ... (insert new records)
```

---

## Service Layer Architecture

### New Services to Create

#### 1. `MACHistoryService` (New)

**Location:** `internal/services/mac_history_service.go`

**Responsibilities:**
- 变更检测（比较两次采集的MAC地址表）
- 历史记录写入（变更事件到`sys_device_mac_history`）
- 时态查询（按时间点/范围查询历史状态）
- 聚合统计（MAC存活时间、变更频率、异常检测）

**Interface:**
```go
type MACHistoryService interface {
    // RecordChanges 记录MAC地址变更
    RecordChanges(ctx context.Context, deviceID string, 
        changes []MACChangeEvent, collectionTime time.Time) error
    
    // QueryHistory 查询历史快照（时态查询）
    QueryHistory(ctx context.Context, req HistoryQueryRequest) 
        ([]MACHistoryRecord, error)
    
    // GetMACTrajectory 查询单个MAC的移动轨迹
    GetMACTrajectory(ctx context.Context, macAddress string, 
        startTime, endTime time.Time) ([]MACTrajectoryPoint, error)
    
    // GetPortHistory 查询端口的历史MAC列表
    GetPortHistory(ctx context.Context, deviceID, interfaceName string, 
        startTime, endTime time.Time) ([]MACHistoryRecord, error)
    
    // DetectAnomalies 异常检测（如MAC频繁移动）
    DetectAnomalies(ctx context.Context, req AnomalyDetectionRequest) 
        ([]AnomalyEvent, error)
    
    // GetMACStatistics MAC地址统计（存活时间、变更频率等）
    GetMACStatistics(ctx context.Context, deviceID string, 
        timeRange TimeRange) (*MACStatistics, error)
}

// MACChangeEvent 变更事件
type MACChangeEvent struct {
    MACAddress    string
    InterfaceName string
    VLANID        *int
    MACType       models.MACType
    ChangeType    ChangeType // appeared, disappeared, moved, vlan_changed
    PrevInterface string     // Previous interface (if moved)
    PrevVLAN      *int       // Previous VLAN (if changed)
}

type ChangeType string

const (
    ChangeTypeAppeared    ChangeType = "appeared"     // 新增MAC
    ChangeTypeDisappeared ChangeType = "disappeared"  // MAC消失
    ChangeTypeMoved       ChangeType = "moved"        // MAC移动到其他端口
    ChangeTypeVLANChanged ChangeType = "vlan_changed" // VLAN变更
)
```

**Key Method - Change Detection:**
```go
// calculateMACChanges 比较当前快照和新采集的MAC地址，计算差异
func (s *MACHistoryService) calculateMACChanges(
    current []models.DeviceMACAddress,
    newMACs []MACAddressEntry,
) []MACChangeEvent {
    currentMap := make(map[string]models.DeviceMACAddress) // key: mac+interface
    newMap := make(map[string]MACAddressEntry)
    
    // Build maps for O(1) lookup
    for _, mac := range current {
        key := fmt.Sprintf("%s|%s", mac.MACAddress, mac.InterfaceName)
        currentMap[key] = mac
    }
    for _, mac := range newMACs {
        key := fmt.Sprintf("%s|%s", mac.MACAddress, mac.InterfaceName)
        newMap[key] = mac
    }
    
    var changes []MACChangeEvent
    
    // Detect appeared (new MACs)
    for key, newMAC := range newMap {
        if _, exists := currentMap[key]; !exists {
            // Check if MAC existed on different interface (moved)
            movedFrom := s.findMACOnDifferentInterface(newMAC.MACAddress, current)
            if movedFrom != "" {
                changes = append(changes, MACChangeEvent{
                    MACAddress:    newMAC.MACAddress,
                    InterfaceName: newMAC.InterfaceName,
                    VLANID:        newMAC.VLANIDPtr,
                    MACType:       newMAC.MACType,
                    ChangeType:    ChangeTypeMoved,
                    PrevInterface: movedFrom,
                })
            } else {
                changes = append(changes, MACChangeEvent{
                    MACAddress:    newMAC.MACAddress,
                    InterfaceName: newMAC.InterfaceName,
                    VLANID:        newMAC.VLANIDPtr,
                    MACType:       newMAC.MACType,
                    ChangeType:    ChangeTypeAppeared,
                })
            }
        }
    }
    
    // Detect disappeared (removed MACs)
    for key, oldMAC := range currentMap {
        if _, exists := newMap[key]; !exists {
            changes = append(changes, MACChangeEvent{
                MACAddress:    oldMAC.MACAddress,
                InterfaceName: oldMAC.InterfaceName,
                VLANID:        oldMAC.VLANID,
                MACType:       oldMAC.MACType,
                ChangeType:    ChangeTypeDisappeared,
            })
        }
    }
    
    // Detect VLAN changes
    for key, newMAC := range newMap {
        if oldMAC, exists := currentMap[key]; exists {
            if s.vlanChanged(oldMAC.VLANID, newMAC.VLANID) {
                changes = append(changes, MACChangeEvent{
                    MACAddress:    newMAC.MACAddress,
                    InterfaceName: newMAC.InterfaceName,
                    VLANID:        newMAC.VLANIDPtr,
                    MACType:       newMAC.MACType,
                    ChangeType:    ChangeTypeVLANChanged,
                    PrevVLAN:      oldMAC.VLANID,
                })
            }
        }
    }
    
    return changes
}
```

---

#### 2. Modified `MACCollectionService`

**Location:** `internal/services/mac_collection_service.go`

**Changes:**
- Inject `MACHistoryService` dependency
- Call `RecordChanges()` before replacing current state
- Add toggle to disable history recording (for backward compatibility)

**New Constructor:**
```go
type MACCollectionService struct {
    db              *gorm.DB
    executor        *device.DeviceExecutor
    maxConcurrent   int
    lldpService     *lldp.LLDPService
    filterRuleService topology.FilterRuleService
    historyService  MACHistoryService  // NEW
    enableHistory   bool               // NEW: configurable
}

func NewMACCollectionService(
    db *gorm.DB, 
    executor *device.DeviceExecutor, 
    lldpSvc *lldp.LLDPService, 
    filterRuleSvc topology.FilterRuleService,
    historySvc MACHistoryService,  // NEW
) *MACCollectionService {
    svc := &MACCollectionService{
        db:                db,
        executor:          executor,
        maxConcurrent:     10,
        lldpService:       lldpSvc,
        filterRuleService: filterRuleSvc,
        historyService:    historySvc,
        enableHistory:     true,  // Read from config
    }
    svc.loadConfigFromDB()
    return svc
}
```

**Modified Collection Flow:**
```go
// collectDeviceMAC 采集设备MAC地址表（内部方法）
func (s *MACCollectionService) collectDeviceMAC(ctx context.Context, device *models.NetworkDevice) *MACCollectionResult {
    // ... (existing code until Line 194)
    
    // NEW: 获取当前快照用于变更检测
    var currentMACs []models.DeviceMACAddress
    if s.enableHistory {
        s.db.WithContext(ctx).
            Where("device_id = ?", device.ID).
            Find(&currentMACs)
    }
    
    // Calculate changes (if history enabled)
    if s.enableHistory && len(currentMACs) > 0 {
        changes := s.historyService.calculateMACChanges(currentMACs, filteredMACAddresses)
        if len(changes) > 0 {
            if err := s.historyService.RecordChanges(ctx, device.ID, changes, collectionTime); err != nil {
                applogger.Warnf("[MAC采集] %s: 记录历史失败: %v", device.DeviceName, err)
                // Don't fail collection if history recording fails
            }
        }
    }
    
    // Replace current state (existing code)
    if err := s.db.WithContext(ctx).
        Where("device_id = ?", device.ID).
        Delete(&models.DeviceMACAddress{}).Error; err != nil {
        result.ErrorMessage = fmt.Sprintf("删除旧MAC地址失败: %v", err)
        return result
    }
    // ... (rest of existing code)
}
```

---

### API Layer Design

#### New Handler: `MACHistoryHandler`

**Location:** `internal/api/v1/network/mac_history_handler.go`

**Endpoints:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/network/mac/history/query` | 时态查询（历史快照） |
| POST | `/network/mac/history/trajectory` | MAC移动轨迹 |
| POST | `/network/mac/history/port` | 端口历史MAC列表 |
| POST | `/network/mac/history/anomalies` | 异常事件查询 |
| POST | `/network/mac/history/statistics` | 统计数据 |
| POST | `/network/mac/history/export` | 历史数据导出 |

**Request/Response Examples:**

1. **Query Historical Snapshot**
```typescript
// Request
interface HistoryQueryRequest {
    deviceId?: string;
    macAddress?: string;
    interfaceName?: string;
    asOfTime: string;           // ISO 8601 timestamp
    timeRange?: {
        start: string;
        end: string;
    };
    changeType?: ('appeared' | 'disappeared' | 'moved' | 'vlan_changed')[];
    current: number;
    pageSize: number;
}

// Response
interface HistoryQueryResponse {
    list: MACHistoryRecord[];
    total: number;
    snapshotTime: string;  // The as-of time for this snapshot
}

interface MACHistoryRecord {
    id: string;
    deviceId: string;
    deviceName: string;
    macAddress: string;
    interfaceName: string;
    vlanId?: number;
    macType: string;
    collectedAt: string;
    changeType: string;
    prevInterface?: string;
    prevVlan?: number;
}
```

2. **MAC Trajectory**
```typescript
// Request
interface MACTrajectoryRequest {
    macAddress: string;
    startTime: string;
    endTime: string;
    deviceId?: string;  // Optional: filter by device
}

// Response
interface MACTrajectoryResponse {
    macAddress: string;
    duration: number;  // Total observation time (seconds)
    hopCount: number;  // Number of moves
    trajectory: TrajectoryPoint[];
}

interface TrajectoryPoint {
    deviceId: string;
    deviceName: string;
    interfaceName: string;
    vlanId?: number;
    firstSeen: string;
    lastSeen: string;
    duration: number;  // Seconds spent on this port
}
```

3. **Anomaly Detection**
```typescript
// Request
interface AnomalyDetectionRequest {
    deviceId?: string;
    startTime: string;
    endTime: string;
    threshold: {
        minHopCount?: number;        // Default: 10
        minDuration?: number;        // Default: 3600s (1 hour)
        maxDisappearedTime?: number; // Default: 86400s (1 day)
    };
}

// Response
interface AnomalyEvent {
    id: string;
    deviceId: string;
    deviceName: string;
    macAddress: string;
    anomalyType: 'frequent_movement' | 'long_absence' | 'unexpected_appearance';
    severity: 'low' | 'medium' | 'high';
    description: string;
    firstSeen: string;
    lastSeen: string;
    metrics: {
        hopCount: number;
        duration: number;
        interfaces: string[];
    };
}
```

**Router Setup:**
```go
// internal/api/v1/network/mac_router.go
func SetupMACRouter(r *gin.RouterGroup, core *core.Core, exportHandler *NetworkExportHandler) {
    handler := NewMACHandler(core)
    historyHandler := NewMACHistoryHandler(core)  // NEW

    // Existing endpoints
    r.POST("/list", handler.List)
    r.POST("/collect", handler.Collect)
    r.POST("/collect/all", handler.CollectAll)
    r.GET("/statistics", handler.GetStats)
    r.POST("/clean", handler.Clean)
    r.POST("/batch-delete", handler.BatchDelete)
    r.POST("/export", exportHandler.ExportMACAddresses)

    // NEW: History endpoints
    r.POST("/history/query", historyHandler.QueryHistory)
    r.POST("/history/trajectory", historyHandler.GetMACTrajectory)
    r.POST("/history/port", historyHandler.GetPortHistory)
    r.POST("/history/anomalies", historyHandler.GetAnomalies)
    r.POST("/history/statistics", historyHandler.GetStatistics)
    r.POST("/history/export", historyHandler.ExportHistory)
}
```

---

## Data Flow Diagrams

### MAC Collection with History Recording

```
User Trigger: "Collect MAC Addresses"
         ↓
MACCollectionService.CollectDevice()
         ↓
┌─────────────────────────────────────────┐
│ 1. Execute SNMP command                  │
│ 2. Parse MAC address table               │
│ 3. Apply LLDP filter (exclude uplinks)   │
│ 4. Apply MAC threshold filter            │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 5. Load current snapshot from DB         │
│    SELECT * FROM sys_device_mac_address  │
│    WHERE device_id = ?                   │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 6. Calculate changes                     │
│    - Compare current vs new              │
│    - Detect appeared/disappeared/moved   │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 7. Record changes to history (NEW)       │
│    INSERT INTO sys_device_mac_history    │
│    (change_type, prev_interface, ...)    │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 8. Replace current state                 │
│    DELETE FROM sys_device_mac_address    │
│    WHERE device_id = ?                   │
│    INSERT new records                    │
└─────────────────────────────────────────┘
         ↓
Return success
```

### Historical Query Flow

```
User Request: "Show MAC address table as of 2026-05-01 10:00"
         ↓
MACHistoryHandler.QueryHistory()
         ↓
┌─────────────────────────────────────────┐
│ 1. Query historical events              │
│    SELECT * FROM sys_device_mac_history  │
│    WHERE collected_at <= '2026-05-01...' │
│    ORDER BY collected_at DESC            │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 2. Reconstruct snapshot (AS OF TIME)    │
│    - Start from current state           │
│    - Rewind changes backwards           │
│    - Apply 'disappeared' in reverse     │
│    - Apply 'appeared' in reverse        │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│ 3. Return reconstructed snapshot        │
│    { list: [...], snapshotTime: '...' } │
└─────────────────────────────────────────┘
```

---

## Query Optimization Strategies

### 1. Partition Pruning

**Scenario:** Query recent MAC history (last 7 days)

**Without Partitioning:**
```sql
-- Scans entire table (millions of rows)
SELECT * FROM sys_device_mac_history
WHERE collected_at >= NOW() - INTERVAL '7 days';
-- Query time: ~5-10 seconds (table has 10M rows)
```

**With Monthly Partitioning:**
```sql
-- PostgreSQL planner only scans current month's partition
SELECT * FROM sys_device_mac_history
WHERE collected_at >= NOW() - INTERVAL '7 days';
-- Query time: ~50-100ms (only scans May 2026 partition)
```

**Implementation:**
```sql
-- Trigger to auto-create partitions (simplified)
CREATE OR REPLACE FUNCTION create_mac_history_partition()
RETURNS void AS $$
DECLARE
    partition_date TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    partition_date := to_char(NOW() + INTERVAL '1 month', 'YYYY_MM');
    start_date := to_char(NOW() + INTERVAL '1 month', 'YYYY-MM-01');
    end_date := to_char(NOW() + INTERVAL '2 months', 'YYYY-MM-01');
    
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS sys_device_mac_history_%s 
         PARTITION OF sys_device_mac_history 
         FOR VALUES FROM (%L) TO (%L)',
        partition_date, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Schedule via pg_cron or external scheduler
SELECT cron.schedule('create-mac-partition', '0 0 1 * *', 'SELECT create_mac_history_partition()');
```

---

### 2. BRIN Index Optimization

**Scenario:** Time-series data with 10M rows

**B-tree Index:**
```sql
CREATE INDEX idx_mac_history_collected_at_btree 
  ON sys_device_mac_history(collected_at);

-- Index size: ~400MB (10M rows)
-- Query time: ~200ms for range scan
```

**BRIN Index (Recommended):**
```sql
CREATE INDEX idx_mac_history_collected_at_brin 
  ON sys_device_mac_history USING BRIN (collected_at);

-- Index size: ~4MB (100x smaller!)
-- Query time: ~150ms for range scan (slightly slower but space-efficient)
-- Best for: append-only time-series data
```

**Why BRIN:**
- MAC历史数据是**追加写入**（collected_at单调递增）
- BRIN按**块范围**索引，不是每行一个索引条目
- 适合磁盘空间有限但查询性能可接受的场景

---

### 3. Materialized Views for Common Queries

**Scenario:** Dashboard显示"最近7天MAC变更统计"

**Without Materialized View:**
```sql
-- Slow (aggregates 10M rows every query)
SELECT 
    device_id,
    COUNT(*) FILTER (WHERE change_type = 'appeared') as appeared_count,
    COUNT(*) FILTER (WHERE change_type = 'disappeared') as disappeared_count,
    COUNT(*) FILTER (WHERE change_type = 'moved') as moved_count
FROM sys_device_mac_history
WHERE collected_at >= NOW() - INTERVAL '7 days'
GROUP BY device_id;
-- Query time: ~2-3 seconds
```

**With Materialized View:**
```sql
CREATE MATERIALIZED VIEW mac_change_stats_7d AS
SELECT 
    device_id,
    COUNT(*) FILTER (WHERE change_type = 'appeared') as appeared_count,
    COUNT(*) FILTER (WHERE change_type = 'disappeared') as disappeared_count,
    COUNT(*) FILTER (WHERE change_type = 'moved') as moved_count,
    MAX(collected_at) as last_updated
FROM sys_device_mac_history
WHERE collected_at >= NOW() - INTERVAL '7 days'
GROUP BY device_id;

CREATE UNIQUE INDEX ON mac_change_stats_7d(device_id);

-- Refresh every hour
REFRESH MATERIALIZED VIEW CONCURRENTLY mac_change_stats_7d;

-- Query time: ~5-10ms (pre-computed!)
```

**Refresh Strategy:**
- Use PostgreSQL's `pg_cron` extension
- Schedule: `REFRESH MATERIALIZED VIEW CONCURRENTLY mac_change_stats_7d` every 1 hour
- `CONCURRENTLY` allows reads during refresh

---

### 4. Covering Indexes for Filter Queries

**Scenario:** Query "所有设备上移动频率最高的10个MAC"

**Without Covering Index:**
```sql
-- Slow (requires heap access to get change_type)
SELECT mac_address, COUNT(*) as hop_count
FROM sys_device_mac_history
WHERE change_type = 'moved'
GROUP BY mac_address
ORDER BY hop_count DESC
LIMIT 10;
-- Query time: ~1-2 seconds
```

**With Covering Index:**
```sql
CREATE INDEX idx_mac_history_covering 
  ON sys_device_mac_history(change_type, mac_address, collected_at);

-- Query time: ~50-100ms (index-only scan)
```

---

## Build Order and Dependencies

### Phase 1: Database Schema (1 week)
**Goal:** Create time-series infrastructure

**Tasks:**
1. Design schema (`sys_device_mac_history`)
2. Write migration script (`12X_create_mac_history_table.sql`)
3. Implement partition creation trigger
4. Create BRIN and B-tree indexes
5. Test partition pruning performance

**Dependencies:** None

**Output:**
- Migration file
- Performance benchmark (before/after)

---

### Phase 2: History Service Layer (1-2 weeks)
**Goal:** Implement change detection and recording

**Tasks:**
1. Create `MACHistoryService` interface
2. Implement `calculateMACChanges()`
3. Implement `RecordChanges()`
4. Write unit tests (mock current/new snapshots)
5. Integration test (run collection, verify history records)

**Dependencies:** Phase 1 (table exists)

**Output:**
- `mac_history_service.go`
- Unit tests (>80% coverage)
- Integration test fixture

---

### Phase 3: Modify MAC Collection Service (1 week)
**Goal:** Integrate history recording into collection flow

**Tasks:**
1. Inject `MACHistoryService` into `MACCollectionService`
2. Modify `collectDeviceMAC()` to record changes
3. Add config toggle (`network.mac.history.enabled`)
4. Test with history disabled (backward compatibility)
5. Test with history enabled (verify records)

**Dependencies:** Phase 2 (history service ready)

**Output:**
- Modified `mac_collection_service.go`
- Config update (`sys_config` table)
- Integration test

---

### Phase 4: API Layer (1 week)
**Goal:** Expose history query endpoints

**Tasks:**
1. Create `MACHistoryHandler`
2. Implement 6 endpoints (query, trajectory, port, anomalies, stats, export)
3. Add Swagger documentation
4. Write integration tests
5. Performance test (query 1M history records)

**Dependencies:** Phase 3 (history data exists)

**Output:**
- `mac_history_handler.go`
- `mac_history_router.go`
- API tests
- Swagger docs

---

### Phase 5: Frontend Integration (1-2 weeks)
**Goal:** Build UI for history exploration

**Tasks:**
1. Create history query page (时态查询)
2. Create MAC trajectory visualization (时间线图表)
3. Create anomaly dashboard (异常事件列表)
4. Add export functionality (Excel/CSV)
5. Performance optimization (virtual scrolling for large lists)

**Dependencies:** Phase 4 (API endpoints ready)

**Output:**
- `ruoyi-react-frontend/src/pages/network/mac-history/`
- Components: `HistoryQuery`, `MACTrajectory`, `AnomalyDashboard`
- API client updates (`lib/api.ts`)

---

### Phase 6: Performance Optimization (1 week)
**Goal:** Optimize for scale (10M+ history records)

**Tasks:**
1. Implement materialized views for dashboard stats
2. Add query result caching (Redis)
3. Optimize slow queries (EXPLAIN ANALYZE + fix)
4. Set up partition archival (drop old partitions)
5. Load test (simultaneous 100 users)

**Dependencies:** Phase 4 (queries implemented)

**Output:**
- Materialized view definitions
- Cache integration code
- Partition archival script
- Performance report

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Store Every Collection Snapshot

**Bad Idea:**
```go
// Every collection run inserts ALL MAC addresses
for _, mac := range newMACs {
    history := MACHistory{
        MACAddress: mac.MACAddress,
        InterfaceName: mac.InterfaceName,
        CollectedAt: time.Now(),
    }
    db.Create(&history)
}
```

**Why Bad:**
- If device has 1000 MAC addresses, every collection inserts 1000 rows
- Most collections have <1% changes (wastes 99% storage)
- 10 collections/day × 1000 MACs × 30 days = 300K rows (99% redundant)

**Correct Approach:** Only record changes (增量快照)

---

### ❌ Anti-Pattern 2: No Partitioning

**Bad Idea:**
```sql
-- Single table for all time
CREATE TABLE sys_device_mac_history (...);
```

**Why Bad:**
- Query performance degrades after 10M rows
- Cannot efficiently drop old data (DELETE is slow)
- VACUUM takes hours on large tables

**Correct Approach:** Partition by `collected_at` (monthly)

---

### ❌ Anti-Pattern 3: Synchronous History Recording

**Bad Idea:**
```go
// Blocking collection until history written
if err := historyService.RecordChanges(ctx, deviceID, changes, time.Now()); err != nil {
    return err  // Fails entire collection!
}
```

**Why Bad:**
- History recording failure blocks MAC collection
- Collection latency increases (DB writes are slow)
- User experience degrades

**Correct Approach:** Async recording (background goroutine)

```go
go func() {
    if err := historyService.RecordChanges(context.Background(), deviceID, changes, time.Now()); err != nil {
        applogger.Warnf("History recording failed: %v", err)
        // Don't fail collection
    }
}()
```

---

### ❌ Anti-Pattern 4: Query Without Time Bounds

**Bad Idea:**
```typescript
// Frontend requests ALL history
const history = await api.get('/network/mac/history/query', {
    macAddress: 'aa:bb:cc:dd:ee:ff'
    // Missing: timeRange!
});
```

**Why Bad:**
- Returns millions of rows
- Frontend/browser crashes
- Database CPU spikes

**Correct Approach:** Always require time bounds

```typescript
const history = await api.get('/network/mac/history/query', {
    macAddress: 'aa:bb:cc:dd:ee:ff',
    timeRange: {
        start: '2026-05-01T00:00:00Z',
        end: '2026-05-09T23:59:59Z'
    }
});
```

---

## Scalability Considerations

| Metric | At 1K Devices | At 10K Devices | At 100K Devices |
|--------|---------------|----------------|-----------------|
| **Collections/day** | ~1K | ~10K | ~100K |
| **Avg MACs/device** | 100 | 100 | 100 |
| **Changes/collection** | 1% (1 MAC) | 1% (1 MAC) | 1% (1 MAC) |
| **History records/day** | 1K | 10K | 100K |
| **History records/month** | 30K | 300K | 3M |
| **Storage/month** | ~10MB | ~100MB | ~1GB |
| **Query latency (7 days)** | <10ms | <50ms | <200ms |
| **Query latency (1 year)** | <50ms | <200ms | <1s |

**Optimizations at Scale:**

1. **At 10K Devices:**
   - Enable connection pooling
   - Increase shared_buffers to 4GB
   - Use async history recording

2. **At 100K Devices:**
   - Implement data archival (drop partitions > 1 year)
   - Use read replicas for query load
   - Cache frequent queries (Redis)

---

## Migration Strategy

### Backward Compatibility

**Requirement:** Existing MAC collection must continue working during migration

**Approach:**

1. **Phase 1 (Zero Downtime):** Create new table alongside existing
   - Run migration to create `sys_device_mac_history`
   - Keep `sys_device_mac_address` unchanged
   - Add config flag `network.mac.history.enabled = false` (default)

2. **Phase 2 (Gradual Rollout):** Enable history for subset of devices
   - Set `network.mac.history.enabled = true` for 10% of devices
   - Monitor performance (collection latency, DB load)
   - Fix any issues

3. **Phase 3 (Full Cutover):** Enable history for all devices
   - Set `network.mac.history.enabled = true` globally
   - Verify history records are created

4. **Phase 4 (Cleanup):** Remove old code paths (optional)
   - Remove `enableHistory` flag (after 1 month of stable operation)

---

### Data Migration (Optional)

**Scenario:** Backfill history for last 30 days from log files

**Approach:**
1. Parse collection logs (if available)
2. Reconstruct historical snapshots
3. Batch insert into `sys_device_mac_history`

**Note:** Not required for MVP (history starts from first collection after migration)

---

## Summary and Recommendations

### Architecture Decisions

| Decision | Rationale | Confidence |
|----------|-----------|------------|
| **Separate history table** | Clear separation of current vs historical data; enables partitioning | HIGH |
| **Incremental snapshot (changes only)** | Reduces storage by 99% compared to full snapshots | HIGH |
| **Monthly partitioning by collected_at** | Query performance + efficient archival | HIGH |
| **BRIN index on collected_at** | 100x smaller than B-tree; optimal for time-series | HIGH |
| **Async history recording** | Non-blocking collection; better UX | HIGH |
| **Materialized views for dashboard** | Pre-computed stats; 100x faster queries | MEDIUM |

---

### Build Priority

**Must-Have (MVP):**
1. Database schema + partitioning
2. History service (change detection + recording)
3. Modify MAC collection service
4. Basic query API (historical snapshot)

**Should-Have (v1.5.1):**
5. MAC trajectory visualization
6. Anomaly detection
7. Performance optimization (materialized views)

**Nice-to-Have (v1.6):**
8. Advanced analytics (MAC lifecycle, network topology changes)
9. Real-time history streaming (WebSocket)
10. ML-based anomaly prediction

---

### Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Collection latency increases** | High | Async history recording; performance monitoring |
| **History table grows too fast** | Medium | Configurable retention policy; partition archival |
| **Query performance degrades** | High | Partition pruning; BRIN indexes; materialized views |
| **Data corruption (bad change detection)** | Medium | Comprehensive unit tests; integration tests |
| **Backward compatibility break** | High | Feature flag; gradual rollout |

---

## Sources

- **PostgreSQL Documentation**: Partitioning, BRIN indexes, materialized views
- **Existing Codebase**: 
  - `internal/services/mac_collection_service.go` (current collection flow)
  - `internal/services/config_backup_service.go` (time-series pattern reference)
  - `internal/core/db/migrations/019_add_performance_indexes.sql` (indexing patterns)
  - `internal/core/db/migrations/020_drop_mac_unique_index.sql` (history preparation)
- **Phase 11 (MAC Filtering)**: LLDP integration, filter rules
- **CLAUDE.md**: Architecture conventions (Handler-Service pattern, UUID foreign keys)

**Next Steps:** Proceed to `.planning/research/FEATURES.md` to define functional requirements for MAC历史数据管理.

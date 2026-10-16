# Phase 12: 数据模型与采集集成 - Pattern Map

**Mapped:** 2026-05-09
**Files analyzed:** 6
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/device_mac_history.go` | model | CRUD | `internal/models/device_mac_address.go` | exact |
| `internal/services/mac_collection_service.go` | service | event-driven | `internal/services/mac_collection_service.go` (self-modify) | role-match |
| `internal/services/mac_history_partition.go` | service | CRUD | `internal/core/db/database.go` (AutoMigrate pattern) | role-match |
| `internal/services/mac_history_cleanup.go` | service | batch | `internal/scheduler/ad_sync_tasks.go` | role-match |
| `internal/scheduler/mac_history_tasks.go` | middleware | event-driven | `internal/scheduler/ad_sync_tasks.go` | exact |

## Pattern Assignments

### `internal/models/device_mac_history.go` (model, CRUD)

**Analog:** `internal/models/device_mac_address.go`

**Imports pattern** (lines 1-8):
```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
```

**Model structure pattern** (lines 19-29):
```go
// DeviceMACAddress 设备MAC地址模型
type DeviceMACAddress struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID      string    `gorm:"type:uuid;not null" json:"deviceId"`
	MACAddress    string    `gorm:"size:30;not null" json:"macAddress"`
	InterfaceName string    `gorm:"size:100;not null" json:"interfaceName"`
	VLANID        *int      `json:"vlanId,omitempty"`
	// ... additional fields
	CollectedAt   time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}
```

**UUID auto-generation pattern** (lines 36-42):
```go
// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (d *DeviceMACAddress) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	return nil
}
```

**TableName pattern** (lines 31-34):
```go
// TableName 设置表名
func (DeviceMACAddress) TableName() string {
	return "sys_device_mac_address"
}
```

---

### `internal/services/mac_collection_service.go` (service, event-driven - MODIFY)

**Analog:** `internal/services/mac_collection_service.go` (self-modify existing code)

**Transaction pattern with DELETE before INSERT** (lines 200-206, 235-243):
```go
// 1. 先删除该设备的旧MAC地址记录
if err := s.db.WithContext(ctx).
	Where("device_id = ?", device.ID).
	Delete(&models.DeviceMACAddress{}).Error; err != nil {
	result.ErrorMessage = fmt.Sprintf("删除旧MAC地址失败: %v", err)
	return result
}

// ... build new records ...

// 3. 批量插入新的MAC地址记录
if len(macRecords) > 0 {
	if err := s.db.WithContext(ctx).
		Create(&macRecords).Error; err != nil {
		result.ErrorMessage = fmt.Sprintf("批量插入MAC地址失败: %v", err)
		return result
	}
	successCount = len(macRecords)
}
```

**Error handling pattern** (lines 111-115):
```go
defer func() {
	if r := recover(); r != nil {
		applogger.Infof("[MAC采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
	}
}()
```

**Context propagation pattern** (line 201):
```go
s.db.WithContext(ctx)
```

---

### `internal/services/mac_history_partition.go` (service, CRUD)

**Analog:** `internal/core/db/database.go` (AutoMigrate pattern)

**Database check pattern** (lines 309-318):
```go
// 检查迁移是否已经执行过（通过检查 protocol_type 列是否存在）
var columnExists bool
err := d.DB.Raw(`
	SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'sys_auth_credential' AND column_name = 'protocol_type'
	)
`).Scan(&columnExists).Error
if err != nil {
	return fmt.Errorf("检查迁移状态失败: %w", err)
}
```

**Transaction with DDL pattern** (lines 349-410):
```go
// 使用事务
if err := d.DB.Transaction(func(tx *gorm.DB) error {
	// 步骤1：添加新列
	if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS protocol_type VARCHAR(10)").Error; err != nil {
		return fmt.Errorf("添加 protocol_type 列失败: %w", err)
	}

	// ... additional steps ...

	return nil
}); err != nil {
	return fmt.Errorf("凭证模型迁移失败: %w", err)
}
```

**PostgreSQL DDL pattern** (lines 351-353):
```go
if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS protocol_type VARCHAR(10)").Error; err != nil {
	return fmt.Errorf("添加 protocol_type 列失败: %w", err)
}
```

---

### `internal/services/mac_history_cleanup.go` (service, batch)

**Analog:** `internal/scheduler/ad_sync_tasks.go`

**Config reading pattern** (from cache_config_service.go):
```go
// 从数据库读取配置
var config models.Config
err := s.db.Where("config_key = ?", macConcurrentConfigKey).First(&config).Error
if err == nil && config.ConfigValue != "" {
	if concurrent, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil && concurrent > 0 {
		s.maxConcurrent = concurrent
		applogger.Infof("[MAC采集] 从数据库读取并发数配置: %d", s.maxConcurrent)
	}
}
```

**Partition discovery pattern** (lines 113-120 from ad_sync_tasks.go):
```go
var configs []models.ADConfig
err := s.db.Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).Find(&configs).Error
if err != nil {
	applogger.Errorf("查询AD配置失败: %v", err)
	return
}
```

**Batch operation pattern** (lines 119-133 from ad_sync_tasks.go):
```go
for _, config := range configs {
	shouldSync := false
	// ... check conditions ...

	if shouldSync {
		// 使用信号量控制并发数，异步执行同步
		go func(configID string, configName string) {
			syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			// Acquire semaphore and execute
			// ...
		}(config.ID, config.ConfigName)
	}
}
```

---

### `internal/scheduler/mac_history_tasks.go` (middleware, event-driven)

**Analog:** `internal/scheduler/ad_sync_tasks.go`

**Task registration pattern** (lines 46-51):
```go
// RegisterADSyncTasks 注册AD域同步定时任务
func RegisterADSyncTasks(scheduler *Scheduler) {
	// AD数据自动同步任务 - 每小时检查一次需要同步的配置
	scheduler.RegisterTask("ad_data_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeADDataSyncTask(ctx, params)
	})
}
```

**Task execution pattern** (lines 195-215):
```go
// executeADDataSyncTask 执行AD数据同步任务（供Scheduler调用）
func executeADDataSyncTask(ctx context.Context, params map[string]interface{}) error {
	if globalADSyncScheduler == nil {
		return fmt.Errorf("AD同步调度器未初始化")
	}

	configID, ok := params["configId"].(string)
	if !ok || configID == "" {
		return fmt.Errorf("配置ID参数无效")
	}

	adService := addomain.NewADDomainService(globalADSyncScheduler.db)

	_, err := adService.Sync.SyncDataByID(ctx, configID, syncType)
	return err
}
```

**Global service getter pattern** (lines 1024-1027):
```go
// SetDeviceMonitorService 设置设备监控服务（线程安全）
func SetDeviceMonitorService(service DeviceMonitorService) {
	GlobalDeviceMonitorServiceMu.Lock()
	defer GlobalDeviceMonitorServiceMu.Unlock()
	GlobalDeviceMonitorService = service
}
```

---

## Shared Patterns

### Database Transaction Pattern
**Source:** `internal/core/db/database.go` lines 349-410
**Apply to:** All services performing multi-step database operations
```go
// 使用事务
if err := d.DB.Transaction(func(tx *gorm.DB) error {
	// Step 1: Query old state
	if err := tx.Where(...).Find(&oldData).Error; err != nil {
		return err
	}

	// Step 2: Write new history
	if err := tx.Create(&history).Error; err != nil {
		return err
	}

	// Step 3: Delete old data
	if err := tx.Where(...).Delete(&oldModel).Error; err != nil {
		return err
	}

	// Step 4: Insert new data
	if err := tx.Create(&newData).Error; err != nil {
		return err
	}

	return nil // Commit transaction
}); err != nil {
	return fmt.Errorf("operation failed: %w", err)
}
```

### Error Handling with Context
**Source:** `internal/services/mac_collection_service.go` lines 111-115
**Apply to:** All services
```go
defer func() {
	if r := recover(); r != nil {
		applogger.Infof("[Service] panic recovered: %v", r)
	}
}()
```

### Context Propagation
**Source:** `internal/services/mac_collection_service.go` line 201
**Apply to:** All database operations
```go
s.db.WithContext(ctx).Where(...).Find(...)
```

### Configuration Reading Pattern
**Source:** `internal/services/cache_config_service.go` (from grep results)
**Apply to:** Services reading runtime configuration
```go
var config models.Config
err := s.db.Where("config_key = ?", configKey).First(&config).Error
if err == nil && config.ConfigValue != "" {
	// Parse and use config value
	if value, parseErr := strconv.Atoi(config.ConfigValue); parseErr == nil {
		// Use parsed value
	}
}
```

### Scheduler Task Registration Pattern
**Source:** `internal/scheduler/cron.go` lines 198-203
**Apply to:** New scheduled tasks
```go
func (s *Scheduler) RegisterTask(taskType string, handler func(ctx context.Context, params map[string]interface{}) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskRegistry[taskType] = handler
}
```

### Panic Recovery in Goroutines
**Source:** `internal/scheduler/ad_sync_tasks.go` lines 150-166
**Apply to:** Concurrent task execution
```go
go func(configID string, configName string) {
	syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := s.sem.Acquire(syncCtx, 1); err != nil {
		if syncCtx.Err() == context.DeadlineExceeded {
			applogger.Warnf("Task timeout: %s", configName)
		} else {
			applogger.Errorf("Task start failed: %s - %v", configName, err)
		}
		return
	}
	defer s.sem.Release(1)

	// Execute task
	s.syncADConfig(syncCtx, configID)
}(config.ID, config.ConfigName)
```

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (All files have close analogs) | - | - | All patterns found in existing codebase |

---

## Metadata

**Analog search scope:**
- `internal/models/` - Model definitions
- `internal/services/mac_collection_service.go` - MAC collection service
- `internal/core/db/database.go` - Database migration patterns
- `internal/scheduler/ad_sync_tasks.go` - Scheduler task patterns
- `internal/scheduler/cron.go` - Scheduler registration patterns
- `internal/services/cache_config_service.go` - Configuration reading patterns

**Files scanned:** 6 primary source files
**Pattern extraction date:** 2026-05-09

---

## Implementation Notes

### Critical Integration Points

1. **MAC Collection Service Modification** (lines 195-246 in mac_collection_service.go):
   - Insert change detection BEFORE line 201 (DELETE operation)
   - Query old MAC addresses first
   - Compare with new MAC addresses to detect changes
   - Write history records
   - Continue with existing DELETE → INSERT flow

2. **Partition Management**:
   - Use PostgreSQL native partitioning (from database.go lines 349-410 pattern)
   - Create partitions with `CREATE TABLE IF NOT EXISTS ... PARTITION OF ...`
   - Apply BRIN index for time-series optimization

3. **Scheduler Integration**:
   - Register cleanup task using `RegisterTask()` pattern
   - Use global service getter/setter pattern for dependency injection
   - Execute cleanup with concurrency control (semaphore pattern)

4. **Configuration Reading**:
   - Use `sys_config` table with `config_key` = "network.mac.history.retention_days"
   - Read at startup and cache in service struct
   - Support runtime reload via `ReloadConfig()` method

---

## PostgreSQL-Specific Patterns

### Partition Table Creation
**Source:** RESEARCH.md lines 310-336 (PostgreSQL documentation)
```sql
-- 创建主表（声明为 PARTITIONED BY）
CREATE TABLE sys_device_mac_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    -- ... other fields ...
    first_seen TIMESTAMP NOT NULL
) PARTITION BY RANGE (first_seen);

-- 创建月度分区
CREATE TABLE sys_device_mac_history_2025_01
PARTITION OF sys_device_mac_history
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### BRIN Index Creation
**Source:** RESEARCH.md lines 338-349
```sql
-- 为时间序列字段创建 BRIN 索引（极小的索引尺寸）
CREATE INDEX CONCURRENTLY idx_mac_history_first_seen_brin
ON sys_device_mac_history USING BRIN (first_seen)
WITH (pages_per_range = 128);
```

### Partition Dropping for Cleanup
**Source:** CONTEXT.md D-08
```sql
-- 直接删除整个过期分区，瞬间释放空间
DROP TABLE IF EXISTS sys_device_mac_history_2024_12;
```

---

## Testing Patterns

### Integration Test Pattern
**Source:** RESEARCH.md lines 473-493
```go
func TestMACChangeDetection(t *testing.T) {
	// Setup test database
	db := setupTestDB()
	defer cleanupTestDB(db)

	// Create test device
	device := createTestDevice(db)

	// Collect MAC addresses first time
	ctx := context.Background()
	service := NewMACCollectionService(db, executor, nil, nil)
	result1 := service.CollectDevice(ctx, device.ID)

	// Modify MAC addresses
	// ...

	// Collect again
	result2 := service.CollectDevice(ctx, device.ID)

	// Verify history records
	var history []models.DeviceMACHistory
	db.Where("device_id = ?", device.ID).Find(&history)

	// Assertions
	assert.Greater(t, len(history), 0, "Should have history records")
}
```

---

## Phase-Specific Considerations

### Change Detection Logic Placement
**CRITICAL:** Must execute DELETE operation at line 201, not before.
**Reason:** Need old state for comparison. Insert change detection between lines 198-201.

### Transaction Boundaries
**CRITICAL:** Entire flow (query → compare → write history → delete → insert) in ONE transaction.
**Pattern:** Use `db.Transaction()` wrapper around entire `collectDeviceMAC` method.

### Non-Blocking History Writes
**CRITICAL:** History write failure MUST NOT block MAC collection.
**Pattern:** Log error but continue with DELETE → INSERT flow (CONTEXT.md D-07).

### Partition Auto-Creation
**TIMING:** Execute on application startup via `core.Init()` or `main.go`.
**Pattern:** Check next 1-2 months, create missing partitions (RESEARCH.md line 220).

### Cleanup Task Scheduling
**CRITICAL:** Schedule for low-traffic hours (2 AM daily).
**Pattern:** Use `0 0 2 * * ?` Cron expression (CONTEXT.md D-09).

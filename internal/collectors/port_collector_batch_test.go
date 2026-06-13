//go:build !skip_db_tests
// +build !skip_db_tests

package collectors

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Pure-Go SQLite driver
	_ "modernc.org/sqlite"
)

// setupPortCollectorTestDB creates an in-memory SQLite DB with the
// sys_device_port_status table.
func setupPortCollectorTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default,
	})
	assert.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE sys_device_port_status (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			interface_name TEXT NOT NULL,
			admin_status TEXT,
			oper_status TEXT,
			description TEXT,
			vlan INTEGER,
			duplex TEXT,
			speed TEXT,
			port_type TEXT,
			dot1x_enabled INTEGER DEFAULT 0,
			dot1x_port_status TEXT,
			port_security_enabled INTEGER DEFAULT 0,
			port_security_mode TEXT,
			max_mac_count INTEGER,
			current_mac_count INTEGER,
			collected_at DATETIME NOT NULL,
			created_at DATETIME
		);
	`).Error
	assert.NoError(t, err)

	return db
}

// TestPortCollector_SaveToDatabase_Batches100PerCall verifies P1-C4 fix:
// port collector should use GORM's CreateInBatches with batch=100, NOT
// N separate Create calls. We verify by counting INSERT statements via
// a hook on the Create callback.
//
// Without batching, 250 entries = 250 round-trips. With batch=100,
// 250 entries = ceil(250/100) = 3 INSERT round-trips.
//
// We assert the count is <= ceil(N/100) — i.e., batched, not per-row.
func TestPortCollector_SaveToDatabase_Batches100PerCall(t *testing.T) {
	db := setupPortCollectorTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	// Install an INSERT counter on the Create callback for the
	// sys_device_port_status table. GORM's CreateInBatches issues
	// ONE Create call total, which internally executes ONE multi-row
	// INSERT per batch chunk. So we should observe 3 Create calls
	// (or fewer) for 250 entries.
	var createCount int32
	db.Callback().Create().Before("gorm:create").Register("count-create-hook", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "sys_device_port_status" {
			atomic.AddInt32(&createCount, 1)
		}
	})

	collector := &PortCollector{db: db}

	// Build 250 entries (sample for an access-layer switch with 48 1G + mGig uplinks)
	const N = 250
	entries := make([]PortStatusEntry, 0, N)
	for i := 0; i < N; i++ {
		entries = append(entries, PortStatusEntry{
			InterfaceName: "GigabitEthernet0/0/" + itoaForTest(i),
			AdminStatus:   "up",
			OperStatus:    "up",
		})
	}

	err := collector.SaveToDatabase("device-batch-1", entries)
	assert.NoError(t, err)

	// Assertion: GORM CreateInBatches(_, 100) issues ceil(N/100)=3 Create
	// callbacks (one per batch of 100, 100, 50). Pre-fix loop would issue N=250.
	createCalls := atomic.LoadInt32(&createCount)
	assert.LessOrEqual(t, createCalls, int32(3),
		"expected ≤3 Create callbacks for batched insert of 250 entries, got %d (likely per-row loop regression)", createCalls)

	// Verify all rows were actually saved
	var count int64
	db.Table("sys_device_port_status").Where("device_id = ?", "device-batch-1").Count(&count)
	assert.Equal(t, int64(N), count, "all 250 entries should be persisted")
}

// TestPortCollector_SaveToDatabase_EmptyEntries is a sanity check —
// empty input should be a no-op without errors.
func TestPortCollector_SaveToDatabase_EmptyEntries(t *testing.T) {
	db := setupPortCollectorTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	var createCount int32
	db.Callback().Create().Before("gorm:create").Register("count-create-hook", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "sys_device_port_status" {
			atomic.AddInt32(&createCount, 1)
		}
	})

	collector := &PortCollector{db: db}
	err := collector.SaveToDatabase("device-empty", nil)
	assert.NoError(t, err)
	err = collector.SaveToDatabase("device-empty", []PortStatusEntry{})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&createCount),
		"empty input should not issue any Create callbacks")
}

// TestPortCollector_SaveToDatabase_SmallBatchIsSingleRoundTrip
// verifies that a sub-batch (e.g., 50 entries < 100) still uses ONE
// Create callback (i.e., CreateInBatches coalesces into 1 round-trip
// when below the batch size).
func TestPortCollector_SaveToDatabase_SmallBatchIsSingleRoundTrip(t *testing.T) {
	db := setupPortCollectorTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	var createCount int32
	db.Callback().Create().Before("gorm:create").Register("count-create-hook", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "sys_device_port_status" {
			atomic.AddInt32(&createCount, 1)
		}
	})

	collector := &PortCollector{db: db}
	const N = 50
	entries := make([]PortStatusEntry, 0, N)
	for i := 0; i < N; i++ {
		entries = append(entries, PortStatusEntry{
			InterfaceName: "GigabitEthernet0/0/" + itoaForTest(i),
			AdminStatus:   "up",
			OperStatus:    "up",
		})
	}

	err := collector.SaveToDatabase("device-small", entries)
	assert.NoError(t, err)

	// 50 entries < batch_size=100, so should be exactly 1 Create callback
	assert.Equal(t, int32(1), atomic.LoadInt32(&createCount),
		"50 entries should issue exactly 1 Create callback (single batch)")
}

// itoaForTest avoids importing strconv just for the test.
func itoaForTest(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := [20]byte{}
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
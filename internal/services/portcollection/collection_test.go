//go:build !skip_db_tests
// +build !skip_db_tests

package portcollection

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"

	// Pure-Go SQLite driver
	_ "modernc.org/sqlite"
)

// setupOwnershipTestDB creates an in-memory SQLite DB with the
// sys_device_port_status table, mirroring the production schema enough
// to exercise the ownership-clash GORM query path.
func setupOwnershipTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
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
	`).Error)

	return db
}

// insertPort helper inserts one row.
func insertPort(t *testing.T, db *gorm.DB, deviceID, iface string) {
	t.Helper()
	port := &models.DevicePortStatus{
		DeviceID:      deviceID,
		InterfaceName: iface,
		CollectedAt:   time.Now(),
	}
	require.NoError(t, db.Create(port).Error)
}

// TestOwnershipClashCheck_FindIntoAnonStruct locks down the F-45-R5-onwards
// port-collection ownership clash query path.
//
// Bug history (2026-06-30 port-collection window 10:00:06->10:00:23):
// the function issued the query:
//
//	db.Model(&models.DevicePortStatus{}).
//	   Select("interface_name, device_id").
//	   Where("interface_name IN ? AND device_id <> ?", ifaceNames, device.ID).
//	   Find(&conflictIfaces)              // <-- BUG: conflictIfaces := make(map[string]string)
//
// `Find` into a `map[string]string` is not a supported GORM destination.
// Internally GORM took the Rows() path and called rows.Scan() without
// iterating rows.Next() first, producing the continuous runtime errors:
//
//	ERRO [GORM错误] SELECT interface_name, device_id FROM "sys_device_port_status"
//	     WHERE interface_name IN (...) AND device_id <> '...'
//	     | 错误: sql: Scan called without calling Next
//	WARN [PortCollection] <device>: check ownership clash failed: sql: Scan called without calling Next
//
// The fix scans into an anonymous-struct slice, then builds the map
// locally. This test asserts that the fix shape:
//  1. compiles and runs without error against a real GORM backend,
//  2. returns the conflicting interface names + their owning device IDs,
//  3. correctly ignores rows belonging to the queried device (the
//     `device_id <> ?` half of the WHERE),
//  4. populates the lookup map used to skip conflicting ports.
func TestOwnershipClashCheck_FindIntoAnonStruct(t *testing.T) {
	db := setupOwnershipTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()
	ctx := context.Background()

	// The device being collected in this run
	ownDevice := uuid.NewString()
	// Other devices that "own" ports with the same interface names
	otherDeviceA := uuid.NewString()
	otherDeviceB := uuid.NewString()

	// 1) a port belonging to our device (should be filtered by `device_id <> ?`)
	insertPort(t, db, ownDevice, "GE0/0/1")
	// 2) ports owned by other devices (these are the conflicts)
	insertPort(t, db, otherDeviceA, "GE0/0/2")
	insertPort(t, db, otherDeviceA, "GE0/0/3")
	insertPort(t, db, otherDeviceB, "GE0/0/4")
	insertPort(t, db, otherDeviceB, "XGE1/0/1")

	// Names we are about to upsert for ownDevice.
	ifaceNames := []string{
		"GE0/0/1",  // exists but belongs to ownDevice — NOT a conflict
		"GE0/0/2",  // belongs to otherDeviceA — conflict
		"GE0/0/3",  // belongs to otherDeviceA — conflict
		"GE0/0/4",  // belongs to otherDeviceB — conflict
		"GE0/0/5",  // not in DB at all — not a conflict (empty row)
		"XGE1/0/1", // belongs to otherDeviceB — conflict
	}

	// ==== the FIX shape (verbatim from collection.go) ====
	conflictIfaces := make(map[string]string)
	var conflicts []struct {
		InterfaceName string
		DeviceID      string
	}
	err := db.WithContext(ctx).
		Model(&models.DevicePortStatus{}).
		Select("interface_name, device_id").
		Where("interface_name IN ? AND device_id <> ?", ifaceNames, ownDevice).
		Find(&conflicts).Error
	require.NoError(t, err, "Find() into anon-struct slice must not return 'Scan called without calling Next'")
	for _, c := range conflicts {
		conflictIfaces[c.InterfaceName] = c.DeviceID
	}

	// 4 conflicts expected (everything except GE0/0/5 and GE0/0/1)
	assert.Len(t, conflictIfaces, 4)
	assert.Equal(t, otherDeviceA, conflictIfaces["GE0/0/2"])
	assert.Equal(t, otherDeviceA, conflictIfaces["GE0/0/3"])
	assert.Equal(t, otherDeviceB, conflictIfaces["GE0/0/4"])
	assert.Equal(t, otherDeviceB, conflictIfaces["XGE1/0/1"])
	_, hasOwn := conflictIfaces["GE0/0/1"]
	assert.False(t, hasOwn, "ports belonging to ownDevice must not appear in conflict map (device_id <> ?)")
}

// TestOwnershipClashCheck_NoConflicts documents the empty-result path:
// a list of iface names with NO matching rows must return cleanly
// (no error, empty map, no row iteration surprises). The pre-fix bug
// would have produced "Scan called without calling Next" on this path
// for any device with a non-empty IN list, regardless of actual matches.
func TestOwnershipClashCheck_NoConflicts(t *testing.T) {
	db := setupOwnershipTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()
	ctx := context.Background()

	deviceID := uuid.NewString()
	ifaceNames := []string{
		"GE0/0/1", "GE0/0/2", "GE0/0/3",
		"XGE1/0/1", "XGE1/0/2", "XGE1/0/3",
		"FE0/0/1", "FE0/0/2", "FE0/0/3",
	}
	// (no inserts - empty table)

	conflictIfaces := make(map[string]string)
	var conflicts []struct {
		InterfaceName string
		DeviceID      string
	}
	err := db.WithContext(ctx).
		Model(&models.DevicePortStatus{}).
		Select("interface_name, device_id").
		Where("interface_name IN ? AND device_id <> ?", ifaceNames, deviceID).
		Find(&conflicts).Error
	require.NoError(t, err, "empty IN-list match must not trigger Scan-without-Next")
	for _, c := range conflicts {
		conflictIfaces[c.InterfaceName] = c.DeviceID
	}
	assert.Empty(t, conflictIfaces)
	assert.Empty(t, conflicts)
}

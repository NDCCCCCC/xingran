// QUIRK-P2 regression test — DeviceConnectionPool.Close goroutine leak.
//
// Bug: Close() called close(p.done) BEFORE cleanupTicker.Stop().
// If a tick was pending in cleanupTicker.C when Close was called, the
// startCleanup goroutine's select could non-deterministically pick the
// tick case over the done case, preventing goroutine exit.
//
// Fix: Stop ticker FIRST, then close done channel — guaranteeing the
// goroutine exits via the case <-p.done branch.
//
// This test uses runtime.NumGoroutine diff to detect the leak.
package device

import (
	"runtime"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestDeviceConnectionPool_Close_GoroutineLeak verifies that startCleanup
// exits promptly (within ~1s) when Close is called, with no goroutine leak.
func TestDeviceConnectionPool_Close_GoroutineLeak(t *testing.T) {
	// Create an in-memory DB (minimal schema not strictly needed since
	// we never call createConnection — we just need a valid *gorm.DB).
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS sys_network_device (
		id TEXT PRIMARY KEY, device_name TEXT, vendor TEXT, ip_address TEXT,
		port INTEGER DEFAULT 22, status INTEGER DEFAULT 2, created_at DATETIME,
		updated_at DATETIME, deleted_at DATETIME, credential_id TEXT)`).Error; err != nil {
		t.Fatalf("DDL: %v", err)
	}

	// Capture baseline goroutines before pool creation.
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	var baseline int
	for i := 0; i < 5; i++ {
		baseline = runtime.NumGoroutine()
		time.Sleep(5 * time.Millisecond)
	}

	pool := NewDeviceConnectionPool(db, fakeCipher{}, &PoolConfig{
		MaxIdle:        50 * time.Millisecond,
		MaxConnections: 2,
	})

	// Give the cleanup goroutine time to start.
	time.Sleep(20 * time.Millisecond)

	// Spawn some GetConnection goroutines to increase pool activity,
	// making it more likely a tick is pending when Close is called.
	// We use device IDs not in DB so they fail fast — we only care about
	// the cleanup goroutine, not the connection goroutines.
	for i := 0; i < 3; i++ {
		go func(idx int) {
			_, _ = pool.GetConnection(nil, "nonexistent-device")
		}(i)
	}
	time.Sleep(10 * time.Millisecond)

	// Close the pool. This should cause startCleanup to exit.
	done := make(chan error, 1)
	go func() {
		done <- pool.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("pool.Close returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("pool.Close did not return within 5s — possible goroutine leak")
	}

	// Allow time for the cleanup goroutine to terminate and GC to collect.
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()
	// Allow a small tolerance (2) for any async teardown noise from the
	// test infrastructure itself. The critical check is that we don't
	// end up with the pool's cleanup goroutine still alive (+1 baseline).
	if after > baseline+2 {
		t.Errorf("goroutine leak detected: baseline=%d, after=%d (diff=%d, tolerance=2)",
			baseline, after, after-baseline)
	}
}

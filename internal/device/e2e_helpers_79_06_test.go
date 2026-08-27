// Phase 79-06 (TAIL-01) — coverage tests for the SeedConnectionForTesting
// helper appended to e2e_helpers.go under D-79-02.
//
// D-79-02: the ONLY sanctioned production-tree touch of Phase 79 — a test-infra
// ForTesting helper (INFRA-02 precedent), zero behavior change, guarded by the
// AST backstop in for_testing_guard_test.go (any production reference to a
// *ForTesting symbol fails the package test with a file:line report).
//
// What the helper unlocks: internal/services tests can already assemble a fully
// wired *DeviceExecutor via the public constructors
// (NewDeviceConnectionPool → NewDeviceTaskScheduler → NewDeviceExecutor), but
// every field of DeviceConnectionPool is unexported, so a FileTransport-backed
// connection could not be seeded into the pool from another package. This file
// proves the seed → GetConnection-reuse path end-to-end with the same public
// scrapligo FileTransport API the cross-package callers use (78-03手法, public
// API portion).
//
// Fixture discipline (78-03 S-2 + portwrite e2e precedent):
//   - FileTransport replays the fixture linearly; fixture exhaustion makes
//     FileTransport.Read block forever on select{}. Tests therefore never call
//     driver.Close() and every fixture carries 8 spare prompt lines.
//   - pool.Close() (t.Cleanup) may block on FileTransport close reads, so it is
//     bounded by a 3s watchdog: on timeout the goroutine is leaked on purpose
//     instead of hanging the whole test binary (QUIRK-P2 note).
//   - No t.Parallel — pool cleanup goroutine + fixture files are shared state.
package device

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// sparePrompt7906 is the number of trailing prompt lines appended to every
// fixture so that GetPrompt probes (pool.GetConnection's IsReady check) and
// close-time reads never exhaust the stream.
const sparePrompt7906 = 8

// prompt7906 is the huawei_vrp prompt literal used by the fixture stream.
const prompt7906 = "<dummy-host>"

// write7906Fixture writes a FileTransport replay fixture into a process temp
// dir (NOT t.TempDir) and returns its path. Shape mirrors
// internal/device/testdata/huawei_vrp_ops.fixture: open banner prompt →
// screen-length echo → post-open prompt → one (prompt, echo, output, prompt)
// block per command → spare prompts.
//
// Windows caveat: the FileTransport keeps the fixture *os.File open for the
// lifetime of the driver (tests never Close drivers — S-2 pitfall), so
// t.TempDir()'s RemoveAll would fail the test with "file in use". The best
// effort cleanup below silently no-ops on that lock instead.
func write7906Fixture(t *testing.T, cmds []string) string {
	t.Helper()

	var b strings.Builder
	b.WriteString(prompt7906 + "\n")             // open banner prompt
	b.WriteString("screen-length 0 temporary\n") // on-open screen-length echo
	b.WriteString(prompt7906 + "\n")             // post-open prompt
	for _, cmd := range cmds {
		b.WriteString(prompt7906 + "\n") // pre-command prompt (IsReady probes)
		b.WriteString(cmd + "\n")        // command echo
		b.WriteString("fixture-output-for-" + cmd + "\n")
		b.WriteString(prompt7906 + "\n") // terminating prompt
	}
	for i := 0; i < sparePrompt7906; i++ {
		b.WriteString(prompt7906 + "\n")
	}

	dir, err := os.MkdirTemp("", "device7906-fixture")
	require.NoError(t, err, "MkdirTemp")
	t.Cleanup(func() {
		_ = os.RemoveAll(dir) // best effort — Windows may hold the file open
	})

	path := filepath.Join(dir, "device7906.fixture")
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644), "write fixture")
	return path
}

// newDriver7906 builds an Open()-ed scrapligo *network.Driver backed by a fresh
// FileTransport fixture. The driver is intentionally never Closed (S-2 pitfall:
// close-time reads on an exhausted fixture block forever).
func newDriver7906(t *testing.T, cmds []string) *network.Driver {
	t.Helper()

	p, err := platform.NewPlatform(
		"huawei_vrp",
		"dummy-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(write7906Fixture(t, cmds)),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
	)
	require.NoError(t, err, "NewPlatform")
	d, err := p.GetNetworkDriver()
	require.NoError(t, err, "GetNetworkDriver")
	require.NoError(t, d.Open(), "driver.Open")
	return d
}

// newPool7906 builds a DeviceConnectionPool over a temp-file sqlite DB and
// binds its cleanup goroutine to a bounded t.Cleanup shutdown.
func newPool7906(t *testing.T) (*DeviceConnectionPool, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "e2e7906.db")), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	cfg := &PoolConfig{MaxIdle: time.Hour, MaxConnections: 8}
	pool := NewDeviceConnectionPool(db, nil, cfg)
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() { _ = pool.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Logf("pool.Close did not return within 3s (FileTransport close reads) — goroutine leaked intentionally")
		}
	})
	return pool, db
}

// TestSeedConnectionForTesting7906_SeedReuse seeds one FileTransport connection
// and proves the pool's GetConnection reuse path hands back the very same
// *PooledConnection, with the pool bookkeeping (GetStats) reflecting it.
func TestSeedConnectionForTesting7906_SeedReuse(t *testing.T) {
	pool, _ := newPool7906(t)
	ctx := context.Background()

	pc := NewPooledConnectionForTesting(newDriver7906(t, nil))
	SeedConnectionForTesting(pool, "device-7906-a", pc)
	// The factory starts refCount at 1 ("caller already holds it"); handing the
	// connection to the pool returns that ref so pool.Close sees an idle conn
	// instead of burning its 2s force-close timeout.
	pc.ReleaseRef()

	require.Equal(t, 1, pool.GetStats()["total_connections"], "seed must be visible in pool stats")

	// Lock consistency contract: the seeded connection must carry the SAME
	// per-device mutex GetConnection uses internally (78-03 seedPool78 rule).
	require.Same(t, pool.getDeviceLock("device-7906-a"), pc.mu, "conn.mu must be the pool device lock")
	require.Equal(t, "device-7906-a", pc.deviceID)
	require.Same(t, pool, pc.pool)

	got, err := pool.GetConnection(ctx, "device-7906-a")
	require.NoError(t, err, "GetConnection reuse path")
	require.Same(t, pc, got, "GetConnection must return the seeded connection pointer")
	require.NotNil(t, got.GetWrapper())
	got.ReleaseRef()
}

// TestSeedConnectionForTesting7906_OverwriteAndGuards seeds twice for the same
// deviceID (overwrite) and exercises every no-op guard branch.
func TestSeedConnectionForTesting7906_OverwriteAndGuards(t *testing.T) {
	pool, _ := newPool7906(t)
	ctx := context.Background()

	first := NewPooledConnectionForTesting(newDriver7906(t, nil))
	second := NewPooledConnectionForTesting(newDriver7906(t, nil))
	SeedConnectionForTesting(pool, "device-7906-b", first)
	SeedConnectionForTesting(pool, "device-7906-b", second)
	first.ReleaseRef() // overwritten entry is caller-owned; drop its factory ref
	second.ReleaseRef()

	got, err := pool.GetConnection(ctx, "device-7906-b")
	require.NoError(t, err)
	require.Same(t, second, got, "second seed must overwrite the first entry")
	got.ReleaseRef()

	// Guard branches: nil pool / nil conn / empty deviceID — no panic, no write.
	require.NotPanics(t, func() {
		SeedConnectionForTesting(nil, "device-7906-x", second)
		SeedConnectionForTesting(pool, "device-7906-y", nil)
		SeedConnectionForTesting(pool, "", second)
	}, "nil/empty guards must be no-ops")

	stats := pool.GetStats()
	require.Equal(t, 1, stats["total_connections"], "guards must not add entries")
	require.Equal(t, 8, stats["max_connections"])
}

// TestSeedConnectionForTesting7906_MultiDevice seeds two devices and proves
// per-device isolation of the reuse path.
func TestSeedConnectionForTesting7906_MultiDevice(t *testing.T) {
	pool, _ := newPool7906(t)
	ctx := context.Background()

	pcA := NewPooledConnectionForTesting(newDriver7906(t, []string{"display version"}))
	pcB := NewPooledConnectionForTesting(newDriver7906(t, []string{"show version"}))
	SeedConnectionForTesting(pool, "device-7906-a", pcA)
	SeedConnectionForTesting(pool, "device-7906-b", pcB)
	pcA.ReleaseRef() // hand the factory ref back so both conns are pool-idle
	pcB.ReleaseRef()

	gotA, err := pool.GetConnection(ctx, "device-7906-a")
	require.NoError(t, err)
	require.Same(t, pcA, gotA)
	gotA.ReleaseRef()

	gotB, err := pool.GetConnection(ctx, "device-7906-b")
	require.NoError(t, err)
	require.Same(t, pcB, gotB)
	gotB.ReleaseRef()

	require.Equal(t, 2, pool.GetStats()["total_connections"])
}

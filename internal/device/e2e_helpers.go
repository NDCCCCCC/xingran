package device

import (
	"sync"
	"time"

	"github.com/scrapli/scrapligo/driver/network"
)

// TEST-ONLY factory for constructing a *PooledConnection outside the normal
// DeviceConnectionPool → createConnection flow.
//
// This helper exists to support Phase 54 (W5) scrapligo FileTransport e2e tests
// (`internal/services/portwrite/port_write_e2e_test.go`), which need to inject
// a PooledConnection backed by a fixture-driven driver into the service's
// `portWriteExecutor.ExecuteCustom` callback. The production createConnection
// path hard-codes NewScrapliWrapper / NewScrapliWrapperWithPort over real SSH
// and never accepts a transport option, so the e2e layer must construct the
// connection struct directly.
//
// Production code MUST NOT call this function. The naming suffix
// `ForTesting` is the physical isolation contract — there is no build tag
// because A1 (PLAN.md) requires e2e tests to run on every `go test ./...`.
// Calling this in production silently skips connection pool bookkeeping,
// device-level locking, reachability checks, and credential decryption.
//
// Related decisions:
//   - A1 (54-01-PLAN.md): naming + comment isolation, no //go:build tag.
//   - RESEARCH Open Questions #1: the least-invasive way to make
//     PooledConnection's private fields addressable from the portwrite
//     test package without touching the production service signature.
func NewPooledConnectionForTesting(d *network.Driver) *PooledConnection {
	return &PooledConnection{
		wrapper:  newScrapliWrapperForTesting(d),
		refCount: 1,
		lastUsed: time.Now(),
		deviceID: "e2e-dummy-device",
		mu:       &sync.Mutex{},
		pool:     nil,
	}
}

// SeedConnectionForTesting places a pre-built *PooledConnection directly into
// the pool's connection map, bypassing the entire createConnection flow
// (device/credential DB lookups, password decryption, reachability check,
// SSH handshake via OpenContext, capacity accounting and TOCTOU connecting
// placeholder).
//
// TEST-ONLY: the naming suffix `ForTesting` is the physical isolation
// contract (same three-layer contract as NewPooledConnectionForTesting above —
// no build tag, AST backstop in for_testing_guard_test.go). Production code
// MUST NOT call this function: it silently skips pool bookkeeping, device-level
// reachability checks and credential decryption. Production references are
// reported by for_testing_guard_test.go with a file:line violation.
//
// Semantics:
//   - nil pool, nil conn or empty deviceID is a no-op (never panics).
//   - The connection's `mu` is aligned to the pool's per-device lock via
//     getDeviceLock so that Acquire/Release on the seeded connection serialize
//     exactly like a pooled one (78-03 seedPool78 lock-consistency rule).
//   - Seeding twice for the same deviceID overwrites the previous entry
//     (the overwritten *PooledConnection is NOT closed — caller owns it).
//   - The cleanup goroutine started by NewDeviceConnectionPool is unaffected;
//     callers must arrange pool shutdown themselves (t.Cleanup(pool.Close)).
//
// Related decisions:
//   - D-79-02 (79-RESEARCH DQ2 / 79-06-PLAN.md Task 1): the only sanctioned
//     production-tree touch of Phase 79 — test-infra class, zero behavior
//     change, INFRA-02 precedent.
//   - Cross-package reachability: all of DeviceConnectionPool/TaskScheduler/
//     DeviceExecutor fields are unexported, so internal/services tests can
//     assemble a fully-wired executor via the public constructors
//     (NewDeviceConnectionPool → NewDeviceTaskScheduler → NewDeviceExecutor)
//     but could not seed a FileTransport connection into the pool before this
//     helper existed.
func SeedConnectionForTesting(pool *DeviceConnectionPool, deviceID string, conn *PooledConnection) {
	if pool == nil || conn == nil || deviceID == "" {
		return
	}

	// Lock consistency: reuse (or create) the pool's per-device lock BEFORE
	// taking poolLock — getDeviceLock acquires poolLock internally and
	// sync.RWMutex is not reentrant.
	conn.mu = pool.getDeviceLock(deviceID)
	conn.deviceID = deviceID
	conn.pool = pool

	pool.poolLock.Lock()
	pool.connections[deviceID] = conn
	pool.poolLock.Unlock()
}

// newScrapliWrapperForTesting constructs a minimal *ScrapliWrapper suitable
// for FileTransport-based e2e tests.
//
// Critically, `state` is set to StateReady so that acquireOp() (scrapli_wrapper.go:245-260)
// accepts subsequent driver operations. The other fields (device, closing,
// initDone, closeOnce, opMu, stateMu, getPromptMu) are populated with zero
// or no-op values because the FileTransport path does not require real
// credential resolution, network reachability, or background initDone
// signalling — fixture replay is fully synchronous.
func newScrapliWrapperForTesting(d *network.Driver) *ScrapliWrapper {
	return &ScrapliWrapper{
		driver:   d,
		device:   nil,
		state:    StateReady,
		closing:  make(chan struct{}),
		initDone: make(chan struct{}),
	}
}

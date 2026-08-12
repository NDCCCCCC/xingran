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
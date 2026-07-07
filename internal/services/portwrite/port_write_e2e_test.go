// Phase 54 (W5) e2e tests — close the gap left by Phase 51's
// mockDeviceExecutor, which returned nil without ever invoking the service's
// fn closure. The fn closure runs wrapper.SendConfigs → Response[] →
// lastResp → parseConfigError; without it, parseConfigError was untested
// against real scrapligo output.
//
// These tests use scrapligo's official transport.NewFileTransport() to replay
// pre-recorded Huawei VRP byte streams against the real SendConfigs pipeline.
// They run unconditionally on every `go test ./...` — no //go:build tag
// (PLAN.md A1 lock — every test run must execute them).
package portwrite

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
)

// fileTransportExecutor is the production-DeviceExecutor stand-in used by
// every e2e test. Unlike mockDeviceExecutor, it actually runs the closure:
// it opens a fresh scrapligo driver on every call (FileTransport replay),
// wraps it in a *PooledConnection via NewPooledConnectionForTesting, and
// invokes fn(ctx, pc).
//
// per-call-fresh-driver is required because FileTransport consumes the
// fixture file linearly; reusing a driver across multiple ports in batch
// tests would replay partial bytes.
//
// We deliberately do NOT call d.Close() in this test harness. The platform's
// `network-on-close` operations (acquire-priv + channel.write 'quit' +
// channel.return) need to read more bytes from the FileTransport after the
// fixture has been fully consumed, which would block on FileTransport.Read
// `select{}` (Pitfall #1 in PLAN.md / RESEARCH.md). Skipping Close is safe
// here because the file-transport driver has no real socket to release —
// Go GC will reap the wrapper when the test function returns.
type fileTransportExecutor struct {
	fixtureProvider func(callIdx int) string
	counter         atomic.Uint64
}

func (e *fileTransportExecutor) nextIdx() int {
	return int(e.counter.Add(1) - 1)
}

// ExecuteCustom opens a fresh scrapligo FileTransport-backed driver per call
// and invokes fn. Returning the same fn as the production DeviceExecutor
// makes it possible for the e2e test to assert that SendConfigs /
// parseConfigError ran against real scrapligo output.
func (e *fileTransportExecutor) ExecuteCustom(
	ctx context.Context,
	deviceID string,
	fn func(context.Context, *device.PooledConnection) error,
	timeout time.Duration,
) error {
	idx := e.nextIdx()
	fixturePath := e.fixtureProvider(idx)

	p, err := platform.NewPlatform(
		"huawei_vrp",
		"dummy-host",
		options.WithTransportType(transport.FileTransport),
		options.WithFileTransportFile(fixturePath),
		options.WithTransportReadSize(1),
		options.WithReadDelay(0),
	)
	if err != nil {
		return fmt.Errorf("e2e: create platform: %w", err)
	}

	d, err := p.GetNetworkDriver()
	if err != nil {
		return fmt.Errorf("e2e: get driver: %w", err)
	}

	if err := d.Open(); err != nil {
		return fmt.Errorf("e2e: driver.Open: %w", err)
	}
	// Intentionally no defer Close — see type comment.

	pc := device.NewPooledConnectionForTesting(d)
	return fn(ctx, pc)
}

// singleFixture returns a fixtureProvider that always picks the same path
// regardless of call index — convenient for single-port tests.
func singleFixture(path string) func(int) string {
	return func(_ int) string { return path }
}

// e2eFixturePath resolves the absolute path of a fixture under
// internal/services/portwrite/testdata/ using runtime.Caller — robust to
// `go test ./...` invocations that change cwd.
func e2eFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}

// TestE2E_Shutdown_Huawei_HappyPath — the foundational happy-path test that
// proves the entire chain (RenderCommand → checkPreState → fileTransportExecutor
// → driver.Open → SendConfigs → parseConfigError) works end-to-end with a
// real scrapligo driver.
//
// Expected flow on the fixture:
//   - driver.Open() reads <Huawei> prompt + echoes screen-length 0 temporary
//   - service acquires default priv (exec) — no further escalation needed
//   - SendConfigs(["shutdown"]) inside the fn: scrapligo writes "shutdown"
//     into the file transport's Writes slice; it then reads the next
//     [Huawei-GE0/0/1] prompt + echo pair from the fixture
//   - parseConfigError sees Result == "" → nil → success
func TestE2E_Shutdown_Huawei_HappyPath(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

	exec := &fileTransportExecutor{
		fixtureProvider: singleFixture(e2eFixturePath(t, "huawei_shutdown_success.fixture")),
	}
	mockColl := new(mockCollectionSvc)
	svc := newTestService(exec, mockColl, db)
	ctx := context.Background()

	mockColl.On("Enqueue", "device-1").Return(nil)

	result, err := svc.Shutdown(ctx, "port-1", "e2e-op")
	if err != nil {
		t.Fatalf("Shutdown returned err: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded; result=%+v", result.Status, result)
	}
	if result.NoOp {
		t.Fatal("NoOp must be false")
	}
	if !contains(result.CommandSent, "shutdown") {
		t.Fatalf("CommandSent %q missing 'shutdown'", result.CommandSent)
	}
	if result.Action != portcollection.ActionShutdown {
		t.Fatalf("Action = %q, want %q", result.Action, portcollection.ActionShutdown)
	}

	mockColl.AssertExpectations(t)
}

// contains is a tiny substring helper to avoid pulling strings into the
// e2e file just for one assertion.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
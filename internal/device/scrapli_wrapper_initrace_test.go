// Regression tests for debug session scrapligo-getprompt-nil-panic
// (.planning/debug/scrapligo-getprompt-nil-panic.md).
//
// Production crash (2026-10-16 00:01:58, exit status 2):
//
//	transport.(*Standard).Write nil pointer dereference inside the goroutine
//	spawned by channel.GetPrompt — uncatchable by any caller-side recover(),
//	so the whole backend process died.
//
// Root cause: OpenContext's initialization window had NO mutual exclusion —
// the init goroutine (driver.Open + GetPrompt probe loop) and the ctx.Done
// fire-and-forget closer (raw w.driver.Close()) could tear down the transport
// concurrently. scrapligo v1.4.0 has zero internal synchronization between
// Transport.Write and Open/Close (Write takes no implLock; the force-close
// path bypasses implLock entirely), so the overlap is undefined behavior that
// faults during interface dispatch in a spawned goroutine.
//
// Fix under test:
//  1. init goroutine holds opMu.RLock for its whole lifetime (Open + probes);
//  2. ctx.Done routes through w.Close() (state machine + closing signal +
//     opMu.Lock ordering) instead of a raw driver.Close();
//  3. Close() is CAS-idempotent (double Close → scrapligo close(Errs) panic);
//  4. Close() releases initDone waiters before queueing on opMu.
//
// FileTransport discipline (see scrapli_wrapper_78_03_test.go header):
// exhausted fixtures block forever in Read, and Close may hang after driver
// operations — therefore every Close here runs in its own goroutine and is
// never waited on; the OS reaps leaked goroutines at test-process exit.
package device

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)

// newInitRaceWrapper injects a FileTransport factory (fixture reused every
// open — FileTransport re-reads the file per Open) and returns a fresh
// wrapper in StateInitializing WITHOUT opening it.
func newInitRaceWrapper(t *testing.T, fixture string) *ScrapliWrapper {
	t.Helper()

	orig := newNetworkDriver
	t.Cleanup(func() { newNetworkDriver = orig })

	fixturePath := factoryFixturePath(t, fixture)

	newNetworkDriver = func(_ any, _ string, _ ...util.Option) (*network.Driver, error) {
		p, err := platform.NewPlatform(
			"huawei_vrp",
			"dummy-host",
			options.WithTransportType(transport.FileTransport),
			options.WithFileTransportFile(fixturePath),
			options.WithTransportReadSize(1),
			options.WithReadDelay(0),
		)
		if err != nil {
			return nil, err
		}
		return p.GetNetworkDriver()
	}

	dev := &models.NetworkDevice{
		DeviceName: "dummy-initrace",
		IPAddress:  "dummy-host",
		Vendor:     models.VendorHuawei,
	}

	w, err := NewScrapliWrapper(dev, "u", "p", models.ProtocolTypeSSH)
	if err != nil {
		t.Fatalf("NewScrapliWrapper returned err: %v", err)
	}
	return w
}

// TestInitRace_CancelDuringInit_Hammer hammers the exact production race:
// the caller ctx is cancelled while OpenContext's init goroutine is inside
// driver.Open()/GetPrompt probing, and a pool-style Close fires concurrently
// from a second goroutine (mirroring ctx.Done closer + pool cleanup overlap).
//
// Under the fix, Close cannot start tearing the transport down while the init
// goroutine holds opMu.RLock, and double-Close is CAS-idempotent, so under
// -race there must be no data race and no panic. Against the pre-fix code
// (raw fire-and-forget driver.Close() + unlocked init loop) this overlaps
// unsynchronized scrapligo internals.
func TestInitRace_CancelDuringInit_Hammer(t *testing.T) {
	for i := 0; i < 15; i++ {
		w := newInitRaceWrapper(t, "huawei_vrp_open.fixture")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i%5)*time.Millisecond)
		resultCh := make(chan error, 1)

		go func() {
			resultCh <- w.OpenContext(ctx)
		}()

		// Concurrent pool-style closer (pool cleanup / WaitForReady timeout /
		// the OpenContext ctx.Done closer all land here in production).
		closeDone := make(chan struct{})
		go func() {
			_ = w.Close() // may block forever on exhausted FileTransport — never waited
			close(closeDone)
		}()

		select {
		case <-resultCh:
			// success or timeout error — both acceptable
		case <-time.After(5 * time.Second):
			t.Fatalf("iteration %d: OpenContext did not return within 5s — hang", i)
		}
		cancel()

		// Post-cancel the wrapper must refuse operations gracefully instead of
		// panicking (the crash regression): every op path is state-gated.
		if w.IsReady() {
			// A race where init completed before Close may legitimately leave
			// Ready momentarily; Close will still tear it down. Not a failure.
			t.Logf("iteration %d: init won the race (Ready) — Close will tear down", i)
		}
		if _, err := w.SendCommand("display version", false); err != nil && w.getState() == StateReady {
			t.Fatalf("iteration %d: SendCommand errored while Ready: %v", i, err)
		}

		// Best-effort: if Close finished quickly (init lost the race), fine;
		// if it is stuck behind the hung probe/FileTransport, leak by design.
		select {
		case <-closeDone:
		default:
		}
	}
}

// TestInitRace_HungProbe_CancelThenClose reproduces the slow/flaky-device
// shape of the production incident: driver.Open succeeds, the first GetPrompt
// probe blocks (fixture exhausted after the on-open consumption), and the
// caller ctx times out mid-probe. OpenContext must return the timeout error
// promptly, WaitForReady must be released immediately (initDone early-close),
// and subsequent API calls must degrade gracefully — with the teardown
// deferred until the probe releases opMu (never a concurrent driver.Close).
func TestInitRace_HungProbe_CancelThenClose(t *testing.T) {
	// Fixture with just enough bytes for driver.Open (prompt + on-open
	// command echo + prompt) and NOTHING more — the init probe then blocks
	// in FileTransport.Read forever, holding the init goroutine.
	// NOTE: deliberately NOT t.TempDir — the leaked init goroutine keeps the
	// fixture file handle open and TempDir's RemoveAll cleanup would fail.
	dir, err := os.MkdirTemp("", "initrace-hung")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) }) // best-effort; may fail while leaked goroutine holds the handle
	fixturePath := filepath.Join(dir, "hung_after_open.fixture")
	content := "<dummy-host>\nscreen-length 0 temporary\n<dummy-host>\n"
	if err := os.WriteFile(fixturePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	orig := newNetworkDriver
	t.Cleanup(func() { newNetworkDriver = orig })

	newNetworkDriver = func(_ any, _ string, _ ...util.Option) (*network.Driver, error) {
		p, err := platform.NewPlatform(
			"huawei_vrp",
			"dummy-host",
			options.WithTransportType(transport.FileTransport),
			options.WithFileTransportFile(fixturePath),
			options.WithTransportReadSize(1),
			options.WithReadDelay(0),
		)
		if err != nil {
			return nil, err
		}
		return p.GetNetworkDriver()
	}

	dev := &models.NetworkDevice{
		DeviceName: "dummy-hung-probe",
		IPAddress:  "dummy-host",
		Vendor:     models.VendorHuawei,
	}

	w, err := NewScrapliWrapper(dev, "u", "p", models.ProtocolTypeSSH)
	if err != nil {
		t.Fatalf("NewScrapliWrapper returned err: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.OpenContext(ctx)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatalf("OpenContext should have hit the ctx timeout while the probe was blocked")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("OpenContext did not return within 5s — hang")
	}

	// initDone must already be closed (Close ran and released waiters early),
	// so WaitForReady returns an error immediately rather than burning 30s.
	start := time.Now()
	if err := w.WaitForReady(30 * time.Second); err == nil {
		t.Fatalf("WaitForReady after cancelled init should error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("WaitForReady took %v after close — initDone was not released early", elapsed)
	}

	// Graceful degradation: state-gated refusals, never panics.
	if w.IsReady() {
		t.Fatalf("IsReady must be false after cancelled init")
	}
	if _, err := w.SendCommand("display version", false); err == nil {
		t.Fatalf("SendCommand on closing/closed wrapper must error")
	}

	// The pool would call Close here; with the closer already having run it
	// must be a cheap no-op (CAS idempotent), not a second driver.Close().
	_ = w.Close()

	// The init goroutine may still be parked in the blocked probe (FileTransport
	// select{}); per package discipline it is abandoned at process exit.
}

// TestInitRace_CloseBeforeOpen covers the residual window where Close()
// runs before the OpenContext init goroutine even starts: the init
// goroutine must notice isClosing after acquiring opMu.RLock, skip
// driver.Open, and return "连接正在关闭" instead of opening an orphan transport.
//
// Close runs in a goroutine and is never awaited: driver.Close() on a
// never-opened FileTransport driver blocks (documented package pitfall).
func TestInitRace_CloseBeforeOpen(t *testing.T) {
	w := newInitRaceWrapper(t, "huawei_vrp_open.fixture")

	go func() {
		_ = w.Close()
	}()

	// Let the closer reach setState(StateClosing) + close(w.closing).
	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- w.OpenContext(ctx)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("OpenContext after Close must error")
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("OpenContext after Close did not return within 8s — hang")
	}

	// The init goroutine must not have opened the transport: it exits via the
	// isClosing guard, so state stays in the Close lifecycle (the background
	// closer may still be parked inside driver.Close on FileTransport).
	if state := w.getState(); state != StateClosing && state != StateClosed {
		t.Fatalf("state after Close+OpenContext = %s, want Closing or Closed", state)
	}
}

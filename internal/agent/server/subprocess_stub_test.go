package server

import (
	"context"
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

// helperStubCommand builds a re-exec subprocess stub command: it executes this
// test binary (os.Args[0]) filtered down to TestHelperProcess only, passes the
// shape argument after "--" (positional args are never consumed by the testing
// flag machinery), and arms the GO_WANT_HELPER_PROCESS gate the helper guard
// requires.
//
// This is the Go stdlib subprocess-stub pattern (cf. GOROOT src/os/os_test.go).
// It replaces the former platform-dependent exec.Command("echo", ...) stubs:
// the stub is the test binary itself, so its behavior is identical on every
// platform ("echo" is a cmd.exe builtin on Windows and only worked by accident
// when a Git Bash echo.exe happened to be on PATH).
func helperStubCommand(t *testing.T, shape string) *exec.Cmd {
	t.Helper()
	ctx := context.Background()
	cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", shape)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

// TestSubprocessStub_Default verifies the fast-exit shape: the helper stub
// prints "hello" and exits 0.
func TestSubprocessStub_Default(t *testing.T) {
	cmd := helperStubCommand(t, "hello")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper stub exited with error: %v", err)
	}
	if !strings.Contains(string(output), "hello") {
		t.Fatalf("expected output to contain %q, got %q", "hello", string(output))
	}
}

// TestSubprocessStub_StdoutFlood verifies the stdout-flood shape: the helper
// stub prints 1000 lines and exits 0 (output capture must not deadlock).
func TestSubprocessStub_StdoutFlood(t *testing.T) {
	cmd := helperStubCommand(t, "stdout-flood")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper stub exited with error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 1000 {
		t.Fatalf("expected >= 1000 output lines, got %d", len(lines))
	}
	if !strings.Contains(string(output), "line") {
		t.Fatalf("expected flood output to contain %q", "line")
	}
}

// TestSubprocessStub_StdinClose verifies the long-lived shape: the helper stub
// keeps running until its stdin is closed, then exits cleanly (Wait == nil).
func TestSubprocessStub_StdinClose(t *testing.T) {
	cmd := helperStubCommand(t, "sleep-until-stdin-close")
	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Closing stdin is the shutdown signal for this shape.
	if err := pipe.Close(); err != nil {
		t.Fatalf("close stdin pipe: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait after stdin close: %v", err)
	}
}

// TestSubprocessStub_IgnoreSigterm verifies the ignore-sigterm shape: the
// helper stub arms a SIGTERM handler, survives a SIGTERM, prints
// "still-alive" and only then exits. syscall.SIGTERM has no meaning on
// Windows, so the test is linux-only.
func TestSubprocessStub_IgnoreSigterm(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("syscall.SIGTERM is not meaningful on %s", runtime.GOOS)
	}

	cmd := helperStubCommand(t, "ignore-sigterm")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait until the child reports its signal handler is armed, so the
	// SIGTERM we send can never race the handler installation and kill the
	// process via the default disposition.
	scanner := bufio.NewScanner(stdout)
	armed := false
	for scanner.Scan() {
		if scanner.Text() == "sigterm-armed" {
			armed = true
			break
		}
	}
	if !armed {
		_ = cmd.Wait()
		t.Fatalf("helper stub never armed its SIGTERM handler (scan err: %v)", scanner.Err())
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	stillAlive := false
	for scanner.Scan() {
		if scanner.Text() == "still-alive" {
			stillAlive = true
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait after SIGTERM: %v", err)
	}
	if !stillAlive {
		t.Fatal("expected stub to print still-alive after ignoring SIGTERM")
	}
}

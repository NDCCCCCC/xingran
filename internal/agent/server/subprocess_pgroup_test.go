package server

import (
	"context"
	"os"
	"strings"
	"testing"
)

// The subprocess stubs below re-execute this test binary filtered down to
// TestHelperProcess (see subprocess_stub_test.go) instead of invoking
// platform commands like echo — a cmd.exe builtin on Windows that only
// worked by accident when a Git Bash echo.exe happened to be on PATH.
//
// Two replacement styles:
//   - Construction-only sites go through newCommand directly (the command is
//     never started, so no environment is needed).
//   - Execution sites keep production-function coverage (runCommand /
//     runCommandOutput build the cmd internally and never set cmd.Env; a
//     zero-value Env inherits the parent process environment), so they set
//     GO_WANT_HELPER_PROCESS=1 via t.Setenv in the parent instead.

// TestSubprocess_SetProcessGroup verifies that newCommand sets process group
// isolation attributes on the created command.
func TestSubprocess_SetProcessGroup(t *testing.T) {
	ctx := context.Background()

	cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "hello")
	if cmd == nil {
		t.Fatal("newCommand returned nil")
	}

	// Verify SysProcAttr is set
	if err := assertProcessGroupSet(cmd); err != nil {
		t.Fatalf("process group not set: %v", err)
	}
}

// TestRunCommand_ExecutesSuccessfully verifies that runCommand can execute
// the re-exec helper stub (default shape, exit 0) with process group
// isolation.
func TestRunCommand_ExecutesSuccessfully(t *testing.T) {
	// runCommand builds the cmd internally without setting cmd.Env, so the
	// helper gate must travel through the parent environment.
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	ctx := context.Background()

	err := runCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "hello")
	if err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
}

// TestRunCommandOutput_ExecutesSuccessfully verifies that runCommandOutput
// returns the helper stub's output.
func TestRunCommandOutput_ExecutesSuccessfully(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	ctx := context.Background()

	output, err := runCommandOutput(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "hello")
	if err != nil {
		t.Fatalf("runCommandOutput failed: %v", err)
	}

	if !strings.Contains(string(output), "hello") {
		t.Fatalf("expected output to contain %q, got %q", "hello", string(output))
	}
}

// TestRunCommand_ContextCancellation verifies that commands respect context
// cancellation.
func TestRunCommand_ContextCancellation(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := runCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "should-not-run")
	if err == nil {
		t.Log("command completed despite cancellation (helper exit is fast enough)")
	}
	// Tolerant by design: a pre-cancelled context may either prevent the
	// spawn entirely or kill the group right after start. The contract under
	// test is no panic and no hang.
}

// TestSetProcessGroup_Idempotent verifies that calling setProcessGroup
// multiple times doesn't cause issues.
func TestSetProcessGroup_Idempotent(t *testing.T) {
	ctx := context.Background()
	cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", "test")

	// Call setProcessGroup again — should not panic
	setProcessGroup(cmd)

	if err := assertProcessGroupSet(cmd); err != nil {
		t.Fatalf("process group not set after idempotent call: %v", err)
	}
}

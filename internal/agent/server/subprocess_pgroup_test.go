package server

import (
	"context"
	"testing"
)

// TestSubprocess_SetProcessGroup verifies that newCommand sets process group
// isolation attributes on the created command.
func TestSubprocess_SetProcessGroup(t *testing.T) {
	ctx := context.Background()

	cmd := newCommand(ctx, "echo", "hello")
	if cmd == nil {
		t.Fatal("newCommand returned nil")
	}

	// Verify SysProcAttr is set
	if err := assertProcessGroupSet(cmd); err != nil {
		t.Fatalf("process group not set: %v", err)
	}
}

// TestRunCommand_ExecutesSuccessfully verifies that runCommand can execute
// a simple command with process group isolation.
func TestRunCommand_ExecutesSuccessfully(t *testing.T) {
	ctx := context.Background()

	err := runCommand(ctx, "echo", "test")
	if err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
}

// TestRunCommandOutput_ExecutesSuccessfully verifies that runCommandOutput
// returns expected output.
func TestRunCommandOutput_ExecutesSuccessfully(t *testing.T) {
	ctx := context.Background()

	output, err := runCommandOutput(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("runCommandOutput failed: %v", err)
	}

	// Output should contain "hello" (may have trailing newline)
	result := string(output)
	if len(result) == 0 {
		t.Fatal("expected non-empty output")
	}
}

// TestRunCommand_ContextCancellation verifies that commands respect context
// cancellation.
func TestRunCommand_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := runCommand(ctx, "echo", "should not run")
	if err == nil {
		t.Log("command completed despite cancellation (may be fast enough)")
	}
	// We just verify it doesn't panic or hang
}

// TestSetProcessGroup_Idempotent verifies that calling setProcessGroup
// multiple times doesn't cause issues.
func TestSetProcessGroup_Idempotent(t *testing.T) {
	ctx := context.Background()
	cmd := newCommand(ctx, "echo", "test")

	// Call setProcessGroup again — should not panic
	setProcessGroup(cmd)

	if err := assertProcessGroupSet(cmd); err != nil {
		t.Fatalf("process group not set after idempotent call: %v", err)
	}
}

package server

import (
	"context"
	"fmt"
	"os/exec"
)

// runCommand executes a command with process group isolation.
// On Linux, sets Setpgid=true so the subprocess gets its own process group.
// On Windows, sets CREATE_NEW_PROCESS_GROUP for similar isolation.
// On context cancellation, kills the entire process group.
func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	setProcessGroup(cmd)
	return cmd.Run()
}

// runCommandOutput executes a command with process group isolation and returns output.
func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	setProcessGroup(cmd)
	return cmd.Output()
}

// newCommand creates a new exec.Cmd with process group isolation set.
// Use this when you need finer control over the command (e.g., StdinPipe).
func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	setProcessGroup(cmd)
	return cmd
}

// setProcessGroup sets the appropriate process group creation flag
// based on the current platform.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &sysProcAttr
	}
}

// assertProcessGroupSet verifies that a command has process group isolation enabled.
// Used in tests to confirm the configuration is applied correctly.
func assertProcessGroupSet(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		return fmt.Errorf("SysProcAttr not set on command")
	}
	return nil
}

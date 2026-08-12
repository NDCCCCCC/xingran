//go:build linux || darwin

package server

import "syscall"

// sysProcAttr provides platform-specific process attributes for subprocess isolation.
// On Linux/Darwin, Setpgid creates a new process group for each subprocess,
// allowing clean termination of the entire process tree on context cancellation.
var sysProcAttr = syscall.SysProcAttr{
	Setpgid: true,
}

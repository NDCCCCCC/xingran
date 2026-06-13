//go:build windows

package server

import "syscall"

// sysProcAttr provides platform-specific process attributes for subprocess isolation.
// On Windows, CREATE_NEW_PROCESS_GROUP creates a new process group for each subprocess,
// allowing clean termination of the entire process tree on context cancellation.
var sysProcAttr = syscall.SysProcAttr{
	CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
}

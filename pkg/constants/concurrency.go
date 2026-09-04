package constants

// concurrency.go — command execution concurrency constants.

const (
	// CommandConcurrency is the default concurrent command execution limit.
	// Shared by command_handler.go:58 and execution_handler.go:99 (D-06).
	CommandConcurrency = 10
)

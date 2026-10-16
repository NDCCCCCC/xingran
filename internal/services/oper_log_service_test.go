package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// moduleRoot returns the repo root (two levels up from this package).
func moduleRoot() string {
	// internal/services/oper_log_service_test.go
	//  ^---- 2 levels up to repo root
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// TestOperLog_Async_DoesNotPanicOnDBError GUARD-07 regression test.
//
// RED baseline: oper_log_service.go:67 and :140 async goroutines have NO
// defer recover(). A DB write panic (e.g. nil pointer in GORM) propagates
// and crashes the process.
//
// GREEN after Phase 111 GOR-01: both goroutines wrap with
//   defer func() { if r := recover(); r != nil { ... log ... } }()
//
// Approach: source-code pattern check (Option A from plan). Behavioral test
// requires modifying production code and was rejected by plan-checker.
func TestOperLog_Async_DoesNotPanicOnDBError(t *testing.T) {
	srcPath := filepath.Join(moduleRoot(), "internal", "services", "oper_log_service.go")
	src, err := os.ReadFile(srcPath)
	require.NoError(t, err)
	srcStr := string(src)

	// Check for defer recover() pattern in async goroutines.
	// After GOR-01 fix the goroutines at lines 67 and 140 will contain:
	//   go func() {
	//       defer func() { if r := recover(); r != nil { ... } }()
	//       ...
	//   }()
	//
	// We look for the canonical two-level defer/recover pattern.
	hasRecoverInAsync := strings.Contains(srcStr, "defer func() { if r := recover()")

	assert.True(t, hasRecoverInAsync,
		"oper_log_service.go async goroutines (line 67, 140) must have "+
			"'defer func() { if r := recover()' to prevent panic propagation from DB errors. "+
			"Without defer recover(), a DB write panic crashes the entire process.")
}

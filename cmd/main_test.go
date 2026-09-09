package main

import (
	"os"
	"strings"
	"testing"
)

// TestMain_AllowedOrigins_FromConfig_NotWildcard verifies that main.go does NOT
// hardcode allowedOrigins to []string{"*"} and instead reads from config.
//
// RED baseline: This test FAILS because cmd/main.go:104 currently hardcodes:
//   allowedOrigins := []string{"*"}
//
// GREEN after Phase 110 TLS-04: When the hardcoded wildcard is removed and replaced
// with a config read (e.g., allowedOrigins := cfg.Server.AllowedOrigins), the test PASSES.
//
// This is a source-code verification test appropriate for Phase 109 regression guard.
func TestMain_AllowedOrigins_FromConfig_NotWildcard(t *testing.T) {
	// Read the main.go source file
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Failed to read cmd/main.go: %v", err)
	}

	src := string(source)

	// The regression guard: allowedOrigins must NOT be hardcoded to wildcard.
	// Pattern that MUST NOT exist in production code:
	hardcodedWildcard := `allowedOrigins := []string{"*"}`

	if strings.Contains(src, hardcodedWildcard) {
		t.Errorf("allowedOrigins should NOT be hardcoded to []string{\"*\"}; "+
			"should read from config (cfg.Server.AllowedOrigins). "+
			"Found hardcoded wildcard at cmd/main.go:104")
	}

	// Log direction for visibility
	if !strings.Contains(src, hardcodedWildcard) {
		t.Log("allowedOrigins is not hardcoded to wildcard — correct direction")
	}
}

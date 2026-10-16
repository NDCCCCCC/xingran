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

// TestMain_AllowedOrigins_ExampleConfigsHavePlaceholder guards against the
// v1.33 deploy-crash regression (2026-09-14): Phase 110 TLS-04 added a fatal
// check in cmd/main.go:106 that requires server.allowed_origins in production,
// but the example configs (config.prod.example.yaml, config.sqlite.example.yaml)
// did NOT include the field. Fresh deployments copying from examples triggered
// the fatal → silent exit 1 → deploy script rolled back to pre-Phase 110 binary.
//
// Production config had no allowed_origins either, so every deploy attempt
// since 2026-09-09 failed (3 consecutive deploys all rolled back). This test
// ensures any future example-config change keeps the required field present
// so fresh deployments don't silently trip the fatal.
func TestMain_AllowedOrigins_ExampleConfigsHavePlaceholder(t *testing.T) {
	requiredFiles := []string{
		"../configs/config.prod.example.yaml",
		"../configs/config.sqlite.example.yaml",
	}

	for _, path := range requiredFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", path, err)
		}
		content := string(data)
		if !strings.Contains(content, "allowed_origins") {
			t.Errorf("%s MUST contain 'allowed_origins' field under server section "+
				"(Phase 110 TLS-04 production requirement). "+
				"Missing this causes fatal exit in cmd/main.go:106 and silent deploy rollback.",
				path)
		}
		// Sanity: at least one non-wildcard origin should be present in examples
		// (wildcard '*' defeats the Phase 110 intent)
		if !strings.Contains(content, "your-production-domain") &&
			!strings.Contains(content, "your-domain") {
			t.Logf("Note: %s may use a different placeholder — verify it's not '*'",
				path)
		}
	}
}

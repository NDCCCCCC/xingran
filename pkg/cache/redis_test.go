package cache

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// =====================================================================
// GUARD-01: TestRedis_TLSConfig_NotInsecureByDefault
//
// RED state (Phase 109): pkg/cache/redis.go:35 hardcodes InsecureSkipVerify: true
// GREEN state (Phase 110 TLS-01): env var REDIS_TLS_INSECURE_SKIP_VERIFY controls
//   the value; default (unset/empty) = false = secure default.
//
// This test verifies the source code contains the env-var control pattern
// rather than a hardcoded InsecureSkipVerify: true. The test FAILS in RED
// (hardcoded true found) and PASSES in GREEN (env-var pattern found).
//
// Strategy: source pattern inspection — read redis.go, extract the TLS config
// block, assert it is NOT hardcoded to true and instead uses env-var control.
// =====================================================================

// TestRedis_TLSConfig_NotInsecureByDefault GUARD-01 regression test.
//
// Assertion: When config.TLS=true and REDIS_TLS_INSECURE_SKIP_VERIFY is not
// set (or is empty), the TLS config InsecureSkipVerify field must be false
// (secure default). This test inspects the source code of NewRedisCache in
// redis.go to determine whether InsecureSkipVerify is hardcoded to true.
//
// RED (Phase 109 baseline): hardcoded InsecureSkipVerify: true found → FAIL
// GREEN (Phase 110 TLS-01): env-var control pattern found → PASS
func TestRedis_TLSConfig_NotInsecureByDefault(t *testing.T) {
	// Ensure the env var is not set (secure default path)
	oldVal := os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY")
	os.Unsetenv("REDIS_TLS_INSECURE_SKIP_VERIFY")
	defer func() {
		if oldVal != "" {
			os.Setenv("REDIS_TLS_INSECURE_SKIP_VERIFY", oldVal)
		}
	}()

	// Read the source file
	src, err := os.ReadFile("redis.go")
	if err != nil {
		t.Fatalf("os.ReadFile(redis.go): %v", err)
	}
	content := string(src)

	// Locate the TLS config block within NewRedisCache.
	// We look for the pattern between "if config.TLS {" and the closing brace
	// of the if block, capturing the InsecureSkipVerify assignment.
	tlsBlockRx := regexp.MustCompile(`if\s+config\.TLS\s*\{[^}]*tlsCfg\s*=\s*&tls\.Config\{([^}]+)\}`)
	matches := tlsBlockRx.FindStringSubmatch(content)
	if matches == nil {
		t.Fatal("TLS config block not found in redis.go — NewRedisCache signature or structure may have changed")
	}

	tlsCfgBody := matches[1] // e.g. "InsecureSkipVerify: true" or "InsecureSkipVerify: <env-var-expr>"

	// Check for the INSECURE hardcoded pattern: InsecureSkipVerify: true
	// with no os.Getenv, strconv, or other env-var reference.
	// Strategy: extract the RHS of InsecureSkipVerify and verify it's not literally "true"
	// without any function call wrapper.
	insecureRHSRx := regexp.MustCompile(`(?i)InsecureSkipVerify\s*:\s*(\S+)`)
	rhsMatch := insecureRHSRx.FindStringSubmatch(tlsCfgBody)
	isHardcodedTrue := false
	if rhsMatch != nil {
		rhs := strings.TrimSpace(rhsMatch[1]) // e.g. "true" or "someFunc()"
		// Hardcoded true if RHS is exactly "true" (not a function call)
		isHardcodedTrue = rhs == "true"
	}

	if isHardcodedTrue {
		t.Errorf("InsecureSkipVerify is hardcoded to true in TLS config — this is INSECURE.\n"+
			"Expected: env-var control (e.g. os.Getenv(\"REDIS_TLS_INSECURE_SKIP_VERIFY\") == \"true\")\n"+
			"Found TLS config body: %s\n"+
			"FIX: Phase 110 TLS-01 must replace hardcoded true with env-var controlled value",
			strings.TrimSpace(tlsCfgBody))
		return
	}

	// Verify that if InsecureSkipVerify is present, it uses an env-var expression.
	// Acceptable patterns:
	//   - InsecureSkipVerify: os.Getenv("REDIS_TLS_INSECURE_SKIP_VERIFY") == "true"
	//   - InsecureSkipVerify: valueFromEnvVar
	//   - InsecureSkipVerify: someFunc()  (as long as not hardcoded true)
	//   - No InsecureSkipVerify field at all (defaults to false, secure)
	hasInsecureSkipVerify := regexp.MustCompile(`(?i)InsecureSkipVerify`).MatchString(tlsCfgBody)

	if hasInsecureSkipVerify {
		// InsecureSkipVerify is present but NOT hardcoded to true — good.
		// Verify it uses an env-var (or equivalent dynamic) expression.
		usesEnvVar := regexp.MustCompile(`(?i)Getenv\s*\(\s*"REDIS_TLS_INSECURE_SKIP_VERIFY"\s*\)`).
			MatchString(tlsCfgBody)

		if !usesEnvVar {
			t.Errorf("InsecureSkipVerify is present but does not use REDIS_TLS_INSECURE_SKIP_VERIFY env var.\n"+
				"Found TLS config body: %s\n"+
				"FIX: Phase 110 TLS-01 must use env var for InsecureSkipVerify control",
				strings.TrimSpace(tlsCfgBody))
			return
		}

		t.Log("PASS: InsecureSkipVerify uses REDIS_TLS_INSECURE_SKIP_VERIFY env var (secure default)")
	} else {
		// No InsecureSkipVerify field — TLS config is clean, defaults to secure.
		t.Log("PASS: No InsecureSkipVerify field in TLS config (secure by default)")
	}
}

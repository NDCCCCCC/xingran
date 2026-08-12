// P1-S5 (Phase 32 Wave 2) regression test: DefaultPasswordConfig.Iterations = 600000
// (OWASP 2023 baseline) and backward compatibility for hashes created with the
// previous 100000 iterations.
//
// These tests live in `package security` (not `security_test`) so they can
// access the unexported `pbkdf2SM3` helper and parse the stored hash format
// the same way VerifyPassword does.
package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestPasswordManager_DefaultIterationsAre600k
// Verifies that the package-level default config has Iterations == 600000
// (the OWASP 2023 PBKDF2 baseline) and that a freshly hashed password embeds
// that same iteration count in its stored format.
func TestPasswordManager_DefaultIterationsAre600k(t *testing.T) {
	// 1. The default config literal must be 600000.
	if DefaultPasswordConfig == nil {
		t.Fatal("DefaultPasswordConfig is nil")
	}
	if DefaultPasswordConfig.Iterations != 600000 {
		t.Fatalf("DefaultPasswordConfig.Iterations = %d, want 600000 (OWASP 2023 baseline)",
			DefaultPasswordConfig.Iterations)
	}

	// 2. Hashing with the default manager should embed 600000 in the hash.
	pm := NewPasswordManager(nil) // nil => default config
	hashed, err := pm.HashPassword("hello-world-2026")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Stored format: $sm3$<iterations>$<salt>$<hash>
	parts := strings.Split(hashed, "$")
	if len(parts) != 5 || parts[1] != "sm3" {
		t.Fatalf("hash format unexpected: %q", hashed)
	}
	embedded, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("failed to parse embedded iterations: %v", err)
	}
	if embedded != 600000 {
		t.Fatalf("embedded iterations in fresh hash = %d, want 600000", embedded)
	}
}

// TestPasswordManager_VerifyBackwardCompat_100k
// Constructs a hash using the legacy 100000 iteration count and verifies that
// VerifyPassword still accepts it. This proves that users whose password hashes
// were created before the 600k bump can still log in.
//
// Speed note: 100k SM3 iterations run in well under a second on developer
// hardware, so this test stays in the standard unit-test budget.
func TestPasswordManager_VerifyBackwardCompat_100k(t *testing.T) {
	pm := NewPasswordManager(nil)

	// Build a legacy 100k-iter hash with a known plaintext.
	const legacyIterations = 100000
	const plaintext = "backward-compat-100k"

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("rand.Read failed: %v", err)
	}
	hashBytes := pm.pbkdf2SM3([]byte(plaintext), salt, legacyIterations, 32)

	legacyHash := fmt.Sprintf("$sm3$%d$%s$%s",
		legacyIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hashBytes),
	)

	// 1. Correct plaintext verifies against the legacy hash.
	ok, err := pm.VerifyPassword(plaintext, legacyHash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error on legacy 100k hash: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword rejected a valid 100k-iteration hash (backward compat broken)")
	}

	// 2. Wrong plaintext must still be rejected.
	ok, err = pm.VerifyPassword("not-the-password", legacyHash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error on wrong password: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword accepted wrong password for 100k hash")
	}
}

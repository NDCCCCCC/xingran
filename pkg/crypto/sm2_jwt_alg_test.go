// P1-S1 (Phase 32 Wave 2) regression test: SM2 JWT must reject tokens that
// declare an unexpected `alg` header value, including the classic confusion
// attacks `alg=none` and `alg=HS256` (where the attacker uses the server's
// public key as an HMAC secret).
//
// The validator's defensive check was added in commit 64b1b40; these tests
// pin that behavior so a future refactor cannot silently re-open the hole.
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestSM2JWT_RejectsAlgNone verifies that a JWT with header alg=none is
// rejected. Per RFC 7519, an alg=none token has an empty signature segment;
// even so, the server must not accept it because it would mean "no signature
// required".
func TestSM2JWT_RejectsAlgNone(t *testing.T) {
	// Fresh key pair per test — hermetic.
	_, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	headerJSON := `{"alg":"none","typ":"JWT"}`
	payloadJSON := `{"sub":"admin","exp":99999999999}`
	header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	signature := "" // alg=none → empty signature segment
	token := header + "." + payload + "." + signature

	claims, err := ValidateTokenWithSM2(token, pubKey)
	if err == nil {
		t.Fatalf("expected ValidateTokenWithSM2 to reject alg=none, got claims=%+v", claims)
	}
	// Error message should mention "algorithm" or "alg" so operators can
	// diagnose quickly.
	msg := err.Error()
	if !strings.Contains(msg, "lgorithm") && !strings.Contains(msg, "alg") {
		t.Fatalf("error message should reference algorithm/alg, got: %q", msg)
	}
}

// TestSM2JWT_RejectsAlgHS256Confusion verifies the classic algorithm-confusion
// attack: an attacker takes the server's public key (often leaked via a
// /jwks.json endpoint or certificate) and uses it as the HMAC secret to sign
// a token with alg=HS256. A naive validator would call HMAC-Verify and
// succeed because the public key is exactly the secret the attacker used.
//
// The defense is to reject any token whose header alg differs from the
// expected SM2 algorithm name BEFORE attempting signature verification.
func TestSM2JWT_RejectsAlgHS256Confusion(t *testing.T) {
	_, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Serialize the SM2 public key to raw bytes the attacker would use as
	// their HMAC secret. Use the project's ExportPublicKeyToHex helper so
	// the byte representation matches what would actually leak.
	pubHex := ExportPublicKeyToHex(pubKey)
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil {
		t.Fatalf("hex decode of public key failed: %v", err)
	}

	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	payloadJSON := `{"sub":"admin","exp":99999999999,"iss":"attacker"}`
	header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	signingString := header + "." + payload

	mac := hmac.New(sha256.New, pubBytes)
	mac.Write([]byte(signingString))
	sig := mac.Sum(nil)
	signature := base64.RawURLEncoding.EncodeToString(sig)
	token := signingString + "." + signature

	claims, err := ValidateTokenWithSM2(token, pubKey)
	if err == nil {
		t.Fatalf("expected ValidateTokenWithSM2 to reject alg=HS256, got claims=%+v", claims)
	}
	if !strings.Contains(err.Error(), "lgorithm") {
		t.Fatalf("error message should mention algorithm mismatch, got: %q", err.Error())
	}
}

// TestSM2JWT_AcceptsCorrectSM2Alg is the positive control: a token correctly
// signed with the SM2 algorithm must validate. This proves the rejection in
// the previous tests is due to the alg check, not a broken validator.
func TestSM2JWT_AcceptsCorrectSM2Alg(t *testing.T) {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	claims := &Claims{
		UserID:   "u-test-001",
		Username: "alice",
		Issuer:   "XingRan-Next",
		Subject:  "u-test-001",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token, err := GenerateTokenWithSM2(claims, privKey)
	if err != nil {
		t.Fatalf("GenerateTokenWithSM2 failed: %v", err)
	}

	got, err := ValidateTokenWithSM2(token, pubKey)
	if err != nil {
		t.Fatalf("ValidateTokenWithSM2 rejected a valid SM2 token: %v", err)
	}
	if got == nil {
		t.Fatal("ValidateTokenWithSM2 returned nil claims on valid token")
	}
	if got.UserID != "u-test-001" {
		t.Fatalf("claims.UserID = %q, want u-test-001", got.UserID)
	}
	if got.Username != "alice" {
		t.Fatalf("claims.Username = %q, want alice", got.Username)
	}
}

package vdi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestVDIError tests VDIError implementation
func TestVDIError(t *testing.T) {
	err := &VDIError{
		Code:    1001,
		Message: "Test error",
	}

	expected := "VDI API error 1001: Test error"
	assert.Equal(t, expected, err.Error())
}

// TestVDIAuthManagerTokenExpiry tests token expiry logic
func TestVDIAuthManagerTokenExpiry(t *testing.T) {
	manager := &VDIAuthManager{}

	// Test expired token
	expiredTime := time.Now().Add(-1 * time.Hour)
	assert.True(t, manager.isTokenExpired(&expiredTime))

	// Test valid token
	validTime := time.Now().Add(1 * time.Hour)
	assert.False(t, manager.isTokenExpired(&validTime))

	// Test nil token
	assert.True(t, manager.isTokenExpired(nil))

	// Test token within 5 minutes (should be considered expired)
	soonTime := time.Now().Add(3 * time.Minute)
	assert.True(t, manager.isTokenExpired(&soonTime))
}

// TestDecryptVDIPassword tests password decryption
func TestDecryptVDIPassword(t *testing.T) {
	// Test encryption/decryption roundtrip
	originalPassword := "test_password_123"
	encrypted := encryptVDIPassword(originalPassword)
	decrypted := decryptVDIPassword(encrypted)

	assert.Equal(t, originalPassword, decrypted)
	assert.NotEqual(t, originalPassword, encrypted)
}

// TestNewVDIAuthManager tests auth manager creation
// Skipped: NewVDIAuthManager requires a real database connection for token caching
func TestNewVDIAuthManager(t *testing.T) {
	t.Skip("Skipping: NewVDIAuthManager requires a real database connection")
}

// TestNewVDIClientExtended tests extended client creation
// Skipped: NewVDIClientExtended requires a real database connection (calls db.First internally)
func TestNewVDIClientExtended(t *testing.T) {
	t.Skip("Skipping: NewVDIClientExtended requires a real database connection")
}

// TestVDIClientExtendedInterface tests that the extended client implements the interface
// Skipped: NewVDIClientExtended requires a real database connection (calls db.First internally)
func TestVDIClientExtendedInterface(t *testing.T) {
	t.Skip("Skipping: NewVDIClientExtended requires a real database connection")
}

// TestVDIClientExtendedMethods tests method signatures (without actual API calls)
func TestVDIClientExtendedMethods(t *testing.T) {
	// Skip this test since it requires a real database
	t.Skip("Skipping test that requires database connection")
}

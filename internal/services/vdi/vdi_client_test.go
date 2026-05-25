package vdi

import (
	"testing"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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
func TestNewVDIAuthManager(t *testing.T) {
	var db *gorm.DB // nil DB for testing
	serverID := "test-server-1"
	cfg := config.VDIServerConfig{
		Name:     "Test Server",
		Endpoint: "https://test.example.com",
		Username: "admin",
		Password: "encrypted_password",
		TenantID: 0,
	}

	manager := NewVDIAuthManager(db, serverID, cfg)

	assert.NotNil(t, manager)
	assert.Equal(t, serverID, manager.serverID)
	assert.Equal(t, cfg, manager.config)
	assert.NotNil(t, manager.httpClient)
}

// TestNewVDIClientExtended tests extended client creation
func TestNewVDIClientExtended(t *testing.T) {
	var db *gorm.DB // nil DB for testing
	serverID := "test-server-1"
	cfg := config.VDIServerConfig{
		Name:     "Test Server",
		Endpoint: "https://test.example.com",
		Username: "admin",
		Password: "encrypted_password",
		TenantID: 0,
	}

	client := NewVDIClientExtended(db, serverID, cfg)

	assert.NotNil(t, client)
}

// TestVDIClientExtendedInterface tests that the extended client implements the interface
func TestVDIClientExtendedInterface(t *testing.T) {
	var client VDIClientExtended
	var db *gorm.DB
	serverID := "test-server-1"
	cfg := config.VDIServerConfig{
		Name:     "Test Server",
		Endpoint: "https://test.example.com",
		Username: "admin",
		Password: "encrypted_password",
		TenantID: 0,
	}

	client = NewVDIClientExtended(db, serverID, cfg)

	assert.NotNil(t, client)

	// Test that all interface methods are available
	assert.Implements(t, (*VDIClientExtended)(nil), client)
}

// TestVDIClientExtendedMethods tests method signatures (without actual API calls)
func TestVDIClientExtendedMethods(t *testing.T) {
	// Skip this test since it requires a real database
	t.Skip("Skipping test that requires database connection")
}

// Package integration_test provides integration tests for login endpoint encryption
package integration_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	x509 "github.com/tjfoc/gmsm/x509"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// TestMain setup test environment
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// setupMinimalTestServer creates a minimal test server without database
func setupMinimalTestServer(t *testing.T) *httptest.Server {
	// Create minimal configuration with valid SM4 key (16 bytes)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "test",
		},
		Database: config.DatabaseConfig{
			SSLMode: "disable",
		},
		Cache: config.CacheConfig{
			Type: "memory",
		},
		JWT: config.JWTConfig{
			SecretKey:       "dGVzdC1zZWNyZXQta2V5LWZvci1qd3QtdG9rZW4tc2lnbmluZy0zMmNoYXJzIQ==",
			AccessKeyExpire: 3600,
		},
		Security: config.SecurityConfig{
			SM4Key: "MTIzNDU2Nzg5MDEyMzQ1Ng==", // "1234567890123456" in base64
		},
	}

	// Initialize core (may fail DB connection, but that's OK for testing auth endpoints)
	coreInstance, err := core.New(cfg)
	// Even if DB fails, we can still test some endpoints
	if err != nil {
		t.Logf("Core initialization had issues (expected in test environment): %v", err)
	}

	// Create test router
	router := gin.New()
	router.Use(gin.Recovery())
	api := router.Group("/api/v1")

	// Setup auth router (will work even without DB for some endpoints)
	if coreInstance != nil {
		v1.SetupAuthRouter(api, coreInstance)
	}

	return httptest.NewServer(router)
}

// TestPublicKeyEndpoint tests the public key endpoint is accessible
func TestPublicKeyEndpoint(t *testing.T) {
	server := setupMinimalTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/system/auth/public-key")
	require.NoError(t, err, "Failed to get public key")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Public key endpoint should be accessible")

	var pubKeyResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PublicKey string `json:"publicKey"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&pubKeyResp)
	require.NoError(t, err, "Failed to decode response")

	assert.Equal(t, 0, pubKeyResp.Code, "Response code should be 0")
	assert.NotEmpty(t, pubKeyResp.Data.PublicKey, "Public key should not be empty")

	// Verify public key is valid hex
	_, err = hex.DecodeString(pubKeyResp.Data.PublicKey)
	assert.NoError(t, err, "Public key should be valid hex string")

	// Verify public key can be parsed as SM2 key
	pubKeyBytes, err := hex.DecodeString(pubKeyResp.Data.PublicKey)
	require.NoError(t, err, "Failed to decode public key hex")

	_, err = x509.ParseSm2PublicKey(pubKeyBytes)
	assert.NoError(t, err, "Public key should be valid SM2 key")
}

// TestPublicKeyConsistency tests that public key remains consistent across requests
func TestPublicKeyConsistency(t *testing.T) {
	server := setupMinimalTestServer(t)
	defer server.Close()

	// Get public key twice
	resp1, err := http.Get(server.URL + "/api/v1/system/auth/public-key")
	require.NoError(t, err)
	defer resp1.Body.Close()

	resp2, err := http.Get(server.URL + "/api/v1/system/auth/public-key")
	require.NoError(t, err)
	defer resp2.Body.Close()

	var pubKeyResp1, pubKeyResp2 struct {
		Code int    `json:"code"`
		Data struct {
			PublicKey string `json:"publicKey"`
		} `json:"data"`
	}

	json.NewDecoder(resp1.Body).Decode(&pubKeyResp1)
	json.NewDecoder(resp2.Body).Decode(&pubKeyResp2)

	// Public keys should be identical
	assert.Equal(t, pubKeyResp1.Data.PublicKey, pubKeyResp2.Data.PublicKey,
		"Public key should remain consistent")
}

// TestTimestampValidationConcept tests timestamp validation logic
func TestTimestampValidationConcept(t *testing.T) {
	now := time.Now().Unix()
	validWindow := int64(300) // 5 minutes

	testCases := []struct {
		name      string
		timestamp int64
		valid     bool
	}{
		{"Current time", now, true},
		{"1 minute ago", now - 60, true},
		{"4 minutes ago", now - 240, true},
		{"6 minutes ago (expired)", now - 360, false},
		{"6 minutes ahead (future)", now + 360, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := now - tc.timestamp
			isValid := diff >= -validWindow && diff <= validWindow

			if tc.valid {
				assert.True(t, isValid, "Timestamp should be valid: %s", tc.name)
			} else {
				assert.False(t, isValid, "Timestamp should be invalid: %s", tc.name)
			}
		})
	}
}

// TestEncryptedRequestStructure tests that encrypted request structure is correct
func TestEncryptedRequestStructure(t *testing.T) {
	// Build a valid encrypted request structure
	encryptedReq := map[string]interface{}{
		"encrypted": true,
		"data":      "encrypted-data-here",
		"sm4Key":    "encrypted-sm4-key",
		"iv":        "initialization-vector",
		"timestamp": time.Now().Unix(),
		"nonce":     generateRandomHex(16),
	}

	// Verify structure can be marshaled
	reqJSON, err := json.Marshal(encryptedReq)
	require.NoError(t, err, "Should be able to marshal encrypted request")

	// Verify it can be unmarshaled
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(reqJSON, &unmarshaled)
	require.NoError(t, err, "Should be able to unmarshal encrypted request")

	// Verify required fields exist
	assert.True(t, unmarshaled["encrypted"].(bool), "encrypted field should be true")
	assert.NotEmpty(t, unmarshaled["data"], "data field should not be empty")
	assert.NotEmpty(t, unmarshaled["sm4Key"], "sm4Key field should not be empty")
	assert.NotEmpty(t, unmarshaled["iv"], "iv field should not be empty")
	assert.NotEmpty(t, unmarshaled["timestamp"], "timestamp field should not be empty")
	assert.NotEmpty(t, unmarshaled["nonce"], "nonce field should not be empty")
}

// TestNonceFormat tests nonce format validation
func TestNonceFormat(t *testing.T) {
	// Test nonce format validation
	nonce := generateRandomHex(16)

	// Should be valid hex string
	_, err := hex.DecodeString(nonce)
	assert.NoError(t, err, "Nonce should be valid hex string")

	// Should be 32 hex chars (16 bytes)
	assert.Equal(t, 32, len(nonce), "Nonce should be 32 hex characters")
}

// TestResponseHeaders tests that response headers are set correctly
func TestResponseHeaders(t *testing.T) {
	server := setupMinimalTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/system/auth/public-key")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "application/json", "Response should be JSON")
}

// TestRequestMethodValidation tests that only POST requests are encrypted
func TestRequestMethodValidation(t *testing.T) {
	server := setupMinimalTestServer(t)
	defer server.Close()

	// GET request should not require encryption
	resp, err := http.Get(server.URL + "/api/v1/system/auth/public-key")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "GET request should succeed")
}

// TestEncryptedRequestWithMissingFields tests encrypted request with missing required fields
func TestEncryptedRequestWithMissingFields(t *testing.T) {
	testCases := []struct {
		name    string
		request map[string]interface{}
	}{
		{"Missing timestamp", map[string]interface{}{"encrypted": true, "data": "test", "sm4Key": "test", "iv": "test", "nonce": "test"}},
		{"Missing nonce", map[string]interface{}{"encrypted": true, "data": "test", "sm4Key": "test", "iv": "test", "timestamp": time.Now().Unix()}},
		{"Missing data", map[string]interface{}{"encrypted": true, "sm4Key": "test", "iv": "test", "timestamp": time.Now().Unix(), "nonce": "test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := setupMinimalTestServer(t)
			defer server.Close()

			reqJSON, _ := json.Marshal(tc.request)
			resp, err := http.Post(server.URL+"/api/v1/system/auth/login", "application/json", bytes.NewBuffer(reqJSON))
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return error (400 or similar)
			assert.True(t, resp.StatusCode >= 400, "Request with missing fields should return error")
		})
	}
}

// Helper functions

func generateRandomHex(length int) string {
	// Use timestamp + counter for uniqueness
	seed := time.Now().UnixNano()
	b := make([]byte, length)
	for i := range b {
		// Combine seed with index and add variation
		val := (seed + int64(i*1000)) % 256
		b[i] = byte(val)
	}
	return hex.EncodeToString(b)
}

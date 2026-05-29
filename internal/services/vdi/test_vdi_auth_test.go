package vdi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockVDIServer creates a test server that simulates Sangfor VDI API responses
func mockVDIServer() *httptest.Server {
	mux := http.NewServeMux()

	// Standard Sangfor VDI auth endpoint
	mux.HandleFunc("/API/V1.0/Auth/Login", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if payload.Username == "admin" && payload.Password == "sangfor@2020" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code":    0,
				"error_message": "",
				"data": map[string]interface{}{
					"token": map[string]interface{}{
						"tenant_id":  0,
						"auth_token": "mock-token-abc123",
					},
				},
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code":    1001,
				"error_message": "Invalid credentials",
			})
		}
	})

	// Code-style auth endpoint
	mux.HandleFunc("/v1/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Auth struct {
				Name     string `json:"name"`
				Password string `json:"password"`
			} `json:"auth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if payload.Auth.Name == "admin" && payload.Auth.Password == "sangfor@2020" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code":    0,
				"error_message": "",
				"data": map[string]interface{}{
					"token": map[string]interface{}{
						"tenant_id":  0,
						"auth_token": "mock-token-xyz789",
					},
				},
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error_code":    1001,
				"error_message": "Invalid credentials",
			})
		}
	})

	return httptest.NewServer(mux)
}

func TestVDIAPIAuthentication(t *testing.T) {
	server := mockVDIServer()
	defer server.Close()

	t.Run("standard auth format success", func(t *testing.T) {
		payload := map[string]string{
			"username": "admin",
			"password": "sangfor@2020",
		}
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := server.Client().Post(server.URL+"/API/V1.0/Auth/Login", "application/json", bytes.NewReader(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, float64(0), result["error_code"])
	})

	t.Run("standard auth format wrong credentials", func(t *testing.T) {
		payload := map[string]string{
			"username": "admin",
			"password": "wrong",
		}
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := server.Client().Post(server.URL+"/API/V1.0/Auth/Login", "application/json", bytes.NewReader(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("code implementation format success", func(t *testing.T) {
		payload := map[string]interface{}{
			"auth": map[string]string{
				"name":     "admin",
				"password": "sangfor@2020",
			},
		}
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := server.Client().Post(server.URL+"/v1/auth/tokens", "application/json", bytes.NewReader(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			ErrorCode    int `json:"error_code"`
			Data         struct {
				Token struct {
					TenantID  int    `json:"tenant_id"`
					AuthToken string `json:"auth_token"`
				} `json:"token"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, 0, result.ErrorCode)
		assert.Equal(t, "mock-token-xyz789", result.Data.Token.AuthToken)
	})
}

func TestVDIAPIWithEncryptedPassword(t *testing.T) {
	server := mockVDIServer()
	defer server.Close()

	testPassword := "sangfor@2020"

	t.Run("encrypt decrypt roundtrip then auth", func(t *testing.T) {
		encrypted := encryptVDIPassword(testPassword)
		decrypted := decryptVDIPassword(encrypted)
		assert.Equal(t, testPassword, decrypted)

		// Use decrypted password with mock server
		payload := map[string]interface{}{
			"auth": map[string]string{
				"name":     "admin",
				"password": decrypted,
			},
		}
		jsonData, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := server.Client().Post(server.URL+"/v1/auth/tokens", "application/json", bytes.NewReader(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

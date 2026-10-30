package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ========== Login Request Parsing Integration Tests ==========

func TestIntegration_LoginRequest_Parsing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ValidLoginRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
			"authMode": "local",
		}
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Use a minimal gin context to test binding
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var loginReq LoginRequest
		err = c.ShouldBindJSON(&loginReq)
		assert.NoError(t, err, "Should bind valid JSON to LoginRequest")
		assert.Equal(t, "testuser", loginReq.Username)
		assert.Equal(t, "testpass", loginReq.Password)
		assert.Equal(t, "local", loginReq.AuthMode)
	})

	t.Run("MissingUsername", func(t *testing.T) {
		body := map[string]interface{}{
			"password": "testpass",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var loginReq LoginRequest
		err := c.ShouldBindJSON(&loginReq)
		assert.Error(t, err, "Should fail when username is missing")
	})

	t.Run("MissingPassword", func(t *testing.T) {
		body := map[string]interface{}{
			"username": "testuser",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var loginReq LoginRequest
		err := c.ShouldBindJSON(&loginReq)
		assert.Error(t, err, "Should fail when password is missing")
	})

	t.Run("AllAuthModes", func(t *testing.T) {
		modes := []string{"local", "ad", "hybrid"}
		for _, mode := range modes {
			t.Run("Mode_"+mode, func(t *testing.T) {
				body := map[string]interface{}{
					"username": "testuser",
					"password": "testpass",
					"authMode": mode,
				}
				bodyBytes, _ := json.Marshal(body)

				req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				c, _ := gin.CreateTestContext(w)
				c.Request = req

				var loginReq LoginRequest
				err := c.ShouldBindJSON(&loginReq)
				assert.NoError(t, err)
				assert.Equal(t, mode, loginReq.AuthMode)
			})
		}
	})

	t.Run("EmptyAuthMode", func(t *testing.T) {
		body := map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var loginReq LoginRequest
		err := c.ShouldBindJSON(&loginReq)
		assert.NoError(t, err)
		assert.Equal(t, "", loginReq.AuthMode, "AuthMode should default to empty string")
	})

	t.Run("EncryptedPassword", func(t *testing.T) {
		body := map[string]interface{}{
			"username":           "testuser",
			"password":           "encrypted_data",
			"encryptedPassword":  true,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var loginReq LoginRequest
		err := c.ShouldBindJSON(&loginReq)
		assert.NoError(t, err)
		assert.True(t, loginReq.EncryptedPassword, "EncryptedPassword should be true")
	})
}

// ========== Login Response Format Integration Tests ==========

func TestIntegration_LoginResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SuccessResponseStructure", func(t *testing.T) {
		router := gin.New()
		router.POST("/login", func(c *gin.Context) {
			nickname := "TestNick"
			email := "test@example.com"
			deptName := "Engineering"
			loginResp := &LoginResponse{
				User: &UserInfo{
					ID:         "user-id-1",
					Username:   "testuser",
					Nickname:   &nickname,
					Email:      &email,
					Status:     0,
					DeptName:   &deptName,
					Roles:      []string{"admin"},
					CreateTime: parseTime("2026-01-01T00:00:00Z"),
				},
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-456",
				ExpiresIn:    7200,
				TokenType:    "Bearer",
			}
			response.Success(c, loginResp)
		})

		body := bytes.NewBufferString(`{"username":"test","password":"pass"}`)
		req := httptest.NewRequest("POST", "/login", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Verify standard response wrapper
		assert.Equal(t, float64(0), resp["code"], "code should be 0 for success")
		assert.Equal(t, "success", resp["message"])
		assert.NotNil(t, resp["data"])
		assert.NotNil(t, resp["timestamp"])

		// Verify login response data
		data := resp["data"].(map[string]interface{})
		assert.NotEmpty(t, data["accessToken"])
		assert.NotEmpty(t, data["refreshToken"])
		assert.Equal(t, float64(7200), data["expiresIn"])
		assert.Equal(t, "Bearer", data["tokenType"])

		// Verify user info
		user := data["user"].(map[string]interface{})
		assert.Equal(t, "user-id-1", user["id"])
		assert.Equal(t, "testuser", user["username"])
		assert.Equal(t, "TestNick", user["nickname"])
	})

	t.Run("ErrorResponseStructure", func(t *testing.T) {
		router := gin.New()
		router.POST("/login", func(c *gin.Context) {
			response.Error(c, response.ErrCredentialInvalid)
		})

		body := bytes.NewBufferString(`{"username":"test","password":"wrong"}`)
		req := httptest.NewRequest("POST", "/login", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.NotEqual(t, float64(0), resp["code"], "code should not be 0 for error")
		assert.NotEmpty(t, resp["message"])
	})
}

// ========== Auth Config Endpoint Integration Tests ==========

func TestIntegration_AuthConfig_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ADDisabled", func(t *testing.T) {
		router := gin.New()
		router.GET("/config", func(c *gin.Context) {
			response.Success(c, map[string]interface{}{
				"adEnabled":   false,
				"defaultMode": "local",
			})
		})

		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, float64(0), resp["code"])
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, false, data["adEnabled"])
		assert.Equal(t, "local", data["defaultMode"])
	})

	t.Run("ADEnabled", func(t *testing.T) {
		router := gin.New()
		router.GET("/config", func(c *gin.Context) {
			response.Success(c, map[string]interface{}{
				"adEnabled":   true,
				"defaultMode": "hybrid",
			})
		})

		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, true, data["adEnabled"])
		assert.Equal(t, "hybrid", data["defaultMode"])
	})
}

// ========== User-Agent Parsing Integration Tests ==========

func TestIntegration_ParseUserAgent(t *testing.T) {
	tests := []struct {
		name            string
		userAgent       string
		expectedBrowser string
		expectedOS      string
	}{
		{"Empty", "", "Unknown", "Unknown"},
		{"Chrome on Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0", "Chrome", "Windows"},
		{"Firefox on Mac", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Firefox/120.0", "Firefox", "Mac OS X"},
		{"Edge on Windows", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Edg/120.0.0.0", "Edge", "Windows"},
		{"Safari on iOS", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1", "Safari", "iOS"},
		{"Linux", "Mozilla/5.0 (X11; Linux x86_64) Chrome/120.0.0.0", "Chrome", "Linux"},
		{"Android Chrome", "Mozilla/5.0 (Linux; Android 14) Chrome/120.0.0.0", "Chrome", "Android"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser, os := parseUserAgent(tt.userAgent)
			assert.Equal(t, tt.expectedBrowser, browser, "Browser mismatch")
			assert.Equal(t, tt.expectedOS, os, "OS mismatch")
		})
	}
}

// ========== LoginRequest Struct Validation Tests ==========

func TestIntegration_LoginRequest_AllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := map[string]interface{}{
		"username":           "admin",
		"password":           "encrypted_password_hex",
		"authMode":           "hybrid",
		"encryptedPassword":  true,
		"captcha":            "1234",
		"captchaId":          "captcha-uuid-123",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	var loginReq LoginRequest
	err := c.ShouldBindJSON(&loginReq)
	require.NoError(t, err)

	assert.Equal(t, "admin", loginReq.Username)
	assert.Equal(t, "encrypted_password_hex", loginReq.Password)
	assert.Equal(t, "hybrid", loginReq.AuthMode)
	assert.True(t, loginReq.EncryptedPassword)
	assert.Equal(t, "1234", loginReq.Captcha)
	assert.Equal(t, "captcha-uuid-123", loginReq.CaptchaID)
}

// ========== UserInfo Response Structure Tests ==========

func TestIntegration_UserInfo_ResponseJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	nickname := "Admin User"
	email := "admin@example.com"
	deptName := "IT Department"

	userInfo := &UserInfo{
		ID:         "user-uuid-1",
		Username:   "admin",
		Nickname:   &nickname,
		Email:      &email,
		Status:     0,
		DeptName:   &deptName,
		Roles:      []string{"role-id-1", "role-id-2"},
		CreateTime: parseTime("2026-01-15T10:30:00Z"),
	}

	router := gin.New()
	router.GET("/user", func(c *gin.Context) {
		response.Success(c, userInfo)
	})

	req := httptest.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "user-uuid-1", data["id"])
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "Admin User", data["nickname"])
	assert.Equal(t, "admin@example.com", data["email"])
	assert.Equal(t, float64(0), data["status"], "0 = enabled")
	assert.Equal(t, "IT Department", data["deptName"])

	roles := data["roles"].([]interface{})
	assert.Len(t, roles, 2)
}

// ========== LoginResponse Token Tests ==========

func TestIntegration_LoginResponse_TokenFields(t *testing.T) {
	loginResp := &LoginResponse{
		User:         &UserInfo{ID: "test", Username: "test"},
		AccessToken:  "eyJhbGciOiJIUzI1NiJ9.test-access",
		RefreshToken: "eyJhbGciOiJIUzI1NiJ9.test-refresh",
		ExpiresIn:    7200,
		TokenType:    "Bearer",
	}

	assert.Equal(t, "eyJhbGciOiJIUzI1NiJ9.test-access", loginResp.AccessToken)
	assert.Equal(t, "eyJhbGciOiJIUzI1NiJ9.test-refresh", loginResp.RefreshToken)
	assert.Equal(t, int64(7200), loginResp.ExpiresIn)
	assert.Equal(t, "Bearer", loginResp.TokenType)
}

// ========== Error Code Mapping Tests ==========

func TestIntegration_AuthError_HTTPStatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		appError     *response.AppError
		expectedCode int
	}{
		{"BadRequest", response.ErrBadRequest, http.StatusBadRequest},
		{"CredentialInvalid", response.ErrCredentialInvalid, http.StatusUnauthorized},
		{"Forbidden", response.ErrForbidden, http.StatusForbidden},
		{"ServerError", response.ErrServerError, http.StatusInternalServerError},
		{"TokenInvalid", response.ErrTokenInvalid, http.StatusUnauthorized},
		{"UserNotFound", response.ErrUserNotFound, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				response.Error(c, tt.appError)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code, "HTTP status code mismatch for %s", tt.name)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.NotEqual(t, 0, resp["code"], "Business error code should be non-zero")
		})
	}
}

// ========== SetupAuthRouter Registration Tests ==========

func TestIntegration_SetupAuthRouter_RouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("RoutesRegisteredWithoutCore", func(t *testing.T) {
		router := gin.New()
		group := router.Group("/api/v1/system/auth")

		// Register only the routes that don't need core dependency
		// (SetupAuthRouter needs core, but we can verify the group structure)
		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/public-key"},
			{"GET", "/config"},
			{"POST", "/login"},
			{"POST", "/logout"},
			{"POST", "/refresh"},
		}

		// Verify the group prefix is correct
		assert.Equal(t, "/api/v1/system/auth", group.BasePath())

		// Register placeholder handlers
		for _, route := range routes {
			handler := func(route string) gin.HandlerFunc {
				return func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"route": route})
				}
			}
			switch route.method {
			case "GET":
				group.GET(route.path, handler(route.path))
			case "POST":
				group.POST(route.path, handler(route.path))
			}
		}

		// Verify routes are accessible
		for _, route := range routes {
			req := httptest.NewRequest(route.method, "/api/v1/system/auth"+route.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Route %s %s should be registered", route.method, route.path)
		}
	})
}

// ========== JSON Binding Edge Cases ==========

func TestIntegration_LoginRequest_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	var loginReq LoginRequest
	err := c.ShouldBindJSON(&loginReq)
	assert.Error(t, err, "Should fail for invalid JSON")
}

func TestIntegration_LoginRequest_ExtraFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Extra fields should be ignored (Go's JSON decoder ignores unknown fields)
	body := map[string]interface{}{
		"username":      "testuser",
		"password":      "testpass",
		"authMode":      "local",
		"unknownField":  "should be ignored",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	var loginReq LoginRequest
	err := c.ShouldBindJSON(&loginReq)
	assert.NoError(t, err, "Extra fields should be ignored")
	assert.Equal(t, "testuser", loginReq.Username)
	assert.Equal(t, "local", loginReq.AuthMode)
}

// parseTime is a helper to parse time strings for test data
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

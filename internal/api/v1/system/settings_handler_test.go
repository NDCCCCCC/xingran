package system

// =====================================================================
// Method Enumeration (Plan 72-11 Task 1)
//
// settings_handler.go (91 lines) — method coverage:
//   - GetUserPreferences  GET /system/settings/preferences
//   - UpdateUserPreferences PUT /system/settings/preferences
// =====================================================================

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupSettingsTestDB creates in-memory SQLite with sys_user_preference schema.
func setupSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_preference (
			id TEXT PRIMARY KEY,
			user_id TEXT UNIQUE,
			theme TEXT,
			theme_style TEXT,
			layout_type TEXT,
			layout_density TEXT,
			sidebar_width INTEGER,
			sidebar_collapsed_width INTEGER,
			sidebar_collapsed INTEGER,
			page_size INTEGER,
			custom_primary_color TEXT,
			custom_sidebar_color TEXT,
			language TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	return db
}

// newSettingsTestHandler wires a real SettingsService into the handler.
func newSettingsTestHandler(t *testing.T, db *gorm.DB) (*SettingsHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewSettingsService(db)
	h := NewSettingsHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// invokeAuthHandler is a variant of invokeHandler that injects user_id into context.
func invokeAuthHandler(t *testing.T, method, path string, body interface{}, params gin.Params,
	handler func(*gin.Context), userID string) *httpRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if body != nil {
		var raw []byte
		if s, ok := body.(string); ok {
			raw = []byte(s)
		} else {
			raw, _ = json.Marshal(body)
		}
		req = httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req
	if params != nil {
		c.Params = params
	}
	if userID != "" {
		c.Set("user_id", userID)
	}
	if handler != nil {
		handler(c)
	}
	return w
}

// httpRecorder is an alias for httptest.ResponseRecorder.
type httpRecorder = httptest.ResponseRecorder

// TC1: GetUserPreferences - returns defaults when no record
func TestSettingsHandler_GetUserPreferences_Defaults(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "light", resp.Data["theme"])
}

// TC2: GetUserPreferences - unauthorized when no user_id
func TestSettingsHandler_GetUserPreferences_Unauthorized(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC3: GetUserPreferences - service error bubbles up
func TestSettingsHandler_GetUserPreferences_ServiceError(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	w := invokeAuthHandler(t, "GET", "/system/settings/preferences", nil, nil, h.GetUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC4: UpdateUserPreferences - success
func TestSettingsHandler_UpdateUserPreferences_Success(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := systemServices.UserPreferences{
		Theme:        "dark",
		ThemeStyle:   "minimal",
		LayoutType:   "classic",
		PageSize:     20,
		Language:     "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	// Verify persisted
	var theme string
	require.NoError(t, db.Raw("SELECT theme FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&theme).Error)
	assert.Equal(t, "dark", theme)
}

// TC5: UpdateUserPreferences - unauthorized
func TestSettingsHandler_UpdateUserPreferences_Unauthorized(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC6: UpdateUserPreferences - bad JSON returns 400
func TestSettingsHandler_UpdateUserPreferences_BadJSON(t *testing.T) {
	h, _ := newSettingsTestHandler(t, setupSettingsTestDB(t))

	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", "{not-json", nil, h.UpdateUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC7: UpdateUserPreferences - updates existing record
func TestSettingsHandler_UpdateUserPreferences_UpdateExisting(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))

	// Seed existing record
	require.NoError(t, db.Exec(`INSERT INTO sys_user_preference (id, user_id, theme, page_size, language, sidebar_width, sidebar_collapsed_width, sidebar_collapsed)
		VALUES (?, 'user-1', 'light', 10, 'zh-CN', 280, 64, 0)`, "id-1").Error)

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 50,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var pageSize int
	require.NoError(t, db.Raw("SELECT page_size FROM sys_user_preference WHERE user_id = ?", "user-1").Scan(&pageSize).Error)
	assert.Equal(t, 50, pageSize)
}

// TC8: UpdateUserPreferences - service error
func TestSettingsHandler_UpdateUserPreferences_ServiceError(t *testing.T) {
	h, db := newSettingsTestHandler(t, setupSettingsTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_user_preference").Error)

	body := systemServices.UserPreferences{
		Theme:    "dark",
		PageSize: 20,
		Language: "en-US",
	}
	w := invokeAuthHandler(t, "PUT", "/system/settings/preferences", body, nil, h.UpdateUserPreferences, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC9: WithCore - nil receiver returns nil
func TestSettingsHandler_WithCore_NilReceiver(t *testing.T) {
	var h *SettingsHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}
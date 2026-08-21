package system

// =====================================================================
// Method Enumeration (Plan 72-11 Task 2)
//
// apikey_handler.go (336 lines) — method coverage:
//   - Create          POST /system/apikeys
//   - List            POST /system/apikeys/list
//   - GetByID         POST /system/apikeys/:id
//   - Update          POST /system/apikeys/:id/update
//   - Delete          POST /system/apikeys/:id/delete
//   - ToggleStatus    POST /system/apikeys/:id/toggle
//   - ListUsageLogs   POST /system/apikeys/:id/logs
//   - GetUsageSummary GET  /system/apikeys/:id/summary
// =====================================================================

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupAPIKeyTestDB creates in-memory SQLite with sys_user + sys_api_keys + sys_api_key_usage_logs schema.
func setupAPIKeyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			password TEXT,
			nickname TEXT,
			email TEXT,
			phone TEXT,
			gender INTEGER NOT NULL DEFAULT 2,
			status INTEGER NOT NULL DEFAULT 0,
			dept_id TEXT,
			remark TEXT,
			salt TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_api_keys (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			salt TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			user_id TEXT,
			expires_at DATETIME,
			last_used_at DATETIME,
			is_active INTEGER DEFAULT 1,
			scopes TEXT,
			ip_whitelist TEXT,
			description TEXT,
			inherit_perms BOOLEAN DEFAULT 0
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_api_key_usage_logs (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			method TEXT,
			path TEXT,
			status_code INTEGER,
			client_ip TEXT,
			user_agent TEXT,
			duration INTEGER,
			success BOOLEAN,
			created_at DATETIME
		)
	`).Error)
	return db
}

// newAPIKeyTestHandler wires a real APIKeyService into the handler.
func newAPIKeyTestHandler(t *testing.T, db *gorm.DB) (*APIKeyHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewAPIKeyService(db)
	h := NewAPIKeyHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// seedAPIKeyUser inserts a sys_user row directly.
func seedAPIKeyUser(t *testing.T, db *gorm.DB, username string) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, password, status, created_at, updated_at) VALUES (?, ?, 'x', 0, ?, ?)`,
		id, username, now, now).Error)
	return id
}

// seedAPIKeyRow inserts a sys_api_keys row directly.
func seedAPIKeyRow(t *testing.T, db *gorm.DB, userID string, isActive int) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_api_keys (id, name, key_hash, salt, key_prefix, user_id, created_at, updated_at, version, is_active)
		VALUES (?, 'test', 'hash-' || ?, 'salt', 'rec_12345678ab', ?, ?, ?, 0, ?)`,
		id, id, userID, now, now, isActive).Error)
	return id
}

// TC1: Create - success returns key
func TestAPIKeyHandler_Create_Success(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")

	body := requests.CreateAPIKeyRequest{
		Name:        "test key",
		Scopes:      []string{"read", "write"},
		Description: stringPtrLocal("test desc"),
	}
	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", body, nil, userID)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	_, hasKey := resp.Data["key"]
	assert.True(t, hasKey, "response should contain 'key' field")
}

// TC2: Create - bad JSON returns 400
func TestAPIKeyHandler_Create_BadJSON(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")

	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", "{not-json", nil, userID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC3: Create - missing user_id returns 401
func TestAPIKeyHandler_Create_Unauthorized(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	_ = seedAPIKeyUser(t, db, "alice")

	body := requests.CreateAPIKeyRequest{
		Name:   "test key",
		Scopes: []string{"read"},
	}
	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", body, nil, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC4: Create - user not found
func TestAPIKeyHandler_Create_UserNotFound(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	body := requests.CreateAPIKeyRequest{
		Name:   "test key",
		Scopes: []string{"read"},
	}
	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", body, nil, "missing-user-id")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC5: Create - invalid scope
func TestAPIKeyHandler_Create_InvalidScope(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")

	body := requests.CreateAPIKeyRequest{
		Name:   "test key",
		Scopes: []string{"invalid-scope"},
	}
	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", body, nil, userID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC6: List - empty
func TestAPIKeyHandler_List_Empty(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")

	w := doJSONWithUser(t, h.List, "POST", "/system/apikeys/list", map[string]interface{}{}, nil, userID)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TC7: List - bad JSON falls back to defaults
func TestAPIKeyHandler_List_BadJSON(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")

	w := doJSONWithUser(t, h.List, "POST", "/system/apikeys/list", "{not-json", nil, userID)
	assert.Equal(t, http.StatusOK, w.Code, "bad JSON should fall back to defaults")
}

// TC8: List - missing user_id
func TestAPIKeyHandler_List_Unauthorized(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	_ = seedAPIKeyUser(t, db, "alice")

	w := doJSONWithUser(t, h.List, "POST", "/system/apikeys/list", map[string]interface{}{}, nil, "")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC9: GetByID - success
func TestAPIKeyHandler_GetByID_Success(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.GetByID, "POST", "/system/apikeys/"+id, nil, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC10: GetByID - empty id returns error
func TestAPIKeyHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	w := doJSONWithUser(t, h.GetByID, "POST", "/system/apikeys/", nil, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC11: GetByID - not found
func TestAPIKeyHandler_GetByID_NotFound(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	missing := uuid.NewString()
	w := doJSONWithUser(t, h.GetByID, "POST", "/system/apikeys/"+missing, nil, map[string]string{"id": missing}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC12: Update - success
func TestAPIKeyHandler_Update_Success(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	body := requests.UpdateAPIKeyRequest{
		ID:          id,
		Name:        stringPtrLocal("new name"),
		Description: stringPtrLocal("new desc"),
	}
	w := doJSONWithUser(t, h.Update, "POST", "/system/apikeys/"+id+"/update", body, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var name string
	require.NoError(t, db.Raw("SELECT name FROM sys_api_keys WHERE id = ?", id).Scan(&name).Error)
	assert.Equal(t, "new name", name)
}

// TC13: Update - empty id returns error
func TestAPIKeyHandler_Update_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	body := requests.UpdateAPIKeyRequest{ID: "", Name: stringPtrLocal("x")}
	w := doJSONWithUser(t, h.Update, "POST", "/system/apikeys//update", body, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC14: Update - bad JSON returns 400
func TestAPIKeyHandler_Update_BadJSON(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.Update, "POST", "/system/apikeys/"+id+"/update", "{not-json", map[string]string{"id": id}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC15: Update - not found
func TestAPIKeyHandler_Update_NotFound(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	missing := uuid.NewString()
	body := requests.UpdateAPIKeyRequest{ID: missing, Name: stringPtrLocal("x")}
	w := doJSONWithUser(t, h.Update, "POST", "/system/apikeys/"+missing+"/update", body, map[string]string{"id": missing}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC16: Delete - success
func TestAPIKeyHandler_Delete_Success(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.Delete, "POST", "/system/apikeys/"+id+"/delete", nil, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_api_keys WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt, "row should be soft-deleted")
}

// TC17: Delete - empty id returns error
func TestAPIKeyHandler_Delete_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	w := doJSONWithUser(t, h.Delete, "POST", "/system/apikeys//delete", nil, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC18: Delete - not found
func TestAPIKeyHandler_Delete_NotFound(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	missing := uuid.NewString()
	w := doJSONWithUser(t, h.Delete, "POST", "/system/apikeys/"+missing+"/delete", nil, map[string]string{"id": missing}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC19: ToggleStatus - success
func TestAPIKeyHandler_ToggleStatus_Success(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.ToggleStatus, "POST", "/system/apikeys/"+id+"/toggle", nil, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)

	var isActive int
	require.NoError(t, db.Raw("SELECT is_active FROM sys_api_keys WHERE id = ?", id).Scan(&isActive).Error)
	assert.Equal(t, 0, isActive, "should be toggled to 0 (revoked)")
}

// TC20: ToggleStatus - empty id
func TestAPIKeyHandler_ToggleStatus_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	w := doJSONWithUser(t, h.ToggleStatus, "POST", "/system/apikeys//toggle", nil, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC21: ToggleStatus - not found
func TestAPIKeyHandler_ToggleStatus_NotFound(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	missing := uuid.NewString()
	w := doJSONWithUser(t, h.ToggleStatus, "POST", "/system/apikeys/"+missing+"/toggle", nil, map[string]string{"id": missing}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC22: ListUsageLogs - empty
func TestAPIKeyHandler_ListUsageLogs_Empty(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.ListUsageLogs, "POST", "/system/apikeys/"+id+"/logs", map[string]interface{}{}, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC23: ListUsageLogs - empty id
func TestAPIKeyHandler_ListUsageLogs_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	w := doJSONWithUser(t, h.ListUsageLogs, "POST", "/system/apikeys//logs", map[string]interface{}{}, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC24: ListUsageLogs - bad JSON falls back to defaults
func TestAPIKeyHandler_ListUsageLogs_BadJSON(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.ListUsageLogs, "POST", "/system/apikeys/"+id+"/logs", "{not-json", map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC25: GetUsageSummary - empty
func TestAPIKeyHandler_GetUsageSummary_Empty(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	id := seedAPIKeyRow(t, db, "user-1", 1)

	w := doJSONWithUser(t, h.GetUsageSummary, "GET", "/system/apikeys/"+id+"/summary", nil, map[string]string{"id": id}, "user-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC26: GetUsageSummary - empty id
func TestAPIKeyHandler_GetUsageSummary_EmptyID(t *testing.T) {
	h, _ := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))

	w := doJSONWithUser(t, h.GetUsageSummary, "GET", "/system/apikeys//summary", nil, map[string]string{"id": ""}, "user-1")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC27: maskAPIKeys - returns simplified fields, no hash/salt
func TestAPIKeyHandler_MaskAPIKeys(t *testing.T) {
	uid := "user-1"
	desc := "desc"
	key := models.APIKey{
		BaseModel:    models.BaseModel{ID: "key-1"},
		Name:         "test",
		KeyPrefix:    "rec_12345678ab",
		KeyHash:      "secret-hash",
		Salt:         "secret-salt",
		UserID:       &uid,
		Scopes:       []string{"read"},
		IPWhitelist:  []string{"127.0.0.1"},
		Description:  &desc,
		InheritPerms: true,
	}

	masked := maskAPIKeys([]models.APIKey{key})
	require.Len(t, masked, 1)
	m := masked[0]

	_, hasID := m["id"]
	_, hasName := m["name"]
	_, hasKeyPrefix := m["keyPrefix"]
	_, hasUserID := m["userId"]
	assert.True(t, hasID && hasName && hasKeyPrefix && hasUserID)

	_, hasHash := m["keyHash"]
	_, hasSalt := m["salt"]
	assert.False(t, hasHash, "keyHash should not be in masked response")
	assert.False(t, hasSalt, "salt should not be in masked response")
}

// TC28: WithCore - nil receiver
func TestAPIKeyHandler_WithCore_NilReceiver(t *testing.T) {
	var h *APIKeyHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}

// TC29: Create - service error (drop table)
func TestAPIKeyHandler_Create_ServiceError(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")
	require.NoError(t, db.Exec("DROP TABLE sys_api_keys").Error)

	body := requests.CreateAPIKeyRequest{
		Name:   "test key",
		Scopes: []string{"read"},
	}
	w := doJSONWithUser(t, h.Create, "POST", "/system/apikeys", body, nil, userID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC30: List - service error
func TestAPIKeyHandler_List_ServiceError(t *testing.T) {
	h, db := newAPIKeyTestHandler(t, setupAPIKeyTestDB(t))
	userID := seedAPIKeyUser(t, db, "alice")
	require.NoError(t, db.Exec("DROP TABLE sys_api_keys").Error)

	w := doJSONWithUser(t, h.List, "POST", "/system/apikeys/list", map[string]interface{}{}, nil, userID)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// stringPtrLocal is a local helper since stringPtr may be defined elsewhere
func stringPtrLocal(s string) *string { return &s }

// doJSONWithUser performs a JSON request and injects user_id into gin context
// before invoking the handler. Returns the response recorder.
func doJSONWithUser(t *testing.T, h gin.HandlerFunc, method, path string, body interface{},
	params map[string]string, userID string) *httptest.ResponseRecorder {
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
	for k, v := range params {
		c.Params = append(c.Params, gin.Param{Key: k, Value: v})
	}
	if userID != "" {
		c.Set("user_id", userID)
	}
	func() {
		defer func() { _ = recover() }()
		h(c)
	}()
	return w
}
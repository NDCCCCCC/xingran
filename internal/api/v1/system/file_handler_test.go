package system

// =====================================================================
// Method Enumeration (Plan 72-11 Task 3)
//
// file_handler.go (217 lines) — method coverage:
//   - Upload         POST /system/files/upload (multipart)
//   - GetByID        GET  /system/files/:id
//   - Delete         DELETE /system/files/:id
//   - List           GET  /system/files
//   - BatchDelete    POST /system/files/batch
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupFileTestDB creates in-memory SQLite with sys_files schema (file_service schema).
func setupFileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_files (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			file_name TEXT,
			file_size INTEGER,
			file_type TEXT,
			extension TEXT,
			storage_path TEXT,
			file_hash TEXT,
			uploader_id TEXT,
			business_type TEXT,
			is_deleted BOOLEAN,
			delete_time DATETIME,
			file_width INTEGER,
			file_height INTEGER,
			metadata TEXT
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_file_access_logs (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			file_id TEXT,
			action_type TEXT,
			user_id TEXT,
			user_name TEXT,
			ip_address TEXT,
			user_agent TEXT
		)
	`).Error)
	return db
}

// newFileTestHandler wires a real FileService into the handler.
func newFileTestHandler(t *testing.T, db *gorm.DB) (*FileHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewFileService(db)
	h := NewFileHandler(svc)
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// seedFileRow inserts a sys_files row directly.
func seedFileRow(t *testing.T, db *gorm.DB, userID string) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_files (id, file_name, file_size, file_type, extension, storage_path, uploader_id, business_type, is_deleted, created_at, updated_at, version)
		VALUES (?, 'test.png', 1024, 'image/png', '.png', '/uploads/test.png', ?, 'avatar', 0, ?, ?, 0)`,
		id, userID, now, now).Error)
	return id
}

// TC1: Upload - unauthorized (no user_id in context)
func TestFileHandler_Upload_Unauthorized(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	// doJSON helper doesn't inject user_id, so Upload returns unauthorized
	w := doJSON(t, h.Upload, "POST", "/system/files/upload", map[string]string{"category": "avatar"}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC2: GetByID - success
func TestFileHandler_GetByID_Success(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	id := seedFileRow(t, db, "user-1")

	w := doJSON(t, h.GetByID, "POST", "/system/files/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC3: GetByID - empty id
func TestFileHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	w := doJSON(t, h.GetByID, "POST", "/system/files/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC4: GetByID - not found
func TestFileHandler_GetByID_NotFound(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/files/"+missing, nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC5: Delete - empty id
func TestFileHandler_Delete_EmptyID(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	w := doJSON(t, h.Delete, "POST", "/system/files/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC6: Delete - not found
func TestFileHandler_Delete_NotFound(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/files/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC7: List - with business type filter
func TestFileHandler_List_WithFilter(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	seedFileRow(t, db, "user-1")
	seedFileRow(t, db, "user-2")

	q := url.Values{}
	q.Set("page", "1")
	q.Set("pageSize", "10")
	q.Set("businessType", "avatar")

	w := doJSON(t, h.List, "GET", "/system/files?"+q.Encode(), nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TC8: List - with user id filter
func TestFileHandler_List_WithUserFilter(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	seedFileRow(t, db, "user-1")
	seedFileRow(t, db, "user-2")

	q := url.Values{}
	q.Set("page", "1")
	q.Set("pageSize", "10")
	q.Set("userId", "user-1")

	w := doJSON(t, h.List, "GET", "/system/files?"+q.Encode(), nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC9: List - bad query
func TestFileHandler_List_BadQuery(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	q := url.Values{}
	q.Set("page", "invalid")

	w := doJSON(t, h.List, "GET", "/system/files?"+q.Encode(), nil, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC10: BatchDelete - empty ids fails
func TestFileHandler_BatchDelete_Empty(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	w := doJSON(t, h.BatchDelete, "POST", "/system/files/batch",
		map[string]interface{}{"ids": []string{}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC11: BatchDelete - bad JSON
func TestFileHandler_BatchDelete_BadJSON(t *testing.T) {
	h, _ := newFileTestHandler(t, setupFileTestDB(t))

	w := doJSON(t, h.BatchDelete, "POST", "/system/files/batch", "{not-json", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC12: WithCore - nil receiver
func TestFileHandler_WithCore_NilReceiver(t *testing.T) {
	var h *FileHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}

// TC13: GetByID - service error
func TestFileHandler_GetByID_ServiceError(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	id := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/files/"+id, nil, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC14: Delete - service error
func TestFileHandler_Delete_ServiceError(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	id := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/files/"+id+"/delete", nil, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC15: List - service error
func TestFileHandler_List_ServiceError(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	w := doJSON(t, h.List, "GET", "/system/files", nil, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC16: BatchDelete - service error
func TestFileHandler_BatchDelete_ServiceError(t *testing.T) {
	h, db := newFileTestHandler(t, setupFileTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	w := doJSON(t, h.BatchDelete, "POST", "/system/files/batch",
		map[string]interface{}{"ids": []string{uuid.NewString()}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
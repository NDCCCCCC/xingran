package vdi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupBaseHandlerDB creates an in-memory sqlite with sys_vdi_server table.
func setupBaseHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.Exec(`
		CREATE TABLE IF NOT EXISTS sys_vdi_server (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			deleted_at DATETIME
		)
	`).Error)
	return gdb
}

// ==================== handleJSONBinding ====================

func TestHandleJSONBinding_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	req := httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	var out map[string]interface{}
	ok := handleJSONBinding(c, &out)
	assert.True(t, ok)
	assert.Equal(t, "x", out["name"])
}

func TestHandleJSONBinding_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	var out map[string]interface{}
	ok := handleJSONBinding(c, &out)
	assert.False(t, ok)
}

// ==================== handleServiceError ====================

func TestHandleServiceError_Nil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", nil)
	c.Request = req

	ok := handleServiceError(c, nil, "test")
	assert.True(t, ok)
}

func TestHandleServiceError_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/x", nil)
	c.Request = req

	ok := handleServiceError(c, errors.New("boom"), "test")
	assert.False(t, ok)
	// Should write error response with non-zero code
	var resp map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, int(resp["code"].(float64)))
}

// ==================== verifyVDIServerExists ====================

func TestVerifyVDIServerExists_Present(t *testing.T) {
	db := setupBaseHandlerDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name) VALUES ('v-1', 'server-1')`).Error)

	server := verifyVDIServerExists(db, "v-1")
	assert.NotNil(t, server)
	assert.Equal(t, "v-1", server.ID)
}

func TestVerifyVDIServerExists_Missing(t *testing.T) {
	db := setupBaseHandlerDB(t)
	server := verifyVDIServerExists(db, "missing")
	assert.Nil(t, server)
}

// ==================== ensureVDIServer ====================

func TestEnsureVDIServer_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", nil)
	c.Request = req

	ok := ensureVDIServer(c, nil, "")
	assert.False(t, ok)
	// Empty ID returns 400 (BadRequest) via response.Error
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnsureVDIServer_ServerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", nil)
	c.Request = req

	db := setupBaseHandlerDB(t)
	ok := ensureVDIServer(c, db, "missing")
	assert.False(t, ok)
	// response.Error with int arg defaults to 400 (pkg/response toAppError int case)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnsureVDIServer_ServerExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/", nil)
	c.Request = req

	db := setupBaseHandlerDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_server (id, name) VALUES ('v-1', 'server-1')`).Error)
	ok := ensureVDIServer(c, db, "v-1")
	assert.True(t, ok)
}

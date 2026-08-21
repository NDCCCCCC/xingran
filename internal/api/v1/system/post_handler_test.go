package system

// =====================================================================
// post_handler_test.go — covers PostHandler (295 lines)
// Per Plan 72-09 Task 2
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupPostTestDB creates in-memory SQLite with sys_post schema.
func setupPostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_post (
			id TEXT PRIMARY KEY,
			post_code TEXT NOT NULL,
			post_name TEXT NOT NULL,
			post_sort INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	return db
}

func setupPostHandler(t *testing.T) (*PostHandler, *gorm.DB) {
	t.Helper()
	db := setupPostTestDB(t)
	svc := systemServices.NewPostService(db)
	return NewPostHandler(svc), db
}

func seedPost(t *testing.T, db *gorm.DB, code, name string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_post
		(id, post_code, post_name, post_sort, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, ?, datetime('now'), datetime('now'), 0)`,
		id, code, name, status).Error)
	return id
}

// TC1: Statistics - returns counts
func TestPostHandler_Statistics(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "a1", "P1", 0)
	seedPost(t, db, "a2", "P2", 0)
	seedPost(t, db, "i1", "P3", 1)

	w := doJSON(t, h.Statistics, "POST", "/system/posts/statistics", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total    int64 `json:"total"`
			Active   int64 `json:"active"`
			Inactive int64 `json:"inactive"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(3), resp.Data.Total)
	assert.Equal(t, int64(2), resp.Data.Active)
	assert.Equal(t, int64(1), resp.Data.Inactive)
}

// TC2: List - empty
func TestPostHandler_List_Empty(t *testing.T) {
	h, _ := setupPostHandler(t)
	w := doJSON(t, h.List, "POST", "/system/posts/list", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC3: List - filter by postCode
func TestPostHandler_List_FilterByPostCode(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "ceo", "CEO", 0)
	seedPost(t, db, "cto", "CTO", 0)

	w := doJSON(t, h.List, "POST", "/system/posts/list",
		map[string]interface{}{"postCode": "ceo", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC4: List - filter by postName
func TestPostHandler_List_FilterByPostName(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "c1", "董事长", 0)
	seedPost(t, db, "c2", "总经理", 0)

	w := doJSON(t, h.List, "POST", "/system/posts/list",
		map[string]interface{}{"postName": "董事", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC5: List - filter by status
func TestPostHandler_List_FilterByStatus(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "p1", "P1", 0)
	seedPost(t, db, "p2", "P2", 1)

	w := doJSON(t, h.List, "POST", "/system/posts/list",
		map[string]interface{}{"status": 0, "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC6: GetByID - success
func TestPostHandler_GetByID_Success(t *testing.T) {
	h, db := setupPostHandler(t)
	id := seedPost(t, db, "p1", "P1", 0)

	w := doJSON(t, h.GetByID, "POST", "/system/posts/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int           `json:"code"`
		Data models.Post   `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "P1", resp.Data.PostName)
}

// TC7: GetByID - not found
func TestPostHandler_GetByID_NotFound(t *testing.T) {
	h, _ := setupPostHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/posts/"+missing, nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC8: GetByID - empty id
func TestPostHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := setupPostHandler(t)
	w := doJSON(t, h.GetByID, "POST", "/system/posts/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC9: Create - success
func TestPostHandler_Create_Success(t *testing.T) {
	h, db := setupPostHandler(t)
	body := requests.PostCreateRequest{
		PostCode: "p1", PostName: "P1", PostSort: 1,
		Status: models.PostStatusEnabled,
	}
	_ = doJSON(t, h.Create, "POST", "/system/posts", body, nil)

	var got models.Post
	require.NoError(t, db.Where("post_code = ?", "p1").First(&got).Error)
	assert.Equal(t, "P1", got.PostName)
}

// TC10: Create - duplicate post_code fails
func TestPostHandler_Create_DuplicateCode(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "dup", "P1", 0)

	body := requests.PostCreateRequest{
		PostCode: "dup", PostName: "P2", PostSort: 1,
		Status: models.PostStatusEnabled,
	}
	w := doJSON(t, h.Create, "POST", "/system/posts", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC11: Create - missing fields
func TestPostHandler_Create_MissingFields(t *testing.T) {
	h, _ := setupPostHandler(t)
	w := doJSON(t, h.Create, "POST", "/system/posts", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC12: Update - success
func TestPostHandler_Update_Success(t *testing.T) {
	h, db := setupPostHandler(t)
	id := seedPost(t, db, "p1", "Old", 0)

	body := requests.PostUpdateRequest{
		PostName: "New", PostSort: 5, Status: models.PostStatusDisabled,
	}
	_ = doJSON(t, h.Update, "POST", "/system/posts/"+id+"/update", body, map[string]string{"id": id})

	var got models.Post
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.PostName)
	assert.Equal(t, 5, got.PostSort)
	assert.Equal(t, models.PostStatusDisabled, got.Status)
}

// TC13: Update - not found
func TestPostHandler_Update_NotFound(t *testing.T) {
	h, _ := setupPostHandler(t)
	missing := uuid.NewString()
	body := requests.PostUpdateRequest{PostName: "X", PostSort: 1, Status: models.PostStatusEnabled}
	w := doJSON(t, h.Update, "POST", "/system/posts/"+missing+"/update", body, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC14: Update - empty id
func TestPostHandler_Update_EmptyID(t *testing.T) {
	h, _ := setupPostHandler(t)
	body := requests.PostUpdateRequest{PostName: "X", PostSort: 1, Status: models.PostStatusEnabled}
	w := doJSON(t, h.Update, "POST", "/system/posts//update", body, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC15: Delete - success
func TestPostHandler_Delete_Success(t *testing.T) {
	h, db := setupPostHandler(t)
	id := seedPost(t, db, "p1", "P1", 0)

	_ = doJSON(t, h.Delete, "POST", "/system/posts/"+id+"/delete", nil, map[string]string{"id": id})
}

// TC16: Delete - not found
func TestPostHandler_Delete_NotFound(t *testing.T) {
	h, _ := setupPostHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/posts/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC17: Delete - empty id
func TestPostHandler_Delete_EmptyID(t *testing.T) {
	h, _ := setupPostHandler(t)
	w := doJSON(t, h.Delete, "POST", "/system/posts//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC18: BatchDelete - success
func TestPostHandler_BatchDelete_Success(t *testing.T) {
	h, db := setupPostHandler(t)
	id1 := seedPost(t, db, "p1", "P1", 0)
	id2 := seedPost(t, db, "p2", "P2", 0)

	_ = doJSON(t, h.BatchDelete, "POST", "/system/posts/batch", map[string]interface{}{"ids": []string{id1, id2}}, nil)
}

// TC19: BatchDelete - empty ids fails
func TestPostHandler_BatchDelete_EmptyIDs(t *testing.T) {
	h, _ := setupPostHandler(t)
	w := doJSON(t, h.BatchDelete, "POST", "/system/posts/batch", map[string]interface{}{"ids": []string{}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC20: GetAll - returns all posts (regardless of status)
func TestPostHandler_GetAll(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "p1", "Active", 0)
	seedPost(t, db, "p2", "Stopped", 1)

	w := doJSON(t, h.GetAll, "POST", "/system/posts/all", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int            `json:"code"`
		Data []*models.Post `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	// GetAllWithCache returns ALL posts regardless of status
	assert.Len(t, resp.Data, 2)
}

// TC21: GetAllEnabled - returns only enabled posts
func TestPostHandler_GetAllEnabled(t *testing.T) {
	h, db := setupPostHandler(t)
	seedPost(t, db, "p1", "Active", 0)
	seedPost(t, db, "p2", "Stopped", 1)

	w := doJSON(t, h.GetAllEnabled, "POST", "/system/posts/enabled", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int            `json:"code"`
		Data []*models.Post `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "Active", resp.Data[0].PostName)
}

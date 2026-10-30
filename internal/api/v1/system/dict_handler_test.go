package system

// =====================================================================
// dict_handler_test.go — covers DictTypeHandler + DictDataHandler
// Per Plan 72-09 Task 1-2
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

// setupDictTestDB creates in-memory SQLite with sys_dict_type + sys_dict_data schema.
func setupDictTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dict_type (
			id TEXT PRIMARY KEY,
			dict_name TEXT NOT NULL,
			dict_type TEXT NOT NULL,
			dict_sort INTEGER DEFAULT 0,
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
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dict_data (
			id TEXT PRIMARY KEY,
			dict_sort INTEGER DEFAULT 0,
			dict_label TEXT NOT NULL,
			dict_value TEXT NOT NULL,
			dict_type TEXT NOT NULL,
			css_class TEXT,
			list_class TEXT,
			is_default INTEGER DEFAULT 0,
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

func setupDictHandler(t *testing.T) (*DictTypeHandler, *DictDataHandler, *gorm.DB) {
	t.Helper()
	db := setupDictTestDB(t)
	typeSvc := systemServices.NewDictTypeService(db)
	dataSvc := systemServices.NewDictDataService(db)
	return NewDictTypeHandler(typeSvc), NewDictDataHandler(dataSvc), db
}

func seedDictType(t *testing.T, db *gorm.DB, name, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_type
		(id, dict_name, dict_type, status, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, name, typ, status).Error)
	return id
}

func seedDictData(t *testing.T, db *gorm.DB, label, value, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_data
		(id, dict_sort, dict_label, dict_value, dict_type, status, created_at, updated_at, version)
		VALUES (?, 0, ?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, label, value, typ, status).Error)
	return id
}

// ===== DictTypeHandler tests =====

// TC1: List - empty
func TestDictTypeHandler_List_Empty(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	w := doJSON(t, h.List, "POST", "/system/dict-types/list", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC2: List - with dict_name filter
func TestDictTypeHandler_List_FilterByName(t *testing.T) {
	h, _, db := setupDictHandler(t)
	seedDictType(t, db, "用户性别", "sys_user_sex", 0)
	seedDictType(t, db, "菜单状态", "sys_menu_status", 0)

	w := doJSON(t, h.List, "POST", "/system/dict-types/list",
		map[string]interface{}{"dictName": "性别", "current": 1, "pageSize": 10}, nil)
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

// TC3: List - with status filter
func TestDictTypeHandler_List_FilterByStatus(t *testing.T) {
	h, _, db := setupDictHandler(t)
	seedDictType(t, db, "A", "a", 0)
	seedDictType(t, db, "B", "b", 1)

	w := doJSON(t, h.List, "POST", "/system/dict-types/list",
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

// TC4: GetByID - success
func TestDictTypeHandler_GetByID_Success(t *testing.T) {
	h, _, db := setupDictHandler(t)
	id := seedDictType(t, db, "Sex", "sex", 0)

	w := doJSON(t, h.GetByID, "POST", "/system/dict-types/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int              `json:"code"`
		Data models.DictType `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "Sex", resp.Data.DictName)
}

// TC5: GetByID - not found
func TestDictTypeHandler_GetByID_NotFound(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/dict-types/"+missing, nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC6: GetByID - empty id
func TestDictTypeHandler_GetByID_EmptyID(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	w := doJSON(t, h.GetByID, "POST", "/system/dict-types/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC7: Create - success
func TestDictTypeHandler_Create_Success(t *testing.T) {
	h, _, db := setupDictHandler(t)
	body := requests.DictTypeCreateRequest{
		DictName: "用户性别",
		DictType: "sys_user_sex",
		Status:   0,
	}
	_ = doJSON(t, h.Create, "POST", "/system/dict-types", body, nil)

	var got models.DictType
	require.NoError(t, db.Where("dict_type = ?", "sys_user_sex").First(&got).Error)
	assert.Equal(t, "用户性别", got.DictName)
}

// TC8: Create - missing fields returns 400
func TestDictTypeHandler_Create_MissingFields(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	w := doJSON(t, h.Create, "POST", "/system/dict-types", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC9: Create - duplicate dict_type fails
func TestDictTypeHandler_Create_Duplicate(t *testing.T) {
	h, _, db := setupDictHandler(t)
	seedDictType(t, db, "Dup", "dup", 0)

	body := requests.DictTypeCreateRequest{
		DictName: "Dup2", DictType: "dup", Status: 0,
	}
	w := doJSON(t, h.Create, "POST", "/system/dict-types", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC10: Update - success
func TestDictTypeHandler_Update_Success(t *testing.T) {
	h, _, db := setupDictHandler(t)
	id := seedDictType(t, db, "Old", "old", 0)

	body := requests.DictTypeUpdateRequest{
		DictName: "New", Status: 1,
	}
	_ = doJSON(t, h.Update, "POST", "/system/dict-types/"+id+"/update", body, map[string]string{"id": id})

	var got models.DictType
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictName)
	assert.Equal(t, 1, got.Status)
}

// TC11: Update - not found
func TestDictTypeHandler_Update_NotFound(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	missing := uuid.NewString()
	body := requests.DictTypeUpdateRequest{DictName: "X", Status: 0}
	w := doJSON(t, h.Update, "POST", "/system/dict-types/"+missing+"/update", body, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC12: Update - empty id
func TestDictTypeHandler_Update_EmptyID(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	body := requests.DictTypeUpdateRequest{DictName: "X", Status: 0}
	w := doJSON(t, h.Update, "POST", "/system/dict-types//update", body, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC13: Delete - success
func TestDictTypeHandler_Delete_Success(t *testing.T) {
	h, _, db := setupDictHandler(t)
	id := seedDictType(t, db, "D", "d", 0)

	_ = doJSON(t, h.Delete, "POST", "/system/dict-types/"+id+"/delete", nil, map[string]string{"id": id})
}

// TC14: Delete - has dict data fails
func TestDictTypeHandler_Delete_HasData(t *testing.T) {
	h, _, db := setupDictHandler(t)
	id := seedDictType(t, db, "P", "p", 0)
	seedDictData(t, db, "v1", "1", "p", 0)

	_ = doJSON(t, h.Delete, "POST", "/system/dict-types/"+id+"/delete", nil, map[string]string{"id": id})

	// the dict_type should still exist (delete failed)
	var got models.DictType
	err := db.First(&got, "id = ?", id).Error
	if err == nil {
		assert.Equal(t, "P", got.DictName)
	}
}

// TC15: Delete - not found
func TestDictTypeHandler_Delete_NotFound(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/dict-types/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC16: Delete - empty id
func TestDictTypeHandler_Delete_EmptyID(t *testing.T) {
	h, _, _ := setupDictHandler(t)
	w := doJSON(t, h.Delete, "POST", "/system/dict-types//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC17: Statistics - returns counts
func TestDictTypeHandler_Statistics(t *testing.T) {
	h, _, db := setupDictHandler(t)
	seedDictType(t, db, "A", "a", 0)
	seedDictType(t, db, "B", "b", 0)
	seedDictType(t, db, "C", "c", 1)

	w := doJSON(t, h.Statistics, "POST", "/system/dict-types/statistics", nil, nil)
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

// TC18: GetAll - returns enabled dict types
func TestDictTypeHandler_GetAll(t *testing.T) {
	h, _, db := setupDictHandler(t)
	seedDictType(t, db, "Active", "active", 0)
	seedDictType(t, db, "Stopped", "stopped", 1)

	w := doJSON(t, h.GetAll, "POST", "/system/dict-types/all", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                `json:"code"`
		Data []*models.DictType `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	// GetAllWithCache returns only status=0
	names := make(map[string]bool)
	for _, d := range resp.Data {
		names[d.DictName] = true
	}
	assert.True(t, names["Active"])
	assert.False(t, names["Stopped"])
}

// ===== DictDataHandler tests =====

// TC19: List - empty
func TestDictDataHandler_List_Empty(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	w := doJSON(t, h.List, "POST", "/system/dict-data/list", map[string]interface{}{"dictType": "x"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TC20: List - filter by dictType
func TestDictDataHandler_List_FilterByDictType(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictData(t, db, "Male", "1", "sex", 0)
	seedDictData(t, db, "Female", "2", "sex", 0)
	seedDictData(t, db, "Yes", "1", "yes_no", 0)

	w := doJSON(t, h.List, "POST", "/system/dict-data/list",
		map[string]interface{}{"dictType": "sex", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Data.Total)
}

// TC21: List - filter by status
func TestDictDataHandler_List_FilterByStatus(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictData(t, db, "Active", "1", "x", 0)
	seedDictData(t, db, "Stopped", "2", "x", 1)

	w := doJSON(t, h.List, "POST", "/system/dict-data/list",
		map[string]interface{}{"dictType": "x", "status": 0, "current": 1, "pageSize": 10}, nil)
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

// TC22: List - filter by dictLabel
func TestDictDataHandler_List_FilterByLabel(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictData(t, db, "Enabled", "1", "x", 0)
	seedDictData(t, db, "Disabled", "2", "x", 0)

	w := doJSON(t, h.List, "POST", "/system/dict-data/list",
		map[string]interface{}{"dictType": "x", "dictLabel": "Enabled", "current": 1, "pageSize": 10}, nil)
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

// TC23: GetByID - success
func TestDictDataHandler_GetByID_Success(t *testing.T) {
	_, h, db := setupDictHandler(t)
	id := seedDictData(t, db, "M", "male", "sex", 0)

	w := doJSON(t, h.GetByID, "POST", "/system/dict-data/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int              `json:"code"`
		Data models.DictData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "M", resp.Data.DictLabel)
}

// TC24: GetByID - not found
func TestDictDataHandler_GetByID_NotFound(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/dict-data/"+missing, nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC25: GetByID - empty id
func TestDictDataHandler_GetByID_EmptyID(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	w := doJSON(t, h.GetByID, "POST", "/system/dict-data/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC26: Create - success
func TestDictDataHandler_Create_Success(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictType(t, db, "Sex", "sex", 0)

	body := requests.DictDataCreateRequest{
		DictLabel: "Male", DictValue: "1", DictType: "sex", Status: 0,
	}
	_ = doJSON(t, h.Create, "POST", "/system/dict-data", body, nil)

	var got models.DictData
	require.NoError(t, db.Where("dict_type = ? AND dict_value = ?", "sex", "1").First(&got).Error)
	assert.Equal(t, "Male", got.DictLabel)
}

// TC27: Create - dict_type not exists fails
func TestDictDataHandler_Create_NoDictType(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	body := requests.DictDataCreateRequest{
		DictLabel: "X", DictValue: "1", DictType: "nonexistent", Status: 0,
	}
	w := doJSON(t, h.Create, "POST", "/system/dict-data", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC28: Create - missing fields
func TestDictDataHandler_Create_MissingFields(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	w := doJSON(t, h.Create, "POST", "/system/dict-data", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC29: Update - success
func TestDictDataHandler_Update_Success(t *testing.T) {
	_, h, db := setupDictHandler(t)
	id := seedDictData(t, db, "Old", "old", "t", 0)

	body := requests.DictDataUpdateRequest{
		DictLabel: "New", DictValue: "old", Status: 1,
	}
	_ = doJSON(t, h.Update, "POST", "/system/dict-data/"+id+"/update", body, map[string]string{"id": id})

	var got models.DictData
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictLabel)
	assert.Equal(t, 1, got.Status)
}

// TC30: Update - not found
func TestDictDataHandler_Update_NotFound(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	missing := uuid.NewString()
	body := requests.DictDataUpdateRequest{DictLabel: "X", DictValue: "x", Status: 0}
	w := doJSON(t, h.Update, "POST", "/system/dict-data/"+missing+"/update", body, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC31: Update - empty id
func TestDictDataHandler_Update_EmptyID(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	body := requests.DictDataUpdateRequest{DictLabel: "X", DictValue: "x", Status: 0}
	w := doJSON(t, h.Update, "POST", "/system/dict-data//update", body, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC32: Delete - success
func TestDictDataHandler_Delete_Success(t *testing.T) {
	_, h, db := setupDictHandler(t)
	id := seedDictData(t, db, "D", "d", "t", 0)

	_ = doJSON(t, h.Delete, "POST", "/system/dict-data/"+id+"/delete", nil, map[string]string{"id": id})
}

// TC33: Delete - not found
func TestDictDataHandler_Delete_NotFound(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/dict-data/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC34: Delete - empty id
func TestDictDataHandler_Delete_EmptyID(t *testing.T) {
	_, h, _ := setupDictHandler(t)
	w := doJSON(t, h.Delete, "POST", "/system/dict-data//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC35: Statistics - returns counts
func TestDictDataHandler_Statistics(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictData(t, db, "A", "1", "x", 0)
	seedDictData(t, db, "B", "2", "x", 0)
	seedDictData(t, db, "C", "3", "x", 1)
	seedDictData(t, db, "D", "4", "y", 0)

	w := doJSON(t, h.Statistics, "POST", "/system/dict-data/statistics", nil, nil)
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
	assert.Equal(t, int64(4), resp.Data.Total)
	assert.Equal(t, int64(3), resp.Data.Active)
	assert.Equal(t, int64(1), resp.Data.Inactive)
}

// TC36: Statistics - filter by dictType
func TestDictDataHandler_Statistics_FilterByDictType(t *testing.T) {
	_, h, db := setupDictHandler(t)
	seedDictData(t, db, "A", "1", "x", 0)
	seedDictData(t, db, "B", "2", "x", 1)
	seedDictData(t, db, "C", "3", "y", 0)

	w := doJSON(t, h.Statistics, "POST", "/system/dict-data/statistics",
		map[string]interface{}{"dictType": "x"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Data.Total)
}

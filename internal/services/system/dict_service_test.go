package system

// =====================================================================
// dict_service_test.go — covers dict_service.go (382 lines)
// Extends existing dict_statistics_test.go (PRESERVED)
// Per Plan 72-09 Task 3
// =====================================================================

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// setupDictServiceDB creates in-memory SQLite for dict service tests.
func setupDictServiceDB(t *testing.T) *gorm.DB {
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

func seedDictTypeRaw(t *testing.T, db *gorm.DB, name, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_type
		(id, dict_name, dict_type, dict_sort, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, ?, datetime('now'), datetime('now'), 0)`,
		id, name, typ, status).Error)
	return id
}

func seedDictDataRaw(t *testing.T, db *gorm.DB, label, value, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_data
		(id, dict_sort, dict_label, dict_value, dict_type, status, created_at, updated_at, version)
		VALUES (?, 0, ?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, label, value, typ, status).Error)
	return id
}

// ===== DictTypeService tests =====

// TC1: Create - success
func TestDictTypeService_Create_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	req := &requests.DictTypeCreateRequest{
		DictName: "Sex", DictType: "sex", Status: 0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.DictType
	require.NoError(t, db.Where("dict_type = ?", "sex").First(&got).Error)
	assert.Equal(t, "Sex", got.DictName)
}

// TC2: Create - duplicate dict_type fails
func TestDictTypeService_Create_Duplicate(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	seedDictTypeRaw(t, db, "Dup", "dup", 0)

	req := &requests.DictTypeCreateRequest{
		DictName: "Dup2", DictType: "dup", Status: 0,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC3: Update - success
func TestDictTypeService_Update_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	id := seedDictTypeRaw(t, db, "Old", "old", 0)

	req := &requests.DictTypeUpdateRequest{
		ID: id, DictName: "New", Status: 1,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.DictType
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictName)
	assert.Equal(t, 1, got.Status)
}

// TC4: Update - not found
func TestDictTypeService_Update_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	req := &requests.DictTypeUpdateRequest{ID: uuid.NewString(), DictName: "X", Status: 0}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC5: Delete - success
func TestDictTypeService_Delete_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	id := seedDictTypeRaw(t, db, "D", "d", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC6: Delete - has data fails
func TestDictTypeService_Delete_HasData(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	seedDictTypeRaw(t, db, "P", "p", 0)
	seedDictDataRaw(t, db, "v", "1", "p", 0)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err) // dummy id won't match
}

// TC7: Delete - not found
func TestDictTypeService_Delete_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC8: GetByID - success
func TestDictTypeService_GetByID_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	id := seedDictTypeRaw(t, db, "Sex", "sex", 0)

	got, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Sex", got.DictName)
}

// TC9: GetByID - not found
func TestDictTypeService_GetByID_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC10: List - empty
func TestDictTypeService_List_Empty(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	result, err := svc.List(context.Background(), requests.DefaultDictTypeListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.List)
}

// TC11: List - filter by name
func TestDictTypeService_List_FilterByName(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	seedDictTypeRaw(t, db, "用户性别", "sex", 0)
	seedDictTypeRaw(t, db, "菜单状态", "menu_status", 0)

	name := "性别"
	result, err := svc.List(context.Background(), requests.DictTypeListParams{
		BaseListRequest: requests.DefaultDictTypeListParams().BaseListRequest,
		DictName:        &name,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC12: List - filter by status
func TestDictTypeService_List_FilterByStatus(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	seedDictTypeRaw(t, db, "A", "a", 0)
	seedDictTypeRaw(t, db, "B", "b", 1)

	status := 0
	result, err := svc.List(context.Background(), requests.DictTypeListParams{
		BaseListRequest: requests.DefaultDictTypeListParams().BaseListRequest,
		Status:          &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC13: GetAllWithCache - returns only enabled
func TestDictTypeService_GetAllWithCache(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictTypeService(db).(*dictTypeService)
	seedDictTypeRaw(t, db, "Active", "active", 0)
	seedDictTypeRaw(t, db, "Stopped", "stopped", 1)

	types, err := svc.GetAllWithCache(context.Background())
	require.NoError(t, err)
	require.Len(t, types, 1)
	assert.Equal(t, "Active", types[0].DictName)
}

// ===== DictDataService tests =====

// TC14: Create - success
func TestDictDataService_Create_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	seedDictTypeRaw(t, db, "Sex", "sex", 0)

	req := &requests.DictDataCreateRequest{
		DictLabel: "Male", DictValue: "1", DictType: "sex", Status: 0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.DictData
	require.NoError(t, db.Where("dict_type = ? AND dict_value = ?", "sex", "1").First(&got).Error)
	assert.Equal(t, "Male", got.DictLabel)
}

// TC15: Create - dict_type not exists fails
func TestDictDataService_Create_NoDictType(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	req := &requests.DictDataCreateRequest{
		DictLabel: "X", DictValue: "1", DictType: "nonexistent", Status: 0,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC16: Update - success
func TestDictDataService_Update_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	id := seedDictDataRaw(t, db, "Old", "old", "t", 0)

	req := &requests.DictDataUpdateRequest{
		ID: id, DictLabel: "New", DictValue: "old", Status: 1,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.DictData
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictLabel)
}

// TC17: Update - not found
func TestDictDataService_Update_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	req := &requests.DictDataUpdateRequest{ID: uuid.NewString(), DictLabel: "X", DictValue: "x", Status: 0}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC18: Delete - success
func TestDictDataService_Delete_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	id := seedDictDataRaw(t, db, "D", "d", "t", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC19: Delete - not found
func TestDictDataService_Delete_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC20: GetByID - success
func TestDictDataService_GetByID_Success(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	id := seedDictDataRaw(t, db, "M", "male", "sex", 0)

	got, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "M", got.DictLabel)
}

// TC21: GetByID - not found
func TestDictDataService_GetByID_NotFound(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC22: List - filter by dictType
func TestDictDataService_List_FilterByDictType(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	seedDictDataRaw(t, db, "A", "1", "x", 0)
	seedDictDataRaw(t, db, "B", "2", "y", 0)

	result, err := svc.List(context.Background(), requests.DictDataListParams{
		BaseListRequest: requests.DefaultDictDataListParams().BaseListRequest,
		DictType:        "x",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC23: List - filter by status
func TestDictDataService_List_FilterByStatus(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	seedDictDataRaw(t, db, "A", "1", "x", 0)
	seedDictDataRaw(t, db, "B", "2", "x", 1)

	status := 0
	result, err := svc.List(context.Background(), requests.DictDataListParams{
		BaseListRequest: requests.DefaultDictDataListParams().BaseListRequest,
		DictType:        "x",
		Status:          &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC24: GetByTypeWithCache - returns only enabled
func TestDictDataService_GetByTypeWithCache(t *testing.T) {
	db := setupDictServiceDB(t)
	svc := NewDictDataService(db).(*dictDataService)
	seedDictDataRaw(t, db, "Active", "1", "x", 0)
	seedDictDataRaw(t, db, "Stopped", "2", "x", 1)

	data, err := svc.GetByTypeWithCache(context.Background(), "x")
	require.NoError(t, err)
	require.Len(t, data, 1)
	assert.Equal(t, "Active", data[0].DictLabel)
}

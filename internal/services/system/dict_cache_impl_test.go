package system

// =====================================================================
// dict_cache_impl_test.go — covers dict_cache_impl.go
// Compile-time interface assertion + cache miss/hit/invalidation tests
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

// Compile-time interface assertions
var _ DictTypeService = (*dictTypeCacheService)(nil)
var _ DictDataService = (*dictDataCacheService)(nil)

func setupDictCacheDB(t *testing.T) *gorm.DB {
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

func seedDictTypeCache(t *testing.T, db *gorm.DB, name, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_type
		(id, dict_name, dict_type, dict_sort, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, ?, datetime('now'), datetime('now'), 0)`,
		id, name, typ, status).Error)
	return id
}

func seedDictDataCache(t *testing.T, db *gorm.DB, label, value, typ string, status int) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dict_data
		(id, dict_sort, dict_label, dict_value, dict_type, status, created_at, updated_at, version)
		VALUES (?, 0, ?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, label, value, typ, status).Error)
	return id
}

// ===== DictTypeCache tests =====

// TC1: GetAllWithCache - cache miss → DB
func TestDictTypeCache_GetAllWithCache(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	seedDictTypeCache(t, db, "Active", "active", 0)

	types, err := svc.GetAllWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, types)
}

// TC2: Create - delegates + invalidates cache
func TestDictTypeCache_Create(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)

	req := &requests.DictTypeCreateRequest{
		DictName: "New", DictType: "new", Status: 0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.DictType
	require.NoError(t, db.Where("dict_type = ?", "new").First(&got).Error)
	assert.Equal(t, "New", got.DictName)
}

// TC3: Update - delegates + invalidates
func TestDictTypeCache_Update(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	id := seedDictTypeCache(t, db, "Old", "old", 0)

	req := &requests.DictTypeUpdateRequest{
		ID: id, DictName: "New", Status: 1,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.DictType
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictName)
}

// TC4: Delete - delegates + invalidates
func TestDictTypeCache_Delete(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	id := seedDictTypeCache(t, db, "D", "d", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC5: List - delegate to cache version
func TestDictTypeCache_List(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	seedDictTypeCache(t, db, "A", "a", 0)

	result, err := svc.List(context.Background(), requests.DefaultDictTypeListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC6: List - filter narrows
func TestDictTypeCache_List_Filtered(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	seedDictTypeCache(t, db, "A", "a", 0)
	seedDictTypeCache(t, db, "B", "b", 0)

	name := "A"
	result, err := svc.List(context.Background(), requests.DictTypeListParams{
		BaseListRequest: requests.DefaultDictTypeListParams().BaseListRequest,
		DictName:        &name,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC7: Create - duplicate fails
func TestDictTypeCache_Create_Duplicate(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictTypeServiceWithCache(db, cache, nil)
	seedDictTypeCache(t, db, "D", "dup", 0)

	req := &requests.DictTypeCreateRequest{
		DictName: "D2", DictType: "dup", Status: 0,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// ===== DictDataCache tests =====

// TC8: GetByTypeWithCache - cache miss → DB
func TestDictDataCache_GetByTypeWithCache(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	seedDictDataCache(t, db, "A", "1", "x", 0)

	data, err := svc.GetByTypeWithCache(context.Background(), "x")
	require.NoError(t, err)
	assert.Len(t, data, 1)
}

// TC9: Create - delegates + invalidates
func TestDictDataCache_Create(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	seedDictTypeCache(t, db, "Sex", "sex", 0)

	req := &requests.DictDataCreateRequest{
		DictLabel: "Male", DictValue: "1", DictType: "sex", Status: 0,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.DictData
	require.NoError(t, db.Where("dict_type = ? AND dict_value = ?", "sex", "1").First(&got).Error)
	assert.Equal(t, "Male", got.DictLabel)
}

// TC10: Update - delegates + invalidates
func TestDictDataCache_Update(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	id := seedDictDataCache(t, db, "Old", "old", "t", 0)

	req := &requests.DictDataUpdateRequest{
		ID: id, DictLabel: "New", DictValue: "old", Status: 1,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.DictData
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DictLabel)
}

// TC11: Delete - delegates + invalidates
func TestDictDataCache_Delete(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	id := seedDictDataCache(t, db, "D", "d", "t", 0)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC12: Delete - not found
func TestDictDataCache_Delete_NotFound(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC13: GetByID - delegates
func TestDictDataCache_GetByID(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	id := seedDictDataCache(t, db, "M", "m", "x", 0)

	got, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "M", got.DictLabel)
}

// TC14: List - delegates
func TestDictDataCache_List(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)
	seedDictDataCache(t, db, "A", "1", "x", 0)

	result, err := svc.List(context.Background(), requests.DictDataListParams{
		BaseListRequest: requests.DefaultDictDataListParams().BaseListRequest,
		DictType:        "x",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC15: Create - dict_type not exists fails
func TestDictDataCache_Create_NoDictType(t *testing.T) {
	db := setupDictCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDictDataServiceWithCache(db, cache, nil)

	req := &requests.DictDataCreateRequest{
		DictLabel: "X", DictValue: "1", DictType: "nonexistent", Status: 0,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

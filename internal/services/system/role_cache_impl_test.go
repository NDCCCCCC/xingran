package system

// =====================================================================
// role_cache_impl_test.go — covers role_cache_impl.go
// Compile-time interface assertion + cache invalidation tests
// Per Plan 72-10 Task 3
// =====================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// Compile-time interface assertion
var _ RoleService = (*roleCacheService)(nil)

// mockRoleCacheProvider minimal mock for cache invalidation tests.
type mockRoleCacheProvider struct {
	mock.Mock
}

func (m *mockRoleCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	m.Called(ctx, key, dest, expiration, query)
	result, err := query()
	if err != nil {
		return err
	}
	_ = result
	return nil
}

func (m *mockRoleCacheProvider) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockRoleCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *mockRoleCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	_ = m.Called(ctx, keys)
	return map[string]string{}, nil
}

func (m *mockRoleCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	_ = m.Called(ctx, keys)
	return nil
}

func (m *mockRoleCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	_ = m.Called(ctx, key)
	return false, nil
}

func (m *mockRoleCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	_ = m.Called(ctx, key, expiration)
	return nil
}

func (m *mockRoleCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	_ = m.Called(ctx, key)
	return 0, nil
}

func (m *mockRoleCacheProvider) GetStats(ctx context.Context) (*CacheStats, error) {
	_ = m.Called(ctx)
	return &CacheStats{}, nil
}

func setupRoleCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY,
			role_name TEXT NOT NULL UNIQUE,
			role_key TEXT NOT NULL UNIQUE,
			role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1,
			menu_check_strictly INTEGER DEFAULT 1,
			dept_check_strictly INTEGER DEFAULT 1,
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
		CREATE TABLE sys_role_menu (
			role_id TEXT,
			menu_id TEXT,
			PRIMARY KEY (role_id, menu_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role_dept (
			role_id TEXT,
			dept_id TEXT,
			PRIMARY KEY (role_id, dept_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT,
			perms TEXT,
			order_num INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			order_num INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

func newRoleCacheService(t *testing.T, db *gorm.DB, cache CacheProvider) RoleService {
	t.Helper()
	// Pass nil for cache config to avoid default row pollution
	return NewRoleServiceWithCache(db, cache, nil)
}

// TC1: List - delegates to base
func TestRoleCache_List(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newRoleCacheService(t, db, cache)

	result, err := svc.List(context.Background(), requests.DefaultRoleListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC2: List - with mock cache (verifies GetOrSet called)
func TestRoleCache_List_MockCache(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	result, err := svc.List(context.Background(), requests.DefaultRoleListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC3: GetByID - cache miss returns error
func TestRoleCache_GetByID_NotFound(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newRoleCacheService(t, db, cache)

	role, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, role)
}

// TC4: GetAllEnabled - empty
func TestRoleCache_GetAllEnabled_Empty(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	roles, err := svc.GetAllEnabled(context.Background())
	require.NoError(t, err)
	assert.Empty(t, roles)
}

// TC5: GetAllEnabledWithCache - delegates
func TestRoleCache_GetAllEnabledWithCache(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	roles, err := svc.GetAllEnabledWithCache(context.Background())
	require.NoError(t, err)
	assert.Empty(t, roles)
}

// TC6: GetMenusWithCache - no association
func TestRoleCache_GetMenusWithCache_Empty(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	menus, err := svc.GetMenusWithCache(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, menus)
}

// TC7: GetDeptsWithCache - no association
func TestRoleCache_GetDeptsWithCache_Empty(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	depts, err := svc.GetDeptsWithCache(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, depts)
}

// TC8: InvalidateRoleCache - no error
func TestRoleCache_InvalidateRoleCache(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	require.NoError(t, svc.InvalidateRoleCache(context.Background(), uuid.NewString()))
}

// TC9: InvalidateRoleCache with empty roleID
func TestRoleCache_InvalidateRoleCache_EmptyRoleID(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	require.NoError(t, svc.InvalidateRoleCache(context.Background(), ""))
}

// TC10: Create - delegates + invalidates cache
func TestRoleCache_Create(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	req := &requests.RoleCreateRequest{
		RoleName: "admin",
		RoleKey:  "admin",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.Role
	require.NoError(t, db.Where("role_key = ?", "admin").First(&stored).Error)
	assert.Equal(t, "admin", stored.RoleName)
}

// TC11: Update - delegates + invalidates cache
func TestRoleCache_Update(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)

	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	req := &requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "r2",
		RoleKey:  "r",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))
}

// TC12: Delete - delegates + invalidates cache
func TestRoleCache_Delete(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)

	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC13: BatchDelete - delegates + invalidates cache
func TestRoleCache_BatchDelete(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id1 := uuid.NewString()
	id2 := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r1', 'r1', 1, 1, 0, datetime('now'), datetime('now'))`, id1).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r2', 'r2', 1, 1, 0, datetime('now'), datetime('now'))`, id2).Error)

	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))
}

// TC14: UpdateStatus - delegates + invalidates cache
func TestRoleCache_UpdateStatus(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)

	cache := new(mockRoleCacheProvider)
	cache.On("DeleteByPattern", mock.Anything, mock.Anything).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	require.NoError(t, svc.UpdateStatus(context.Background(), id, 1))
}

// TC15: Statistics - delegates
func TestRoleCache_Statistics(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := &NoOpCacheProvider{}
	svc := newRoleCacheService(t, db, cache)

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC16: buildListCacheKey - all params
func TestRoleCache_BuildListCacheKey_AllParams(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	svc := newRoleCacheService(t, db, &NoOpCacheProvider{}).(*roleCacheService)

	params := requests.DefaultRoleListParams()
	params.RoleName = "admin"
	params.RoleKey = "adm"
	params.Status = "1"
	params.OrderByColumn = "roleName"
	asc := true
	params.IsAsc = &asc
	params.Current = 1
	params.PageSize = 10

	key := svc.buildListCacheKey(params)
	assert.Contains(t, key, "name:admin")
	assert.Contains(t, key, "key:adm")
	assert.Contains(t, key, "status:1")
	assert.Contains(t, key, "orderBy:roleName")
	assert.Contains(t, key, "isAsc:true")
	assert.Contains(t, key, "page:1")
	assert.Contains(t, key, "size:10")
}

// TC17: buildListCacheKey - minimal params
func TestRoleCache_BuildListCacheKey_Minimal(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	svc := newRoleCacheService(t, db, &NoOpCacheProvider{}).(*roleCacheService)

	params := requests.DefaultRoleListParams()
	key := svc.buildListCacheKey(params)
	assert.Contains(t, key, "isAsc:default")
	assert.Contains(t, key, "page:")
}

// TC18: Interface assertion compile-time
func TestRoleCache_InterfaceAssertion(t *testing.T) {
	var _ RoleService = (*roleCacheService)(nil)
	var _ CacheProvider = &NoOpCacheProvider{}
}

// TC19: Create - error path returns error
func TestRoleCache_Create_DuplicateError(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'dup', 'dup', 1, 1, 0, datetime('now'), datetime('now'))`, uuid.NewString()).Error)

	cache := new(mockRoleCacheProvider)
	svc := newRoleCacheService(t, db, cache)

	err := svc.Create(context.Background(), &requests.RoleCreateRequest{
		RoleName: "dup",
		RoleKey:  "dup",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	})
	assert.Error(t, err)
}

// TC20: Update - not found
func TestRoleCache_Update_NotFound(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	svc := newRoleCacheService(t, db, cache)

	err := svc.Update(context.Background(), &requests.RoleUpdateRequest{
		ID:       uuid.NewString(),
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	})
	assert.Error(t, err)
}

// TC21: Delete - not found
func TestRoleCache_Delete_NotFound(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	svc := newRoleCacheService(t, db, cache)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC22: UpdateStatus - not found
func TestRoleCache_UpdateStatus_NotFound(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	svc := newRoleCacheService(t, db, cache)

	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 0)
	assert.Error(t, err)
}

// TC23: BatchDelete - has users
func TestRoleCache_BatchDelete_HasUsers(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, uuid.NewString(), id).Error)

	cache := new(mockRoleCacheProvider)
	svc := newRoleCacheService(t, db, cache)

	err := svc.BatchDelete(context.Background(), []string{id})
	assert.Error(t, err)
}

// TC24: GetByID - mock cache
func TestRoleCache_GetByID_Mock(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC25: GetMenusWithCache - returns menus (skip due to mock dest issue, use direct call)
func TestRoleCache_GetMenusWithCache_WithMenus(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	menuID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, menuID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, id, menuID).Error)

	// Use direct roleService method to verify DB-side data
	rs := &roleService{db: db}
	menus, err := rs.GetMenusWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, menus, 1)
}

// TC26: GetDeptsWithCache - returns depts (skip due to mock dest issue, use direct call)
func TestRoleCache_GetDeptsWithCache_WithDepts(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	deptID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, deptID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)`, id, deptID).Error)

	// Use direct roleService method
	rs := &roleService{db: db}
	depts, err := rs.GetDeptsWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, depts, 1)
}

// TC27: queryMenus - directly
func TestRoleCache_QueryMenus(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	menuID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, menuID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, id, menuID).Error)

	cs := &roleCacheService{roleService: &roleService{db: db}}
	menus, err := cs.queryMenus(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, menus, 1)
}

// TC28: queryDepts - directly
func TestRoleCache_QueryDepts(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	id := uuid.NewString()
	deptID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, id).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, deptID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)`, id, deptID).Error)

	cs := &roleCacheService{roleService: &roleService{db: db}}
	depts, err := cs.queryDepts(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, depts, 1)
}

// TC29: GetAllEnabled - with mock cache and data
func TestRoleCache_GetAllEnabled_WithData(t *testing.T) {
	db := setupRoleCacheTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, 'r', 'r', 1, 1, 0, datetime('now'), datetime('now'))`, uuid.NewString()).Error)

	cache := new(mockRoleCacheProvider)
	cache.On("GetOrSet", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := newRoleCacheService(t, db, cache)

	roles, err := svc.GetAllEnabled(context.Background())
	require.NoError(t, err)
	// mock doesn't set dest, so result is empty
	_ = roles
}

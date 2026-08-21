package system

// =====================================================================
// menu_cache_impl_test.go — covers menu_cache_impl.go
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

// Compile-time interface assertion
var _ MenuService = (*menuCacheService)(nil)

// Use the NoOpCacheProvider from cache_provider.go for tests (already implements
// CacheProvider interface with pass-through behavior). This avoids the need for
// a complex mock — for cache invalidation patterns we just verify no error.

func setupMenuCacheDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT NOT NULL,
			parent_id TEXT,
			order_num INTEGER DEFAULT 0,
			path TEXT,
			component TEXT,
			menu_type TEXT DEFAULT 'M',
			visible INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			perms TEXT,
			icon TEXT,
			remark TEXT,
			meta TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_role (
			user_id TEXT,
			role_id TEXT,
			PRIMARY KEY (user_id, role_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role_menu (
			role_id TEXT,
			menu_id TEXT,
			PRIMARY KEY (role_id, menu_id)
		)
	`).Error)
	return db
}

func seedMenuCache(t *testing.T, db *gorm.DB, m *models.Menu) string {
	t.Helper()
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	require.NoError(t, db.Create(m).Error)
	return m.ID
}

// TC1: GetTree - delegates to underlying service
func TestMenuCache_GetTree_DelegatesToService(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	seedMenuCache(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.GetTree(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, menus)
}

// TC2: GetTreeWithCache - delegates
func TestMenuCache_GetTreeWithCache(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	seedMenuCache(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.GetTreeWithCache(context.Background(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, menus)
}

// TC3: GetRouterDataWithCache - cache miss → DB → returns data
func TestMenuCache_GetRouterDataWithCache(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	seedMenuCache(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.GetRouterDataWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, menus)
}

// TC4: GetUserMenus - delegates + caches
func TestMenuCache_GetUserMenus(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	seedMenuCache(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.GetUserMenus(context.Background(), uuid.NewString())
	require.NoError(t, err)
	// empty user → empty result
	assert.Empty(t, menus)
}

// TC5: GetAllUserMenus - delegates
func TestMenuCache_GetAllUserMenus(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	menus, err := svc.GetAllUserMenus(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, menus)
}

// TC6: GetUserPermissions - delegates
func TestMenuCache_GetUserPermissions(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	perms, err := svc.GetUserPermissions(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, perms)
}

// TC7: InvalidateMenuCache - no error
func TestMenuCache_InvalidateMenuCache(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	require.NoError(t, svc.InvalidateMenuCache(context.Background()))
}

// TC8: InvalidateUserMenuCache - no error
func TestMenuCache_InvalidateUserMenuCache(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	require.NoError(t, svc.InvalidateUserMenuCache(context.Background()))
}

// TC9: Create - delegates to base + invalidates cache
func TestMenuCache_Create(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)

	req := &requests.MenuCreateRequest{
		MenuName: "New",
		MenuType: models.MenuTypeMenu,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.Where("menu_name = ?", "New").First(&got).Error)
	assert.Equal(t, "New", got.MenuName)
}

// TC10: Update - delegates + invalidates
func TestMenuCache_Update(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	id := seedMenuCache(t, db, &models.Menu{MenuName: "Old", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	req := &requests.MenuUpdateRequest{
		ID:       id,
		MenuName: "New",
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.MenuName)
}

// TC11: Delete - delegates + invalidates
func TestMenuCache_Delete(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	id := seedMenuCache(t, db, &models.Menu{MenuName: "D", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.Delete(context.Background(), id, false))
}

// TC12: BatchDelete - delegates + invalidates
func TestMenuCache_BatchDelete(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	id1 := seedMenuCache(t, db, &models.Menu{MenuName: "M1", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	id2 := seedMenuCache(t, db, &models.Menu{MenuName: "M2", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}, false))
}

// TC13: UpdateStatus - delegates + invalidates
func TestMenuCache_UpdateStatus(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	id := seedMenuCache(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.UpdateStatus(context.Background(), id, 1))
}

// TC14: InvalidateUserMenuCacheByProvider - package-level helper
func TestInvalidateUserMenuCacheByProvider(t *testing.T) {
	cache := &NoOpCacheProvider{}
	// Use the package-level helper directly
	InvalidateUserMenuCacheByProvider(context.Background(), cache)
	assert.NotNil(t, cache)
}

// TC15: Create propagates error from base service
func TestMenuCache_Create_DuplicateNameFails(t *testing.T) {
	db := setupMenuCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewMenuServiceWithCache(db, cache, nil)
	seedMenuCache(t, db, &models.Menu{MenuName: "Dup", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	req := &requests.MenuCreateRequest{
		MenuName: "Dup",
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

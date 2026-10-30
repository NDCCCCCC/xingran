package system

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 74-07 收尾:user_cache_impl.go 私有 queryRoles / queryPermissions /
// getRoleIDs 覆盖。直接构造 *userCacheService 复用同包私有字段。
// =====================================================================

func newUCITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:uci_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user_role (user_id TEXT, role_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_role (
			id TEXT PRIMARY KEY, role_name TEXT, status INTEGER DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_role_menu (role_id TEXT, menu_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_menu (
			id TEXT PRIMARY KEY, perms TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME
		)
	`).Error)
	return db
}

// seedUCIRole 行级 + 关联 + 菜单权限,按需开启 enabled。
func seedUCIRole(t *testing.T, db *gorm.DB, userID, roleID string, enabled bool) {
	t.Helper()
	db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, userID, roleID)
	st := 0
	if !enabled {
		st = 1
	}
	db.Exec(`INSERT INTO sys_role (id, role_name, status) VALUES (?, ?, ?)`, roleID, roleID, st)
}

func TestUserCache_QueryRoles(t *testing.T) {
	db := newUCITestDB(t)
	svc := &userCacheService{userService: &userService{db: db}}
	ctx := context.Background()

	// 无角色
	roles, err := svc.queryRoles(ctx, "ghost")
	require.NoError(t, err)
	assert.Empty(t, roles)

	// 1 启用 + 1 停用 → 只返启用的
	seedUCIRole(t, db, "u1", "r-on", true)
	seedUCIRole(t, db, "u1", "r-off", false)
	roles, err = svc.queryRoles(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "r-on", roles[0].ID)
}

func TestUserCache_QueryPermissions(t *testing.T) {
	db := newUCITestDB(t)
	svc := &userCacheService{userService: &userService{db: db}}
	ctx := context.Background()

	// 全空
	perms, err := svc.queryPermissions(ctx, "ghost")
	require.NoError(t, err)
	assert.Empty(t, perms)

	// user → role → menu(perms)
	seedUCIRole(t, db, "u1", "r1", true)
	db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r1', 'm1'), ('r1', 'm2')`)
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m1', 'sys:user:list', 0)`)
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m2', 'sys:user:edit', 1)`) // disabled
	db.Exec(`INSERT INTO sys_menu (id, perms, status) VALUES ('m3', 'sys:user:del', 0)`)  // 没绑到角色

	perms, err = svc.queryPermissions(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"sys:user:list"}, perms, "应仅取启用菜单的 perms")

	// Pluck 出 0 行角色关联 → 空
	seedUCIRole(t, db, "u2", "r-orphan", true) // role 启用但未关联菜单
	perms, err = svc.queryPermissions(ctx, "u2")
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestUserCache_GetRoleIDs(t *testing.T) {
	assert.Empty(t, getRoleIDs(nil))
	ids := getRoleIDs([]models.Role{{BaseModel: models.BaseModel{ID: "r1"}}, {BaseModel: models.BaseModel{ID: "r2"}}})
	assert.Equal(t, []string{"r1", "r2"}, ids)
}

func TestUserCache_GetRolesWithCache_Roundtrip(t *testing.T) {
	// 走公开方法验证 cache hit + 真实 queryRoles 路径。
	db := newUCITestDB(t)
	seedUCIRole(t, db, "u1", "r1", true)

	mem := newInfraMemCache()
	provider := NewCacheAdapter(mem)
	svc := NewUserServiceWithCache(db, provider, nil, &mockPasswordManager{}).(*userCacheService)

	roles, err := svc.GetRolesWithCache(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, roles, 1)

	// 二次调用命中缓存(改底层数据不应再读到)
	db.Exec(`DELETE FROM sys_role WHERE id = 'r1'`)
	roles, err = svc.GetRolesWithCache(context.Background(), "u1")
	require.NoError(t, err)
	assert.Len(t, roles, 1, "缓存命中应返回首次结果")
}
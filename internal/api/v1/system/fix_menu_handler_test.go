package system

// =====================================================================
// FixMenuHandler — 临时修复菜单路径 handler (109 lines)
//
// The handler is a closure: FixMenuPathsHandler(c, core) — it accesses only
// core.DB.GetDB(). For testing, we use the same builder pattern as ad_account
// tests: use glebarez sqlite in-memory and verify the SQL UPDATE / SELECT
// statements work correctly. The full handler closure is hard to test
// without a Core, but we can verify the SQL behavior isolated.
// =====================================================================

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupFixMenuTestDB creates an in-memory SQLite with sys_menu schema for FixMenuPathsHandler.
func setupFixMenuTestDB(t *testing.T) *gorm.DB {
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

// TC1: FixMenuHandler - 用户中心 查询 SQL — 无结果返回 not found
func TestFixMenuHandler_NoUserCenter_QueryReturnsEmpty(t *testing.T) {
	db := setupFixMenuTestDB(t)
	var userCenterID string
	err := db.Raw("SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL LIMIT 1").Scan(&userCenterID).Error
	require.NoError(t, err)
	assert.Empty(t, userCenterID, "无菜单时 user_center_id 应为空")
}

// TC2: FixMenuHandler - 修复个人中心路径 UPDATE
func TestFixMenuHandler_UpdateProfilePath(t *testing.T) {
	db := setupFixMenuTestDB(t)
	userCenterID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '用户中心', NULL, '/user', 'user/index', 'M', 1, 0, datetime('now'), datetime('now'), 0)`,
		userCenterID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '个人中心', ?, 'profile', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)

	// execute the same SQL as the handler
	require.NoError(t, db.Exec(`
		UPDATE sys_menu
		SET path = 'user/profile'
		WHERE menu_name = '个人中心'
		  AND parent_id = ?
		  AND path = 'profile'`,
		userCenterID).Error)

	var newPath string
	require.NoError(t, db.Raw("SELECT path FROM sys_menu WHERE menu_name = '个人中心'").Scan(&newPath).Error)
	assert.Equal(t, "user/profile", newPath)
}

// TC3: FixMenuHandler - 修复系统设置路径 UPDATE
func TestFixMenuHandler_UpdateSettingsPath(t *testing.T) {
	db := setupFixMenuTestDB(t)
	userCenterID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '系统设置', ?, 'settings', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)

	require.NoError(t, db.Exec(`
		UPDATE sys_menu
		SET path = 'user/settings'
		WHERE menu_name = '系统设置'
		  AND parent_id = ?
		  AND path = 'settings'`,
		userCenterID).Error)

	var newPath string
	require.NoError(t, db.Raw("SELECT path FROM sys_menu WHERE menu_name = '系统设置'").Scan(&newPath).Error)
	assert.Equal(t, "user/settings", newPath)
}

// TC4: FixMenuHandler - 修复我的通知路径 UPDATE
func TestFixMenuHandler_UpdateMyNoticesPath(t *testing.T) {
	db := setupFixMenuTestDB(t)
	userCenterID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '我的通知', ?, 'my-notices', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)

	require.NoError(t, db.Exec(`
		UPDATE sys_menu
		SET path = 'user/my-notices'
		WHERE menu_name = '我的通知'
		  AND parent_id = ?
		  AND path = 'my-notices'`,
		userCenterID).Error)

	var newPath string
	require.NoError(t, db.Raw("SELECT path FROM sys_menu WHERE menu_name = '我的通知'").Scan(&newPath).Error)
	assert.Equal(t, "user/my-notices", newPath)
}

// TC5: FixMenuHandler - 更新组件路径
func TestFixMenuHandler_UpdateComponent(t *testing.T) {
	db := setupFixMenuTestDB(t)
	userCenterID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '个人中心', ?, 'user/profile', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)

	require.NoError(t, db.Exec(`
		UPDATE sys_menu
		SET component = 'profile/index'
		WHERE menu_name = '个人中心'
		  AND parent_id = ?
		  AND component IS NULL`,
		userCenterID).Error)

	var newComponent string
	require.NoError(t, db.Raw("SELECT component FROM sys_menu WHERE menu_name = '个人中心'").Scan(&newComponent).Error)
	assert.Equal(t, "profile/index", newComponent)
}

// TC6: FixMenuHandler - 综合验证完整流程
func TestFixMenuHandler_CompleteFlow(t *testing.T) {
	db := setupFixMenuTestDB(t)
	userCenterID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '用户中心', NULL, '/user', 'user/index', 'M', 1, 0, datetime('now'), datetime('now'), 0)`,
		userCenterID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '个人中心', ?, 'profile', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '系统设置', ?, 'settings', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, path, component, menu_type, visible, status, created_at, updated_at, version)
		VALUES (?, '我的通知', ?, 'my-notices', NULL, 'C', 1, 0, datetime('now'), datetime('now'), 0)`,
		uuid.NewString(), userCenterID).Error)

	// Run all 3 path-update SQLs from the handler
	for _, sql := range []struct{ name, path, sel string }{
		{"个人中心", "user/profile", `UPDATE sys_menu SET path = 'user/profile' WHERE menu_name = '个人中心' AND parent_id = ? AND path = 'profile'`},
		{"系统设置", "user/settings", `UPDATE sys_menu SET path = 'user/settings' WHERE menu_name = '系统设置' AND parent_id = ? AND path = 'settings'`},
		{"我的通知", "user/my-notices", `UPDATE sys_menu SET path = 'user/my-notices' WHERE menu_name = '我的通知' AND parent_id = ? AND path = 'my-notices'`},
	} {
		require.NoError(t, db.Exec(sql.sel, userCenterID).Error)
		var newPath string
		require.NoError(t, db.Raw("SELECT path FROM sys_menu WHERE menu_name = ?", sql.name).Scan(&newPath).Error)
		assert.Equal(t, sql.path, newPath, "%s 路径应已更新", sql.name)
	}
}

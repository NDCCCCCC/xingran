// Package middleware 提供 P1-S4 (child menu permission inherit) 的回归测试
//
// 背景: 之前用户拥有任一子页面 (menu_type='C') 就自动获得父页面的全部 :list/:view 权限,
// 在精细化数据权限场景构成越权读取。
// P1 fix (commit 2b55e0d): checkUserPermission 只允许 menu_type='F' (按钮) 子菜单
// 的权限自动继承到父菜单,menu_type='C' (菜单/页面) 子菜单不再触发父菜单权限。
//
// 验证:
//   - 用户拥有 C 型子菜单权限时,父菜单权限检查不通过
//   - 用户拥有 F 型子菜单权限时,父菜单权限检查通过(向后兼容)
//   - 用户无任何子菜单权限时,父菜单权限检查不通过
package middleware

import (
	"testing"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestCoreWithSQLite 构造一个最小可用的 *core.Core,DB 指向 SQLite 内存库。
// 其他字段保持 nil — checkUserPermission 只使用 core.DB。
func newTestCoreWithSQLite(t *testing.T, setup func(*gorm.DB)) *core.Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 手工建表 — 避免 AutoMigrate 与 PostgreSQL 特有的 UUID 语法冲突
	createTables(t, gormDB)

	if setup != nil {
		setup(gormDB)
	}

	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
	}
}

// createTables 手工建表以匹配 sys_user_role / sys_role_menu / sys_menu / sys_role
// 用 sqlite 兼容的子集 (UUID 用 TEXT)。
func createTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sys_role (
			id TEXT PRIMARY KEY,
			role_name TEXT NOT NULL,
			role_key TEXT NOT NULL,
			role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1,
			menu_check_strictly BOOLEAN DEFAULT 1,
			dept_check_strictly BOOLEAN DEFAULT 1,
			status INTEGER DEFAULT 0,
			del_flag BOOLEAN DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_menu (
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
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sys_role_menu (
			role_id TEXT NOT NULL,
			menu_id TEXT NOT NULL,
			PRIMARY KEY (role_id, menu_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sys_user_role (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			PRIMARY KEY (user_id, role_id)
		)`,
	}
	for _, s := range stmts {
		assert.NoError(t, db.Exec(s).Error)
	}
}

// TestPermissionCheck_DoesNotInheritCTypeChild 验证 C 型子菜单不再继承到父菜单
//
// P1-S4 验收:
//   - 用户拥有 C 型子菜单 (menu_type='C') 的权限时,父菜单权限检查不通过
//   - 用户拥有 F 型子菜单 (menu_type='F') 的权限时,父菜单权限检查通过(向后兼容)
func TestPermissionCheck_DoesNotInheritCTypeChild(t *testing.T) {
	tests := []struct {
		name           string
		childType      models.MenuType
		grantChildPerm bool
		grantParentPerm bool
		expectPass     bool
	}{
		{
			name:           "C型子菜单有权限_父菜单权限检查应失败",
			childType:      models.MenuTypeMenu,
			grantChildPerm: true,
			grantParentPerm: false,
			expectPass:     false,
		},
		{
			name:           "F型子菜单有权限_父菜单权限检查应通过",
			childType:      models.MenuTypeButton,
			grantChildPerm: true,
			grantParentPerm: false,
			expectPass:     true,
		},
		{
			name:           "无任何子菜单权限_父菜单权限检查应失败",
			childType:      models.MenuTypeMenu,
			grantChildPerm: false,
			grantParentPerm: false,
			expectPass:     false,
		},
		{
			name:           "M型子菜单(目录)有权限_不应继承",
			childType:      models.MenuTypeDir,
			grantChildPerm: true,
			grantParentPerm: false,
			expectPass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New().String()
			roleID := uuid.New().String()
			parentMenuID := uuid.New().String()
			childMenuID := uuid.New().String()

			core := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
				// 1) 角色
				assert.NoError(t, gormDB.Exec(
					`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
					roleID, "test-role", "test_role",
				).Error)
				// 2) 用户-角色关联
				assert.NoError(t, gormDB.Exec(
					`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
					userID, roleID,
				).Error)
				// 3) 父菜单 (perms='system:menu:list')
				assert.NoError(t, gormDB.Exec(
					`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'M', ?, 0)`,
					parentMenuID, "父菜单", "system:menu:list",
				).Error)
				// 4) 子菜单
				childPerms := "system:menu:query"
				assert.NoError(t, gormDB.Exec(
					`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, ?, ?, ?, 0)`,
					childMenuID, "子菜单", parentMenuID, tt.childType, childPerms,
				).Error)
				// 5) 角色-菜单关联
				if tt.grantChildPerm {
					assert.NoError(t, gormDB.Exec(
						`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
						roleID, childMenuID,
					).Error)
				}
				if tt.grantParentPerm {
					assert.NoError(t, gormDB.Exec(
						`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
						roleID, parentMenuID,
					).Error)
				}
			})

			// 检查"父菜单权限 system:menu:list"是否对用户通过
			got := checkUserPermission(core, userID, "system:menu:list")
			assert.Equal(t, tt.expectPass, got, tt.name)
		})
	}
}

// TestPermissionCheck_ExactMatchStillWorks 验证精确权限匹配未被破坏
func TestPermissionCheck_ExactMatchStillWorks(t *testing.T) {
	userID := uuid.New().String()
	roleID := uuid.New().String()
	menuID := uuid.New().String()

	core := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
			roleID, "test-role", "test_role",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
			userID, roleID,
		).Error)
		// 直接给 system:user:list 权限 (M 型,无子菜单)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'M', ?, 0)`,
			menuID, "用户管理", "system:user:list",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
			roleID, menuID,
		).Error)
	})

	// 精确匹配应通过
	assert.True(t, checkUserPermission(core, userID, "system:user:list"),
		"exact match should pass")

	// 不存在的权限应失败
	assert.False(t, checkUserPermission(core, userID, "system:user:delete"),
		"non-existent permission should fail")
}

// TestPermissionCheck_ModuleLevelMatch 验证模块级权限匹配
func TestPermissionCheck_ModuleLevelMatch(t *testing.T) {
	userID := uuid.New().String()
	roleID := uuid.New().String()
	menuID := uuid.New().String()

	core := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
			roleID, "test-role", "test_role",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
			userID, roleID,
		).Error)
		// 用户拥有 system:user 模块权限 (无具体 perms 后缀)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'M', ?, 0)`,
			menuID, "用户模块", "system:user",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
			roleID, menuID,
		).Error)
	})

	// 模块级匹配应通过 (system:user:list 命中 system:user)
	assert.True(t, checkUserPermission(core, userID, "system:user:list"),
		"module-level match (system:user:list → system:user) should pass")
	assert.True(t, checkUserPermission(core, userID, "system:user:delete"),
		"module-level match (system:user:delete → system:user) should pass")
}

// TestPermissionCheck_StoppedMenuIgnored 验证停用菜单不参与权限匹配
func TestPermissionCheck_StoppedMenuIgnored(t *testing.T) {
	userID := uuid.New().String()
	roleID := uuid.New().String()
	menuID := uuid.New().String()

	core := newTestCoreWithSQLite(t, func(gormDB *gorm.DB) {
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role (id, role_name, role_key, status, del_flag) VALUES (?, ?, ?, 0, 0)`,
			roleID, "test-role", "test_role",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
			userID, roleID,
		).Error)
		// 菜单被停用 (status=1)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_menu (id, menu_name, parent_id, menu_type, perms, status) VALUES (?, ?, NULL, 'M', ?, 1)`,
			menuID, "已停用菜单", "system:stopped:list",
		).Error)
		assert.NoError(t, gormDB.Exec(
			`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
			roleID, menuID,
		).Error)
	})

	// 停用菜单的权限不应通过
	assert.False(t, checkUserPermission(core, userID, "system:stopped:list"),
		"stopped menu permission should not pass")
}

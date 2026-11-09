package migrations

// FIX-13 / OQ-7：migration_236 数据侧证据。
//
// 验证 5 个读码被建成 F 型按钮行（status=0, visible=0, parent 指向 live 父菜单），
// 且精准授权给持有父 C 菜单的角色；连跑两次幂等。
// 复用 migration_226 测试的 SQLite 夹具（freshSQLiteDBForMigrate226）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedOQ7Parent(db *gorm.DB, t *testing.T) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_menu (id, menu_name, menu_type, status, perms) VALUES (?, ?, 'C', 0, ?)`,
		"oq7-floor-parent", "楼层管理", "ops:floor:list",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role (id, role_name, status) VALUES (?, ?, 0)`, "oq7-role", "持楼层菜单角色",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, "oq7-role", "oq7-floor-parent",
	).Error)
}

func TestMigrate236OQ7ReadPermSeed_SeedsFiveCodes(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedOQ7Parent(db, t)

	require.NoError(t, Migrate236OQ7ReadPermSeed(db))

	for _, code := range actionPermissionFTypeReadCodes {
		var row struct {
			MenuType string
			Status   int
			Visible  int
			ParentID string
		}
		require.NoError(t, db.Raw(
			`SELECT menu_type, status, visible, parent_id FROM sys_menu WHERE perms = ?`, code,
		).Scan(&row).Error, "读码 %s 应落库", code)
		assert.Equal(t, "F", row.MenuType, "%s 应为 F 型按钮", code)
		assert.Equal(t, 0, row.Status, "%s status=0 启用", code)
		assert.Equal(t, 0, row.Visible, "%s visible=0 隐藏", code)
		assert.Equal(t, "oq7-floor-parent", row.ParentID, "%s 应挂到 live 父菜单", code)

		var granted int64
		require.NoError(t, db.Raw(
			`SELECT COUNT(*) FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id = m.id
			 WHERE rm.role_id = ? AND m.perms = ?`, "oq7-role", code,
		).Scan(&granted).Error)
		assert.Equal(t, int64(1), granted, "%s 应授权给持父菜单的角色", code)
	}
}

func TestMigrate236OQ7ReadPermSeed_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedOQ7Parent(db, t)

	require.NoError(t, Migrate236OQ7ReadPermSeed(db))
	var first int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&first).Error)

	require.NoError(t, Migrate236OQ7ReadPermSeed(db))
	var second int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&second).Error)
	assert.Equal(t, first, second, "连跑两次菜单行数不变（幂等）")

	var grants int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_role_menu`).Scan(&grants).Error)
	// 父菜单 1 行 + 5 个读码 = 6 条授权
	assert.Equal(t, int64(6), grants, "授权行不重复")
}

func TestMigrate236OQ7ReadPermSeed_NoParentIsNoop(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	// 不建 live 父菜单 → 优雅降级，零新增行
	require.NoError(t, Migrate236OQ7ReadPermSeed(db))
	var count int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&count).Error)
	assert.Equal(t, int64(0), count, "父菜单缺失时不创建孤立按钮行")
}

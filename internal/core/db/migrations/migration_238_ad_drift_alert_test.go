package migrations

// Phase 144 · migration_238 数据侧证据（ALRT-01/02/03）。
//
// 覆盖：
//  1. C 型页面菜单 + 4 个 F 型按钮 + 精准授权 + 9 行 sys_config seed；
//  2. 连跑两次幂等（菜单 / 授权 / config 行数不重复）；
//  3. 「系统管理」根缺失 → 优雅降级（菜单零新增，config 仍 seed，非阻断）；
//  4. 双方言：sqlite 全量；PG 仅在 TEST_PG_DSN 设置时跑并显式 skip 可观测。
//
// 复用 migration_226 测试的 freshSQLiteDBForMigrate226 夹具（sys_menu /
// sys_role / sys_role_menu），另 AutoMigrate sys_config。

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func freshSQLiteDBForMigrate238(t *testing.T) *gorm.DB {
	t.Helper()
	db := freshSQLiteDBForMigrate226(t)
	require.NoError(t, db.AutoMigrate(&models.Config{}))
	return db
}

// seed238SystemRoot 插入 canonicalSystemRootID「系统管理」根 + 持有它的角色。
func seed238SystemRoot(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_menu (id, menu_name, menu_type, status, perms) VALUES (?, ?, 'M', 0, NULL)`,
		canonicalSystemRootID, "系统管理",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role (id, role_name, status) VALUES (?, ?, 0)`, "r238", "持系统管理根角色",
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, "r238", canonicalSystemRootID,
	).Error)
}

func TestMigrate238ADDriftAlert_SeedsMenuPermsAndConfig(t *testing.T) {
	db := freshSQLiteDBForMigrate238(t)
	seed238SystemRoot(t, db)

	require.NoError(t, Migrate238ADDriftAlert(db))

	// C 型页面菜单
	var page struct {
		MenuType  string
		Component string
		Perms     string
		ParentID  string
	}
	require.NoError(t, db.Raw(
		`SELECT menu_type, component, perms, parent_id FROM sys_menu WHERE menu_name = ?`, "同步漂移告警",
	).Scan(&page).Error)
	assert.Equal(t, "C", page.MenuType)
	assert.Equal(t, "system/ad-drift-alert/index", page.Component)
	assert.Equal(t, "system:adDriftAlert", page.Perms)
	assert.Equal(t, canonicalSystemRootID, page.ParentID)

	// 4 个 F 型按钮 + 精准授权
	for _, code := range adDriftAlertFTypeSeedCodes {
		var row struct {
			MenuType string
			Status   int
			Visible  int
		}
		require.NoError(t, db.Raw(
			`SELECT menu_type, status, visible FROM sys_menu WHERE perms = ?`, code,
		).Scan(&row).Error, "权限码 %s 应落库", code)
		assert.Equal(t, "F", row.MenuType, "%s 应为 F 型按钮", code)
		assert.Equal(t, 0, row.Status, "%s status=0 启用", code)
		assert.Equal(t, 0, row.Visible, "%s visible=0 隐藏", code)

		var granted int64
		require.NoError(t, db.Raw(
			`SELECT COUNT(*) FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id = m.id
			 WHERE rm.role_id = ? AND m.perms = ?`, "r238", code,
		).Scan(&granted).Error)
		assert.Equal(t, int64(1), granted, "%s 应授权给持父菜单的角色", code)
	}

	// 9 行 sys_config seed（逐值）
	for _, seed := range adDriftAlertConfigSeeds {
		var value string
		require.NoError(t, db.Raw(
			`SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL`, seed.key,
		).Scan(&value).Error, "sys_config %s 应落库", seed.key)
		assert.Equal(t, seed.value, value, "sys_config %s 默认值", seed.key)
	}
	var configCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_config`).Scan(&configCount).Error)
	assert.Equal(t, int64(9), configCount)
}

func TestMigrate238ADDriftAlert_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate238(t)
	seed238SystemRoot(t, db)

	require.NoError(t, Migrate238ADDriftAlert(db))
	var menuFirst, grantFirst, configFirst int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&menuFirst).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_role_menu`).Scan(&grantFirst).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_config`).Scan(&configFirst).Error)

	require.NoError(t, Migrate238ADDriftAlert(db))
	var menuSecond, grantSecond, configSecond int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&menuSecond).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_role_menu`).Scan(&grantSecond).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_config`).Scan(&configSecond).Error)

	assert.Equal(t, menuFirst, menuSecond, "连跑两次菜单行数不变")
	assert.Equal(t, grantFirst, grantSecond, "连跑两次授权行数不变")
	assert.Equal(t, configFirst, configSecond, "连跑两次 config 行数不变（均为 9）")
	assert.Equal(t, int64(9), configSecond)
}

func TestMigrate238ADDriftAlert_NoRootStillSeedsConfig(t *testing.T) {
	db := freshSQLiteDBForMigrate238(t)
	// 不建「系统管理」根 → 菜单/授权优雅降级，config seed 照常
	require.NoError(t, Migrate238ADDriftAlert(db))

	var menuCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_menu`).Scan(&menuCount).Error)
	assert.Equal(t, int64(0), menuCount, "根缺失不得创建孤立菜单行")

	var configCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_config`).Scan(&configCount).Error)
	assert.Equal(t, int64(9), configCount, "config seed 不依赖菜单树")
}

// TestMigrate238ADDriftAlert_PostgresDialect PG 双方言：仅当 TEST_PG_DSN 显式
// 设置时执行；否则显式 t.Skip（可观测，不伪造 PG 覆盖）。
func TestMigrate238ADDriftAlert_PostgresDialect(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN 未设置，PG 双方言用例显式跳过（sqlite 用例已全量覆盖逻辑）")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	// 只做「纯 GORM / db.Exec 无 PG 专属语法」的冒烟：用独立 schema 太重，
	// 这里断言迁移函数可执行（真实生产库不连，见 PLAN 禁止项）。
	require.NoError(t, Migrate238ADDriftAlert(db))
	t.Log("PG 方言 migration 238 执行通过（无 ::text 等专属语法）")
}

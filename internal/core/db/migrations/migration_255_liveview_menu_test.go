package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Phase 155 / LIVEVIEW-04：migration 255 的菜单播种契约。
//
//   TestMigration255_CreatesLiveMenu      全新库（父菜单在）→ 建出 C 型菜单
//   TestMigration255_Idempotent           重复执行不重复建、不覆盖
//   TestMigration255_SkipsWhenParentMissing 父菜单缺失 → 跳过且不报错（非阻断）
//   TestMigration255_NoPermissionCodes    本迁移**不扩权限面**（不创建 F 型行）
// ============================================================================

// freshP255DB 每个用例一个**独立**内存库。
//
// 不能用既有的 freshSQLiteDB：它用 "file::memory:?cache=shared"，同进程内所有
// 用例共享同一库 → 上一用例播下的 liveViewMenuID 会串到下一用例（UNIQUE 冲突
// 与「父缺失」用例误读到父行）。这里用 sqlite 的独立内存库（"file:<name>?mode=memory"）。
func freshP255DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	require.NoError(t, db.AutoMigrate(&models.Menu{}, &models.Role{}, &models.RoleMenu{}), "AutoMigrate")
	// 每个用例结束时关闭连接，避免同名库残留影响同包后续用例。
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedParentMenu 建出「RPA 管理」父菜单（复刻 menu_catalog_seed.sql:193 的关键字段）。
func seedParentMenu(t *testing.T, db *gorm.DB) {
	t.Helper()
	parentRoot := canonicalOpsRootID
	path := "rpa"
	comp := "pages/operations/rpa/index"
	parentID := liveViewParent
	m := models.Menu{
		MenuName:  "RPA 管理",
		ParentID:  &parentRoot,
		OrderNum:  7,
		Path:      &path,
		Component: &comp,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
	}
	m.ID = parentID
	require.NoError(t, db.Create(&m).Error)
}

// TestMigration255_CreatesLiveMenu 全新库上建出实时画面 C 型菜单。
func TestMigration255_CreatesLiveMenu(t *testing.T) {
	db := freshP255DB(t)
	seedParentMenu(t, db)

	require.NoError(t, Migrate255LiveViewMenu(db))

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", liveViewMenuID).Count(&count).Error)
	require.EqualValues(t, 1, count, "实时画面菜单应被创建且只有一行")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", liveViewMenuID).First(&menu).Error)
	require.Equal(t, liveViewMenuName, menu.MenuName)
	require.Equal(t, liveViewComp, *menu.Component, "component 必须是 pages/operations/rpa/live/index（走 glob 自动加载）")
	require.Equal(t, liveViewPath, *menu.Path)
	require.Equal(t, liveViewParent, *menu.ParentID, "父菜单必须是「RPA 管理」（继承既有授权链）")
	require.Equal(t, models.MenuTypeMenu, menu.MenuType, "必须是 C 型页面菜单，不是按钮")
	require.Equal(t, models.VisibleShow, menu.Visible)
	require.Nil(t, menu.Perms, "页面菜单不占权限码（复用 ops:rpa:execution:list）")
}

// TestMigration255_Idempotent 重复执行不重复建、不覆盖管理员改动。
func TestMigration255_Idempotent(t *testing.T) {
	db := freshP255DB(t)
	seedParentMenu(t, db)
	require.NoError(t, Migrate255LiveViewMenu(db))

	// 管理员改了排序 → 再跑迁移不得覆盖。
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", liveViewMenuID).
		Update("order_num", 99).Error)

	require.NoError(t, Migrate255LiveViewMenu(db))

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", liveViewMenuID).Count(&count).Error)
	require.EqualValues(t, 1, count, "重复执行不得重复建")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", liveViewMenuID).First(&menu).Error)
	require.Equal(t, 99, menu.OrderNum, "迁移不得覆盖管理员改过的排序（幂等先查后建）")
}

// TestMigration255_SkipsWhenParentMissing 父菜单缺失 → 跳过且返回 nil（非阻断）。
func TestMigration255_SkipsWhenParentMissing(t *testing.T) {
	db := freshP255DB(t)
	// 不 seed 父菜单。
	require.NoError(t, Migrate255LiveViewMenu(db), "父菜单缺失时应返回 nil（非阻断，下次启动重试）")

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", liveViewMenuID).Count(&count).Error)
	require.EqualValues(t, 0, count, "父菜单缺失时不得创建孤儿菜单")
}

// TestMigration255_NoPermissionCodes 本迁移不扩权限面（无 F 型行、无新 perms）。
func TestMigration255_NoPermissionCodes(t *testing.T) {
	db := freshP255DB(t)
	seedParentMenu(t, db)
	require.NoError(t, Migrate255LiveViewMenu(db))

	var buttons int64
	require.NoError(t, db.Model(&models.Menu{}).
		Where("menu_type = ?", models.MenuTypeButton).Count(&buttons).Error)
	require.EqualValues(t, 0, buttons, "migration 255 不得创建 F 型按钮（LIVEVIEW-05 复用执行读码）")
}

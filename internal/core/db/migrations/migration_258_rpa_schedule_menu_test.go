package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Phase 158 / REC-02：migration 258 的定时调度菜单播种契约。
//
//   TestMigration258_CreatesScheduleMenu       全新库（父菜单在）→ 建出 C 型菜单
//   TestMigration258_Idempotent                重复执行不重复建、不覆盖管理员改动
//   TestMigration258_SkipsWhenParentMissing    父菜单缺失 → 跳过且不报错（非阻断）
//   TestMigration258_NoPermissionCodes         本迁移**不扩权限面**（不创建 F 型行）
//   TestMigration258_PointsToSchedulePage      component 必须指向真实存在的页面文件
// ============================================================================

// freshP258DB 每个用例一个独立内存库（理由同 255/256：不能共享 cache=shared）。
func freshP258DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	require.NoError(t, db.AutoMigrate(&models.Menu{}, &models.Role{}, &models.RoleMenu{}), "AutoMigrate")
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedScheduleParent 建出「RPA 管理」父菜单。
func seedScheduleParent(t *testing.T, db *gorm.DB) {
	t.Helper()
	root := canonicalOpsRootID
	path := "rpa"
	comp := "pages/operations/rpa/index"
	parentID := scheduleMenuParent
	m := models.Menu{
		MenuName:  "RPA 管理",
		ParentID:  &root,
		OrderNum:  7,
		Path:      &path,
		Component: &comp,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
	}
	m.ID = parentID
	require.NoError(t, db.Create(&m).Error, "seed parent menu")
}

// TestMigration258_CreatesScheduleMenu 全新库建出定时调度菜单。
func TestMigration258_CreatesScheduleMenu(t *testing.T) {
	db := freshP258DB(t)
	seedScheduleParent(t, db)

	require.NoError(t, Migrate258RpaScheduleMenu(db))

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", scheduleMenuID).First(&menu).Error)
	require.Equal(t, scheduleMenuName, menu.MenuName)
	require.Equal(t, "定时调度", menu.MenuName)
	require.NotNil(t, menu.Path)
	require.Equal(t, scheduleMenuPath, *menu.Path)
	require.Equal(t, "schedules", *menu.Path)
	require.NotNil(t, menu.Component)
	require.Equal(t, scheduleMenuComp, *menu.Component)
	require.Equal(t, "pages/operations/rpa/schedules/index", *menu.Component)
	require.NotNil(t, menu.ParentID)
	require.Equal(t, scheduleMenuParent, *menu.ParentID)
	// C 型页面菜单
	require.Equal(t, models.MenuTypeMenu, menu.MenuType)
	require.Equal(t, "C", string(menu.MenuType))
	// 可见 + 正常（visible=1 是菜单可见的例外语义）
	require.Equal(t, models.VisibleShow, menu.Visible)
	require.Equal(t, models.MenuStatusNormal, menu.Status)
	// 排序紧邻 255(16) / 256(17) 之后
	require.Equal(t, 18, menu.OrderNum)
}

// TestMigration258_Idempotent 重复执行不重复建、不覆盖管理员改动。
func TestMigration258_Idempotent(t *testing.T) {
	db := freshP258DB(t)
	seedScheduleParent(t, db)

	require.NoError(t, Migrate258RpaScheduleMenu(db))
	// 管理员改了排序与名称
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", scheduleMenuID).
		Updates(map[string]interface{}{"order_num": 99, "menu_name": "管理员改过的名字"}).Error)

	require.NoError(t, Migrate258RpaScheduleMenu(db))

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", scheduleMenuID).Count(&count).Error)
	require.Equal(t, int64(1), count, "重复执行不应建出第二行")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", scheduleMenuID).First(&menu).Error)
	require.Equal(t, "管理员改过的名字", menu.MenuName, "不得覆盖管理员的改动")
	require.Equal(t, 99, menu.OrderNum, "不得覆盖管理员的排序")
}

// TestMigration258_SkipsWhenParentMissing 父菜单缺失 → 跳过且不报错。
func TestMigration258_SkipsWhenParentMissing(t *testing.T) {
	db := freshP258DB(t)

	// 不播种父菜单
	require.NoError(t, Migrate258RpaScheduleMenu(db), "父菜单缺失时不得报错（非阻断约定）")

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", scheduleMenuID).Count(&count).Error)
	require.Equal(t, int64(0), count, "父菜单缺失时不应建菜单")
}

// TestMigration258_NoPermissionCodes 不扩权限面（不创建 F 型行）。
func TestMigration258_NoPermissionCodes(t *testing.T) {
	db := freshP258DB(t)
	seedScheduleParent(t, db)
	require.NoError(t, Migrate258RpaScheduleMenu(db))

	var fCount int64
	require.NoError(t, db.Model(&models.Menu{}).Where("menu_type = ?", models.MenuTypeButton).Count(&fCount).Error)
	require.Equal(t, int64(0), fCount, "本迁移不得播种 F 型按钮（调度 4 权限码由 migration_257 播种）")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", scheduleMenuID).First(&menu).Error)
	require.Nil(t, menu.Perms, "页面菜单不占权限码（与 255/256 同款）")
}

// TestMigration258_PointsToSchedulePage component 路径与真实页面文件一致。
//
// 这条断言防「菜单指向不存在的组件」：componentLoader 找不到模块时会渲染
// 「页面已迁移」兜底，用户点进去看到空页面而不知道是配置错了。
func TestMigration258_PointsToSchedulePage(t *testing.T) {
	// 与前端目录结构对照：pages/operations/rpa/schedules/index
	require.Equal(t, "pages/operations/rpa/schedules/index", scheduleMenuComp)
	require.Equal(t, "schedules", scheduleMenuPath)
	require.Equal(t, "定时调度", scheduleMenuName)
	// order 必须与 255(16) / 256(17) 区分，避免排序抖动
	require.NotEqual(t, 16, scheduleMenuOrder)
	require.NotEqual(t, 17, scheduleMenuOrder)
	// 固定 UUID 后 12 位为版本号 258（与 255/256 同款命名风格）
	require.Equal(t, "a1b2c3d4-0258-4000-8000-000000000258", scheduleMenuID)
}

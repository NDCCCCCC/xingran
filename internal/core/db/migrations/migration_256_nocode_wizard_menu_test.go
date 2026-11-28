package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Phase 156 / NOCODE-04：migration 256 的菜单播种契约。
//
//   TestMigration256_CreatesWizardMenu        全新库（父菜单在）→ 建出 C 型菜单
//   TestMigration256_Idempotent               重复执行不重复建、不覆盖管理员改动
//   TestMigration256_SkipsWhenParentMissing   父菜单缺失 → 跳过且不报错（非阻断）
//   TestMigration256_NoPermissionCodes        本迁移**不扩权限面**（不创建 F 型行）
//   TestMigration256_PointsToWizardPage       component 必须指向真实存在的页面文件
// ============================================================================

// freshP256DB 每个用例一个独立内存库（理由同 255：不能共享 cache=shared）。
func freshP256DB(t *testing.T) *gorm.DB {
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

// seedWizardParent 建出「RPA 管理」父菜单。
func seedWizardParent(t *testing.T, db *gorm.DB) {
	t.Helper()
	root := canonicalOpsRootID
	path := "rpa"
	comp := "pages/operations/rpa/index"
	parentID := wizardParent
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

// TestMigration256_CreatesWizardMenu 全新库建出向导菜单。
func TestMigration256_CreatesWizardMenu(t *testing.T) {
	db := freshP256DB(t)
	seedWizardParent(t, db)

	require.NoError(t, Migrate256NocodeWizardMenu(db))

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", wizardMenuID).First(&menu).Error)
	require.Equal(t, wizardMenuName, menu.MenuName)
	require.NotNil(t, menu.Path)
	require.Equal(t, wizardPath, *menu.Path)
	require.NotNil(t, menu.Component)
	require.Equal(t, wizardComp, *menu.Component)
	require.NotNil(t, menu.ParentID)
	require.Equal(t, wizardParent, *menu.ParentID)
	// C 型页面菜单
	require.Equal(t, models.MenuTypeMenu, menu.MenuType)
	// 可见 + 正常（visible=1 是菜单可见的例外语义）
	require.Equal(t, models.VisibleShow, menu.Visible)
	require.Equal(t, models.MenuStatusNormal, menu.Status)
}

// TestMigration256_Idempotent 重复执行不重复建、不覆盖管理员改动。
func TestMigration256_Idempotent(t *testing.T) {
	db := freshP256DB(t)
	seedWizardParent(t, db)

	require.NoError(t, Migrate256NocodeWizardMenu(db))
	// 管理员改了排序与名称
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", wizardMenuID).
		Updates(map[string]interface{}{"order_num": 99, "menu_name": "管理员改过的名字"}).Error)

	require.NoError(t, Migrate256NocodeWizardMenu(db))

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", wizardMenuID).Count(&count).Error)
	require.Equal(t, int64(1), count, "重复执行不应建出第二行")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", wizardMenuID).First(&menu).Error)
	require.Equal(t, "管理员改过的名字", menu.MenuName, "不得覆盖管理员的改动")
	require.Equal(t, 99, menu.OrderNum, "不得覆盖管理员的排序")
}

// TestMigration256_SkipsWhenParentMissing 父菜单缺失 → 跳过且不报错。
func TestMigration256_SkipsWhenParentMissing(t *testing.T) {
	db := freshP256DB(t)

	// 不播种父菜单
	require.NoError(t, Migrate256NocodeWizardMenu(db), "父菜单缺失时不得报错（非阻断约定）")

	var count int64
	require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", wizardMenuID).Count(&count).Error)
	require.Equal(t, int64(0), count, "父菜单缺失时不应建菜单")
}

// TestMigration256_NoPermissionCodes 不扩权限面（不创建 F 型行）。
func TestMigration256_NoPermissionCodes(t *testing.T) {
	db := freshP256DB(t)
	seedWizardParent(t, db)
	require.NoError(t, Migrate256NocodeWizardMenu(db))

	var fCount int64
	require.NoError(t, db.Model(&models.Menu{}).Where("menu_type = ?", models.MenuTypeButton).Count(&fCount).Error)
	require.Equal(t, int64(0), fCount, "本迁移不得播种 F 型按钮（复用既有 ops:rpa:task:* 权限码）")

	var menu models.Menu
	require.NoError(t, db.Where("id = ?", wizardMenuID).First(&menu).Error)
	require.Nil(t, menu.Perms, "页面菜单不占权限码（与 255 同款）")
}

// TestMigration256_PointsToWizardPage component 路径与真实页面文件一致。
//
// 这条断言防「菜单指向不存在的组件」：componentLoader 找不到模块时会渲染
// 「页面已迁移」兜底，用户点进去看到空页面而不知道是配置错了。
func TestMigration256_PointsToWizardPage(t *testing.T) {
	// 与前端目录结构对照：pages/operations/rpa/wizard/index
	require.Equal(t, "pages/operations/rpa/wizard/index", wizardComp)
	require.Equal(t, "wizard", wizardPath)
	require.Equal(t, "新建自动化任务", wizardMenuName)
	// order 必须与 155 的 live 菜单（16）区分，避免排序抖动
	require.NotEqual(t, 16, wizardMenuOrder)
}

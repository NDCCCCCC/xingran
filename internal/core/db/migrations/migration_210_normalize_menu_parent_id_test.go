package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate210 构建内存 SQLite 库并 AutoMigrate sys_menu。
func freshSQLiteDBForMigrate210(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seedMenuWithParent 以指定 parent_id 创建菜单,返回 id。
func seedMenuWithParent(t *testing.T, db *gorm.DB, name, parentID string) string {
	t.Helper()
	menu := models.Menu{
		MenuName: name,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	if parentID != "" {
		menu.ParentID = &parentID
	}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("seed menu %s: %v", name, err)
	}
	return menu.ID
}

// getParentID 读取菜单 parent_id(nil 以 ok=false 表示,与 GORM 零值语义一致)。
func getParentID(t *testing.T, db *gorm.DB, id string) *string {
	t.Helper()
	var menu models.Menu
	if err := db.Where("id = ?", id).First(&menu).Error; err != nil {
		t.Fatalf("get menu %s: %v", id, err)
	}
	return menu.ParentID
}

// TestMigrate210NormalizeParentIdempotently 第一次将 '0' 归 NULL,第二次幂等。
func TestMigrate210NormalizeParentIdempotently(t *testing.T) {
	db := freshSQLiteDBForMigrate210(t)

	rootID := seedMenuWithParent(t, db, "Root", "")
	childID := seedMenuWithParent(t, db, "Child", rootID)
	dirtyID := seedMenuWithParent(t, db, "Dirty", "0")

	changed, err := Migrate210NormalizeMenuParentID(db)
	require.NoError(t, err)
	require.True(t, changed, "first run should change the dirty row")

	// 根与正常子节点不应被触碰
	require.Nil(t, getParentID(t, db, rootID), "root parent should stay nil")
	parentOfChild := getParentID(t, db, childID)
	require.NotNil(t, parentOfChild, "child parent should stay non-nil")
	require.Equal(t, rootID, *parentOfChild, "child parent should stay root")

	// 脏行已归 NULL
	require.Nil(t, getParentID(t, db, dirtyID), "dirty row parent_id='0' should become nil")

	// 第二次执行幂等
	changed, err = Migrate210NormalizeMenuParentID(db)
	require.NoError(t, err)
	require.False(t, changed, "second run should be idempotent")
}

// TestMigrate210NoDirtyRows 无 '0' 行时 changed=false 且不报错。
func TestMigrate210NoDirtyRows(t *testing.T) {
	db := freshSQLiteDBForMigrate210(t)
	_ = seedMenuWithParent(t, db, "Root", "")
	_ = seedMenuWithParent(t, db, "Child", "00000000-0000-0000-0000-000000000001")

	changed, err := Migrate210NormalizeMenuParentID(db)
	require.NoError(t, err)
	require.False(t, changed, "no dirty rows should produce changed=false")
}

// TestMigrate210NoOrphansAfterMigration 迁移后不存在孤儿节点:
// 所有非 NULL parent_id 均指向存在的菜单行。
func TestMigrate210NoOrphansAfterMigration(t *testing.T) {
	db := freshSQLiteDBForMigrate210(t)

	rootID := seedMenuWithParent(t, db, "Root", "")
	_ = seedMenuWithParent(t, db, "Child", rootID)
	dirtyID := seedMenuWithParent(t, db, "Dirty", "0")

	_, err := Migrate210NormalizeMenuParentID(db)
	require.NoError(t, err)

	var orphanCount int64
	err = db.Raw(`
		SELECT COUNT(*) FROM sys_menu m
		WHERE m.parent_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM sys_menu p WHERE p.id = m.parent_id)
	`).Scan(&orphanCount).Error
	require.NoError(t, err)
	require.Zero(t, orphanCount, "migration should leave zero orphan nodes")

	// 原脏行现在作为根节点出现,不再丢失
	var dirty models.Menu
	require.NoError(t, db.Where("id = ?", dirtyID).First(&dirty).Error)
	require.Nil(t, dirty.ParentID, "dirty row should become root after normalization")
}

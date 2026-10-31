package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate229 构建内存 SQLite 库（仅 sys_menu 单表最小表集），
// seed 网络设备 M 根 + devices 参照子菜单（parent 反查锚点）。
func freshSQLiteDBForMigrate229(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	networkRoot := "0013f129-3ec0-4e55-8ffc-25d97b20c37b"
	networkRootPath := "network"
	rootMenu := models.Menu{
		MenuName: "网络设备",
		MenuType: models.MenuTypeDir,
		Path:     &networkRootPath,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	rootMenu.ID = networkRoot

	parentID := networkRoot
	devicesPath := "devices"
	devicesComponent := "network/devices/index"
	devices := models.Menu{
		MenuName:  "设备管理",
		ParentID:  &parentID,
		Path:      &devicesPath,
		Component: &devicesComponent,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
	}
	devices.ID = "80b06462-e4d2-4c6b-8e43-01bdb9260ace"

	if err := db.Create(&[]models.Menu{rootMenu, devices}).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	return db
}

// TestMigrate229SeedsAuditMenu 无菜单 → 按网络设备参照反查父补种; 重复执行幂等。
func TestMigrate229SeedsAuditMenu(t *testing.T) {
	db := freshSQLiteDBForMigrate229(t)

	for run := 1; run <= 2; run++ {
		if err := Migrate229PortWriteAuditMenu(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}

	var menus []models.Menu
	if err := db.Where("component = ?", "network/port-audits/index").Find(&menus).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(menus) != 1 {
		t.Fatalf("期望恰好 1 条菜单, 实际 %d（幂等破坏）", len(menus))
	}
	menu := menus[0]
	if menu.Perms == nil || *menu.Perms != "network:port:write" {
		t.Fatalf("perms 期望 network:port:write, 实际 %v", menu.Perms)
	}
	if menu.ParentID == nil || *menu.ParentID != "0013f129-3ec0-4e55-8ffc-25d97b20c37b" {
		t.Fatalf("parent 期望网络设备根, 实际 %v", menu.ParentID)
	}
	if menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal {
		t.Fatalf("visible=%d status=%d, 期望 1/0", menu.Visible, menu.Status)
	}
}

// TestMigrate229SkipsWhenNoAnchor 无网络设备参照菜单 → 降级跳过不报错。
func TestMigrate229SkipsWhenNoAnchor(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	if err := Migrate229PortWriteAuditMenu(db); err != nil {
		t.Fatalf("无参照时应降级跳过, 实际报错: %v", err)
	}
	var count int64
	if err := db.Model(&models.Menu{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("无参照时不应插入菜单, 实际 %d 条", count)
	}
}

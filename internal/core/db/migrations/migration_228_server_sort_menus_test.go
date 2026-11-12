package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate228 构建内存 SQLite 库（仅 sys_menu 单表最小表集），
// 并 seed migration 228 依赖的参照节点：运维管理根 / 值班管理根 / 网络设备根 +
// network/devices/index 子菜单（parent 反查锚点）。
func freshSQLiteDBForMigrate228(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	opsRoot := "c50b5b01-fdbc-4821-a3e8-2e2178339b20"
	dutyRoot := "e7d2962c-43c1-4302-be00-ff8e4e1878fb"
	networkRoot := "0013f129-3ec0-4e55-8ffc-25d97b20c37b"
	networkRootPath := "network"
	dir := models.MenuTypeDir
	m := func(id, name, parent string, order int) models.Menu {
		menu := models.Menu{
			MenuName: name,
			MenuType: dir,
			Visible:  models.VisibleShow,
			Status:   models.MenuStatusNormal,
			OrderNum: order,
		}
		menu.ID = id
		if parent != "" {
			menu.ParentID = &parent
		}
		return menu
	}
	devicesPath := "devices"
	devicesComponent := "network/devices/index"
	devices := models.Menu{
		MenuName:  "设备管理",
		ParentID:  &networkRoot,
		Path:      &devicesPath,
		Component: &devicesComponent,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		OrderNum:  2,
	}
	devices.ID = "80b06462-e4d2-4c6b-8e43-01bdb9260ace"

	seeds := []models.Menu{
		m(opsRoot, "运维管理", "", 1),
		m(dutyRoot, "值班管理", opsRoot, 4),
		m(networkRoot, "网络设备", "", 3),
	}
	// 网络设备根的 path 需要（M 节点也有 path）
	for i := range seeds {
		if seeds[i].ID == networkRoot {
			seeds[i].Path = &networkRootPath
		}
	}
	seeds = append(seeds, devices)
	if err := db.Create(&seeds).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	return db
}

// get228Menu 按 component 读取菜单（不存在时 ok=false）。
func get228Menu(t *testing.T, db *gorm.DB, component string) (models.Menu, bool) {
	t.Helper()
	var menu models.Menu
	if err := db.Where("component = ?", component).First(&menu).Error; err != nil {
		return models.Menu{}, false
	}
	return menu, true
}

// TestMigrate228DoesNotActivateDutyPoolMenu 值班池：228 修订后不再激活为可见可用。
//
// 锁定：无论 duty/pools 行原本是「隐藏停用」还是「被 228 旧逻辑改过的可见可用」，
// 228 执行后都不产生把该行改成 visible=1/status=0 的 UPDATE —— 行状态逐列保持不变
// （visible / status / meta.Hidden 都不被 228 触碰）。存量行归位由 migration_249 负责。
func TestMigrate228DoesNotActivateDutyPoolMenu(t *testing.T) {
	cases := []struct {
		name    string
		visible models.VisibleType
		status  models.MenuStatus
		hidden  bool
	}{
		{"隐藏停用基线(seed 0/1)", models.VisibleHidden, models.MenuStatusStop, true},
		{"被旧228改成可见可用(1/0)", models.VisibleShow, models.MenuStatusNormal, false},
		{"幽灵态(visible=0/status=0)", models.VisibleHidden, models.MenuStatusNormal, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := freshSQLiteDBForMigrate228(t)

			path := "pools"
			component := "duty/pools/index"
			perms := "ops:duty:pools:view"
			seed := models.Menu{
				MenuName:  "值班池管理",
				ParentID:  strPtr228("e7d2962c-43c1-4302-be00-ff8e4e1878fb"),
				Path:      &path,
				Component: &component,
				MenuType:  models.MenuTypeMenu,
				Visible:   tc.visible,
				Status:    tc.status,
				Perms:     &perms,
				Meta:      &models.MenuMeta{Title: "值班池管理", Hidden: tc.hidden},
			}
			seed.ID = "c4a5ff39-96ad-48d5-b1c1-543ef88f8373"
			if err := db.Create(&seed).Error; err != nil {
				t.Fatalf("seed: %v", err)
			}
			// models.Menu.Visible 有 `gorm:"default:1"`；db.Create 遇到零值(visible=0)
			// 会套用该默认值，导致 seed 落库即变 visible=1，把「228 未改动」误判成
			// 「228 激活」。这里用 Updates(map) 强制写入用例期望的初值（map 不受 default 影响）。
			if err := db.Model(&models.Menu{}).Where("id = ?", seed.ID).
				Updates(map[string]interface{}{
					"visible": tc.visible,
					"status":  tc.status,
					"meta":    &models.MenuMeta{Title: "值班池管理", Hidden: tc.hidden},
				}).Error; err != nil {
				t.Fatalf("force seed state: %v", err)
			}

			for run := 1; run <= 2; run++ {
				if err := Migrate228ServerSortMenus(db); err != nil {
					t.Fatalf("run %d: %v", run, err)
				}
				menu, ok := get228Menu(t, db, component)
				if !ok {
					t.Fatalf("run %d: 值班池菜单丢失（228 不应删除行）", run)
				}
				// 228 完全不碰 duty/pools：三列逐项与初始态一致
				if menu.Visible != tc.visible || menu.Status != tc.status {
					t.Fatalf("run %d: 228 改动了 duty/pools: visible=%d status=%d, 期望保持 %d/%d",
						run, menu.Visible, menu.Status, tc.visible, tc.status)
				}
				if menu.Meta == nil || menu.Meta.Hidden != tc.hidden {
					t.Fatalf("run %d: 228 改动了 meta.Hidden: got %v, want %v",
						run, menu.Meta, tc.hidden)
				}
			}
		})
	}
}

// TestMigrate228KeepsDutySchedulesIdempotentFix 排班：228 仍保留幂等保底（可见可用）。
func TestMigrate228KeepsDutySchedulesIdempotentFix(t *testing.T) {
	db := freshSQLiteDBForMigrate228(t)

	component := "duty/schedules/index"
	path := "schedules"
	perms := "ops:duty:schedule:list"
	seed := models.Menu{
		MenuName:  "排班管理",
		ParentID:  strPtr228("e7d2962c-43c1-4302-be00-ff8e4e1878fb"),
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleHidden, // 存量脏数据
		Status:    models.MenuStatusStop,
		Perms:     &perms,
	}
	seed.ID = "d2280000-0000-4000-8000-000000000001"
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	for run := 1; run <= 2; run++ {
		if err := Migrate228ServerSortMenus(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		menu, ok := get228Menu(t, db, component)
		if !ok {
			t.Fatalf("run %d: 排班菜单丢失", run)
		}
		if menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal {
			t.Fatalf("run %d: 排班 visible=%d status=%d, 期望 1/0", run, menu.Visible, menu.Status)
		}
	}
}

// TestMigrate228NormalizesKnowledgeArticleComponent 知识库文章：legacy 前导斜杠
// component → 标准化为无斜杠并确保可见。
func TestMigrate228NormalizesKnowledgeArticleComponent(t *testing.T) {
	db := freshSQLiteDBForMigrate228(t)

	legacy := "/knowledge/articles/index"
	perms := "ops:knowledge:article:list"
	seed := models.Menu{
		MenuName:  "知识库文章",
		ParentID:  strPtr228("c50b5b01-fdbc-4821-a3e8-2e2178339b20"),
		Path:      strPtr228("knowledge/articles"),
		Component: &legacy,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := Migrate228ServerSortMenus(db); err != nil {
		t.Fatalf("run: %v", err)
	}

	if _, ok := get228Menu(t, db, "/knowledge/articles/index"); ok {
		t.Fatalf("legacy component 应被改写")
	}
	menu, ok := get228Menu(t, db, "knowledge/articles/index")
	if !ok {
		t.Fatalf("标准化 component 未找到")
	}
	if menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal {
		t.Fatalf("visible=%d status=%d, 期望 1/0", menu.Visible, menu.Status)
	}
}

// TestMigrate228SeedsKnowledgeArticleWhenMissing 知识库文章整行缺失 → 挂运维管理根补种。
func TestMigrate228SeedsKnowledgeArticleWhenMissing(t *testing.T) {
	db := freshSQLiteDBForMigrate228(t)

	if err := Migrate228ServerSortMenus(db); err != nil {
		t.Fatalf("run: %v", err)
	}

	menu, ok := get228Menu(t, db, "knowledge/articles/index")
	if !ok {
		t.Fatalf("知识库文章菜单未补种")
	}
	if menu.ParentID == nil || *menu.ParentID != "c50b5b01-fdbc-4821-a3e8-2e2178339b20" {
		t.Fatalf("parent 期望运维管理根, 实际 %v", menu.ParentID)
	}
}

// TestMigrate228SeedsNetworkTemplateMenu 网络模板：无菜单 → 按网络设备参照反查父补种。
func TestMigrate228SeedsNetworkTemplateMenu(t *testing.T) {
	db := freshSQLiteDBForMigrate228(t)

	for run := 1; run <= 2; run++ {
		if err := Migrate228ServerSortMenus(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}
	menu, ok := get228Menu(t, db, "network/templates/index")
	if !ok {
		t.Fatalf("网络模板菜单未补种")
	}
	if menu.MenuType != models.MenuTypeMenu {
		t.Fatalf("menu_type 期望 C, 实际 %s", menu.MenuType)
	}
	if menu.Perms == nil || *menu.Perms != "network:template:list" {
		t.Fatalf("perms 期望 network:template:list, 实际 %v", menu.Perms)
	}
	if menu.Visible != models.VisibleShow || menu.Status != models.MenuStatusNormal {
		t.Fatalf("visible=%d status=%d, 期望 1/0", menu.Visible, menu.Status)
	}
}

// TestMigrate228IdempotentNoOp 对已全部可达的目录重复执行不报错且不产生重复行。
func TestMigrate228IdempotentNoOp(t *testing.T) {
	db := freshSQLiteDBForMigrate228(t)

	// 预置：知识库文章（标准形态）与网络模板均已存在且可达
	kc := "knowledge/articles/index"
	perms := "ops:knowledge:article:list"
	ka := models.Menu{
		MenuName:  "知识库文章",
		ParentID:  strPtr228("c50b5b01-fdbc-4821-a3e8-2e2178339b20"),
		Path:      strPtr228("knowledge/articles"),
		Component: &kc,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
	}
	ka.ID = "24cd03cd-4780-4bad-ae4c-cef46b8f1418"
	nc := "network/templates/index"
	nperms := "network:template:list"
	nt := models.Menu{
		MenuName:  "网络模板",
		ParentID:  strPtr228("0013f129-3ec0-4e55-8ffc-25d97b20c37b"),
		Path:      strPtr228("templates"),
		Component: &nc,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &nperms,
	}
	nt.ID = "f2280000-0000-0000-0000-000000000001"
	if err := db.Create(&[]models.Menu{ka, nt}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	for run := 1; run <= 3; run++ {
		if err := Migrate228ServerSortMenus(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}

	var total int64
	if err := db.Model(&models.Menu{}).Count(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	// 4 参照 + 2 预置 = 6；重复执行不得产生重复行
	if total != 6 {
		t.Fatalf("菜单总数期望 6, 实际 %d（存在重复插入）", total)
	}
}

// strPtr228 字符串指针 helper（避免与包内其它测试 helper 冲突）。
func strPtr228(s string) *string { return &s }

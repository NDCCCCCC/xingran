package migrations

// migration_243 数据侧证据：低密度页面 Tab 化后的菜单收敛。
//
// 覆盖：
//  1. duty/holidays、duty/config、workorder/categories 三个 C 菜单被隐藏 + 停用（行保留）；
//  2. 它们名下的 F 型按钮权限 parent_id 改挂宿主菜单，perms 原值不变；
//  3. 连跑两次幂等（行数/父级不变）；
//  4. 宿主菜单缺失 → 非阻断（仍隐藏旧菜单，不改挂）；
//  5. 菜单不存在（存量瘦库）→ 跳过且不报错。
//
// 复用 migration_241 的 sqlite 最小表夹具（sys_menu 单表）。

import (
	"testing"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

const (
	test243DutyHolidayMenuID  = "c0000000-0000-4000-8000-000000002431"
	test243DutyConfigMenuID   = "c0000000-0000-4000-8000-000000002432"
	test243WOCategoryMenuID   = "c0000000-0000-4000-8000-000000002433"
	test243DutyManageHostID   = "c0000000-0000-4000-8000-000000002441"
	test243WorkOrderHostID    = "c0000000-0000-4000-8000-000000002442"
	test243HolidayListBtnID   = "c0000000-0000-4000-8000-000000002451"
	test243HolidayAddBtnID    = "c0000000-0000-4000-8000-000000002452"
	test243ConfigEditBtnID    = "c0000000-0000-4000-8000-000000002453"
	test243CategoryListBtnID  = "c0000000-0000-4000-8000-000000002454"
	test243DutyRootMenuID     = "c0000000-0000-4000-8000-000000002450"
	test243WorkorderRootMenID = "c0000000-0000-4000-8000-000000002455"
)

// seedFixture243 构造「合并前」快照：两个域根 + 两个宿主 C + 三个待隐藏 C + 4 个 F。
func seedFixture243(t *testing.T, db *gorm.DB) {
	t.Helper()
	dutyRoot := seedMenu241(t, db, test243DutyRootMenuID, "值班管理", "duty", "", "",
		models.MenuTypeDir, nil, models.VisibleShow)
	woRoot := seedMenu241(t, db, test243WorkorderRootMenID, "工单系统", "workorder", "", "",
		models.MenuTypeDir, nil, models.VisibleShow)

	seedMenu241(t, db, test243DutyManageHostID, "配置", "management", "duty/management/index",
		"ops:duty:management:view", models.MenuTypeMenu, &dutyRoot.ID, models.VisibleShow)
	seedMenu241(t, db, test243WorkOrderHostID, "工单管理", "orders", "workorder/orders/index",
		"ops:workorder:list", models.MenuTypeMenu, &woRoot.ID, models.VisibleShow)

	holidayPage := seedMenu241(t, db, test243DutyHolidayMenuID, "节假日管理", "holidays",
		"duty/holidays/index", "ops:duty:holidays:view", models.MenuTypeMenu,
		&dutyRoot.ID, models.VisibleShow)
	configPage := seedMenu241(t, db, test243DutyConfigMenuID, "值班配置", "config",
		"duty/config/index", "ops:duty:config:view", models.MenuTypeMenu,
		&dutyRoot.ID, models.VisibleShow)
	categoryPage := seedMenu241(t, db, test243WOCategoryMenuID, "工单分类", "categories",
		"workorder/categories/index", "ops:workorder:category:list", models.MenuTypeMenu,
		&woRoot.ID, models.VisibleShow)

	seedMenu241(t, db, test243HolidayListBtnID, "节假日查询", "", "",
		"ops:duty:holiday:list", models.MenuTypeButton, &holidayPage.ID, models.VisibleShow)
	seedMenu241(t, db, test243HolidayAddBtnID, "节假日新增", "", "",
		"ops:duty:holiday:add", models.MenuTypeButton, &holidayPage.ID, models.VisibleShow)
	seedMenu241(t, db, test243ConfigEditBtnID, "值班配置修改", "", "",
		"ops:duty:config:edit", models.MenuTypeButton, &configPage.ID, models.VisibleShow)
	seedMenu241(t, db, test243CategoryListBtnID, "工单分类查询", "", "",
		"ops:workorder:category:list", models.MenuTypeButton, &categoryPage.ID, models.VisibleShow)
}

func TestMigrate243HidesPagesAndReparentsButtons(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, "migrate243_hide")
	seedFixture243(t, db)

	if err := Migrate243LowDensityMenus(db); err != nil {
		t.Fatalf("migrate 243: %v", err)
	}

	// 1) 三个页面菜单：隐藏 + 停用，行仍在
	for _, id := range []string{
		test243DutyHolidayMenuID, test243DutyConfigMenuID, test243WOCategoryMenuID,
	} {
		menu := loadMenu241(t, db, id)
		if menu.Visible != models.VisibleHidden {
			t.Errorf("menu %s visible = %d, want %d(隐藏)", id, menu.Visible, models.VisibleHidden)
		}
		if menu.Status != models.MenuStatusStop {
			t.Errorf("menu %s status = %d, want %d(停用)", id, menu.Status, models.MenuStatusStop)
		}
	}

	// 2) F 权限改挂宿主，perms 原值不变
	assertParent241(t, db, test243HolidayListBtnID, test243DutyManageHostID)
	assertParent241(t, db, test243HolidayAddBtnID, test243DutyManageHostID)
	assertParent241(t, db, test243ConfigEditBtnID, test243DutyManageHostID)
	assertParent241(t, db, test243CategoryListBtnID, test243WorkOrderHostID)

	for id, want := range map[string]string{
		test243HolidayListBtnID:  "ops:duty:holiday:list",
		test243ConfigEditBtnID:   "ops:duty:config:edit",
		test243CategoryListBtnID: "ops:workorder:category:list",
	} {
		btn := loadMenu241(t, db, id)
		if btn.Perms == nil || *btn.Perms != want {
			t.Errorf("menu %s perms 被改动: got %v want %s", id, btn.Perms, want)
		}
	}
}

func TestMigrate243Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, "migrate243_idem")
	seedFixture243(t, db)

	if err := Migrate243LowDensityMenus(db); err != nil {
		t.Fatalf("首次 migrate 243: %v", err)
	}
	var countAfterFirst int64
	if err := db.Model(&models.Menu{}).Count(&countAfterFirst).Error; err != nil {
		t.Fatalf("count: %v", err)
	}

	if err := Migrate243LowDensityMenus(db); err != nil {
		t.Fatalf("二次 migrate 243: %v", err)
	}
	var countAfterSecond int64
	if err := db.Model(&models.Menu{}).Count(&countAfterSecond).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if countAfterFirst != countAfterSecond {
		t.Errorf("二次执行菜单行数变化: %d → %d", countAfterFirst, countAfterSecond)
	}
	assertParent241(t, db, test243HolidayListBtnID, test243DutyManageHostID)
	assertVisible241(t, db, test243DutyHolidayMenuID, models.VisibleHidden)
}

func TestMigrate243NoHostMenuStillHides(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, "migrate243_nohost")
	dutyRoot := seedMenu241(t, db, test243DutyRootMenuID, "值班管理", "duty", "", "",
		models.MenuTypeDir, nil, models.VisibleShow)
	page := seedMenu241(t, db, test243DutyHolidayMenuID, "节假日管理", "holidays",
		"duty/holidays/index", "ops:duty:holidays:view", models.MenuTypeMenu,
		&dutyRoot.ID, models.VisibleShow)
	seedMenu241(t, db, test243HolidayListBtnID, "节假日查询", "", "",
		"ops:duty:holiday:list", models.MenuTypeButton, &page.ID, models.VisibleShow)

	if err := Migrate243LowDensityMenus(db); err != nil {
		t.Fatalf("宿主缺失时应非阻断, got: %v", err)
	}
	assertVisible241(t, db, test243DutyHolidayMenuID, models.VisibleHidden)
	// 无宿主 → F 保持原父级（不制造孤立挂载）
	assertParent241(t, db, test243HolidayListBtnID, test243DutyHolidayMenuID)
}

func TestMigrate243MissingMenusSkipped(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, "migrate243_missing")

	if err := Migrate243LowDensityMenus(db); err != nil {
		t.Fatalf("空菜单表应跳过而非报错, got: %v", err)
	}
	var count int64
	if err := db.Model(&models.Menu{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("迁移不应新建菜单, got %d 行", count)
	}
}

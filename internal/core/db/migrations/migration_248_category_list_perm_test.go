package migrations

// Migration 248 数据侧证据：补 status=0 的 F 型授权载体。
//
// 覆盖：
//  1. 迁移后存在 perms='ops:workorder:category:list' AND status=0 AND menu_type='F' 的行，
//     且 parent_id 挂宿主「工单管理」C；旧 C 行保留、perms/status 不变；
//  2. 连跑两次不产生重复行（幂等）；
//  3. 按 perms 复用既有 F 行（不重复插入），并把其 status/parent 统一；
//  4. 同 perms 多条既有 F 行时全部归位、不删行、不重复插入。
//
// 复用 migration_241 的 sqlite 最小表夹具（sys_menu 单表）。

import (
	"testing"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

const (
	test248HostMenuID   = "633bac92-d1e4-416c-bdce-16231a1a4c32"
	test248LegacyCID    = "da379d50-8db3-487a-934b-8ed46fd00da3"
	test248ExistingFID1 = "d2480000-0000-4000-8000-000000000001"
	test248ExistingFID2 = "d2480000-0000-4000-8000-000000000002"
	test248RootID       = "d2480000-0000-4000-8000-000000000010"
)

// seed248HostAndLegacy 造「工单管理」宿主 C + 旧「工单分类」C（模拟生产库）。
func seed248HostAndLegacy(t *testing.T, db *gorm.DB) {
	t.Helper()
	root := seedMenu241(t, db, test248RootID, "工单系统", "workorder", "", "",
		models.MenuTypeDir, nil, models.VisibleShow)
	seedMenu241(t, db, test248HostMenuID, "工单管理", "orders", "workorder/orders/index",
		"ops:workorder:list", models.MenuTypeMenu, &root.ID, models.VisibleShow)
	legacy := seedMenu241(t, db, test248LegacyCID, "工单分类", "categories",
		"workorder/categories/index", categoryListPermCode, models.MenuTypeMenu,
		&root.ID, models.VisibleHidden)
	// 生产库里旧 C 行是 status=1（停用）
	if err := db.Model(&models.Menu{}).Where("id = ?", legacy.ID).
		Update("status", models.MenuStatusStop).Error; err != nil {
		t.Fatalf("set legacy C status: %v", err)
	}
}

func count248F(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.Menu{}).
		Where("perms = ? AND menu_type = ?", categoryListPermCode, models.MenuTypeButton).
		Count(&n).Error; err != nil {
		t.Fatalf("count F: %v", err)
	}
	return n
}

func TestMigrate248SeedsStatus0FButton(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed248HostAndLegacy(t, db)

	if err := Migrate248CategoryListPerm(db); err != nil {
		t.Fatalf("migrate 248: %v", err)
	}

	// ① 存在 perms 匹配 + status=0 + F 型的行
	var btn models.Menu
	if err := db.Where("perms = ? AND menu_type = ? AND status = ?",
		categoryListPermCode, models.MenuTypeButton, models.MenuStatusNormal).
		First(&btn).Error; err != nil {
		t.Fatalf("未找到 status=0 的 F 型载体: %v", err)
	}
	// 挂宿主「工单管理」C
	if btn.ParentID == nil || *btn.ParentID != test248HostMenuID {
		t.Fatalf("F 行 parent_id = %v, want %s", btn.ParentID, test248HostMenuID)
	}
	// 不重复插入
	if n := count248F(t, db); n != 1 {
		t.Fatalf("F 行数 = %d, want 1", n)
	}
	// 旧 C 行保留、perms 未变、status 未被改
	legacy := loadMenu241(t, db, test248LegacyCID)
	if legacy.MenuType != models.MenuTypeMenu {
		t.Fatalf("旧 C 行类型被改: %s", legacy.MenuType)
	}
	if legacy.Perms == nil || *legacy.Perms != categoryListPermCode {
		t.Fatalf("旧 C 行 perms 被改: %v", legacy.Perms)
	}
	if legacy.Status != models.MenuStatusStop {
		t.Fatalf("旧 C 行 status 被改: %d", legacy.Status)
	}
}

func TestMigrate248Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed248HostAndLegacy(t, db)

	for run := 1; run <= 2; run++ {
		if err := Migrate248CategoryListPerm(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if n := count248F(t, db); n != 1 {
			t.Fatalf("run %d: F 行数 = %d, want 1（重复插入）", run, n)
		}
	}
	var total int64
	if err := db.Model(&models.Menu{}).Count(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 4 { // 根 + 宿主 + 旧C + 新F
		t.Fatalf("总行数 = %d, want 4（连跑两次不得产生重复行）", total)
	}
}

func TestMigrate248ReusesExistingFButton(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed248HostAndLegacy(t, db)

	// 预置一条 perms 相同但 status=1 / 挂旧 C 的 F 行（升级库随机 UUID 行）
	legacy := loadMenu241(t, db, test248LegacyCID)
	f := seedMenu241(t, db, test248ExistingFID1, "工单分类查询", "", "",
		categoryListPermCode, models.MenuTypeButton, &legacy.ID, models.VisibleHidden)
	if err := db.Model(&models.Menu{}).Where("id = ?", f.ID).
		Update("status", models.MenuStatusStop).Error; err != nil {
		t.Fatalf("set F status: %v", err)
	}

	if err := Migrate248CategoryListPerm(db); err != nil {
		t.Fatalf("migrate 248: %v", err)
	}

	// 复用既有行：不新增、不删除
	if n := count248F(t, db); n != 1 {
		t.Fatalf("F 行数 = %d, want 1（应复用既有行）", n)
	}
	got := loadMenu241(t, db, test248ExistingFID1)
	if got.Status != models.MenuStatusNormal {
		t.Fatalf("复用行 status = %d, want 0", got.Status)
	}
	if got.ParentID == nil || *got.ParentID != test248HostMenuID {
		t.Fatalf("复用行 parent_id = %v, want %s", got.ParentID, test248HostMenuID)
	}
}

func TestMigrate248NormalizesAllDuplicateFButtons(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed248HostAndLegacy(t, db)

	legacy := loadMenu241(t, db, test248LegacyCID)
	for _, id := range []string{test248ExistingFID1, test248ExistingFID2} {
		f := seedMenu241(t, db, id, "工单分类查询", "", "",
			categoryListPermCode, models.MenuTypeButton, &legacy.ID, models.VisibleHidden)
		if err := db.Model(&models.Menu{}).Where("id = ?", f.ID).
			Update("status", models.MenuStatusStop).Error; err != nil {
			t.Fatalf("set F status: %v", err)
		}
	}

	if err := Migrate248CategoryListPerm(db); err != nil {
		t.Fatalf("migrate 248: %v", err)
	}

	// 两条都已归位，且没有新增/删除
	if n := count248F(t, db); n != 2 {
		t.Fatalf("F 行数 = %d, want 2（全部保留、不重复插入、不删除）", n)
	}
	for _, id := range []string{test248ExistingFID1, test248ExistingFID2} {
		got := loadMenu241(t, db, id)
		if got.Status != models.MenuStatusNormal || got.ParentID == nil || *got.ParentID != test248HostMenuID {
			t.Fatalf("F 行 %s 未归位: status=%d parent=%v", id, got.Status, got.ParentID)
		}
	}
}

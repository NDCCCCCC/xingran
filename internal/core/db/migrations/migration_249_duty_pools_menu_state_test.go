package migrations

// Migration 249 数据侧证据：duty/pools 未软删 C 行统一为 hidden+stop。
//
// 覆盖：
//  1. 同 component 多条重复行（含带前导斜杠变体）全部被归位到 visible=0/status=1；
//  2. 已是目标态的行幂等短路；
//  3. 只更新活行：软删行不受影响（Unscoped 验证），非 C 型（M/F）行不受影响；
//  4. 不删行、不增行、不改 perms；
//  5. 连跑两次幂等。
//
// 复用 migration_241 的 sqlite 最小表夹具（sys_menu 单表）。

import (
	"testing"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

const (
	test249RootID      = "d2490000-0000-4000-8000-000000000010"
	test249BaselineID  = "c4a5ff39-96ad-48d5-b1c1-543ef88f8373"
	test249Dup1ID      = "d2490000-0000-4000-8000-000000000001"
	test249Dup2ID      = "d2490000-0000-4000-8000-000000000002"
	test249AlreadyOKID = "d2490000-0000-4000-8000-000000000003"
	test249FRowID      = "d2490000-0000-4000-8000-000000000004"
	test249MDirID      = "d2490000-0000-4000-8000-000000000005"
	test249SoftDelID   = "d2490000-0000-4000-8000-000000000006"
	test249Perms       = "ops:duty:pools:view"
)

// seed249Root 造「值班管理」目录根，供 parent 结构真实。
func seed249Root(t *testing.T, db *gorm.DB) {
	t.Helper()
	seedMenu241(t, db, test249RootID, "值班管理", "duty", "", "",
		models.MenuTypeDir, nil, models.VisibleShow)
}

// seed249Pool 造一条 duty/pools 行（可指定 component / visible / status / menu_type）。
func seed249Pool(t *testing.T, db *gorm.DB, id, component string,
	visible models.VisibleType, status models.MenuStatus, menuType models.MenuType) {
	t.Helper()
	root := test249RootID
	menu := seedMenu241(t, db, id, "值班池管理", "pools", component,
		test249Perms, menuType, &root, visible)
	if status != models.MenuStatusNormal {
		if err := db.Model(&models.Menu{}).Where("id = ?", menu.ID).
			Update("status", status).Error; err != nil {
			t.Fatalf("set status of %s: %v", id, err)
		}
	}
}

func count249All(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Unscoped().Model(&models.Menu{}).Count(&n).Error; err != nil {
		t.Fatalf("count all: %v", err)
	}
	return n
}

func assert249State(t *testing.T, db *gorm.DB, id string,
	wantVisible models.VisibleType, wantStatus models.MenuStatus) {
	t.Helper()
	m := loadMenu241(t, db, id)
	if m.Visible != wantVisible || m.Status != wantStatus {
		t.Fatalf("menu %s: visible=%d status=%d, want %d/%d",
			id, m.Visible, m.Status, wantVisible, wantStatus)
	}
	if m.Perms == nil || *m.Perms != test249Perms {
		t.Fatalf("menu %s perms 被改动: %v", id, m.Perms)
	}
}

func TestMigrate249NormalizesAllDuplicateRows(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed249Root(t, db)

	// 3 条待归位活行：基线 + 2 条随机重复（其一为带前导斜杠变体），全为 1/0（被 228 改过的态）
	seed249Pool(t, db, test249BaselineID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	seed249Pool(t, db, test249Dup1ID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	seed249Pool(t, db, test249Dup2ID, "/duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	// 已是目标态（0/1）→ 幂等短路
	seed249Pool(t, db, test249AlreadyOKID, "duty/pools/index",
		models.VisibleHidden, models.MenuStatusStop, models.MenuTypeMenu)
	// 非 C 型（F 按钮 / M 目录）→ 不应被归位
	seed249Pool(t, db, test249FRowID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeButton)
	seed249Pool(t, db, test249MDirID, "/duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeDir)
	// 软删行 → 不应被归位（GORM 默认软删过滤）
	seed249Pool(t, db, test249SoftDelID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	if err := db.Where("id = ?", test249SoftDelID).Delete(&models.Menu{}).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	before := count249All(t, db)

	if err := Migrate249DutyPoolsMenuState(db); err != nil {
		t.Fatalf("migrate 249: %v", err)
	}

	// ① 4 条活 C 行全部归位（含带斜杠变体、含已是目标态的行）
	for _, id := range []string{
		test249BaselineID, test249Dup1ID, test249Dup2ID, test249AlreadyOKID,
	} {
		assert249State(t, db, id, models.VisibleHidden, models.MenuStatusStop)
	}
	// ② 非 C 型不受影响（仍 1/0）
	assert249State(t, db, test249FRowID, models.VisibleShow, models.MenuStatusNormal)
	assert249State(t, db, test249MDirID, models.VisibleShow, models.MenuStatusNormal)
	// ③ 软删行不受影响：Unscoped 读，仍为 1/0 且 deleted_at 非空
	var soft models.Menu
	if err := db.Unscoped().Where("id = ?", test249SoftDelID).First(&soft).Error; err != nil {
		t.Fatalf("load soft-deleted: %v", err)
	}
	if soft.Visible != models.VisibleShow || soft.Status != models.MenuStatusNormal {
		t.Fatalf("软删行被改动: visible=%d status=%d", soft.Visible, soft.Status)
	}
	if !soft.DeletedAt.Valid {
		t.Fatalf("软删行 deleted_at 应非空")
	}
	// ④ 不删行、不增行
	if after := count249All(t, db); after != before {
		t.Fatalf("总行数变化: %d → %d（不应删/增任何行）", before, after)
	}
}

func TestMigrate249Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seed249Root(t, db)
	seed249Pool(t, db, test249BaselineID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	seed249Pool(t, db, test249Dup1ID, "duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)
	seed249Pool(t, db, test249Dup2ID, "/duty/pools/index",
		models.VisibleShow, models.MenuStatusNormal, models.MenuTypeMenu)

	for run := 1; run <= 2; run++ {
		if err := Migrate249DutyPoolsMenuState(db); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		for _, id := range []string{test249BaselineID, test249Dup1ID, test249Dup2ID} {
			assert249State(t, db, id, models.VisibleHidden, models.MenuStatusStop)
		}
	}
	// 根 + 3 行，连跑两次不得产生重复行
	if total := count249All(t, db); total != 4 {
		t.Fatalf("总行数 = %d, want 4", total)
	}
}

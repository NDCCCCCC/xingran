package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 244 测试 — monitor 域 RBAC F 型按钮播种
//
// 覆盖：
//   - 244-01: 声明块与种子表逐条 1:1（条数 + 顺序 + 码值）
//   - 244-02: 种子执行后 monitor F 型行数 == 种子表条数（父菜单缺失的码被降级跳过不计）
//   - 244-03: 幂等性（连跑 2 次行数不变）
// 防假绿约束：不硬编码码字面量，统一引用包内 monitorPermissionSeeds。
// ============================================================================

func freshSQLiteDBForMigrate244(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, stmt := range migrate226Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec schema: %v", err)
		}
	}
	return db
}

func seedMonitorParents(t *testing.T, db *gorm.DB) {
	t.Helper()
	parentPerms := map[string]string{
		"monitor:server": "服务监控",
		"monitor:cache":  "缓存管理",
		"monitor:job":    "定时任务",
		"monitor:log":    "日志管理",
	}
	for perms, name := range parentPerms {
		p := perms
		m := &models.Menu{
			MenuName: name,
			MenuType: models.MenuTypeMenu,
			Status:   models.MenuStatusNormal,
			Perms:    &p,
		}
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed parent %s: %v", perms, err)
		}
	}
}

func countMonitorFMenus(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.Menu{}).
		Where("menu_type = ?", models.MenuTypeButton).
		Where("perms LIKE ?", "monitor:%").
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestMigrate244DeclarationConsistency(t *testing.T) {
	if len(monitorPermissionFTypeSeedCodes) != len(monitorPermissionSeeds) {
		t.Fatalf("声明块 %d != 种子表 %d",
			len(monitorPermissionFTypeSeedCodes), len(monitorPermissionSeeds))
	}
	for i, code := range monitorPermissionFTypeSeedCodes {
		if code != monitorPermissionSeeds[i].perms {
			t.Fatalf("第 %d 项漂移: 声明(%s) != 种子(%s)", i, code, monitorPermissionSeeds[i].perms)
		}
	}
}

func TestMigrate244SeedsAndIsIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate244(t)
	seedMonitorParents(t, db)

	if err := Migrate244MonitorPermissionFTypeSeed(db); err != nil {
		t.Fatalf("run 244: %v", err)
	}
	first := countMonitorFMenus(t, db)
	// 全部父菜单 live → 24 条种子全部建行（22 + online 2 码因父非 C 型降级跳过 → 实际 22；
	// online 码不在种子表内，故期望值 = len(monitorPermissionSeeds)）
	if want := int64(len(monitorPermissionSeeds)); first != want {
		t.Fatalf("monitor F menus = %d, want %d", first, want)
	}

	// 幂等：第二次执行行数不变
	if err := Migrate244MonitorPermissionFTypeSeed(db); err != nil {
		t.Fatalf("rerun 244: %v", err)
	}
	if second := countMonitorFMenus(t, db); second != first {
		t.Fatalf("idempotency broken: first=%d second=%d", first, second)
	}
}

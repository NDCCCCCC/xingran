package migrations

// Migration 251 数据侧证据：验证码背景图 status 极性归一（1=启用 → 0=启用）。
//
// 覆盖：
//  1. 存量行 status 被翻转：旧 1（启用）→ 新 0、旧 0（禁用）→ 新 1；
//  2. 非 {0,1} 的异常值不被翻转（WHERE status IN (0,1) 限定）；
//  3. 软删行同样被翻转（迁移用 raw SQL，不受 GORM 默认软删作用域影响）——
//     软删行若不翻转，恢复后语义仍是旧的；
//  4. 不删行、不增行；
//  5. 表不存在时非阻断跳过；
//  6. 列 DEFAULT 归一：sqlite 方言下依赖 model tag（default:0），
//     PG 方言下走 ALTER COLUMN SET DEFAULT 0。

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// freshSQLiteDBForMigrate251 建一个仅含 sys_captcha_background 的内存 sqlite 库。
func freshSQLiteDBForMigrate251(t *testing.T, name string) *gorm.DB {
	t.Helper()
	safe := strings.NewReplacer("/", "_", " ", "_", "=", "_").Replace(name)
	dsn := "file:" + safe + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.CaptchaBackground{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seed251Bg 落一行 sys_captcha_background（指定 status 原始值）。
func seed251Bg(t *testing.T, db *gorm.DB, id string, status int) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO sys_captcha_background (id, file_name, file_path, status, use_count)
		 VALUES (?, ?, ?, ?, 0)`,
		id, id+".png", "/tmp/"+id+".png", status,
	).Error; err != nil {
		t.Fatalf("seed bg %s: %v", id, err)
	}
}

// read251Status 读某行的 status（Unscoped 以便校验软删行）。
func read251Status(t *testing.T, db *gorm.DB, id string) int {
	t.Helper()
	var status int
	if err := db.Raw("SELECT status FROM sys_captcha_background WHERE id = ?", id).
		Scan(&status).Error; err != nil {
		t.Fatalf("read status of %s: %v", id, err)
	}
	return status
}

func TestMigrate251FlipsExistingStatus(t *testing.T) {
	db := freshSQLiteDBForMigrate251(t, t.Name())

	// 旧语义：1=启用、0=禁用
	seed251Bg(t, db, "bg-old-enabled", 1)
	seed251Bg(t, db, "bg-old-disabled", 0)
	// 异常值：不在 {0,1}，不应被翻转
	seed251Bg(t, db, "bg-weird", 7)

	if err := Migrate251CaptchaBgStatusPolarity(db); err != nil {
		t.Fatalf("migrate 251: %v", err)
	}

	// 旧 1（启用）→ 新 0（启用）
	if got := read251Status(t, db, "bg-old-enabled"); got != int(models.CaptchaBgEnabled) {
		t.Fatalf("旧启用行翻转后 status=%d, want %d", got, int(models.CaptchaBgEnabled))
	}
	// 旧 0（禁用）→ 新 1（禁用）
	if got := read251Status(t, db, "bg-old-disabled"); got != int(models.CaptchaBgDisabled) {
		t.Fatalf("旧禁用行翻转后 status=%d, want %d", got, int(models.CaptchaBgDisabled))
	}
	// 异常值保持不变
	if got := read251Status(t, db, "bg-weird"); got != 7 {
		t.Fatalf("异常值行被改动: status=%d, want 7", got)
	}
}

func TestMigrate251FlipsSoftDeletedRows(t *testing.T) {
	db := freshSQLiteDBForMigrate251(t, t.Name())

	seed251Bg(t, db, "bg-soft", 1)
	// 软删：迁移走 raw SQL，不受 GORM 默认软删作用域影响，软删行同样需翻转
	if err := db.Exec(
		"UPDATE sys_captcha_background SET deleted_at = ? WHERE id = ?",
		"2026-01-01 00:00:00", "bg-soft",
	).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	if err := Migrate251CaptchaBgStatusPolarity(db); err != nil {
		t.Fatalf("migrate 251: %v", err)
	}

	if got := read251Status(t, db, "bg-soft"); got != int(models.CaptchaBgEnabled) {
		t.Fatalf("软删行未翻转: status=%d, want %d", got, int(models.CaptchaBgEnabled))
	}
}

func TestMigrate251DoesNotAddOrRemoveRows(t *testing.T) {
	db := freshSQLiteDBForMigrate251(t, t.Name())

	seed251Bg(t, db, "bg-1", 1)
	seed251Bg(t, db, "bg-2", 0)

	var before int64
	if err := db.Raw("SELECT COUNT(*) FROM sys_captcha_background").Scan(&before).Error; err != nil {
		t.Fatalf("count before: %v", err)
	}

	if err := Migrate251CaptchaBgStatusPolarity(db); err != nil {
		t.Fatalf("migrate 251: %v", err)
	}

	var after int64
	if err := db.Raw("SELECT COUNT(*) FROM sys_captcha_background").Scan(&after).Error; err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before {
		t.Fatalf("总行数变化: %d → %d（不应删/增任何行）", before, after)
	}
}

func TestMigrate251SkipsWhenTableMissing(t *testing.T) {
	// 空库（无 sys_captcha_background 表）→ 非阻断跳过，不返回 error
	db := freshSQLiteDBForMigrate241(t, t.Name()) // 该 helper 只建 sys_menu 表
	if err := Migrate251CaptchaBgStatusPolarity(db); err != nil {
		t.Fatalf("表缺失时应非阻断跳过, got error: %v", err)
	}
}

// TestMigrate251NewRowDefaultsToEnabled 归一后新行（不显式给 status）应落「启用」(0)，
// 与翻转后的极性一致 —— 否则新插入的行会与存量行语义相反。
func TestMigrate251NewRowDefaultsToEnabled(t *testing.T) {
	db := freshSQLiteDBForMigrate251(t, t.Name())

	if err := Migrate251CaptchaBgStatusPolarity(db); err != nil {
		t.Fatalf("migrate 251: %v", err)
	}

	// raw INSERT 不给 status → 取列 DEFAULT
	seed251Bg(t, db, "bg-default", 0) // 显式 0
	// 另测「省略 status 列」的插入路径
	if err := db.Exec(
		`INSERT INTO sys_captcha_background (id, file_name, file_path, use_count)
		 VALUES ('bg-nodefault', 'n.png', '/tmp/n.png', 0)`,
	).Error; err != nil {
		t.Fatalf("insert without status: %v", err)
	}

	if got := read251Status(t, db, "bg-nodefault"); got != int(models.CaptchaBgEnabled) {
		t.Fatalf("省略 status 的新行落 status=%d, want %d（默认应为启用）",
			got, int(models.CaptchaBgEnabled))
	}
}

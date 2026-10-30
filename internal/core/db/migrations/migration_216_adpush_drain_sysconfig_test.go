package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate216 构建内存 SQLite 库（sys_config 最小表集）。
// 形状照抄 freshSQLiteDBForMigrate208。
func freshSQLiteDBForMigrate216(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Config{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

const migrate216SeedKey = "ad_push_queue.drain_interval_seconds"

// TestPhase58_Migration216_SeedIdempotent 锁死迁移的核心不变量：
// 幂等（重复跑不新增行）且**绝不覆盖管理员已改的值**（T-58-05-06）。
func TestPhase58_Migration216_SeedIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate216(t)

	// 1) 干净库跑一次 → 恰好 1 行，值为 "30"
	if err := Migrate216ADPushDrainSysConfig(db); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}

	var count int64
	if err := db.Model(&models.Config{}).Where("config_key = ?", migrate216SeedKey).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("首次迁移后行数 = %d, 期望 1", count)
	}

	var value string
	if err := db.Raw("SELECT config_value FROM sys_config WHERE config_key = ?", migrate216SeedKey).Scan(&value).Error; err != nil {
		t.Fatalf("select value: %v", err)
	}
	if value != "30" {
		t.Fatalf("seed 值 = %q, 期望 \"30\"", value)
	}

	// 2) 模拟管理员把值改成 300
	if err := db.Exec("UPDATE sys_config SET config_value = '300' WHERE config_key = ?", migrate216SeedKey).Error; err != nil {
		t.Fatalf("手工改值: %v", err)
	}

	// 3) 再跑一次迁移 → 仍 1 行，且值仍是 '300'（核心断言：不覆盖管理员的改动）
	if err := Migrate216ADPushDrainSysConfig(db); err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}
	if err := db.Model(&models.Config{}).Where("config_key = ?", migrate216SeedKey).Count(&count).Error; err != nil {
		t.Fatalf("二次 count: %v", err)
	}
	if count != 1 {
		t.Fatalf("二次迁移后行数 = %d, 期望 1（幂等）", count)
	}
	if err := db.Raw("SELECT config_value FROM sys_config WHERE config_key = ?", migrate216SeedKey).Scan(&value).Error; err != nil {
		t.Fatalf("二次 select value: %v", err)
	}
	if value != "300" {
		t.Fatalf("二次迁移后 seed 值 = %q, 期望 \"300\"（迁移不得覆盖管理员改动）", value)
	}

	// 4) config_type == "N"（ConfigTypeNo）且 is_system == 0
	var configType string
	var isSystem int
	if err := db.Raw("SELECT config_type, is_system FROM sys_config WHERE config_key = ?", migrate216SeedKey).
		Row().Scan(&configType, &isSystem); err != nil {
		t.Fatalf("select type/system: %v", err)
	}
	if configType != string(models.ConfigTypeNo) {
		t.Fatalf("config_type = %q, 期望 %q", configType, models.ConfigTypeNo)
	}
	if isSystem != int(models.ConfigIsSystemNo) {
		t.Fatalf("is_system = %d, 期望 %d", isSystem, models.ConfigIsSystemNo)
	}
}

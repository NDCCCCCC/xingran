//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate149FixColumnVisibleDefault 修复 sys_user_column_config 表 visible 字段默认值问题。
//
// 背景: 早期版本的 sys_user_column_config 表的 visible 列可能没有正确的 DEFAULT TRUE
// 约束(取决于建表路径)。本迁移确保 DEFAULT TRUE 已设置,并将历史 NULL 值规范化为 TRUE。
//
// 027_create_user_column_config.sql 已经声明了 `visible BOOLEAN DEFAULT TRUE`,
// 因此本迁移在新部署中是 no-op,只对早期通过 GORM AutoMigrate 创建的实例做兜底。
func Migrate149FixColumnVisibleDefault(db *gorm.DB) error {
	log.Println("Running migration 149: Fix sys_user_column_config.visible default value")

	if !db.Migrator().HasTable("sys_user_column_config") {
		log.Println("Table sys_user_column_config does not exist yet, skipping migration 149")
		return nil
	}

	// 确保 DEFAULT TRUE 约束存在(对历史实例做兜底,IF NOT EXISTS 语义由 PG 的 SET DEFAULT 提供幂等性)
	if err := db.Exec("ALTER TABLE sys_user_column_config ALTER COLUMN visible SET DEFAULT TRUE").Error; err != nil {
		log.Printf("Migration 149: ALTER DEFAULT failed (可能是 SQLite 或权限不足,继续): %v", err)
	}

	// 将历史 NULL 值规范化为 TRUE
	if err := db.Exec("UPDATE sys_user_column_config SET visible = TRUE WHERE visible IS NULL").Error; err != nil {
		log.Printf("Migration 149: 规范化 NULL 失败: %v", err)
		return err
	}

	log.Println("Migration 149 completed")
	return nil
}

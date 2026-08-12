//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate033CreateMacOuiVendorTable 创建 OUI 厂商表 sys_mac_oui_vendor
//
// 背景: 项目没有 SQL 文件自动加载器, 033_create_mac_oui_vendor_table.up.sql 一直未被执行,
// 导致 mac_history_query_service.go 调用 GORM 时报 "relation sys_mac_oui_vendor does not exist"。
// 本迁移与该 SQL 文件等价: 建表 + 索引。
//
// 幂等性: 表存在则跳过整段执行。
func Migrate033CreateMacOuiVendorTable(db *gorm.DB) error {
	log.Println("Running migration 033: Create sys_mac_oui_vendor table")

	// 检查表是否已存在(已存在则跳过, 保持幂等)
	if db.Migrator().HasTable("sys_mac_oui_vendor") {
		log.Println("Table sys_mac_oui_vendor already exists, skipping migration 033...")
		return nil
	}

	// 建表 + 索引(与 033_create_mac_oui_vendor_table.up.sql 完全等价)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS sys_mac_oui_vendor (
		oui_prefix VARCHAR(6) PRIMARY KEY,
		vendor_name VARCHAR(255) NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_mac_oui_vendor_updated ON sys_mac_oui_vendor(updated_at);
	`

	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Printf("Failed to create sys_mac_oui_vendor table: %v", err)
		return err
	}

	log.Println("Migration 033 completed successfully")
	return nil
}

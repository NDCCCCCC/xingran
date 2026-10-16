//go:build archive_skip


package migrations

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// Migrate150AddWorkstationDeviceIPAddress 为工位设备关联表添加 IP 地址字段
func Migrate150AddWorkstationDeviceIPAddress(db *gorm.DB) error {
	log.Println("Running migration 150: Add Workstation Device IP Address")

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v, using inline SQL", err)
		return executeInlineSQLFor150(db)
	}

	sqlFile := filepath.Join(wd, "internal/core/db/migrations/150_add_workstation_device_ip_address.sql")
	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		sqlFile = "150_add_workstation_device_ip_address.sql"
		if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
			log.Println("SQL file not found, using inline SQL")
			return executeInlineSQLFor150(db)
		}
	}

	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Printf("Failed to read SQL file: %v, using inline SQL", err)
		return executeInlineSQLFor150(db)
	}

	return executeSQL(db, string(content), 150)
}

func executeInlineSQLFor150(db *gorm.DB) error {
	log.Println("Executing inline SQL for migration 150...")
	sql := `ALTER TABLE ops_workstation_device ADD COLUMN IF NOT EXISTS ip_address VARCHAR(64) COMMENT 'IP地址';
CREATE INDEX IF NOT EXISTS idx_workstation_device_ip ON ops_workstation_device(ip_address);`
	if err := db.Exec(sql).Error; err != nil {
		log.Printf("Failed to execute migration 150: %v", err)
		return err
	}
	log.Println("Migration 150 completed successfully")
	return nil
}

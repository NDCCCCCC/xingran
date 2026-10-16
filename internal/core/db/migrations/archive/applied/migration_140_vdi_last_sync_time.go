//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate140VDILastSyncTime adds last_sync_time column to sys_vdi_server table
func Migrate140VDILastSyncTime(db *gorm.DB) error {
	log.Println("Running migration 140: VDI Server Last Sync Time")

	// Check if column already exists
	if db.Migrator().HasColumn(&VDIServer{}, "last_sync_time") {
		log.Println("Column 'last_sync_time' already exists in sys_vdi_server, skipping...")
		return nil
	}

	// Add last_sync_time column
	if err := db.Migrator().AddColumn(&VDIServer{}, "last_sync_time"); err != nil {
		log.Printf("Failed to add last_sync_time column: %v", err)
		return err
	}

	log.Println("Migration 140 completed successfully: last_sync_time column added")
	return nil
}

// VDIServer 临时模型定义，用于迁移
type VDIServer struct {
	gorm.Model
	LastSyncTime *string `gorm:"column:last_sync_time;type:timestamp"`
}

// TableName 指定表名
func (VDIServer) TableName() string {
	return "sys_vdi_server"
}

//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate143VDIRemoveVMStatus removes the status column from sys_vdi_vm table
func Migrate143VDIRemoveVMStatus(db *gorm.DB) error {
	log.Println("Running migration 143: Remove status column from sys_vdi_vm")

	if err := db.Exec("ALTER TABLE sys_vdi_vm DROP COLUMN IF EXISTS status").Error; err != nil {
		log.Printf("Failed to drop status column from sys_vdi_vm: %v", err)
		return err
	}

	log.Println("Migration 143 completed successfully: Removed status column from sys_vdi_vm")
	return nil
}

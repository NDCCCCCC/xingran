//go:build archive_skip


package migrations

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// Migrate141VDICPUFields updates VDI virtual machine CPU fields to support detailed CPU information
func Migrate141VDICPUFields(db *gorm.DB) error {
	log.Println("Running migration 141: VDI CPU Fields Enhancement")

	// Check if new columns already exist
	if db.Migrator().HasColumn(&VDIVirtualMachine{}, "cpu_number") {
		log.Println("New CPU columns already exist in sys_vdi_vm, skipping...")
		return nil
	}

	// Add new columns for detailed CPU information
	newColumns := []struct {
		name     string
		colType  string
		comment  string
	}{
		{"cpu_number", "int", "CPU颗数"},
		{"cpu_core", "int", "每颗CPU的核数"},
		{"cpu_per", "int", "CPU使用率"},
		{"memory_per", "int", "内存使用率"},
		{"disk_per", "int", "磁盘使用率"},
	}

	for _, col := range newColumns {
		if err := db.Migrator().AddColumn(&VDIVirtualMachine{}, col.name); err != nil {
			log.Printf("Failed to add %s column: %v", col.name, err)
			return err
		}
		log.Printf("Added column: %s (%s)", col.name, col.comment)
	}

	// Migrate existing CPU data to cpu_number field
	log.Println("Migrating existing CPU data to cpu_number field...")
	result := db.Exec(`
		UPDATE sys_vdi_vm
		SET cpu_number = cpu
		WHERE cpu_number IS NULL OR cpu_number = 0
	`)
	if result.Error != nil {
		log.Printf("Warning: Failed to migrate existing CPU data: %v", result.Error)
	} else {
		log.Printf("Migrated %d existing VM records", result.RowsAffected)
	}

	// Set default values for new fields based on common configurations
	log.Println("Setting default values for new CPU fields...")
	// For existing VMs with cpu_number but no cpu_core, set reasonable defaults
	result = db.Exec(`
		UPDATE sys_vdi_vm
		SET cpu_core = 8
		WHERE cpu_number > 0 AND (cpu_core IS NULL OR cpu_core = 0)
	`)
	if result.Error != nil {
		log.Printf("Warning: Failed to set default cpu_core values: %v", result.Error)
	} else {
		log.Printf("Set default cpu_core for %d VM records", result.RowsAffected)
	}

	// Note: We keep the old 'cpu' column for backward compatibility during transition
	// It can be removed in a future migration after confirming the new fields work correctly

	log.Println("Migration 141 completed successfully: VDI CPU fields enhanced")
	return nil
}

// Rollback141VDICPUFields reverts the CPU field changes (for emergency rollback only)
func Rollback141VDICPUFields(db *gorm.DB) error {
	log.Println("Running rollback 141: VDI CPU Fields Enhancement")

	// Restore cpu field from cpu_number if needed
	result := db.Exec(`
		UPDATE sys_vdi_vm
		SET cpu = cpu_number
		WHERE cpu IS NULL OR cpu = 0
	`)
	if result.Error != nil {
		log.Printf("Warning: Failed to restore cpu field: %v", result.Error)
	}

	// Remove new columns
	columnsToRemove := []string{"cpu_number", "cpu_core", "cpu_per", "memory_per", "disk_per"}
	for _, col := range columnsToRemove {
		if err := db.Migrator().DropColumn(&VDIVirtualMachine{}, col); err != nil {
			log.Printf("Failed to drop column %s: %v", col, err)
			return err
		}
		log.Printf("Dropped column: %s", col)
	}

	log.Println("Rollback 141 completed")
	return nil
}

// VDIVirtualMachine temporary model definition for migration
type VDIVirtualMachine struct {
	gorm.Model
	VMID          string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name          string    `gorm:"type:varchar(200);not null"`
	ResourceID    string    `gorm:"type:varchar(100);index"`
	Status        int       `gorm:"type:int;default:0"`
	PowerState    string    `gorm:"type:varchar(50)"`
	IPAddress     string    `gorm:"type:varchar(50)"`
	MACAddress    string    `gorm:"type:varchar(50)"`
	OSType        string    `gorm:"type:varchar(50)"`
	CPU           int       `gorm:"type:int"`              // Legacy field
	CPUNumber     int       `gorm:"type:int"`              // CPU颗数
	CPUCore       int       `gorm:"type:int"`              // 每颗CPU的核数
	CPUPer        int       `gorm:"type:int"`              // CPU使用率
	Memory        int       `gorm:"type:int"`
	MemoryPer     int       `gorm:"type:int"`              // 内存使用率
	Disk          int       `gorm:"type:int"`
	DiskPer       int       `gorm:"type:int"`              // 磁盘使用率
	BoundUserID   *string   `gorm:"type:varchar(100)"`
	BoundUserName *string   `gorm:"type:varchar(200)"`
	PolicyGroupID *string   `gorm:"type:varchar(100)"`
	LastSyncAt    *time.Time
	VdiServerID   string    `gorm:"type:varchar(100);index;not null"`
}

// TableName 指定表名
func (VDIVirtualMachine) TableName() string {
	return "sys_vdi_vm"
}
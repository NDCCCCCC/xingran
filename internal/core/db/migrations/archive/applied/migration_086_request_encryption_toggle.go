//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate086RequestEncryptionToggle creates the request encryption toggle config parameter
func Migrate086RequestEncryptionToggle(db *gorm.DB) error {
	log.Println("Running migration 086: Request Encryption Toggle Config")

	// Check if config already exists
	var count int64
	if err := db.Table("sys_config").Where("config_key = ?", "sys.request.encryption.enabled").Count(&count).Error; err != nil {
		log.Printf("Warning: Failed to check existing config: %v", err)
		return err
	}

	if count > 0 {
		log.Println("Config 'sys.request.encryption.enabled' already exists, skipping...")
		return nil
	}

	// Create the request encryption toggle config
	config := models.Config{
		ConfigName:  "请求加密开关",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    models.ConfigIsSystemYes,
		Remark:      "控制请求体加密功能的启停（true=启用，false=停用），修改后立即生效",
	}

	if err := db.Create(&config).Error; err != nil {
		log.Printf("Failed to create request encryption toggle config: %v", err)
		return err
	}

	log.Println("Migration 086 completed successfully: Request encryption toggle config created")
	return nil
}

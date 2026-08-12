//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate085APIKeys creates API key management tables
func Migrate085APIKeys(db *gorm.DB) error {
	log.Println("Running migration 085: API Keys Management")

	// Check if tables already exist
	if db.Migrator().HasTable(&models.APIKey{}) {
		log.Println("Table sys_api_keys already exists, skipping...")
	} else {
		log.Println("Creating table sys_api_keys...")
		if err := db.AutoMigrate(&models.APIKey{}); err != nil {
			return err
		}
	}

	if db.Migrator().HasTable(&models.APIKeyUsageLog{}) {
		log.Println("Table sys_api_key_usage_logs already exists, skipping...")
	} else {
		log.Println("Creating table sys_api_key_usage_logs...")
		if err := db.AutoMigrate(&models.APIKeyUsageLog{}); err != nil {
			return err
		}
	}

	// Create additional indexes that AutoMigrate might miss
	// Index on APIKey.UserID for faster user-based queries
	if !db.Migrator().HasIndex(&models.APIKey{}, "idx_api_keys_user_id") {
		log.Println("Creating index idx_api_keys_user_id...")
		if err := db.Exec(`CREATE INDEX idx_api_keys_user_id ON sys_api_keys(user_id)`).Error; err != nil {
			log.Printf("Warning: Failed to create idx_api_keys_user_id: %v", err)
		}
	}

	// Index on APIKey.Key for faster key lookups (unique index should already exist)
	if !db.Migrator().HasIndex(&models.APIKey{}, "idx_api_keys_key") {
		log.Println("Creating index idx_api_keys_key...")
		if err := db.Exec(`CREATE INDEX idx_api_keys_key ON sys_api_keys(key)`).Error; err != nil {
			log.Printf("Warning: Failed to create idx_api_keys_key: %v", err)
		}
	}

	// Index on APIKeyUsageLog.APIKeyID for faster log queries by key
	if !db.Migrator().HasIndex(&models.APIKeyUsageLog{}, "idx_api_key_logs_api_key_id") {
		log.Println("Creating index idx_api_key_logs_api_key_id...")
		if err := db.Exec(`CREATE INDEX idx_api_key_logs_api_key_id ON sys_api_key_usage_logs(api_key_id)`).Error; err != nil {
			log.Printf("Warning: Failed to create idx_api_key_logs_api_key_id: %v", err)
		}
	}

	// Index on APIKeyUsageLog.CreatedAt for time-based queries
	if !db.Migrator().HasIndex(&models.APIKeyUsageLog{}, "idx_api_key_logs_created_at") {
		log.Println("Creating index idx_api_key_logs_created_at...")
		if err := db.Exec(`CREATE INDEX idx_api_key_logs_created_at ON sys_api_key_usage_logs(created_at)`).Error; err != nil {
			log.Printf("Warning: Failed to create idx_api_key_logs_created_at: %v", err)
		}
	}

	// Index on APIKeyUsageLog.UserID for user-based log queries
	if !db.Migrator().HasIndex(&models.APIKeyUsageLog{}, "idx_api_key_logs_user_id") {
		log.Println("Creating index idx_api_key_logs_user_id...")
		if err := db.Exec(`CREATE INDEX idx_api_key_logs_user_id ON sys_api_key_usage_logs(user_id)`).Error; err != nil {
			log.Printf("Warning: Failed to create idx_api_key_logs_user_id: %v", err)
		}
	}

	// Add foreign key constraints if they don't exist
	// APIKey.UserID -> sys_user.id
	if !db.Migrator().HasConstraint(&models.APIKey{}, "fk_api_keys_user") {
		log.Println("Creating foreign key fk_api_keys_user...")
		if err := db.Exec(`ALTER TABLE sys_api_keys ADD CONSTRAINT fk_api_keys_user
			FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE SET NULL`).Error; err != nil {
			log.Printf("Warning: Failed to create fk_api_keys_user: %v", err)
		}
	}

	// APIKeyUsageLog.APIKeyID -> sys_api_keys.id
	if !db.Migrator().HasConstraint(&models.APIKeyUsageLog{}, "fk_api_key_logs_api_key") {
		log.Println("Creating foreign key fk_api_key_logs_api_key...")
		if err := db.Exec(`ALTER TABLE sys_api_key_usage_logs ADD CONSTRAINT fk_api_key_logs_api_key
			FOREIGN KEY (api_key_id) REFERENCES sys_api_keys(id) ON DELETE CASCADE`).Error; err != nil {
			log.Printf("Warning: Failed to create fk_api_key_logs_api_key: %v", err)
		}
	}

	// APIKeyUsageLog.UserID -> sys_user.id
	if !db.Migrator().HasConstraint(&models.APIKeyUsageLog{}, "fk_api_key_logs_user") {
		log.Println("Creating foreign key fk_api_key_logs_user...")
		if err := db.Exec(`ALTER TABLE sys_api_key_usage_logs ADD CONSTRAINT fk_api_key_logs_user
			FOREIGN KEY (user_id) REFERENCES sys_user(id) ON DELETE CASCADE`).Error; err != nil {
			log.Printf("Warning: Failed to create fk_api_key_logs_user: %v", err)
		}
	}

	log.Println("Migration 085 completed successfully")
	return nil
}

package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Migration_145_FixBoundUserIDUUID converts bound_user_id from username to UUID
//
// Context: Data scope filtering was failing because bound_user_id stored usernames
// but the filter compared against user UUIDs. This migration converts existing data.
//
// Changes:
// - Updates sys_vdi_vm.bound_user_id from username string to user UUID
// - Only affects VMs where bound_user_id is currently a username (not UUID format)
// - Null/empty bound_user_id values remain unchanged
//
// Rollback: Not easily reversible (username lost after conversion)
func Migration_145_FixBoundUserIDUUID(db *gorm.DB) error {
	log.Println("Running Migration 145: Fix bound_user_id UUID conversion...")

	// Check if migration already ran (look for UUID-format bound_user_id)
	var count int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM sys_vdi_vm
		WHERE bound_user_id IS NOT NULL
		AND bound_user_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
	`).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	// If we already have UUID-format bound_user_ids, assume migration ran
	if count > 0 {
		log.Printf("Migration 145: Found %d UUID-format bound_user_id(s), skipping (assumes already migrated)", count)
		return nil
	}

	// Update bound_user_id from username to UUID
	// Strategy: For each VM with a username in bound_user_id, look up the user's UUID
	result := db.Exec(`
		UPDATE sys_vdi_vm vm
		SET bound_user_id = u.id,
			bound_user_name = CASE
				WHEN u.nickname IS NULL OR u.nickname = '' THEN u.username
				ELSE CONCAT(u.nickname, ' (', u.username, ')')
			END
		FROM sys_user u
		WHERE vm.bound_user_id IS NOT NULL
		AND vm.bound_user_id != ''
		AND vm.bound_user_id = u.username
		AND vm.deleted_at IS NULL
		AND u.deleted_at IS NULL
		AND vm.bound_user_id !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' -- Not already a UUID
	`)

	if result.Error != nil {
		return fmt.Errorf("failed to convert bound_user_id to UUID: %w", result.Error)
	}

	log.Printf("Migration 145: Converted %d VM(s) bound_user_id from username to UUID", result.RowsAffected)

	// Verify: Check for any remaining username-format bound_user_id
	var remaining int64
	if err := db.Raw(`
		SELECT COUNT(*) FROM sys_vdi_vm
		WHERE bound_user_id IS NOT NULL
		AND bound_user_id != ''
		AND bound_user_id !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
		AND deleted_at IS NULL
	`).Count(&remaining).Error; err != nil {
		log.Printf("Warning: Could not verify migration completeness: %v", err)
	} else if remaining > 0 {
		log.Printf("Warning: %d VM(s) still have username-format bound_user_id (user may not exist in sys_user)", remaining)
	}

	log.Println("Migration 145 completed successfully")
	return nil
}

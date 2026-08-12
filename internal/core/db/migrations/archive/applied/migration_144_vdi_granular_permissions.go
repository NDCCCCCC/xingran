//go:build archive_skip


package migrations

import (
	"log"

	"gorm.io/gorm"
)

// Migrate144VDIGranularPermissions adds granular VDI VM operation permissions
// Deletes vdi:vm:operate and adds vdi:vm:start, vdi:vm:stop, vdi:vm:restart, vdi:vm:delete
func Migrate144VDIGranularPermissions(db *gorm.DB) error {
	log.Println("Running migration 144: Add VDI granular permissions")

	// Wrap entire migration in transaction for atomicity
	// If any step fails, all changes are rolled back
	return db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Migrate role permissions before deleting vdi:vm:operate
		// For all roles that have vdi:vm:operate, add all 5 granular permissions
		// The 5 permissions are: vdi:vm:start, vdi:vm:stop, vdi:vm:restart, vdi:vm:sync, vdi:vm:delete

		// First, find the menu_id for vdi:vm:operate
		var operateMenuID string
		if err := tx.Raw("SELECT id FROM sys_menu WHERE perms = ? LIMIT 1", "vdi:vm:operate").Scan(&operateMenuID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Println("vdi:vm:operate permission not found, skipping role migration")
			} else {
				log.Printf("Failed to query vdi:vm:operate menu_id: %v", err)
				return err
			}
		}

		// If vdi:vm:operate exists, migrate role permissions
		if operateMenuID != "" {
			// Find all role IDs that have vdi:vm:operate permission
			var roleIDs []string
			if err := tx.Raw(`
				SELECT DISTINCT ur.role_id
				FROM sys_user_role ur
				INNER JOIN sys_role_menu rm ON ur.role_id = rm.role_id
				WHERE rm.menu_id = ?
			`, operateMenuID).Scan(&roleIDs).Error; err != nil {
				log.Printf("Failed to query roles with vdi:vm:operate: %v", err)
				return err
			}

			// Find the menu IDs for the 5 granular permissions
			var granularMenuIDs []string
			if err := tx.Raw(`
				SELECT id FROM sys_menu WHERE perms IN (
					'vdi:vm:start',
					'vdi:vm:stop',
					'vdi:vm:restart',
					'vdi:vm:sync',
					'vdi:vm:delete'
				)
			`).Scan(&granularMenuIDs).Error; err != nil {
				log.Printf("Failed to query granular permission menu IDs: %v", err)
				return err
			}

			// For each role, add all granular permissions
			for _, roleID := range roleIDs {
				for _, menuID := range granularMenuIDs {
					// Use INSERT IGNORE pattern to avoid duplicate key errors
					var exists int
					if err := tx.Raw("SELECT 1 FROM sys_role_menu WHERE role_id = ? AND menu_id = ? LIMIT 1", roleID, menuID).Scan(&exists).Error; err != nil {
						if err == gorm.ErrRecordNotFound {
							// Insert the new role-menu mapping
							if err := tx.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
								log.Printf("Failed to insert role_menu (role_id=%s, menu_id=%s): %v", roleID, menuID, err)
								return err
							}
						} else {
							log.Printf("Failed to check role_menu existence: %v", err)
							return err
						}
					}
				}
			}

			// Delete role_menu mappings for vdi:vm:operate
			if err := tx.Exec("DELETE FROM sys_role_menu WHERE menu_id = ?", operateMenuID).Error; err != nil {
				log.Printf("Failed to delete role_menu for vdi:vm:operate: %v", err)
				return err
			}

			// Delete vdi:vm:operate menu
			if err := tx.Exec("DELETE FROM sys_menu WHERE perms = ?", "vdi:vm:operate").Error; err != nil {
				log.Printf("Failed to delete vdi:vm:operate menu: %v", err)
				return err
			}
		}

		// Step 2: Add 4 new granular permissions (vdi:vm:sync already exists)
		// Parent menu ID for VM list: 770e8400-e29b-41d4-a716-446655440002
		granularPermissions := []struct {
			ID        string
			Name      string
			Perms     string
			OrderNum  int
			Remark    string
		}{
			{
				ID:       "770e8400-e29b-41d4-a716-446655440020",
				Name:     "启动虚拟机",
				Perms:    "vdi:vm:start",
				OrderNum: 10,
				Remark:   "启动虚拟机操作",
			},
			{
				ID:       "770e8400-e29b-41d4-a716-446655440021",
				Name:     "关机虚拟机",
				Perms:    "vdi:vm:stop",
				OrderNum: 11,
				Remark:   "关机虚拟机操作",
			},
			{
				ID:       "770e8400-e29b-41d4-a716-446655440022",
				Name:     "重启虚拟机",
				Perms:    "vdi:vm:restart",
				OrderNum: 12,
				Remark:   "重启虚拟机操作",
			},
			{
				ID:       "770e8400-e29b-41d4-a716-446655440023",
				Name:     "删除虚拟机",
				Perms:    "vdi:vm:delete",
				OrderNum: 13,
				Remark:   "删除虚拟机操作",
			},
		}

		parentID := "770e8400-e29b-41d4-a716-446655440002" // VM list menu

		for _, perm := range granularPermissions {
			// Check if menu already exists
			var exists int
			if err := tx.Raw("SELECT 1 FROM sys_menu WHERE id = ? LIMIT 1", perm.ID).Scan(&exists).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// Insert new menu
					if err := tx.Exec(`
						INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at)
						VALUES (?, ?, ?, 0, '', '', 'F', '1', '0', ?, '#', ?, NOW(), NOW(), NULL)
					`, perm.ID, perm.Name, parentID, perm.Perms, perm.Remark).Error; err != nil {
						log.Printf("Failed to insert menu %s: %v", perm.Perms, err)
						return err
					}
				} else {
					log.Printf("Failed to check menu existence: %v", err)
					return err
				}
			}
		}

		log.Println("Migration 144 completed successfully: Added VDI granular permissions")
		return nil
	})
}

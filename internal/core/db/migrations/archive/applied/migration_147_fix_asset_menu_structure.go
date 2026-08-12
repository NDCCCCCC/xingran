//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate147FixAssetMenuStructure fixes the asset management menu created by migration 146.
// The original migration incorrectly created the top-level menu with a page component
// and no 'C' type child sub-menu, making the menu non-expandable and non-navigable.
func Migrate147FixAssetMenuStructure(db *gorm.DB) error {
	log.Println("Running migration 147: Fix Asset Management Menu Structure")

	// Find the top-level "资产管理" menu
	var assetMenu models.Menu
	if err := db.Where("menu_name = ? AND parent_id IS NULL AND menu_type = 'M'", "资产管理").First(&assetMenu).Error; err != nil {
		log.Println("Asset management top-level menu not found, skipping migration 147...")
		return nil
	}

	// Check if "资产列表" child menu already exists
	var childCount int64
	db.Model(&models.Menu{}).Where("parent_id = ? AND menu_name = ?", assetMenu.ID, "资产列表").Count(&childCount)
	if childCount > 0 {
		log.Println("Asset list child menu already exists, skipping migration 147...")
		return nil
	}

	log.Printf("Found asset menu: %s (ID: %s), fixing structure...", assetMenu.MenuName, assetMenu.ID)

	// Step 1: Fix the top-level menu — change component to 'Layout' (directory container)
	layoutComponent := "Layout"
	db.Model(&assetMenu).Update("component", layoutComponent)
	log.Println("Updated asset menu component to 'Layout'")

	// Step 2: Create "资产列表" child menu (menu_type = 'C')
	// 关键修正:子菜单 path 必须与父菜单 path 不同,否则 routeGenerator.resolvePath
	// 会拼接出 'assets/assets' 这种无意义路径(React Router 找不到 → fallback /dashboard)。
	// 用 'list' 作为子路径(其他子菜单也是单词: dashboard / exceptions)。
	childPath := "list"
	childComponent := "operations/assets/index"
	childIcon := "UnorderedListOutlined"

	assetListMenu := &models.Menu{
		MenuName:  "资产列表",
		ParentID:  &assetMenu.ID,
		OrderNum:  1,
		Path:      &childPath,
		Component: &childComponent,
		MenuType:  models.MenuTypeMenu, // 'C' = component/page
		Visible:   models.VisibleShow,  // 1 = visible
		Status:    models.MenuStatusNormal,
		Icon:      &childIcon,
		Perms:     nil,
		Remark:    "资产列表页面",
	}

	if err := db.Create(assetListMenu).Error; err != nil {
		log.Printf("Failed to create asset list child menu: %v", err)
		return err
	}
	log.Printf("Created asset list child menu: %s (ID: %s)", assetListMenu.MenuName, assetListMenu.ID)

	// Step 3: Re-parent button permissions to be children of "资产列表" instead of "资产管理"
	result := db.Model(&models.Menu{}).
		Where("parent_id = ? AND menu_type = 'F'", assetMenu.ID).
		Update("parent_id", assetListMenu.ID)
	if result.Error != nil {
		log.Printf("Warning: failed to re-parent button permissions: %v", result.Error)
	} else {
		log.Printf("Re-parented %d button permissions under 资产列表", result.RowsAffected)
	}

	// Step 4: Assign new child menu to all active roles
	var roleIDs []string
	db.Table("sys_role").Where("status = 0").Pluck("id", &roleIDs)

	for _, roleID := range roleIDs {
		var existingCount int64
		db.Table("sys_role_menu").Where("role_id = ? AND menu_id = ?", roleID, assetListMenu.ID).Count(&existingCount)
		if existingCount == 0 {
			if err := db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, assetListMenu.ID).Error; err != nil {
				log.Printf("Failed to assign menu %s to role %s: %v", assetListMenu.ID, roleID, err)
			}
		}
	}

	log.Printf("Migration 147 completed successfully")
	return nil
}

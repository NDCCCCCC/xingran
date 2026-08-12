//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate146AddAssetMenuPermissions adds asset management menu and permissions as top-level menu
func Migrate146AddAssetMenuPermissions(db *gorm.DB) error {
	log.Println("Running migration 146: Asset Management Menu and Permissions (Top-Level)")

	// Check if menu already exists
	var count int64
	db.Table("sys_menu").Where("menu_name = ? AND parent_id IS NULL", "资产管理").Count(&count)
	if count > 0 {
		log.Println("Top-level menu '资产管理' already exists, skipping migration 146...")
		return nil
	}

	// Create asset management as top-level menu (一级菜单)
	path := "assets"
	component := "operations/assets/index"
	icon := "DatabaseOutlined"

	assetMenu := &models.Menu{
		MenuName:  "资产管理",
		ParentID:  nil, // nil = top-level menu
		OrderNum:  6,
		Path:      &path,
		Component: &component,
		MenuType:  "M", // M = 一级菜单
		Visible:   1,
		Status:    0,
		Icon:      &icon,
		Remark:    "资产管理一级菜单",
	}

	if err := db.Create(assetMenu).Error; err != nil {
		log.Printf("Failed to create asset menu: %v", err)
		return err
	}

	log.Printf("Created top-level menu: %s (ID: %s)", assetMenu.MenuName, assetMenu.ID)

	// Create button permissions
	buttons := []struct {
		name   string
		perms  string
		order  int
		remark string
	}{
		{"资产查询", "ops:asset:list", 1, "查询资产列表"},
		{"资产新增", "ops:asset:add", 2, "新增资产"},
		{"资产修改", "ops:asset:edit", 3, "修改资产信息"},
		{"资产删除", "ops:asset:delete", 4, "删除资产"},
	}

	for _, btn := range buttons {
		emptyPath := ""
		emptyIcon := ""

		buttonMenu := &models.Menu{
			MenuName:  btn.name,
			ParentID:  &assetMenu.ID,
			OrderNum:  btn.order,
			Path:      &emptyPath,
			MenuType:  "F", // F = 按钮权限
			Visible:   0,
			Status:    0,
			Perms:     &btn.perms,
			Icon:      &emptyIcon,
			Remark:    btn.remark,
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Failed to create button menu %s: %v", btn.name, err)
		} else {
			log.Printf("Created button menu: %s (ID: %s)", buttonMenu.MenuName, buttonMenu.ID)
		}
	}

	// Assign permissions to admin role (role_id with status=0)
	var roleIDs []string
	db.Table("sys_role").Where("status = 0").Pluck("id", &roleIDs)

	menuIDs := []string{assetMenu.ID}
	for _, btn := range buttons {
		var btnMenu models.Menu
		if err := db.Where("perms = ?", btn.perms).First(&btnMenu).Error; err == nil {
			menuIDs = append(menuIDs, btnMenu.ID)
		}
	}

	for _, roleID := range roleIDs {
		for _, menuID := range menuIDs {
			var existingCount int64
			db.Table("sys_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&existingCount)
			if existingCount == 0 {
				if err := db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
					log.Printf("Failed to assign menu %s to role %s: %v", menuID, roleID, err)
				}
			}
		}
	}

	log.Printf("Migration 146 completed successfully")
	return nil
}

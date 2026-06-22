package migrations

import (
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate163ADAccountPoolMenu Phase 36: AD 服务账号池菜单 + 权限
//
// 关联: migration_162_ad_service_accounts.go（建表）+ 8 个 API 端点
//       + 前端 pages/ad-domain/accounts/index.tsx
//
// 此迁移在 AD域管理 下新增子菜单「服务账号池」，并分配给所有 status=0 的角色
func Migrate163ADAccountPoolMenu(db *gorm.DB) error {
	log.Println("Running migration 163: AD Service Account Pool Menu")

	// 幂等检查
	var count int64
	db.Table("sys_menu").Where("menu_name = ?", "服务账号池").Count(&count)
	if count > 0 {
		log.Println("Menu '服务账号池' already exists, skipping migration 163...")
		return nil
	}

	// 找父菜单（AD域管理）
	var parentMenu models.Menu
	err := db.Where("menu_name = ?", "AD域管理").First(&parentMenu).Error
	if err != nil {
		log.Printf("Error: Could not find AD domain management menu: %v", err)
		return err
	}

	// 创建主菜单项
	path := "accounts"
	component := "ad-domain/accounts/index"
	perms := "ops:ad:config:account:list"
	icon := "KeyOutlined"

	accountPoolMenu := &models.Menu{
		MenuName:  "服务账号池",
		MenuType:  "C",
		ParentID:  &parentMenu.ID,
		Path:      &path,
		Component: &component,
		Visible:   1,
		Status:    0,
		Perms:     &perms,
		Icon:      &icon,
		OrderNum:  10,
		Remark:    "AD 服务账号池管理（多账号故障切换）",
	}

	if err := db.Create(accountPoolMenu).Error; err != nil {
		log.Printf("Failed to create account pool menu: %v", err)
		return err
	}
	log.Printf("Created menu: %s (ID: %s)", accountPoolMenu.MenuName, accountPoolMenu.ID)

	// 按钮权限（对应 8 个端点的 3 类权限粒度）
	buttons := []struct {
		name   string
		perms  string
		order  int
		remark string
	}{
		{"账号查询", "ops:ad:config:account:list", 1, "查看账号列表/统计"},
		{"账号新增", "ops:ad:config:account:add", 2, "新增账号"},
		{"账号编辑", "ops:ad:config:account:edit", 3, "编辑/启用停用/解锁账号"},
		{"账号删除", "ops:ad:config:account:delete", 4, "删除账号"},
	}

	for _, btn := range buttons {
		emptyPath := ""
		icon := "#"
		btnPerms := btn.perms

		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &accountPoolMenu.ID,
			Path:     &emptyPath,
			MenuType: "F",
			Visible:  0,
			Status:   0,
			Perms:    &btnPerms,
			Icon:     &icon,
			OrderNum: btn.order,
			Remark:   btn.remark,
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Failed to create button menu %s: %v", btn.name, err)
		} else {
			log.Printf("Created button menu: %s (ID: %s)", btn.name, buttonMenu.ID)
		}
	}

	// 分配给所有 status=0 的角色
	var roleIDs []string
	db.Table("sys_role").Where("status = 0").Pluck("id", &roleIDs)

	menuIDs := []string{accountPoolMenu.ID}
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

	log.Println("Migration 163 completed: AD service account pool menu ready")
	return nil
}
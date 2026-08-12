//go:build archive_skip


package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate163ADAccountPoolMenu Phase 36: AD 服务账号池按钮权限（不创建独立菜单）
//
// 设计变更: 账号池不是独立菜单，而是 AD 配置详情页的 Tabs
//
// 此迁移创建 4 个按钮权限（ops:ad:config:account:*），分配给所有 status=0 的角色
// 但**不**创建独立菜单项（避免与 AD域控配置 重复）
func Migrate163ADAccountPoolMenu(db *gorm.DB) error {
	log.Println("Running migration 163: AD Service Account Pool Permissions (no menu)")

	// 按钮权限（对应 8 个端点的 4 类权限粒度）
	// 注意：不创建独立菜单，只创建按钮权限供详情页 Tabs 使用
	buttons := []struct {
		name   string
		perms  string
		remark string
	}{
		{"账号查询", "ops:ad:config:account:list", "查看账号列表/统计"},
		{"账号新增", "ops:ad:config:account:add", "新增账号"},
		{"账号编辑", "ops:ad:config:account:edit", "编辑/启用停用/解锁账号"},
		{"账号删除", "ops:ad:config:account:delete", "删除账号"},
	}

	for _, btn := range buttons {
		var existingCount int64
		db.Table("sys_menu").Where("perms = ?", btn.perms).Count(&existingCount)
		if existingCount > 0 {
			log.Printf("Permission %s already exists, skipping", btn.perms)
			continue
		}

		emptyPath := ""
		icon := "#"
		btnPerms := btn.perms

		// 挂在 AD域控配置 菜单下（这样按钮权限归属于配置管理）
		var configMenu models.Menu
		if err := db.Where("menu_name = ? AND menu_type = ?", "AD域控配置", "C").First(&configMenu).Error; err != nil {
			log.Printf("Warning: Could not find AD域控配置 menu, skip %s: %v", btn.perms, err)
			continue
		}

		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &configMenu.ID,
			Path:     &emptyPath,
			MenuType: "F",
			Visible:  0,
			Status:   0,
			Perms:    &btnPerms,
			Icon:     &icon,
			OrderNum: 20, // 排在其他按钮之后
			Remark:   btn.remark,
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Failed to create button menu %s: %v", btn.name, err)
		} else {
			log.Printf("Created button permission: %s (ID: %s)", btn.name, buttonMenu.ID)
		}
	}

	// 分配给所有 status=0 的角色
	var roleIDs []string
	db.Table("sys_role").Where("status = 0").Pluck("id", &roleIDs)

	for _, btn := range buttons {
		var btnMenu models.Menu
		if err := db.Where("perms = ?", btn.perms).First(&btnMenu).Error; err != nil {
			continue
		}
		for _, roleID := range roleIDs {
			var existingCount int64
			db.Table("sys_role_menu").Where("role_id = ? AND menu_id = ?", roleID, btnMenu.ID).Count(&existingCount)
			if existingCount == 0 {
				if err := db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, btnMenu.ID).Error; err != nil {
					log.Printf("Failed to assign %s to role %s: %v", btn.perms, roleID, err)
				}
			}
		}
	}

	// 同时回滚上一版可能已写入的 '服务账号池' 独立菜单（如果存在）
	var standaloneMenu models.Menu
	if err := db.Where("menu_name = ? AND menu_type = ?", "服务账号池", "C").First(&standaloneMenu).Error; err == nil {
		log.Println("Cleaning up standalone '服务账号池' menu (rolled back to embedded design)")
		// 删除角色关联
		db.Exec("DELETE FROM sys_role_menu WHERE menu_id = ?", standaloneMenu.ID)
		// 删除菜单
		db.Delete(&standaloneMenu)
	}

	log.Println("Migration 163 completed: button permissions ready, no standalone menu")
	return nil
}
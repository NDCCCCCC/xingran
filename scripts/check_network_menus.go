package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

func main() {
	// 设置环境变量覆盖数据库配置
	os.Setenv("DB_HOST", "10.62.10.34")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "Cpic1234")
	os.Setenv("DB_NAME", "ruoyi")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_SSLMODE", "disable")

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	postgres, err := db.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer postgres.Close()

	db := postgres.DB

	log.Println("\n=== 检查网络设备管理父菜单 ===")
	var parentMenu models.Menu
	db.Where("menu_name = ? AND path = ?", "网络设备管理", "/network").First(&parentMenu)

	if parentMenu.ID == "" {
		log.Println("✗ 父菜单不存在！")
	} else {
		path := ""
		if parentMenu.Path != nil {
			path = *parentMenu.Path
		}
		log.Printf("✓ 父菜单: ID=%s, 名称=%s, 路径=%s, 类型=%s, 可见=%d, 状态=%d",
			parentMenu.ID, parentMenu.MenuName, path,
			parentMenu.MenuType, parentMenu.Visible, parentMenu.Status)
	}

	log.Println("\n=== 检查网络设备子菜单 ===")
	var children []models.Menu
	db.Where("path LIKE ?", "/network%").Order("order_num").Find(&children)

	log.Printf("找到 %d 个网络设备相关菜单：\n", len(children))
	for i, menu := range children {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		icon := ""
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		parentID := ""
		if menu.ParentID != nil {
			parentID = *menu.ParentID
		}
		log.Printf("  %d. %s | Path: %s | ParentID: %s | Type: %s | Visible: %d | Status: %d | Icon: %s",
			i+1, menu.MenuName, path, parentID, menu.MenuType, menu.Visible, menu.Status, icon)
	}

	log.Println("\n=== 检查管理员角色 ===")
	var adminRole models.Role
	db.Where("role_key = ?", "admin").First(&adminRole)

	if adminRole.ID == "" {
		log.Println("✗ 管理员角色不存在！")
	} else {
		log.Printf("✓ 管理员角色: ID=%s, Name=%s, Key=%s", adminRole.ID, adminRole.RoleName, adminRole.RoleKey)

		// 检查管理员是否有网络设备管理父菜单的权限
		var parentRoleMenu models.RoleMenu
		db.Where("role_id = ? AND menu_id = ?", adminRole.ID, parentMenu.ID).First(&parentRoleMenu)
		if parentRoleMenu.RoleID != "" {
			log.Printf("  ✓ 管理员有父菜单权限")
		} else {
			log.Printf("  ✗ 管理员没有父菜单权限！")
		}
	}

	log.Println("\n=== 检查管理员角色菜单关联 ===")
	if adminRole.ID != "" {
		var roleMenus []models.RoleMenu
		db.Where("role_id = ?", adminRole.ID).Find(&roleMenus)
		log.Printf("管理员共有 %d 个菜单权限", len(roleMenus))

		// 检查网络设备相关菜单的权限
		var networkMenuCount int64
		db.Table("sys_role_menu").
			Joins("INNER JOIN sys_menu ON sys_role_menu.menu_id = sys_menu.id").
			Where("role_id = ? AND sys_menu.path LIKE ?", adminRole.ID, "/network%").
			Count(&networkMenuCount)
		log.Printf("管理员有 %d 个网络设备菜单权限", networkMenuCount)
	}

	log.Println("\n=== 检查admin用户 ===")
	var adminUser models.User
	db.Where("username = ?", "admin").First(&adminUser)

	if adminUser.ID == "" {
		log.Println("✗ admin用户不存在！")
	} else {
		log.Printf("✓ admin用户: ID=%s, Name=%s", adminUser.ID, adminUser.Username)

		// 检查admin用户的角色
		var userRoles []models.UserRole
		db.Where("user_id = ?", adminUser.ID).Find(&userRoles)
		log.Printf("  admin用户有 %d 个角色：", len(userRoles))
		for _, ur := range userRoles {
			var role models.Role
			db.Where("id = ?", ur.RoleID).First(&role)
			log.Printf("    - %s (%s)", role.RoleName, role.RoleKey)
		}
	}

	log.Println("\n=== 检查所有一级菜单 ===")
	var topLevelMenus []models.Menu
	db.Where("parent_id IS NULL").Order("order_num").Find(&topLevelMenus)

	log.Printf("共有 %d 个一级菜单：\n", len(topLevelMenus))
	for _, menu := range topLevelMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		log.Printf("  - %s | Path: %s | Type: %s | Visible: %d | Status: %d",
			menu.MenuName, path, menu.MenuType, menu.Visible, menu.Status)
	}

	fmt.Println("\n==============================================")
	fmt.Println("检查完成！请查看上面的输出信息")
	fmt.Println("==============================================")
}

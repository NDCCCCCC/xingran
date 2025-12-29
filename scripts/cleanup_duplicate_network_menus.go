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

	log.Println("PostgreSQL连接成功")

	db := postgres.DB

	log.Println("\n=== 检查网络设备相关菜单 ===")

	// 查询所有网络设备相关的一级菜单
	var networkMenus []models.Menu
	db.Where("parent_id IS NULL AND (menu_name LIKE '%网络设备%' OR path LIKE '/network%')").Find(&networkMenus)

	log.Printf("找到 %d 个网络设备相关的一级菜单：\n", len(networkMenus))

	for i, menu := range networkMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		icon := ""
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		log.Printf("  %d. ID: %s | 名称: %s | 路径: %s | 图标: %s",
			i+1, menu.ID, menu.MenuName, path, icon)
	}

	if len(networkMenus) == 0 {
		log.Println("⚠ 没有找到网络设备相关菜单")
		return
	}

	// 找出正确的菜单（网络设备管理，路径为 /network）
	var correctMenu models.Menu
	for _, menu := range networkMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		if menu.MenuName == "网络设备管理" && path == "/network" {
			correctMenu = menu
			break
		}
	}

	if correctMenu.ID == "" {
		log.Println("⚠ 未找到正确的网络设备管理菜单，将创建一个")
		// 这里可以创建正确的菜单，但为了安全起见，我们先不创建
		log.Println("请先运行 create_network_parent_menu.go 脚本创建正确的菜单")
		return
	}

	log.Printf("\n✓ 找到正确的菜单: %s (ID: %s, Path: /network)\n", correctMenu.MenuName, correctMenu.ID)

	// 删除其他重复的一级菜单
	log.Println("\n=== 删除重复的一级菜单 ===")
	for _, menu := range networkMenus {
		if menu.ID != correctMenu.ID {
			log.Printf("  - 删除: %s (ID: %s)", menu.MenuName, menu.ID)

			// 先删除角色菜单关联
			db.Table("sys_role_menu").Where("menu_id = ?", menu.ID).Delete(&models.RoleMenu{})

			// 删除子菜单（包括按钮）
			var children []models.Menu
			db.Where("parent_id = ?", menu.ID).Find(&children)
			for _, child := range children {
				// 删除子菜单的子菜单
				var grandChildren []models.Menu
				db.Where("parent_id = ?", child.ID).Find(&grandChildren)
				for _, gc := range grandChildren {
					db.Table("sys_role_menu").Where("menu_id = ?", gc.ID).Delete(&models.RoleMenu{})
					db.Delete(&gc)
				}
				// 删除子菜单
				db.Table("sys_role_menu").Where("menu_id = ?", child.ID).Delete(&models.RoleMenu{})
				db.Delete(&child)
			}

			// 删除一级菜单
			db.Delete(&menu)
		}
	}

	log.Println("✓ 重复的一级菜单已删除")

	// 确保所有网络设备子菜单都挂在正确的父菜单下
	log.Println("\n=== 检查网络设备子菜单 ===")

	// 定义网络设备子菜单的正确路径
	correctPaths := map[string]string{
		"设备管理":   "/network/devices",
		"授权凭证":   "/network/credentials",
		"配置模板":   "/network/templates",
		"命令分发":   "/network/command",
		"配置执行":   "/network/executions",
		"配置备份":   "/network/backups",
		"设备发现":   "/network/discoveries",
		"MAC地址":  "/network/mac",
		"端口状态":   "/network/ports",
	}

	for menuName, correctPath := range correctPaths {
		var menu models.Menu
		db.Where("path = ?", correctPath).First(&menu)

		if menu.ID == "" {
			log.Printf("  ⚠ 未找到菜单: %s (%s)", menuName, correctPath)
			continue
		}

		// 检查父菜单是否正确
		if menu.ParentID == nil || *menu.ParentID != correctMenu.ID {
			log.Printf("  - 更新 %s 的父菜单为: %s", menuName, correctMenu.ID)
			db.Model(&menu).Update("parent_id", correctMenu.ID)
		} else {
			log.Printf("  ✓ %s 父菜单正确", menuName)
		}
	}

	// 查找没有父菜单或父菜单无效的网络设备子菜单
	log.Println("\n=== 清理孤儿子菜单 ===")
	var orphanMenus []models.Menu
	db.Where("path LIKE '/network%' AND (parent_id IS NULL OR parent_id NOT IN (SELECT id FROM sys_menu WHERE parent_id IS NULL))").
		Find(&orphanMenus)

	if len(orphanMenus) > 0 {
		log.Printf("找到 %d 个孤儿子菜单，重新挂载到正确的父菜单下：", len(orphanMenus))
		for _, menu := range orphanMenus {
			log.Printf("  - 将 %s 挂载到父菜单 %s", menu.MenuName, correctMenu.ID)
			db.Model(&menu).Update("parent_id", correctMenu.ID)
		}
	} else {
		log.Println("✓ 没有孤儿子菜单")
	}

	// 验证最终结果
	log.Println("\n=== 验证最终结果 ===")

	var finalParentMenus []models.Menu
	db.Where("parent_id IS NULL AND (menu_name LIKE '%网络设备%' OR path LIKE '/network%')").Find(&finalParentMenus)

	if len(finalParentMenus) == 1 {
		menu := finalParentMenus[0]
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		log.Printf("✓ 网络设备管理菜单已唯一: %s (Path: %s)", menu.MenuName, path)
	} else {
		log.Printf("⚠ 仍然有 %d 个网络设备一级菜单", len(finalParentMenus))
	}

	// 查询子菜单数量
	var childCount int64
	db.Where("parent_id = ?", correctMenu.ID).Count(&childCount)
	log.Printf("✓ 正确的父菜单下有 %d 个子菜单", childCount)

	// 为管理员角色添加网络设备管理菜单权限
	log.Println("\n=== 添加管理员角色菜单权限 ===")

	var adminRole models.Role
	db.Where("role_key = ?", "admin").First(&adminRole)

	if adminRole.ID != "" {
		// 获取所有网络设备相关菜单ID
		var allNetworkMenus []models.Menu
		db.Where("path LIKE ?", "/network%").Find(&allNetworkMenus)

		count := 0
		for _, menu := range allNetworkMenus {
			// 检查是否已存在关联
			var existingRoleMenu models.RoleMenu
			db.Where("role_id = ? AND menu_id = ?", adminRole.ID, menu.ID).First(&existingRoleMenu)

			// 如果不存在关联，则创建
			if existingRoleMenu.RoleID == "" {
				roleMenu := models.RoleMenu{
					RoleID: adminRole.ID,
					MenuID: menu.ID,
				}
				if err := db.Create(&roleMenu).Error; err != nil {
					log.Printf("✗ 添加菜单权限失败 %s: %v", menu.MenuName, err)
				} else {
					count++
				}
			}
		}
		log.Printf("✓ 为管理员角色添加了 %d 个菜单权限", count)
	} else {
		log.Println("⚠ 未找到管理员角色")
	}

	fmt.Println("\n网络设备菜单清理完成！")
	fmt.Println("==============================================")
	fmt.Println("重要提示:")
	fmt.Println("1. 请刷新前端页面以重新获取菜单")
	fmt.Println("2. 如果菜单仍然不正确，请清除浏览器 localStorage")
	fmt.Println("3. 在浏览器控制台运行: localStorage.clear()")
	fmt.Println("==============================================")
}

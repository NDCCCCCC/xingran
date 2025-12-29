package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
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

	log.Println("\n=== 创建网络设备管理子菜单 ===")

	// 获取父菜单
	var parentMenu models.Menu
	db.Where("menu_name = ? AND path = ?", "网络设备管理", "/network").First(&parentMenu)

	if parentMenu.ID == "" {
		log.Println("✗ 未找到网络设备管理父菜单，请先运行 create_network_parent_menu.go")
		return
	}

	log.Printf("✓ 找到父菜单: %s (ID: %s)\n", parentMenu.MenuName, parentMenu.ID)

	// 定义子菜单
	subMenus := []struct {
		name      string
		path      string
		component string
		icon      string
		orderNum  int
		perms     string
	}{
		{"设备管理", "/network/devices", "network/device/index", "cloud-server", 1, "network:device"},
		{"授权凭证", "/network/credentials", "network/credential/index", "key", 2, "network:credential"},
		{"配置模板", "/network/templates", "network/template/index", "file-text", 3, "network:template"},
		{"命令分发", "/network/command", "network/command/index", "code", 4, "network:command"},
		{"配置执行", "/network/executions", "network/execution/index", "play-circle", 5, "network:command"},
		{"配置备份", "/network/backups", "network/backup/index", "database", 6, "network:backup"},
		{"设备发现", "/network/discoveries", "network/discovery/index", "radar-chart", 7, "network:discovery"},
		{"MAC地址", "/network/mac", "network/mac/index", "audit", 8, "network:mac"},
		{"端口状态", "/network/ports", "network/port/index", "apartment", 9, "network:port"},
	}

	createdCount := 0
	for _, item := range subMenus {
		// 检查是否已存在
		var existingMenu models.Menu
		db.Where("path = ?", item.path).First(&existingMenu)

		var menuID string
		if existingMenu.ID != "" {
			// 更新现有菜单
			menuID = existingMenu.ID
			db.Model(&existingMenu).Updates(map[string]interface{}{
				"parent_id": &parentMenu.ID,
				"order_num": item.orderNum,
			})
			log.Printf("  ~ 更新菜单: %s", item.name)
		} else {
			// 创建新菜单
			menuID = uuid.New().String()
			menu := models.Menu{
				BaseModel: models.BaseModel{
					ID:        menuID,
					CreatedBy: "system",
					UpdatedBy: "system",
				},
				MenuName:  item.name,
				ParentID:  &parentMenu.ID,
				OrderNum:  item.orderNum,
				Path:      &item.path,
				Component: &item.component,
				MenuType:  models.MenuTypeMenu, // C=菜单
				Visible:   models.VisibleShow,  // 1=显示
				Status:    models.MenuStatusNormal, // 0=正常
				Icon:      &item.icon,
				Perms:     &item.perms,
			}
			if err := db.Create(&menu).Error; err != nil {
				log.Printf("  ✗ 创建 %s 失败: %v", item.name, err)
				continue
			}
			log.Printf("  + 创建菜单: %s", item.name)
			createdCount++
		}

		// 创建按钮权限
		buttons := createButtons(item.name, menuID)
		for _, button := range buttons {
			// 检查按钮是否已存在
			var existingBtn models.Menu
			db.Where("parent_id = ? AND perms = ?", menuID, button.Perms).First(&existingBtn)

			if existingBtn.ID == "" {
				if err := db.Create(&button).Error; err != nil {
					log.Printf("    ✗ 创建按钮 %s 失败: %v", button.MenuName, err)
				} else {
					log.Printf("    + 创建按钮: %s", button.MenuName)
				}
			}
		}
	}

	log.Printf("\n✓ 创建了 %d 个新菜单，更新了 %d 个现有菜单", createdCount, len(subMenus)-createdCount)

	// 为管理员角色添加菜单权限
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
					log.Printf("  ✗ 添加菜单权限失败 %s: %v", menu.MenuName, err)
				} else {
					count++
				}
			}
		}
		log.Printf("✓ 为管理员角色添加了 %d 个菜单权限", count)
	} else {
		log.Println("⚠ 未找到管理员角色")
	}

	// 验证最终结果
	log.Println("\n=== 验证最终结果 ===")

	var children []models.Menu
	db.Where("parent_id = ?", parentMenu.ID).Order("order_num").Find(&children)

	log.Printf("✓ 子菜单数量: %d", len(children))
	for _, child := range children {
		icon := ""
		if child.Icon != nil {
			icon = *child.Icon
		}
		log.Printf("  %d. %s | 路径: %s | 图标: %s", child.OrderNum, child.MenuName,
			func() string {
				if child.Path != nil {
					return *child.Path
				}
				return ""
			}(), icon)
	}

	fmt.Println("\n网络设备管理菜单初始化完成！")
	fmt.Println("==============================================")
	fmt.Println("请刷新前端页面查看菜单")
	fmt.Println("如果菜单仍然不正确，请清除浏览器 localStorage")
	fmt.Println("在浏览器控制台运行: localStorage.clear()")
	fmt.Println("==============================================")
}

func createButtons(menuName string, parentID string) []models.Menu {
	buttonMap := map[string][]struct {
		name  string
		perms string
		icon  string
	}{
		"设备管理": {
			{"查询", "network:device:list", "search"},
			{"新增", "network:device:add", "plus"},
			{"修改", "network:device:edit", "edit"},
			{"删除", "network:device:delete", "delete"},
			{"导出", "network:device:export", "export"},
		},
		"授权凭证": {
			{"查询", "network:credential:list", "search"},
			{"新增", "network:credential:add", "plus"},
			{"修改", "network:credential:edit", "edit"},
			{"删除", "network:credential:delete", "delete"},
		},
		"配置模板": {
			{"查询", "network:template:list", "search"},
			{"新增", "network:template:add", "plus"},
			{"修改", "network:template:edit", "edit"},
			{"删除", "network:template:delete", "delete"},
		},
		"命令分发": {
			{"执行", "network:command:execute", "play"},
			{"查看", "network:command:view", "eye"},
		},
		"配置执行": {
			{"执行", "network:command:execute", "play"},
			{"查看", "network:command:view", "eye"},
		},
		"配置备份": {
			{"查询", "network:backup:list", "search"},
			{"备份", "network:backup:add", "download"},
			{"恢复", "network:backup:restore", "upload"},
			{"对比", "network:backup:diff", "diff"},
		},
		"设备发现": {
			{"发现", "network:discovery:add", "radar"},
			{"查看", "network:discovery:view", "eye"},
		},
		"MAC地址": {
			{"查询", "network:mac:query", "search"},
		},
		"端口状态": {
			{"查询", "network:port:query", "search"},
		},
	}

	buttons, ok := buttonMap[menuName]
	if !ok {
		return []models.Menu{}
	}

	var result []models.Menu
	for _, btn := range buttons {
		result = append(result, models.Menu{
			BaseModel: models.BaseModel{
				ID:        uuid.New().String(),
				CreatedBy: "system",
				UpdatedBy: "system",
			},
			MenuName:  btn.name,
			ParentID:  &parentID,
			OrderNum:  0,
			MenuType:  models.MenuTypeButton, // F=按钮
			Visible:   models.VisibleHidden,  // 0=隐藏
			Status:    models.MenuStatusNormal,
			Perms:     &btn.perms,
			Icon:      &btn.icon,
		})
	}

	return result
}

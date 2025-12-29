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

	log.Println("PostgreSQL连接成功")

	db := postgres.DB

	log.Println("\n=== 创建网络设备管理父菜单 ===")

	// 1. 查找系统管理的父菜单ID（用于order_num排序参考）
	var systemMenu models.Menu
	db.Where("menu_name = ?", "系统管理").First(&systemMenu)

	baseOrderNum := 1
	if systemMenu.ID != "" {
		baseOrderNum = systemMenu.OrderNum + 10
	}

	// 2. 检查是否已存在网络设备管理父菜单
	var existingParent models.Menu
	db.Where("menu_name = ? OR path = ?", "网络设备管理", "/network").First(&existingParent)

	var parentID string
	if existingParent.ID != "" {
		parentID = existingParent.ID
		log.Printf("✓ 父菜单已存在: %s (ID: %s)", existingParent.MenuName, existingParent.ID)
	} else {
		// 3. 创建网络设备管理父菜单
		parentID = uuid.New().String()
		parentPath := "/network"
		icon := "cloud"

		parentMenu := models.Menu{
			BaseModel: models.BaseModel{
				ID:        parentID,
				CreatedBy: "system",
				UpdatedBy: "system",
			},
			MenuName: "网络设备管理",
			ParentID: nil, // 顶级菜单
			Path:     &parentPath,
			Icon:     &icon,
			MenuType: models.MenuTypeDir,     // M=目录
			Visible:  models.VisibleShow,    // 1=显示
			Status:   models.MenuStatusNormal, // 0=正常
			OrderNum: baseOrderNum,
		}

		if err := db.Create(&parentMenu).Error; err != nil {
			log.Fatalf("✗ 创建父菜单失败: %v", err)
		}
		log.Printf("✓ 创建父菜单成功: %s (ID: %s)", parentMenu.MenuName, parentMenu.ID)
	}

	// 4. 更新所有子菜单的parent_id
	log.Println("\n=== 更新子菜单关联 ===")

	childMenus := map[string]string{
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

	orderNum := 1
	for menuName, path := range childMenus {
		var menu models.Menu
		db.Where("path = ?", path).First(&menu)

		if menu.ID == "" {
			log.Printf("⚠ 未找到菜单: %s (%s)", menuName, path)
			continue
		}

		// 更新parent_id和order_num
		updates := map[string]interface{}{
			"parent_id": parentID,
			"order_num": orderNum,
		}

		result := db.Model(&models.Menu{}).Where("id = ?", menu.ID).Updates(updates)

		if result.Error != nil {
			log.Printf("✗ 更新 %s 失败: %v", menuName, result.Error)
		} else {
			log.Printf("✓ 更新 %s -> parent_id: %s, order_num: %d", menuName, parentID, orderNum)
		}
		orderNum++
	}

	// 5. 为管理员角色添加网络设备管理菜单权限
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

	// 6. 验证最终结果
	log.Println("\n=== 验证最终结果 ===")

	var finalParent models.Menu
	db.Where("id = ?", parentID).First(&finalParent)

	if finalParent.ID != "" {
		icon := ""
		if finalParent.Icon != nil {
			icon = *finalParent.Icon
		}
		path := ""
		if finalParent.Path != nil {
			path = *finalParent.Path
		}
		log.Printf("✓ 父菜单: %s | 路径: %s | 图标: %s", finalParent.MenuName, path, icon)

		// 查询所有子菜单
		var children []models.Menu
		db.Where("parent_id = ?", finalParent.ID).Order("order_num").Find(&children)

		log.Printf("✓ 子菜单数量: %d", len(children))
		for _, child := range children {
			log.Printf("  - %s (order: %d)", child.MenuName, child.OrderNum)
		}
	}

	fmt.Println("\n网络设备管理菜单创建完成！")
	fmt.Println("==============================================")
	fmt.Println("请刷新前端页面查看菜单")
	fmt.Println("==============================================")
}

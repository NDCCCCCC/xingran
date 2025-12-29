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

	log.Println("\n=== 更新网络设备管理菜单图标 ===")

	// 定义网络设备管理子菜单的图标映射
	menuIcons := map[string]string{
		"设备管理":   "cloud-server",  // 设备管理
		"授权凭证":   "key",           // 授权凭证 - 钥匙图标
		"配置模板":   "file-text",     // 配置模板 - 文档模板
		"命令分发":   "code",          // 命令分发 - 代码图标
		"配置执行":   "play-circle",   // 配置执行 - 播放/执行图标
		"配置备份":   "database",      // 配置备份 - 数据库图标
		"设备发现":   "radar-chart",   // 设备发现 - 雷达扫描图标
		"MAC地址":  "audit",         // MAC地址 - 审计/列表图标
		"端口状态":   "apartment",     // 端口状态 - 接口/端口图标
	}

	// 更新每个子菜单的图标
	for menuName, icon := range menuIcons {
		result := db.Model(&models.Menu{}).
			Where("menu_name = ?", menuName).
			Update("icon", icon)

		if result.Error != nil {
			log.Printf("✗ 更新 %s 图标失败: %v", menuName, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("✓ 更新 %s 图标: %s", menuName, icon)
		} else {
			log.Printf("⚠ 未找到菜单: %s", menuName)
		}
	}

	// 验证更新结果
	// 查询所有网络设备相关菜单
	log.Println("\n=== 查询所有网络设备相关菜单 ===")
	var allMenus []models.Menu
	db.Where("path LIKE ?", "/network%").Order("order_num").Find(&allMenus)

	for _, menu := range allMenus {
		icon := ""
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		log.Printf("  菜单名称: %-20s 路径: %-30s 图标: %s", menu.MenuName,
			func() string {
				if menu.Path != nil {
					return *menu.Path
				}
				return ""
			}(), icon)
	}

	log.Println("\n=== 验证更新结果 ===")

	// 检查是否有网络设备管理父菜单
	var parentMenu models.Menu
	db.Where("menu_name = ? OR path = ?", "网络设备管理", "/network").First(&parentMenu)

	if parentMenu.ID != "" {
		icon := ""
		if parentMenu.Icon != nil {
			icon = *parentMenu.Icon
		}
		log.Printf("✓ 找到父菜单: %s, 路径: %s, 图标: %s", parentMenu.MenuName,
			func() string {
				if parentMenu.Path != nil {
					return *parentMenu.Path
				}
				return ""
			}(), icon)
	} else {
		log.Println("⚠ 未找到网络设备管理父菜单！需要创建父菜单")
	}

	var menus []models.Menu
	db.Where("menu_name IN (?)",
		[]string{"设备管理", "授权凭证", "配置模板", "命令分发",
			"配置执行", "配置备份", "设备发现", "MAC地址", "端口状态"}).
		Find(&menus)

	if len(menus) == 0 {
		log.Println("⚠ 警告: 未找到任何网络设备管理菜单")
	} else {
		for _, menu := range menus {
			icon := ""
			if menu.Icon != nil {
				icon = *menu.Icon
			}
			log.Printf("✓ %s: %s", menu.MenuName, icon)
		}
	}

	fmt.Println("\n网络设备管理菜单图标更新完成！")
	fmt.Println("==============================================")
	fmt.Println("菜单图标映射:")
	for menuName, icon := range menuIcons {
		fmt.Printf("  %-12s -> %s\n", menuName, icon)
	}
	fmt.Println("==============================================")
}

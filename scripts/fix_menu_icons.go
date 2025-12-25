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

	log.Println("\n=== 修复菜单图标以匹配前端 iconMap ===")

	// 定义菜单图标映射 (使用前端 iconMap 中的简写键名)
	menuIcons := map[string]string{
		// 系统管理图标
		"用户管理":    "user",
		"角色管理":    "users",
		"菜单管理":    "menu",
		"部门管理":    "folder",   // ApartmentOutlined -> folder (前端没有 apartment)
		"字典管理":    "file",     // BookOutlined -> file
		"参数设置":    "setting",
		"通知公告":    "bell",
		"工位管理":    "laptop",   // DesktopOutlined -> laptop
		"岗位管理":    "check",    // IdcardOutlined -> check

		// 系统监控图标
		"监控仪表盘":   "dashboard",
		"服务监控":    "server",
		"日志管理":    "log",
		"定时任务":    "job",
		"缓存管理":    "cache",

		// 其他图标
		"仪表盘":     "dashboard",
		"系统管理":    "system",
		"系统监控":    "monitor",
	}

	// 更新菜单图标
	for menuName, icon := range menuIcons {
		result := db.Model(&models.Menu{}).
			Where("menu_name = ?", menuName).
			Update("icon", icon)

		if result.Error != nil {
			log.Printf("✗ 更新 %s 图标失败: %v", menuName, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("✓ %s: %s", menuName, icon)
		}
	}

	// 验证更新结果
	log.Println("\n=== 验证菜单图标更新结果 ===")

	var menus []models.Menu
	db.Select("menu_name", "icon").Find(&menus)

	for _, menu := range menus {
		icon := ""
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		if icon != "" {
			log.Printf("✓ %s: %s", menu.MenuName, icon)
		}
	}

	fmt.Println("\n菜单图标修复完成！")
	fmt.Println("==============================================")
	fmt.Println("所有图标名称已更新为前端 iconMap 兼容格式")
	fmt.Println("==============================================")
}

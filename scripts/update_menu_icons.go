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

	log.Println("\n=== 更新菜单图标 ===")

	// 定义菜单图标映射
	menuIcons := map[string]string{
		// 系统管理图标
		"用户管理":    "UserOutlined",
		"角色管理":    "TeamOutlined",
		"菜单管理":    "MenuOutlined",
		"部门管理":    "ApartmentOutlined",
		"字典管理":    "BookOutlined",
		"参数设置":    "SettingOutlined",
		"通知公告":    "BellOutlined",
		"工位管理":    "DesktopOutlined",
		"岗位管理":    "IdcardOutlined",

		// 系统监控图标
		"监控仪表盘":   "DashboardOutlined",
		"服务监控":    "CloudServerOutlined",
		"日志管理":    "FileTextOutlined",
		"定时任务":    "ClockCircleOutlined",
		"缓存管理":    "DatabaseOutlined",

		// 其他图标
		"仪表盘":     "DashboardFilled",
		"系统管理":    "SettingFilled",
		"系统监控":    "MonitorOutlined",
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
	db.Select("menu_name", "icon").Where("menu_name IN (?)",
		[]string{"字典管理", "参数设置", "通知公告", "工位管理"}).Find(&menus)

	for _, menu := range menus {
		icon := ""
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		log.Printf("✓ %s: %s", menu.MenuName, icon)
	}

	fmt.Println("\n菜单图标更新完成！")
	fmt.Println("==============================================")
	fmt.Println("图标映射:")
	fmt.Println("  字典管理 → BookOutlined (书籍图标)")
	fmt.Println("  参数设置 → SettingOutlined (设置图标)")
	fmt.Println("  通知公告 → BellOutlined (铃铛图标)")
	fmt.Println("  工位管理 → DesktopOutlined (桌面图标)")
	fmt.Println("==============================================")
}

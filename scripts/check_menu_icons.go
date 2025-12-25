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

	log.Println("\n=== 检查所有菜单的图标 ===")

	var menus []models.Menu
	db.Select("menu_name", "icon").Order("order_num ASC").Find(&menus)

	log.Println("\n当前菜单图标状态：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, menu := range menus {
		icon := "空"
		if menu.Icon != nil {
			icon = *menu.Icon
		}
		fmt.Printf("%-15s → %s\n", menu.MenuName, icon)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 检查报告的缺失图标的菜单
	problemMenus := []string{"角色管理", "字典管理", "服务监控", "日志管理", "定时任务"}
	log.Println("\n=== 检查报告缺失图标的菜单 ===")
	for _, name := range problemMenus {
		var menu models.Menu
		db.Where("menu_name = ?", name).First(&menu)
		if menu.Icon == nil || *menu.Icon == "" {
			log.Printf("⚠ %s: 图标为空或 NULL", name)
		} else {
			log.Printf("✓ %s: '%s'", name, *menu.Icon)
		}
	}
}

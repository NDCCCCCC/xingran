package main

import (
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

	// 查询所有父级菜单
	log.Println("\n=== 查询所有菜单 ===")
	var menus []models.Menu
	db.Where("menu_type IN ('M', 'C')").Order("order_num").Find(&menus)

	for _, menu := range menus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		component := ""
		if menu.Component != nil {
			component = *menu.Component
		}
		log.Printf("ID: %s, 名称: %s, 类型: %s, 路径: %s, 组件: %s",
			menu.ID, menu.MenuName, menu.MenuType, path, component)
	}

	// 查询新增的菜单
	log.Println("\n=== 查询新增的菜单（字典管理、参数设置、通知公告、工位管理、操作日志、登录日志）===")
	var newMenus []models.Menu
	db.Where("menu_name IN (?)", []string{"字典管理", "参数设置", "通知公告", "工位管理", "操作日志", "登录日志"}).Find(&newMenus)

	for _, menu := range newMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		log.Printf("ID: %s, 名称: %s, 父级ID: %v, 类型: %s, 路径: %s",
			menu.ID, menu.MenuName, menu.ParentID, menu.MenuType, path)
	}
}

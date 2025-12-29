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

	log.Println("\n=== 修复网络设备管理父菜单 ===")

	// 查找网络设备管理父菜单
	var parentMenu models.Menu
	db.Where("menu_name = ? AND path = ?", "网络设备管理", "/network").First(&parentMenu)

	if parentMenu.ID == "" {
		log.Println("✗ 父菜单不存在！请先运行 init_network_menus.go")
		return
	}

	log.Printf("找到父菜单: ID=%s, ParentID=%v", parentMenu.ID, parentMenu.ParentID)

	// 检查 parent_id 是否指向自己
	if parentMenu.ParentID != nil && *parentMenu.ParentID == parentMenu.ID {
		log.Println("✗ 检测到循环引用：父菜单的 parent_id 指向自己")

		// 将 parent_id 设置为 NULL
		err := db.Model(&parentMenu).Update("parent_id", nil).Error
		if err != nil {
			log.Fatalf("✗ 修复失败: %v", err)
		}
		log.Println("✓ 已将 parent_id 设置为 NULL")
	} else if parentMenu.ParentID != nil {
		log.Printf("⚠ 父菜单的 parent_id=%s 不是 NULL", *parentMenu.ParentID)

		// 检查是否真的有父菜单
		var realParent models.Menu
		db.Where("id = ?", *parentMenu.ParentID).First(&realParent)
		if realParent.ID == "" {
			log.Println("✗ 指向的父菜单不存在，将 parent_id 设置为 NULL")
			err := db.Model(&parentMenu).Update("parent_id", nil).Error
			if err != nil {
				log.Fatalf("✗ 修复失败: %v", err)
			}
			log.Println("✓ 已将 parent_id 设置为 NULL")
		} else {
			log.Printf("⚠ 父菜单确实存在: %s，这可能是正常的", realParent.MenuName)
		}
	} else {
		log.Println("✓ 父菜单的 parent_id 已经是 NULL，无需修复")
	}

	// 验证修复结果
	log.Println("\n=== 验证修复结果 ===")
	db.Where("menu_name = ? AND path = ?", "网络设备管理", "/network").First(&parentMenu)
	log.Printf("ParentID = %v", parentMenu.ParentID)

	// 检查一级菜单数量
	var topLevelCount int64
	db.Table("sys_menu").Where("parent_id IS NULL").Count(&topLevelCount)
	log.Printf("一级菜单总数: %d", topLevelCount)

	// 列出所有一级菜单
	var topLevelMenus []models.Menu
	db.Where("parent_id IS NULL").Order("order_num").Find(&topLevelMenus)
	log.Println("一级菜单列表：")
	for _, menu := range topLevelMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		log.Printf("  - %s | Path: %s | Order: %d", menu.MenuName, path, menu.OrderNum)
	}

	fmt.Println("\n==============================================")
	fmt.Println("修复完成！")
	fmt.Println("请刷新前端页面查看菜单")
	fmt.Println("==============================================")
}

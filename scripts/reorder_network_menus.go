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

	db := postgres.DB

	log.Println("\n=== 调整网络设备管理子菜单顺序 ===")

	// 定义期望的菜单顺序
	menuOrder := []struct {
		name     string
		path     string
		orderNum int
	}{
		{"设备管理", "/network/devices", 1},
		{"授权凭证", "/network/credentials", 2},
		{"配置模板", "/network/templates", 3},
		{"命令分发", "/network/command", 4},
		{"配置执行", "/network/executions", 5},
		{"配置备份", "/network/backups", 6},
		{"设备发现", "/network/discoveries", 7},
		{"MAC地址", "/network/mac", 8},
		{"端口状态", "/network/ports", 9},
	}

	for _, item := range menuOrder {
		result := db.Model(&models.Menu{}).
			Where("path = ?", item.path).
			Update("order_num", item.orderNum)

		if result.Error != nil {
			log.Printf("✗ 更新 %s 失败: %v", item.name, result.Error)
		} else {
			log.Printf("✓ %s -> order_num: %d", item.name, item.orderNum)
		}
	}

	log.Println("\n=== 验证最终结果 ===")

	var parentMenu models.Menu
	db.Where("menu_name = ?", "网络设备管理").First(&parentMenu)

	if parentMenu.ID != "" {
		// 查询所有子菜单
		var children []models.Menu
		db.Where("parent_id = ?", parentMenu.ID).Order("order_num").Find(&children)

		log.Printf("✓ 子菜单数量: %d", len(children))
		for _, child := range children {
			icon := ""
			if child.Icon != nil {
				icon = *child.Icon
			}
			log.Printf("  %d. %s | 图标: %s", child.OrderNum, child.MenuName, icon)
		}
	}

	log.Println("\n子菜单顺序调整完成！")
}

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

	// 1. 更新字典管理、参数设置、通知公告、工位管理的路径
	//    将 path 从 dict 改为 /system/dict
	log.Println("\n=== 更新系统管理菜单路径 ===")

	// 字典管理
	db.Model(&models.Menu{}).
		Where("menu_name = ? AND path = ?", "字典管理", "dict").
		Update("path", "/system/dict")
	log.Println("✓ 字典管理路径: dict → /system/dict")

	// 参数设置
	db.Model(&models.Menu{}).
		Where("menu_name = ? AND path = ?", "参数设置", "config").
		Update("path", "/system/config")
	log.Println("✓ 参数设置路径: config → /system/config")

	// 通知公告
	db.Model(&models.Menu{}).
		Where("menu_name = ? AND path = ?", "通知公告", "notice").
		Update("path", "/system/notice")
	log.Println("✓ 通知公告路径: notice → /system/notice")

	// 工位管理
	db.Model(&models.Menu{}).
		Where("menu_name = ? AND path = ?", "工位管理", "workstation").
		Update("path", "/system/workstation")
	log.Println("✓ 工位管理路径: workstation → /system/workstation")

	// 2. 删除操作日志和登录日志的独立菜单（因为已经在日志管理页面中实现了）
	log.Println("\n=== 删除重复的操作日志和登录日志菜单 ===")

	// 查找操作日志菜单
	var operLogMenu models.Menu
	db.Where("menu_name = ? AND path = ?", "操作日志", "operLog").First(&operLogMenu)
	if operLogMenu.ID != "" {
		// 先删除子菜单
		db.Where("parent_id = ?", operLogMenu.ID).Delete(&models.Menu{})
		// 删除父菜单
		db.Delete(&operLogMenu)
		log.Println("✓ 删除操作日志菜单 (operLog)")
	}

	// 查找登录日志菜单
	var loginLogMenu models.Menu
	db.Where("menu_name = ? AND path = ?", "登录日志", "loginLog").First(&loginLogMenu)
	if loginLogMenu.ID != "" {
		// 先删除子菜单
		db.Where("parent_id = ?", loginLogMenu.ID).Delete(&models.Menu{})
		// 删除父菜单
		db.Delete(&loginLogMenu)
		log.Println("✓ 删除登录日志菜单 (loginLog)")
	}

	// 3. 验证更新结果
	log.Println("\n=== 验证菜单更新结果 ===")

	var systemMenus []models.Menu
	db.Where("menu_name IN (?)", []string{"字典管理", "参数设置", "通知公告", "工位管理"}).
		Find(&systemMenus)

	for _, menu := range systemMenus {
		path := ""
		if menu.Path != nil {
			path = *menu.Path
		}
		log.Printf("✓ %s: %s", menu.MenuName, path)
	}

	// 检查是否还有操作日志和登录日志的独立菜单
	var logMenus []models.Menu
	db.Where("menu_name IN (?) AND path IN (?)", []string{"操作日志", "登录日志"}, []string{"operLog", "loginLog"}).
		Find(&logMenus)

	if len(logMenus) > 0 {
		log.Printf("⚠ 警告: 仍有 %d 个日志菜单未删除", len(logMenus))
	} else {
		log.Println("✓ 独立的操作日志和登录日志菜单已清理")
	}

	fmt.Println("\n菜单修复完成！")
	fmt.Println("==============================================")
	fmt.Println("修复内容:")
	fmt.Println("1. 字典管理、参数设置、通知公告、工位管理路径已添加 /system/ 前缀")
	fmt.Println("2. 删除了重复的操作日志和登录日志菜单")
	fmt.Println("3. 统一使用 /monitor/logs 页面管理两种日志")
	fmt.Println("==============================================")
}

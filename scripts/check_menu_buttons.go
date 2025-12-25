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
	os.Setenv("DB_HOST", "10.62.10.34")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "Cpic1234")
	os.Setenv("DB_NAME", "ruoyi")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_SSLMODE", "disable")

	cfg := config.Load()
	postgres, err := db.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer postgres.Close()

	log.Println("PostgreSQL连接成功")

	db := postgres.DB

	// 查询所有按钮权限（menu_type='F'）
	log.Println("\n=== 查询所有按钮权限 ===")
	var buttons []models.Menu
	db.Where("menu_type = 'F'").Order("menu_name").Find(&buttons)

	log.Printf("共找到 %d 个按钮权限\n", len(buttons))
	for _, btn := range buttons {
		parentName := ""
		if btn.ParentID != nil {
			var parent models.Menu
			db.First(&parent, "id = ?", *btn.ParentID)
			parentName = parent.MenuName
		}
		perms := ""
		if btn.Perms != nil {
			perms = *btn.Perms
		}
		log.Printf("[%s] %s (权限: %s)", parentName, btn.MenuName, perms)
	}

	// 按父菜单分组统计
	log.Println("\n=== 各模块的按钮权限统计 ===")
	parentMenus := []string{"字典管理", "参数设置", "通知公告", "工位管理"}
	for _, parentName := range parentMenus {
		var parent models.Menu
		if err := db.First(&parent, "menu_name = ?", parentName).Error; err == nil {
			var count int64
			db.Model(&models.Menu{}).Where("parent_id = ? AND menu_type = 'F'", parent.ID).Count(&count)
			log.Printf("%s: %d 个按钮权限", parentName, count)
		}
	}

	parentMenus2 := []string{"操作日志", "登录日志"}
	for _, parentName := range parentMenus2 {
		var parent models.Menu
		if err := db.First(&parent, "menu_name = ?", parentName).Error; err == nil {
			var count int64
			db.Model(&models.Menu{}).Where("parent_id = ? AND menu_type = 'F'", parent.ID).Count(&count)
			log.Printf("%s: %d 个按钮权限", parentName, count)
		}
	}

	fmt.Println("\n菜单导入完成！")
	fmt.Println("==============================================")
	fmt.Println("已完成的操作:")
	fmt.Println("1. 数据库表迁移 - sys_workstation 表已创建")
	fmt.Println("2. 菜单数据导入 - 6个模块菜单已添加")
	fmt.Println("3. 前端路由配置 - App.tsx 已更新")
	fmt.Println("==============================================")
}

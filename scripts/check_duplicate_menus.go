package main

import (
	"fmt"
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化核心组件
	c := core.New(cfg)
	if err := c.Init(); err != nil {
		log.Fatalf("初始化核心组件失败: %v", err)
	}
	defer c.Close()

	db := c.GetDB()

	// 查询所有系统管理下的子菜单
	type MenuInfo struct {
		ID       string
		MenuName string
		Path     string
		OrderNum int
	}

	var menus []MenuInfo
	if err := db.Table("sys_menu").
		Select("id, menu_name, path, order_num").
		Where("parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理')").
		Order("order_num ASC").
		Find(&menus).Error; err != nil {
		log.Fatalf("查询菜单失败: %v", err)
	}

	fmt.Println("=== 系统管理菜单列表 ===")
	for _, menu := range menus {
		fmt.Printf("%d. %s (path: %s, id: %s)\n", menu.OrderNum, menu.MenuName, menu.Path, menu.ID)
	}

	// 检查是否有重复的路由
	pathMap := make(map[string][]string)
	for _, menu := range menus {
		if menu.Path != "" {
			pathMap[menu.Path] = append(pathMap[menu.Path], menu.MenuName)
		}
	}

	fmt.Println("\n=== 检查重复路由 ===")
	foundDup := false
	for path, names := range pathMap {
		if len(names) > 1 {
			foundDup = true
			fmt.Printf("路由 %s 被多个菜单使用: %v\n", path, names)
		}
	}

	if !foundDup {
		fmt.Println("没有发现重复的路由")
	}
}

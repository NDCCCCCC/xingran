package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MenuInfo struct {
	ID         string
	MenuName   string
	ParentID   string
	ParentName string
	Path       string
	MenuType   string
	OrderNum   int
	Perms      string
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 查找用户提到的具体菜单
	searchNames := []string{
		"设备管理",
		"条包操作",
		"节假日查询",
		"值班池管理",
	}

	fmt.Println("=== 查找指定菜单 ===\n")

	for _, name := range searchNames {
		fmt.Printf("【%s】\n", name)
		var menus []MenuInfo
		db.Raw(`
			SELECT m.id, m.menu_name, m.parent_id,
			       p.menu_name as parent_name,
			       m.path, m.menu_type, m.order_num, m.perms
			FROM sys_menu m
			LEFT JOIN sys_menu p ON m.parent_id = p.id
			WHERE m.menu_name = ? AND m.deleted_at IS NULL
			ORDER BY m.created_at
		`, name).Scan(&menus)

		if len(menus) == 0 {
			fmt.Println("  未找到\n")
		} else {
			for i, m := range menus {
				parent := "根菜单"
				if m.ParentName != "" {
					parent = m.ParentName
				}
				fmt.Printf("  [%d] ID: %s | 父菜单: %s | 路径: %s | 类型: %s | 权限: %s\n",
					i+1, m.ID, parent, m.Path, m.MenuType, m.Perms)
			}
			fmt.Println()
		}
	}
}

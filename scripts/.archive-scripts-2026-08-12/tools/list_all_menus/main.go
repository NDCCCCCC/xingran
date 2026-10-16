package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MenuItem struct {
	ID         string
	MenuName   string
	ParentID   string
	ParentName string
	Path       string
	MenuType   string
	CreatedAt  int64
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 列出所有菜单（按名称分组）===\n")

	rows, err := db.Raw(`
		SELECT m.id, m.menu_name, m.parent_id, p.menu_name as parent_name,
		       m.path, m.menu_type, m.created_at
		FROM sys_menu m
		LEFT JOIN sys_menu p ON m.parent_id = p.id
		WHERE m.deleted_at IS NULL AND m.menu_type IN ('M', 'C')
		ORDER BY m.menu_name, m.created_at
	`).Rows()

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	currentName := ""
	groupCount := 0
	totalGroups := 0

	for rows.Next() {
		var item MenuItem
		rows.Scan(&item.ID, &item.MenuName, &item.ParentID, &item.ParentName,
			&item.Path, &item.MenuType, &item.CreatedAt)

		if item.MenuName != currentName {
			if groupCount > 1 {
				fmt.Printf("  [该组共 %d 个]\n\n", groupCount)
				totalGroups++
			}
			currentName = item.MenuName
			groupCount = 0
			fmt.Printf("【%s】\n", item.MenuName)
		}

		parent := "根菜单"
		if item.ParentName != "" {
			parent = item.ParentName
		}

		groupCount++
		fmt.Printf("  %d. ID:%s | 父菜单:%s | 路径:%s\n",
			groupCount, item.ID, parent, item.Path)
	}

	if groupCount > 1 {
		fmt.Printf("  [该组共 %d 个]\n\n", groupCount)
		totalGroups++
	}

	fmt.Printf("总计 %d 组重复菜单\n", totalGroups)
}

package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 检查所有类型菜单重复（包括按钮）===\n")

	// 统计每个菜单名称的数量
	rows, err := db.Raw(`
		SELECT menu_name, COUNT(*) as count,
		       array_agg(id::text ORDER BY id) as ids
		FROM sys_menu
		WHERE deleted_at IS NULL
		GROUP BY menu_name
		HAVING COUNT(*) > 1
		ORDER BY count DESC
	`).Rows()

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var name string
		var dupCount int
		var ids []string

		rows.Scan(&name, &dupCount, &ids)

		count++
		fmt.Printf("[%d] %s (出现 %d 次)\n", count, name, dupCount)

		// 显示每个ID的详细信息
		for _, id := range ids {
			var menuName, parentName, menuType, path string
			db.Raw(`
				SELECT m.menu_name, p.menu_name, m.menu_type, m.path
				FROM sys_menu m
				LEFT JOIN sys_menu p ON m.parent_id = p.id
				WHERE m.id = ?
			`, id).Row().Scan(&menuName, &parentName, &menuType, &path)

			parent := "根菜单"
			if parentName != "" {
				parent = parentName
			}
			fmt.Printf("    - ID:%s | 父菜单:%s | 类型:%s | 路径:%s\n",
				id, parent, menuType, path)
		}
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("✓ 数据库中没有任何重复菜单名称")
	} else {
		fmt.Printf("总计发现 %d 组重复菜单\n", count)
	}

	// 总体统计
	var totalMenus int64
	db.Table("sys_menu").Where("deleted_at IS NULL").Count(&totalMenus)
	fmt.Printf("\n当前数据库菜单总数: %d\n", totalMenus)
}

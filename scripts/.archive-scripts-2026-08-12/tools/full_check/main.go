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

	fmt.Println("=== 全面检查所有菜单重复 ===\n")

	// 查询所有菜单名称重复
	rows, err := db.Raw(`
		SELECT menu_name, COUNT(*) as dup_count,
		       array_agg(id::text ORDER BY created_at) as ids,
		       array_agg(parent_id::text ORDER BY created_at) as parent_ids
		FROM sys_menu
		WHERE deleted_at IS NULL
		  AND menu_type IN ('M', 'C')
		GROUP BY menu_name
		HAVING COUNT(*) > 1
		ORDER BY dup_count DESC
	`).Rows()

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var name string
		var dupCount int
		var ids, parentIds []string

		rows.Scan(&name, &dupCount, &ids, &parentIds)

		count++
		fmt.Printf("[%d] %s (重复 %d 次)\n", count, name, dupCount)

		// 获取详细信息
		for i, id := range ids {
			var menuName, parentName, path, menuType string
			db.Raw(`
				SELECT m.menu_name, p.menu_name, m.path, m.menu_type
				FROM sys_menu m
				LEFT JOIN sys_menu p ON m.parent_id = p.id
				WHERE m.id = ?
			`, id).Row().Scan(&menuName, &parentName, &path, &menuType)

			parent := "根菜单"
			if parentName != "" {
				parent = parentName
			}
			mark := "✓ 保留"
			if i > 0 {
				mark = "✗ 删除"
			}
			fmt.Printf("    [%s] ID: %s | 父菜单: %s | 路径: %s\n", mark, id, parent, path)
		}
		fmt.Println()
	}

	if count == 0 {
		fmt.Println("未发现重复菜单")
	}
}

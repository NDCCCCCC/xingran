package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DuplicateGroup struct {
	MenuName string
	IDs      []string
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 生成重复菜单清理脚本 ===\n")

	// 查询所有重复菜单
	var groups []DuplicateGroup
	db.Raw(`
		SELECT menu_name, array_agg(id::text ORDER BY created_at) as ids
		FROM sys_menu
		WHERE deleted_at IS NULL
		GROUP BY menu_name
		HAVING COUNT(*) > 1
		ORDER BY COUNT(*) DESC
	`).Scan(&groups)

	if len(groups) == 0 {
		fmt.Println("未发现重复菜单")
		return
	}

	fmt.Printf("发现 %d 组重复菜单\n\n", len(groups))

	// 生成清理脚本
	f, err := os.Create("scripts/cleanup_duplicate_menus.sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.WriteString("-- 清理 sys_menu 重复数据脚本\n")
	f.WriteString("-- 生成时间: 2026-05-19\n")
	f.WriteString("-- 保留规则: 每组保留创建时间最早的记录（即 IDs 数组中的第一个）\n")
	f.WriteString("-- ⚠️ 执行前请备份数据库！\n\n")

	deletedCount := 0
	for i, g := range groups {
		fmt.Printf("[%d] %s (共 %d 个)\n", i+1, g.MenuName, len(g.IDs))

		if len(g.IDs) > 1 {
			keepID := g.IDs[0]
			deleteIDs := g.IDs[1:]

			// 获取保留记录的父菜单信息
			var parentName string
			db.Raw(`
				SELECT p.menu_name FROM sys_menu m
				LEFT JOIN sys_menu p ON m.parent_id = p.id
				WHERE m.id = ?
			`, keepID).Scan(&parentName)

			if parentName == "" {
				parentName = "根菜单"
			}

			f.WriteString(fmt.Sprintf("-- %s: 保留 %s (父菜单: %s)\n", g.MenuName, keepID, parentName))

			for _, delID := range deleteIDs {
				// 获取要删除记录的父菜单信息
				var delParentName string
				db.Raw(`
					SELECT p.menu_name FROM sys_menu m
					LEFT JOIN sys_menu p ON m.parent_id = p.id
					WHERE m.id = ?
				`, delID).Scan(&delParentName)

				if delParentName == "" {
					delParentName = "根菜单"
				}

				f.WriteString(fmt.Sprintf("--   删除 %s (父菜单: %s)\n", delID, delParentName))
				f.WriteString(fmt.Sprintf("DELETE FROM sys_role_menu WHERE menu_id = '%s';\n", delID))
				f.WriteString(fmt.Sprintf("DELETE FROM sys_menu WHERE id = '%s';\n", delID))
				deletedCount++
			}
			f.WriteString("\n")
		}
	}

	f.WriteString(fmt.Sprintf("-- 总计删除 %d 条重复记录\n", deletedCount))
	fmt.Printf("\n✓ 清理脚本已生成: scripts/cleanup_duplicate_menus.sql\n")
	fmt.Printf("  预计删除 %d 条重复记录\n", deletedCount)
	fmt.Printf("\n执行前请备份:\n")
	fmt.Printf("  pg_dump -h 10.62.10.34 -U postgres -d xingran -t sys_menu > backup_sys_menu_$(date %%Y%%m%%d_%%H%%M%%S).sql\n\n")
	fmt.Printf("执行清理:\n")
	fmt.Printf("  psql -h 10.62.10.34 -U postgres -d xingran -f scripts/cleanup_duplicate_menus.sql\n")
}

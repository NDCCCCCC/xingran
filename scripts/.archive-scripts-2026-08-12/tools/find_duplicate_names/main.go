package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MenuGroup struct {
	MenuName string
	DupCount int
	IDs      []string
}

type MenuDetail struct {
	ID         string
	MenuName   string
	ParentID   string
	ParentName string
	Path       string
	MenuType   string
	OrderNum   int
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 按菜单名称统计重复 ===\n")

	// 1. 按菜单名称分组统计
	var groups []MenuGroup
	db.Raw(`
		SELECT menu_name, COUNT(*) as dup_count,
		       array_agg(id::text ORDER BY created_at) as ids
		FROM sys_menu
		WHERE deleted_at IS NULL
		GROUP BY menu_name
		HAVING COUNT(*) > 1
		ORDER BY dup_count DESC, menu_name
	`).Scan(&groups)

	if len(groups) == 0 {
		fmt.Println("未发现重复菜单名称")
		return
	}

	fmt.Printf("发现 %d 个重复菜单名称：\n\n", len(groups))

	// 2. 显示每个重复组的详细信息
	for i, g := range groups {
		fmt.Printf("[%d] 菜单名称: %s (重复 %d 次)\n", i+1, g.MenuName, g.DupCount)

		var details []MenuDetail
		db.Raw(`
			SELECT m.id, m.menu_name, m.parent_id,
			       p.menu_name as parent_name,
			       m.path, m.menu_type, m.order_num
			FROM sys_menu m
			LEFT JOIN sys_menu p ON m.parent_id = p.id
			WHERE m.id = ANY(?) AND m.deleted_at IS NULL
			ORDER BY m.created_at
		`, g.IDs).Scan(&details)

		for _, d := range details {
			parent := "根菜单"
			if d.ParentName != "" {
				parent = d.ParentName
			}
			fmt.Printf("    - ID: %s | 父菜单: %s | 路径: %s | 类型: %s\n",
				d.ID, parent, d.Path, d.MenuType)
		}
		fmt.Println()
	}

	// 3. 生成清理脚本
	fmt.Println("\n是否生成清理脚本？(y/n)")
	var answer string
	fmt.Scanln(&answer)
	if answer == "y" || answer == "Y" {
		generateCleanupScript(db, groups)
	}
}

func generateCleanupScript(db *gorm.DB, groups []MenuGroup) {
	fmt.Println("\n=== 生成清理脚本 ===")

	// 生成清理脚本：保留每个组中创建时间最早的，删除其他的
	// 同时检查是否有角色关联
}

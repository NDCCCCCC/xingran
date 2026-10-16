package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MenuDupInfo struct {
	MenuName string
	DupCount int
}

type MenuDetail struct {
	ID         string
	MenuName   string
	ParentID   string
	ParentName string
	Path       string
	MenuType   string
	OrderNum   int
	Perms      string
	CreatedAt  int64
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 查找实际菜单的重复（排除按钮权限）===\n")

	// 1. 查找菜单类型为 C (目录/菜单) 的重复
	var dups []MenuDupInfo
	db.Raw(`
		SELECT menu_name, COUNT(*) as dup_count
		FROM sys_menu
		WHERE deleted_at IS NULL
		  AND menu_type IN ('M', 'C')  -- M=目录, C=菜单
		  AND menu_name NOT IN ('查询', '查看', '修改', '删除', '新增', '导出', '导入', '下载', '上传')
		GROUP BY menu_name
		HAVING COUNT(*) > 1
		ORDER BY dup_count DESC, menu_name
	`).Scan(&dups)

	if len(dups) == 0 {
		fmt.Println("✓ 未发现重复的实际菜单")
		return
	}

	fmt.Printf("发现 %d 个重复的实际菜单：\n\n", len(dups))

	// 2. 显示每个重复的详细信息
	for i, d := range dups {
		fmt.Printf("[%d] 菜单名称: %s (重复 %d 次)\n", i+1, d.MenuName, d.DupCount)

		var details []MenuDetail
		db.Raw(`
			SELECT m.id, m.menu_name, m.parent_id,
			       p.menu_name as parent_name,
			       m.path, m.menu_type, m.order_num, m.perms, m.created_at
			FROM sys_menu m
			LEFT JOIN sys_menu p ON m.parent_id = p.id
			WHERE m.menu_name = ? AND m.deleted_at IS NULL
			ORDER BY m.created_at
		`, d.MenuName).Scan(&details)

		for j, det := range details {
			parent := "根菜单"
			if det.ParentName != "" {
				parent = det.ParentName
			}
			mark := "保留"
			if j > 0 {
				mark = "删除"
			}
			fmt.Printf("    [%s] ID: %s | 父菜单: %s | 路径: %s | 类型: %s | 权限: %s\n",
				mark, det.ID, parent, det.Path, det.MenuType, det.Perms)
		}
		fmt.Println()
	}

	// 3. 生成清理脚本
	fmt.Println("是否生成清理脚本（保留最早的，删除其余的）？(y/n)")
	var answer string
	fmt.Scanln(&answer)

	if answer == "y" || answer == "Y" {
		generateCleanupScript(db, dups)
	}
}

func generateCleanupScript(db *gorm.DB, dups []MenuDupInfo) {
	fmt.Println("\n=== 生成清理脚本 ===\n")

	// 写入文件
	f, err := os.Create("scripts/cleanup_duplicate_menus.sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.WriteString("-- 清理 sys_menu 重复数据脚本\n")
	f.WriteString("-- 保留规则：每个重复组保留创建时间最早的记录\n")
	f.WriteString("-- ⚠️ 执行前请备份数据库！\n\n")

	deletedCount := 0
	for _, d := range dups {
		var details []MenuDetail
		db.Raw(`
			SELECT id, menu_name, parent_id
			FROM sys_menu
			WHERE menu_name = ? AND deleted_at IS NULL
			ORDER BY created_at
		`, d.MenuName).Scan(&details)

		if len(details) > 1 {
			keepID := details[0].ID
			deleteIDs := details[1:]

			f.WriteString(fmt.Sprintf("-- 菜单 '%s' 的重复记录\n", d.MenuName))
			f.WriteString(fmt.Sprintf("-- 保留: %s (创建时间最早)\n", keepID))

			for _, delID := range deleteIDs {
				f.WriteString(fmt.Sprintf("-- 删除: %s\n", delID))
				f.WriteString(fmt.Sprintf("DELETE FROM sys_role_menu WHERE menu_id = '%s';\n", delID))
				f.WriteString(fmt.Sprintf("DELETE FROM sys_menu WHERE id = '%s';\n\n", delID))
				deletedCount++
			}
			f.WriteString("\n")
		}
	}

	f.WriteString(fmt.Sprintf("-- 总计删除 %d 条重复记录\n", deletedCount))
	fmt.Printf("✓ 清理脚本已生成: scripts/cleanup_duplicate_menus.sql\n")
	fmt.Printf("  预计删除 %d 条重复记录\n", deletedCount)
	fmt.Println("\n执行命令:")
	fmt.Println("  psql -h 10.62.10.34 -U postgres -d xingran -f scripts/cleanup_duplicate_menus.sql")
}

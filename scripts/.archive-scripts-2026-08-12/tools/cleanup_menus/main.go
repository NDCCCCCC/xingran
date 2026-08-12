package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Menu 菜单模型
type Menu struct {
	ID        string `gorm:"type:uuid;primary_key"`
	MenuName  string `gorm:"type:varchar(50)"`
	ParentID  string `gorm:"type:uuid"`
	Path      string `gorm:"type:varchar(200)"`
	Component string `gorm:"type:varchar(200)"`
	MenuType  string `gorm:"type:char(1)"`
	OrderNum  int
	Visible   string `gorm:"type:char(1)"`
	Status    string `gorm:"type:char(1)"`
	Perms     string `gorm:"type:varchar(100)"`
	Icon      string `gorm:"type:varchar(100)"`
	Remark    string `gorm:"type:varchar(500)"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64 `gorm:"index"`
}

// DuplicateGroup 重复菜单组
type DuplicateGroup struct {
	ParentID     string
	MenuName     string
	DupCount     int64
	DuplicateIDs []string
}

func main() {
	// 连接数据库
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 开始检查 sys_menu 重复数据 ===\n")

	// 1. 检查相同 parent_id + menu_name 的重复
	fmt.Println("【1】检查相同父菜单下的重复菜单名称:")
	var duplicates []DuplicateGroup
	db.Raw(`
		SELECT parent_id, menu_name, COUNT(*) as dup_count,
		       array_agg(id::text) as duplicate_ids
		FROM sys_menu
		WHERE deleted_at IS NULL
		GROUP BY parent_id, menu_name
		HAVING COUNT(*) > 1
		ORDER BY dup_count DESC
	`).Scan(&duplicates)

	if len(duplicates) == 0 {
		fmt.Println("  ✓ 未发现重复")
	} else {
		fmt.Printf("  ✗ 发现 %d 组重复:\n", len(duplicates))
		for _, dup := range duplicates {
			fmt.Printf("    - 父菜单ID: %s, 菜单名: %s, 重复数: %d, IDs: %v\n",
				dup.ParentID, dup.MenuName, dup.DupCount, dup.DuplicateIDs)
		}
	}

	// 2. 检查重复路径
	fmt.Println("\n【2】检查重复的路由路径:")
	type PathDuplicate struct {
		Path     string
		DupCount int64
	}
	var pathDups []PathDuplicate
	db.Raw(`
		SELECT path, COUNT(*) as dup_count
		FROM sys_menu
		WHERE deleted_at IS NULL AND path IS NOT NULL AND path != ''
		GROUP BY path
		HAVING COUNT(*) > 1
		ORDER BY dup_count DESC
	`).Scan(&pathDups)

	if len(pathDups) == 0 {
		fmt.Println("  ✓ 未发现重复路径")
	} else {
		fmt.Printf("  ✗ 发现 %d 条重复路径:\n", len(pathDups))
		for _, p := range pathDups {
			fmt.Printf("    - %s (重复 %d 次)\n", p.Path, p.DupCount)
		}
	}

	// 3. 统计总数
	var total int64
	db.Table("sys_menu").Where("deleted_at IS NULL").Count(&total)
	fmt.Printf("\n【3】当前菜单总数: %d\n", total)

	// 4. 如果有重复，询问是否清理
	if len(duplicates) > 0 || len(pathDups) > 0 {
		fmt.Println("\n=== 检测到重复数据 ===")
		fmt.Println("是否生成清理脚本？(y/n)")
		var answer string
		fmt.Scanln(&answer)

		if answer == "y" || answer == "Y" {
			generateCleanupScript(db, duplicates)
		}
	} else {
		fmt.Println("\n✓ 数据库正常，无需清理")
	}
}

func generateCleanupScript(db *gorm.DB, duplicates []DuplicateGroup) {
	_ = db // 避免未使用警告
	fmt.Println("\n=== 生成清理脚本 ===")

	// 创建清理SQL文件
	f, err := os.Create("scripts/cleanup_duplicate_menus.sql")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	f.WriteString("-- 清理 sys_menu 重复数据脚本\n")
	f.WriteString("-- ⚠️ 执行前请备份数据库！\n")
	f.WriteString("-- 备份命令: pg_dump -h 10.62.10.34 -U postgres -d xingran -t sys_menu > backup_sys_menu.sql\n\n")

	deletedCount := 0
	for _, dup := range duplicates {
		if len(dup.DuplicateIDs) > 1 {
			// 保留第一个ID，删除其余的
			keepID := dup.DuplicateIDs[0]
			deleteIDs := dup.DuplicateIDs[1:]

			f.WriteString(fmt.Sprintf("-- 菜单 '%s' (父ID: %s) 的重复记录\n", dup.MenuName, dup.ParentID))
			f.WriteString(fmt.Sprintf("-- 保留: %s\n", keepID))
			f.WriteString(fmt.Sprintf("-- 删除: %v\n", deleteIDs))

			for _, deleteID := range deleteIDs {
				// 先删除关联的角色菜单关系
				f.WriteString(fmt.Sprintf("DELETE FROM sys_role_menu WHERE menu_id = '%s';\n", deleteID))
				// 再删除菜单
				f.WriteString(fmt.Sprintf("DELETE FROM sys_menu WHERE id = '%s';\n", deleteID))
				deletedCount++
			}
			f.WriteString("\n")
		}
	}

	f.WriteString(fmt.Sprintf("-- 总计删除 %d 条重复记录\n", deletedCount))
	fmt.Printf("✓ 清理脚本已生成: scripts/cleanup_duplicate_menus.sql\n")
	fmt.Printf("  预计删除 %d 条重复记录\n", deletedCount)
	fmt.Println("\n执行前请:")
	fmt.Println("  1. 备份数据库")
	fmt.Println("  2. 检查生成的SQL脚本")
	fmt.Println("  3. 执行: psql -h 10.62.10.34 -U postgres -d xingran -f scripts/cleanup_duplicate_menus.sql")
}

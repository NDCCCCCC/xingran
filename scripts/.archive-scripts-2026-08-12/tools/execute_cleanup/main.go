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

	fmt.Println("=== 执行菜单清理 ===\n")

	// 1. 清理前统计
	fmt.Println("【清理前】")
	var beforeCount int64
	db.Table("sys_menu").Where("deleted_at IS NULL").Count(&beforeCount)
	fmt.Printf("  菜单总数: %d\n", beforeCount)

	var dupCount int64
	db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT menu_name FROM sys_menu
			WHERE deleted_at IS NULL
			GROUP BY menu_name
			HAVING COUNT(*) > 1
		) t
	`).Scan(&dupCount)
	fmt.Printf("  重复菜单组数: %d\n", dupCount)

	// 2. 执行清理
	fmt.Println("\n【开始清理...】")

	// 删除角色菜单关联
	result1 := db.Exec(`
		DELETE FROM sys_role_menu
		WHERE menu_id IN (
		    SELECT id FROM (
		        SELECT id, ROW_NUMBER() OVER (PARTITION BY menu_name ORDER BY created_at) as rn
		        FROM sys_menu
		        WHERE deleted_at IS NULL
		    ) t WHERE rn > 1
		)
	`)
	fmt.Printf("  删除角色菜单关联: %d 条\n", result1.RowsAffected)

	// 删除重复菜单
	result2 := db.Exec(`
		DELETE FROM sys_menu
		WHERE id IN (
		    SELECT id FROM (
		        SELECT id, ROW_NUMBER() OVER (PARTITION BY menu_name ORDER BY created_at) as rn
		        FROM sys_menu
		        WHERE deleted_at IS NULL
		    ) t WHERE rn > 1
		)
	`)
	fmt.Printf("  删除重复菜单: %d 条\n", result2.RowsAffected)

	// 3. 清理后统计
	fmt.Println("\n【清理后】")
	var afterCount int64
	db.Table("sys_menu").Where("deleted_at IS NULL").Count(&afterCount)
	fmt.Printf("  菜单总数: %d\n", afterCount)
	fmt.Printf("  清理数量: %d\n", beforeCount-afterCount)

	var afterDupCount int64
	db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT menu_name FROM sys_menu
			WHERE deleted_at IS NULL
			GROUP BY menu_name
			HAVING COUNT(*) > 1
		) t
	`).Scan(&afterDupCount)
	fmt.Printf("  剩余重复组: %d\n", afterDupCount)

	if afterDupCount == 0 {
		fmt.Println("\n✓ 清理完成！没有重复菜单了")
	} else {
		fmt.Println("\n⚠ 仍有重复，可能需要手动检查")
	}
}

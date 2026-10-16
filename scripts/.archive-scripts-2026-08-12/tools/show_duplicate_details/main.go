package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MenuDetail struct {
	ID       string
	MenuName string
	ParentID string
	Path     string
	MenuType string
	OrderNum int
}

func main() {
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=xingran port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	fmt.Println("=== 重复路径菜单详情 ===\n")

	// 查询重复路径的菜单详情
	var menus []MenuDetail
	db.Raw(`
		SELECT id, menu_name, parent_id, path, menu_type, order_num
		FROM sys_menu
		WHERE deleted_at IS NULL
		  AND path IN (
		    SELECT path FROM sys_menu
		    WHERE deleted_at IS NULL AND path IS NOT NULL AND path != ''
		    GROUP BY path
		    HAVING COUNT(*) > 1
		  )
		ORDER BY path, order_num
	`).Scan(&menus)

	if len(menus) == 0 {
		fmt.Println("未发现重复")
		return
	}

	currentPath := ""
	for _, m := range menus {
		if m.Path != currentPath {
			fmt.Printf("\n【路径: %s】\n", m.Path)
			currentPath = m.Path
		}
		fmt.Printf("  - ID: %s | 名称: %s | 父ID: %s | 类型: %s | 排序: %d\n",
			m.ID, m.MenuName, m.ParentID, m.MenuType, m.OrderNum)
	}
}

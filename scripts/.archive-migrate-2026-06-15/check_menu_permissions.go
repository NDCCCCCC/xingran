//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置(密码必须从环境变量 DB_PASSWORD 读取)
	host := "10.62.10.34"
	port := 5432
	user := "postgres"
	password := os.Getenv("DB_PASSWORD")
	dbname := "xingran"

	if password == "" {
		log.Fatal("环境变量 DB_PASSWORD 未设置")
	}

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	log.Println("数据库连接成功")

	// 检查隐藏菜单和权限关联
	query := `
		SELECT
			m.id,
			m.menu_name,
			m.path,
			m.visible,
			m.parent_id,
			r.role_name,
			rm.role_id
		FROM sys_menu m
		LEFT JOIN sys_role_menu rm ON m.id = rm.menu_id
		LEFT JOIN sys_role r ON rm.role_id = r.id
		WHERE m.path IN ('profile', 'settings', 'my-notices', 'user-center')
		ORDER BY m.order_num
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	log.Println("隐藏菜单及其权限关联:")
	log.Println("========================================================================")
	for rows.Next() {
		var id, menuName, path, parentID sql.NullString
		var visible sql.NullInt64
		var roleName sql.NullString
		var roleID sql.NullString

		if err := rows.Scan(&id, &menuName, &path, &visible, &parentID, &roleName, &roleID); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		roleStr := roleName.String
		if !roleName.Valid {
			roleStr = "(无权限)"
		}

		log.Printf("菜单: %s (path: %s, visible: %d) -> 角色: %s",
			menuName.String, path.String, visible.Int64, roleStr)
	}
	log.Println("========================================================================")
}

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

	// 读取 SQL 文件
	sqlFile := "internal/core/db/migrations/042_add_hidden_special_page_menus.sql"
	sqlContent, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("读取 SQL 文件失败: %v", err)
	}

	// 执行 SQL
	log.Println("开始执行迁移...")
	result, err := db.Exec(string(sqlContent))
	if err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	// 检查是否影响了行
	rowsAffected, _ := result.RowsAffected()
	log.Printf("迁移执行成功，影响行数: %d", rowsAffected)

	// 验证结果
	log.Println("验证迁移结果...")

	// 查询创建的菜单
	rows, err := db.Query(`
		SELECT menu_name, path, visible, menu_type
		FROM sys_menu
		WHERE menu_name = '用户中心'
		   OR path IN ('profile', 'settings', 'my-notices')
		ORDER BY order_num
	`)
	if err != nil {
		log.Printf("查询验证结果失败: %v", err)
	} else {
		log.Println("创建的菜单:")
		for rows.Next() {
			var menuName, path string
			var visible, menuType string
			if err := rows.Scan(&menuName, &path, &visible, &menuType); err != nil {
				log.Printf("扫描行失败: %v", err)
				continue
			}
			log.Printf("  - %s (path: %s, visible: %s, type: %s)", menuName, path, visible, menuType)
		}
		rows.Close()
	}

	log.Println("迁移完成!")
}

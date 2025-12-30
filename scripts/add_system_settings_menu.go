package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置
	host := "10.62.10.34"
	port := 5432
	user := "postgres"
	password := "Cpic1234"
	dbname := "ruoyi"

	// 构造连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	fmt.Println("成功连接到数据库")

	// 读取SQL文件
	sqlFile := "internal/core/db/migrations/008_add_system_settings_menu.sql"
	content, err := ioutil.ReadFile(sqlFile)
	if err != nil {
		log.Fatal("读取SQL文件失败:", err)
	}

	// 执行SQL
	_, err = db.Exec(string(content))
	if err != nil {
		log.Fatal("执行SQL失败:", err)
	}

	fmt.Println("迁移执行成功: 008_add_system_settings_menu.sql")

	// 验证结果
	var menuCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sys_menu WHERE menu_name = '系统设置'").Scan(&menuCount)
	if err != nil {
		log.Printf("查询菜单失败: %v\n", err)
	} else {
		fmt.Printf("找到 '系统设置' 菜单: %d 条\n", menuCount)
	}

	// 查询菜单详情
	var id, path, icon string
	err = db.QueryRow("SELECT id, path, icon FROM sys_menu WHERE menu_name = '系统设置'").Scan(&id, &path, &icon)
	if err == nil {
		fmt.Printf("菜单ID: %s\n", id)
		fmt.Printf("菜单路径: %s\n", path)
		fmt.Printf("菜单图标: %s\n", icon)
	}

	fmt.Println("\n请刷新前端页面查看效果！")
}

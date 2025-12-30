package main

import (
	"database/sql"
	"fmt"
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

	// PostgreSQL 语法创建表（PostgreSQL不支持内联COMMENT，需要单独添加）
	createTableSQLPostgres := `
	CREATE TABLE IF NOT EXISTS sys_user_preference (
		id VARCHAR(36) PRIMARY KEY,
		user_id VARCHAR(36) NOT NULL UNIQUE,
		theme VARCHAR(20) DEFAULT 'light',
		language VARCHAR(10) DEFAULT 'zh-CN',
		page_size INT DEFAULT 10,
		sidebar_collapsed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_user_id ON sys_user_preference(user_id);

	-- 添加字段注释
	COMMENT ON COLUMN sys_user_preference.user_id IS '用户ID';
	COMMENT ON COLUMN sys_user_preference.theme IS '主题：light/dark';
	COMMENT ON COLUMN sys_user_preference.language IS '语言：zh-CN/en-US';
	COMMENT ON COLUMN sys_user_preference.page_size IS '默认分页大小';
	COMMENT ON COLUMN sys_user_preference.sidebar_collapsed IS '侧边栏是否折叠';
	`

	// 执行 SQL
	_, err = db.Exec(createTableSQLPostgres)
	if err != nil {
		log.Fatal("创建表失败:", err)
	}

	fmt.Println("成功创建 sys_user_preference 表")
}

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

	database := getDB(host, port, user, password, dbname)
	defer database.Close()

	fmt.Println("数据库连接成功")

	migrationFile := "internal/core/db/migrations/126_create_dept_ou_mapping_table.sql"
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("读取迁移文件失败: %v", err)
	}

	fmt.Println("开始执行迁移: 创建部门-OU映射表...")

	result, err := database.Exec(string(content))
	if err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("迁移执行成功！影响行数: %d\n", rowsAffected)
	fmt.Println("sys_dept_ou_mapping 表已创建")

	// 验证表是否创建成功
	var tableName string
	err = database.QueryRow("SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = 'sys_dept_ou_mapping'").Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("警告: 表未找到，可能创建失败")
		} else {
			log.Fatalf("验证表创建失败: %v", err)
		}
	} else {
		fmt.Println("验证成功: 表 sys_dept_ou_mapping 已存在")
	}

	// 显示表结构
	fmt.Println("\n表结构:")
	rows, _ := database.Query("SELECT column_name, data_type, character_maximum_length FROM information_schema.columns WHERE table_name = 'sys_dept_ou_mapping' ORDER BY ordinal_position")
	defer rows.Close()
	for rows.Next() {
		var colName, dataType string
		var maxLen *int
		_ = rows.Scan(&colName, &dataType, &maxLen)
		fmt.Printf("  %s: %s\n", colName, dataType)
	}
}

func getDB(host string, port int, user, password, dbname string) *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("打开数据库连接失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	log.Printf("数据库连接成功: %s@%s:%d/%s", user, host, port, dbname)
	return db
}

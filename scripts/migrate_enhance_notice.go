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
	// 注意: 如果项目运行在Docker中，请将host改为"postgres"
	// 如果数据库运行在本地，使用"localhost"
	dbHost := "localhost"
	dbPort := 5432
	dbUser := "ruoyi"
	dbPassword := "ruoyi123"
	dbName := "ruoyi_next"

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

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

	fmt.Println("成功连接到数据库")

	// 读取SQL文件
	sqlContent, err := ioutil.ReadFile("internal/core/db/migrations/005_enhance_notice_system.sql")
	if err != nil {
		log.Fatalf("读取SQL文件失败: %v", err)
	}

	// 执行SQL
	_, err = db.Exec(string(sqlContent))
	if err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	fmt.Println("成功执行迁移: 005_enhance_notice_system.sql")

	// 验证迁移结果
	fmt.Println("\n验证迁移结果...")

	// 检查sys_notice表的新增列
	rows, err := db.Query(`
		SELECT column_name, data_type, column_default
		FROM information_schema.columns
		WHERE table_name = 'sys_notice'
		AND column_name IN ('priority', 'publish_time', 'publish_status', 'target_type', 'created_by_name', 'is_markdown')
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("查询列信息失败: %v", err)
	} else {
		fmt.Println("\nsys_notice表新增列:")
		for rows.Next() {
			var columnName, dataType, columnDefault sql.NullString
			rows.Scan(&columnName, &dataType, &columnDefault)
			fmt.Printf("  - %s: %s (default: %s)\n", columnName.String, dataType.String, columnDefault.String)
		}
		rows.Close()
	}

	// 检查新表
	newTables := []string{"sys_notice_target", "sys_notice_read", "sys_notice_attachment"}
	for _, table := range newTables {
		var exists bool
		err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
		if err == nil && exists {
			var columnCount int
			db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1", table).Scan(&columnCount)
			fmt.Printf("  ✓ %s 表已创建 (%d列)\n", table, columnCount)
		} else {
			fmt.Printf("  ✗ %s 表创建失败\n", table)
		}
	}

	fmt.Println("\n迁移完成!")
}

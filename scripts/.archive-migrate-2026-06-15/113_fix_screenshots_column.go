//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	database := getDB()
	defer database.Close()

	fmt.Println("数据库连接成功")
	fmt.Println("\n========================================")
	fmt.Println("修复 RPA 截图字段类型")
	fmt.Println("========================================")

	// 读取迁移文件
	content, err := os.ReadFile("internal/core/db/migrations/113_fix_screenshots_column_type.sql")
	if err != nil {
		log.Fatalf("读取迁移文件失败: %v", err)
	}

	// 执行迁移
	result, err := database.Exec(string(content))
	if err != nil {
		log.Printf("执行失败: %v\n", err)
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "does not exist") {
			fmt.Println("✓ 可能已修改或列不存在，请手动检查")
		}
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✓ 成功（影响 %d 行）\n", rowsAffected)
	}

	// 验证列类型
	fmt.Println("\n验证 screenshots 列类型：")
	var columnType string
	err = database.QueryRow(`
		SELECT data_type
		FROM information_schema.columns
		WHERE table_name = 'sys_rpa_executions'
		AND column_name = 'screenshots'
	`).Scan(&columnType)

	if err != nil {
		fmt.Printf("  ❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 列类型: %s\n", columnType)
		if columnType == "text" {
			fmt.Println("  ✅ 类型正确 (JSON 文本存储)")
		} else {
			fmt.Printf("  ⚠️  类型为 %s，期望为 text\n", columnType)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("迁移完成！")
	fmt.Println("========================================")
	fmt.Println("\n请重启后端服务以应用更改")
}

func getDB() *sql.DB {
	host := getEnv("DB_HOST", "10.62.10.34")
	port := getEnvInt("DB_PORT", 5432)
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "xingran")

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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var num int
		if _, err := fmt.Sscanf(value, "%d", &num); err == nil {
			return num
		}
	}
	return defaultValue
}

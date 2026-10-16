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
	fmt.Println("执行 RPA 扩缩容事件表迁移")
	fmt.Println("========================================")

	// 读取迁移文件
	content, err := os.ReadFile("../../internal/core/db/migrations/104_add_rpa_scaling_events.sql")
	if err != nil {
		log.Fatalf("读取迁移文件失败: %v", err)
	}

	// 分割 SQL 语句并逐个执行
	statements := splitSQL(string(content))
	successCount := 0
	failCount := 0

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		fmt.Printf("\n[%d/%d] 执行: %s\n", i+1, len(statements), truncateStmt(stmt, 60))

		_, err := database.Exec(stmt)
		if err != nil {
			failCount++
			log.Printf("  ❌ 失败: %v", err)
			// 某些错误是可以接受的（如表已存在、索引已存在）
			if strings.Contains(err.Error(), "already exists") {
				fmt.Println("  ℹ️  对象已存在，跳过")
			}
		} else {
			successCount++
			fmt.Println("  ✅ 成功")
		}
	}

	fmt.Println("\n========================================")
	fmt.Printf("迁移完成！成功: %d, 失败: %d\n", successCount, failCount)
	fmt.Println("========================================")

	// 验证表是否创建成功
	fmt.Println("\n验证表创建：")
	var tableName string
	err = database.QueryRow(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_name = 'sys_rpa_scaling_events'
	`).Scan(&tableName)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("  ❌ 表不存在")
		} else {
			fmt.Printf("  ❌ 查询失败: %v\n", err)
		}
	} else {
		fmt.Printf("  ✅ 表 '%s' 创建成功\n", tableName)
	}

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

// splitSQL 分割 SQL 语句
func splitSQL(sql string) []string {
	// 按分号分割，但忽略注释中的分号
	var statements []string
	var currentStmt strings.Builder
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过注释行
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")
		// 如果行以分号结尾，结束当前语句
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, currentStmt.String())
			currentStmt.Reset()
		}
	}

	// 添加最后一个语句（如果没有分号结尾）
	if currentStmt.Len() > 0 {
		statements = append(statements, currentStmt.String())
	}

	return statements
}

// truncateStmt 截断 SQL 语句用于显示
func truncateStmt(stmt string, maxLen int) string {
	stmt = strings.TrimSpace(stmt)
	stmt = strings.ReplaceAll(stmt, "\n", " ")
	if len(stmt) > maxLen {
		return stmt[:maxLen] + "..."
	}
	return stmt
}

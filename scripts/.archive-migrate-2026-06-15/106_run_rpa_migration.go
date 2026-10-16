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

	// RPA 迁移文件列表
	migrations := []struct {
		name string
		file string
	}{
		{"RPA 表结构", "../../internal/core/db/migrations/102_add_rpa_tables.sql"},
		{"RPA 菜单", "../../internal/core/db/migrations/106_add_rpa_menus.sql"},
	}

	for _, m := range migrations {
		fmt.Printf("\n========================================\n")
		fmt.Printf("执行迁移: %s\n", m.name)
		fmt.Printf("文件: %s\n", m.file)
		fmt.Printf("========================================\n")

		content, err := os.ReadFile(m.file)
		if err != nil {
			log.Fatalf("读取迁移文件失败 %s: %v", m.file, err)
		}

		// 直接执行整个 SQL 文件
		result, err := database.Exec(string(content))
		if err != nil {
			// 某些语句可能因为对象已存在而失败，这是正常的
			if strings.Contains(err.Error(), "already exists") ||
			   strings.Contains(err.Error(), "duplicate key") {
				fmt.Printf("✓ 跳过（已存在）\n")
			} else {
				log.Printf("执行失败: %v\n", err)
				log.Printf("提示: 这可能是因为 SQL 文件包含多个语句\n")
				log.Printf("请尝试直接在数据库工具中执行该文件\n")
			}
		} else {
			rowsAffected, _ := result.RowsAffected()
			fmt.Printf("✓ 成功（影响 %d 行）\n", rowsAffected)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("所有 RPA 迁移执行完成！")
	fmt.Println("========================================")
	fmt.Println("\n验证表是否创建成功：")

	// 验证表是否创建
	tables := []string{
		"sys_rpa_tasks",
		"sys_rpa_workers",
		"sys_rpa_executions",
		"sys_rpa_schedules",
		"sys_rpa_variables",
		"sys_rpa_notifications",
		"sys_rpa_audit_logs",
		"sys_rpa_templates",
	}

	for _, table := range tables {
		var exists bool
		err := database.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			fmt.Printf("  ❌ %s: 查询失败\n", table)
		} else if exists {
			fmt.Printf("  ✅ %s: 已创建\n", table)
		} else {
			fmt.Printf("  ❌ %s: 未创建\n", table)
		}
	}

	fmt.Println("\n验证菜单是否创建成功：")
	var menuExists bool
	err := database.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sys_menu
			WHERE menu_name = 'RPA 管理'
		)
	`).Scan(&menuExists)

	if err != nil {
		fmt.Printf("  ❌ RPA 管理菜单: 查询失败\n")
	} else if menuExists {
		fmt.Printf("  ✅ RPA 管理菜单: 已创建\n")
	} else {
		fmt.Printf("  ❌ RPA 管理菜单: 未创建\n")
	}

	fmt.Println("\n========================================")
	fmt.Println("请重启后端服务以加载新表和菜单")
	fmt.Println("========================================")
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
	var statements []string
	var current strings.Builder
	inParenthesis := 0
	inCreateTable := false
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过空行和纯注释行
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			current.WriteString(line)
			current.WriteString("\n")
			continue
		}

		// 检测 CREATE TABLE 开始
		if strings.HasPrefix(trimmed, "CREATE TABLE") || strings.HasPrefix(trimmed, "CREATE UNIQUE INDEX") {
			inCreateTable = true
		}

		// 计算括号
		inParenthesis += strings.Count(line, "(") - strings.Count(line, ")")

		current.WriteString(line)
		current.WriteString("\n")

		// 检查语句结束
		if strings.HasSuffix(trimmed, ";") && inParenthesis == 0 && !inCreateTable {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			inCreateTable = false
		} else if strings.HasSuffix(trimmed, ");") && inParenthesis == 0 {
			// CREATE TABLE 语句结束
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			inCreateTable = false
			inParenthesis = 0
		}
	}

	// 添加最后一个语句（可能没有分号结尾）
	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

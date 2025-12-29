package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置
	// 从环境变量读取，如果不存在则使用默认值
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "ruoyi")

	// 构造连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
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
	fmt.Println("========================================")

	// 查找所有迁移文件
	migrationsDir := "internal/core/db/migrations"
	migrationFiles, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatal("查找迁移文件失败:", err)
	}

	// 排序文件名
	sort.Strings(migrationFiles)

	fmt.Printf("找到 %d 个迁移文件\n\n", len(migrationFiles))

	// 执行每个迁移文件
	successCount := 0
	skipCount := 0
	for i, migrationFile := range migrationFiles {
		filename := filepath.Base(migrationFile)
		fmt.Printf("[%d/%d] 执行迁移: %s\n", i+1, len(migrationFiles), filename)

		// 读取 SQL 文件内容
		content, err := os.ReadFile(migrationFile)
		if err != nil {
			log.Printf("  ❌ 读取文件失败: %v\n", err)
			continue
		}

		// 分割 SQL 语句（按分号分割）
		sqlStatements := splitSQL(string(content))

		fileSuccess := 0
		for j, stmt := range sqlStatements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// 跳过注释行
			if strings.HasPrefix(stmt, "--") {
				continue
			}

			// 执行 SQL
			_, err := db.Exec(stmt)
			if err != nil {
				// 检查是否是 "已经存在" 类型的错误
				if strings.Contains(err.Error(), "already exists") ||
					strings.Contains(err.Error(), "duplicate key") ||
					strings.Contains(err.Error(), "IF NOT EXISTS") {
					// 这是可接受的错误，说明迁移已经应用过了
					log.Printf("  ⏭️  语句已存在或已应用，跳过: %s\n", truncateString(stmt, 100))
					skipCount++
					continue
				}
				log.Printf("  ❌ 执行失败 (语句 %d): %v\n", j+1, err)
				log.Printf("  SQL: %s\n", truncateString(stmt, 200))
				// 继续执行下一条语句，不中断整个迁移
				continue
			}
			fileSuccess++
		}

		if fileSuccess > 0 {
			fmt.Printf("  ✅ 成功执行 %d 条语句\n", fileSuccess)
			successCount++
		} else {
			fmt.Printf("  ⏭️  所有语句都已应用或跳过\n")
			skipCount++
		}
		fmt.Println()
	}

	fmt.Println("========================================")
	fmt.Printf("迁移执行完成！\n")
	fmt.Printf("成功执行: %d 个文件\n", successCount)
	fmt.Printf("跳过: %d 个文件\n", skipCount)

	// 验证 updated_at 字段
	fmt.Println("\n验证迁移结果...")
	var hasColumn bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'sys_config_backup'
			AND column_name = 'updated_at'
		)`).Scan(&hasColumn)
	if err != nil {
		log.Printf("验证失败: %v\n", err)
	} else if hasColumn {
		fmt.Println("✅ updated_at 字段已成功添加到 sys_config_backup 表")
	} else {
		fmt.Println("⚠️  updated_at 字段未找到，请检查迁移是否成功执行")
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// splitSQL 分割 SQL 语句
func splitSQL(content string) []string {
	// 移除注释行
	lines := strings.Split(content, "\n")
	var filteredLines []string
	inComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过单行注释
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		// 处理多行注释
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			continue
		}
		if strings.HasSuffix(trimmed, "*/") {
			inComment = false
			continue
		}
		if inComment {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	// 按分号分割
	content = strings.Join(filteredLines, "\n")
	var statements []string

	// 分割语句，考虑括号内的内容
	var currentStmt strings.Builder
	parenLevel := 0

	for _, char := range content {
		if char == '(' {
			parenLevel++
		}
		if char == ')' {
			parenLevel--
		}
		if char == ';' && parenLevel == 0 {
			statements = append(statements, currentStmt.String())
			currentStmt.Reset()
			continue
		}
		currentStmt.WriteRune(char)
	}

	if currentStmt.Len() > 0 {
		statements = append(statements, currentStmt.String())
	}

	return statements
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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
	// 数据库连接配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "xingran_next")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	fmt.Println("开始执行迁移...")

	// 执行迁移文件
	migrations := []struct {
		name string
		file string
	}{
		{"095_add_floor_area_field", "internal/core/db/migrations/095_add_floor_area_field.sql"},
		{"096_add_sys_files_deleted_at", "internal/core/db/migrations/096_add_sys_files_deleted_at.sql"},
		{"097_add_sys_file_access_logs_updated_at", "internal/core/db/migrations/097_add_sys_file_access_logs_updated_at.sql"},
	}

	for _, m := range migrations {
		fmt.Printf("\n执行迁移 %s...\n", m.name)
		if err := executeMigration(db, m.file); err != nil {
			log.Printf("警告: 迁移 %s 执行失败: %v", m.name, err)
			// 继续执行其他迁移
		} else {
			fmt.Printf("✓ 迁移 %s 执行成功\n", m.name)
		}
	}

	fmt.Println("\n所有迁移执行完成!")
}

func executeMigration(db *sql.DB, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	_, err = db.Exec(string(content))
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

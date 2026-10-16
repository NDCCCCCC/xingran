//go:build ignore
// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	database := getDB()
	defer database.Close()

	fmt.Println("数据库连接成功")

	migrationFile := "internal/core/db/migrations/074_fix_computer_unique_constraint.sql"
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("读取迁移文件失败: %v", err)
	}

	result, err := database.Exec(string(content))
	if err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("迁移执行成功！影响行数: %d\n", rowsAffected)
	fmt.Println("电脑设备唯一约束已更新为 (ad_config_id, computer_name) 组合唯一")
}

func getDB() *sql.DB {
	host := getEnv("DB_HOST", "localhost")
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

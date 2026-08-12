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

	log.Println("========== 修复用户中心父菜单路径 ==========")

	var currentPath string
	var menuID string

	query := `SELECT id, COALESCE(path, '') as path FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL LIMIT 1`
	err := database.QueryRow(query).Scan(&menuID, &currentPath)
	if err != nil {
		log.Fatalf("查询用户中心菜单失败: %v", err)
	}

	log.Printf("当前用户中心路径: '%s'\n", currentPath)

	if currentPath != "user" {
		result, err := database.Exec(`UPDATE sys_menu SET path = 'user' WHERE id = $1`, menuID)
		if err != nil {
			log.Fatalf("更新失败: %v", err)
		}
		rows, _ := result.RowsAffected()
		log.Printf("更新用户中心路径: '%s' -> 'user', 影响行数: %d\n", currentPath, rows)
	} else {
		log.Println("用户中心路径已经是 'user'，无需更新")
	}

	log.Println("\n========== 验证修复结果 ==========")
	rows, err := database.Query(`
		SELECT menu_name, path, parent_id
		FROM sys_menu
		WHERE menu_name = '用户中心' OR parent_id = $1
		ORDER BY menu_name
	`, menuID)
	if err != nil {
		log.Printf("验证查询失败: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name, path string
		var parentID sql.NullString
		rows.Scan(&name, &path, &parentID)

		parentInfo := " (父菜单)"
		if parentID.Valid {
			parentInfo = " (子菜单)"
		}
		log.Printf("  %s: path='%s'%s\n", name, path, parentInfo)
	}

	log.Println("\n========== 修复完成 ==========")
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

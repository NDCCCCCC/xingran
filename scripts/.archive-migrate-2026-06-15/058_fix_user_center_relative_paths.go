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

	log.Println("========== 修复用户中心子菜单为相对路径 ==========")

	var userCenterID string
	err := database.QueryRow(`SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL LIMIT 1`).Scan(&userCenterID)
	if err != nil {
		log.Fatalf("查询用户中心菜单失败: %v", err)
	}

	type ChildMenu struct {
		ID        string
		MenuName  string
		Path      string
		Component string
	}

	rows, err := database.Query(`
		SELECT id, menu_name, COALESCE(path, '') as path, COALESCE(component, '') as component
		FROM sys_menu
		WHERE parent_id = $1
		ORDER BY menu_name
	`, userCenterID)
	if err != nil {
		log.Fatalf("查询子菜单失败: %v", err)
	}
	defer rows.Close()

	var children []ChildMenu
	for rows.Next() {
		var c ChildMenu
		rows.Scan(&c.ID, &c.MenuName, &c.Path, &c.Component)
		children = append(children, c)
	}

	log.Printf("\n当前子菜单状态（%d 条）:\n", len(children))
	for _, c := range children {
		log.Printf("  %s: path='%s'\n", c.MenuName, c.Path)
	}

	log.Println("\n========== 执行路径修复 ==========")
	fixCount := 0

	for _, c := range children {
		var newPath string
		var needsUpdate bool

		switch c.MenuName {
		case "个人中心":
			if c.Path == "user/profile" || c.Path == "profile" {
				newPath = "profile"
				needsUpdate = true
			}
		case "用户设置":
			if c.Path == "user/settings" || c.Path == "settings" {
				newPath = "settings"
				needsUpdate = true
			}
		case "我的通知":
			if c.Path == "user/my-notices" || c.Path == "my-notices" {
				newPath = "my-notices"
				needsUpdate = true
			}
		}

		if needsUpdate && c.Path != newPath {
			result, err := database.Exec(`UPDATE sys_menu SET path = $1 WHERE id = $2`, newPath, c.ID)
			if err != nil {
				log.Printf("  [失败] %s: %v\n", c.MenuName, err)
			} else {
				log.Printf("  [成功] %s: '%s' -> '%s'\n", c.MenuName, c.Path, newPath)
				fixCount++
			}
			_ = result
		}
	}

	log.Println("\n========== 验证修复结果 ==========")
	rows2, err := database.Query(`
		SELECT menu_name, COALESCE(path, '') as path
		FROM sys_menu
		WHERE parent_id = $1
		ORDER BY menu_name
	`, userCenterID)
	if err != nil {
		log.Printf("验证查询失败: %v\n", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var name, path string
		rows2.Scan(&name, &path)
		log.Printf("  %s: path='%s'\n", name, path)
	}

	log.Printf("\n========== 修复完成，共修复 %d 条记录 ==========\n", fixCount)
	log.Println("路径说明：")
	log.Println("  - 父菜单 '用户中心': path='user'")
	log.Println("  - 子菜单使用相对路径，完整路径由前端拼接: user/profile, user/settings, user/my-notices")
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

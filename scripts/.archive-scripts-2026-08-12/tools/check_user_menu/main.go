package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Menu struct {
	ID        string
	MenuName  string
	MenuType  string
	Path      string
	ParentID  *string
	Component string
}

func main() {
	database := getDB()
	defer database.Close()

	query := `
		SELECT id, menu_name, menu_type, path, parent_id, component
		FROM sys_menu
		WHERE menu_name LIKE '%用户中心%' OR parent_id IN (
			SELECT id FROM sys_menu WHERE menu_name LIKE '%用户中心%'
		)
		ORDER BY parent_id NULLS LAST, id
	`

	rows, err := database.Query(query)
	if err != nil {
		log.Fatalf("查询菜单失败: %v", err)
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		var m Menu
		err := rows.Scan(&m.ID, &m.MenuName, &m.MenuType, &m.Path, &m.ParentID, &m.Component)
		if err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}
		menus = append(menus, m)
	}

	fmt.Println("=== 用户中心菜单信息 ===")
	for _, m := range menus {
		parentID := "NULL"
		if m.ParentID != nil {
			parentID = *m.ParentID
		}
		fmt.Printf("名称: %-20s 类型: %-3s 路径: %-25s 父ID: %-38s 组件: %s\n",
			m.MenuName, m.MenuType, m.Path, parentID, m.Component)
	}

	var userCenterID string
	for _, m := range menus {
		if m.MenuName == "用户中心" {
			userCenterID = m.ID
			fmt.Printf("\n用户中心 ID: %s, Path: %s\n", userCenterID, m.Path)
			break
		}
	}

	fmt.Println("\n=== 预期路径 vs 实际路径 ===")
	for _, m := range menus {
		if m.MenuName != "用户中心" {
			expectedPath := "user/" + m.Path
			fmt.Printf("子菜单: %-15s 实际路径: %-20s 预期路径: %s\n", m.MenuName, m.Path, expectedPath)
		}
	}
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

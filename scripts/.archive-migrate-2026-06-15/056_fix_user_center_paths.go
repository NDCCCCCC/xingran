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

type MenuInfo struct {
	ID        string
	MenuName  string
	Path      string
	Component string
	ParentID  *string
}

func main() {
	database := getDB()
	defer database.Close()

	log.Println("========== 开始修复用户中心菜单路径 ==========")

	userCenterID, err := getUserCenterID(database)
	if err != nil {
		log.Fatalf("获取用户中心菜单失败: %v", err)
	}

	if userCenterID == "" {
		log.Fatal("用户中心父菜单不存在，请先创建用户中心菜单")
	}

	log.Printf("找到用户中心菜单 ID: %s\n", userCenterID)

	children, err := getUserCenterChildren(database, userCenterID)
	if err != nil {
		log.Fatalf("查询子菜单失败: %v", err)
	}

	log.Printf("\n当前子菜单状态（%d 条）:\n", len(children))
	for _, child := range children {
		log.Printf("  - %s: path='%s', component='%s'\n", child.MenuName, child.Path, child.Component)
	}

	fixCount := fixMenuPaths(database, userCenterID, children)

	log.Println("\n========== 验证修复结果 ==========")
	updatedChildren, err := getUserCenterChildren(database, userCenterID)
	if err != nil {
		log.Fatalf("验证查询失败: %v", err)
	}

	for _, child := range updatedChildren {
		log.Printf("  - %s: path='%s', component='%s'\n", child.MenuName, child.Path, child.Component)
	}

	log.Printf("\n========== 修复完成，共修复 %d 条记录 ==========\n", fixCount)
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

func getUserCenterID(database *sql.DB) (string, error) {
	var id string
	query := `SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL LIMIT 1`
	err := database.QueryRow(query).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

func getUserCenterChildren(database *sql.DB, parentID string) ([]MenuInfo, error) {
	query := `
		SELECT id, menu_name, COALESCE(path, '') as path,
		       COALESCE(component, '') as component, parent_id
		FROM sys_menu
		WHERE parent_id = $1
		ORDER BY id
	`

	rows, err := database.Query(query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []MenuInfo
	for rows.Next() {
		var m MenuInfo
		var parentID sql.NullString
		err := rows.Scan(&m.ID, &m.MenuName, &m.Path, &m.Component, &parentID)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			m.ParentID = &parentID.String
		}
		menus = append(menus, m)
	}

	return menus, nil
}

func fixMenuPaths(database *sql.DB, parentID string, children []MenuInfo) int {
	fixCount := 0

	log.Println("\n========== 执行路径修复 ==========")

	for _, child := range children {
		var newPath, newComponent string
		var needsUpdate bool

		switch child.MenuName {
		case "个人中心":
			if child.Path == "profile" {
				newPath = "user/profile"
				needsUpdate = true
			}
			if child.Component == "" {
				newComponent = "profile/index"
				needsUpdate = true
			} else {
				newComponent = child.Component
			}

		case "系统设置", "用户设置":
			if child.Path == "settings" {
				newPath = "user/settings"
				needsUpdate = true
			}
			if child.Component == "" || child.Component == "settings/index" {
				newComponent = "system/settings-page/index"
				needsUpdate = true
			} else {
				newComponent = child.Component
			}

		case "我的通知":
			if child.Path == "my-notices" {
				newPath = "user/my-notices"
				needsUpdate = true
			}
			if child.Component == "" {
				newComponent = "my-notices/index"
				needsUpdate = true
			} else {
				newComponent = child.Component
			}

		default:
			continue
		}

		if needsUpdate {
			result, err := updateMenu(database, child.ID, newPath, newComponent)
			if err != nil {
				log.Printf("  [失败] %s: %v\n", child.MenuName, err)
			} else {
				log.Printf("  [成功] %s: path='%s' -> '%s', component='%s'\n",
					child.MenuName, child.Path, newPath, newComponent)
				fixCount++
			}
			_ = result
		}
	}

	return fixCount
}

func updateMenu(database *sql.DB, id, path, component string) (sql.Result, error) {
	query := `
		UPDATE sys_menu
		SET path = $1, component = $2
		WHERE id = $3
	`
	return database.Exec(query, path, component, id)
}

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

type MenuInfo struct {
	ID        string
	MenuName  string
	Path      string
	Component string
	MenuType  string
	ParentID  *string
	OrderNum  int
}

func main() {
	db := getDB()
	defer db.Close()

	log.Println("数据库连接成功")

	menus, err := queryAllMenus(db)
	if err != nil {
		log.Fatalf("查询菜单数据失败: %v", err)
	}

	log.Printf("查询到 %d 条菜单记录\n", len(menus))

	analyzeMenuPaths(menus)
	generateFixSQL(menus)

	fmt.Println("\n是否执行路径修复？(y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) == "y" {
		executePathFixes(db, menus)
	} else {
		log.Println("取消执行修复")
	}

	log.Println("脚本执行完成!")
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
		log.Fatalf("连接数据库失败: %v", err)
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

func queryAllMenus(db *sql.DB) ([]MenuInfo, error) {
	query := `
		SELECT id, menu_name,
		       COALESCE(path, '') as path,
		       COALESCE(component, '') as component,
		       menu_type, parent_id, order_num
		FROM sys_menu
		ORDER BY order_num
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []MenuInfo
	for rows.Next() {
		var m MenuInfo
		var parentID sql.NullString
		err := rows.Scan(&m.ID, &m.MenuName, &m.Path, &m.Component, &m.MenuType, &parentID, &m.OrderNum)
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

func analyzeMenuPaths(menus []MenuInfo) {
	log.Println("\n========== 菜单路径分析 ==========")

	hasSlash := 0
	emptyPath := 0
	componentMismatch := 0
	needsFix := 0

	log.Println("\n--- 带前导斜杠的路径 ---")
	for _, m := range menus {
		if strings.HasPrefix(m.Path, "/") {
			log.Printf("  %s: %s (组件: %s)", m.MenuName, m.Path, m.Component)
			hasSlash++
		}
	}

	log.Println("\n--- 空路径但有组件的菜单 ---")
	for _, m := range menus {
		if m.MenuType == "C" && m.Path == "" && m.Component != "" {
			log.Printf("  %s: path为空, 组件=%s", m.MenuName, m.Component)
			emptyPath++
			needsFix++
		}
	}

	log.Println("\n--- 组件路径与路由路径可能不匹配 ---")
	for _, m := range menus {
		if m.MenuType == "C" && m.Path != "" && m.Component != "" {
			expectedPath := componentToPath(m.Component)
			if expectedPath != m.Path {
				log.Printf("  %s: path='%s', 组件='%s', 期望path='%s'",
					m.MenuName, m.Path, m.Component, expectedPath)
				componentMismatch++
			}
		}
	}

	log.Println("\n--- 统计结果 ---")
	log.Printf("  带前导斜杠: %d", hasSlash)
	log.Printf("  空路径: %d", emptyPath)
	log.Printf("  组件路径不匹配: %d", componentMismatch)
	log.Printf("  需要修复总数: %d", needsFix+hasSlash)
}

func componentToPath(component string) string {
	component = strings.TrimPrefix(component, "pages/")
	component = strings.TrimPrefix(component, "/")
	component = strings.TrimPrefix(component, "src/pages/")

	component = strings.TrimSuffix(component, "/index")
	component = strings.TrimSuffix(component, ".tsx")
	component = strings.TrimSuffix(component, ".ts")

	return component
}

func pathToComponent(path string) string {
	if path == "" || path == "dashboard" {
		return "pages/dashboard-system/index"
	}

	componentMap := map[string]string{
		"system/user":                   "pages/system/user/index",
		"system/role":                   "pages/system/role/index",
		"system/menu":                   "pages/system/menu/index",
		"system/dept":                   "pages/system/dept/index",
		"system/post":                   "pages/system/post/index",
		"system/dict":                   "pages/system/dict/index",
		"system/config":                 "pages/system/config/index",
		"system/notice":                 "pages/system/notice/index",
		"system/captcha-background":     "pages/system/captcha-background/index",
		"system/settings-page":          "pages/system/settings-page/index",
		"monitor/dashboard":             "pages/monitor/dashboard/index",
		"monitor/server":                "pages/monitor/server/index",
		"monitor/cache":                 "pages/monitor/cache/index",
		"monitor/job":                   "pages/monitor/job/index",
		"monitor/logs":                  "pages/monitor/logs/index",
		"operations/buildings":          "pages/operations/buildings/index",
		"operations/floors":             "pages/operations/floors/index",
		"operations/server-rooms":       "pages/operations/server-rooms/index",
		"operations/workstations":       "pages/operations/workstations/index",
		"operations/building-spaces":    "pages/operations/building-spaces/index",
		"operations/building-spaces-3d": "pages/operations/building-spaces-3d/index",
		"operations/room-devices":       "pages/operations/room-devices/index",
		"operations/info-points":        "pages/operations/info-points/index",
		"operations/dedicated-lines":    "pages/operations/dedicated-lines/index",
		"network/devices":               "pages/network/devices/index",
		"network/ports":                 "pages/network/ports/index",
		"network/templates":             "pages/network/templates/index",
		"network/credentials":           "pages/network/credentials/index",
		"network/executions":            "pages/network/executions/index",
		"network/backups":               "pages/network/backups/index",
		"network/command":               "pages/network/command/index",
		"network/discoveries":           "pages/network/discoveries/index",
		"network/mac":                   "pages/network/mac/index",
		"workorder/orders":              "pages/workorder/orders/index",
		"workorder/categories":          "pages/workorder/categories/index",
		"workorder/statistics":          "pages/workorder/statistics/index",
		"workorder/periodic/templates":  "pages/workorder/periodic/templates/index",
		"duty/schedules":                "pages/duty/schedules/index",
		"duty/config":                   "pages/duty/config/index",
		"duty/pools":                    "pages/duty/pools/index",
		"duty/my-duty":                  "pages/duty/my-duty/index",
		"duty/holidays":                 "pages/duty/holidays/index",
		"duty/management":               "pages/duty/management/index",
		"ad-domain/configs":             "pages/ad-domain/configs/index",
		"ad-domain/users":               "pages/ad-domain/users/index",
		"ad-domain/groups":              "pages/ad-domain/groups/index",
		"ad-domain/ous":                 "pages/ad-domain/ous/index",
		"ad-domain/logs":                "pages/ad-domain/logs/index",
		"knowledge/articles":            "pages/knowledge/articles/index",
		"knowledge/view":                "pages/knowledge/view/index",
		"profile":                       "pages/profile/index",
		"settings":                      "pages/settings/index",
		"my-notices":                    "pages/my-notices/index",
	}

	if comp, ok := componentMap[path]; ok {
		return comp
	}

	return fmt.Sprintf("pages/%s/index", path)
}

func generateFixSQL(menus []MenuInfo) {
	log.Println("\n========== 生成的修复 SQL ==========")

	for _, m := range menus {
		if m.MenuType != "C" {
			continue
		}

		updates := []string{}
		newPath := ""
		newComponent := ""

		if strings.HasPrefix(m.Path, "/") {
			newPath = strings.TrimPrefix(m.Path, "/")
			updates = append(updates, fmt.Sprintf("path = '%s'", newPath))
		}

		if m.Path != "" {
			expectedComponent := pathToComponent(m.Path)
			if m.Component != expectedComponent {
				newComponent = expectedComponent
				updates = append(updates, fmt.Sprintf("component = '%s'", newComponent))
			}
		}

		if m.Path == "" && m.Component != "" {
			newPath = componentToPath(m.Component)
			updates = append(updates, fmt.Sprintf("path = '%s'", newPath))
		}

		if len(updates) > 0 {
			sql := fmt.Sprintf("UPDATE sys_menu SET %s WHERE id = '%s';",
				strings.Join(updates, ", "), m.ID)
			log.Printf("-- %s\n%s\n", m.MenuName, sql)
		}
	}
}

func executePathFixes(db *sql.DB, menus []MenuInfo) {
	log.Println("\n========== 执行路径修复 ==========")

	for _, m := range menus {
		if m.MenuType != "C" {
			continue
		}

		updates := []string{}
		values := []interface{}{}
		paramCount := 1

		if strings.HasPrefix(m.Path, "/") {
			newPath := strings.TrimPrefix(m.Path, "/")
			updates = append(updates, fmt.Sprintf("path = $%d", paramCount))
			values = append(values, newPath)
			paramCount++
		}

		if m.Path != "" {
			expectedComponent := pathToComponent(m.Path)
			if m.Component != expectedComponent {
				updates = append(updates, fmt.Sprintf("component = $%d", paramCount))
				values = append(values, expectedComponent)
				paramCount++
			}
		}

		if m.Path == "" && m.Component != "" {
			newPath := componentToPath(m.Component)
			updates = append(updates, fmt.Sprintf("path = $%d", paramCount))
			values = append(values, newPath)
			paramCount++
		}

		if len(updates) > 0 {
			query := fmt.Sprintf("UPDATE sys_menu SET %s WHERE id = $%d",
				strings.Join(updates, ", "), paramCount)
			values = append(values, m.ID)

			result, err := db.Exec(query, values...)
			if err != nil {
				log.Printf("修复 %s 失败: %v", m.MenuName, err)
			} else {
				rows, _ := result.RowsAffected()
				log.Printf("修复 %s 成功，影响 %d 行", m.MenuName, rows)
			}
		}
	}

	log.Println("\n========== 验证修复结果 ==========")
	rows, err := db.Query(`
		SELECT menu_name, path, component
		FROM sys_menu
		WHERE menu_type = 'C'
		ORDER BY order_num
	`)
	if err != nil {
		log.Printf("验证失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name, path, component string
		rows.Scan(&name, &path, &component)
		log.Printf("  %s: path='%s', component='%s'", name, path, component)
	}
}

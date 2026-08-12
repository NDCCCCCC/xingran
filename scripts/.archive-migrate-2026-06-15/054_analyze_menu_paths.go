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

type MenuPathInfo struct {
	ID          string
	MenuName    string
	ParentID    *string
	Path        string
	Component   string
	MenuType    string
	Level       int
	ParentPath  string
	ParentName  string
}

func main() {
	db := getDB()
	defer db.Close()

	log.Println("========== 菜单路径分析 ==========")

	// 构建菜单树
	menus := buildMenuTree(db)

	// 分析路径模式
	analyzePathPatterns(menus)

	// 输出建议
	printRecommendation()
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

func buildMenuTree(db *sql.DB) []MenuPathInfo {
	// 查询所有菜单
	query := `
		SELECT
			m1.id,
			m1.menu_name,
			m1.parent_id,
			COALESCE(m1.path, '') as path,
			COALESCE(m1.component, '') as component,
			m1.menu_type,
			COALESCE(m2.path, '') as parent_path,
			m2.menu_name as parent_name
		FROM sys_menu m1
		LEFT JOIN sys_menu m2 ON m1.parent_id = m2.id
		ORDER BY m1.order_num
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	var menus []MenuPathInfo
	for rows.Next() {
		var m MenuPathInfo
		var parentID, parentPath, parentName sql.NullString

		err := rows.Scan(&m.ID, &m.MenuName, &parentID, &m.Path, &m.Component,
			&m.MenuType, &parentPath, &parentName)
		if err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		if parentID.Valid {
			m.ParentID = &parentID.String
		}
		if parentPath.Valid {
			m.ParentPath = parentPath.String
		}
		if parentName.Valid {
			m.ParentName = parentName.String
		}

		// 计算层级
		m.Level = calculateLevel(m.ParentPath)

		menus = append(menus, m)
	}

	return menus
}

func calculateLevel(parentPath string) int {
	if parentPath == "" {
		return 1
	}
	return strings.Count(parentPath, "/") + 2
}

func analyzePathPatterns(menus []MenuPathInfo) {
	log.Println("\n========== 路径模式分析 ==========")

	// 按父菜单分组分析
	type PathAnalysis struct {
		ParentName  string
		ParentPath  string
		ChildName   string
		ChildPath   string
		IsDuplicate bool
		Pattern     string
	}

	var analyses []PathAnalysis

	for _, m := range menus {
		if m.MenuType != "C" && m.MenuType != "M" {
			continue
		}

		analysis := PathAnalysis{
			ParentName: m.ParentName,
			ParentPath: m.ParentPath,
			ChildName:  m.MenuName,
			ChildPath:  m.Path,
		}

		// 检查路径重复模式
		if m.ParentPath != "" && m.Path != "" {
			// 检查子路径是否以父路径的最后一个段开头
			parentSegments := strings.Split(m.ParentPath, "/")
			lastParentSegment := parentSegments[len(parentSegments)-1]

			if strings.HasPrefix(m.Path, lastParentSegment+"/") {
				analysis.IsDuplicate = true
				analysis.Pattern = fmt.Sprintf("子路径 '%s' 以父路径最后一段 '%s' 开头",
					m.Path, lastParentSegment)
			}

			// 检查完整路径重复
			expectedFullPath := m.ParentPath + "/" + m.Path
			if strings.Contains(expectedFullPath, "//") {
				// 说明有重复段
				analysis.Pattern = "会产生双斜杠路径重复"
			}
		}

		analyses = append(analyses, analysis)
	}

	// 打印有问题的路径
	log.Println("\n--- 可能存在路径重复的情况 ---")
	count := 0
	for _, a := range analyses {
		if a.IsDuplicate {
			log.Printf("父: %s (%s) -> 子: %s (%s)",
				a.ParentName, a.ParentPath, a.ChildName, a.ChildPath)
			log.Printf("  问题: %s\n", a.Pattern)
			count++
		}
	}
	log.Printf("发现 %d 个可能存在路径重复的菜单\n", count)

	// 打印路径样本
	log.Println("\n--- 当前路径存储样本（前20个） ---")
	count = 0
	for _, m := range menus {
		if m.Path != "" {
			log.Printf("[%d级] %s: path='%s', parent='%s'",
				m.Level, m.MenuName, m.Path, m.ParentPath)
			count++
			if count >= 20 {
				break
			}
		}
	}
}


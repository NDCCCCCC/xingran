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
	ParentPath  string
	ParentName  string
}

func main() {
	db := getDB()
	defer db.Close()

	log.Println("========== 菜单路径分析 ==========")

	// 查询菜单数据并分析
	analyzeMenuPaths(db)

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

func analyzeMenuPaths(db *sql.DB) {
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
		WHERE m1.menu_type IN ('M', 'C')
		ORDER BY m1.order_num
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	log.Println("\n=== 路径关系分析（前30个） ===")
	log.Println("父菜单 -> 子菜单")
	log.Println("父路径 -> 子路径")
	log.Println(strings.Repeat("-", 80))

	count := 0
	duplicateCount := 0

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

		// 只显示有父菜单的项
		if m.ParentName != "" {
			// 检查路径重复
			isDuplicate := false
			pattern := ""

			if m.ParentPath != "" && m.Path != "" {
				parentSegments := strings.Split(m.ParentPath, "/")
				lastParentSeg := parentSegments[len(parentSegments)-1]

				// 检查是否以父路径的最后一段开头
				if strings.HasPrefix(m.Path, lastParentSeg+"/") {
					isDuplicate = true
					pattern = fmt.Sprintf("子路径 '%s' 以父路径段 '%s' 开头", m.Path, lastParentSeg)
				}

				// 计算拼接后的路径
				fullPath := m.ParentPath + "/" + m.Path
				if strings.Contains(fullPath, "//") {
					isDuplicate = true
					if pattern == "" {
						pattern = "拼接会产生双斜杠"
					}
				}
			}

			if count < 30 {
				if isDuplicate {
					log.Printf("⚠️  %s -> %s", m.ParentName, m.MenuName)
					log.Printf("    %s -> %s", m.ParentPath, m.Path)
					log.Printf("    问题: %s\n", pattern)
					duplicateCount++
				} else {
					log.Printf("✓   %s -> %s", m.ParentName, m.MenuName)
					log.Printf("    %s -> %s", m.ParentPath, m.Path)
				}
				log.Println("")
			}

			if isDuplicate {
				duplicateCount++
			}
			count++
		}
	}

	log.Printf("\n总计: 检查了 %d 个父子关系\n", count)
	log.Printf("发现: %d 个路径重复问题\n", duplicateCount)
}

func printRecommendation() {
	log.Println("\n========== 统一路径规则建议 ==========")
	log.Println(`
═══════════════════════════════════════════════════════════════
建议采用：相对路径规则
═══════════════════════════════════════════════════════════════

【规则定义】
1. 每个菜单的 path 字段只存储当前层级的路径段
2. 不包含父路径，不包含前导斜杠
3. 通过父子关系在运行时拼接完整路径

【示例】
一级菜单（运维管理）:
  - path = "ops"
  - 完整路径: /ops

二级菜单（工单管理）:
  - path = "workorder"
  - 完整路径: /ops/workorder

三级菜单（工单列表）:
  - path = "orders"
  - 完整路径: /ops/workorder/orders

【数据库存储】
┌─────────────────┬─────────┬──────────────┐
│ 菜单名称         │ path    │ 组件路径     │
├─────────────────┼─────────┼──────────────┤
│ 运维管理         │ ops     │ -            │
│ 工单管理         │ workorder │ workorder/...│
│ 工单列表         │ orders  │ orders/...   │
│ 网络设备管理     │ network │ -            │
│ 设备发现         │ discoveries │ discoveries/...│
│ 设备管理         │ devices  │ devices/...   │
│ 系统管理         │ system  │ -            │
│ 用户管理         │ user    │ user/...     │
│ 角色管理         │ role    │ role/...     │
└─────────────────┴─────────┴──────────────┘

【优点】
1. 数据库存储简洁，无冗余
2. 移动子树时只需修改父节点
3. 完全避免路径重复问题
4. 代码逻辑简单清晰

【数据迁移方案】
需要将当前数据库中的路径：
  ops/workorder/workorder/orders  →  ops
  ops/workorder/workorder         →  workorder
  ops/workorder                   →  orders

  ops/duty/duty/schedules          →  ops
  ops/duty/duty                    →  duty
  ops/duty/schedules               →  schedules

改为：
  ops        →  ops
  workorder  →  workorder
  orders     →  orders

  ops        →  ops
  duty       →  duty
  schedules  →  schedules
`)
}

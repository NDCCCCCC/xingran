// 临时工具：通用只读 SQL 查询器（跑完即删）
// 用法: ./dbq <config.yaml> "<SQL>" [行数限制]
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "用法: dbq <config.yaml> <SQL> [limit]")
		os.Exit(2)
	}
	cfgPath, sqlText := os.Args[1], os.Args[2]
	limit := 100
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &limit)
	}
	host, port, user, pass, db := loadConfig(cfgPath)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, db)
	dbh, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		os.Exit(3)
	}
	// 只读护栏：仅允许 SELECT / WITH
	head := strings.ToUpper(strings.TrimSpace(sqlText))
	if !strings.HasPrefix(head, "SELECT") && !strings.HasPrefix(head, "WITH") {
		fmt.Fprintln(os.Stderr, "拒绝：只允许 SELECT/WITH 查询")
		os.Exit(4)
	}
	rows, err := dbh.Raw(sqlText + fmt.Sprintf(" LIMIT %d", limit)).Rows()
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询失败: %v\n", err)
		os.Exit(5)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	fmt.Println(strings.Join(cols, " | "))
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Fprintf(os.Stderr, "扫描失败: %v\n", err)
			continue
		}
		parts := make([]string, len(cols))
		for i, v := range vals {
			switch t := v.(type) {
			case nil:
				parts[i] = "-"
			case []byte:
				parts[i] = strings.ReplaceAll(string(t), "\n", "\\n")
			default:
				parts[i] = fmt.Sprint(t)
			}
		}
		fmt.Println(strings.Join(parts, " | "))
	}
}

func loadConfig(path string) (host, port, user, pass, db string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读配置失败: %v\n", err)
		os.Exit(2)
	}
	lines := strings.Split(string(raw), "\n")
	inDB := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, "database:") {
			inDB = true
			continue
		}
		if inDB {
			if ln != "" && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
				break
			}
			m := regexp.MustCompile(`^\s+(\w+):\s*"?([^"\n]*)"?`).FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			switch m[1] {
			case "host":
				host = strings.TrimSpace(m[2])
			case "port":
				port = strings.TrimSpace(m[2])
			case "user":
				user = strings.TrimSpace(m[2])
			case "password":
				pass = strings.TrimSpace(m[2])
			case "dbname":
				db = strings.TrimSpace(m[2])
			}
		}
	}
	return
}

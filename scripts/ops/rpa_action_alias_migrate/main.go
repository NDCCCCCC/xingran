// rpa_action_alias_migrate：对指定库执行 W7.1 / 缺陷 A 的数据迁移（261）。
//
// 用途：不重启后端也能把 sys_rpa_tasks.script 里的动作类型别名（goto/url）
// 就地改为规范写法 navigate。迁移实现与启动路径**完全同一个函数**
// （migrations.Migrate261RPActionTypeAliasNormalize），因此不会出现
// 「工具改过了但启动迁移逻辑不一样」的双实现漂移。
//
// 幂等：逐任务解析、只在命中别名时 UPDATE；重复执行零副作用。
// 安全：执行前先用 census 打印现状（本次会改哪些），执行后再打印一次结果。
//
// 用法：
//
//	go run ./scripts/ops/rpa_action_alias_migrate
//	go run ./scripts/ops/rpa_action_alias_migrate -dsn "postgres://..."
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
)

func readEnv(key string) string {
	data, _ := os.ReadFile(".env")
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.Trim(strings.TrimPrefix(line, key+"="), `"'`)
		}
	}
	return os.Getenv(key)
}

// censusAliasTypes 统计库里各动作 type 的出现次数（只读）。
func censusAliasTypes(gdb *gorm.DB) (map[string]int, error) {
	type row struct {
		ID        string
		ScriptTxt string
	}
	var rows []row
	if err := gdb.Raw(`SELECT id::text AS id, script::text AS script_txt
		FROM sys_rpa_tasks ORDER BY created_at`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, r := range rows {
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(r.ScriptTxt), &actions); err != nil {
			counts["<unparsable>"]++
			continue
		}
		for _, a := range actions {
			raw, _ := a["type"].(string)
			if raw == "" {
				raw = "<missing>"
			}
			counts[raw]++
		}
	}
	return counts, nil
}

func main() {
	dsn := flag.String("dsn", "", "直接给 DSN（覆盖 .env 拼接）")
	flag.Parse()

	if *dsn == "" {
		host := readEnv("DB_HOST")
		user := readEnv("DB_USER")
		port := readEnv("DB_PORT")
		name := readEnv("DB_NAME")
		pw := readEnv("DB_PASSWORD")
		if host == "" || pw == "" {
			fmt.Fprintln(os.Stderr, "DB_HOST / DB_PASSWORD 缺失")
			os.Exit(1)
		}
		if port == "" {
			port = "5432"
		}
		*dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=20",
			user, url.QueryEscape(pw), host, port, name)
	}

	gdb, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect failed:", err)
		os.Exit(1)
	}

	before, err := censusAliasTypes(gdb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "census(before) failed:", err)
		os.Exit(1)
	}
	fmt.Println("=== 迁移前 type 分布 ===")
	for k, v := range before {
		mark := "OK "
		if k == "goto" || k == "url" {
			mark = "ALIAS"
		}
		fmt.Printf("  %-6s %-14s %d\n", mark, k, v)
	}

	if err := migrations.Migrate261RPActionTypeAliasNormalize(gdb); err != nil {
		fmt.Fprintln(os.Stderr, "migration failed:", err)
		os.Exit(2)
	}

	after, err := censusAliasTypes(gdb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "census(after) failed:", err)
		os.Exit(1)
	}
	fmt.Println("=== 迁移后 type 分布 ===")
	for k, v := range after {
		mark := "OK "
		if k == "goto" || k == "url" {
			mark = "ALIAS(残留!)"
		}
		fmt.Printf("  %-12s %-14s %d\n", mark, k, v)
	}
	if after["goto"] > 0 || after["url"] > 0 {
		fmt.Fprintln(os.Stderr, "仍有别名字段残留，请核查")
		os.Exit(3)
	}
	fmt.Println("MIGRATION_261_OK")
}

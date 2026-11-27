// rpa_navigate_task_dump：导出库里 type="navigate" 任务的脚本片段（只读）。
//
// 用途（W7.1 缺陷 A 的对照取证）：
//
//	worker 的 navigate case 读的是 params["url"]，而后端 extractURLFromScript /
//	convertToWorkerAction 都把 URL 放在 value 字段。两种形状都可能存在于存量
//	数据里 —— 本工具把 navigate 动作的**实际键位**列出来，用来决定别名归一化
//	是否还必须同时做 value→params.url 的提升（否则 goto 归一化成 navigate 之后
//	仍然会撞上 "missing url parameter"，只是把「不支持的 type」换成「缺参数」）。
//
// 输出：每个 navigate 动作的 type/selector/value/params 键。
//
// 用法：
//
//	go run ./scripts/ops/rpa_navigate_task_dump
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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

	type row struct {
		ID        string
		Name      string
		ScriptTxt string
	}
	var rows []row
	if err := gdb.Raw(`SELECT id::text AS id, name, script::text AS script_txt
		FROM sys_rpa_tasks WHERE script::text LIKE '%navigate%' ORDER BY id`).Scan(&rows).Error; err != nil {
		fmt.Fprintln(os.Stderr, "query failed:", err)
		os.Exit(1)
	}

	urlInParams := 0
	urlInValue := 0
	urlNowhere := 0
	for _, r := range rows {
		var script []map[string]interface{}
		if err := json.Unmarshal([]byte(r.ScriptTxt), &script); err != nil {
			fmt.Fprintf(os.Stderr, "WARN 跳过 %s: %v\n", r.ID, err)
			continue
		}
		for i, a := range script {
			t, _ := a["type"].(string)
			if !strings.EqualFold(t, "navigate") {
				continue
			}
			val, _ := a["value"].(string)
			var pkeys []string
			params, _ := a["params"].(map[string]interface{})
			purl := ""
			if params != nil {
				for k := range params {
					pkeys = append(pkeys, k)
				}
				sort.Strings(pkeys)
				if u, ok := params["url"].(string); ok {
					purl = u
				}
			}
			where := "NOWHERE"
			switch {
			case purl != "":
				where = "params.url"
				urlInParams++
			case val != "":
				where = "value"
				urlInValue++
			default:
				urlNowhere++
			}
			fmt.Printf("%s | %-18s | step %d | url@%-10s | value=%q params=%v\n",
				r.ID, r.Name, i+1, where, trunc(val, 60), pkeys)
		}
	}
	fmt.Printf("\n=== navigate 动作 URL 键位分布：params.url=%d, value=%d, 都没有=%d ===\n",
		urlInParams, urlInValue, urlNowhere)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

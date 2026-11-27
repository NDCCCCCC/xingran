// rpa_action_type_census：RPA 任务脚本里「动作类型」写法的存量普查（只读）。
//
// 用途（W7.1 缺陷 A 取证）：
//
//	worker 的 ActionType 枚举只认规范写法（navigate/click/...），但存量脚本与
//	历史前端向导里出现过 goto / url 等别名。别名归一化表**必须先看库里真实
//	写法再定**，不能凭想象扩表；本工具就是那次取证，并留作后续新增别名时的
//	回归核查入口。
//
// 输出：
//  1. 每个原始 type 值的出现次数（跨任务累计）
//  2. 非规范写法所在的任务（id / name / 创建时间 / 步序号）
//  3. 结论行：需要归一化的别名集合
//
// 只读：不 UPDATE、不建表、不写任何文件。默认读 .env 的 DB_* 变量，
// 与 scripts/dbprovision 同款连接方式（PG only：本工具面向生产库）。
//
// 用法：
//
//	go run ./scripts/ops/rpa_action_type_census
//	go run ./scripts/ops/rpa_action_type_census -dsn "postgres://..."
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 规范写法：与 rpa-worker/internal/types/types.go 的 ActionType 常量表一一对应。
// 这里刻意**复制一份**而不是 import worker 模块 —— scripts 与 rpa-worker 是两个
// 独立 module，跨模块 import 会把一次性诊断工具耦合进构建图。普查工具一旦与
// 枚举漂移，下面的「非规范」判定会自我暴露（把规范写法也列为可疑）。
var canonicalActionTypes = map[string]bool{
	"navigate": true, "click": true, "fill": true, "select": true,
	"wait": true, "screenshot": true, "extract": true, "scroll": true,
	"upload": true, "download": true, "evaluate": true, "waitFor": true,
	"close": true, "loop": true, "pause": true, "condition": true,
	"autologin": true, "assert": true, "hover": true, "doubleClick": true,
	"rightClick": true, "drag": true, "keyboard": true, "switchTab": true,
	"iframe": true, "script": true,
}

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

type taskRow struct {
	ID        string
	TaskName  string
	CreatedAt time.Time
	ScriptTxt string
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
			fmt.Fprintln(os.Stderr, "DB_HOST / DB_PASSWORD 缺失（读 .env 或设置环境变量）")
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

	var rows []taskRow
	if err := gdb.Raw(`SELECT id::text AS id, name AS task_name, created_at,
		script::text AS script_txt FROM sys_rpa_tasks ORDER BY created_at`).Scan(&rows).Error; err != nil {
		fmt.Fprintln(os.Stderr, "query failed:", err)
		os.Exit(1)
	}

	counts := map[string]int{}
	stepsByType := map[string]int{}
	offenders := map[string][]string{}
	var unparsable []string

	for _, r := range rows {
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(r.ScriptTxt), &actions); err != nil {
			unparsable = append(unparsable, fmt.Sprintf("%s (%s): %v", r.ID, r.TaskName, err))
			continue
		}
		for i, a := range actions {
			raw, _ := a["type"].(string)
			if raw == "" {
				raw = "<missing>"
			}
			counts[raw]++
			stepsByType[raw]++
			if !canonicalActionTypes[raw] {
				offenders[raw] = append(offenders[raw],
					fmt.Sprintf("%s | %s | %s | step %d",
						r.ID, r.TaskName, r.CreatedAt.Format("2006-01-02 15:04"), i+1))
			}
		}
	}

	fmt.Printf("=== %d 个任务，共 %d 个动作 ===\n", len(rows), len(stepsByType))
	type kv struct {
		k string
		v int
	}
	var list []kv
	for k, v := range counts {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].v != list[j].v {
			return list[i].v > list[j].v
		}
		return list[i].k < list[j].k
	})
	for _, e := range list {
		mark := "OK "
		if !canonicalActionTypes[e.k] {
			mark = "ALIAS"
		}
		fmt.Printf("  %-6s %-14s %5d\n", mark, e.k, e.v)
	}

	if len(offenders) > 0 {
		fmt.Printf("\n=== 非规范写法明细（需要 alias 归一化 / 数据迁移）===\n")
		var okeys []string
		for k := range offenders {
			okeys = append(okeys, k)
		}
		sort.Strings(okeys)
		for _, k := range okeys {
			fmt.Printf("\n-- type=%q（%d 步）\n", k, len(offenders[k]))
			for _, line := range offenders[k] {
				fmt.Printf("   %s\n", line)
			}
		}
	}

	if len(unparsable) > 0 {
		fmt.Printf("\n=== script 无法解析（需人工核查）===\n")
		for _, u := range unparsable {
			fmt.Printf("   %s\n", u)
		}
	}

	// 结论行供 SUMMARY 引用（机器可读）。
	var aliasSet []string
	for k := range offenders {
		aliasSet = append(aliasSet, k)
	}
	sort.Strings(aliasSet)
	fmt.Printf("\nALIAS_SET=%s\n", strings.Join(aliasSet, ","))
}

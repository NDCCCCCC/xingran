// rpa_goto_task_dump：把库里 type="goto"/别名的任务脚本原样导出为 JSON（只读）。
//
// 用途（W7.1 缺陷 A 用户视角验收）：把生产库里 6 个 goto 任务的**原始脚本**导出，
// 供「不改库、靠 alias 也能跑」的真跑用例直接消费 —— 验收要求证明的是
// 「alias 修复后原样数据能跑」，而不是「把数据改掉再跑」。数据迁移是根治手段，
// 但迁移后的复跑不能替代 alias 的证明。
//
// 输出：out/rpa-goto-tasks.json（构建产物目录，gitignored）。
//
// 用法：
//
//	go run ./scripts/ops/rpa_goto_task_dump
//	go run ./scripts/ops/rpa_goto_task_dump -dsn "postgres://..."
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// exportTask 导出条目（保留原始 type 字符串，不归一化）。
type exportTask struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	CreatedAt time.Time                `json:"createdAt"`
	AliasType string                   `json:"aliasType"`
	Script    []map[string]interface{} `json:"script"`
}

func main() {
	dsn := flag.String("dsn", "", "直接给 DSN（覆盖 .env 拼接）")
	outPath := flag.String("out", filepath.Join("out", "rpa-goto-tasks.json"), "输出路径")
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

	type row struct {
		ID        string
		Name      string
		CreatedAt time.Time
		ScriptTxt string
	}
	var rows []row
	// 只取 script 里带 goto 的任务（迁移前）：迁移后这些行会变成 navigate，
	// 那时本工具的输出为空 —— 属预期（alias 证明改用导出的快照文件）。
	if err := gdb.Raw(`SELECT id::text AS id, name, created_at, script::text AS script_txt
		FROM sys_rpa_tasks WHERE script::text LIKE '%"goto"%' ORDER BY created_at, id`).Scan(&rows).Error; err != nil {
		fmt.Fprintln(os.Stderr, "query failed:", err)
		os.Exit(1)
	}

	var out []exportTask
	for _, r := range rows {
		var script []map[string]interface{}
		if err := json.Unmarshal([]byte(r.ScriptTxt), &script); err != nil {
			fmt.Fprintf(os.Stderr, "WARN 跳过无法解析的任务 %s: %v\n", r.ID, err)
			continue
		}
		alias := ""
		for _, a := range script {
			if t, ok := a["type"].(string); ok && t != "navigate" {
				// 第一个非 navigate 的 type 即别名（这些测试任务第一步都是 goto）
				if t == "goto" || strings.EqualFold(t, "goto") || t == "url" {
					alias = t
					break
				}
			}
		}
		out = append(out, exportTask{
			ID: r.ID, Name: r.Name, CreatedAt: r.CreatedAt,
			AliasType: alias, Script: script,
		})
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir failed:", err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal failed:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write failed:", err)
		os.Exit(1)
	}

	fmt.Printf("导出 %d 个任务 → %s\n", len(out), *outPath)
	for _, t := range out {
		steps := len(t.Script)
		var kinds []string
		for _, a := range t.Script {
			if ty, ok := a["type"].(string); ok {
				kinds = append(kinds, ty)
			}
		}
		fmt.Printf("  %s | %-22s | alias=%-5s | %d 步: %s\n",
			t.ID, t.Name, t.AliasType, steps, strings.Join(kinds, ","))
	}
}

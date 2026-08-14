package main

import (
	"flag"
	"fmt"
	"os"
)

// main.go — 临时菜单去重/导入工具
//
// 用法：
//   go run ./scripts/tmp_menuimport -mode=parse   # 解析计数断言（386/309/5）
//   go run ./scripts/tmp_menuimport -mode=gen     # 去重 + 生成 xingran_menus_dedup.sql + dedupe-report.md
//   go run ./scripts/tmp_menuimport -mode=import  # 导入 dev 库 + admin 授权 + 复核（需 .env 环境变量）
const inputFile = "xingran_menus_clean.sql"

func main() {
	mode := flag.String("mode", "parse", "parse|gen|import")
	flag.Parse()
	switch *mode {
	case "parse":
		runParse()
	case "gen":
		runGen()
	case "import":
		runImport()
	default:
		fail("unknown -mode: " + *mode)
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "FAIL: "+msg)
	os.Exit(1)
}

// loadAll 读取输入 SQL 并解析为三类行集合。
func loadAll() (menus []*MenuRow, roleMenus []*RoleMenuRow, roleCount int) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fail("read " + inputFile + ": " + err.Error())
	}
	parsed := parseInserts(string(data))
	menus = toMenuRows(parsed["sys_menu"])
	roleMenus = toRoleMenuRows(parsed["sys_role_menu"])
	roleCount = len(parsed["sys_role"])
	return menus, roleMenus, roleCount
}

// runImport 将在 Task 3 (import.go) 实现，当前为占位 stub。
func runImport() { fail("-mode=import not implemented yet (Task 3)") }

// runParse Task 1 验证：全量计数断言 + 抽样打印。
func runParse() {
	menus, roleMenus, roleCount := loadAll()
	fmt.Printf("sys_menu=%d sys_role_menu=%d sys_role=%d\n", len(menus), len(roleMenus), roleCount)
	if len(menus) != 386 {
		fail(fmt.Sprintf("sys_menu 计数断言失败: got %d want 386", len(menus)))
	}
	if len(roleMenus) != 309 {
		fail(fmt.Sprintf("sys_role_menu 计数断言失败: got %d want 309", len(roleMenus)))
	}
	if roleCount != 5 {
		fail(fmt.Sprintf("sys_role 计数断言失败: got %d want 5", roleCount))
	}
	// 抽样：前 3 个顶级 M 目录名
	shown := 0
	for _, m := range menus {
		if m.ParentID() == nil && m.MenuType() == "M" {
			fmt.Printf("  top-M sample: %s (id=%s deleted_at=%v)\n", m.Name(), m.ID()[:8], m.DeletedAt() != nil)
			if shown++; shown >= 3 {
				break
			}
		}
	}
	fmt.Println("PARSE PASS")
}

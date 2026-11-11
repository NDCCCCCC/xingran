// db-backup-restore.go —— SQLite 全库备份离线恢复工具（一次性运维 CLI）。
//
// 背景：在线服务的 GORM 连接池持有 SQLite 库文件句柄，进程内无法安全热替换
// 自身数据库（关闭连接池会令运行中的服务 DB 半死）。因此 SQLite 恢复必须停服后
// 用本工具执行；PostgreSQL 恢复走在线 pg_restore，无需本工具。
//
// 用法（必须先停服）：
//
//	go run ./scripts/ops/db-backup-restore.go -db <sqlite-db-path> -pending <db-path>.pending_restore
//
// 行为：校验 pending 文件是合法 SQLite 库 → 备份原库为 .pre_restore_bak →
// 原子 rename 替换 → 成功后再启动服务。任一失败都保留原库不动（可回滚）。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	// 注册纯 Go SQLite 驱动（"sqlite"），供 database/sql.Open 使用。
	_ "github.com/glebarez/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "目标 SQLite 库文件路径（必填）")
	pending := flag.String("pending", "", "待恢复产物路径（默认 <db>.pending_restore）")
	flag.Parse()

	if *dbPath == "" {
		fail("-db 不能为空")
	}
	p := *pending
	if p == "" {
		p = *dbPath + ".pending_restore"
	}

	absDB, err := filepath.Abs(*dbPath)
	if err != nil {
		fail("解析 -db 路径失败: %v", err)
	}
	absPending, err := filepath.Abs(p)
	if err != nil {
		fail("解析 -pending 路径失败: %v", err)
	}

	// 1. 校验待恢复产物
	if err := probe(absPending); err != nil {
		fail("待恢复产物校验失败（原库保持不动）: %v", err)
	}
	fmt.Printf("✓ 待恢复产物校验通过: %s\n", absPending)

	// 2. 备份原库
	bak := absDB + ".pre_restore_bak"
	if _, err := os.Stat(absDB); err == nil {
		_ = os.Remove(bak)
		if err := os.Rename(absDB, bak); err != nil {
			fail("备份原库失败（原库保持不动）: %v", err)
		}
		fmt.Printf("✓ 原库已备份为: %s\n", bak)
	} else if !os.IsNotExist(err) {
		fail("检查原库失败: %v", err)
	}

	// 3. 原子替换
	if err := os.Rename(absPending, absDB); err != nil {
		// 回滚：把原库改回来
		if _, statErr := os.Stat(bak); statErr == nil {
			_ = os.Rename(bak, absDB)
		}
		fail("原子替换失败（已回滚原库）: %v", err)
	}
	fmt.Printf("✓ 恢复完成: %s\n", absDB)
	fmt.Println("现在可以重新启动后端服务。若新库有问题，停服后用 .pre_restore_bak 回滚。")
}

// probe 校验文件是结构完整的 SQLite 库（魔数 + sqlite_master 可查询）。
func probe(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开文件: %w", err)
	}
	header := make([]byte, 16)
	if _, err := f.Read(header); err != nil {
		_ = f.Close()
		return fmt.Errorf("读取文件头: %w", err)
	}
	_ = f.Close()
	if string(header) != "SQLite format 3\x00" {
		return fmt.Errorf("文件头不是 SQLite format 3")
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("打开独立连接: %w", err)
	}
	defer sqlDB.Close()
	var n int
	if err := sqlDB.QueryRow("SELECT count(*) FROM sqlite_master").Scan(&n); err != nil {
		return fmt.Errorf("sqlite_master 不可查询（文件损坏/半截）: %w", err)
	}
	return nil
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}

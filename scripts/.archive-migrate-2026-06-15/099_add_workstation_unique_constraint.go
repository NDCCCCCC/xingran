//go:build ignore
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/xingran-next/xingran-go-backend/internal/config"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 构建数据库连接
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}

	fmt.Println("开始应用迁移: 099_add_workstation_unique_constraint")

	// 添加唯一约束（排除已删除的记录）
	sql := `
		CREATE UNIQUE INDEX IF NOT EXISTS sys_workstation_floor_name_idx
		ON sys_workstation (floor_id, workstation_name)
		WHERE deleted_at IS NULL;
	`

	if _, err := db.Exec(sql); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	fmt.Println("迁移完成!")
	fmt.Println("已创建唯一索引: sys_workstation_floor_name_idx ON sys_workstation (floor_id, workstation_name) WHERE deleted_at IS NULL")
}

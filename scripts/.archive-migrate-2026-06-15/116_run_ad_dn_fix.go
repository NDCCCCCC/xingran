//go:build ignore
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	// 初始化日志
	applogger.Init(&applogger.Config{
		Level:         "info",
		ConsoleOutput: true,
	})

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	database, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	gormDB := database.GetDB()

	// 读取 SQL 迁移文件
	migrationPath := filepath.Join("internal", "core", "db", "migrations", "116_fix_sys_user_ad_dn_column_name.sql")
	sqlContent, err := os.ReadFile(migrationPath)
	if err != nil {
		fmt.Printf("读取迁移文件失败: %v\n", err)
		os.Exit(1)
	}

	// 执行 SQL
	result := gormDB.Exec(string(sqlContent))
	if result.Error != nil {
		fmt.Printf("执行迁移失败: %v\n", result.Error)
		os.Exit(1)
	}

	fmt.Printf("✅ 迁移执行成功！影响行数: %d\n", result.RowsAffected)
	fmt.Println("sys_user.addn 列已重命名为 ad_dn")
}

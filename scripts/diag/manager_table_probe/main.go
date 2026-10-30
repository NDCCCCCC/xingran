// manager_table_probe 诊断: 检查经理同步两张表在目标库的存在性/列清单,
// 并试跑 ListRules 的关键 SQL, 用于定位 /system/manager-rules/list 500。
// 用法: go run ./scripts/diag/manager_table_probe
package main

import (
	"fmt"
	"context"

	"github.com/joho/godotenv"
	"os"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Println("加载配置失败:", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
	g, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	for _, table := range []string{"sys_manager_rule", "sys_manager_org", "sys_user_post", "user_posts"} {
		var reg string
		g.Raw("SELECT to_regclass(?)", table).Scan(&reg)
		fmt.Printf("table %-18s -> %s\n", table, reg)
	}

	for _, table := range []string{"sys_manager_rule", "sys_manager_org"} {
		var cols []string
		g.Raw(`SELECT column_name FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, table).Scan(&cols)
		fmt.Printf("columns of %s (%d): %v\n", table, len(cols), cols)
	}

	var cnt int64
	if err := g.Raw(`SELECT count(*) FROM sys_manager_rule`).Scan(&cnt).Error; err != nil {
		fmt.Println("count(sys_manager_rule) 失败:", err)
	} else {
		fmt.Println("count(sys_manager_rule) =", cnt)
	}
	if err := g.Raw(`SELECT count(*) FROM sys_dept`).Scan(&cnt).Error; err != nil {
		fmt.Println("count(sys_dept) 失败:", err)
	} else {
		fmt.Println("count(sys_dept) =", cnt)
	}
	// 直接调用真实服务层，复现 /system/manager-rules/list 的 500
	svc := addomainServices.NewManagerRuleService(g)
	page, err := svc.ListRules(context.Background(), 1, 10, "", nil)
	if err != nil {
		fmt.Printf("ListRules 失败: %v\n", err)
	} else {
		fmt.Printf("ListRules OK: total=%d list=%d\n", page.Total, len(page.List))
	}
	if _, err := svc.LoadResolver(context.Background()); err != nil {
		fmt.Printf("LoadResolver 失败: %v\n", err)
	} else {
		fmt.Println("LoadResolver OK")
	}
}

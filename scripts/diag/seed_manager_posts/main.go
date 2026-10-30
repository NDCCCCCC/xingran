// 一次性 seed：在 sys_post 创建经理同步规则所需的核心岗位（幂等）。
// 岗位是「通用经理同步方案」的配置基础：机构一把手 / 分管总 / 部门经理
// 都通过岗位任职（sys_user_post）判定，而非硬编码。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type seedPost struct {
	Code     string
	Name     string
	Sort     int
	Remark   string
}

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Println("config.Load 失败:", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
	g, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	posts := []seedPost{
		{"GM", "总经理", 1, "分公司一把手（机构一把手岗位，用于域控 manager 同步）"},
		{"VP", "分管副总", 2, "分管总（分公司部门经理的经理）"},
		{"BRANCH_GM", "中心支公司总经理", 3, "中支一把手（中支机构一把手岗位）"},
		{"DEPT_MGR", "部门经理", 4, "部门负责人岗位（一般员工的经理）"},
	}

	for _, p := range posts {
		res := g.Exec(
			`INSERT INTO sys_post (id, post_code, post_name, post_sort, status, remark, created_at, updated_at)
			 VALUES (gen_random_uuid(), ?, ?, ?, 0, ?, now(), now())
			 ON CONFLICT (post_code) DO NOTHING`,
			p.Code, p.Name, p.Sort, p.Remark)
		if res.Error != nil {
			fmt.Printf("✗ %-10s %s: %v\n", p.Code, p.Name, res.Error)
			continue
		}
		if res.RowsAffected == 1 {
			fmt.Printf("✓ 已创建岗位 %-10s %s\n", p.Code, p.Name)
		} else {
			fmt.Printf("= 已存在岗位 %-10s %s（跳过）\n", p.Code, p.Name)
		}
	}

	// 回显
	type Row struct {
		PostCode string
		PostName string
		Status   int
	}
	var rows []Row
	g.Table("sys_post").Select("post_code, post_name, status").
		Where("post_code IN ? AND deleted_at IS NULL", []string{"GM", "VP", "BRANCH_GM", "DEPT_MGR"}).
		Order("post_sort").Find(&rows)
	fmt.Println("\n=== sys_post 现状 ===")
	for _, r := range rows {
		st := "正常"
		if r.Status != 0 {
			st = "停用"
		}
		fmt.Printf("%-12s %-14s %s\n", r.PostCode, r.PostName, st)
	}
}

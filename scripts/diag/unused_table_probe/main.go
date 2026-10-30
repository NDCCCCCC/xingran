// unused_table_probe 诊断: 盘点目标库中的表与行数, 用于识别「有表无数据 / 有数据无模型 / 双份重复表」。
// 用法: go run ./scripts/diag/unused_table_probe
package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type tableRow struct {
	TableName string `gorm:"column:table_name"`
}

type countRow struct {
	TableName string `gorm:"column:table_name"`
	RowCount  int64  `gorm:"column:row_count"`
}

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

	// 1) 岗位关联双表对比
	fmt.Println("=== 岗位关联表对比 ===")
	for _, t := range []string{"sys_user_post", "user_posts"} {
		var reg string
		g.Raw("SELECT to_regclass(?)", t).Scan(&reg)
		if reg == "" {
			fmt.Printf("table %-14s -> 不存在\n", t)
			continue
		}
		var cnt int64
		if err := g.Raw("SELECT count(*) FROM " + t).Scan(&cnt).Error; err != nil {
			fmt.Printf("table %-14s -> 存在，count 失败: %v\n", t, err)
			continue
		}
		var cols []string
		g.Raw(`SELECT column_name FROM information_schema.columns WHERE table_name = ? ORDER BY ordinal_position`, t).Scan(&cols)
		fmt.Printf("table %-14s -> 存在, rows=%d, cols=%v\n", t, cnt, cols)
	}

	// 1b) 真实关联表 DDL（判断 many2many 改名后 AutoMigrate 是否会改主键/加约束）
	fmt.Println("\n=== 真实关联表 DDL ===")
	for _, t := range []string{"sys_user_role", "sys_user_post", "sys_role_menu", "sys_role_dept"} {
		var ddl string
		if err := g.Raw(`SELECT COALESCE(indexdef, '') FROM pg_indexes WHERE tablename = ?`, t).Scan(&ddl).Error; err != nil {
			continue
		}
		var cons []string
		g.Raw(`SELECT conname || ' => ' || pg_get_constraintdef(oid) FROM pg_constraint
			WHERE conrelid = ?::regclass ORDER BY conname`, t).Scan(&cons)
		fmt.Printf("table %-14s constraints=%v\n", t, cons)
	}

	// 1b) 待删表的 FK 反查（谁引用了它）
	fmt.Println("\n=== 待删表的 FK 反查 ===")
	for _, t := range []string{"sys_dept_group_mapping", "user_posts", "user_roles", "role_menus", "role_depts"} {
		var refs []string
		g.Raw(`SELECT c.conrelid::regclass::text || ' -> ' || c.confrelid::regclass::text
			FROM pg_constraint c WHERE c.contype='f' AND c.confrelid = ?::regclass`, t).Scan(&refs)
		fmt.Printf("table %-26s 被引用方=%v\n", t, refs)
	}

	// 2) 全表行数盘点（按行数升序，空表排在最前 → 空表最可能是废弃表）
	fmt.Println("\n=== 全部空表（public schema） ===")
	var tables []tableRow
	if err := g.Raw(`SELECT table_name FROM information_schema.tables
		WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name`).Scan(&tables).Error; err != nil {
		fmt.Println("列表失败:", err)
		os.Exit(1)
	}
	var counts []countRow
	for _, tb := range tables {
		var c int64
		if err := g.Raw("SELECT count(*) FROM \"" + tb.TableName + "\"").Scan(&c).Error; err != nil {
			c = -1
		}
		counts = append(counts, countRow{TableName: tb.TableName, RowCount: c})
	}
	sort.Slice(counts, func(i, j int) bool { return counts[i].RowCount < counts[j].RowCount })
	for i, c := range counts {
		if i >= 200 {
			break
		}
		if c.RowCount == 0 {
			fmt.Printf("%-42s rows=0  <== 空表\n", c.TableName)
		}
	}
	fmt.Printf("... 共 %d 张表\n", len(counts))
}

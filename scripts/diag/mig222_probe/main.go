// Command mig222_probe 在目标库的**临时 schema** 内验证 migration_222 的真实行为，
// 验证完即删除该 schema，全程不触碰 public schema 的任何表。
//
// 用法：go run ./scripts/diag/mig222_probe
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const scratchSchema = "mig222_scratch"

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Println("配置加载失败:", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
	g, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	// 1) 建临时 schema
	if err := g.Exec("DROP SCHEMA IF EXISTS " + scratchSchema + " CASCADE").Error; err != nil {
		fmt.Println("清理旧 schema 失败:", err)
		os.Exit(1)
	}
	if err := g.Exec("CREATE SCHEMA " + scratchSchema).Error; err != nil {
		fmt.Println("建 schema 失败:", err)
		os.Exit(1)
	}
	defer func() {
		g.Exec("DROP SCHEMA IF EXISTS " + scratchSchema + " CASCADE")
		fmt.Println("已删除临时 schema", scratchSchema)
	}()

	// 2) 在临时 schema 内造出 4 张幻影表 + 1 张遗留表（必须固定连接 + search_path，
	//    否则连接池默认走 public，会撞上真实的幻影表）
	createSQL := []string{
		`CREATE TABLE user_posts (user_id uuid, post_id uuid)`,
		`CREATE TABLE user_roles (user_id uuid, role_id uuid)`,
		`CREATE TABLE role_menus (role_id uuid, menu_id uuid)`,
		`CREATE TABLE role_depts (role_id uuid, dept_id uuid)`,
		`CREATE TABLE sys_dept_group_mapping (dept_id uuid, group_dn text)`,
		// 有数据的表：必须被跳过（保守策略验证）
		`CREATE TABLE user_posts_withdata (user_id uuid)`,
	}
	if err := g.Connection(func(tx *gorm.DB) error {
		if err := tx.Exec("SET search_path TO " + scratchSchema + ", public").Error; err != nil {
			return err
		}
		for _, s := range createSQL {
			if err := tx.Exec(s).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		fmt.Println("建表失败:", err)
		os.Exit(1)
	}

	// 3) 固定单连接 + search_path，在临时 schema 内跑真实迁移
	runErr := g.Connection(func(tx *gorm.DB) error {
		if err := tx.Exec("SET search_path TO " + scratchSchema + ", public").Error; err != nil {
			return err
		}
		// 先塞一行数据到某张"待删表"变体，验证保守跳过
		if err := tx.Exec(`INSERT INTO user_posts_withdata (user_id) VALUES ('00000000-0000-0000-0000-000000000001')`).Error; err != nil {
			return err
		}
		return migrations.Migrate222DropPhantomJoinTables(tx)
	})
	if runErr != nil {
		fmt.Println("迁移执行返回错误(预期为 nil):", runErr)
	}

	// 4) 校验：幻影表应已消失
	fmt.Println("\n=== 校验（临时 schema 内） ===")
	for _, t := range []string{"user_posts", "user_roles", "role_menus", "role_depts", "sys_dept_group_mapping"} {
		var reg string
		g.Raw("SELECT COALESCE(to_regclass(?)::text, '')", scratchSchema+"."+t).Scan(&reg)
		status := "已删除"
		if reg != "" {
			status = "仍存在 <== 未删干净"
		}
		fmt.Printf("%-26s %s\n", t, status)
	}

	// 5) 幂等：再跑一次应无副作用
	fmt.Println("\n=== 幂等复跑 ===")
	runErr2 := g.Connection(func(tx *gorm.DB) error {
		if err := tx.Exec("SET search_path TO " + scratchSchema + ", public").Error; err != nil {
			return err
		}
		return migrations.Migrate222DropPhantomJoinTables(tx)
	})
	fmt.Println("复跑返回:", runErr2)
}

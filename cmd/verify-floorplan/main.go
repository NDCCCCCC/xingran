// 一次性诊断工具：查楼层平面图底图的数据链路（ops_floors → sys_files → plan_image_url）。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

func main() {
	_ = godotenv.Load(".env", "configs/.env")
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Printf("!! 配置加载失败: %v\n", err)
		os.Exit(1)
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("!! 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	// sys_files 表是否存在
	var filesTableExists bool
	db.Raw(`SELECT to_regclass('public.sys_files') IS NOT NULL`).Scan(&filesTableExists)
	fmt.Printf("sys_files 表存在: %v\n\n", filesTableExists)

	type row struct {
		ID           string
		Name         string
		FloorNo      string
		PlanImageID  *string
		PlanImageURL *string
	}
	var rows []row
	if err := db.Raw(`
		SELECT f.id, f.name, f.floor_no, f.plan_image_id,
		       '/uploads/' || sf.storage_path AS plan_image_url
		FROM ops_floors f
		LEFT JOIN sys_files sf ON sf.id = f.plan_image_id
		WHERE f.deleted_at IS NULL
		ORDER BY f.created_at DESC
		LIMIT 15`).Scan(&rows).Error; err != nil {
		fmt.Printf("!! 查询失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== 最近 15 个楼层 ===\n")
	withImg := 0
	for _, r := range rows {
		pid, purl := "-", "-"
		if r.PlanImageID != nil && *r.PlanImageID != "" {
			pid = *r.PlanImageID
		}
		if r.PlanImageURL != nil && *r.PlanImageURL != "" && *r.PlanImageURL != "/uploads/" {
			purl = *r.PlanImageURL
			withImg++
		}
		fmt.Printf("  楼层=%-12s %-4s planImageId=%-38s url=%s\n", trunc(r.Name, 12), r.FloorNo, pid, purl)
	}
	fmt.Printf("\n带底图的楼层数: %d / %d\n", withImg, len(rows))

	if filesTableExists {
		var total int64
		db.Raw(`SELECT count(*) FROM sys_files`).Scan(&total)
		fmt.Printf("\nsys_files 总数: %d\n", total)

		var orphans []struct {
			ID          string
			FileName    string
			StoragePath string
			Category    string
			RefCount    int
		}
		db.Raw(`
			SELECT sf.id, sf.file_name, sf.storage_path, 
			       (SELECT count(*) FROM ops_floors f2 WHERE f2.plan_image_id = sf.id AND f2.deleted_at IS NULL) AS ref_count
			FROM sys_files sf
			ORDER BY sf.created_at DESC
			LIMIT 10`).Scan(&orphans)
		fmt.Println("=== 最近 10 个上传文件 ===")
		for _, o := range orphans {
			fmt.Printf("  %-38s path=%-40s 被楼层引用=%d\n",
				o.ID, trunc(o.StoragePath, 40),  o.RefCount)
		}
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

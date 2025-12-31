package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type CaptchaBackground struct {
	ID              string
	FileName        string
	PieceShape      string
	DifficultyLevel int
	Status          int
	DeletedAt       *string
}

func main() {
	// 连接数据库
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=ruoyi port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 更新难度级别为 1
	result := db.Exec("UPDATE sys_captcha_background SET difficulty_level = 1 WHERE id = '1767152337763405900'")
	if result.Error != nil {
		log.Fatal("更新失败:", result.Error)
	}
	fmt.Printf("✓ 已更新 %d 条记录的难度级别为 1\n", result.RowsAffected)

	// 查询验证
	var count int64
	db.Table("sys_captcha_background").
		Where("status = ? AND piece_shape = ? AND difficulty_level = ? AND deleted_at IS NULL", 1, "circle", 1).
		Count(&count)

	fmt.Printf("\n现在有 %d 条符合条件的背景图 (status=1, shape=circle, difficulty=1)\n", count)

	if count > 0 {
		fmt.Println("\n✅ 修复成功！请刷新登录页面测试。")
	}
}

type CaptchaBackground struct {
	ID              string
	FileName        string
	PieceShape      string
	DifficultyLevel int
	Status          int
	DeletedAt       *string
}

func main() {
	// 连接数据库
	dsn := "host=10.62.10.34 user=postgres password=Cpic1234 dbname=ruoyi port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 查询背景图数据
	var backgrounds []CaptchaBackground
	result := db.Table("sys_captcha_background").
		Select("id, file_name, piece_shape, difficulty_level, status, deleted_at::text").
		Order("created_at DESC").
		Limit(10).
		Scan(&backgrounds)

	if result.Error != nil {
		log.Fatal("查询失败:", result.Error)
	}

	fmt.Println("\n=== 验证码背景图数据 ===")
	fmt.Printf("%-36s | %-20s | %-10s | %-10s | %-6s | %-10s\n", "ID", "文件名", "形状", "难度", "状态", "已删除")
	fmt.Println("------------------------------------------------------------------------------------------------------")

	for _, bg := range backgrounds {
		deleted := "否"
		if bg.DeletedAt != nil {
			deleted = "是"
		}
		status := "禁用"
		if bg.Status == 1 {
			status = "启用"
		}
		fmt.Printf("%-36s | %-20s | %-10s | %-10d | %-6s | %-10s\n",
			bg.ID, bg.FileName, bg.PieceShape, bg.DifficultyLevel, status, deleted)
	}

	// 检查是否有符合条件的数据
	fmt.Println("\n=== 查询条件 ===")
	fmt.Println("需要的条件: status = 1 AND piece_shape = 'circle' AND difficulty_level = 1")

	var count int64
	db.Table("sys_captcha_background").
		Where("status = ? AND piece_shape = ? AND difficulty_level = ? AND deleted_at IS NULL", 1, "circle", 1).
		Count(&count)

	fmt.Printf("符合条件的记录数: %d\n", count)
	if count == 0 {
		fmt.Println("\n⚠️ 没有符合条件的背景图！")
		fmt.Println("请检查:")
		fmt.Println("1. 状态是否为 '启用'")
		fmt.Println("2. 拼图形状是否为 'circle'")
		fmt.Println("3. 难度级别是否为 1")
	}
}

package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

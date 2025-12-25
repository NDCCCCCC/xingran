package main

import (
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 连接数据库
	postgres, err := db.NewPostgres(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer postgres.Close()

	// 自动迁移所有模型
	err = postgres.DB.AutoMigrate(
		// 基础模型
		&models.User{},
		&models.Role{},
		&models.Menu{},
		&models.Department{},
		&models.Post{},

		// 配置模型
		&models.Config{},
		&models.DictType{},
		&models.DictData{},

		// 关联模型
		&models.UserRole{},
		&models.RoleMenu{},
		&models.RoleDept{},
		&models.UserPost{},

		// 日志模型
		&models.OperLog{},
		&models.LoginLog{},

		// 任务模型
		&models.Job{},
		&models.JobLog{},

		// 通知模型
		&models.Notice{},

		// 工位模型
		&models.Workstation{},

		// 监控模型
		&models.ServerInfo{},
		&models.SystemMetrics{},
		&models.CacheInfo{},
		&models.CacheStats{},

		// 代码生成模型
		&models.GenTable{},
		&models.GenColumn{},
	)

	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("数据库迁移成功")
}
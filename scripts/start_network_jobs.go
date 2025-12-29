package main

import (
	"fmt"
	"log"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	database, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()

	db := database.GetDB()

	// 查询所有网络设备相关的定时任务
	var jobs []models.Job
	if err := db.Where("job_group = ?", "NETWORK").Find(&jobs).Error; err != nil {
		log.Fatalf("查询定时任务失败: %v", err)
	}

	fmt.Printf("\n找到 %d 个网络设备定时任务:\n", len(jobs))
	fmt.Println("================================================================================")

	for _, job := range jobs {
		statusText := "暂停"
		if job.Status == 0 {
			statusText = "正常"
		}
		fmt.Printf("\n任务名称: %s\n", job.JobName)
		fmt.Printf("  ID: %s\n", job.ID)
		fmt.Printf("  调用目标: %s\n", job.InvokeTarget)
		fmt.Printf("  Cron表达式: %s\n", job.CronExpression)
		fmt.Printf("  当前状态: %s (%d)\n", statusText, job.Status)
		fmt.Printf("  备注: %s\n", func() string {
			if job.Remark != nil {
				return *job.Remark
			}
			return ""
		}())
	}

	fmt.Println("\n================================================================================")

	// 启动所有暂停的任务
	count := 0
	for _, job := range jobs {
		if job.Status == 1 { // 暂停状态
			if err := db.Model(&job).Update("status", 0).Error; err != nil {
				log.Printf("启动任务 %s 失败: %v", job.JobName, err)
			} else {
				log.Printf("✓ 启动任务: %s", job.JobName)
				count++
			}
		}
	}

	fmt.Printf("\n成功启动 %d 个定时任务\n", count)
}

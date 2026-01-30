package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	"gorm.io/gorm"
)

var (
	adSyncCron    *cron.Cron
	adSyncStarted bool
)

// ==================== AD同步任务注册函数 ====================

// RegisterADSyncTasks 注册AD域同步定时任务
func RegisterADSyncTasks(scheduler *Scheduler) {
	// AD数据自动同步任务 - 每小时检查一次需要同步的配置
	scheduler.RegisterTask("ad_data_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeADDataSyncTask(ctx, params)
	})
}

// StartADSyncScheduler 启动AD同步调度器
func StartADSyncScheduler(db *gorm.DB) {
	if adSyncStarted {
		return
	}

	adSyncCron = cron.New(cron.WithSeconds())

	// 每5分钟检查一次需要同步的AD配置
	_, err := adSyncCron.AddFunc("0 */5 * * * *", func() {
		checkAndSyncADConfigs(db)
	})
	if err != nil {
		log.Printf("启动AD同步定时任务失败: %v", err)
		return
	}

	adSyncCron.Start()
	adSyncStarted = true
	log.Println("AD同步定时任务已启动")
}

// StopADSyncScheduler 停止AD同步调度器
func StopADSyncScheduler() {
	if adSyncCron != nil {
		adSyncCron.Stop()
		adSyncStarted = false
		log.Println("AD同步定时任务已停止")
	}
}

// checkAndSyncADConfigs 检查并同步需要同步的AD配置
func checkAndSyncADConfigs(db *gorm.DB) {
	var configs []models.ADConfig
	err := db.Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).Find(&configs).Error
	if err != nil {
		log.Printf("查询AD配置失败: %v", err)
		return
	}

	for _, config := range configs {
		shouldSync := false
		var elapsed float64

		// 从未同步过
		if config.LastSyncAt == nil {
			shouldSync = true
			log.Printf("AD配置 %s 从未同步过，准备进行首次同步 (配置的同步间隔: %d秒)", config.ConfigName, config.SyncInterval)
		} else {
			// 检查是否超过同步间隔
			elapsed = time.Since(*config.LastSyncAt).Seconds()
			log.Printf("AD配置 %s 检查同步: 距离上次同步 %.0f 秒，配置的同步间隔 %d 秒", config.ConfigName, elapsed, config.SyncInterval)

			// 如果已经超过同步间隔，或者在下次检查之前会超过间隔，则触发同步
			// 下次检查时间是 5 分钟后（300秒）
			nextCheckIn := 300.0
			timeUntilNextSync := float64(config.SyncInterval) - elapsed

			if elapsed >= float64(config.SyncInterval) {
				shouldSync = true
				log.Printf("AD配置 %s 距离上次同步已超过 %.0f 秒 (配置间隔: %d秒)，准备同步", config.ConfigName, elapsed, config.SyncInterval)
			} else if timeUntilNextSync <= nextCheckIn && timeUntilNextSync > 0 {
				// 如果在下次检查之前就会达到同步间隔，提前触发同步，避免多等待一个检查周期
				shouldSync = true
				log.Printf("AD配置 %s 将在下次检查前达到同步间隔(剩余 %.0f 秒)，提前触发同步", config.ConfigName, timeUntilNextSync)
			}
		}

		if shouldSync {
			// 异步执行同步，避免阻塞其他配置的检查
			go syncADConfig(config.ID)
		}
	}
}

// syncADConfig 同步指定的AD配置
func syncADConfig(configID string) {
	// 从全局数据库获取DB实例
	if GlobalDB == nil {
		log.Printf("AD同步失败: 数据库未初始化")
		return
	}

	db := GlobalDB.GetDB()
	ctx := context.Background()

	// 获取配置
	var config models.ADConfig
	err := db.Where("id = ?", configID).First(&config).Error
	if err != nil {
		log.Printf("AD同步失败: 查询配置失败 - %v", err)
		return
	}

	log.Printf("开始同步AD配置: %s", config.ConfigName)

	// 创建Service并执行同步（SyncData内部会创建和管理同步日志）
	adService := addomain.NewADDomainService(db)
	result, err := adService.Sync.SyncDataByID(ctx, configID, "full")

	if err != nil {
		log.Printf("AD同步失败: %s - %v", config.ConfigName, err)
	} else {
		log.Printf("AD同步成功: %s - OU=%d, 用户组=%d, 用户=%d, 电脑=%d",
			config.ConfigName, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount)
	}
}

// executeADDataSyncTask 执行AD数据同步任务（供Scheduler调用）
func executeADDataSyncTask(ctx context.Context, params map[string]interface{}) error {
	configID, ok := params["configId"].(string)
	if !ok || configID == "" {
		return fmt.Errorf("配置ID参数无效")
	}

	syncType := "full"
	if syncTypeParam, ok := params["syncType"].(string); ok {
		syncType = syncTypeParam
	}

	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()
	adService := addomain.NewADDomainService(db)

	_, err := adService.Sync.SyncDataByID(ctx, configID, syncType)
	return err
}

// ==================== 辅助函数 ====================

// ScheduleADSyncForConfig 为指定配置安排一次性同步
func ScheduleADSyncForConfig(configID string, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		syncADConfig(configID)
	}()
}

// GetADSyncStatus 获取AD同步状态
func GetADSyncStatus() map[string]interface{} {
	status := map[string]interface{}{
		"started": adSyncStarted,
	}

	if adSyncCron != nil {
		status["next_run"] = getNextRunTime(adSyncCron)
	}

	return status
}

// getNextRunTime 获取下次运行时间
func getNextRunTime(c *cron.Cron) string {
	entries := c.Entries()
	if len(entries) > 0 {
		nextRun := entries[0].Next
		if !nextRun.IsZero() {
			return nextRun.Format("2006-01-02 15:04:05")
		}
	}
	return "未设置"
}

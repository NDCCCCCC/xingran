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
	adSyncCron  *cron.Cron
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

		// 从未同步过
		if config.LastSyncAt == nil {
			shouldSync = true
			log.Printf("AD配置 %s 从未同步过，准备进行首次同步", config.ConfigName)
		} else {
			// 检查是否超过同步间隔
			elapsed := time.Since(*config.LastSyncAt).Seconds()
			if elapsed >= float64(config.SyncInterval) {
				shouldSync = true
				log.Printf("AD配置 %s 距离上次同步已超过 %d 秒，准备同步", config.ConfigName, config.SyncInterval)
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

	// 创建同步日志
	syncLog := &models.ADSyncLog{
		ADConfigID: config.ID,
		SyncType:   models.ADSyncTypeFull,
		SyncStatus: models.ADSyncStatusRunning,
		StartTime:  time.Now(),
	}
	if err := db.Create(syncLog).Error; err != nil {
		log.Printf("AD同步失败: 创建同步日志失败 - %v", err)
		return
	}

	// 创建Service并执行同步
	adService := addomain.NewADDomainService(db)
	result, err := adService.Sync.SyncDataByID(ctx, configID, "full")

	// 更新同步日志
	endTime := time.Now()
	duration := int(endTime.Sub(syncLog.StartTime).Seconds())

	updates := map[string]interface{}{
		"end_time":  endTime,
		"duration":  duration,
		"sync_status": models.ADSyncStatusSuccess,
		"ou_count":   0,
		"group_count": 0,
		"user_count":  0,
		"error_count": 0,
	}

	if err != nil {
		log.Printf("AD同步失败: %s - %v", config.ConfigName, err)
		updates["sync_status"] = models.ADSyncStatusFailed
		updates["error_message"] = err.Error()
		updates["error_count"] = 1
	} else {
		log.Printf("AD同步成功: %s - OU=%d, 用户组=%d, 用户=%d",
			config.ConfigName, result.OUCount, result.GroupCount, result.UserCount)
		updates["ou_count"] = result.OUCount
		updates["group_count"] = result.GroupCount
		updates["user_count"] = result.UserCount
	}

	db.Model(syncLog).Updates(updates)
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

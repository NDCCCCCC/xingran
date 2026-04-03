package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// ADSyncScheduler AD同步调度器
// 遵循 Go 最佳实践：将全局变量封装为结构体，便于测试和并发控制
type ADSyncScheduler struct {
	cron        *cron.Cron
	started     bool
	sem         *semaphore.Weighted // 并发控制信号量
	db          *gorm.DB
	mu          sync.Mutex         // 保护状态变更
}

// 全局 AD 同步调度器实例
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
var (
	globalADSyncScheduler   *ADSyncScheduler
	globalADSyncSchedulerOnce sync.Once
)

// NewADSyncScheduler 创建AD同步调度器
func NewADSyncScheduler(db *gorm.DB, maxConcurrent int64) *ADSyncScheduler {
	return &ADSyncScheduler{
		cron: cron.New(cron.WithSeconds()),
		sem:  semaphore.NewWeighted(maxConcurrent),
		db:   db,
	}
}

// ==================== AD同步任务注册函数 ====================

// RegisterADSyncTasks 注册AD域同步定时任务
func RegisterADSyncTasks(scheduler *Scheduler) {
	// AD数据自动同步任务 - 每小时检查一次需要同步的配置
	scheduler.RegisterTask("ad_data_sync", func(ctx context.Context, params map[string]interface{}) error {
		return executeADDataSyncTask(ctx, params)
	})
}

// StartADSyncScheduler 启动全局AD同步调度器
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
func StartADSyncScheduler(db *gorm.DB) {
	globalADSyncSchedulerOnce.Do(func() {
		globalADSyncScheduler = NewADSyncScheduler(db, constants.MaxConcurrentADSync)
		globalADSyncScheduler.Start()
	})
}

// StopADSyncScheduler 停止全局AD同步调度器
func StopADSyncScheduler() {
	if globalADSyncScheduler != nil {
		globalADSyncScheduler.Stop()
	}
}

// GetADSyncScheduler 获取全局AD同步调度器
func GetADSyncScheduler() *ADSyncScheduler {
	return globalADSyncScheduler
}

// Start 启动AD同步调度器
func (s *ADSyncScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}

	// 每5分钟检查一次需要同步的AD配置
	_, err := s.cron.AddFunc("0 */5 * * * *", func() {
		s.checkAndSyncADConfigs()
	})
	if err != nil {
		applogger.Errorf("启动AD同步定时任务失败: %v", err)
		return
	}

	s.cron.Start()
	s.started = true
	applogger.Infof("AD同步定时任务已启动")
}

// Stop 停止AD同步调度器
func (s *ADSyncScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
		s.started = false
		applogger.Infof("AD同步定时任务已停止")
	}
}

// checkAndSyncADConfigs 检查并同步需要同步的AD配置
func (s *ADSyncScheduler) checkAndSyncADConfigs() {
	var configs []models.ADConfig
	err := s.db.Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).Find(&configs).Error
	if err != nil {
		applogger.Errorf("查询AD配置失败: %v", err)
		return
	}

	for _, config := range configs {
		shouldSync := false
		var elapsed float64

		// 从未同步过
		if config.LastSyncAt == nil {
			shouldSync = true
			applogger.Infof("AD配置 %s 从未同步过，准备进行首次同步 (配置的同步间隔: %d秒)", config.ConfigName, config.SyncInterval)
		} else {
			// 检查是否超过同步间隔
			elapsed = time.Since(*config.LastSyncAt).Seconds()
			applogger.Debugf("AD配置 %s 检查同步: 距离上次同步 %.0f 秒，配置的同步间隔 %d 秒", config.ConfigName, elapsed, config.SyncInterval)

			// 如果已经超过同步间隔，或者在下次检查之前会超过间隔，则触发同步
			// 下次检查时间是 5 分钟后（300秒）
			nextCheckIn := 300.0
			timeUntilNextSync := float64(config.SyncInterval) - elapsed

			if elapsed >= float64(config.SyncInterval) {
				shouldSync = true
				applogger.Infof("AD配置 %s 距离上次同步已超过 %.0f 秒 (配置间隔: %d秒)，准备同步", config.ConfigName, elapsed, config.SyncInterval)
			} else if timeUntilNextSync <= nextCheckIn && timeUntilNextSync > 0 {
				// 如果在下次检查之前就会达到同步间隔，提前触发同步，避免多等待一个检查周期
				shouldSync = true
				applogger.Infof("AD配置 %s 将在下次检查前达到同步间隔(剩余 %.0f 秒)，提前触发同步", config.ConfigName, timeUntilNextSync)
			}
		}

		if shouldSync {
			// 使用信号量控制并发数，异步执行同步
			// 遵循 Go 最佳实践：使用带超时的 context
			go func(configID string, configName string) {
				syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()

				// 尝试获取信号量
				if err := s.sem.Acquire(syncCtx, 1); err != nil {
					if syncCtx.Err() == context.DeadlineExceeded {
						applogger.Warnf("AD同步启动超时: %s", configName)
					} else {
						applogger.Errorf("AD同步启动失败: %s - %v", configName, err)
					}
					return
				}
				defer s.sem.Release(1)

				s.syncADConfig(syncCtx, configID)
			}(config.ID, config.ConfigName)
		}
	}
}

// syncADConfig 同步指定的AD配置
func (s *ADSyncScheduler) syncADConfig(ctx context.Context, configID string) {
	// 获取配置
	var config models.ADConfig
	err := s.db.WithContext(ctx).Where("id = ?", configID).First(&config).Error
	if err != nil {
		applogger.Errorf("AD同步失败: 查询配置失败 - %v", err)
		return
	}

	applogger.Infof("开始同步AD配置: %s", config.ConfigName)

	// 创建Service并执行同步（SyncData内部会创建和管理同步日志）
	adService := addomain.NewADDomainService(s.db)
	result, err := adService.Sync.SyncDataByID(ctx, configID, "full")

	if err != nil {
		applogger.Errorf("AD同步失败: %s - %v", config.ConfigName, err)
	} else {
		applogger.Infof("AD同步成功: %s - OU=%d, 用户组=%d, 用户=%d, 电脑=%d",
			config.ConfigName, result.OUCount, result.GroupCount, result.UserCount, result.ComputerCount)
	}
}

// executeADDataSyncTask 执行AD数据同步任务（供Scheduler调用）
func executeADDataSyncTask(ctx context.Context, params map[string]interface{}) error {
	if globalADSyncScheduler == nil {
		return fmt.Errorf("AD同步调度器未初始化")
	}

	configID, ok := params["configId"].(string)
	if !ok || configID == "" {
		return fmt.Errorf("配置ID参数无效")
	}

	syncType := "full"
	if syncTypeParam, ok := params["syncType"].(string); ok {
		syncType = syncTypeParam
	}

	adService := addomain.NewADDomainService(globalADSyncScheduler.db)

	_, err := adService.Sync.SyncDataByID(ctx, configID, syncType)
	return err
}

// ==================== 辅助函数 ====================

// ScheduleADSyncForConfig 为指定配置安排一次性同步
func ScheduleADSyncForConfig(configID string, delay time.Duration) {
	if globalADSyncScheduler == nil {
		applogger.Errorf("AD同步调度器未初始化")
		return
	}

	go func() {
		time.Sleep(delay)
		syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		globalADSyncScheduler.syncADConfig(syncCtx, configID)
	}()
}

// GetADSyncStatus 获取AD同步状态
func GetADSyncStatus() map[string]interface{} {
	status := map[string]interface{}{
		"started": false,
	}

	if globalADSyncScheduler != nil {
		status["started"] = globalADSyncScheduler.IsStarted()
		if globalADSyncScheduler.cron != nil {
			status["next_run"] = getNextRunTime(globalADSyncScheduler.cron)
		}
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

// IsStarted 返回调度器是否已启动
func (s *ADSyncScheduler) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

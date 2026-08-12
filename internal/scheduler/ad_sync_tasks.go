package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
)

// ADSyncScheduler AD同步调度器
// 遵循 Go 最佳实践：将全局变量封装为结构体，便于测试和并发控制
type ADSyncScheduler struct {
	cron    *cron.Cron
	started bool
	sem     *semaphore.Weighted // 并发控制信号量
	db      *gorm.DB
	// Phase 38 Wave 1 (W-04): 全局账号池单例字段，供 dept_sync_tasks 等复用（Pitfall 4 缓存共享）
	pool    addomain.AccountPool
	mu      sync.Mutex // 保护状态变更
	ctx     context.Context    // 调度器上下文，用于取消延迟任务
	cancel  context.CancelFunc // 取消函数
}

// 全局 AD 同步调度器实例
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
var (
	globalADSyncScheduler     *ADSyncScheduler
	globalADSyncSchedulerOnce sync.Once
	globalADSM4Cipher        addomain.PasswordCipher // 全局AD域SM4加密器
	globalADSM4CipherMu      sync.RWMutex
)

// adSchedulerSyncTimeout AD调度器同步任务超时
const adSchedulerSyncTimeout = 30 * time.Minute

// SetADSM4Cipher 设置全局AD域SM4加密器（线程安全）
func SetADSM4Cipher(cipher addomain.PasswordCipher) {
	globalADSM4CipherMu.Lock()
	defer globalADSM4CipherMu.Unlock()
	globalADSM4Cipher = cipher
	// 同时设置addomain包中的全局加密器
	addomain.SetADSM4Cipher(cipher)
}

// getADSM4Cipher 获取全局AD域SM4加密器（线程安全）
func getADSM4Cipher() addomain.PasswordCipher {
	globalADSM4CipherMu.RLock()
	defer globalADSM4CipherMu.RUnlock()
	return globalADSM4Cipher
}

// NewADSyncScheduler 创建AD同步调度器
func NewADSyncScheduler(db *gorm.DB, maxConcurrent int64) *ADSyncScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &ADSyncScheduler{
		cron:   cron.New(cron.WithSeconds()),
		sem:    semaphore.NewWeighted(maxConcurrent),
		db:     db,
		ctx:    ctx,
		cancel: cancel,
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

// RegisterADAccountPoolTasks Phase 36: 注册 AD 账号池维护定时任务
//
// 任务：
//   1. recover_breakers - 每 5 分钟恢复已过期熔断的账号（Issue 4/11/17）
func RegisterADAccountPoolTasks(scheduler *Scheduler) {
	scheduler.RegisterTask("ad_account_pool_recover_breakers", func(ctx context.Context, params map[string]interface{}) error {
		return executeADAccountPoolRecoverBreakersTask(ctx, params)
	})
}

// executeADAccountPoolRecoverBreakersTask 执行恢复过期熔断账号任务
//
// 流程：
//   1. 调 AccountPool.RecoverExpiredBreakers（事务 + 写 DB）
//   2. 内部自动 InvalidateCache 本地缓存
//   3. 内部自动通过 Redis pub/sub 广播（如果启用）
//   4. 记录日志
//
// 频率：建议每 5 分钟（与 BreakerRecoveryCron 常量一致）
func executeADAccountPoolRecoverBreakersTask(ctx context.Context, params map[string]interface{}) error {
	if globalADSyncScheduler == nil {
		return fmt.Errorf("AD同步调度器未初始化")
	}

	// 创建账号池实例（与 core.Core 共享 DB）
	pool := addomain.NewAccountPool(globalADSyncScheduler.db, nil)
	recovered, err := pool.RecoverExpiredBreakers(ctx)
	if err != nil {
		applogger.Errorf("[ADAccountPool] cron 恢复过期熔断账号失败: %v", err)
		return err
	}
	if recovered > 0 {
		applogger.Infof("[ADAccountPool] cron 恢复了 %d 个过期熔断账号", recovered)
	}
	return nil
}

// StartADSyncScheduler 启动全局AD同步调度器
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
func StartADSyncScheduler(db *gorm.DB) {
	globalADSyncSchedulerOnce.Do(func() {
		globalADSyncScheduler = NewADSyncScheduler(db, constants.MaxConcurrentADSync)
		// Phase 38 Wave 1 (W-04): 创建全局账号池单例供 scheduler 内各 task 与 dept_sync_tasks 复用
		// （Pitfall 4：避免 per-task NewAccountPool 创建独立缓存，导致熔断后账号仍被选中）
		globalADSyncScheduler.pool = addomain.NewAccountPool(db, nil)
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

	// 每5分钟检查一次需要同步的AD配置（全量数据同步）
	_, err := s.cron.AddFunc("0 */5 * * * *", func() {
		s.checkAndSyncADConfigs()
	})
	if err != nil {
		applogger.Errorf("启动AD同步定时任务失败: %v", err)
		return
	}

	// Phase 36: 每 5 分钟恢复已过期熔断的 AD 账号（Issue 4/11）
	// Cron 表达式: 6 字段（含秒）"秒 分 时 日 月 周" → "0 */5 * * * *" = 每 5 分钟整点
	_, err = s.cron.AddFunc("0 */5 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		if err := executeADAccountPoolRecoverBreakersTask(ctx, nil); err != nil {
			applogger.Errorf("[ADAccountPool] cron 任务执行失败: %v", err)
		}
	})
	if err != nil {
		applogger.Errorf("启动AD账号池恢复定时任务失败: %v", err)
	}

	s.cron.Start()
	s.started = true
	applogger.Infof("AD同步定时任务已启动（全量同步: 每5分钟 + 账号池恢复: 每5分钟）")
}

// Stop 停止AD同步调度器
func (s *ADSyncScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
	}
	if s.cancel != nil {
		s.cancel() // 取消所有延迟任务的上下文
	}
	s.started = false
	applogger.Infof("AD同步定时任务已停止")
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
				syncCtx, cancel := context.WithTimeout(context.Background(), adSchedulerSyncTimeout)
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
	// Phase 38 Wave 1 (W-04): 复用全局 pool 单例（避免 per-task NewAccountPool，Pitfall 4）
	pool := s.getPool()
	adService := addomain.NewADDomainService(s.db, pool, getADSM4Cipher())
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

	// Phase 38 Wave 1 (W-04): 复用全局 pool 单例（避免 per-task NewAccountPool，Pitfall 4）
	pool := globalADSyncScheduler.getPool()
	adService := addomain.NewADDomainService(globalADSyncScheduler.db, pool, getADSM4Cipher())

	_, err := adService.Sync.SyncDataByID(ctx, configID, syncType)
	return err
}

// getDefaultADConfigID 获取默认的 AD 配置 ID
// 优先从参数中获取，如果没有则查询第一个启用的配置
func getDefaultADConfigID(ctx context.Context, db *gorm.DB, params map[string]interface{}) (string, error) {
	// 1. 尝试从参数中获取配置ID（支持两种参数名）
	if configID, ok := params["configId"].(string); ok && configID != "" {
		return configID, nil
	}
	if configID, ok := params["adConfigId"].(string); ok && configID != "" {
		return configID, nil
	}

	// 2. 自动获取第一个启用的 AD 配置
	var adConfig models.ADConfig
	err := db.WithContext(ctx).
		Where("status = ?", models.ADConfigStatusEnabled).
		Order("created_at ASC").
		First(&adConfig).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("未找到启用的AD配置，请先在AD域配置中启用至少一个配置")
		}
		return "", fmt.Errorf("查询AD配置失败: %w", err)
	}

	applogger.Infof("自动使用AD配置: %s (ID: %s)", adConfig.ConfigName, adConfig.ID)
	return adConfig.ID, nil
}

// ==================== 辅助函数 ====================

// ScheduleADSyncForConfig 为指定配置安排一次性同步
func ScheduleADSyncForConfig(configID string, delay time.Duration) {
	if globalADSyncScheduler == nil {
		applogger.Errorf("AD同步调度器未初始化")
		return
	}

	go func() {
		select {
		case <-globalADSyncScheduler.ctx.Done():
			// 调度器已停止，取消延迟任务
			return
		case <-time.After(delay):
			// 延迟结束，执行同步
			syncCtx, cancel := context.WithTimeout(context.Background(), adSchedulerSyncTimeout)
			defer cancel()
			globalADSyncScheduler.syncADConfig(syncCtx, configID)
		}
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
			entries := globalADSyncScheduler.cron.Entries()
			if len(entries) > 0 {
				// Entry 0: 全量同步 (每5分钟)
				status["full_sync_next_run"] = getNextRunTime(globalADSyncScheduler.cron, entries[0])
			}
			status["next_run"] = getNextRunTime(globalADSyncScheduler.cron)
		}
	}

	return status
}

// getNextRunTime 获取下次运行时间（支持指定entry）
func getNextRunTime(cron *cron.Cron, entries ...cron.Entry) string {
	if len(entries) == 0 {
		allEntries := cron.Entries()
		if len(allEntries) == 0 {
			return "未设置"
		}
		entry := allEntries[0]
		if !entry.Next.IsZero() {
			return entry.Next.Format("2006-01-02 15:04:05")
		}
	} else {
		entry := entries[0]
		if !entry.Next.IsZero() {
			return entry.Next.Format("2006-01-02 15:04:05")
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

// getPool 返回 scheduler 持有的全局账号池单例（W-04）。
// 若 StartADSyncScheduler 未执行（pool 为 nil），则惰性初始化并写回字段，避免所有 caller 各自 NewAccountPool
// 造成 Pitfall 4（独立缓存 → 熔断后账号仍被选中）。所有 caller 必须通过此方法取池。
func (s *ADSyncScheduler) getPool() addomain.AccountPool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pool == nil {
		// 惰性初始化（兜底 StartADSyncScheduler 未设置 pool 的极端场景）
		s.pool = addomain.NewAccountPool(s.db, nil)
	}
	return s.pool
}

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// PartitionService 分区服务接口（避免循环依赖）
// 定义为最小接口，由 internal/services/mac_history_partition 实现
type PartitionService interface {
	CreateMonthlyPartition(ctx context.Context, year int, month int) error
	EnsurePartitionsExist(ctx context.Context, monthsAhead int) error
	DropExpiredPartitions(ctx context.Context) error
	GetRetentionDays(ctx context.Context) int
}

// 全局分区服务实例（依赖注入）
// 遵循 Go 最佳实践：使用 sync.Once 确保单例初始化的线程安全
var (
	globalPartitionService     PartitionService
	globalPartitionServiceOnce sync.Once
)

// SetPartitionService 设置全局分区服务（线程安全）
// 在 Core 初始化时调用，注入分区服务实例
func SetPartitionService(service PartitionService) {
	globalPartitionServiceOnce.Do(func() {
		globalPartitionService = service
		applogger.Infof("MAC历史分区服务已注入")
	})
}

// RegisterMACHistoryTasks 注册MAC地址历史相关的定时任务
// 包括：
// - mac_history_cleanup: 每天凌晨2点清理过期分区
func RegisterMACHistoryTasks(scheduler *Scheduler, db *gorm.DB) {
	// 注册MAC历史清理任务处理器
	scheduler.RegisterTask("mac_history_cleanup", func(ctx context.Context, params map[string]interface{}) error {
		return executeMACHistoryCleanup(ctx, params)
	})

	applogger.Infof("MAC历史清理任务处理器已注册")

	// 向数据库添加任务记录
	if db == nil {
		applogger.Warnf("MAC历史清理任务数据库注册跳过（数据库未设置）")
		return
	}

	// 检查任务是否已存在
	var existingJob models.Job
	err := db.Where("job_name = ?", "MAC历史数据清理").First(&existingJob).Error
	if err == nil {
		applogger.Infof("MAC历史清理任务已存在于数据库，跳过创建")
		return
	}

	if err != gorm.ErrRecordNotFound {
		applogger.Warnf("查询MAC历史清理任务失败: %v", err)
		return
	}

	// 计算下次执行时间（明天凌晨2点）
	now := time.Now()
	nextRunTime := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	remark := "MAC地址历史数据清理任务，每天凌晨2点执行，清理过期的月度分区"

	newJob := &models.Job{
		JobName:        "MAC历史数据清理",
		JobGroup:       "SYSTEM",
		InvokeTarget:   "mac_history_cleanup",
		CronExpression: "0 0 2 * * ?",
		MisfirePolicy:  models.MisfirePolicyExecuteOnce,
		Status:         0,
		NextRunTime:    &nextRunTime,
		Remark:         &remark,
	}

	if err := scheduler.AddJob(newJob); err != nil {
		applogger.Warnf("创建MAC历史清理任务失败: %v", err)
		return
	}

	applogger.Infof("MAC历史清理任务已添加到数据库（下次执行: %v）", nextRunTime.Format("2006-01-02 15:04:05"))
}

// executeMACHistoryCleanup 执行MAC历史清理任务
// 清理过期的MAC历史表分区，释放存储空间
func executeMACHistoryCleanup(ctx context.Context, params map[string]interface{}) error {
	// 检查分区服务是否已初始化
	if globalPartitionService == nil {
		return fmt.Errorf("MAC历史分区服务未初始化")
	}

	applogger.Infof("[MAC历史清理] 开始清理过期分区")

	// 执行分区清理
	if err := globalPartitionService.DropExpiredPartitions(ctx); err != nil {
		applogger.Errorf("[MAC历史清理] 清理失败: %v", err)
		return fmt.Errorf("清理MAC历史分区失败: %w", err)
	}

	applogger.Infof("[MAC历史清理] 清理完成")
	return nil
}

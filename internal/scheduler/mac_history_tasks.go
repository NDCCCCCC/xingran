package scheduler

import (
	"context"
	"fmt"
	"sync"

	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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
func RegisterMACHistoryTasks(scheduler *Scheduler) {
	// 注册MAC历史清理任务
	scheduler.RegisterTask("mac_history_cleanup", func(ctx context.Context, params map[string]interface{}) error {
		return executeMACHistoryCleanup(ctx, params)
	})

	applogger.Infof("MAC历史清理任务处理器已注册")
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

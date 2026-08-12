package rpa

import (
	"context"
	"fmt"
	"sync"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"gorm.io/gorm"
)

// MetricsService 监控指标服务接口
type MetricsService interface {
	// GetQueueLength 获取任务队列长度
	GetQueueLength(ctx context.Context) (int, error)

	// GetActiveWorkers 获取活跃 Worker 数量
	GetActiveWorkers(ctx context.Context) (int, error)

	// GetWorkerCapacity 获取 Worker 总容量
	GetWorkerCapacity(ctx context.Context) (int, error)

	// GetPendingExecutions 获取待执行任务数
	GetPendingExecutions(ctx context.Context) (int, error)

	// GetRunningExecutions 获取执行中任务数
	GetRunningExecutions(ctx context.Context) (int, error)

	// GetAverageExecutionTime 获取平均执行时间
	GetAverageExecutionTime(ctx context.Context) (time.Duration, error)

	// RecordScalingEvent 记录扩缩容事件
	RecordScalingEvent(ctx context.Context, event *ScalingEvent) error

	// GetScalingHistory 获取扩缩容历史
	GetScalingHistory(ctx context.Context, limit int) ([]ScalingEvent, error)
}

// metricsServiceImpl 监控指标服务实现
type metricsServiceImpl struct {
	db *gorm.DB
}

// NewMetricsService 创建监控指标服务
func NewMetricsService(db *gorm.DB) MetricsService {
	return &metricsServiceImpl{db: db}
}

// GetQueueLength 获取任务队列长度
// 统计状态为 pending 的执行记录数
func (s *metricsServiceImpl) GetQueueLength(ctx context.Context) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Execution{}).
		Where("status = ? AND created_at > ?",
			string(rpamodels.RPAExecutionStatusPending),
			time.Now().Add(-24*time.Hour)).
		Count(&count).Error
	return int(count), err
}

// GetActiveWorkers 获取活跃 Worker 数量
func (s *metricsServiceImpl) GetActiveWorkers(ctx context.Context) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Worker{}).
		Where("status = ? OR status = ?",
			rpamodels.WorkerStatusOnline,
			rpamodels.WorkerStatusBusy).
		Count(&count).Error
	return int(count), err
}

// GetWorkerCapacity 获取 Worker 总容量
// 计算所有在线 Worker 的最大并发数之和
func (s *metricsServiceImpl) GetWorkerCapacity(ctx context.Context) (int, error) {
	type Result struct {
		TotalCapacity int
	}

	var result Result
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Worker{}).
		Select("COALESCE(SUM(max_concurrency), 0) as total_capacity").
		Where("status = ? OR status = ?",
			rpamodels.WorkerStatusOnline,
			rpamodels.WorkerStatusBusy).
		Scan(&result).Error

	return result.TotalCapacity, err
}

// GetPendingExecutions 获取待执行任务数
func (s *metricsServiceImpl) GetPendingExecutions(ctx context.Context) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Execution{}).
		Where("status = ?", string(rpamodels.RPAExecutionStatusPending)).
		Count(&count).Error
	return int(count), err
}

// GetRunningExecutions 获取执行中任务数
func (s *metricsServiceImpl) GetRunningExecutions(ctx context.Context) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Execution{}).
		Where("status = ?", string(rpamodels.RPAExecutionStatusRunning)).
		Count(&count).Error
	return int(count), err
}

// GetAverageExecutionTime 获取平均执行时间（毫秒）
func (s *metricsServiceImpl) GetAverageExecutionTime(ctx context.Context) (time.Duration, error) {
	type Result struct {
		AvgDuration float64 // PostgreSQL AVG() 返回 float64
	}

	var result Result
	// 查询过去 24 小时内成功执行的平均时间
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Execution{}).
		Select("AVG(duration) as avg_duration").
		Where("status = ? AND end_time > ?",
			string(rpamodels.RPAExecutionStatusSuccess),
			time.Now().Add(-24*time.Hour)).
		Scan(&result).Error

	if err != nil || result.AvgDuration == 0 {
		return 30 * time.Second, nil // 默认 30 秒
	}

	return time.Duration(result.AvgDuration) * time.Millisecond, nil
}

// RecordScalingEvent 记录扩缩容事件
func (s *metricsServiceImpl) RecordScalingEvent(ctx context.Context, event *ScalingEvent) error {
	return s.db.WithContext(ctx).Create(event).Error
}

// GetScalingHistory 获取扩缩容历史
func (s *metricsServiceImpl) GetScalingHistory(ctx context.Context, limit int) ([]ScalingEvent, error) {
	var events []ScalingEvent
	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ScalingEvent 扩缩容事件记录
type ScalingEvent struct {
	ID              string    `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	EventType       string    `gorm:"size:20;not null" json:"eventType"` // scale_up, scale_down
	FromCount       int       `json:"fromCount"`
	ToCount         int       `json:"toCount"`
	TriggerReason   string    `gorm:"type:text" json:"triggerReason"`
	QueueLength     int       `json:"queueLength"`
	ActiveWorkers   int       `json:"activeWorkers"`
	WorkerCapacity  int       `json:"workerCapacity"`
	AverageExecTime int       `json:"averageExecTime"`                         // 毫秒
	ContainerIDs    string    `gorm:"type:text" json:"containerIds"`           // JSON 数组
	Status          string    `gorm:"size:20;default:'success'" json:"status"` // success, failed
	ErrorMessage    string    `gorm:"type:text" json:"errorMessage"`
}

// TableName 指定表名
func (ScalingEvent) TableName() string {
	return "sys_rpa_scaling_events"
}

// ScalingMetrics 扩缩容决策指标
type ScalingMetrics struct {
	QueueLength     int           // 待执行任务数
	RunningTasks    int           // 执行中任务数
	ActiveWorkers   int           // 活跃 Worker 数
	WorkerCapacity  int           // Worker 总容量（最大并发数）
	AverageExecTime time.Duration // 平均执行时间
	Timestamp       time.Time     // 采集时间
}

// MetricsCollector 指标采集器
type MetricsCollector struct {
	metricsService MetricsService
	interval       time.Duration
	stopCh         chan struct{}
	mu             sync.RWMutex
	latestMetrics  *ScalingMetrics
}

// NewMetricsCollector 创建指标采集器
func NewMetricsCollector(service MetricsService, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metricsService: service,
		interval:       interval,
		stopCh:         make(chan struct{}),
	}
}

// Start 启动指标采集
func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	// 立即采集一次
	mc.collectMetrics(ctx)

	for {
		select {
		case <-ticker.C:
			mc.collectMetrics(ctx)
		case <-mc.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop 停止指标采集
func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
}

// GetLatestMetrics 获取最新指标
func (mc *MetricsCollector) GetLatestMetrics() *ScalingMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.latestMetrics
}

// collectMetrics 采集指标
func (mc *MetricsCollector) collectMetrics(ctx context.Context) {
	pending, _ := mc.metricsService.GetPendingExecutions(ctx)
	running, _ := mc.metricsService.GetRunningExecutions(ctx)
	activeWorkers, _ := mc.metricsService.GetActiveWorkers(ctx)
	capacity, _ := mc.metricsService.GetWorkerCapacity(ctx)
	avgTime, _ := mc.metricsService.GetAverageExecutionTime(ctx)

	metrics := &ScalingMetrics{
		QueueLength:     pending,
		RunningTasks:    running,
		ActiveWorkers:   activeWorkers,
		WorkerCapacity:  capacity,
		AverageExecTime: avgTime,
		Timestamp:       time.Now(),
	}

	mc.mu.Lock()
	mc.latestMetrics = metrics
	mc.mu.Unlock()
}

// ShouldScaleUp 判断是否需要扩容
func (m *ScalingMetrics) ShouldScaleUp(minWorkers, maxWorkers int, scaleUpThreshold float64) bool {
	if m.ActiveWorkers >= maxWorkers {
		return false
	}

	// 条件1: 队列积压超过容量的一定比例
	utilization := 0.0
	if m.WorkerCapacity > 0 {
		utilization = float64(m.RunningTasks) / float64(m.WorkerCapacity)
	}

	queuePressure := 0.0
	if m.WorkerCapacity > 0 {
		queuePressure = float64(m.QueueLength) / float64(m.WorkerCapacity)
	}

	// 队列积压或利用率高
	return queuePressure > scaleUpThreshold || utilization > scaleUpThreshold
}

// ShouldScaleDown 判断是否需要缩容
func (m *ScalingMetrics) ShouldScaleDown(minWorkers int, scaleDownThreshold float64) bool {
	if m.ActiveWorkers <= minWorkers {
		return false
	}

	// 容量利用率低
	utilization := 0.0
	if m.WorkerCapacity > 0 {
		utilization = float64(m.RunningTasks) / float64(m.WorkerCapacity)
	}

	return utilization < scaleDownThreshold && m.QueueLength == 0
}

// CalculateTargetWorkers 计算目标 Worker 数量
func (m *ScalingMetrics) CalculateTargetWorkers(minWorkers, maxWorkers int, scaleUpThreshold float64) int {
	if m.WorkerCapacity == 0 {
		return minWorkers
	}

	// 基于队列长度和利用率计算
	totalLoad := float64(m.QueueLength + m.RunningTasks)

	// 需要的容量 = 当前负载 / 目标利用率
	targetCapacity := int(totalLoad / scaleUpThreshold)

	// 估算需要的 Worker 数量（假设每个 Worker 容量为 3）
	avgCapacityPerWorker := 3.0
	if m.ActiveWorkers > 0 {
		avgCapacityPerWorker = float64(m.WorkerCapacity) / float64(m.ActiveWorkers)
	}

	targetWorkers := int(float64(targetCapacity) / avgCapacityPerWorker)

	// 限制在最小和最大值之间
	if targetWorkers < minWorkers {
		targetWorkers = minWorkers
	}
	if targetWorkers > maxWorkers {
		targetWorkers = maxWorkers
	}

	return targetWorkers
}

// DescribeDecision 描述扩缩容决策原因
func (m *ScalingMetrics) DescribeDecision(action string) string {
	switch action {
	case "scale_up":
		return fmt.Sprintf("队列长度: %d, 运行中: %d, Worker容量: %d/%d, 需要扩容",
			m.QueueLength, m.RunningTasks, m.RunningTasks, m.WorkerCapacity)
	case "scale_down":
		return fmt.Sprintf("队列长度: %d, 运行中: %d, Worker容量: %d/%d, 利用率低可以缩容",
			m.QueueLength, m.RunningTasks, m.RunningTasks, m.WorkerCapacity)
	default:
		return fmt.Sprintf("队列长度: %d, 运行中: %d, Worker容量: %d/%d, 无需调整",
			m.QueueLength, m.RunningTasks, m.RunningTasks, m.WorkerCapacity)
	}
}

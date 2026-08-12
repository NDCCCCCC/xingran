package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ScalingService 扩缩容服务接口
type ScalingService interface {
	// Start 启动自动扩缩容监控
	Start(ctx context.Context) error

	// Stop 停止自动扩缩容监控
	Stop()

	// MonitorAndScale 执行一次监控和扩缩容检查
	MonitorAndScale(ctx context.Context) error

	// ScaleUp 手动扩容
	ScaleUp(ctx context.Context, count int) ([]string, error)

	// ScaleDown 手动缩容
	ScaleDown(ctx context.Context, containerIDs []string) error

	// GetStatus 获取扩缩容状态
	GetStatus() *ScalingStatus

	// GetHistory 获取扩缩容历史
	GetHistory(ctx context.Context, limit int) ([]ScalingEvent, error)
}

// ScalingConfig 扩缩容配置
type ScalingConfig struct {
	Enabled           bool          `mapstructure:"enabled"`             // 是否启用自动扩缩容
	CheckInterval     time.Duration `mapstructure:"check_interval"`      // 检查间隔
	MinWorkers        int           `mapstructure:"min_workers"`         // 最小 Worker 数量
	MaxWorkers        int           `mapstructure:"max_workers"`         // 最大 Worker 数量
	ScaleUpThreshold  float64       `mapstructure:"scale_up_threshold"`  // 扩容阈值 (0-1)
	ScaleDownCooldown time.Duration `mapstructure:"scale_down_cooldown"` // 缩容冷却时间
	ScaleUpLimit      int           `mapstructure:"scale_up_limit"`      // 单次扩容上限
	EnableMockDocker  bool          `mapstructure:"enable_mock_docker"`  // 是否使用模拟 Docker 客户端
}

// DefaultScalingConfig 默认扩缩容配置
func DefaultScalingConfig() *ScalingConfig {
	return &ScalingConfig{
		Enabled:           false,
		CheckInterval:     30 * time.Second,
		MinWorkers:        2,
		MaxWorkers:        10,
		ScaleUpThreshold:  0.7, // 70% 容量使用时扩容
		ScaleDownCooldown: 5 * time.Minute,
		ScaleUpLimit:      3,
		EnableMockDocker:  true, // 默认使用模拟模式
	}
}

// scalingServiceImpl 扩缩容服务实现
type scalingServiceImpl struct {
	db             *gorm.DB
	config         *ScalingConfig
	dockerConfig   *DockerConfig
	metricsService MetricsService
	dockerClient   DockerClient
	collector      *MetricsCollector

	stopCh        chan struct{}
	mu            sync.RWMutex
	status        *ScalingStatus
	lastScaleDown time.Time // 上次缩容时间
}

// ScalingStatus 扩缩容状态
type ScalingStatus struct {
	IsRunning          bool      `json:"isRunning"`
	LastCheckTime      time.Time `json:"lastCheckTime"`
	LastScaleTime      time.Time `json:"lastScaleTime"`
	TotalScaleUps      int       `json:"totalScaleUps"`
	TotalScaleDowns    int       `json:"totalScaleDowns"`
	CurrentWorkers     int       `json:"currentWorkers"`
	AutoScalingEnabled bool      `json:"autoScalingEnabled"`
}

// NewScalingService 创建扩缩容服务
func NewScalingService(db *gorm.DB, config *ScalingConfig, dockerConfig *DockerConfig) ScalingService {
	metricsService := NewMetricsService(db)

	// 创建 Docker 客户端
	var dockerClient DockerClient
	if config.EnableMockDocker {
		dockerClient = NewMockDockerClient()
		applogger.Warnf("RPA 扩缩容使用模拟 Docker 客户端（测试模式）")
	} else {
		dockerClient = NewDockerClient(dockerConfig)
	}

	// 创建指标采集器
	collector := NewMetricsCollector(metricsService, config.CheckInterval)

	return &scalingServiceImpl{
		db:             db,
		config:         config,
		dockerConfig:   dockerConfig,
		metricsService: metricsService,
		dockerClient:   dockerClient,
		collector:      collector,
		stopCh:         make(chan struct{}),
		status: &ScalingStatus{
			AutoScalingEnabled: config.Enabled,
		},
	}
}

// Start 启动自动扩缩容监控
func (s *scalingServiceImpl) Start(ctx context.Context) error {
	if !s.config.Enabled {
		applogger.Infof("RPA 自动扩缩容未启用")
		return nil
	}

	// 检查 Docker 健康状态
	if !s.dockerClient.IsHealthy(ctx) {
		return fmt.Errorf("Docker 服务不可用，无法启动扩缩容服务")
	}

	s.mu.Lock()
	s.status.IsRunning = true
	s.mu.Unlock()

	// 启动指标采集
	go s.collector.Start(ctx)

	// 启动扩缩容监控循环
	go s.monitoringLoop(ctx)

	applogger.Infof("RPA 自动扩缩容服务已启动 (检查间隔: %v, Worker范围: %d-%d)",
		s.config.CheckInterval, s.config.MinWorkers, s.config.MaxWorkers)

	return nil
}

// Stop 停止自动扩缩容监控
func (s *scalingServiceImpl) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.status.IsRunning {
		return
	}

	close(s.stopCh)
	s.collector.Stop()
	s.status.IsRunning = false

	applogger.Infof("RPA 自动扩缩容服务已停止")
}

// monitoringLoop 监控循环
func (s *scalingServiceImpl) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// 立即执行一次检查（后台监控循环，忽略错误）
	_ = s.MonitorAndScale(ctx)

	for {
		select {
		case <-ticker.C:
			_ = s.MonitorAndScale(ctx) // 后台监控循环，忽略错误
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// MonitorAndScale 执行一次监控和扩缩容检查
func (s *scalingServiceImpl) MonitorAndScale(ctx context.Context) error {
	metrics := s.collector.GetLatestMetrics()
	if metrics == nil {
		return fmt.Errorf("指标数据未就绪")
	}

	s.mu.Lock()
	s.status.LastCheckTime = time.Now()
	s.status.CurrentWorkers = metrics.ActiveWorkers
	s.mu.Unlock()

	applogger.Debugf("扩缩容检查: %s", metrics.DescribeDecision("check"))

	// 判断是否需要扩容
	if metrics.ShouldScaleUp(s.config.MinWorkers, s.config.MaxWorkers, s.config.ScaleUpThreshold) {
		return s.scaleUp(ctx, metrics)
	}

	// 判断是否需要缩容（检查冷却时间）
	if time.Since(s.lastScaleDown) >= s.config.ScaleDownCooldown {
		if metrics.ShouldScaleDown(s.config.MinWorkers, 0.3) {
			return s.scaleDown(ctx, metrics)
		}
	}

	return nil
}

// scaleUp 扩容
func (s *scalingServiceImpl) scaleUp(ctx context.Context, metrics *ScalingMetrics) error {
	targetWorkers := metrics.CalculateTargetWorkers(
		s.config.MinWorkers,
		s.config.MaxWorkers,
		s.config.ScaleUpThreshold,
	)

	// 计算需要扩容的数量
	needCount := targetWorkers - metrics.ActiveWorkers
	if needCount > s.config.ScaleUpLimit {
		needCount = s.config.ScaleUpLimit
	}

	if needCount <= 0 {
		return nil
	}

	applogger.Infof("开始扩容: 当前 %d -> 目标 %d (本次 +%d)",
		metrics.ActiveWorkers, targetWorkers, needCount)

	containerIDs, err := s.dockerClient.ScaleUp(ctx, needCount)
	if err != nil {
		s.recordScalingEvent(ctx, "scale_up", metrics.ActiveWorkers, metrics.ActiveWorkers,
			metrics, fmt.Sprintf("扩容失败: %v", err), "failed", "")
		return fmt.Errorf("Docker 扩容失败: %w", err)
	}

	// 记录扩容事件
	containerIDsJSON, _ := json.Marshal(containerIDs)
	s.recordScalingEvent(ctx, "scale_up", metrics.ActiveWorkers, metrics.ActiveWorkers+needCount,
		metrics, metrics.DescribeDecision("scale_up"), "success", string(containerIDsJSON))

	s.mu.Lock()
	s.status.LastScaleTime = time.Now()
	s.status.TotalScaleUps++
	s.mu.Unlock()

	applogger.Infof("扩容完成: 创建了 %d 个 Worker 容器 %v", needCount, containerIDs)

	return nil
}

// scaleDown 缩容
func (s *scalingServiceImpl) scaleDown(ctx context.Context, metrics *ScalingMetrics) error {
	// 计算需要缩容的数量（每次缩容 1 个，避免过度缩容）
	needCount := 1
	if metrics.ActiveWorkers-needCount < s.config.MinWorkers {
		needCount = metrics.ActiveWorkers - s.config.MinWorkers
	}

	if needCount <= 0 {
		return nil
	}

	applogger.Infof("开始缩容: 当前 %d -> 目标 %d (本次 -%d)",
		metrics.ActiveWorkers, metrics.ActiveWorkers-needCount, needCount)

	// 获取需要停止的容器（选择最空闲的）
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		s.recordScalingEvent(ctx, "scale_down", metrics.ActiveWorkers, metrics.ActiveWorkers,
			metrics, fmt.Sprintf("获取容器列表失败: %v", err), "failed", "")
		return fmt.Errorf("获取容器列表失败: %w", err)
	}

	// 选择需要停止的容器（优先选择运行中的容器）
	var containerIDs []string
	for _, container := range containers {
		if len(containerIDs) >= needCount {
			break
		}
		if container.State == "running" {
			containerIDs = append(containerIDs, container.ID)
		}
	}

	if len(containerIDs) == 0 {
		applogger.Infof("没有可缩容的容器")
		return nil
	}

	err = s.dockerClient.ScaleDown(ctx, containerIDs)
	if err != nil {
		s.recordScalingEvent(ctx, "scale_down", metrics.ActiveWorkers, metrics.ActiveWorkers,
			metrics, fmt.Sprintf("缩容失败: %v", err), "failed", "")
		return fmt.Errorf("Docker 缩容失败: %w", err)
	}

	// 记录缩容事件
	containerIDsJSON, _ := json.Marshal(containerIDs)
	s.recordScalingEvent(ctx, "scale_down", metrics.ActiveWorkers, metrics.ActiveWorkers-needCount,
		metrics, metrics.DescribeDecision("scale_down"), "success", string(containerIDsJSON))

	s.lastScaleDown = time.Now()

	s.mu.Lock()
	s.status.LastScaleTime = time.Now()
	s.status.TotalScaleDowns++
	s.mu.Unlock()

	applogger.Infof("缩容完成: 停止了 %d 个 Worker 容器 %v", needCount, containerIDs)

	return nil
}

// recordScalingEvent 记录扩缩容事件
func (s *scalingServiceImpl) recordScalingEvent(ctx context.Context, eventType string, fromCount, toCount int,
	metrics *ScalingMetrics, reason string, status string, containerIDs string) {

	event := &ScalingEvent{
		ID:              uuid.New().String(),
		EventType:       eventType,
		FromCount:       fromCount,
		ToCount:         toCount,
		TriggerReason:   reason,
		QueueLength:     metrics.QueueLength,
		ActiveWorkers:   metrics.ActiveWorkers,
		WorkerCapacity:  metrics.WorkerCapacity,
		AverageExecTime: int(metrics.AverageExecTime.Milliseconds()),
		ContainerIDs:    containerIDs,
		Status:          status,
	}

	if err := s.metricsService.RecordScalingEvent(ctx, event); err != nil {
		applogger.Errorf("记录扩缩容事件失败: %v", err)
	}
}

// ScaleUp 手动扩容
func (s *scalingServiceImpl) ScaleUp(ctx context.Context, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("扩容数量必须大于 0")
	}

	if count > s.config.ScaleUpLimit {
		return nil, fmt.Errorf("单次扩容数量不能超过 %d", s.config.ScaleUpLimit)
	}

	// 获取当前指标
	metrics := s.collector.GetLatestMetrics()
	if metrics == nil {
		// 临时获取
		pending, _ := s.metricsService.GetPendingExecutions(ctx)
		running, _ := s.metricsService.GetRunningExecutions(ctx)
		active, _ := s.metricsService.GetActiveWorkers(ctx)
		capacity, _ := s.metricsService.GetWorkerCapacity(ctx)
		avgTime, _ := s.metricsService.GetAverageExecutionTime(ctx)

		metrics = &ScalingMetrics{
			QueueLength:     pending,
			RunningTasks:    running,
			ActiveWorkers:   active,
			WorkerCapacity:  capacity,
			AverageExecTime: avgTime,
			Timestamp:       time.Now(),
		}
	}

	// 检查是否超过最大值
	if metrics.ActiveWorkers+count > s.config.MaxWorkers {
		return nil, fmt.Errorf("扩容后 Worker 数量 (%d) 将超过最大值 (%d)",
			metrics.ActiveWorkers+count, s.config.MaxWorkers)
	}

	containerIDs, err := s.dockerClient.ScaleUp(ctx, count)
	if err != nil {
		return nil, err
	}

	// 记录事件
	containerIDsJSON, _ := json.Marshal(containerIDs)
	s.recordScalingEvent(ctx, "scale_up", metrics.ActiveWorkers, metrics.ActiveWorkers+count,
		metrics, fmt.Sprintf("手动扩容 +%d", count), "success", string(containerIDsJSON))

	applogger.Infof("手动扩容: 创建了 %d 个 Worker 容器 %v", count, containerIDs)

	return containerIDs, nil
}

// ScaleDown 手动缩容
func (s *scalingServiceImpl) ScaleDown(ctx context.Context, containerIDs []string) error {
	if len(containerIDs) == 0 {
		return fmt.Errorf("请指定要停止的容器 ID")
	}

	// 获取当前指标
	metrics := s.collector.GetLatestMetrics()
	if metrics == nil {
		active, _ := s.metricsService.GetActiveWorkers(ctx)
		metrics = &ScalingMetrics{
			ActiveWorkers: active,
			Timestamp:     time.Now(),
		}
	}

	// 检查是否低于最小值
	if metrics.ActiveWorkers-len(containerIDs) < s.config.MinWorkers {
		return fmt.Errorf("缩容后 Worker 数量 (%d) 将低于最小值 (%d)",
			metrics.ActiveWorkers-len(containerIDs), s.config.MinWorkers)
	}

	err := s.dockerClient.ScaleDown(ctx, containerIDs)
	if err != nil {
		return err
	}

	// 记录事件
	containerIDsJSON, _ := json.Marshal(containerIDs)
	s.recordScalingEvent(ctx, "scale_down", metrics.ActiveWorkers, metrics.ActiveWorkers-len(containerIDs),
		metrics, fmt.Sprintf("手动缩容 -%d", len(containerIDs)), "success", string(containerIDsJSON))

	applogger.Infof("手动缩容: 停止了 %d 个 Worker 容器 %v", len(containerIDs), containerIDs)

	return nil
}

// GetStatus 获取扩缩容状态
func (s *scalingServiceImpl) GetStatus() *ScalingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本
	status := *s.status
	return &status
}

// GetHistory 获取扩缩容历史
func (s *scalingServiceImpl) GetHistory(ctx context.Context, limit int) ([]ScalingEvent, error) {
	return s.metricsService.GetScalingHistory(ctx, limit)
}

// SyncWorkersWithContainers 同步 Worker 记录与 Docker 容器状态
// 这个方法用于确保数据库中的 Worker 记录与实际运行的容器一致
func (s *scalingServiceImpl) SyncWorkersWithContainers(ctx context.Context) error {
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("获取容器列表失败: %w", err)
	}

	// 获取数据库中的 Docker Worker 记录
	var dbWorkers []rpamodels.Worker
	err = s.db.WithContext(ctx).
		Where("docker_container_id IS NOT NULL AND docker_container_id != ''").
		Find(&dbWorkers).Error
	if err != nil {
		return fmt.Errorf("获取数据库 Worker 记录失败: %w", err)
	}

	// 构建 container ID 映射
	containerMap := make(map[string]bool)
	for _, container := range containers {
		containerMap[container.ID] = true
	}

	// 标记不存在的容器为离线
	now := time.Now().Unix()
	for _, worker := range dbWorkers {
		if !containerMap[worker.DockerContainerID] {
			if err := s.db.WithContext(ctx).
				Model(&rpamodels.Worker{}).
				Where("id = ?", worker.ID).
				Updates(map[string]interface{}{
					"status":         rpamodels.WorkerStatusOffline,
					"last_heartbeat": &now,
				}).Error; err != nil {
				applogger.Warnf("标记 Worker %s 离线失败: %v", worker.WorkerName, err)
			} else {
				applogger.Infof("标记 Worker %s 为离线 (容器 %s 不存在)", worker.WorkerName, worker.DockerContainerID)
			}
		}
	}

	return nil
}

// GetWorkerStats 获取 Worker 统计信息
func (s *scalingServiceImpl) GetWorkerStats(ctx context.Context) (*WorkerStats, error) {
	metrics := s.collector.GetLatestMetrics()
	if metrics == nil {
		return nil, fmt.Errorf("指标数据未就绪")
	}

	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	var running, stopped, other int
	for _, container := range containers {
		switch container.State {
		case "running":
			running++
		case "exited", "dead":
			stopped++
		default:
			other++
		}
	}

	return &WorkerStats{
		TotalContainers:   len(containers),
		RunningContainers: running,
		StoppedContainers: stopped,
		OtherContainers:   other,
		QueueLength:       metrics.QueueLength,
		RunningTasks:      metrics.RunningTasks,
		WorkerCapacity:    metrics.WorkerCapacity,
		AverageExecTime:   metrics.AverageExecTime.String(),
	}, nil
}

// WorkerStats Worker 统计信息
type WorkerStats struct {
	TotalContainers   int    `json:"totalContainers"`
	RunningContainers int    `json:"runningContainers"`
	StoppedContainers int    `json:"stoppedContainers"`
	OtherContainers   int    `json:"otherContainers"`
	QueueLength       int    `json:"queueLength"`
	RunningTasks      int    `json:"runningTasks"`
	WorkerCapacity    int    `json:"workerCapacity"`
	AverageExecTime   string `json:"averageExecTime"`
}

// FindIdleContainers 查找空闲容器
func (s *scalingServiceImpl) FindIdleContainers(ctx context.Context, limit int) ([]string, error) {
	containers, err := s.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	var idleContainers []string
	for _, container := range containers {
		if len(idleContainers) >= limit {
			break
		}

		// 获取容器统计信息
		stats, err := s.dockerClient.GetContainerStats(ctx, container.ID)
		if err != nil {
			continue
		}

		// 判断是否空闲（CPU < 10% 且内存使用 < 20%）
		memoryPercent := 0.0
		if stats.MemoryLimit > 0 {
			memoryPercent = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
		}

		if stats.CPUPercent < 10.0 && memoryPercent < 20.0 {
			idleContainers = append(idleContainers, container.ID)
		}
	}

	return idleContainers, nil
}

// ValidateScalingConfig 验证扩缩容配置
func ValidateScalingConfig(config *ScalingConfig) error {
	if config.MinWorkers < 0 {
		return fmt.Errorf("最小 Worker 数量不能为负数")
	}
	if config.MaxWorkers < config.MinWorkers {
		return fmt.Errorf("最大 Worker 数量不能小于最小 Worker 数量")
	}
	if config.ScaleUpThreshold <= 0 || config.ScaleUpThreshold > 1 {
		return fmt.Errorf("扩容阈值必须在 0-1 之间")
	}
	if config.ScaleUpLimit <= 0 {
		return fmt.Errorf("单次扩容上限必须大于 0")
	}
	if config.CheckInterval < 10*time.Second {
		return fmt.Errorf("检查间隔不能小于 10 秒")
	}
	return nil
}

// ParseContainerIDs 从字符串解析容器 ID 列表
func ParseContainerIDs(idsStr string) ([]string, error) {
	idsStr = strings.TrimSpace(idsStr)
	if idsStr == "" {
		return nil, fmt.Errorf("容器 ID 列表为空")
	}

	var ids []string
	if strings.HasPrefix(idsStr, "[") {
		// JSON 数组格式
		if err := json.Unmarshal([]byte(idsStr), &ids); err != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}
	} else {
		// 逗号分隔格式
		ids = strings.Split(idsStr, ",")
		for i, id := range ids {
			ids[i] = strings.TrimSpace(id)
		}
	}

	return ids, nil
}

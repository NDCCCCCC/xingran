package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// JobTask 任务执行接口
type JobTask interface {
	Execute(ctx context.Context) error
}

// JobExecutor 任务执行器
type JobExecutor struct {
	job       *models.Job
	db        *gorm.DB
	logger    Logger
	scheduler *Scheduler // 引用调度器以访问 taskRegistry
}

// Execute 执行任务
func (e *JobExecutor) Execute(ctx context.Context) error {
	startTime := time.Now()

	// 创建任务日志
	jobLog := &models.JobLog{
		JobName:      e.job.JobName,
		JobGroup:     e.job.JobGroup,
		InvokeTarget: e.job.InvokeTarget,
		JobMessage:   "任务开始执行",
		Status:       0, // 成功
		StartTime:    &startTime,
	}

	// 执行任务逻辑
	err := e.executeTask(ctx, jobLog)

	endTime := time.Now()
	jobLog.EndTime = &endTime
	jobLog.Duration = endTime.Sub(startTime).Milliseconds()

	// 更新任务的执行时间
	e.db.Model(e.job).Updates(map[string]interface{}{
		"prev_run_time": startTime,
		"next_run_time": e.calculateNextRunTime(),
	})

	// 保存执行日志
	if err != nil {
		jobLog.Status = 1 // 失败
		errMsg := err.Error()
		jobLog.ExceptionInfo = &errMsg
		jobLog.JobMessage = fmt.Sprintf("任务执行失败: %s", errMsg)
		e.logger.Errorf("任务执行失败 [%s.%s]: %v", e.job.JobName, e.job.JobGroup, err)
	} else {
		jobLog.JobMessage = "任务执行成功"
		e.logger.Infof("任务执行成功 [%s.%s], 耗时: %dms", e.job.JobName, e.job.JobGroup, jobLog.Duration)
	}

	if logErr := e.db.Create(jobLog).Error; logErr != nil {
		e.logger.Errorf("保存任务日志失败: %v", logErr)
	}

	return err
}

// executeTask 执行具体任务逻辑
func (e *JobExecutor) executeTask(ctx context.Context, jobLog *models.JobLog) error {
	// 根据 InvokeTarget 查找并执行注册的任务处理函数
	if e.scheduler == nil {
		return fmt.Errorf("调度器引用为空")
	}

	taskHandler := e.scheduler.GetTaskHandler(e.job.InvokeTarget)
	if taskHandler == nil {
		return fmt.Errorf("未找到任务处理器: %s", e.job.InvokeTarget)
	}

	e.logger.Infof("执行任务: %s, 目标: %s", e.job.JobName, e.job.InvokeTarget)

	// 调用注册的任务处理函数
	// 任务处理函数接收 context 和 params，这里传入空参数
	return taskHandler(ctx, nil)
}

// calculateNextRunTime 计算下次执行时间
func (e *JobExecutor) calculateNextRunTime() *time.Time {
	if e.job.CronExpression == "" {
		return nil
	}

	// 使用支持秒的解析器 - 与 cron.WithSeconds() 保持一致
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(e.job.CronExpression)
	if err != nil {
		return nil
	}

	nextTime := schedule.Next(time.Now())
	return &nextTime
}

// Logger 日志接口
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *defaultLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Scheduler 定时任务调度器
type Scheduler struct {
	cron         *cron.Cron
	db           *gorm.DB
	jobs         map[string]*models.Job
	executors    map[string]cron.EntryID
	mu           sync.RWMutex
	logger       Logger
	running      bool
	taskRegistry map[string]func(ctx context.Context, params map[string]interface{}) error
}

// NewScheduler 创建新的调度器
func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
		db:   db,
		jobs: make(map[string]*models.Job),
		executors: make(map[string]cron.EntryID),
		logger:       &defaultLogger{},
		taskRegistry: make(map[string]func(ctx context.Context, params map[string]interface{}) error),
		running:      false,
	}
}

// SetLogger 设置日志
func (s *Scheduler) SetLogger(logger Logger) {
	s.logger = logger
}

// RegisterTask 注册任务类型
func (s *Scheduler) RegisterTask(taskType string, handler func(ctx context.Context, params map[string]interface{}) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskRegistry[taskType] = handler
}

// GetTaskHandler 获取任务处理器
func (s *Scheduler) GetTaskHandler(taskType string) func(ctx context.Context, params map[string]interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taskRegistry[taskType]
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("调度器已经在运行")
	}

	// 从数据库加载所有启用的任务
	var jobs []models.Job
	if err := s.db.Where("status = ?", 0).Find(&jobs).Error; err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	// 添加所有启用的任务
	for _, job := range jobs {
		if err := s.addJob(&job); err != nil {
			s.logger.Errorf("添加任务失败 [%s]: %v", job.JobName, err)
		}
	}

	s.cron.Start()
	s.running = true
	s.logger.Infof("调度器已启动，共加载 %d 个任务", len(s.executors))

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()

	// 等待所有正在运行的任务完成，但设置 5 秒超时
	// 避免某个任务卡住导致整个服务无法关闭
	timeout := time.After(5 * time.Second)
	select {
	case <-ctx.Done():
		// 所有任务正常完成
		s.logger.Infof("调度器已停止")
	case <-timeout:
		// 超时强制停止
		s.logger.Warnf("调度器停止超时，强制关闭")
	}

	s.running = false
}

// AddJob 添加任务
func (s *Scheduler) AddJob(job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		// 调度器运行中，直接添加
		return s.addJob(job)
	}

	// 调度器未运行，只保存到内存
	s.jobs[job.ID] = job
	return nil
}

// addJob 内部添加任务方法
func (s *Scheduler) addJob(job *models.Job) error {
	// 验证 Cron 表达式 - 使用 Parse 支持6位表达式(秒 分 时 日 月 周)
	// ParseStandard 只支持5位，所以移除这个验证，由 AddFunc 自动验证

	// 创建执行器，传入调度器引用
	executor := &JobExecutor{
		job:       job,
		db:        s.db,
		logger:    s.logger,
		scheduler: s, // 传递调度器引用以访问 taskRegistry
	}

	// 添加到调度器 - cron.AddFunc 会自动验证表达式格式
	jobID, err := s.cron.AddFunc(job.CronExpression, func() {
		ctx := context.Background()
		if err := executor.Execute(ctx); err != nil {
			s.logger.Errorf("任务执行异常 [%s]: %v", job.JobName, err)
		}
	})

	if err != nil {
		return fmt.Errorf("添加任务到调度器失败: %w", err)
	}

	s.jobs[job.ID] = job
	s.executors[job.ID] = jobID

	// 计算并更新下次执行时间
	nextTime := executor.calculateNextRunTime()
	s.db.Model(job).Update("next_run_time", nextTime)

	s.logger.Infof("任务已添加到调度器 [%s], 下次执行: %v", job.JobName, nextTime)
	return nil
}

// UpdateJob 更新任务
func (s *Scheduler) UpdateJob(job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 先移除旧任务
	if err := s.removeJob(job.ID); err != nil {
		return err
	}

	// 添加新任务
	if s.running {
		return s.addJob(job)
	}

	s.jobs[job.ID] = job
	return nil
}

// RemoveJob 移除任务
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.removeJob(jobID)
}

// removeJob 内部移除任务方法
func (s *Scheduler) removeJob(jobID string) error {
	if cronID, exists := s.executors[jobID]; exists {
		s.cron.Remove(cronID)
		delete(s.executors, jobID)
	}

	delete(s.jobs, jobID)
	s.logger.Infof("任务已从调度器移除 [%s]", jobID)
	return nil
}

// StartJob 启动指定任务
func (s *Scheduler) StartJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从数据库获取任务
	var job models.Job
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查是否已存在
	if _, exists := s.executors[jobID]; exists {
		return fmt.Errorf("任务已经在运行")
	}

	// 添加到调度器
	return s.addJob(&job)
}

// StopJob 停止指定任务
func (s *Scheduler) StopJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.executors[jobID]; !exists {
		return fmt.Errorf("任务未在运行")
	}

	return s.removeJob(jobID)
}

// ExecuteJob 立即执行任务
func (s *Scheduler) ExecuteJob(jobID string) error {
	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	// 如果不在内存中，从数据库获取
	if !exists || job == nil {
		var dbJob models.Job
		if err := s.db.First(&dbJob, "id = ?", jobID).Error; err != nil {
			return fmt.Errorf("任务不存在: %w", err)
		}
		job = &dbJob
	}

	executor := &JobExecutor{
		job:       job,
		db:        s.db,
		logger:    s.logger,
		scheduler: s, // 传递调度器引用
	}

	ctx := context.Background()
	return executor.Execute(ctx)
}

// GetJobStatus 获取任务状态
func (s *Scheduler) GetJobStatus(jobID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.executors[jobID]
	return exists, nil
}

// GetJobCount 获取任务统计
func (s *Scheduler) GetJobCount() (total int, running int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total = len(s.jobs)
	running = len(s.executors)
	return
}

// ==================== 网络设备定时任务注册函数 ====================
// 这些函数需要在Core初始化时注册到Scheduler

// DeviceMonitorService 设备监控服务接口
// 使用接口避免循环依赖
type DeviceMonitorService interface {
	CheckAllDevicesStatus(ctx context.Context) (int, int, error)
	UpdateAllDeviceInfo(ctx context.Context) error
	CollectAllPortStatus(ctx context.Context) error
	CollectAllMACAddresses(ctx context.Context) error
	BackupAllConfigurations(ctx context.Context) error
}

// GlobalDeviceMonitorService 全局设备监控服务
// 在Core初始化时设置
var GlobalDeviceMonitorService DeviceMonitorService

// RegisterNetworkDeviceTasks 注册网络设备相关定时任务
func RegisterNetworkDeviceTasks(scheduler *Scheduler, db *gorm.DB) {
	// 设备状态检查任务 - 每5分钟执行
	scheduler.RegisterTask("device_status_check", func(ctx context.Context, params map[string]interface{}) error {
		return executeDeviceStatusCheckTask(ctx)
	})

	// 设备信息更新任务 - 每小时执行
	scheduler.RegisterTask("device_info_update", func(ctx context.Context, params map[string]interface{}) error {
		return executeDeviceInfoUpdateTask(ctx)
	})

	// 端口状态采集任务 - 每小时执行
	scheduler.RegisterTask("port_collection", func(ctx context.Context, params map[string]interface{}) error {
		return executePortCollectionTask(ctx)
	})

	// MAC地址采集任务 - 每小时执行
	scheduler.RegisterTask("mac_collection", func(ctx context.Context, params map[string]interface{}) error {
		return executeMACCollectionTask(ctx)
	})

	// 配置备份任务 - 每天凌晨2点执行
	scheduler.RegisterTask("config_backup", func(ctx context.Context, params map[string]interface{}) error {
		return executeConfigBackupTask(ctx)
	})
}

// executeDeviceStatusCheckTask 执行设备状态检查任务
// 通过SNMP定时检查所有设备的在线/离线状态
func executeDeviceStatusCheckTask(ctx context.Context) error {
	if GlobalDeviceMonitorService == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	online, offline, err := GlobalDeviceMonitorService.CheckAllDevicesStatus(ctx)
	if err != nil {
		return fmt.Errorf("设备状态检查失败: %w", err)
	}

	log.Printf("设备状态检查完成: 在线=%d, 离线=%d", online, offline)
	return nil
}

// executeDeviceInfoUpdateTask 执行设备信息更新任务
// 通过SSH采集设备的详细信息（型号、版本、序列号等）
func executeDeviceInfoUpdateTask(ctx context.Context) error {
	if GlobalDeviceMonitorService == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := GlobalDeviceMonitorService.UpdateAllDeviceInfo(ctx); err != nil {
		return fmt.Errorf("设备信息更新失败: %w", err)
	}

	log.Printf("设备信息更新完成")
	return nil
}

// executePortCollectionTask 执行端口采集任务
// 采集所有在线设备的端口状态信息
func executePortCollectionTask(ctx context.Context) error {
	if GlobalDeviceMonitorService == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := GlobalDeviceMonitorService.CollectAllPortStatus(ctx); err != nil {
		return fmt.Errorf("端口状态采集失败: %w", err)
	}

	log.Printf("端口状态采集完成")
	return nil
}

// executeMACCollectionTask 执行MAC采集任务
// 采集所有在线设备的MAC地址表
func executeMACCollectionTask(ctx context.Context) error {
	if GlobalDeviceMonitorService == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := GlobalDeviceMonitorService.CollectAllMACAddresses(ctx); err != nil {
		return fmt.Errorf("MAC地址采集失败: %w", err)
	}

	log.Printf("MAC地址采集完成")
	return nil
}

// executeConfigBackupTask 执行配置备份任务
// 备份所有设备配置
func executeConfigBackupTask(ctx context.Context) error {
	if GlobalDeviceMonitorService == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := GlobalDeviceMonitorService.BackupAllConfigurations(ctx); err != nil {
		return fmt.Errorf("配置备份失败: %w", err)
	}

	log.Printf("配置备份完成")
	return nil
}

// SetDeviceMonitorService 设置设备监控服务
func SetDeviceMonitorService(service DeviceMonitorService) {
	GlobalDeviceMonitorService = service
}

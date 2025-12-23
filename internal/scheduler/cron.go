package scheduler

import (
	"context"
	"fmt"
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
	job    *models.Job
	db     *gorm.DB
	logger Logger
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
	// 这里可以根据 InvokeTarget 调用不同的任务
	// 示例：可以支持调用内部函数、HTTP请求、脚本执行等

	// 简单示例：模拟任务执行
	e.logger.Infof("执行任务: %s, 目标: %s", e.job.JobName, e.job.InvokeTarget)

	// TODO: 实现实际的任务调用逻辑
	// 1. 解析 InvokeTarget
	// 2. 根据类型执行不同的任务：
	//    - 内部函数调用
	//    - HTTP请求
	//    - Shell脚本
	//    - Python脚本等

	return nil
}

// calculateNextRunTime 计算下次执行时间
func (e *JobExecutor) calculateNextRunTime() *time.Time {
	if e.job.CronExpression == "" {
		return nil
	}

	schedule, err := cron.ParseStandard(e.job.CronExpression)
	if err != nil {
		return nil
	}

	nextTime := schedule.Next(time.Now())
	return &nextTime
}

// Logger 日志接口
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Infof(format string, args ...interface{}) {
	// 使用项目的日志系统
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	// 使用项目的日志系统
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
	<-ctx.Done()

	s.running = false
	s.logger.Infof("调度器已停止")
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
	// 验证 Cron 表达式
	if _, err := cron.ParseStandard(job.CronExpression); err != nil {
		return fmt.Errorf("无效的 Cron 表达式: %w", err)
	}

	// 创建执行器
	executor := &JobExecutor{
		job:    job,
		db:     s.db,
		logger: s.logger,
	}

	// 添加到调度器
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
		job:    job,
		db:     s.db,
		logger: s.logger,
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

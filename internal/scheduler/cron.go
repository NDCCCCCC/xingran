package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// defaultShutdownTimeout 调度器关闭时等待任务完成的默认超时时间
const defaultShutdownTimeout = 5 * time.Second

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

	// 解析 InvokeTarget，格式可能为 "task_type" 或 "task_type:param"
	taskType, params := e.parseInvokeTarget(e.job.InvokeTarget)

	taskHandler := e.scheduler.GetTaskHandler(taskType)
	if taskHandler == nil {
		return fmt.Errorf("未找到任务处理器: %s", taskType)
	}

	e.logger.Infof("执行任务: %s, 目标: %s", e.job.JobName, e.job.InvokeTarget)

	// 调用注册的任务处理函数
	return taskHandler(ctx, params)
}

// parseInvokeTarget 解析任务调用目标，返回任务类型和参数
//
// InvokeTarget 格式：taskType 或 taskType:param
//
// 参数格式：
//   - 无参数：taskType（如 duty_reminder）
//   - 单参数：taskType:param（如 notice_publish:xxx, periodic_workorder_create:xxx）
//
// 所有任务处理器统一通过 params["param"] 获取参数值
func (e *JobExecutor) parseInvokeTarget(invokeTarget string) (string, map[string]interface{}) {
	// 简单处理：如果包含冒号，则分割
	for i, c := range invokeTarget {
		if c == ':' {
			taskType := invokeTarget[:i]
			param := invokeTarget[i+1:]
			// 统一使用 "param" 作为参数 key，所有任务处理器都通过 params["param"] 获取参数
			return taskType, map[string]interface{}{"param": param}
		}
	}
	// 没有冒号，返回原始值
	return invokeTarget, nil
}

// calculateNextRunTime 计算下次执行时间
func (e *JobExecutor) calculateNextRunTime() *time.Time {
	if e.job.CronExpression == "" {
		return nil
	}

	// 使用调度器的时区配置创建临时 cron 来获取正确的下次执行时间
	c := cron.New(cron.WithSeconds(), cron.WithLocation(e.scheduler.cron.Location()))
	entryID, err := c.AddFunc(e.job.CronExpression, func() {})
	if err != nil {
		return nil
	}
	// 必须启动 cron 才能获取正确的 Next 时间
	c.Start()
	nextTime := c.Entry(entryID).Next
	c.Stop()
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
	applogger.Infof(format, args...)
}

func (l *defaultLogger) Warnf(format string, args ...interface{}) {
	applogger.Warnf(format, args...)
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	applogger.Errorf(format, args...)
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
	// 使用系统配置的时区 (Asia/Shanghai)
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果加载失败，使用本地时区
		loc = time.Local
	}

	return &Scheduler{
		cron:         cron.New(cron.WithSeconds(), cron.WithLocation(loc)),
		db:           db,
		jobs:         make(map[string]*models.Job),
		executors:    make(map[string]cron.EntryID),
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

// IsTaskRegistered 检查任务类型是否已在调度器中注册。
// 用于 jobService 在创建/更新定时任务时校验 InvokeTarget 白名单 (F-08)。
func (s *Scheduler) IsTaskRegistered(taskType string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.taskRegistry[taskType]
	return ok
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

	// 等待所有正在运行的任务完成，但设置 defaultShutdownTimeout 超时
	// 避免某个任务卡住导致整个服务无法关闭
	timeout := time.After(defaultShutdownTimeout)
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

	// 先保存到数据库以获取 ID
	if err := s.db.Create(job).Error; err != nil {
		return fmt.Errorf("保存任务到数据库失败: %w", err)
	}

	if s.running {
		// 调度器运行中，直接添加
		return s.addJob(job)
	}

	// 调度器未运行，只保存到数据库和内存
	s.jobs[job.ID] = job
	s.logger.Infof("任务已保存到数据库，等待调度器启动 [%s]", job.JobName)
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

	// 先移除旧任务（从调度器和数据库）
	if err := s.removeJob(job.ID); err != nil {
		return err
	}

	// 更新数据库记录
	if err := s.db.Save(job).Error; err != nil {
		return fmt.Errorf("更新任务到数据库失败: %w", err)
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
	// 从调度器中移除
	if cronID, exists := s.executors[jobID]; exists {
		s.cron.Remove(cronID)
		delete(s.executors, jobID)
	}

	// 从内存中移除
	delete(s.jobs, jobID)

	// 从数据库中删除
	s.db.Delete(&models.Job{}, "id = ?", jobID)

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
	if err := s.addJob(&job); err != nil {
		return err
	}

	// 更新数据库中的状态为正常
	if err := s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", 0).Error; err != nil {
		s.logger.Errorf("更新任务状态失败: %v", err)
		return err
	}

	s.logger.Infof("任务已启动 [%s]", jobID)
	return nil
}

// StopJob 停止指定任务
func (s *Scheduler) StopJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.executors[jobID]; !exists {
		return fmt.Errorf("任务未在运行")
	}

	// 从调度器中移除
	if cronID, exists := s.executors[jobID]; exists {
		s.cron.Remove(cronID)
		delete(s.executors, jobID)
	}

	// 从内存中移除（保留 jobs 中的引用，以便后续可以启动）
	delete(s.executors, jobID)

	// 更新数据库中的状态为暂停（不删除任务记录）
	if err := s.db.Model(&models.Job{}).Where("id = ?", jobID).Update("status", 1).Error; err != nil {
		s.logger.Errorf("更新任务状态失败: %v", err)
		return err
	}

	s.logger.Infof("任务已暂停 [%s]", jobID)
	return nil
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

// NoticeHub 通知中心接口
// 定义为函数类型以避免循环依赖
type NoticeHub interface {
	BroadcastToUsers(userIDs []string, message interface{})
}

// GlobalNoticeHub 全局通知中心
// 在Core初始化时设置
// 遵循 Go 最佳实践：使用 sync.RWMutex 保护全局变量
var (
	GlobalNoticeHub   NoticeHub
	GlobalNoticeHubMu sync.RWMutex
)

// SetNoticeHub 设置通知中心（线程安全）
func SetNoticeHub(hub NoticeHub) {
	GlobalNoticeHubMu.Lock()
	defer GlobalNoticeHubMu.Unlock()
	GlobalNoticeHub = hub
}

// GetNoticeHub 获取通知中心（线程安全）
func GetNoticeHub() NoticeHub {
	GlobalNoticeHubMu.RLock()
	defer GlobalNoticeHubMu.RUnlock()
	return GlobalNoticeHub
}

// DBGetter 数据库获取接口
type DBGetter interface {
	GetDB() *gorm.DB
}

// GlobalDB 全局数据库访问器
// 遵循 Go 最佳实践：使用 sync.RWMutex 保护全局变量
var (
	GlobalDB   DBGetter
	GlobalDBMu sync.RWMutex
)

// SetDB 设置数据库访问器（线程安全）
func SetDB(db DBGetter) {
	GlobalDBMu.Lock()
	defer GlobalDBMu.Unlock()
	GlobalDB = db
}

// GetDB 获取数据库访问器（线程安全）
func GetDB() DBGetter {
	GlobalDBMu.RLock()
	defer GlobalDBMu.RUnlock()
	return GlobalDB
}

// GlobalScheduler 全局调度器引用
// 在Core初始化时设置，用于任务执行函数中访问调度器
// 遵循 Go 最佳实践：使用 sync.RWMutex 保护全局变量
var (
	GlobalScheduler   *Scheduler
	GlobalSchedulerMu sync.RWMutex
)

// SetGlobalScheduler 设置全局调度器（线程安全）
func SetGlobalScheduler(scheduler *Scheduler) {
	GlobalSchedulerMu.Lock()
	defer GlobalSchedulerMu.Unlock()
	GlobalScheduler = scheduler
}

// GetGlobalScheduler 获取全局调度器（线程安全）
func GetGlobalScheduler() *Scheduler {
	GlobalSchedulerMu.RLock()
	defer GlobalSchedulerMu.RUnlock()
	return GlobalScheduler
}

// ==================== 通知定时任务注册函数 ====================

// RegisterNoticeTasks 注册通知相关定时任务
func RegisterNoticeTasks(scheduler *Scheduler) {
	// 通知发布任务 - 在指定时间发布通知
	scheduler.RegisterTask("notice_publish", func(ctx context.Context, params map[string]interface{}) error {
		return executeNoticePublishTask(ctx, params)
	})

	// 值班提醒任务 - 每天早上8点提醒今日值班人员
	scheduler.RegisterTask("duty_reminder", func(ctx context.Context, params map[string]interface{}) error {
		return executeDutyReminderTask(ctx, params)
	})
}

// executeNoticePublishTask 执行通知发布任务
func executeNoticePublishTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 从参数中获取通知ID（统一使用 "param" key）
	noticeID, ok := params["param"].(string)
	if !ok || noticeID == "" {
		return fmt.Errorf("通知ID参数无效")
	}

	db := GlobalDB.GetDB()

	// 查询通知信息（包含结束时间）
	var notice struct {
		ID            string
		NoticeTitle   string
		Priority      int
		PublishStatus int
		TargetType    int
		EndDate       *string // 结束时间（RFC3339格式）
	}
	var job models.Job

	if err := db.Table("sys_notice").
		Select("id, notice_title, priority, publish_status, target_type, end_date").
		Where("id = ? AND publish_status = ?", noticeID, models.PublishStatusScheduled).
		First(&notice).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			applogger.Infof("通知不存在或已发布: %s", noticeID)
			return nil
		}
		return fmt.Errorf("查询通知失败: %w", err)
	}

	// 检查是否超过结束时间
	if shouldStopNotice(notice.EndDate, notice.NoticeTitle, noticeID, db) {
		return nil
	}

	// 查询对应的 Job，检查任务类型
	jobName := getNoticeJobName(noticeID)
	if err := db.Where("job_name = ?", jobName).First(&job).Error; err != nil {
		applogger.Warnf("未找到对应任务 [%s]", jobName)
	} else if job.MisfirePolicy == models.MisfirePolicyExecuteOnce {
		applogger.Infof("检测到一次性任务，执行完成后将删除: %s", jobName)
	}

	// 处理发布状态：一次性任务更新为已发布
	if job.MisfirePolicy == models.MisfirePolicyExecuteOnce {
		if err := db.Table("sys_notice").
			Where("id = ?", noticeID).
			Update("publish_status", models.PublishStatusPublished).Error; err != nil {
			return fmt.Errorf("更新发布状态失败: %w", err)
		}
	}

	// 处理任务生命周期：一次性任务删除，周期性任务保留
	handleTaskLifecycle(job, jobName, noticeID)

	// 通过 WebSocket 推送给目标用户
	broadcastNoticeToUsers(db, noticeID, notice.NoticeTitle, notice.Priority, notice.TargetType)

	// 发送多渠道通知（邮件、短信、API等）
	senderService := services.NewNotificationSenderService(db)
	if err := senderService.SendNotification(ctx, noticeID); err != nil {
		applogger.Warnf("发送多渠道通知失败: %v", err)
	}

	applogger.Infof("通知发布成功: %s", notice.NoticeTitle)
	return nil
}

// shouldStopNotice 检查通知是否应因结束时间而停止
func shouldStopNotice(endDate *string, title, noticeID string, db *gorm.DB) bool {
	if endDate == nil || *endDate == "" {
		return false
	}

	endTime, err := time.Parse(time.RFC3339, *endDate)
	if err != nil || !time.Now().After(endTime) {
		return false
	}

	applogger.Infof("通知 [%s] 已超过结束时间，停止任务", title)

	// 删除对应的定时任务
	jobName := getNoticeJobName(noticeID)
	if GlobalScheduler != nil {
		var job models.Job
		if err := db.Where("job_name = ?", jobName).First(&job).Error; err == nil {
			if err := GlobalScheduler.RemoveJob(job.ID); err != nil {
				applogger.Warnf("删除任务失败: %v", err)
			} else {
				applogger.Infof("周期性任务因结束时间已停止: %s", jobName)
			}
		}
	}
	return true
}

// handleTaskLifecycle 处理任务生命周期：一次性任务删除，周期性任务保留
func handleTaskLifecycle(job models.Job, jobName, noticeID string) {
	if GlobalScheduler == nil {
		return
	}

	if job.MisfirePolicy == models.MisfirePolicyExecuteOnce {
		if err := GlobalScheduler.RemoveJob(job.ID); err != nil {
			applogger.Warnf("删除一次性任务失败: %v", err)
		} else {
			applogger.Infof("一次性任务已自动删除: %s", jobName)
		}
	} else if job.ID != "" {
		applogger.Infof("周期性任务继续执行: %s", jobName)
	}
}

// broadcastNoticeToUsers 通过 WebSocket 推送通知给目标用户
func broadcastNoticeToUsers(db *gorm.DB, noticeID, title string, priority, targetType int) {
	if GlobalNoticeHub == nil {
		return
	}

	targetUserIDs, err := getTargetUserIDs(db, noticeID, targetType)
	if err != nil {
		applogger.Warnf("获取目标用户失败: %v", err)
		return
	}

	if len(targetUserIDs) == 0 {
		return
	}

	message := websocket.NoticeMessage{
		Type:      "new_notice",
		NoticeID:  noticeID,
		Title:     title,
		Priority:  priority,
		Timestamp: time.Now().Unix(),
	}
	GlobalNoticeHub.BroadcastToUsers(targetUserIDs, message)
	applogger.Infof("通知已发布并推送给 %d 个用户: %s", len(targetUserIDs), title)
}

// getTargetUserIDs 获取通知的目标用户ID列表
func getTargetUserIDs(db *gorm.DB, noticeID string, targetType int) ([]string, error) {
	var userIDs []string

	// 全部用户：从用户表获取所有未删除的用户ID
	if targetType == 0 {
		if err := db.Table("sys_user").Where("deleted_at IS NULL").Pluck("id", &userIDs).Error; err != nil {
			return nil, fmt.Errorf("查询用户列表失败: %w", err)
		}
		return userIDs, nil
	}

	// 指定用户：从通知目标表查询
	if err := db.Table("sys_notice_target").
		Where("notice_id = ?", noticeID).
		Pluck("target_id", &userIDs).Error; err != nil {
		return nil, fmt.Errorf("查询目标用户失败: %w", err)
	}

	return userIDs, nil
}

// getNoticeJobName 获取通知定时任务的名称
func getNoticeJobName(noticeID string) string {
	return fmt.Sprintf("notice_publish_%s", noticeID)
}

// executeDutyReminderTask 执行值班提醒任务
// 每天早上8点查询当日值班人员，并发送值班提醒通知
// 支持通过 sys_duty_config 表配置是否启用提醒和提醒时间
func executeDutyReminderTask(ctx context.Context, params map[string]interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	db := GlobalDB.GetDB()

	// 检查值班提醒是否启用
	enabled, err := isDutyReminderEnabled(db)
	if err != nil {
		return fmt.Errorf("查询值班配置失败: %w", err)
	}
	if !enabled {
		applogger.Infof("值班提醒已禁用，跳过执行")
		return nil
	}

	// 查询今日值班人员
	today := time.Now().Format("2006-01-02")
	members, err := getTodayDutyMembers(db, today)
	if err != nil {
		return fmt.Errorf("查询今日值班人员失败: %w", err)
	}

	// 如果没有值班人员，记录日志并返回
	if len(members) == 0 {
		applogger.Infof("今日暂无值班人员: %s", today)
		return nil
	}

	// 发送值班提醒通知
	sendDutyReminderNotification(members)

	return nil
}

// isDutyReminderEnabled 检查值班提醒是否启用
func isDutyReminderEnabled(db *gorm.DB) (bool, error) {
	type DutyConfig struct {
		ReminderEnabled bool
	}
	var config DutyConfig

	result := db.Table("sys_duty_config").
		Select("reminder_enabled").
		First(&config)

	if result.Error == nil {
		return config.ReminderEnabled, nil
	}
	if result.Error == gorm.ErrRecordNotFound {
		// 默认启用
		return true, nil
	}
	return false, result.Error
}

// dutyMember 值班成员信息
type dutyMember struct {
	UserID   string
	Username string
	NickName string
	PoolName string
}

// getTodayDutyMembers 获取今日值班人员
func getTodayDutyMembers(db *gorm.DB, date string) ([]dutyMember, error) {
	var members []dutyMember

	query := db.Table("sys_duty_schedule AS ds").
		Select("ds.user_id, u.username, u.nickname AS nick_name, dp.pool_name").
		Joins("LEFT JOIN sys_user AS u ON ds.user_id = u.id").
		Joins("LEFT JOIN sys_duty_pool AS dp ON ds.pool_id = dp.id").
		Where("ds.schedule_date = ? AND ds.status = 0", date)

	if err := query.Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

// sendDutyReminderNotification 发送值班提醒通知
func sendDutyReminderNotification(members []dutyMember) {
	if GlobalNoticeHub == nil {
		return
	}

	// 提取用户ID列表
	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}

	// 构建值班人员列表文本
	memberList := formatMemberList(members)

	message := websocket.NoticeMessage{
		Type:      "duty_reminder",
		Title:     "值班提醒",
		Content:   fmt.Sprintf("今日值班人员：%s", memberList),
		Priority:  1, // 普通优先级
		Timestamp: time.Now().Unix(),
	}
	GlobalNoticeHub.BroadcastToUsers(userIDs, message)
	applogger.Infof("值班提醒已发送给 %d 个用户: %s", len(userIDs), memberList)
}

// formatMemberList 格式化值班人员列表
func formatMemberList(members []dutyMember) string {
	if len(members) == 0 {
		return ""
	}

	result := make([]string, 0, len(members))
	for _, m := range members {
		name := m.NickName
		if name == "" {
			name = m.Username
		}
		result = append(result, name)
	}

	// 使用中文顿号连接
	return joinWithSeparator(result, "、")
}

// joinWithSeparator 使用指定分隔符连接字符串切片
func joinWithSeparator(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}

// DeviceMonitorService 设备监控服务接口
// 使用接口避免循环依赖
type DeviceMonitorService interface {
	CheckAllDevicesStatus(ctx context.Context) (int, int, error)
	CollectAllPortStatus(ctx context.Context) error
	CollectAllMACAddresses(ctx context.Context) error
	BackupAllConfigurations(ctx context.Context) error
}

// GlobalDeviceMonitorService 全局设备监控服务
// 遵循 Go 最佳实践：使用 sync.RWMutex 保护全局变量
var (
	GlobalDeviceMonitorService        DeviceMonitorService
	GlobalDeviceMonitorServiceMu      sync.RWMutex
)

// GlobalDeviceInfoCollectionService 全局设备信息采集服务
var (
	GlobalDeviceInfoCollectionService interface {
		EnqueueAllOnlineDevices(ctx context.Context) error
	}
	GlobalDeviceInfoCollectionServiceMu sync.RWMutex
)

// VDIVMService VDI虚拟机服务接口
// 使用接口避免循环依赖
type VDIVMService interface {
	SyncAllVMs(ctx context.Context, serverID string) error
	SyncVMsFromVDIByServer(ctx context.Context, server *models.VDIServer) error
}

// GlobalVDIVMService 全局VDI虚拟机服务
var (
	GlobalVDIVMService   VDIVMService
	GlobalVDIVMServiceMu sync.RWMutex
)

// SetVDIVMService 设置VDI虚拟机服务（线程安全）
func SetVDIVMService(service VDIVMService) {
	GlobalVDIVMServiceMu.Lock()
	defer GlobalVDIVMServiceMu.Unlock()
	GlobalVDIVMService = service
}

// GetVDIVMService 获取VDI虚拟机服务（线程安全）
func GetVDIVMService() VDIVMService {
	GlobalVDIVMServiceMu.RLock()
	defer GlobalVDIVMServiceMu.RUnlock()
	return GlobalVDIVMService
}

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
	svc := GetDeviceMonitorService()
	if svc == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	online, offline, err := svc.CheckAllDevicesStatus(ctx)
	if err != nil {
		return fmt.Errorf("设备状态检查失败: %w", err)
	}

	applogger.Infof("设备状态检查完成: 在线=%d, 离线=%d", online, offline)
	return nil
}

// executeDeviceInfoUpdateTask 执行设备信息更新任务
// 通过SSH异步采集设备的详细信息（型号、序列号、版本、运行时间等）
func executeDeviceInfoUpdateTask(ctx context.Context) error {
	svc := GetDeviceInfoCollectionService()
	if svc == nil {
		return fmt.Errorf("设备信息采集服务未初始化")
	}

	if err := svc.EnqueueAllOnlineDevices(ctx); err != nil {
		return fmt.Errorf("设备信息更新失败: %w", err)
	}

	applogger.Infof("设备信息更新任务已提交")
	return nil
}

// executePortCollectionTask 执行端口采集任务
// 采集所有在线设备的端口状态信息
func executePortCollectionTask(ctx context.Context) error {
	svc := GetDeviceMonitorService()
	if svc == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := svc.CollectAllPortStatus(ctx); err != nil {
		return fmt.Errorf("端口状态采集失败: %w", err)
	}

	applogger.Infof("端口状态采集完成")
	return nil
}

// executeMACCollectionTask 执行MAC采集任务
// 采集所有在线设备的MAC地址表
func executeMACCollectionTask(ctx context.Context) error {
	svc := GetDeviceMonitorService()
	if svc == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := svc.CollectAllMACAddresses(ctx); err != nil {
		return fmt.Errorf("MAC地址采集失败: %w", err)
	}

	applogger.Infof("MAC地址采集完成")
	return nil
}

// executeConfigBackupTask 执行配置备份任务
// 备份所有设备配置
func executeConfigBackupTask(ctx context.Context) error {
	svc := GetDeviceMonitorService()
	if svc == nil {
		return fmt.Errorf("设备监控服务未初始化")
	}

	if err := svc.BackupAllConfigurations(ctx); err != nil {
		return fmt.Errorf("配置备份失败: %w", err)
	}

	applogger.Infof("配置备份完成")
	return nil
}

// SetDeviceMonitorService 设置设备监控服务（线程安全）
func SetDeviceMonitorService(service DeviceMonitorService) {
	GlobalDeviceMonitorServiceMu.Lock()
	defer GlobalDeviceMonitorServiceMu.Unlock()
	GlobalDeviceMonitorService = service
}

// GetDeviceMonitorService 获取设备监控服务（线程安全）
func GetDeviceMonitorService() DeviceMonitorService {
	GlobalDeviceMonitorServiceMu.RLock()
	defer GlobalDeviceMonitorServiceMu.RUnlock()
	return GlobalDeviceMonitorService
}

// SetDeviceInfoCollectionService 设置设备信息采集服务（线程安全）
func SetDeviceInfoCollectionService(service interface {
	EnqueueAllOnlineDevices(ctx context.Context) error
}) {
	GlobalDeviceInfoCollectionServiceMu.Lock()
	defer GlobalDeviceInfoCollectionServiceMu.Unlock()
	GlobalDeviceInfoCollectionService = service
}

// GetDeviceInfoCollectionService 获取设备信息采集服务（线程安全）
func GetDeviceInfoCollectionService() interface {
	EnqueueAllOnlineDevices(ctx context.Context) error
} {
	GlobalDeviceInfoCollectionServiceMu.RLock()
	defer GlobalDeviceInfoCollectionServiceMu.RUnlock()
	return GlobalDeviceInfoCollectionService
}

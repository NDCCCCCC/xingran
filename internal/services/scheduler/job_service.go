package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/common"
	"gorm.io/gorm"
)

// SchedulerClient 调度器客户端接口
type SchedulerClient interface {
	AddJob(job *models.Job) error
	UpdateJob(job *models.Job) error
	RemoveJob(id string) error
	StartJob(id string) error
	StopJob(id string) error
	ExecuteJob(id string) error
	// F-08: IsTaskRegistered 校验给定 taskType 是否在调度器中注册,
	// 用于 Create/Update 时拒绝未注册的 InvokeTarget (白名单)。
	IsTaskRegistered(taskType string) bool
}

// JobService 定时任务服务接口
type JobService interface {
	// Create 创建定时任务
	Create(ctx context.Context, req *JobCreateRequest) (*models.Job, error)

	// Update 更新定时任务
	Update(ctx context.Context, req *JobUpdateRequest) error

	// Delete 删除定时任务
	Delete(ctx context.Context, id string) error

	// GetByID 根据ID获取定时任务
	GetByID(ctx context.Context, id string) (*models.Job, error)

	// List 查询定时任务列表
	List(ctx context.Context, params *JobListParams) (*common.PageResult, error)

	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, id string, status int) error

	// Execute 执行任务
	Execute(ctx context.Context, id string) error
}

// jobServiceImpl 定时任务服务实现
type jobServiceImpl struct {
	db        *gorm.DB
	scheduler SchedulerClient
}

// NewJobService 创建定时任务服务实例
func NewJobService(db *gorm.DB, scheduler SchedulerClient) JobService {
	return &jobServiceImpl{
		db:        db,
		scheduler: scheduler,
	}
}

// JobListParams 任务列表查询参数
type JobListParams struct {
	JobName  string
	JobGroup string
	Status   *int
	common.ListParams
}

// jobAllowedSortFields 定时任务可排序字段白名单(对应 sys_job 表列名)。
var jobAllowedSortFields = map[string]string{
	"jobName":    "job_name",
	"jobGroup":   "job_group",
	"status":     "status",
	"createdAt":  "created_at",
}

// JobCreateRequest 创建任务请求
type JobCreateRequest struct {
	JobName        string
	JobGroup       string
	InvokeTarget   string
	CronExpression string
	MisfirePolicy  models.MisfirePolicy
	Concurrent     bool
	Remark         *string
}

// JobUpdateRequest 更新任务请求
type JobUpdateRequest struct {
	ID             string
	JobName        string
	JobGroup       string
	InvokeTarget   string
	CronExpression string
	MisfirePolicy  models.MisfirePolicy
	Concurrent     bool
	Remark         *string
}

// Create 创建定时任务
func (s *jobServiceImpl) Create(ctx context.Context, req *JobCreateRequest) (*models.Job, error) {
	// 检查任务名称是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Job{}).
		Where("job_name = ? AND job_group = ?", req.JobName, req.JobGroup).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查任务名称失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("任务名称在该组中已存在")
	}

	// 验证Cron表达式
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
		return nil, fmt.Errorf("无效的Cron表达式: %w", err)
	}

	// F-08: 校验 InvokeTarget 的 taskType 必须在调度器中注册 (白名单)。
	// 拒绝任意字符串被持久化为定时任务,杜绝攻击者通过 API 写入未注册的
	// 任务名作为命令注入跳板。InvokeTarget 格式: "taskType" 或 "taskType:param"。
	if s.scheduler != nil {
		taskType := req.InvokeTarget
		if idx := strings.Index(req.InvokeTarget, ":"); idx >= 0 {
			taskType = req.InvokeTarget[:idx]
		}
		if !s.scheduler.IsTaskRegistered(taskType) {
			return nil, fmt.Errorf("InvokeTarget 任务类型未在调度器注册: %q", taskType)
		}
	}

	// 创建任务
	job := &models.Job{
		JobName:        req.JobName,
		JobGroup:       req.JobGroup,
		InvokeTarget:   req.InvokeTarget,
		CronExpression: req.CronExpression,
		MisfirePolicy:  req.MisfirePolicy,
		Concurrent:     req.Concurrent,
		Status:         models.JobStatusNormal, // 默认正常状态（0=正常, 1=暂停）
		Remark:         req.Remark,
	}

	if err := s.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("创建定时任务失败: %w", err)
	}

	// 添加任务到调度器
	if s.scheduler != nil {
		if err := s.scheduler.AddJob(job); err != nil {
			return nil, fmt.Errorf("添加任务到调度器失败: %w", err)
		}
	}

	return job, nil
}

// Update 更新定时任务
func (s *jobServiceImpl) Update(ctx context.Context, req *JobUpdateRequest) error {
	// 检查任务是否存在
	job, err := s.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}

	// 检查任务名称是否与其他任务重复
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Job{}).
		Where("job_name = ? AND job_group = ? AND id != ?", req.JobName, req.JobGroup, req.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查任务名称失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("任务名称在该组中已存在")
	}

	// 验证Cron表达式
	if _, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor).Parse(req.CronExpression); err != nil {
		return fmt.Errorf("无效的Cron表达式: %w", err)
	}

	// F-08: Update 同样校验 InvokeTarget taskType 白名单
	if s.scheduler != nil {
		taskType := req.InvokeTarget
		if idx := strings.Index(req.InvokeTarget, ":"); idx >= 0 {
			taskType = req.InvokeTarget[:idx]
		}
		if !s.scheduler.IsTaskRegistered(taskType) {
			return fmt.Errorf("InvokeTarget 任务类型未在调度器注册: %q", taskType)
		}
	}

	// 更新任务信息
	job.JobName = req.JobName
	job.JobGroup = req.JobGroup
	job.InvokeTarget = req.InvokeTarget
	job.CronExpression = req.CronExpression
	job.MisfirePolicy = req.MisfirePolicy
	job.Concurrent = req.Concurrent
	job.Remark = req.Remark

	if err := s.db.WithContext(ctx).Save(job).Error; err != nil {
		return fmt.Errorf("更新定时任务失败: %w", err)
	}

	// 更新调度器中的任务
	if s.scheduler != nil && job.Status == models.JobStatusNormal {
		if err := s.scheduler.UpdateJob(job); err != nil {
			return fmt.Errorf("更新调度器中的任务失败: %w", err)
		}
	}

	return nil
}

// Delete 删除定时任务
func (s *jobServiceImpl) Delete(ctx context.Context, id string) error {
	// 检查任务是否存在
	job, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 从调度器中移除任务
	if s.scheduler != nil {
		_ = s.scheduler.RemoveJob(id) // 忽略错误，继续删除数据库记录
	}

	// 使用事务删除数据库记录
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除任务
		if err := tx.Delete(job).Error; err != nil {
			return fmt.Errorf("删除定时任务失败: %w", err)
		}

		// 删除任务执行日志
		if err := tx.Where("job_name = ? AND job_group = ?", job.JobName, job.JobGroup).
			Delete(&models.JobLog{}).Error; err != nil {
			return fmt.Errorf("删除任务日志失败: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// GetByID 根据ID获取定时任务
func (s *jobServiceImpl) GetByID(ctx context.Context, id string) (*models.Job, error) {
	var job models.Job
	if err := s.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("定时任务不存在")
		}
		return nil, fmt.Errorf("查询定时任务失败: %w", err)
	}
	return &job, nil
}

// List 查询定时任务列表
func (s *jobServiceImpl) List(ctx context.Context, params *JobListParams) (*common.PageResult, error) {
	// 设置默认值
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// 构建查询条件
	db := s.db.WithContext(ctx).Model(&models.Job{})

	if params.JobName != "" {
		db = db.Where("job_name LIKE ?", "%"+params.JobName+"%")
	}
	if params.JobGroup != "" {
		db = db.Where("job_group = ?", params.JobGroup)
	}
	if params.Status != nil {
		db = db.Where("status = ?", *params.Status)
	}

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询定时任务总数失败: %w", err)
	}

	// 查询分页数据 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	var jobs []models.Job
	offset := (params.Current - 1) * params.PageSize
	db = base.ApplySort(db, params.BaseListRequest, jobAllowedSortFields)
	if params.OrderByColumn == "" {
		db = db.Order("created_at DESC")
	}
	if err := db.Offset(offset).Limit(params.PageSize).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("查询定时任务列表失败: %w", err)
	}

	return &common.PageResult{
		List:     jobs,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// UpdateStatus 更新任务状态
func (s *jobServiceImpl) UpdateStatus(ctx context.Context, id string, status int) error {
	job, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 更新状态
	job.Status = models.JobStatus(status)

	if err := s.db.WithContext(ctx).Save(job).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 根据状态启动或停止任务
	if s.scheduler != nil {
		if status == 0 { // 启用
			if err := s.scheduler.StartJob(id); err != nil {
				return fmt.Errorf("启动任务失败: %w", err)
			}
		} else { // 禁用
			if err := s.scheduler.StopJob(id); err != nil {
				return fmt.Errorf("停止任务失败: %w", err)
			}
		}
	}

	return nil
}

// Execute 执行任务
func (s *jobServiceImpl) Execute(ctx context.Context, id string) error {
	job, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 执行任务
	if s.scheduler != nil {
		if err := s.scheduler.ExecuteJob(id); err != nil {
			return fmt.Errorf("执行任务失败: %w", err)
		}
	} else {
		// 如果调度器未启动，记录简单日志
		startTime := time.Now()
		jobLog := models.JobLog{
			JobName:      job.JobName,
			JobGroup:     job.JobGroup,
			InvokeTarget: job.InvokeTarget,
			JobMessage:   "手动执行",
			Status:       int(models.JobLogStatusSuccess),
			StartTime:    &startTime,
		}

		endTime := time.Now()
		jobLog.EndTime = &endTime
		jobLog.Duration = endTime.Sub(startTime).Milliseconds()
		jobLog.JobMessage = "任务执行成功(调度器未启动)"

		if err := s.db.WithContext(ctx).Create(&jobLog).Error; err != nil {
			return fmt.Errorf("记录任务日志失败: %w", err)
		}
	}

	return nil
}

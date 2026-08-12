package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/common"
	"gorm.io/gorm"
)

// JobLogService 任务日志服务接口
type JobLogService interface {
	// Create 创建任务日志
	Create(ctx context.Context, log *models.JobLog) error

	// List 查询任务日志列表
	List(ctx context.Context, params *JobLogListParams) (*common.PageResult, error)

	// Statistics 统计任务日志总数及成功/失败计数(按 jobName/jobGroup 过滤)
	Statistics(ctx context.Context, params *JobLogListParams) (*JobLogStatistics, error)

	// CleanOldLogs 清理旧日志
	CleanOldLogs(ctx context.Context, days int) error
}

// JobLogStatistics 任务日志统计结果。status: 0=成功 1=失败。
type JobLogStatistics struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"` // status = 0
	Fail    int64 `json:"fail"`    // status = 1
}

// jobLogServiceImpl 任务日志服务实现
type jobLogServiceImpl struct {
	db *gorm.DB
}

// NewJobLogService 创建任务日志服务实例
func NewJobLogService(db *gorm.DB) JobLogService {
	return &jobLogServiceImpl{
		db: db,
	}
}

// JobLogListParams 任务日志列表查询参数
type JobLogListParams struct {
	JobName   string
	JobGroup  string
	Status    *int
	StartTime *string
	EndTime   *string
	common.ListParams
}

// jobLogAllowedSortFields 任务日志可排序字段白名单(对应 sys_job_log 表列名)。
var jobLogAllowedSortFields = map[string]string{
	"jobName":   "job_name",
	"jobGroup":  "job_group",
	"status":    "status",
	"createdAt": "created_at",
}

// Create 创建任务日志
func (s *jobLogServiceImpl) Create(ctx context.Context, log *models.JobLog) error {
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("创建任务日志失败: %w", err)
	}
	return nil
}

// List 查询任务日志列表
func (s *jobLogServiceImpl) List(ctx context.Context, params *JobLogListParams) (*common.PageResult, error) {
	// 设置默认值
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// 构建查询条件
	db := s.db.WithContext(ctx).Model(&models.JobLog{})

	if params.JobName != "" {
		db = db.Where("job_name LIKE ?", "%"+params.JobName+"%")
	}
	if params.JobGroup != "" {
		db = db.Where("job_group = ?", params.JobGroup)
	}
	if params.Status != nil {
		db = db.Where("status = ?", *params.Status)
	}
	if params.StartTime != nil && *params.StartTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", *params.StartTime); err == nil {
			db = db.Where("start_time >= ?", t)
		}
	}
	if params.EndTime != nil && *params.EndTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", *params.EndTime); err == nil {
			db = db.Where("start_time <= ?", t)
		}
	}

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询任务日志总数失败: %w", err)
	}

	// 查询分页数据 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	var logs []models.JobLog
	offset := (params.Current - 1) * params.PageSize
	db = base.ApplySort(db, params.BaseListRequest, jobLogAllowedSortFields)
	if params.OrderByColumn == "" {
		db = db.Order("created_at DESC")
	}
	if err := db.Offset(offset).Limit(params.PageSize).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询任务日志失败: %w", err)
	}

	return &common.PageResult{
		List:     logs,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Statistics 统计任务日志总数及成功/失败计数(按 jobName/jobGroup 过滤,与 List 口径一致)。
// 用条件聚合(SUM CASE)避免「按当前页 list(pageSize:50)算统计」——频繁任务执行 >50 次时
// 旧前端「总执行次数」卡片会卡在 50。
func (s *jobLogServiceImpl) Statistics(ctx context.Context, params *JobLogListParams) (*JobLogStatistics, error) {
	db := s.db.WithContext(ctx).Model(&models.JobLog{})
	if params.JobName != "" {
		db = db.Where("job_name LIKE ?", "%"+params.JobName+"%")
	}
	if params.JobGroup != "" {
		db = db.Where("job_group = ?", params.JobGroup)
	}

	var result JobLogStatistics
	if err := db.Select(
		"COUNT(*) AS total",
		"SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) AS success",
		"SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS fail",
	).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("统计任务日志失败: %w", err)
	}
	return &result, nil
}

// CleanOldLogs 清理旧日志
func (s *jobLogServiceImpl) CleanOldLogs(ctx context.Context, days int) error {
	if days <= 0 {
		return fmt.Errorf("天数必须大于0")
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)
	result := s.db.WithContext(ctx).Where("created_at < ?", cutoffTime).Delete(&models.JobLog{})
	if result.Error != nil {
		return fmt.Errorf("清理旧日志失败: %w", result.Error)
	}

	return nil
}

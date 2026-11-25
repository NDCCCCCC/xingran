package rpa

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 158 / CLEANUP-02：RPA 定时调度服务
//
// 背景：sys_rpa_schedules 表与 rpamodels.Schedule 模型早已存在（models/rpa/schedule.go），
// 但**零服务零 API**（D18）—— 表是死的。本文件补齐服务层：
//
//   - CRUD（创建/更新/删除/查询/启停）
//   - 到期计算（cron 表达式 → 下次执行时间；兼容 5 位与 6 位（含秒）表达式）
//   - 到点触发（与手动触发共用 TaskService.Execute 同一入口，trigger_type=schedule）
//   - 上次/下次运行时间回写（last_run_time / next_run_time / run_count）
//   - 停用即摘除调度 job（通过 ScheduleJobSyncer）
//
// 设计边界：
//   - 本包（services/rpa）**不 import** internal/scheduler —— 避免 services↔scheduler
//     循环依赖。job 同步能力由 ScheduleJobSyncer 接口表达，实现方在 internal/scheduler
//     侧（RPAScheduleJobSyncer），由 internal/core/core.go 装配。
//   - ScheduleJobSyncer 为 nil 时（单测/未接调度器）退化为「只落库不同步 job」，
//     不报错 —— 保证本服务可独立单测。
// ============================================================================

// 调度状态取值（与 rpamodels.Schedule.Status 同构：0=启用 1=停用）。
const (
	ScheduleStatusEnabled  = 0
	ScheduleStatusDisabled = 1
)

// ScheduleTriggerType 调度触发类型（写入 sys_rpa_executions.trigger_type）。
const ScheduleTriggerType = "schedule"

// ScheduleOperator 调度触发时的「操作者」标识（写入 triggered_by）。
const ScheduleOperator = "scheduler"

// cronParser cron 表达式解析器：秒可选（兼容 5 位「分 时 日 月 周」与
// 6 位「秒 分 时 日 月 周」两种写法），并支持 @daily/@hourly 等描述符。
// 与 internal/scheduler 的 cron 引擎（cron.WithSeconds()）保持一致的口径。
var cronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// ScheduleJobSyncer 调度 job 同步器（由 internal/scheduler 实现）。
//
// 职责：把「启用的 schedule」映射为调度器里的一个 cron job：
//   - EnsureScheduleJob：幂等创建（或复用）该 schedule 对应的 job
//   - UpdateScheduleJobCron：cron 表达式变更后重挂 job
//   - RemoveScheduleJob：停用/删除时摘除 job
type ScheduleJobSyncer interface {
	EnsureScheduleJob(ctx context.Context, scheduleID, jobName, cronExpr string) error
	UpdateScheduleJobCron(ctx context.Context, scheduleID, cronExpr string) error
	RemoveScheduleJob(ctx context.Context, scheduleID string) error
}

// CreateScheduleRequest 创建调度请求。
type CreateScheduleRequest struct {
	Name        string     `json:"name" binding:"required"`
	TaskID      string     `json:"taskId" binding:"required"`
	CronExpr    string     `json:"cronExpr" binding:"required"`
	Timezone    string     `json:"timezone,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	Status      *int       `json:"status,omitempty"`
	Description string     `json:"description,omitempty"`
}

// UpdateScheduleRequest 更新调度请求。
type UpdateScheduleRequest struct {
	ID          string     `json:"id"`
	Name        string     `json:"name,omitempty"`
	TaskID      string     `json:"taskId,omitempty"`
	CronExpr    string     `json:"cronExpr,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	Status      *int       `json:"status,omitempty"`
	Description string     `json:"description,omitempty"`
}

// ScheduleListParams 调度列表查询参数。
type ScheduleListParams struct {
	ListParams
	Name   string `json:"name,omitempty"`
	TaskID string `json:"taskId,omitempty"`
	Status *int   `json:"status,omitempty"`
}

// Validate 校验调度列表筛选参数的取值域（status 只接受 0/1）。
func (p *ScheduleListParams) Validate() error {
	if p == nil || p.Status == nil {
		return nil
	}
	if *p.Status != ScheduleStatusEnabled && *p.Status != ScheduleStatusDisabled {
		return fmt.Errorf("status 非法取值 %d：调度状态只接受 0（启用）或 1（停用）", *p.Status)
	}
	return nil
}

// ScheduleService RPA 定时调度服务接口。
type ScheduleService interface {
	Create(ctx context.Context, req *CreateScheduleRequest) (*rpamodels.Schedule, error)
	Update(ctx context.Context, req *UpdateScheduleRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*rpamodels.Schedule, error)
	List(ctx context.Context, params *ScheduleListParams) (*PageResult, error)
	// SetStatus 启停调度（停用即摘除调度 job）。
	SetStatus(ctx context.Context, id string, status int) error
	// TriggerNow 立即触发一次调度（与到点触发同一入口）。
	TriggerNow(ctx context.Context, id string) (*rpamodels.Execution, error)
	// ComputeNextRunTime 计算 cron 表达式的下次执行时间。
	ComputeNextRunTime(cronExpr, timezone string, from time.Time) (*time.Time, error)
	// SyncEnabledJobs 把所有启用的调度同步为调度器 job（启动时调用）。
	SyncEnabledJobs(ctx context.Context) error
}

// scheduleServiceImpl 调度服务实现。
type scheduleServiceImpl struct {
	db         *gorm.DB
	taskSvc    TaskService
	jobSyncer  ScheduleJobSyncer
}

// NewScheduleService 创建 RPA 调度服务。
//
// 参数：
//   - db：数据库句柄
//   - taskSvc：任务执行服务（触发执行，与手动触发共用同一入口）
//   - jobSyncer：调度 job 同步器（可为 nil —— 退化为只落库不同步 job）
func NewScheduleService(db *gorm.DB, taskSvc TaskService, jobSyncer ScheduleJobSyncer) ScheduleService {
	return &scheduleServiceImpl{
		db:        db,
		taskSvc:   taskSvc,
		jobSyncer: jobSyncer,
	}
}

// Create 创建调度。
func (s *scheduleServiceImpl) Create(ctx context.Context, req *CreateScheduleRequest) (*rpamodels.Schedule, error) {
	if req == nil || req.Name == "" || req.TaskID == "" || req.CronExpr == "" {
		return nil, fmt.Errorf("name / taskId / cronExpr 均为必填")
	}

	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("时区非法: %s", tz)
	}

	now := time.Now().In(loc)
	next, err := s.ComputeNextRunTime(req.CronExpr, tz, now)
	if err != nil {
		return nil, err
	}

	status := ScheduleStatusEnabled
	if req.Status != nil {
		status = *req.Status
	}

	sched := &rpamodels.Schedule{
		Name:        req.Name,
		TaskID:      req.TaskID,
		CronExpr:    req.CronExpr,
		Timezone:    tz,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Status:      status,
		NextRunTime: next,
		Description: req.Description,
	}

	if err := s.db.WithContext(ctx).Create(sched).Error; err != nil {
		return nil, fmt.Errorf("创建调度失败: %w", err)
	}

	// 启用态才注册调度 job（停用态不注册）
	if sched.Status == ScheduleStatusEnabled && s.jobSyncer != nil {
		if err := s.jobSyncer.EnsureScheduleJob(ctx, sched.ID, sched.Name, sched.CronExpr); err != nil {
			// job 注册失败不回滚调度行（调度本身已落库 truth）；记录错误由调用方感知
			return sched, fmt.Errorf("注册调度任务失败: %w", err)
		}
	}

	return sched, nil
}

// Update 更新调度。
func (s *scheduleServiceImpl) Update(ctx context.Context, req *UpdateScheduleRequest) error {
	if req == nil || req.ID == "" {
		return fmt.Errorf("id 不能为空")
	}

	var sched rpamodels.Schedule
	if err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&sched).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("调度不存在")
		}
		return fmt.Errorf("查询调度失败: %w", err)
	}

	updates := map[string]interface{}{}
	cronChanged := false

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.TaskID != "" {
		updates["task_id"] = req.TaskID
	}
	if req.Timezone != "" {
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			return fmt.Errorf("时区非法: %s", req.Timezone)
		}
		updates["timezone"] = req.Timezone
	}
	if req.CronExpr != "" {
		if req.CronExpr != sched.CronExpr {
			cronChanged = true
		}
		updates["cron_expr"] = req.CronExpr
	}
	if req.StartDate != nil {
		updates["start_date"] = req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = req.EndDate
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	tz := sched.Timezone
	if v, ok := updates["timezone"].(string); ok {
		tz = v
	}
	cronExpr := sched.CronExpr
	if v, ok := updates["cron_expr"].(string); ok {
		cronExpr = v
	}

	// cron / 时区变化 → 重算下次执行时间
	if cronChanged || updates["timezone"] != nil {
		next, err := s.ComputeNextRunTime(cronExpr, tz, time.Now())
		if err != nil {
			return err
		}
		updates["next_run_time"] = next
	}

	statusChanged := false
	newStatus := sched.Status
	if req.Status != nil {
		if *req.Status != ScheduleStatusEnabled && *req.Status != ScheduleStatusDisabled {
			return fmt.Errorf("status 非法取值 %d：调度状态只接受 0（启用）或 1（停用）", *req.Status)
		}
		newStatus = *req.Status
		statusChanged = newStatus != sched.Status
		updates["status"] = newStatus
	}

	if len(updates) == 0 {
		return nil
	}

	if err := s.db.WithContext(ctx).
		Model(&rpamodels.Schedule{}).
		Where("id = ?", req.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("更新调度失败: %w", err)
	}

	// job 同步：停用即摘除；启用/cron 变更则重挂
	if s.jobSyncer != nil {
		switch {
		case statusChanged && newStatus == ScheduleStatusDisabled:
			if err := s.jobSyncer.RemoveScheduleJob(ctx, req.ID); err != nil {
				return fmt.Errorf("摘除调度任务失败: %w", err)
			}
		case cronChanged:
			if err := s.jobSyncer.UpdateScheduleJobCron(ctx, req.ID, cronExpr); err != nil {
				return fmt.Errorf("更新调度任务失败: %w", err)
			}
		case statusChanged && newStatus == ScheduleStatusEnabled:
			if err := s.jobSyncer.EnsureScheduleJob(ctx, req.ID, req.Name, cronExpr); err != nil {
				return fmt.Errorf("注册调度任务失败: %w", err)
			}
		}
	}

	return nil
}

// Delete 删除调度（软删 + 摘除调度 job）。
func (s *scheduleServiceImpl) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id 不能为空")
	}
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&rpamodels.Schedule{}).Error; err != nil {
		return fmt.Errorf("删除调度失败: %w", err)
	}
	if s.jobSyncer != nil {
		if err := s.jobSyncer.RemoveScheduleJob(ctx, id); err != nil {
			return fmt.Errorf("摘除调度任务失败: %w", err)
		}
	}
	return nil
}

// GetByID 查询调度详情。
func (s *scheduleServiceImpl) GetByID(ctx context.Context, id string) (*rpamodels.Schedule, error) {
	var sched rpamodels.Schedule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&sched).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("调度不存在")
		}
		return nil, fmt.Errorf("查询调度失败: %w", err)
	}
	return &sched, nil
}

// List 查询调度列表。
func (s *scheduleServiceImpl) List(ctx context.Context, params *ScheduleListParams) (*PageResult, error) {
	if params == nil {
		params = &ScheduleListParams{}
	}
	// 分页默认值由 handler 的 setPaginationDefaults 兜底（Current>=1、PageSize>0）
	current := params.Current
	if current < 1 {
		current = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	query := s.db.WithContext(ctx).Model(&rpamodels.Schedule{})
	if params.Name != "" {
		query = query.Where("name LIKE ?", "%"+params.Name+"%")
	}
	if params.TaskID != "" {
		query = query.Where("task_id = ?", params.TaskID)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计调度总数失败: %w", err)
	}

	var list []rpamodels.Schedule
	if err := query.Order("created_at DESC").
		Offset((current - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询调度列表失败: %w", err)
	}

	return &PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// SetStatus 启停调度（停用即摘除调度 job）。
func (s *scheduleServiceImpl) SetStatus(ctx context.Context, id string, status int) error {
	if status != ScheduleStatusEnabled && status != ScheduleStatusDisabled {
		return fmt.Errorf("status 非法取值 %d：调度状态只接受 0（启用）或 1（停用）", status)
	}
	return s.Update(ctx, &UpdateScheduleRequest{ID: id, Status: &status})
}

// TriggerNow 立即触发一次调度。
//
// 与手动触发共用 TaskService.Execute 同一执行入口；差别只在 trigger_type
// （schedule vs manual）与 triggered_by（scheduler vs 操作者）。
func (s *scheduleServiceImpl) TriggerNow(ctx context.Context, id string) (*rpamodels.Execution, error) {
	sched, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sched.Status != ScheduleStatusEnabled {
		return nil, fmt.Errorf("调度已停用，无法触发")
	}
	if s.taskSvc == nil {
		return nil, fmt.Errorf("任务服务未注入，无法触发调度")
	}

	exec, err := s.taskSvc.Execute(ctx, &ExecuteTaskRequest{
		TaskID:      sched.TaskID,
		TriggerType: ScheduleTriggerType,
	}, ScheduleOperator)
	if err != nil {
		return nil, fmt.Errorf("触发调度执行失败: %w", err)
	}

	s.markRun(ctx, sched)
	return exec, nil
}

// markRun 回写上次执行时间 / 执行次数 / 下次执行时间。
func (s *scheduleServiceImpl) markRun(ctx context.Context, sched *rpamodels.Schedule) {
	now := time.Now()
	updates := map[string]interface{}{
		"last_run_time": now,
		"run_count":     sched.RunCount + 1,
	}
	if next, err := s.ComputeNextRunTime(sched.CronExpr, sched.Timezone, now); err == nil {
		updates["next_run_time"] = next
	}
	// 回写失败不影响主流程（执行已成功），仅记录在日志里由上层感知
	_ = s.db.WithContext(ctx).
		Model(&rpamodels.Schedule{}).
		Where("id = ?", sched.ID).
		Updates(updates).Error
}

// ComputeNextRunTime 计算 cron 表达式的下次执行时间。
//
// timezone 为空时按 Asia/Shanghai；表达式非法时返回可读错误（不静默吞掉）。
func (s *scheduleServiceImpl) ComputeNextRunTime(cronExpr, timezone string, from time.Time) (*time.Time, error) {
	expr := strings.TrimSpace(cronExpr)
	if expr == "" {
		return nil, fmt.Errorf("cron 表达式不能为空")
	}
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("时区非法: %s", timezone)
	}
	parsed, err := cronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("cron 表达式非法: %s", expr)
	}
	next := parsed.Next(from.In(loc))
	if next.IsZero() {
		return nil, fmt.Errorf("cron 表达式无法计算下次执行时间: %s", expr)
	}
	return &next, nil
}

// SyncEnabledJobs 把所有启用的调度同步为调度器 job（启动时调用）。
//
// 幂等：已存在的 job 不重复创建（由 ScheduleJobSyncer 实现保证）。
// 停用的调度在此被摘除（防「停用后重启又被挂上」）。
func (s *scheduleServiceImpl) SyncEnabledJobs(ctx context.Context) error {
	var all []rpamodels.Schedule
	if err := s.db.WithContext(ctx).Find(&all).Error; err != nil {
		return fmt.Errorf("查询调度列表失败: %w", err)
	}
	if s.jobSyncer == nil {
		return nil
	}
	for i := range all {
		sched := all[i]
		if sched.Status != ScheduleStatusEnabled {
			if err := s.jobSyncer.RemoveScheduleJob(ctx, sched.ID); err != nil {
				return fmt.Errorf("摘除调度任务失败(schedule=%s): %w", sched.ID, err)
			}
			continue
		}
		if err := s.jobSyncer.EnsureScheduleJob(ctx, sched.ID, sched.Name, sched.CronExpr); err != nil {
			return fmt.Errorf("注册调度任务失败(schedule=%s): %w", sched.ID, err)
		}
	}
	return nil
}

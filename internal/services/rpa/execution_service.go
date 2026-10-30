package rpa

import (
	"context"
	"fmt"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"gorm.io/gorm"
	"time"
)

// ExecutionService 执行记录服务接口
type ExecutionService interface {
	Create(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	UpdateProgress(ctx context.Context, id string, current, total int, message string) error
	AddLog(ctx context.Context, id string, log string) error
	Cancel(ctx context.Context, id string) error
	List(ctx context.Context, params *ExecutionListParams) (*PageResult, error)
	GetByID(ctx context.Context, id string) (*rpamodels.Execution, error)
	PublishProgress(ctx context.Context, update *ProgressUpdate) error
	// Statistics 执行记录统计(专用 COUNT 聚合,不依赖分页列表,不受 pageSize 钳制)。
	Statistics(ctx context.Context) (*ExecutionStatisticsResult, error)
}

// ExecutionStatisticsResult 执行记录统计结果。
// Status 为字符串(pending/running/success/failed/cancelled/timeout),按真实状态聚合。
type ExecutionStatisticsResult struct {
	Total   int64 `json:"total"`
	Pending int64 `json:"pending"`  // 待执行
	Running int64 `json:"running"`  // 执行中
	Success int64 `json:"success"`  // 成功
	Failed  int64 `json:"failed"`   // 失败
}

// executionServiceImpl 执行记录服务实现
type executionServiceImpl struct {
	db        *gorm.DB
	noticeHub *websocket.NoticeHub
}

// NewExecutionService 创建执行记录服务
func NewExecutionService(db *gorm.DB, noticeHub *websocket.NoticeHub) ExecutionService {
	return &executionServiceImpl{db: db, noticeHub: noticeHub}
}

// Create 创建执行记录
func (s *executionServiceImpl) Create(ctx context.Context, taskID, taskName, triggeredBy string) (*rpamodels.Execution, error) {
	execution := &rpamodels.Execution{
		TaskID:      taskID,
		TaskName:    taskName,
		Status:      string(rpamodels.RPAExecutionStatusPending),
		TriggeredBy: triggeredBy,
		TriggerType: "manual",
	}

	if err := s.db.WithContext(ctx).Create(execution).Error; err != nil {
		return nil, err
	}

	return execution, nil
}

// Update 更新执行记录
func (s *executionServiceImpl) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&rpamodels.Execution{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateProgress 更新进度
func (s *executionServiceImpl) UpdateProgress(ctx context.Context, id string, current, total int, message string) error {
	updates := map[string]interface{}{
		"progress_current": current,
		"progress_total":   total,
	}

	if message != "" {
		timestamp := time.Now().Format("15:04:05")
		logEntry := "\n[" + timestamp + "] " + message
		updates["logs"] = gorm.Expr("COALESCE(logs, '') || ?", logEntry)
	}

	return s.db.WithContext(ctx).Model(&rpamodels.Execution{}).Where("id = ?", id).Updates(updates).Error
}

// AddLog 添加日志
func (s *executionServiceImpl) AddLog(ctx context.Context, id string, log string) error {
	logEntry := "\n" + FormatLog(log)
	return s.db.WithContext(ctx).Model(&rpamodels.Execution{}).
		Where("id = ?", id).
		Update("logs", gorm.Expr("COALESCE(logs, '') || ?", logEntry)).Error
}

// Cancel 取消执行
func (s *executionServiceImpl) Cancel(ctx context.Context, id string) error {
	// 检查当前状态
	var execution rpamodels.Execution
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&execution).Error; err != nil {
		return err
	}

	status := rpamodels.RPAExecutionStatus(execution.Status)
	if status == rpamodels.RPAExecutionStatusSuccess || status == rpamodels.RPAExecutionStatusFailed || status == rpamodels.RPAExecutionStatusCancelled {
		return fmt.Errorf("执行已结束，无法取消")
	}

	updates := map[string]interface{}{
		"status":   string(rpamodels.RPAExecutionStatusCancelled),
		"end_time": time.Now(),
	}

	// 如果有开始时间，计算执行时长
	if execution.StartTime != nil {
		duration := time.Since(*execution.StartTime).Milliseconds()
		updates["duration"] = duration
	}

	return s.db.WithContext(ctx).Model(&rpamodels.Execution{}).Where("id = ?", id).Updates(updates).Error
}

// List 查询执行记录列表
func (s *executionServiceImpl) List(ctx context.Context, params *ExecutionListParams) (*PageResult, error) {
	var executions []rpamodels.Execution
	var total int64

	query := s.db.WithContext(ctx).Model(&rpamodels.Execution{}).Where("deleted_at IS NULL")

	if params.TaskID != "" {
		query = query.Where("task_id = ?", params.TaskID)
	}
	if params.WorkerID != "" {
		query = query.Where("worker_id = ?", params.WorkerID)
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, executionAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&executions).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     executions,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Statistics 统计执行记录(按 status 字符串聚合,排除软删除)。
// 不依赖分页列表,避免「用当前页 length 充当总数」导致统计恒 ≤ pageSize。
func (s *executionServiceImpl) Statistics(ctx context.Context) (*ExecutionStatisticsResult, error) {
	var result ExecutionStatisticsResult
	err := s.db.WithContext(ctx).
		Model(&rpamodels.Execution{}).
		Where("deleted_at IS NULL").
		Select(
			"COUNT(*) AS total",
			"COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending",
			"COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0) AS running",
			"COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) AS success",
			"COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed",
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByID 获取执行记录详情
func (s *executionServiceImpl) GetByID(ctx context.Context, id string) (*rpamodels.Execution, error) {
	var execution rpamodels.Execution
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// PublishProgress 发布进度更新到 WebSocket
func (s *executionServiceImpl) PublishProgress(ctx context.Context, update *ProgressUpdate) error {
	if s.noticeHub == nil {
		return nil // 如果没有配置 NoticeHub，静默跳过
	}

	// 确定消息类型
	messageType := websocket.MessageTypeRPAProgress
	switch update.Status {
	case "success", "completed":
		messageType = websocket.MessageTypeRPACompleted
	case "failed", "error":
		messageType = websocket.MessageTypeRPAFailed
	}

	// 构建 WebSocket 消息
	wsMessage := websocket.RPAProgressMessage{
		Type:        messageType,
		ExecutionID: update.ExecutionID,
		TaskID:      update.TaskID,
		TaskName:    update.TaskName,
		Step:        update.Step,
		Total:       update.Total,
		Message:     update.Message,
		Status:      update.Status,
		Timestamp:   time.Now().Unix(),
	}

	// 向触发人推送进度
	if update.TriggeredBy != "" {
		s.noticeHub.BroadcastRPAProgressToUser(update.TriggeredBy, wsMessage)
	} else {
		// 如果没有指定触发人，广播给所有用户
		s.noticeHub.BroadcastRPAProgress(wsMessage)
	}

	return nil
}

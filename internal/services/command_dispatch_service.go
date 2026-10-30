package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// CommandDispatchService 命令分发服务
type CommandDispatchService struct {
	db       *gorm.DB
	executor *device.DeviceExecutor
}

// NewCommandDispatchService 创建命令分发服务
func NewCommandDispatchService(db *gorm.DB, executor *device.DeviceExecutor) *CommandDispatchService {
	return &CommandDispatchService{
		db:       db,
		executor: executor,
	}
}

// GetStatistics 统计命令执行的总数及各状态计数(execution_type='command',与 GetExecutionList 口径一致)。
// 状态机由 models.ExecutionStatus 定义。
func (s *CommandDispatchService) GetStatistics(ctx context.Context) (*ExecutionStatistics, error) {
	var result ExecutionStatistics
	err := s.db.WithContext(ctx).Model(&models.ConfigExecution{}).
		Where("execution_type = ?", models.ExecutionTypeCommand).
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS pending", int(models.ExecutionStatusPending)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS running", int(models.ExecutionStatusRunning)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS success", int(models.ExecutionStatusSuccess)),
			fmt.Sprintf("SUM(CASE WHEN status = %d THEN 1 ELSE 0 END) AS failed", int(models.ExecutionStatusFailed)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计命令执行失败: %w", err)
	}
	return &result, nil
}

// DispatchRequest 分发请求
type DispatchRequest struct {
	ExecutionName     string
	DeviceIDs         []string
	CommandContent    string
	ExecutionStrategy models.ExecutionStrategy
	Concurrency       int
	Timeout           int
	CreatedBy         string
}

// DispatchResult 分发结果
type DispatchResult struct {
	ExecutionID string
	Results     map[string]*DeviceCommandResult
	Summary     *CommandExecutionSummary
}

// DeviceCommandResult 设备命令执行结果
type DeviceCommandResult struct {
	DeviceID       string
	DeviceName     string
	IPAddress      string
	Status         models.ExecutionStatus
	CommandSent    string
	OutputReceived string
	ErrorMessage   string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Duration       int
}

// CommandExecutionSummary 命令执行汇总
type CommandExecutionSummary struct {
	TotalDevices int
	SuccessCount int
	FailureCount int
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// Dispatch 分发命令到设备
func (s *CommandDispatchService) Dispatch(ctx context.Context, req *DispatchRequest) (*DispatchResult, error) {
	// 验证设备ID
	if len(req.DeviceIDs) == 0 {
		return nil, fmt.Errorf("请选择要执行命令的设备")
	}

	// 查询设备信息
	var devices []models.NetworkDevice
	if err := s.db.Where("id IN ?", req.DeviceIDs).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if len(devices) != len(req.DeviceIDs) {
		return nil, fmt.Errorf("部分设备不存在")
	}

	// 创建执行记录
	execution := &models.ConfigExecution{
		ExecutionName:     req.ExecutionName,
		ExecutionType:     models.ExecutionTypeCommand,
		DeviceIDs:         models.DeviceIDList(req.DeviceIDs),
		Status:            models.ExecutionStatusPending,
		TotalDevices:      len(devices),
		ExecutionStrategy: req.ExecutionStrategy,
		Concurrency:       req.Concurrency,
		Timeout:           req.Timeout,
		CommandContent:    req.CommandContent,
		CreatedBy:         req.CreatedBy,
	}

	if err := s.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 更新状态为执行中
	now := time.Now()
	execution.Status = models.ExecutionStatusRunning
	execution.StartedAt = &now
	s.db.Save(execution)

	// 执行命令
	results, summary := s.executeCommand(ctx, execution.ID, devices, req)

	// 更新执行记录
	execution.Status = models.ExecutionStatusSuccess
	execution.SuccessCount = summary.SuccessCount
	execution.FailureCount = summary.FailureCount
	execution.CompletedAt = summary.CompletedAt
	s.db.Save(execution)

	return &DispatchResult{
		ExecutionID: execution.ID,
		Results:     results,
		Summary:     summary,
	}, nil
}

// executeCommand 执行命令
func (s *CommandDispatchService) executeCommand(ctx context.Context, executionID string, devices []models.NetworkDevice, req *DispatchRequest) (map[string]*DeviceCommandResult, *CommandExecutionSummary) {
	results := make(map[string]*DeviceCommandResult)
	var mu sync.Mutex
	var summary CommandExecutionSummary

	summary.TotalDevices = len(devices)
	startTime := time.Now()
	summary.StartedAt = &startTime

	if req.ExecutionStrategy == models.ExecutionStrategyParallel {
		// 并行执行
		// 使用errgroup进行并发控制
		g, execCtx := errgroup.WithContext(ctx)
		g.SetLimit(req.Concurrency)

		for _, dev := range devices {
			dev := dev // 创建局部变量
			g.Go(func() error {
				result := s.executeOnDevice(execCtx, executionID, &dev, req.CommandContent, req.Timeout)

				mu.Lock()
				results[dev.ID] = result
				if result.Status == models.ExecutionStatusSuccess {
					summary.SuccessCount++
				} else {
					summary.FailureCount++
				}
				mu.Unlock()

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			// 记录错误但不中断
			_ = err
		}

	} else {
		// 串行执行
		for _, dev := range devices {
			result := s.executeOnDevice(ctx, executionID, &dev, req.CommandContent, req.Timeout)
			results[dev.ID] = result
			if result.Status == models.ExecutionStatusSuccess {
				summary.SuccessCount++
			} else {
				summary.FailureCount++
			}
		}
	}

	endTime := time.Now()
	summary.CompletedAt = &endTime

	return results, &summary
}

// executeOnDevice 在单个设备上执行命令
func (s *CommandDispatchService) executeOnDevice(ctx context.Context, executionID string, device *models.NetworkDevice, command string, timeout int) *DeviceCommandResult {
	startTime := time.Now()
	result := &DeviceCommandResult{
		DeviceID:   device.ID,
		DeviceName: device.DeviceName,
		IPAddress:  device.IPAddress,
		Status:     models.ExecutionStatusPending,
		StartedAt:  &startTime,
	}

	// 创建执行明细记录
	detail := &models.ConfigExecutionDetail{
		ExecutionID: executionID,
		DeviceID:    device.ID,
		DeviceName:  device.DeviceName,
		IPAddress:   device.IPAddress,
		Status:      models.ExecutionStatusPending,
		CommandSent: command,
		StartedAt:   &startTime,
	}
	s.db.Create(detail)

	// 设置超时上下文
	execCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// 使用 DeviceExecutor 执行命令
	output, err := s.executor.ExecuteOnDevice(execCtx, device.ID, command, true)
	if err != nil {
		result.Status = models.ExecutionStatusFailed
		result.ErrorMessage = err.Error()
		detail.Status = models.ExecutionStatusFailed
		detail.ErrorMessage = result.ErrorMessage
		s.db.Save(detail)
		return result
	}

	// 更新结果
	result.Status = models.ExecutionStatusSuccess
	result.CommandSent = command
	result.OutputReceived = output
	endTime := time.Now()
	result.CompletedAt = &endTime
	result.Duration = int(endTime.Sub(startTime).Milliseconds())

	detail.Status = models.ExecutionStatusSuccess
	detail.OutputReceived = output
	detail.CompletedAt = &endTime
	detail.Duration = result.Duration
	s.db.Save(detail)

	return result
}

// GetExecutionResult 获取执行结果
func (s *CommandDispatchService) GetExecutionResult(ctx context.Context, executionID string) (*DispatchResult, error) {
	var execution models.ConfigExecution
	if err := s.db.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	// 获取执行明细
	var details []models.ConfigExecutionDetail
	if err := s.db.Where("execution_id = ?", executionID).Find(&details).Error; err != nil {
		return nil, fmt.Errorf("查询执行明细失败: %w", err)
	}

	// 构建结果
	results := make(map[string]*DeviceCommandResult)
	for _, detail := range details {
		results[detail.DeviceID] = &DeviceCommandResult{
			DeviceID:       detail.DeviceID,
			DeviceName:     detail.DeviceName,
			IPAddress:      detail.IPAddress,
			Status:         detail.Status,
			CommandSent:    detail.CommandSent,
			OutputReceived: detail.OutputReceived,
			ErrorMessage:   detail.ErrorMessage,
			StartedAt:      detail.StartedAt,
			CompletedAt:    detail.CompletedAt,
			Duration:       detail.Duration,
		}
	}

	summary := &CommandExecutionSummary{
		TotalDevices: execution.TotalDevices,
		SuccessCount: execution.SuccessCount,
		FailureCount: execution.FailureCount,
		StartedAt:    execution.StartedAt,
		CompletedAt:  execution.CompletedAt,
	}

	return &DispatchResult{
		ExecutionID: executionID,
		Results:     results,
		Summary:     summary,
	}, nil
}

// QuickCommand 快速命令执行（单设备）
func (s *CommandDispatchService) QuickCommand(ctx context.Context, deviceID, command string, timeout int) (*DeviceCommandResult, error) {
	// 查询设备
	var device models.NetworkDevice
	if err := s.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	result := s.executeOnDevice(ctx, "", &device, command, timeout)

	return result, nil
}

// GetExecutionList 获取执行列表
// orderByColumn/isAsc 为服务端排序参数(可选,透传给 base.ApplySort 白名单)。
func (s *CommandDispatchService) GetExecutionList(ctx context.Context, current, pageSize int, orderByColumn string, isAsc *bool) ([]models.ConfigExecution, int64, error) {
	var executions []models.ConfigExecution
	var total int64

	query := s.db.Model(&models.ConfigExecution{}).Where("execution_type = ?", models.ExecutionTypeCommand)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询执行记录总数失败: %w", err)
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (current - 1) * pageSize
	sortReq := base.BaseListRequest{
		Current:       current,
		PageSize:      pageSize,
		OrderByColumn: orderByColumn,
		IsAsc:         isAsc,
	}
	query = base.ApplySort(query, sortReq, commandExecutionAllowedSortFields)
	if orderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(pageSize).Find(&executions).Error; err != nil {
		return nil, 0, fmt.Errorf("查询执行记录失败: %w", err)
	}

	return executions, total, nil
}

// commandExecutionAllowedSortFields 命令执行记录可排序字段白名单(对应 sys_config_execution 表列名)。
var commandExecutionAllowedSortFields = map[string]string{
	"deviceId":  "device_id",
	"commandId": "command_id",
	"status":    "status",
	"startTime": "start_time",
	"endTime":   "end_time",
	"createdAt": "created_at",
}

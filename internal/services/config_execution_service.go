package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// ConfigExecutionService 配置执行服务
type ConfigExecutionService struct {
	db       *gorm.DB
	executor *device.DeviceExecutor
}

// NewConfigExecutionService 创建配置执行服务
func NewConfigExecutionService(db *gorm.DB, executor *device.DeviceExecutor) *ConfigExecutionService {
	return &ConfigExecutionService{
		db:       db,
		executor: executor,
	}
}

// TemplateExecutionRequest 模板执行请求
type TemplateExecutionRequest struct {
	ExecutionName     string
	TemplateID        string
	DeviceIDs         []string
	TemplateVariables map[string]string
	ExecutionStrategy models.ExecutionStrategy
	Concurrency       int
	Timeout           int
	CreatedBy         string
}

// TemplateExecutionResult 模板执行结果
type TemplateExecutionResult struct {
	ExecutionID string
	Results     map[string]*DeviceExecutionResult
	Summary     *ExecutionSummary
}

// DeviceExecutionResult 设备执行结果
type DeviceExecutionResult struct {
	DeviceID       string
	DeviceName     string
	IPAddress      string
	Status         models.ExecutionStatus
	ConfigSent     string
	OutputReceived string
	ErrorMessage   string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Duration       int
}

// ExecutionSummary 执行汇总
type ExecutionSummary struct {
	TotalDevices int
	SuccessCount int
	FailureCount int
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// ExecuteByTemplate 通过模板执行配置
func (s *ConfigExecutionService) ExecuteByTemplate(ctx context.Context, req *TemplateExecutionRequest) (*TemplateExecutionResult, error) {
	// 验证设备ID
	if len(req.DeviceIDs) == 0 {
		return nil, fmt.Errorf("请选择要执行配置的设备")
	}

	// 查询模板
	var template models.ConfigTemplate
	if err := s.db.WithContext(ctx).Where("id = ?", req.TemplateID).First(&template).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}

	// 查询设备信息
	var devices []models.NetworkDevice
	if err := s.db.Where("id IN ?", req.DeviceIDs).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if len(devices) != len(req.DeviceIDs) {
		return nil, fmt.Errorf("部分设备不存在")
	}

	// 渲染模板配置
	templateSvc := NewTemplateService(s.db)
	renderedConfig, err := templateSvc.Render(ctx, template.TemplateCode, req.TemplateVariables)
	if err != nil {
		return nil, fmt.Errorf("渲染模板失败: %w", err)
	}

	// 创建执行记录
	execution := &models.ConfigExecution{
		ExecutionName:     req.ExecutionName,
		ExecutionType:     models.ExecutionTypeTemplate,
		TemplateID:        &template.ID,
		DeviceIDs:         models.DeviceIDList(req.DeviceIDs),
		Status:            models.ExecutionStatusPending,
		TotalDevices:      len(devices),
		ExecutionStrategy: req.ExecutionStrategy,
		Concurrency:       req.Concurrency,
		Timeout:           req.Timeout,
		CommandContent:    renderedConfig,
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

	// 执行配置
	results, summary := s.executeConfig(ctx, execution.ID, devices, renderedConfig, req)

	// 更新执行记录
	execution.Status = models.ExecutionStatusSuccess
	execution.SuccessCount = summary.SuccessCount
	execution.FailureCount = summary.FailureCount
	execution.CompletedAt = summary.CompletedAt
	s.db.Save(execution)

	return &TemplateExecutionResult{
		ExecutionID: execution.ID,
		Results:     results,
		Summary:     summary,
	}, nil
}

// executeConfig 执行配置
func (s *ConfigExecutionService) executeConfig(ctx context.Context, executionID string, devices []models.NetworkDevice, config string, req *TemplateExecutionRequest) (map[string]*DeviceExecutionResult, *ExecutionSummary) {
	results := make(map[string]*DeviceExecutionResult)
	var mu sync.Mutex
	var summary ExecutionSummary

	summary.TotalDevices = len(devices)
	startTime := time.Now()
	summary.StartedAt = &startTime

	if req.ExecutionStrategy == models.ExecutionStrategyParallel {
		// 并行执行 - 使用errgroup进行并发控制
		g, execCtx := errgroup.WithContext(ctx)
		g.SetLimit(req.Concurrency)

		for _, dev := range devices {
			dev := dev // 创建局部变量
			g.Go(func() error {
				result := s.executeConfigOnDevice(execCtx, executionID, &dev, config, req.Timeout)

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
			// 如果有任务返回错误，记录日志但不中断其他任务
			// 这里所有任务都会执行完成，因为上面都返回nil
			_ = err
		}

	} else {
		// 串行执行
		for _, dev := range devices {
			result := s.executeConfigOnDevice(ctx, executionID, &dev, config, req.Timeout)
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

// executeConfigOnDevice 在单个设备上执行配置
func (s *ConfigExecutionService) executeConfigOnDevice(ctx context.Context, executionID string, device *models.NetworkDevice, config string, timeout int) *DeviceExecutionResult {
	startTime := time.Now()
	result := &DeviceExecutionResult{
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
		CommandSent: config,
		StartedAt:   &startTime,
	}
	s.db.Create(detail)

	// 设置超时上下文
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// 通过连接池直接执行配置发送
	pool := s.executor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		result.Status = models.ExecutionStatusFailed
		result.ErrorMessage = fmt.Sprintf("获取连接失败: %v", err)
		detail.Status = models.ExecutionStatusFailed
		detail.ErrorMessage = result.ErrorMessage
		s.db.Save(detail)
		return result
	}

	if !conn.Acquire() {
		result.Status = models.ExecutionStatusFailed
		result.ErrorMessage = "无法获取连接锁"
		detail.Status = models.ExecutionStatusFailed
		detail.ErrorMessage = result.ErrorMessage
		s.db.Save(detail)
		return result
	}
	defer conn.Release()

	wrapper := conn.GetWrapper()
	response, err := wrapper.SendConfig(config)
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
	result.ConfigSent = config
	result.OutputReceived = response.Result
	endTime := time.Now()
	result.CompletedAt = &endTime
	result.Duration = int(endTime.Sub(startTime).Milliseconds())

	detail.Status = models.ExecutionStatusSuccess
	detail.OutputReceived = response.Result
	detail.CompletedAt = &endTime
	detail.Duration = result.Duration
	s.db.Save(detail)

	return result
}

// GetExecutionResult 获取执行结果
func (s *ConfigExecutionService) GetExecutionResult(ctx context.Context, executionID string) (*TemplateExecutionResult, error) {
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
	results := make(map[string]*DeviceExecutionResult)
	for _, detail := range details {
		results[detail.DeviceID] = &DeviceExecutionResult{
			DeviceID:       detail.DeviceID,
			DeviceName:     detail.DeviceName,
			IPAddress:      detail.IPAddress,
			Status:         detail.Status,
			ConfigSent:     detail.CommandSent,
			OutputReceived: detail.OutputReceived,
			ErrorMessage:   detail.ErrorMessage,
			StartedAt:      detail.StartedAt,
			CompletedAt:    detail.CompletedAt,
			Duration:       detail.Duration,
		}
	}

	summary := &ExecutionSummary{
		TotalDevices: execution.TotalDevices,
		SuccessCount: execution.SuccessCount,
		FailureCount: execution.FailureCount,
		StartedAt:    execution.StartedAt,
		CompletedAt:  execution.CompletedAt,
	}

	return &TemplateExecutionResult{
		ExecutionID: executionID,
		Results:     results,
		Summary:     summary,
	}, nil
}

// GetExecutionList 获取执行列表
func (s *ConfigExecutionService) GetExecutionList(ctx context.Context, current, pageSize int) ([]models.ConfigExecution, int64, error) {
	var executions []models.ConfigExecution
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ConfigExecution{}).Where("execution_type = ?", models.ExecutionTypeTemplate)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询执行记录总数失败: %w", err)
	}

	// 分页查询
	offset := (current - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&executions).Error; err != nil {
		return nil, 0, fmt.Errorf("查询执行记录失败: %w", err)
	}

	return executions, total, nil
}

// CancelExecution 取消执行任务
func (s *ConfigExecutionService) CancelExecution(ctx context.Context, executionID string) error {
	var execution models.ConfigExecution
	if err := s.db.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return fmt.Errorf("执行记录不存在: %w", err)
	}

	if execution.Status != models.ExecutionStatusPending && execution.Status != models.ExecutionStatusRunning {
		return fmt.Errorf("只能取消待执行或执行中的任务")
	}

	execution.Status = models.ExecutionStatusFailed
	execution.ErrorMessage = "用户取消执行"
	if err := s.db.WithContext(ctx).Save(&execution).Error; err != nil {
		return fmt.Errorf("取消任务失败: %w", err)
	}

	return nil
}

// DeleteExecution 删除执行记录
func (s *ConfigExecutionService) DeleteExecution(ctx context.Context, executionID string) error {
	var execution models.ConfigExecution
	if err := s.db.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return fmt.Errorf("执行记录不存在: %w", err)
	}

	// 检查任务状态
	if execution.Status == models.ExecutionStatusRunning {
		return fmt.Errorf("无法删除执行中的任务")
	}

	// 删除执行明细
	if err := s.db.Where("execution_id = ?", executionID).Delete(&models.ConfigExecutionDetail{}).Error; err != nil {
		return fmt.Errorf("删除执行明细失败: %w", err)
	}

	// 删除执行记录
	if err := s.db.WithContext(ctx).Delete(&execution).Error; err != nil {
		return fmt.Errorf("删除执行记录失败: %w", err)
	}

	return nil
}

// BatchDeleteExecutions 批量删除执行记录
func (s *ConfigExecutionService) BatchDeleteExecutions(ctx context.Context, executionIDs []string) error {
	for _, executionID := range executionIDs {
		if err := s.DeleteExecution(ctx, executionID); err != nil {
			continue // 继续处理其他任务
		}
	}
	return nil
}

package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"gorm.io/gorm"
)

// ErrorHandlingStrategy 错误处理策略
type ErrorHandlingStrategy string

const (
	ErrorStrategyIgnore   ErrorHandlingStrategy = "ignore"   // 忽略错误继续执行
	ErrorStrategyRetry    ErrorHandlingStrategy = "retry"    // 重试
	ErrorStrategyRollback ErrorHandlingStrategy = "rollback" // 回滚
	ErrorStrategySkip     ErrorHandlingStrategy = "skip"     // 跳过当前步骤
	ErrorStrategyAbort    ErrorHandlingStrategy = "abort"    // 中止执行
	ErrorStrategyFallback ErrorHandlingStrategy = "fallback" // 降级执行
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxAttempts  int           `json:"maxAttempts"`  // 最大重试次数
	InitialDelay time.Duration `json:"initialDelay"` // 初始延迟
	MaxDelay     time.Duration `json:"maxDelay"`     // 最大延迟
	BackoffType  BackoffType   `json:"backoffType"`  // 退避类型
	RetryOn      []string      `json:"retryOn"`      // 重试的错误类型
}

// BackoffType 退避类型
type BackoffType string

const (
	BackoffTypeFixed       BackoffType = "fixed"       // 固定延迟
	BackoffTypeLinear      BackoffType = "linear"      // 线性增长
	BackoffTypeExponential BackoffType = "exponential" // 指数增长
)

// ErrorHandlingConfig 错误处理配置
type ErrorHandlingConfig struct {
	Strategy        ErrorHandlingStrategy `json:"strategy"`
	RetryPolicy     *RetryPolicy          `json:"retryPolicy,omitempty"`
	FallbackAction  *json.RawMessage      `json:"fallbackAction,omitempty"`
	ContinueOnError bool                  `json:"continueOnError"`
}

// ErrorRecoveryAction 错误恢复动作
type ErrorRecoveryAction struct {
	Type       string              `json:"type"`       // recover, notify, compensate
	Actions    []json.RawMessage   `json:"actions"`    // 恢复动作列表
	Notify     *NotificationConfig `json:"notify"`     // 通知配置
	Compensate *CompensationAction `json:"compensate"` // 补偿动作
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Channels   []string `json:"channels"`   // 通知渠道
	Recipients []string `json:"recipients"` // 接收人
	Template   string   `json:"template"`   // 消息模板
	Severity   string   `json:"severity"`   // 严重程度
}

// CompensationAction 补偿动作
type CompensationAction struct {
	TransactionID string                 `json:"transactionId"`
	Actions       []json.RawMessage      `json:"actions"`
	Variables     map[string]interface{} `json:"variables"`
}

// ErrorHandlingService 错误处理服务
type ErrorHandlingService interface {
	// HandleError 处理错误
	HandleError(ctx context.Context, req *ErrorHandleRequest) (*ErrorHandleResult, error)

	// ExecuteRetry 执行重试
	ExecuteRetry(ctx context.Context, req *RetryRequest) (*RetryResult, error)

	// ExecuteRollback 执行回滚
	ExecuteRollback(ctx context.Context, req *RollbackRequest) error

	// ExecuteFallback 执行降级
	ExecuteFallback(ctx context.Context, req *FallbackRequest) (*FallbackResult, error)
}

// errorHandlingServiceImpl 错误处理服务实现
type errorHandlingServiceImpl struct {
	db               *gorm.DB
	flowService      FlowControlService
	executionService ExecutionService
}

// NewErrorHandlingService 创建错误处理服务
func NewErrorHandlingService(db *gorm.DB, flowService FlowControlService, execService ExecutionService) ErrorHandlingService {
	return &errorHandlingServiceImpl{
		db:               db,
		flowService:      flowService,
		executionService: execService,
	}
}

// ErrorHandleRequest 错误处理请求
type ErrorHandleRequest struct {
	ExecutionID string                 `json:"executionId"`
	StepIndex   int                    `json:"stepIndex"`
	Error       error                  `json:"error"`
	Config      *ErrorHandlingConfig   `json:"config"`
	Variables   map[string]interface{} `json:"variables"`
	Context     map[string]interface{} `json:"context"`
}

// ErrorHandleResult 错误处理结果
type ErrorHandleResult struct {
	Handled         bool                   `json:"handled"`
	Action          string                 `json:"action"`
	NextStepIndex   *int                   `json:"nextStepIndex,omitempty"`
	Variables       map[string]interface{} `json:"variables"`
	RecoveryActions []json.RawMessage      `json:"recoveryActions,omitempty"`
}

// HandleError 处理错误
func (s *errorHandlingServiceImpl) HandleError(ctx context.Context, req *ErrorHandleRequest) (*ErrorHandleResult, error) {
	result := &ErrorHandleResult{
		Handled:   false,
		Action:    "abort",
		Variables: req.Variables,
	}

	switch req.Config.Strategy {
	case ErrorStrategyIgnore:
		result.Handled = true
		result.Action = "continue"
		result.NextStepIndex = intPtr(req.StepIndex + 1)

	case ErrorStrategySkip:
		result.Handled = true
		result.Action = "skip"
		result.NextStepIndex = intPtr(req.StepIndex + 1)

	case ErrorStrategyAbort:
		result.Handled = true
		result.Action = "abort"
		// 更新执行状态为失败
		s.markExecutionFailed(ctx, req.ExecutionID, req.Error)

	case ErrorStrategyRetry:
		retryResult, err := s.ExecuteRetry(ctx, &RetryRequest{
			ExecutionID: req.ExecutionID,
			StepIndex:   req.StepIndex,
			Policy:      req.Config.RetryPolicy,
			Variables:   req.Variables,
			LastError:   req.Error,
		})
		if err != nil {
			return nil, err
		}
		result.Handled = retryResult.Success
		result.Action = "retry"
		result.Variables = retryResult.Variables

	case ErrorStrategyRollback:
		if err := s.ExecuteRollback(ctx, &RollbackRequest{
			ExecutionID: req.ExecutionID,
			Variables:   req.Variables,
		}); err != nil {
			return nil, err
		}
		result.Handled = true
		result.Action = "rollback"
		s.markExecutionFailed(ctx, req.ExecutionID, req.Error)

	case ErrorStrategyFallback:
		fallbackResult, err := s.ExecuteFallback(ctx, &FallbackRequest{
			ExecutionID:    req.ExecutionID,
			StepIndex:      req.StepIndex,
			FallbackAction: req.Config.FallbackAction,
			Variables:      req.Variables,
			Error:          req.Error,
		})
		if err != nil {
			return nil, err
		}
		result.Handled = true
		result.Action = "fallback"
		result.Variables = fallbackResult.Variables
		result.RecoveryActions = fallbackResult.Actions

	default:
		result.Handled = false
	}

	return result, nil
}

// RetryRequest 重试请求
type RetryRequest struct {
	ExecutionID string                 `json:"executionId"`
	StepIndex   int                    `json:"stepIndex"`
	Policy      *RetryPolicy           `json:"policy"`
	Variables   map[string]interface{} `json:"variables"`
	LastError   error                  `json:"lastError"`
	Attempt     int                    `json:"attempt"`
}

// RetryResult 重试结果
type RetryResult struct {
	Success     bool                   `json:"success"`
	Attempt     int                    `json:"attempt"`
	Variables   map[string]interface{} `json:"variables"`
	ShouldRetry bool                   `json:"shouldRetry"`
}

// ExecuteRetry 执行重试
func (s *errorHandlingServiceImpl) ExecuteRetry(ctx context.Context, req *RetryRequest) (*RetryResult, error) {
	result := &RetryResult{
		Success:   false,
		Attempt:   req.Attempt + 1,
		Variables: req.Variables,
	}

	if req.Policy == nil {
		return result, fmt.Errorf("重试策略未配置")
	}

	// 检查是否超过最大重试次数
	if result.Attempt > req.Policy.MaxAttempts {
		result.ShouldRetry = false
		return result, fmt.Errorf("超过最大重试次数: %d", req.Policy.MaxAttempts)
	}

	// 检查错误类型是否在重试列表中
	if len(req.Policy.RetryOn) > 0 {
		shouldRetry := false
		errorType := fmt.Sprintf("%T", req.LastError)
		for _, retryType := range req.Policy.RetryOn {
			if contains(errorType, retryType) {
				shouldRetry = true
				break
			}
		}
		if !shouldRetry {
			result.ShouldRetry = false
			return result, nil
		}
	}

	// 计算延迟时间
	delay := s.calculateDelay(result.Attempt, req.Policy)

	// 记录重试日志
	s.logRetry(ctx, req.ExecutionID, result.Attempt, delay, req.LastError)

	// 等待延迟时间
	select {
	case <-time.After(delay):
		// 继续执行
	case <-ctx.Done():
		return result, ctx.Err()
	}

	result.ShouldRetry = true
	return result, nil
}

// calculateDelay 计算延迟时间
func (s *errorHandlingServiceImpl) calculateDelay(attempt int, policy *RetryPolicy) time.Duration {
	delay := policy.InitialDelay

	switch policy.BackoffType {
	case BackoffTypeFixed:
		delay = policy.InitialDelay

	case BackoffTypeLinear:
		delay = policy.InitialDelay * time.Duration(attempt)

	case BackoffTypeExponential:
		delay = policy.InitialDelay * time.Duration(1<<uint(attempt-1))
	}

	// 不超过最大延迟
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	return delay
}

// RollbackRequest 回滚请求
type RollbackRequest struct {
	ExecutionID string                 `json:"executionId"`
	Variables   map[string]interface{} `json:"variables"`
	Reason      string                 `json:"reason"`
}

// ExecuteRollback 执行回滚
func (s *errorHandlingServiceImpl) ExecuteRollback(ctx context.Context, req *RollbackRequest) error {
	// 获取执行记录
	var execution rpamodels.Execution
	if err := s.db.WithContext(ctx).Where("id = ?", req.ExecutionID).First(&execution).Error; err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 记录回滚日志
	logEntry := FormatLog(fmt.Sprintf("开始回滚: %s", req.Reason))
	s.db.WithContext(ctx).Model(&execution).
		Update("logs", gorm.Expr("COALESCE(logs, '') || ?", "\n"+logEntry))

	// TODO: 实现实际的回滚逻辑
	// 1. 获取已完成的步骤
	// 2. 按相反顺序执行补偿动作
	// 3. 更新执行状态

	return nil
}

// FallbackRequest 降级请求
type FallbackRequest struct {
	ExecutionID    string                 `json:"executionId"`
	StepIndex      int                    `json:"stepIndex"`
	FallbackAction *json.RawMessage       `json:"fallbackAction"`
	Variables      map[string]interface{} `json:"variables"`
	Error          error                  `json:"error"`
}

// FallbackResult 降级结果
type FallbackResult struct {
	Success   bool                   `json:"success"`
	Variables map[string]interface{} `json:"variables"`
	Actions   []json.RawMessage      `json:"actions"`
}

// ExecuteFallback 执行降级
func (s *errorHandlingServiceImpl) ExecuteFallback(ctx context.Context, req *FallbackRequest) (*FallbackResult, error) {
	result := &FallbackResult{
		Success:   false,
		Variables: req.Variables,
		Actions:   []json.RawMessage{},
	}

	if req.FallbackAction == nil {
		return result, fmt.Errorf("降级动作未配置")
	}

	// 记录降级日志
	s.logFallback(ctx, req.ExecutionID, req.StepIndex, req.Error)

	// 解析降级动作
	var action map[string]interface{}
	if err := json.Unmarshal(*req.FallbackAction, &action); err != nil {
		return result, fmt.Errorf("解析降级动作失败: %w", err)
	}

	// 执行降级动作
	result.Variables["fallback"] = true
	result.Variables["originalError"] = req.Error.Error()
	result.Actions = append(result.Actions, *req.FallbackAction)
	result.Success = true

	return result, nil
}

// markExecutionFailed 标记执行失败
func (s *errorHandlingServiceImpl) markExecutionFailed(ctx context.Context, executionID string, err error) {
	s.db.WithContext(ctx).Model(&rpamodels.Execution{}).
		Where("id = ?", executionID).
		Updates(map[string]interface{}{
			"status":        string(rpamodels.RPAExecutionStatusFailed),
			"end_time":      time.Now(),
			"error_message": err.Error(),
		})
}

// logRetry 记录重试日志
func (s *errorHandlingServiceImpl) logRetry(ctx context.Context, executionID string, attempt int, delay time.Duration, err error) {
	logEntry := FormatLog(fmt.Sprintf("重试 %d/%d，延迟 %v: %v", attempt, 100, delay, err))
	s.db.WithContext(ctx).Model(&rpamodels.Execution{}).
		Where("id = ?", executionID).
		Update("logs", gorm.Expr("COALESCE(logs, '') || ?", "\n"+logEntry))
}

// logFallback 记录降级日志
func (s *errorHandlingServiceImpl) logFallback(ctx context.Context, executionID string, stepIndex int, err error) {
	logEntry := FormatLog(fmt.Sprintf("步骤 %d 降级执行: %v", stepIndex, err))
	s.db.WithContext(ctx).Model(&rpamodels.Execution{}).
		Where("id = ?", executionID).
		Update("logs", gorm.Expr("COALESCE(logs, '') || ?", "\n"+logEntry))
}

// contains 检查字符串包含
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		len(s) > 0 && (s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// intPtr 返回 int 指针
func intPtr(i int) *int {
	return &i
}

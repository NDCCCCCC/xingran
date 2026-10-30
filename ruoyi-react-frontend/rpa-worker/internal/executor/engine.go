package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/communication"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// Engine 执行引擎
type Engine struct {
	config           *config.ExecutorConfig
	browserPool      *browser.Pool
	progressReporter *communication.ProgressReporter
	logger           logger.Logger
}

// NewEngine 创建执行引擎
func NewEngine(
	cfg *config.ExecutorConfig,
	browserPool *browser.Pool,
	progressReporter *communication.ProgressReporter,
	log logger.Logger,
) *Engine {
	return &Engine{
		config:           cfg,
		browserPool:      browserPool,
		progressReporter: progressReporter,
		logger:           log,
	}
}

// Execute 执行任务
func (e *Engine) Execute(ctx context.Context, task *types.TaskMessage) (*types.ExecutionResult, error) {
	startTime := time.Now()

	e.logger.Info("开始执行任务",
		logger.String("execution_id", task.ExecutionID),
		logger.String("task_id", task.TaskID),
		logger.Int("actions", len(task.Script)),
	)

	result := &types.ExecutionResult{
		ExecutionID: task.ExecutionID,
		TaskID:      task.TaskID,
		Status:      types.StatusRunning,
		StartedAt:   startTime,
		Step:        0,
		Total:       len(task.Script),
	}

	// 获取浏览器页面
	pooledPage, err := e.browserPool.Acquire(ctx)
	if err != nil {
		return e.createErrorResult(result, err, 0, len(task.Script))
	}
	defer e.browserPool.Release(pooledPage)

	// 创建页面管理器
	pageMgr := browser.NewPageManager(pooledPage.page, e.logger)

	// 初始化变量上下文
	variables := make(map[string]interface{})
	for k, v := range task.Variables {
		variables[k] = v
	}

	// 导航到目标 URL
	if task.TargetURL != "" {
		if err := pageMgr.Goto(task.TargetURL); err != nil {
			return e.createErrorResult(result, err, 0, len(task.Script))
		}
	}

	// 执行脚本动作
	for i, action := range task.Script {
		result.Step = i + 1
		result.Status = types.StatusRunning

		// 上报进度
		if err := e.progressReporter.ReportProgress(ctx, &types.ProgressReport{
			ExecutionID:     task.ExecutionID,
			ProgressCurrent: i + 1,
			ProgressTotal:   len(task.Script),
			Message:         fmt.Sprintf("执行: %s %s", action.Type, action.Description),
			Status:          types.StatusRunning,
		}); err != nil {
			e.logger.Warn("上报进度失败", logger.Err(err))
		}

		// 执行动作
		if err := e.executeAction(ctx, pageMgr, &action, variables); err != nil {
			// 重试逻辑
			if action.Retry > 0 {
				lastErr := err
				for retry := 0; retry <= action.Retry; retry++ {
					e.logger.Warn("动作执行失败，重试",
						logger.Int("retry", retry),
						logger.Int("max_retries", action.Retry),
						logger.Err(err),
					)
					time.Sleep(e.config.InitialDelay)

					if err = e.executeAction(ctx, pageMgr, &action, variables); err == nil {
						break
					}
					lastErr = err
				}
				err = lastErr
			}

			if err != nil {
				return e.createErrorResult(result, err, i+1, len(task.Script))
			}
		}
	}

	// 任务成功完成
	result.Status = types.StatusSuccess
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime)

	e.logger.Info("任务执行成功",
		logger.String("execution_id", task.ExecutionID),
		logger.Duration("duration", result.Duration),
	)

	return result, nil
}

// executeAction 执行单个动作
func (e *Engine) executeAction(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
	e.logger.Debug("执行动作",
		logger.String("type", string(action.Type)),
		logger.String("selector", action.Selector),
		logger.String("description", action.Description),
	)

	// 变量替换
	params := e.substituteVariables(action.Params, variables)

	// 根据动作类型执行
	switch action.Type {
	case types.ActionNavigate:
		url, ok := params["url"].(string)
		if !ok {
			return fmt.Errorf("缺少 url 参数")
		}
		return pageMgr.Goto(url)

	case types.ActionClick:
		if action.Selector == "" {
			return fmt.Errorf("缺少 selector")
		}
		return pageMgr.Click(action.Selector)

	case types.ActionFill:
		if action.Selector == "" {
			return fmt.Errorf("缺少 selector")
		}
		value, ok := params["value"].(string)
		if !ok {
			return fmt.Errorf("缺少 value 参数")
		}
		return pageMgr.Fill(action.Selector, value)

	case types.ActionSelect:
		if action.Selector == "" {
			return fmt.Errorf("缺少 selector")
		}
		value, ok := params["value"].(string)
		if !ok {
			return fmt.Errorf("缺少 value 参数")
		}
		return pageMgr.Select(action.Selector, value)

	case types.ActionWait:
		duration, ok := params["duration"].(int)
		if !ok {
			duration = 1000
		}
		time.Sleep(time.Duration(duration) * time.Millisecond)
		return nil

	case types.ActionScreenshot:
		_, err := pageMgr.Screenshot()
		return err

	case types.ActionWaitFor:
		if action.Selector == "" {
			return fmt.Errorf("缺少 selector")
		}
		timeout := time.Duration(action.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		return pageMgr.WaitFor(action.Selector, timeout)

	default:
		return fmt.Errorf("不支持的动作类型: %s", action.Type)
	}
}

// substituteVariables 替换变量
func (e *Engine) substituteVariables(params map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range params {
		switch val := v.(type) {
		case string:
			result[k] = e.replaceVariables(val, variables)
		default:
			result[k] = v
		}
	}
	return result
}

// replaceVariables 替换字符串中的变量
func (e *Engine) replaceVariables(s string, variables map[string]interface{}) string {
	result := s
	for k, v := range variables {
		placeholder := fmt.Sprintf("${%s}", k)
		result = e.replaceAll(result, placeholder, fmt.Sprintf("%v", v))
	}
	return result
}

// replaceAll 替换所有匹配
func (e *Engine) replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result + s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// createErrorResult 创建错误结果
func (e *Engine) createErrorResult(result *types.ExecutionResult, err error, step, total int) *types.ExecutionResult {
	result.Status = types.StatusFailed
	result.ErrorMessage = err.Error()
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Step = step
	result.Total = total

	e.logger.Error("任务执行失败",
		logger.String("execution_id", result.ExecutionID),
		logger.Int("step", step),
		logger.Int("total", total),
		logger.Err(err),
	)

	return result
}

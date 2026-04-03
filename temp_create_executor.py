#!/usr/bin/env python3

engine_content = """package executor

import (
    "context"
    "fmt"
    "time"

    "github.com/ruoyi-next/rpa-worker/internal/browser"
    "github.com/ruoyi-next/rpa-worker/internal/communication"
    "github.com/ruoyi-next/rpa-worker/internal/config"
    "github.com/ruoyi-next/rpa-worker/internal/logger"
    "github.com/ruoyi-next/rpa-worker/internal/types"
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
    pageMgr := browser.NewPageManager(pooledPage.Page(), e.logger)

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
"""

page_manager_content = """package browser

import (
    "context"
    "fmt"
    "time"

    "github.com/ruoyi-next/rpa-worker/internal/logger"
)

// PageManager 页面管理器
type PageManager struct {
    page   Page
    logger logger.Logger
}

// NewPageManager 创建页面管理器
func NewPageManager(page Page, log logger.Logger) *PageManager {
    return &PageManager{
        page:   page,
        logger: log,
    }
}

// Goto 导航到 URL
func (m *PageManager) Goto(url string) error {
    m.logger.Debug("导航到页面", logger.String("url", url))
    return m.page.Goto(url)
}

// Click 点击元素
func (m *PageManager) Click(selector string) error {
    m.logger.Debug("点击元素", logger.String("selector", selector))
    return m.page.Click(selector)
}

// Fill 填写表单
func (m *PageManager) Fill(selector, value string) error {
    m.logger.Debug("填写表单",
        logger.String("selector", selector),
        logger.String("value", maskValue(value)))
    return m.page.Fill(selector, value)
}

// Select 选择下拉选项
func (m *PageManager) Select(selector, value string) error {
    m.logger.Debug("选择选项",
        logger.String("selector", selector),
        logger.String("value", value))
    return m.page.Select(selector, value)
}

// WaitFor 等待元素出现
func (m *PageManager) WaitFor(selector string, timeout time.Duration) error {
    m.logger.Debug("等待元素",
        logger.String("selector", selector),
        logger.Duration("timeout", timeout))
    return m.page.WaitFor(selector, timeout)
}

// Screenshot 截图
func (m *PageManager) Screenshot() ([]byte, error) {
    return m.page.Screenshot()
}

// Evaluate 执行 JavaScript
func (m *PageManager) Evaluate(script string) (interface{}, error) {
    return m.page.Evaluate(script)
}

// WaitForURL 等待 URL 变化
func (m *PageManager) WaitForURL(pattern string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("等待 URL 超时")
        case <-ticker.C:
            // TODO: 检查当前 URL 是否匹配
            return nil
        }
    }
}

// WaitForText 等待文本出现
func (m *PageManager) WaitForText(selector, text string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("等待文本超时")
        case <-ticker.C:
            // TODO: 检查文本是否存在
            return nil
        }
    }
}

// maskValue 隐藏敏感值
func maskValue(value string) string {
    if len(value) <= 4 {
        return "****"
    }
    return value[:2] + "****" + value[len(value)-2:]
}
"""

# Update pool.go to add Page() method
pool_update = """// Page returns the underlying page
func (p *PooledPage) Page() Page {
    return p.page
}
"""

# Create files
with open('rpa-worker/internal/executor/engine.go', 'w') as f:
    f.write(engine_content)
print("internal/executor/engine.go created")

with open('rpa-worker/internal/browser/page_manager.go', 'w') as f:
    f.write(page_manager_content)
print("internal/browser/page_manager.go created")

# Read existing pool.go and add Page() method
with open('rpa-worker/internal/browser/pool.go', 'r') as f:
    pool_content = f.read()

# Add Page() method to PooledPage struct
if 'func (p *PooledPage) Page()' not in pool_content:
    # Find the end of PooledPage struct and add the method
    pool_content = pool_content + "\n" + pool_update
    with open('rpa-worker/internal/browser/pool.go', 'w') as f:
        f.write(pool_content)
    print("Updated pool.go with Page() method")
else:
    print("pool.go already has Page() method")

print("\nAll executor and browser files created successfully!")

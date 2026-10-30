package device

import (
	"context"
	"fmt"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// deviceExecutorTimeoutBuffer 设备执行器等待任务完成时相对于任务超时的额外缓冲时间
const deviceExecutorTimeoutBuffer = 1 * time.Minute

// ExecutionConfig 执行配置
type ExecutionConfig struct {
	MaxRetries          int           // 最大重试次数
	RetryDelay          time.Duration // 重试延迟
	Timeout             time.Duration // 默认超时时间
	EnablePanicRecovery bool          // 是否启用 panic 恢复
}

// DefaultExecutionConfig 默认执行配置
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		MaxRetries:          3,
		RetryDelay:          time.Second,
		Timeout:             30 * time.Second,
		EnablePanicRecovery: true,
	}
}

// DeviceExecutor 设备执行器
// 提供统一的命令执行接口，支持重试、超时、panic 恢复
type DeviceExecutor struct {
	scheduler *DeviceTaskScheduler
	config    *ExecutionConfig
}

// NewDeviceExecutor 创建设备执行器
func NewDeviceExecutor(scheduler *DeviceTaskScheduler, config *ExecutionConfig) *DeviceExecutor {
	if config == nil {
		config = DefaultExecutionConfig()
	}

	return &DeviceExecutor{
		scheduler: scheduler,
		config:    config,
	}
}

// ExecuteOnDevice 在设备上执行单个命令
func (e *DeviceExecutor) ExecuteOnDevice(ctx context.Context, deviceID string, command string, stripPrompt bool) (string, error) {
	var result string
	var taskErr error

	// 创建任务
	task := &DeviceTask{
		ID:       generateTaskID(),
		DeviceID: deviceID,
		Timeout:  e.config.Timeout,
		Execute: func(taskCtx context.Context, conn *PooledConnection) error {
			// 执行命令（带重试）
			response, err := e.executeWithRetry(taskCtx, conn, command, stripPrompt)
			if err != nil {
				return err
			}
			result = response.Result
			return nil
		},
		Callback: func(err error) {
			taskErr = err
		},
	}

	// 提交任务
	if err := e.scheduler.Submit(task); err != nil {
		return "", err
	}

	// 等待任务完成
	waitCtx, cancel := context.WithTimeout(ctx, e.config.Timeout+deviceExecutorTimeoutBuffer)
	defer cancel()

	// 使用 ticker 进行轮询
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return "", fmt.Errorf("任务执行超时: taskID=%s", task.ID)
		case <-ticker.C:
			if taskErr != nil || result != "" {
				return result, taskErr
			}
		}
	}
}

// ExecuteMultipleOnDevice 在设备上执行多个命令
func (e *DeviceExecutor) ExecuteMultipleOnDevice(ctx context.Context, deviceID string, commands []string, stripPrompt bool) ([]string, error) {
	var results []string
	var taskErr error

	// 创建任务 - 每个命令至少给1分钟超时
	taskTimeout := time.Duration(len(commands)) * time.Minute
	task := &DeviceTask{
		ID:       generateTaskID(),
		DeviceID: deviceID,
		Timeout:  taskTimeout,
		Execute: func(taskCtx context.Context, conn *PooledConnection) error {
			for _, cmd := range commands {
				response, err := e.executeWithRetry(taskCtx, conn, cmd, stripPrompt)
				if err != nil {
					return fmt.Errorf("命令 '%s' 执行失败: %w", cmd, err)
				}
				results = append(results, response.Result)
			}
			return nil
		},
		Callback: func(err error) {
			taskErr = err
		},
	}

	// 提交任务
	if err := e.scheduler.Submit(task); err != nil {
		return nil, err
	}

	// 等待任务完成
	waitCtx, cancel := context.WithTimeout(ctx, taskTimeout+deviceExecutorTimeoutBuffer)
	defer cancel()

	// 使用 ticker 进行轮询
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("任务执行超时: taskID=%s", task.ID)
		case <-ticker.C:
			if taskErr != nil || len(results) == len(commands) {
				return results, taskErr
			}
		}
	}
}

// ExecuteCustom 执行自定义任务
//
// 任务完成信号（修复端口写操作 30s 必超时 bug）：
// executeFunc 返回后 scheduler 会调用 task.Callback（成功传 nil，失败传 err），
// 本函数在 Callback 内 close(done) 作为"任务已结束"信号，select 命中 <-done 立即返回 taskErr。
//
// 早期实现仅用 ticker 轮询 `taskErr != nil`（只覆盖失败路径），成功时 taskErr 恒为 nil，
// 导致循环空转到 waitCtx.Done()（timeout + 1min buffer ≈ 90s）才返回；
// 而前端 networkApi axios timeout=30s 先于后端返回，触发 ECONNABORTED "请求超时"。
// 此处与 SubmitAndWait 的 resultCh + select 模式对齐。
func (e *DeviceExecutor) ExecuteCustom(ctx context.Context, deviceID string, executeFunc func(context.Context, *PooledConnection) error, timeout time.Duration) error {
	var taskErr error
	done := make(chan struct{})

	// 创建任务
	task := &DeviceTask{
		ID:       generateTaskID(),
		DeviceID: deviceID,
		Timeout:  timeout,
		Execute:  executeFunc,
		Callback: func(err error) {
			taskErr = err
			close(done)
		},
	}

	// 提交任务
	if err := e.scheduler.Submit(task); err != nil {
		return err
	}

	// 等待任务完成：成功/失败立即返回（<-done），超时由 timeout+buffer 兜底
	waitCtx, cancel := context.WithTimeout(ctx, timeout+deviceExecutorTimeoutBuffer)
	defer cancel()

	select {
	case <-waitCtx.Done():
		return fmt.Errorf("任务执行超时: taskID=%s", task.ID)
	case <-done:
		// 任务已结束（Callback 已设 taskErr：成功=nil，失败=err）
		return taskErr
	}
}

// GetScheduler 获取调度器
func (e *DeviceExecutor) GetScheduler() *DeviceTaskScheduler {
	return e.scheduler
}

// executeWithRetry 执行命令（带重试和 panic 恢复）
func (e *DeviceExecutor) executeWithRetry(ctx context.Context, conn *PooledConnection, command string, stripPrompt bool) (*Response, error) {
	var lastErr error

	applogger.Debugf("[执行器] executeWithRetry 开始: command='%s', maxRetries=%d", command, e.config.MaxRetries)

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		applogger.Debugf("[执行器] 尝试执行命令: command='%s', attempt=%d/%d",
			command, attempt+1, e.config.MaxRetries+1)

		if attempt > 0 {
			// 重试延迟
			applogger.Debugf("[执行器] 等待重试: command='%s', 尝试=%d/%d, 延迟=%v",
				command, attempt, e.config.MaxRetries, e.config.RetryDelay)

			select {
			case <-time.After(e.config.RetryDelay):
			case <-ctx.Done():
				applogger.Debugf("[执行器] 重试被取消: command='%s'", command)
				return nil, ctx.Err()
			}
		}

		// 执行命令（带 panic 恢复）
		var response *Response
		var execErr error

		applogger.Debugf("[执行器] 准备调用 wrapper.SendCommand: command='%s', stripPrompt=%v", command, stripPrompt)

		if e.config.EnablePanicRecovery {
			func() {
				defer func() {
					if r := recover(); r != nil {
						applogger.Warnf("[执行器] Panic 恢复: command='%s', panic=%v", command, r)
						execErr = fmt.Errorf("命令执行 panic: %v", r)
					}
				}()

				// 注意：conn 已经在 task_scheduler 中获取，直接使用 wrapper
				response, execErr = conn.wrapper.SendCommand(command, stripPrompt)
				applogger.Debugf("[执行器] wrapper.SendCommand 返回: command='%s', error=%v", command, execErr)
				if execErr == nil && response != nil {
					applogger.Debugf("[执行器] 命令响应: resultSize=%d", len(response.Result))
				}
			}()
		} else {
			// 注意：conn 已经在 task_scheduler 中获取，直接使用 wrapper
			response, execErr = conn.wrapper.SendCommand(command, stripPrompt)
			applogger.Debugf("[执行器] wrapper.SendCommand 返回: command='%s', error=%v", command, execErr)
		}

		if execErr == nil {
			applogger.Debugf("[执行器] 命令执行成功: command='%s', 尝试=%d/%d",
				command, attempt+1, e.config.MaxRetries+1)
			return response, nil
		}

		lastErr = execErr
		applogger.Debugf("[执行器] 命令执行失败: command='%s', 尝试=%d/%d, error=%v",
			command, attempt+1, e.config.MaxRetries+1, execErr)
	}

	applogger.Warnf("[执行器] 所有重试均失败: command='%s'", command)
	return nil, fmt.Errorf("执行失败，已达最大重试次数: %w", lastErr)
}

// GetConfig 获取设备配置（根据设备厂商自动选择命令）
func (e *DeviceExecutor) GetConfig(ctx context.Context, deviceID string) (string, error) {
	var result string
	var taskErr error

	applogger.Debugf("[执行器] GetConfig 开始: deviceID=%s", deviceID)

	// 获取连接池
	pool := e.scheduler.GetConnectionPool()
	applogger.Debugf("[执行器] 获取连接池成功")

	// 获取设备信息以确定厂商
	device, err := pool.GetDevice(deviceID)
	if err != nil {
		applogger.Warnf("[执行器] 获取设备信息失败: deviceID=%s, error=%v", deviceID, err)
		return "", fmt.Errorf("获取设备信息失败: %w", err)
	}

	applogger.Debugf("[执行器] 设备信息获取成功: name=%s, ip=%s, vendor=%s",
		device.DeviceName, device.IPAddress, device.Vendor)

	// 根据厂商选择正确的配置命令
	var configCommand string
	switch device.Vendor {
	case "huawei", "h3c":
		configCommand = "display current-configuration"
	case "ruijie", "maipu":
		configCommand = "show running-config"
	default:
		configCommand = "show running-config" // 默认使用 Cisco 风格命令
	}

	applogger.Debugf("[执行器] 设备 %s (%s), 厂商: %s, 使用命令: %s",
		device.DeviceName, device.IPAddress, device.Vendor, configCommand)

	// 生成任务ID
	taskID := generateTaskID()
	applogger.Debugf("[执行器] 任务已生成: taskID=%s", taskID)

	// 创建任务 - 配置备份需要更长的超时时间
	task := &DeviceTask{
		ID:       taskID,
		DeviceID: deviceID,
		Timeout:  3 * time.Minute, // 配置备份使用3分钟超时
		Execute: func(taskCtx context.Context, conn *PooledConnection) error {
			applogger.Debugf("[执行器] 任务开始执行: taskID=%s, deviceID=%s", taskID, deviceID)
			// 根据厂商获取配置命令
			config, err := e.executeWithRetry(taskCtx, conn, configCommand, true)
			if err != nil {
				applogger.Warnf("[执行器] 任务执行失败: taskID=%s, error=%v", taskID, err)
				return err
			}
			result = config.Result
			applogger.Debugf("[执行器] 任务执行成功: taskID=%s, resultSize=%d", taskID, len(result))
			return nil
		},
		Callback: func(err error) {
			applogger.Debugf("[执行器] 任务回调: taskID=%s, error=%v", taskID, err)
			taskErr = err
		},
	}

	applogger.Debugf("[执行器] 任务已创建: timeout=%v", task.Timeout)

	// 提交任务
	applogger.Debugf("[执行器] 准备提交任务: taskID=%s", taskID)
	if err := e.scheduler.Submit(task); err != nil {
		applogger.Warnf("[执行器] 任务提交失败: taskID=%s, error=%v", taskID, err)
		return "", err
	}
	applogger.Debugf("[执行器] 任务提交成功: taskID=%s", taskID)

	// 等待任务完成 - 使用更长的超时时间
	waitTimeout := 3 * time.Minute
	applogger.Debugf("[执行器] 开始等待任务完成: taskID=%s, waitTimeout=%v", taskID, waitTimeout)
	waitCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

	// 使用 ticker 进行轮询，确保超时时间正确
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	pollCount := 0
	for {
		select {
		case <-waitCtx.Done():
			applogger.Warnf("[执行器] 等待超时: taskID=%s, pollCount=%d", taskID, pollCount)
			return "", fmt.Errorf("任务执行超时: taskID=%s", taskID)
		case <-ticker.C:
			pollCount++
			if pollCount%50 == 0 {
				// 每5秒打印一次轮询状态
				applogger.Debugf("[执行器] 轮询中: taskID=%s, pollCount=%d, taskErr=%v, hasResult=%v",
					taskID, pollCount, taskErr, result != "")
			}
			if taskErr != nil || result != "" {
				applogger.Debugf("[执行器] 任务完成: taskID=%s, pollCount=%d, error=%v",
					taskID, pollCount, taskErr)
				return result, taskErr
			}
		}
	}
}

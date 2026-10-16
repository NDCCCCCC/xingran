package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
	"golang.org/x/sync/errgroup"
)

// TaskSplitter 任务拆分器
type TaskSplitter struct {
	redisClient   *redis.Client
	streamName    string
	groupName     string
	config        *config.HybridModeConfig
	logger        logger.Logger
	activeMonitors sync.Map // map[parentExecutionID]context.CancelFunc
}

// NewTaskSplitter 创建任务拆分器
func NewTaskSplitter(
	redisClient *redis.Client,
	streamName string,
	groupName string,
	cfg *config.HybridModeConfig,
	log logger.Logger,
) *TaskSplitter {
	return &TaskSplitter{
		redisClient: redisClient,
		streamName:  streamName,
		groupName:   groupName,
		config:      cfg,
		logger:      log,
	}
}

// ShouldSplit 判断是否需要拆分任务
func (ts *TaskSplitter) ShouldSplit(task *types.TaskMessage) bool {
	if !ts.config.Enabled {
		return false
	}

	// 检查是否已经是子任务（避免递归拆分）
	if _, ok := task.Variables[types.VariableIsSubTask]; ok {
		return false
	}

	// 检查脚本中是否包含 Loop 动作
	for _, action := range task.Script {
		if action.Type == types.ActionLoop {
			return true
		}
	}

	return false
}

// SplitResult 拆分结果
type SplitResult struct {
	ParentExecutionID string
	SubTaskCount      int
}

// SplitTask 拆分任务为子任务并提交到 Redis
func (ts *TaskSplitter) SplitTask(ctx context.Context, task *types.TaskMessage) (*SplitResult, error) {
	// 查找 Loop 动作
	loopAction, loopIndex := ts.findLoopAction(task.Script)
	if loopAction == nil {
		return nil, fmt.Errorf("no loop action found")
	}

	// 提取 Loop 数据源
	dataSource, err := ts.extractDataSource(loopAction, task.Variables)
	if err != nil {
		return nil, fmt.Errorf("extract data source failed: %w", err)
	}

	if len(dataSource) == 0 {
		return nil, fmt.Errorf("loop data source is empty")
	}

	ts.logger.Info("splitting task into subtasks",
		logger.String("execution_id", task.ExecutionID),
		logger.Int("subtasks", len(dataSource)),
		logger.Int("loop_action_index", loopIndex))

	// 使用 errgroup 进行并发子任务提交
	// 设置并发限制以避免过载 Redis
	const maxConcurrentSubmissions = 10
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentSubmissions)

	// 使用通道收集提交结果
	type submitResult struct {
		index int
		err   error
	}
	resultChan := make(chan submitResult, len(dataSource))

	// 并发提交所有子任务
	for i, item := range dataSource {
		i, item := i, item // 捕获循环变量
		subTask := ts.createSubTask(task, loopIndex, loopAction, i, len(dataSource), item)

		g.Go(func() error {
			err := ts.submitSubTask(ctx, subTask)
			resultChan <- submitResult{index: i, err: err}
			return err
		})
	}

	// 等待所有提交完成（或遇到错误）
	go func() {
		g.Wait()
		close(resultChan)
	}()

	// 收集结果并统计
	submitCount := 0
	failedCount := 0
	const maxFailureRate = 0.5 // 允许最大 50% 失败率

	for result := range resultChan {
		if result.err != nil {
			ts.logger.Error("failed to submit subtask",
				logger.Int("index", result.index),
				logger.Err(result.err))
			failedCount++

			// 检查失败率是否过高
			currentRate := float64(failedCount) / float64(result.index+1)
			if currentRate > maxFailureRate && result.index > 4 {
				// 取消剩余的提交
				return nil, fmt.Errorf("subtask submission failure rate too high: %d/%d (%.1f%%)",
					failedCount, result.index+1, currentRate*100)
			}
		} else {
			submitCount++
			ts.logger.Debug("subtask submitted",
				logger.Int("index", result.index),
				logger.Int("total", len(dataSource)))
		}
	}

	// 等待所有 goroutine 完成
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("subtask submission failed: %w", err)
	}

	if submitCount == 0 {
		return nil, fmt.Errorf("failed to submit any subtask (all %d submissions failed)", len(dataSource))
	}

	// 创建带超时和取消功能的监控上下文
	// 超时时间 = 子任务超时 × 数量 + 10分钟缓冲
	// 注意：避免整数溢出，设置合理的上限
	monitorTimeout := time.Duration(len(dataSource)) * ts.config.SubTaskTimeout
	if monitorTimeout < 0 {
		// 防止整数溢出
		monitorTimeout = 24 * time.Hour
	}
	if monitorTimeout > 24*time.Hour {
		// 设置合理的上限：最长24小时
		monitorTimeout = 24 * time.Hour
	}
	monitorTimeout += 10 * time.Minute // 缓冲时间
	monitorCtx, monitorCancel := context.WithTimeout(ctx, monitorTimeout)
	ts.activeMonitors.Store(task.ExecutionID, monitorCancel)

	// 启动进度监听
	go func() {
		ts.monitorSubTaskProgress(monitorCtx, task.ExecutionID, len(dataSource))
		// 监控完成后清理
		ts.activeMonitors.Delete(task.ExecutionID)
	}()

	return &SplitResult{
		ParentExecutionID: task.ExecutionID,
		SubTaskCount:      submitCount,
	}, nil
}

// StopMonitor 停止指定任务的进度监控
func (ts *TaskSplitter) StopMonitor(parentExecutionID string) {
	value, loaded := ts.activeMonitors.LoadAndDelete(parentExecutionID)
	if !loaded {
		ts.logger.Debug("monitor not found or already stopped",
			logger.String("parent_execution_id", parentExecutionID))
		return
	}

	cancelFunc, ok := value.(context.CancelFunc)
	if !ok {
		ts.logger.Warn("invalid cancel function type in active monitors",
			logger.String("parent_execution_id", parentExecutionID),
			logger.String("type", fmt.Sprintf("%T", value)))
		return
	}

	cancelFunc()
	ts.logger.Info("stopped subtask progress monitor",
		logger.String("parent_execution_id", parentExecutionID))
}

// StopAllMonitors 停止所有活跃的进度监控
func (ts *TaskSplitter) StopAllMonitors() {
	var stoppedCount int
	ts.activeMonitors.Range(func(key, value interface{}) bool {
		cancelFunc, ok := value.(context.CancelFunc)
		if !ok {
			ts.logger.Warn("skipping invalid cancel function",
				logger.String("key", fmt.Sprintf("%v", key)),
				logger.String("type", fmt.Sprintf("%T", value)))
			return true
		}
		cancelFunc()
		stoppedCount++
		return true
	})
	ts.logger.Info("stopped all subtask progress monitors",
		logger.Int("count", stoppedCount))
}

// findLoopAction 查找 Loop 动作及其索引
func (ts *TaskSplitter) findLoopAction(actions []types.Action) (*types.Action, int) {
	for i, action := range actions {
		if action.Type == types.ActionLoop {
			return &action, i
		}
	}
	return nil, -1
}

// extractDataSource 提取 Loop 数据源
func (ts *TaskSplitter) extractDataSource(loopAction *types.Action, variables map[string]interface{}) ([]map[string]interface{}, error) {
	dataSource, ok := loopAction.Params["dataSource"]
	if !ok {
		return nil, fmt.Errorf("dataSource parameter not found")
	}

	var dataArray []map[string]interface{}
	switch v := dataSource.(type) {
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				dataArray = append(dataArray, itemMap)
			}
		}
	case []map[string]interface{}:
		dataArray = v
	default:
		return nil, fmt.Errorf("invalid dataSource type: %T", dataSource)
	}

	return dataArray, nil
}

// createSubTask 创建子任务
func (ts *TaskSplitter) createSubTask(
	parentTask *types.TaskMessage,
	loopIndex int,
	loopAction *types.Action,
	itemIndex int,
	totalItems int,
	itemData map[string]interface{},
) *types.SubTaskMessage {
	// 生成子任务执行ID
	subExecutionID := fmt.Sprintf("%s-sub-%d", parentTask.ExecutionID, itemIndex)

	// 获取嵌套动作
	nestedActions := ts.extractNestedActions(loopAction.Params["actions"])

	// 构建子任务脚本：Loop 之前的动作 + 嵌套动作
	subTaskScript := make([]types.Action, 0, loopIndex+len(nestedActions))
	subTaskScript = append(subTaskScript, parentTask.Script[:loopIndex]...) // Loop 之前的动作
	subTaskScript = append(subTaskScript, nestedActions...)                // Loop 内的动作

	// 获取循环项变量名
	itemName := "item"
	if name, ok := loopAction.Params["itemVar"].(string); ok && name != "" {
		itemName = name
	}

	// 创建子任务变量上下文
	subTaskVariables := make(map[string]interface{})
	for k, v := range parentTask.Variables {
		subTaskVariables[k] = v
	}

	// 添加循环项数据到变量
	for k, v := range itemData {
		subTaskVariables[itemName+"."+k] = v
		subTaskVariables[k] = v
	}
	subTaskVariables[itemName+".index"] = itemIndex
	subTaskVariables["index"] = itemIndex
	subTaskVariables[types.VariableIsSubTask] = true // 标记为子任务
	subTaskVariables[types.VariableParentExecutionID] = parentTask.ExecutionID
	subTaskVariables[types.VariableSubTaskIndex] = itemIndex

	// 获取会话数据（如果有）
	if sessionData, ok := parentTask.Variables[types.VariableSessionData]; ok {
		subTaskVariables[types.VariableSessionData] = sessionData
	}
	if credentials, ok := parentTask.Variables[types.VariableCredentials]; ok {
		subTaskVariables[types.VariableCredentials] = credentials
	}

	return &types.SubTaskMessage{
		TaskMessage: &types.TaskMessage{
			ExecutionID:   subExecutionID,
			TaskID:        parentTask.TaskID,
			TaskName:      fmt.Sprintf("%s [SubTask %d/%d]", parentTask.TaskName, itemIndex+1, totalItems),
			TargetURL:     parentTask.TargetURL,
			Script:        subTaskScript,
			Timeout:       ts.config.SubTaskTimeout,
			MaxRetry:      ts.config.SubTaskRetryCount,
			Variables:     subTaskVariables,
			TriggeredBy:   parentTask.TriggeredBy,
			TriggerType:   "subtask",
			CredentialID:  parentTask.CredentialID,
			SessionID:     parentTask.SessionID,
			SessionData:   parentTask.SessionData,
			CreatedAt:     time.Now(),
		},
		ParentExecutionID: parentTask.ExecutionID,
		SubTaskIndex:      itemIndex,
		SubTaskTotal:      totalItems,
		LoopItemVar:       itemName,
		LoopItemData:      itemData,
	}
}

// extractNestedActions 提取嵌套动作
func (ts *TaskSplitter) extractNestedActions(actionsRaw interface{}) []types.Action {
	// 如果已经是正确的类型，直接返回
	if actions, ok := actionsRaw.([]types.Action); ok {
		return actions
	}

	// 使用 JSON 序列化/反序列化进行类型转换
	data, err := json.Marshal(actionsRaw)
	if err != nil {
		ts.logger.Warn("failed to marshal nested actions", logger.Err(err))
		return nil
	}

	var actions []types.Action
	if err := json.Unmarshal(data, &actions); err != nil {
		ts.logger.Warn("failed to unmarshal nested actions", logger.Err(err))
		return nil
	}

	return actions
}

// submitSubTask 提交子任务到 Redis Stream
func (ts *TaskSplitter) submitSubTask(ctx context.Context, subTask *types.SubTaskMessage) error {
	// 序列化子任务
	data, err := json.Marshal(subTask)
	if err != nil {
		return fmt.Errorf("marshal subtask failed: %w", err)
	}

	// 提交到 Redis Stream
	err = ts.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: ts.streamName,
		Values: map[string]interface{}{"data": string(data)},
	}).Err()

	return err
}

// monitorSubTaskProgress 监控子任务进度并聚合上报
func (ts *TaskSplitter) monitorSubTaskProgress(ctx context.Context, parentExecutionID string, totalSubTasks int) {
	ts.logger.Info("starting subtask progress monitor",
		logger.String("parent_execution_id", parentExecutionID),
		logger.Int("total_subtasks", totalSubTasks))

	// 订阅子任务进度频道
	progressChannel := fmt.Sprintf("subtask:progress:%s", parentExecutionID)
	pubsub := ts.redisClient.Subscribe(ctx, progressChannel)
	defer pubsub.Close()

	// 节流定时器
	ticker := time.NewTicker(ts.config.ProgressThrottle)
	defer ticker.Stop()

	// 进度跟踪
	var (
		mu               sync.RWMutex
		completedCount   int
		successCount     int
		failureCount     int
		lastReportTime   time.Time
		subTaskResults   = make(map[int]*types.SubTaskResult)
	)

	// 结果通道
	resultChannel := fmt.Sprintf("subtask:result:%s", parentExecutionID)
	resultPubsub := ts.redisClient.Subscribe(ctx, resultChannel)
	defer resultPubsub.Close()

	for {
		select {
		case <-ctx.Done():
			ts.logger.Info("progress monitor context done",
				logger.String("parent_execution_id", parentExecutionID))
			return

		case msg := <-pubsub.Channel():
			if msg.Payload == "" {
				continue
			}

			// 解析进度消息
			var progressMsg struct {
				SubTaskIndex int                    `json:"subTaskIndex"`
				Status       types.ExecutionStatus `json:"status"`
				Step         int                    `json:"step"`
				Total        int                    `json:"total"`
				Message      string                 `json:"message"`
				Timestamp    time.Time              `json:"timestamp"`
			}

			if err := json.Unmarshal([]byte(msg.Payload), &progressMsg); err != nil {
				ts.logger.Warn("failed to parse progress message", logger.Err(err))
				continue
			}

			// 更新进度状态
			mu.Lock()
			if progressMsg.Status == types.StatusSuccess || progressMsg.Status == types.StatusFailed {
				if _, exists := subTaskResults[progressMsg.SubTaskIndex]; !exists {
					subTaskResults[progressMsg.SubTaskIndex] = &types.SubTaskResult{
						SubTaskIndex: progressMsg.SubTaskIndex,
						Status:       progressMsg.Status,
						CompletedAt:  time.Now(),
					}
					completedCount++
					if progressMsg.Status == types.StatusSuccess {
						successCount++
					} else {
						failureCount++
					}
				}
			}
			mu.Unlock()

		case msg := <-resultPubsub.Channel():
			if msg.Payload == "" {
				continue
			}

			// 解析结果消息
			var result types.SubTaskResult
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				ts.logger.Warn("failed to parse result message", logger.Err(err))
				continue
			}

			mu.Lock()
			subTaskResults[result.SubTaskIndex] = &result
			completedCount++
			if result.Status == types.StatusSuccess {
				successCount++
			} else {
				failureCount++
			}
			mu.Unlock()

		case <-ticker.C:
			// 定时聚合上报进度
			mu.Lock()
			currentCompleted := completedCount
			currentSuccess := successCount
			currentFailure := failureCount
			// 创建结果副本，避免在锁外访问 map
			resultsCopy := make(map[int]*types.SubTaskResult, len(subTaskResults))
			for k, v := range subTaskResults {
				resultsCopy[k] = v
			}
			mu.Unlock()

			// 节流控制
			if time.Since(lastReportTime) < ts.config.ProgressThrottle {
				continue
			}

			// 检查是否所有子任务都已完成
			if currentCompleted >= totalSubTasks {
				ts.reportFinalResult(ctx, parentExecutionID, totalSubTasks, currentSuccess, currentFailure, resultsCopy)
				return
			}

			// 上报聚合进度
			ts.reportAggregatedProgress(ctx, parentExecutionID, currentCompleted, totalSubTasks, currentSuccess, currentFailure)
			lastReportTime = time.Now()
		}
	}
}

// reportAggregatedProgress 聚合上报进度
func (ts *TaskSplitter) reportAggregatedProgress(ctx context.Context, parentExecutionID string, completed, total, success, failure int) {
	progressMsg := types.ProgressAggregateMessage{
		ParentExecutionID: parentExecutionID,
		CompletedCount:    completed,
		TotalCount:        total,
		SuccessCount:      success,
		FailureCount:      failure,
		Status:            types.StatusRunning,
		Timestamp:         time.Now(),
	}

	progressChannel := fmt.Sprintf("task:progress:%s", parentExecutionID)
	data, err := json.Marshal(progressMsg)
	if err != nil {
		ts.logger.Error("failed to marshal progress message", logger.Err(err))
		return
	}

	if err := ts.redisClient.Publish(ctx, progressChannel, data).Err(); err != nil {
		ts.logger.Warn("failed to publish aggregated progress", logger.Err(err))
	}
}

// reportFinalResult 报告最终结果
func (ts *TaskSplitter) reportFinalResult(ctx context.Context, parentExecutionID string, total, success, failure int, results map[int]*types.SubTaskResult) {
	ts.logger.Info("all subtasks completed",
		logger.String("parent_execution_id", parentExecutionID),
		logger.Int("completed", total),
		logger.Int("success", success),
		logger.Int("failure", failure))

	// 构建最终结果
	finalResult := types.ProgressAggregateMessage{
		ParentExecutionID: parentExecutionID,
		CompletedCount:    total,
		TotalCount:        total,
		SuccessCount:      success,
		FailureCount:      failure,
		Status:            ts.calculateOverallStatus(total, success, failure),
		Timestamp:         time.Now(),
	}

	// 转换结果 map 为切片
	if len(results) > 0 {
		finalResult.SubTaskResults = make([]types.SubTaskResult, 0, len(results))
		for _, result := range results {
			if result != nil {
				finalResult.SubTaskResults = append(finalResult.SubTaskResults, *result)
			}
		}
	}

	resultChannel := fmt.Sprintf("task:result:%s", parentExecutionID)
	data, err := json.Marshal(finalResult)
	if err != nil {
		ts.logger.Error("failed to marshal final result", logger.Err(err))
		return
	}

	if err := ts.redisClient.Publish(ctx, resultChannel, data).Err(); err != nil {
		ts.logger.Error("failed to publish final result", logger.Err(err))
	}
}

// calculateOverallStatus 计算整体状态
func (ts *TaskSplitter) calculateOverallStatus(total, success, failure int) types.ExecutionStatus {
	if total == 0 {
		return types.StatusRunning
	}
	if failure == 0 {
		return types.StatusSuccess
	}
	if success == 0 {
		return types.StatusFailed
	}
	// 部分失败仍视为成功（根据业务需求可调整）
	return types.StatusSuccess
}

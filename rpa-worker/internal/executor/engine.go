package executor

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/communication"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/liveview"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// Engine execution engine
type Engine struct {
	config             *config.ExecutorConfig
	workerConfig       *config.WorkerConfig
	browserManager     *browser.Manager
	progressReporter   *communication.ProgressReporter
	logger             logger.Logger
	redisClient        *redis.Client
	currentExecutionID string
	currentWorkerID    string
	currentStep        int
	currentTotal       int
	loopConcurrency    int // 循环内并发执行数量
	// liveView Phase 155 / LIVEVIEW-01~05：实时画面会话表（nil 安全：单测与
	// 离线执行可无推流）。归属：Worker 构造时注入，engine 只在该表上调用。
	liveView *liveview.Hub
}

// SetLiveViewHub 注入实时画面会话表（Phase 155）。
//
// nil 安全：未启用 liveview 时传 nil，全部调用点短路，行为与 Phase 154 完全一致。
func (e *Engine) SetLiveViewHub(hub *liveview.Hub) {
	if e == nil {
		return
	}
	e.liveView = hub
}

// NewEngine create execution engine
func NewEngine(
	cfg *config.ExecutorConfig,
	workerCfg *config.WorkerConfig,
	browserManager *browser.Manager,
	progressReporter *communication.ProgressReporter,
	log logger.Logger,
	redisClient *redis.Client,
) *Engine {
	return &Engine{
		config:           cfg,
		workerConfig:     workerCfg,
		browserManager:   browserManager,
		progressReporter: progressReporter,
		logger:           log,
		redisClient:      redisClient,
		loopConcurrency:  workerCfg.LoopConcurrency, // 从配置中获取循环并发数
	}
}

// reportProgress 统一上报入口（nil 安全：集成测试与离线执行可无上报器）。
func (e *Engine) reportProgress(ctx context.Context, req *types.ProgressReport) {
	if e.progressReporter == nil {
		return
	}
	if err := e.progressReporter.ReportProgress(ctx, req); err != nil {
		e.logger.Warn("failed to report progress", logger.Err(err))
	}
}

// Execute execute task
func (e *Engine) Execute(ctx context.Context, task *types.TaskMessage, workerID string) (*types.ExecutionResult, error) {
	startTime := time.Now()

	// Set current execution context for pause action
	e.currentExecutionID = task.ExecutionID
	e.currentWorkerID = workerID

	// W7.1 / 缺陷 A：反序列化阶段命中的别名在这里统一打 WARN（types 包没有
	// logger）。两条进入路径（顶层 script / 嵌套动作）共用同一份文案口径，
	// 运维因此能持续发现历史写法。
	for _, raw := range task.NormalizedAliases() {
		if normalized, _ := types.NormalizeActionType(raw); normalized != types.ActionType(raw) {
			e.logNormalizedActionType(raw, normalized, "")
		}
	}

	// 检查是否是子任务
	isSubTask, parentExecutionID, subTaskIndex := e.extractSubTaskMeta(task)

	// 记录日志 - 子任务使用 debug 级别减少日志量
	if isSubTask {
		e.logger.Debug("start executing task",
			logger.String("execution_id", task.ExecutionID),
			logger.String("task_id", task.TaskID),
			logger.String("worker_id", workerID),
			logger.Int("actions", len(task.Script)),
			logger.String("task_type", "subtask"),
		)
	} else {
		e.logger.Info("start executing task",
			logger.String("execution_id", task.ExecutionID),
			logger.String("task_id", task.TaskID),
			logger.String("worker_id", workerID),
			logger.Int("actions", len(task.Script)),
		)
	}

	result := &types.ExecutionResult{
		ExecutionID: task.ExecutionID,
		TaskID:      task.TaskID,
		Status:      types.StatusRunning,
		StartedAt:   startTime,
		Step:        0,
		Total:       len(task.Script),
	}

	// Acquire browser page（Phase 154 / ENGINE-01：1 Chrome + N page）
	page, err := e.browserManager.Acquire(ctx)
	if err != nil {
		errorResult := e.createErrorResult(result, err, 0, len(task.Script))
		e.reportSubTaskResult(ctx, task, errorResult)
		return errorResult, fmt.Errorf("failed to acquire browser: %w", err)
	}
	defer e.browserManager.Release(page)
	// Phase 155 / LIVEVIEW：Attach 与 Release 的顺序必须谨慎 ——
	// detach defer 在 Release defer 之后注册，LIFO 故先执行 Detach（停 screencast），
	// 再由 Release 回收 page（「Close 只关 page」的语义由此保住）。
	e.attachLiveView(task, workerID, page, len(task.Script))
	defer e.detachLiveView(task.ExecutionID)

	// Create page manager
	pageMgr := browser.NewPageManager(page, e.logger)

	// Initialize variable context（Phase 154 / ENGINE-04：三层作用域显式化）
	variables := NewVariableScope(task.Variables)

	// Phase 153 / RUNTIME-02：任务携带 __sessionData 时真实注入会话
	//（cookies + localStorage + token 头），先注入后导航保证目标页首载即登录态。
	if sd, ok := task.Variables[types.VariableSessionData]; ok && sd != nil {
		if err := e.applySessionData(ctx, page, sd, task.TargetURL); err != nil {
			e.logger.Warn("apply session data failed",
				logger.String("execution_id", task.ExecutionID), logger.Err(err))
		}
	}

	// Navigate to target URL
	if task.TargetURL != "" {
		if err := pageMgr.Goto(task.TargetURL); err != nil {
			errorResult := e.createErrorResult(result, err, 0, len(task.Script))
			e.reportSubTaskResult(ctx, task, errorResult)
			return errorResult, fmt.Errorf("failed to navigate to %s: %w", task.TargetURL, err)
		}
	}

	// Execute script actions
	for i, action := range task.Script {
		// Phase 153 / RUNTIME-03：动作边界 ctx 取消检查点（每个动作至少一次）。
		// 取消（用户 Cancel / 超时 / abort）后立即停止，结果状态为 cancelled，
		// 绝不以 success 收尾。
		if ctx.Err() != nil {
			cancelled := e.createCancelledResult(result, ctx.Err(), i, len(task.Script))
			e.reportSubTaskResult(ctx, task, cancelled)
			return cancelled, nil
		}

		// Update current step and total for screenshot reporting
		e.currentStep = i + 1
		e.currentTotal = len(task.Script)

		result.Step = i + 1
		result.Status = types.StatusRunning

		// Report progress
		// 子任务额外上报到 Redis Pub/Sub
		if isSubTask && parentExecutionID != "" {
			e.reportSubTaskProgress(ctx, parentExecutionID, subTaskIndex, i+1, len(task.Script), types.StatusRunning, fmt.Sprintf("executing: %s %s", action.Type, action.Description))
		}

		e.reportProgress(ctx, &types.ProgressReport{
			ExecutionID:     task.ExecutionID,
			ProgressCurrent: i + 1,
			ProgressTotal:   len(task.Script),
			Message:         fmt.Sprintf("executing: %s %s", action.Type, action.Description),
			Status:          types.StatusRunning,
		})

		// Phase 155 / LIVEVIEW-04：动作边界刷新叠加层（步骤/动作/脱敏变量）。
		e.refreshLiveContext(task.ExecutionID, &action, i+1, len(task.Script),
			types.StatusRunning, fmt.Sprintf("executing: %s %s", action.Type, action.Description), variables)

		// Phase 155 / LIVEVIEW-02/03：动作执行前闸门 —— 接管中/copilot 关键节点/
		// manual 会在此阻塞等人确认；返回 ctx 错误则走 RUNTIME-03 既有取消路径。
		if gateErr := e.liveActionGate(ctx, task.ExecutionID, &action, false); gateErr != nil {
			if ctx.Err() != nil {
				cancelled := e.createCancelledResult(result, ctx.Err(), i, len(task.Script))
				e.reportSubTaskResult(ctx, task, cancelled)
				return cancelled, nil
			}
			errorResult := e.createErrorResult(result, gateErr, i+1, len(task.Script))
			e.reportSubTaskResult(ctx, task, errorResult)
			return errorResult, nil
		}

		// Execute action
		if err := e.executeAction(ctx, page, pageMgr, &action, variables); err != nil {
			// RUNTIME-04：人工干预 abort → cancelled 终态
			var abortErr *types.AbortError
			if errors.As(err, &abortErr) {
				cancelled := e.createCancelledResult(result, err, i+1, len(task.Script))
				e.reportSubTaskResult(ctx, task, cancelled)
				return cancelled, nil
			}

			// Retry logic
			if action.Retry > 0 {
				// ============================================================
				// W7.1 / 缺陷 B：先判定「这个错误重试有没有意义」
				//
				// 生产实证（缺陷 B 的完整因果链）：
				//   goto 任务 step1 retry=2 > 0 → 进入本分支 → 先调 gate(force=true)
				//   → copilot 模式下 needWait 恒 true → 永久阻塞，10 分钟内无任何
				//   后续动作，最终 context deadline exceeded；页面停在 about:blank，
				//   screencast 一帧不发（后端 30 分钟 /liveview/frame 请求数 = 0）。
				//
				// 根因：force 闸门的设计意图是「动作**真的执行失败**时，copilot/
				// manual 停下来等人看一眼」；而 unsupported action type / 缺选择器 /
				// 缺 url 这类是**不可重试的配置错误**，重试 100 次也不会成功，
				// 触发人工闸门只会把人锁死在「等一个没有意义的确认」上。
				//
				// 处理：配置错误 → 不 gate、不空转重试，直接 failed 并给可读原因。
				// ============================================================
				if types.IsNonRetryable(err) {
					e.logger.Warn("non-retryable configuration error, skipping retry and human gate",
						logger.String("execution_id", task.ExecutionID),
						logger.String("action_type", string(action.Type)),
						logger.Int("declared_retry", action.Retry),
						logger.Int("step", i+1),
						logger.Err(err),
					)
					// 叠加层也要立刻反映失败原因（用户在看画面时能读到「为什么停」）。
					e.refreshLiveContext(task.ExecutionID, &action, i+1, len(task.Script),
						types.StatusFailed,
						fmt.Sprintf("配置错误（已跳过重试）：%s", err.Error()), variables)
					errorResult := e.createErrorResult(result, err, i+1, len(task.Script))
					e.reportSubTaskResult(ctx, task, errorResult)
					return errorResult, nil
				}

				// Phase 155 / LIVEVIEW-03：失败自动暂停 —— copilot/manual 在重试前
				// 停下等人看一眼；auto 不打断（否则自动任务会被人工拖死）。
				// 注意（W7.1 / 缺陷 B）：本闸门**只应**由可重试失败触发；
				// 配置错误在上面的 IsNonRetryable 分支已 fail-fast。
				if gateErr := e.liveActionGate(ctx, task.ExecutionID, &action, true); gateErr != nil {
					if ctx.Err() != nil {
						cancelled := e.createCancelledResult(result, ctx.Err(), i+1, len(task.Script))
						e.reportSubTaskResult(ctx, task, cancelled)
						return cancelled, nil
					}
					errorResult := e.createErrorResult(result, gateErr, i+1, len(task.Script))
					e.reportSubTaskResult(ctx, task, errorResult)
					return errorResult, nil
				}
				lastErr := err
				for retry := 0; retry <= action.Retry; retry++ {
					// RUNTIME-03：重试边界同样响应取消
					if ctx.Err() != nil {
						cancelled := e.createCancelledResult(result, ctx.Err(), i+1, len(task.Script))
						e.reportSubTaskResult(ctx, task, cancelled)
						return cancelled, nil
					}
					e.logger.Warn("action failed, retrying",
						logger.Int("retry", retry),
						logger.Int("max_retries", action.Retry),
						logger.Err(err),
					)
					time.Sleep(e.config.InitialDelay)

					if err = e.executeAction(ctx, page, pageMgr, &action, variables); err == nil {
						break
					}
					lastErr = err
				}
				err = lastErr
			}

			if err != nil {
				errorResult := e.createErrorResult(result, err, i+1, len(task.Script))
				e.reportSubTaskResult(ctx, task, errorResult)
				return errorResult, nil
			}
		}
	}

	// Task completed successfully
	result.Status = types.StatusSuccess
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// TIDY-03 配套：选择器明细契约信封行（空事件不发——验收 #2）
	if envelope := pageMgr.SelectorEnvelopeLine(); envelope != "" {
		result.Logs = append(result.Logs, envelope)
	}

	e.logger.Info("task executed successfully",
		logger.String("execution_id", task.ExecutionID),
		logger.Duration("duration", result.Duration),
	)

	// 上报子任务结果
	e.reportSubTaskResult(ctx, task, result)

	return result, nil
}

// extractSubTaskMeta 从任务变量中提取子任务元数据
func (e *Engine) extractSubTaskMeta(task *types.TaskMessage) (isSubTask bool, parentExecutionID string, subTaskIndex int) {
	if isSubTaskVal := task.Variables[types.VariableIsSubTask]; isSubTaskVal != nil {
		isSubTask = true
		if parentID, ok := task.Variables[types.VariableParentExecutionID].(string); ok {
			parentExecutionID = parentID
		}
		if idx, ok := task.Variables[types.VariableSubTaskIndex].(int); ok {
			subTaskIndex = idx
		}
	}
	return
}

// executeAction execute single action
func (e *Engine) executeAction(ctx context.Context, page browser.Page, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	// Phase 153 / RUNTIME-03：动作入口 ctx 取消检查点
	if err := ctx.Err(); err != nil {
		return err
	}

	e.logger.Debug("executing action",
		logger.String("type", string(action.Type)),
		logger.String("selector", action.Selector),
		logger.String("description", action.Description),
	)

	// Substitute variables（ENGINE-04：${var} + 嵌套 a.b[0].c，策略可配）
	params, err := e.substituteVariables(action.Params, variables)
	if err != nil {
		return err
	}

	// Execute by action type
	switch action.Type {
	case types.ActionNavigate:
		url, ok := params["url"].(string)
		if !ok || url == "" {
			// W7.1 / 缺陷 B：参数缺失属**配置错误**，重试不会成功 → 不可重试。
			return types.NewNonRetryableError("该步骤没有设置要打开的网址（缺少 url 参数）")
		}
		return pageMgr.Goto(url)

	case types.ActionClick:
		if action.Selector == "" {
			return types.NewNonRetryableError("该步骤没有设置点击目标（缺少选择器）")
		}
		return pageMgr.Click(action.Selector)

	case types.ActionFill:
		if action.Selector == "" {
			return types.NewNonRetryableError("填写步骤没有设置目标输入框（缺少选择器）")
		}
		value, ok := params["value"].(string)
		if !ok {
			return types.NewNonRetryableError("填写步骤没有设置要填入的内容（缺少 value 参数）")
		}
		return pageMgr.Fill(action.Selector, value)

	case types.ActionSelect:
		if action.Selector == "" {
			return types.NewNonRetryableError("下拉选择步骤没有设置目标（缺少选择器）")
		}
		value, ok := params["value"].(string)
		if !ok {
			return types.NewNonRetryableError("下拉选择步骤没有设置要选的项（缺少 value 参数）")
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
		// 获取截图数据
		data, err := pageMgr.Screenshot()
		if err != nil {
			return err
		}

		// 转换为 base64
		base64Data := base64.StdEncoding.EncodeToString(data)

		// 获取截图名称
		screenshotName := fmt.Sprintf("screenshot_%d", time.Now().UnixMilli())
		if name, ok := params["name"].(string); ok && name != "" {
			screenshotName = name
		}

		// 通过进度上报发送截图
		e.reportProgress(ctx, &types.ProgressReport{
			ExecutionID:     e.currentExecutionID,
			ProgressCurrent: e.currentStep,
			ProgressTotal:   e.currentTotal,
			Message:         fmt.Sprintf("截图: %s", screenshotName),
			Status:          types.StatusRunning,
			Screenshot:      base64Data,
		})

		return nil

	case types.ActionWaitFor:
		if action.Selector == "" {
			return types.NewNonRetryableError("等待元素步骤没有设置目标（缺少选择器）")
		}
		timeout := time.Duration(action.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = e.workerConfig.ActionTimeout
		}
		return pageMgr.WaitFor(action.Selector, timeout)

	case types.ActionLoop:
		// Loop action - iterate over data array and execute nested actions
		return e.executeLoop(ctx, page, pageMgr, action, variables)

	case types.ActionPause:
		// Pause action - wait for human input (for captcha handling)
		return e.executePause(ctx, pageMgr, action, variables)

	case types.ActionExtract:
		// Extract action - extract data from page
		return e.executeExtract(ctx, pageMgr, action, variables)

	case types.ActionScroll:
		// Scroll action - scroll the page
		return e.executeScroll(ctx, pageMgr, action, variables)

	case types.ActionAutoLogin:
		// Auto login action - use stored credentials or session
		return e.executeAutoLogin(ctx, page, pageMgr, action, variables)

	// ===== Phase 154 / ENGINE-02：差集 A（常量已声明、switch 原本无 case）=====
	case types.ActionUpload:
		return e.executeUpload(ctx, pageMgr, action, variables)

	case types.ActionDownload:
		return e.executeDownload(ctx, pageMgr, action, variables)

	case types.ActionEvaluate:
		return e.executeEvaluate(ctx, pageMgr, action, variables)

	case types.ActionClose:
		return e.executeClose(ctx, pageMgr, action, variables)

	case types.ActionCondition:
		return e.executeCondition(ctx, page, pageMgr, action, variables)

	// ===== Phase 154 / ENGINE-02：差集 B（常量与 case 双缺）=====
	case types.ActionAssert:
		return e.executeAssert(ctx, pageMgr, action, variables)

	case types.ActionHover:
		return e.executeHover(ctx, pageMgr, action, variables)

	case types.ActionDoubleClick:
		return e.executeDoubleClick(ctx, pageMgr, action, variables)

	case types.ActionRightClick:
		return e.executeRightClick(ctx, pageMgr, action, variables)

	case types.ActionDrag:
		return e.executeDrag(ctx, pageMgr, action, variables)

	case types.ActionKeyboard:
		return e.executeKeyboard(ctx, pageMgr, action, variables)

	case types.ActionSwitchTab:
		return e.executeSwitchTab(ctx, pageMgr, action, variables)

	case types.ActionIframe:
		return e.executeIframe(ctx, pageMgr, action, variables)

	case types.ActionScript:
		return e.executeScript(ctx, pageMgr, action, variables)

	default:
		// W7.1 / 缺陷 B：未知动作类型是**配置错误**（脚本写错 / 历史别名未归一化），
		// 重试永远不会成功，必须 fail-fast 而不是送进人工闸门空转。
		//
		// 文案面向一般员工：不出现 ActionType / switch / enum 这类内部说法，
		// 只说「这个步骤的类型不被支持」，并把原始 type 原样带上便于排查。
		return types.NewNonRetryableErrorf(
			"该步骤的类型「%s」不被支持，请在任务脚本里修正这一步", action.Type)
	}
}

// executeLoop execute loop action for batch processing
// 注意：混合模式下，Loop 动作应该由 Worker 的 TaskSplitter 拆分为子任务
// 这里保留简单的串行执行是为了向后兼容（当混合模式未启用时）
func (e *Engine) executeLoop(ctx context.Context, page browser.Page, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	// Get loop data source
	dataSource, ok := action.Params["dataSource"]
	if !ok {
		// W7.1 / 缺陷 B：脚本参数缺失 = 配置错误，重试无意义。
		return types.NewNonRetryableError("循环步骤没有设置数据来源（缺少 dataSource 参数）")
	}

	// Convert dataSource to array
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
		return types.NewNonRetryableError("循环步骤的数据来源格式不正确（应为对象数组）")
	}

	if len(dataArray) == 0 {
		e.logger.Info("loop data source is empty, skipping")
		return nil
	}

	// Get nested actions to execute for each iteration
	nestedActionsRaw, ok := action.Params["actions"]
	if !ok {
		return types.NewNonRetryableError("循环步骤没有设置要重复执行的子步骤（缺少 actions 参数）")
	}

	// Convert nested actions
	var nestedActions []types.Action
	switch v := nestedActionsRaw.(type) {
	case []interface{}:
		for _, item := range v {
			if actionMap, ok := item.(map[string]interface{}); ok {
				nestedActions = append(nestedActions, e.mapToAction(actionMap))
			}
		}
	case []types.Action:
		nestedActions = v
	}

	// Get item variable name (default: "item")
	itemName := "item"
	if name, ok := action.Params["itemVar"].(string); ok && name != "" {
		itemName = name
	}

	e.logger.Info("starting loop (serial execution for backward compatibility)",
		logger.String("item_var", itemName),
		logger.Int("iterations", len(dataArray)))

	// 串行执行（向后兼容）
	//
	// Phase 154 / ENGINE-04：每轮迭代使用独立 Loop 作用域 —— 上一轮的
	// item.* / index 不会泄漏到下一轮；Task 与 Session 层共享同一份数据。
	// 嵌套结构用 item 对象整体挂载（支持 ${item.a.b[0].c}），同时保留
	// 扁平字段直取（${fieldName}）的向后兼容。
	for i, item := range dataArray {
		e.logger.Debug("loop iteration",
			logger.Int("index", i),
			logger.Int("total", len(dataArray)),
		)

		variables.BreakLoopScope()
		if err := variables.Set(ScopeLoop, itemName, item); err != nil {
			return fmt.Errorf("loop iteration %d 注入 item 失败: %w", i, err)
		}
		if err := variables.Set(ScopeLoop, itemName+".index", i); err != nil {
			return fmt.Errorf("loop iteration %d 注入 index 失败: %w", i, err)
		}
		if err := variables.Set(ScopeLoop, "index", i); err != nil {
			return fmt.Errorf("loop iteration %d 注入 index 失败: %w", i, err)
		}
		for k, v := range item {
			if err := variables.Set(ScopeLoop, k, v); err != nil {
				return fmt.Errorf("loop iteration %d 注入字段 %s 失败: %w", i, k, err)
			}
		}

		// Execute nested actions
		for _, nestedAction := range nestedActions {
			if err := e.executeAction(ctx, page, pageMgr, &nestedAction, variables); err != nil {
				return fmt.Errorf("loop iteration %d failed: %w", i, err)
			}
		}
	}

	e.logger.Info("loop completed", logger.Int("iterations", len(dataArray)))
	return nil
}

// executePause execute pause action for human intervention
func (e *Engine) executePause(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	// Get pause message
	message := "等待人工输入..."
	if msg, ok := action.Params["message"].(string); ok && msg != "" {
		message = msg
	}

	// Get timeout (default from config)
	timeout := e.workerConfig.PauseTimeout
	if timeoutMs, ok := action.Params["timeout"].(int); ok && timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	e.logger.Info("pausing for human intervention",
		logger.String("message", message),
		logger.Duration("timeout", timeout),
	)

	// 1. Report pause status to backend via progress reporter
	// Include screenshot for UI display
	screenshot := ""
	if pageMgr != nil {
		if data, err := pageMgr.Screenshot(); err == nil && len(data) > 0 {
			// Convert to base64 for transmission
			screenshot = fmt.Sprintf("data:image/png;base64,%s", data)
		}
	}

	progressReport := &types.ProgressReport{
		ExecutionID: e.currentExecutionID,
		Status:      types.StatusPaused,
		Message:     message,
		Screenshot:  screenshot,
	}

	e.reportProgress(ctx, progressReport)

	// 2. Create input channel to receive user response
	userInputChan := make(chan map[string]interface{}, 1)

	// 3. Subscribe to Redis Pub/Sub for human input
	// Channel: worker:human-input:{workerId}:{executionId}
	if e.redisClient != nil {
		go e.subscribeForHumanInput(ctx, e.currentExecutionID, e.currentWorkerID, userInputChan)
	}

	// 4. Wait for user input or timeout
	select {
	case input := <-userInputChan:
		// Phase 153 / RUNTIME-04：按干预类型分派
		//   resume（含无 action 的纯输入）→ 输入写任务变量后继续
		//   skip  → 跳过当前等待与后续动作？（跳过 pause 本身，继续执行）
		//   abort → ErrExecutionAborted 哨兵，Execute 置 cancelled 终态
		action, _ := input["action"].(string)
		switch action {
		case "abort":
			e.logger.Info("human intervention abort received",
				logger.String("execution_id", e.currentExecutionID),
				logger.String("reason", fmt.Sprintf("%v", input["reason"])))
			return types.ErrExecutionAborted
		case "skip":
			e.logger.Info("human intervention skip received, continuing without input",
				logger.String("execution_id", e.currentExecutionID))
			return nil
		default: // resume / 未携带 action 的输入
			// User input received, add to variables（ENGINE-04：写任务层，
			// 敏感键在日志中脱敏）。
			for key, value := range input {
				if key == "action" || key == "reason" || key == "executionId" || key == "operator" {
					continue // 控制字段不写入业务变量
				}
				if err := variables.Set(ScopeTask, key, value); err != nil {
					return fmt.Errorf("人工输入回写变量 %s 失败: %w", key, err)
				}
				e.logger.Info("received user input",
					logger.String("key", key),
					logger.String("value", fmt.Sprintf("%v", MaskSensitive(key, value))))
			}
			return nil
		}

	case <-time.After(timeout):
		return fmt.Errorf("pause timeout: no user input received within %v", timeout)

	case <-ctx.Done():
		return ctx.Err()
	}
}

// subscribeForHumanInput subscribes to Redis Pub/Sub for human input
func (e *Engine) subscribeForHumanInput(ctx context.Context, executionID, workerID string, inputChan chan map[string]interface{}) {
	// Use Redis Pub/Sub to receive user input
	// Subscribe to: worker:human-input:{workerId}:{executionId}
	channel := fmt.Sprintf("worker:human-input:%s:%s", workerID, executionID)

	// Get Redis client from cache
	if e.redisClient == nil {
		e.logger.Warn("redis client not available for human input subscription")
		return
	}

	// Subscribe to channel
	pubsub := e.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Wait for message
	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		e.logger.Error("failed to receive human input", logger.Err(err))
		return
	}

	// Parse user input
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(msg.Payload), &input); err != nil {
		e.logger.Error("failed to parse human input", logger.Err(err))
		return
	}

	// Send to channel
	select {
	case inputChan <- input:
	case <-ctx.Done():
	}
}

// executeExtract execute extract action to get data from page
func (e *Engine) executeExtract(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	if action.Selector == "" {
		return types.NewNonRetryableError("提取步骤没有设置要提取的目标（缺少选择器）")
	}

	// Get extract target variable name
	targetVar := "extracted"
	if tv, ok := action.Params["targetVar"].(string); ok && tv != "" {
		targetVar = tv
	}

	// Get extract type (text, value, html, attribute)
	extractType := "text"
	if et, ok := action.Params["type"].(string); ok && et != "" {
		extractType = et
	}

	// W7.1 / 缺陷 A 同族（批量提取）：历史脚本用 extractType="multiple" +
	// fields[] 批量取多条（生产库实证：百度搜索结果提取测试 / 综合端到端 两个
	// goto 任务）。修复前它落到 default → 「提取方式 multiple 不被支持」，
	// 而这一步恰好是那两个任务的核心目的（提搜索结果）。
	if extractType == "multiple" || extractType == "list" || extractType == "table" {
		return e.executeExtractMultiple(ctx, pageMgr, action, variables, targetVar)
	}

	// Extract data based on type
	var extractedValue interface{}
	var err error

	switch extractType {
	case "text":
		extractedValue, err = pageMgr.GetText(action.Selector)
	case "value":
		extractedValue, err = pageMgr.GetValue(action.Selector)
	case "html":
		extractedValue, err = pageMgr.GetHTML(action.Selector)
	case "attribute":
		attrName, ok := action.Params["attribute"].(string)
		if !ok || attrName == "" {
			return types.NewNonRetryableError("提取属性时没有指定属性名（缺少 attribute 参数）")
		}
		extractedValue, err = pageMgr.GetAttribute(action.Selector, attrName)
	default:
		return types.NewNonRetryableErrorf(
			"提取步骤的提取方式「%s」不被支持（可选 text/value/html/attribute）", extractType)
	}

	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Store extracted value in variables（ENGINE-04：支持嵌套路径写入
	// 如 rows[0].name；默认写任务层）。
	scope := ScopeTask
	if sc, ok := action.Params["scope"].(string); ok {
		switch sc {
		case "session":
			scope = ScopeSession
		case "loop":
			scope = ScopeLoop
		}
	}
	if err := variables.Set(scope, targetVar, extractedValue); err != nil {
		return fmt.Errorf("extract 回写变量 %s 失败: %w", targetVar, err)
	}

	e.logger.Debug("data extracted",
		logger.String("target_var", targetVar),
		logger.String("scope", scope.String()),
		logger.String("value", fmt.Sprintf("%v", MaskSensitive(targetVar, extractedValue))),
	)

	return nil
}

// executeExtractMultiple 批量提取（extractType=multiple/list/table）。
//
// 历史脚本形状（生产库实证）：
//
//	{"type":"multiple","fields":[
//	   {"attribute":"text","name":"title","selector":"h3"},
//	   {"attribute":"href","name":"link","selector":"a"}],
//	 "maxItems":5,"outputVariable":"searchResults"}
//
// 语义：在 action.Selector 命中的**每个容器**里，按 fields 逐条取属性，
// 汇总成 [{title, link}, ...]（最多 maxItems 条）。fields 里的 attribute 既
// 支持 text/html/value/href 等 DOM 属性，也是「取哪个字段」的唯一键名。
//
// 为什么单独实现：worker 原有 extract 只有 text/value/html/attribute 四种
// 单值语义；multiple 是「一页多条记录」的语义，两者不可混用（否则会退化成
// 「只取第一条」的假成功）。
func (e *Engine) executeExtractMultiple(ctx context.Context, pageMgr *browser.PageManager,
	action *types.Action, variables *VariableScope, targetVar string) error {

	fieldsRaw := action.Params["fields"]
	fields, ok := fieldsRaw.([]interface{})
	if !ok || len(fields) == 0 {
		return types.NewNonRetryableError("批量提取步骤没有设置要提取的字段（缺少 fields 参数）")
	}

	// maxItems 缺省 20（避免无界抓取）。
	maxItems := 20
	if m, ok := action.Params["maxItems"].(int); ok && m > 0 {
		maxItems = m
	} else if f, ok := action.Params["maxItems"].(float64); ok && f > 0 {
		maxItems = int(f)
	}

	// 把字段定义序列化成脚本参数（Go → JS），由页面一次算完，避免逐字段往返。
	fieldSpec := make([]map[string]string, 0, len(fields))
	for _, f := range fields {
		m, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		sel, _ := m["selector"].(string)
		attr, _ := m["attribute"].(string)
		if name == "" {
			continue
		}
		if attr == "" {
			attr = "text"
		}
		fieldSpec = append(fieldSpec, map[string]string{"name": name, "selector": sel, "attribute": attr})
	}
	if len(fieldSpec) == 0 {
		return types.NewNonRetryableError("批量提取的 fields 定义不完整（每项需要 name）")
	}

	specJSON, err := json.Marshal(fieldSpec)
	if err != nil {
		return fmt.Errorf("批量提取字段定义序列化失败: %w", err)
	}
	containerJSON, _ := json.Marshal(action.Selector)

	script := fmt.Sprintf(`(function(){
		var spec = %s, containerSel = %s, max = %d;
		var containers = [];
		if (containerSel) {
			containers = Array.from(document.querySelectorAll(containerSel));
		}
		if (containers.length === 0) containers = [document];
		var out = [];
		for (var i = 0; i < containers.length && out.length < max; i++) {
			var root = containers[i];
			var item = {};
			var any = false;
			for (var j = 0; j < spec.length; j++) {
				var f = spec[j];
				var el = null;
				try { el = f.selector ? root.querySelector(f.selector) : root; } catch (e) { el = null; }
				if (!el) continue;
				var v = "";
				switch (f.attribute) {
				case "text": v = (el.textContent || "").trim(); break;
				case "html": v = el.innerHTML || ""; break;
				case "value": v = el.value || ""; break;
				default:
					var a = el.getAttribute(f.attribute);
					v = a === null ? "" : a;
				}
				item[f.name] = v;
				if (v !== "") any = true;
			}
			if (any) out.push(item);
		}
		return out;
	})()`, string(specJSON), string(containerJSON), maxItems)

	val, err := pageMgr.Evaluate(script)
	if err != nil {
		return fmt.Errorf("批量提取执行失败: %w", err)
	}

	scope := ScopeTask
	if sc, ok := action.Params["scope"].(string); ok {
		switch sc {
		case "session":
			scope = ScopeSession
		case "loop":
			scope = ScopeLoop
		}
	}
	if err := variables.Set(scope, targetVar, val); err != nil {
		return fmt.Errorf("批量提取回写变量 %s 失败: %w", targetVar, err)
	}

	count := 0
	if arr, ok := val.([]interface{}); ok {
		count = len(arr)
	}
	e.logger.Debug("批量提取完成",
		logger.String("target_var", targetVar),
		logger.Int("items", count),
		logger.Int("max_items", maxItems))
	return nil
}

// executeScroll execute scroll action
func (e *Engine) executeScroll(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	// Get scroll direction (down, up, toElement)
	direction := "down"
	if d, ok := action.Params["direction"].(string); ok && d != "" {
		direction = d
	}

	switch direction {
	case "down":
		pixels := 500
		if p, ok := action.Params["pixels"].(int); ok && p > 0 {
			pixels = p
		}
		return pageMgr.ScrollDown(pixels)
	case "up":
		pixels := 500
		if p, ok := action.Params["pixels"].(int); ok && p > 0 {
			pixels = p
		}
		return pageMgr.ScrollUp(pixels)
	case "toElement":
		if action.Selector == "" {
			return types.NewNonRetryableError("滚动到元素步骤没有设置目标（缺少选择器）")
		}
		return pageMgr.ScrollToElement(action.Selector)
	case "in", "element":
		// ENGINE-02：元素内滚动（滚动条在容器里，不滚动整页）
		if action.Selector == "" {
			return types.NewNonRetryableError("元素内滚动步骤没有设置容器（缺少选择器）")
		}
		dx, _ := action.Params["deltaX"].(int)
		dy, _ := action.Params["deltaY"].(int)
		if dy == 0 {
			if p, ok := action.Params["pixels"].(int); ok && p != 0 {
				dy = p
			}
		}
		return pageMgr.ScrollInElement(action.Selector, dx, dy)
	default:
		return types.NewNonRetryableErrorf(
			"滚动方向的设置「%s」不被支持（可选 down/up/toElement/in）", direction)
	}
}

// executeAutoLogin execute auto login action
func (e *Engine) executeAutoLogin(ctx context.Context, page browser.Page, pageMgr *browser.PageManager, action *types.Action, variables *VariableScope) error {
	// Get login URL from params
	loginURL, ok := action.Params["loginUrl"].(string)
	if !ok {
		return types.NewNonRetryableError("自动登录步骤没有设置登录页地址（缺少 loginUrl 参数）")
	}

	// Get session data from variables (passed from TaskMessage)
	sessionDataInterface, _, hasSession := variables.Lookup(types.VariableSessionData)
	credInterface, _, hasCred := variables.Lookup(types.VariableCredentials)

	// If we have session data (token/cookies), apply it directly
	if hasSession && sessionDataInterface != nil {
		e.logger.Info("using existing session for auto login")

		// Navigate to target URL first
		if err := pageMgr.Goto(loginURL); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}

		// Apply session data (cookies, localStorage)
		return e.applySessionData(ctx, page, sessionDataInterface, loginURL)
	}

	// If we have credentials, perform login
	if hasCred && credInterface != nil {
		e.logger.Info("performing auto login with stored credentials")

		// Navigate to login page
		if err := pageMgr.Goto(loginURL); err != nil {
			return fmt.Errorf("navigation to login page failed: %w", err)
		}

		// Get credential data
		credMap, ok := credInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid credential format")
		}

		username, _ := credMap["username"].(string)
		password, _ := credMap["password"].(string)

		// Get selectors for login form
		usernameSelector, _ := action.Params["usernameSelector"].(string)
		passwordSelector, _ := action.Params["passwordSelector"].(string)
		submitSelector, _ := action.Params["submitSelector"].(string)

		// Default selectors if not provided
		if usernameSelector == "" {
			usernameSelector = "input[name='username'], input[type='text'], #username"
		}
		if passwordSelector == "" {
			passwordSelector = "input[name='password'], input[type='password'], #password"
		}
		if submitSelector == "" {
			submitSelector = "button[type='submit'], input[type='submit'], .login-btn"
		}

		// Wait for login form to be ready
		time.Sleep(e.workerConfig.AutoLoginWaitDelay)

		// Fill username
		if err := pageMgr.Fill(usernameSelector, username); err != nil {
			return fmt.Errorf("fill username failed: %w", err)
		}

		// Fill password
		if err := pageMgr.Fill(passwordSelector, password); err != nil {
			return fmt.Errorf("fill password failed: %w", err)
		}

		// Wait a bit
		time.Sleep(e.workerConfig.AutoLoginFillDelay)

		// Click submit button
		if err := pageMgr.Click(submitSelector); err != nil {
			return fmt.Errorf("click submit button failed: %w", err)
		}

		// Wait for navigation after login
		time.Sleep(e.workerConfig.AutoLoginNavDelay)

		// Check if login was successful by looking for login failure indicators
		if errorMsgSelector, ok := action.Params["errorSelector"].(string); ok {
			if text, err := pageMgr.GetText(errorMsgSelector); err == nil && text != "" {
				return fmt.Errorf("login failed: %s", text)
			}
		}

		e.logger.Info("auto login completed successfully")
		return nil
	}

	return fmt.Errorf("no session data or credentials available for auto login")
}

// applySessionData 把会话数据真实注入页面（Phase 153 / RUNTIME-02）。
//
// 注入通道（经 Page.RawPage() 消费 browser.Manager 冻结契约，不改 browser/*）：
//  1. cookies   → rod Page.SetCookies（domain 缺省按 targetURL host 补齐）
//  2. localStorage / token → EvalOnNewDocument：目标页每次文档创建前写入
//     localStorage（键 + 常见 token 键），使 SPA 首载即登录态
//  3. Authorization 头 → 同一脚本 patch window.fetch，为同源请求附加
//     Authorization: Bearer <accessToken>
//
// 调用时机：Acquire 之后、导航之前 —— cookie 在首个请求即生效。
func (e *Engine) applySessionData(ctx context.Context, page browser.Page, sessionDataInterface interface{}, targetURL string) error {
	sessionDataBytes, err := json.Marshal(sessionDataInterface)
	if err != nil {
		return err
	}

	var sessionData types.SessionData
	if err := json.Unmarshal(sessionDataBytes, &sessionData); err != nil {
		return err
	}

	raw := page.RawPage()
	if raw == nil {
		return fmt.Errorf("RawPage 不可用，无法注入会话")
	}

	// 1. cookies：缺省 domain/path 用 targetURL host 补齐
	if len(sessionData.Cookies) > 0 {
		cookieParams := make([]*proto.NetworkCookieParam, 0, len(sessionData.Cookies))
		for _, cookie := range sessionData.Cookies {
			param := &proto.NetworkCookieParam{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Path:     cookie.Path,
				Secure:   cookie.Secure,
				HTTPOnly: cookie.HTTPOnly,
			}
			if param.Path == "" {
				param.Path = "/"
			}
			if cookie.Domain != "" {
				param.Domain = cookie.Domain
			} else if targetURL != "" {
				param.URL = targetURL // rod 按 URL 推导 domain
			}
			if cookie.Expires != nil {
				exp := float64(cookie.Expires.Unix())
				param.Expires = proto.TimeSinceEpoch(exp)
			}
			cookieParams = append(cookieParams, param)
			e.logger.Debug("setting cookie",
				logger.String("name", cookie.Name),
				logger.String("domain", cookie.Domain))
		}
		if err := raw.SetCookies(cookieParams); err != nil {
			return fmt.Errorf("注入 cookies 失败: %w", err)
		}
	}

	// 2 + 3. localStorage / token / fetch 头 patch：EvalOnNewDocument
	sessionItems := sessionData.SessionData
	accessToken := sessionData.AccessToken
	if sessionItems != nil {
		if token, ok := sessionItems["accessToken"].(string); ok && accessToken == "" {
			accessToken = token
		}
		if token, ok := sessionItems["token"].(string); ok && accessToken == "" {
			accessToken = token
		}
	}

	var jsParts []string
	if sessionItems != nil {
		for key, value := range sessionItems {
			rawKey := strings.ReplaceAll(key, "'", "\\'")
			valueJSON, err := json.Marshal(value)
			if err != nil {
				continue
			}
			jsParts = append(jsParts, fmt.Sprintf(
				"try{localStorage.setItem('%s', %s);}catch(e){}", rawKey, string(valueJSON)))
		}
	}
	if accessToken != "" {
		// 常见 token 键兜底写入（各内网 SPA 读取习惯不一）
		tokenJSON, _ := json.Marshal(accessToken)
		jsParts = append(jsParts,
			fmt.Sprintf("try{localStorage.setItem('token', %s);localStorage.setItem('accessToken', %s);}catch(e){}",
				string(tokenJSON), string(tokenJSON)))
		// 同源 API 请求附加 Authorization 头
		jsParts = append(jsParts, fmt.Sprintf(
			`try{(function(){var o=window.fetch;window.fetch=function(){var r=o.apply(this,arguments);`+
				`try{var u=(typeof arguments[0]==='string')?arguments[0]:(arguments[0]&&arguments[0].url)||'';`+
				`if(u && (u.startsWith('/')||u.indexOf(location.host)>-1)){var h=new Headers((arguments[0]&&arguments[0].headers)||{});`+
				`if(!h.has('Authorization')){h.set('Authorization','Bearer %s');`+
				`if(arguments[1]){arguments[1].headers=h;}}}}catch(e){}return r;};})();}catch(e){}`, accessToken))
		e.logger.Debug("setting access token")
	}
	if len(jsParts) > 0 {
		script := strings.Join(jsParts, "\n")
		if _, err := raw.EvalOnNewDocument(script); err != nil {
			return fmt.Errorf("EvalOnNewDocument 注入失败: %w", err)
		}
		// 已处于目标页时补一次即时注入（EvalOnNewDocument 只影响后续文档）
		_, _ = raw.Eval(script)
	}

	e.logger.Info("session data injected",
		logger.Int("cookies", len(sessionData.Cookies)),
		logger.Int("storage_items", len(sessionItems)),
		logger.String("has_token", fmt.Sprintf("%v", accessToken != "")))
	return nil
}

// ============================================================================
// W7.1 / 缺陷 A：动作类型别名归一化（唯一入口，不在各 case 里分散判断）
//
// 生产实证（v1.56 W7 修复后用户实测仍 0 帧）：
//   库里 6 个任务（全部 2026-03-10 创建的测试数据）第一步都是 type="goto"，
//   而 worker 的 ActionType 常量表只有 "navigate"（types.go:70）。mapToAction
//   原样把 "goto" 塞进 action.Type → executeAction switch 无 case → default
//   立即返回 "unsupported action type: goto"。
//
//   生产日志铁证：liveview attach(00:12:00.009) → gate waiting(00:12:00.094)
//   间隔仅 85ms —— 真实导航失败不可能这么快，证明动作压根没执行。
//
// 别名表取值依据（先取证再定表，不凭想象扩）：
//   · scripts/ops/rpa_action_type_census 普查生产库：19 个任务 / 12 种 type，
//     非规范写法只有 "goto"（6 步）→ ALIAS_SET=goto
//   · 前端 RPA_ACTION_TYPES（src/types/rpa.ts）只有规范写法，无 goto/url
//   · "url" 是历史脚本把 URL 当类型名的惯用写法（后端 extractURLFromScript
//     与 convertToWorkerAction 都允许把 URL 放在 value 字段，说明存量脚本里
//     确有「URL 就是这一步主角」的形状），任务书也明确要求覆盖
//   · goTo / GOTO 等大小写变体：统一按**大小写不敏感**查表覆盖，避免每种
//     大小写各写一行（表会随历史数据增长而失控）
// ============================================================================

// actionTypeAliases 历史/前端惯用写法 → 规范 ActionType（唯一真相已迁到 types 包）。
//
// 保留这个薄封装是为了让本文件的调用点语义自解释（executor 只做「映射」），
// 而**真正的别名表与归一化实现**在 types.ActionTypeAliases / types.NormalizeAction：
// 动作有两条进入路径（顶层 script 走 json.Unmarshal、嵌套动作走本文件
// mapToAction），两条都必须在「Action.Type 赋值处」归一化，因此表不能只放在
// 本文件里 —— 否则 rpa-worker 的 TaskMessage.UnmarshalJSON 无法复用（包依赖方向
// 是 executor → types，反过来 import 会成环）。
var actionTypeAliases = types.ActionTypeAliases

// normalizeActionType 把原始 type 字符串归一化成规范 ActionType（委托 types 包）。
//
// 返回 (归一化后的类型, 命中的别名原文)。命中别名时调用方负责打 WARN，
// 便于从日志里继续发现历史数据（别名是兜底，数据迁移才是根治）。
//
// 未命中别名时原样返回（**不做**任何近似猜测）：未知类型必须继续走
// executeAction 的 default 分支给出可读错误，不能被静默吞掉。
func normalizeActionType(raw string) (types.ActionType, string) {
	return types.NormalizeActionType(raw)
}

// mapToAction convert map to Action struct
// 后端已经发送正确格式的数据，这里只需简单映射字段
//
// W7.1 / 缺陷 A：type 的别名归一化在**这里一次性完成**（action.Type 赋值处），
// 不在每个 case 里分散判断 —— 否则 loop/condition 里的嵌套动作（同样经本函数
// 转换）会漏掉，而它们恰好是最容易藏历史写法的位置。
func (e *Engine) mapToAction(m map[string]interface{}) types.Action {
	action := types.Action{}
	if id, ok := m["id"].(string); ok {
		action.ID = id
	}
	if actionType, ok := m["type"].(string); ok {
		normalized, alias := normalizeActionType(actionType)
		if alias != "" && alias != string(normalized) {
			e.logNormalizedActionType(actionType, normalized, action.ID)
		}
		action.Type = normalized
	}
	if desc, ok := m["description"].(string); ok {
		action.Description = desc
	}
	if selector, ok := m["selector"].(string); ok {
		action.Selector = selector
	}
	if value, ok := m["value"].(string); ok {
		action.Value = value
	}
	if timeout, ok := m["timeout"].(int); ok {
		action.Timeout = timeout
	}
	if retry, ok := m["retry"].(int); ok {
		action.Retry = retry
	}

	// Params 已经由后端正确设置，直接使用
	if params, ok := m["params"].(map[string]interface{}); ok {
		action.Params = params
	}

	// W7.1 / 缺陷 A（第二层）：navigate 的 URL 键位提升 + 别名归一（委托 types）。
	//
	// 为什么必须做：生产库 6 个 goto 任务的 URL 都在 **value** 字段、params 里
	// 只有 waitUntil（普查实证：navigate 动作的 URL 键位 params.url=6 / value=7，
	// 两种形状都在存量里）。后端 convertToWorkerAction 只在 type=="navigate"
	// 时把 value 提升到 params.url —— goto 走不到那句。若只在 worker 侧把类型
	// 归一化成 navigate 而不补这次提升，错误会从 "unsupported action type"
	// 变成 "missing url parameter"，任务照样跑不通（只是换了个报错）。
	//
	// NormalizeAction 是 types 包的**唯一入口**，与 TaskMessage.UnmarshalJSON
	// （顶层 script 路径）共用同一份实现与别名表 —— 两条路径不会漂移。
	types.NormalizeAction(&action)

	return action
}

// logNormalizedActionType 别名命中的 WARN（nil 安全：nestedActionsFrom 里用
// `var tmp Engine` 走本函数，logger 为 nil）。
func (e *Engine) logNormalizedActionType(raw string, normalized types.ActionType, actionID string) {
	if e == nil || e.logger == nil {
		return
	}
	e.logger.Warn("normalized action type "+raw+" -> "+string(normalized),
		logger.String("raw_type", raw),
		logger.String("normalized_type", string(normalized)),
		logger.String("action_id", actionID),
	)
}

// substituteVariables substitute variables（ENGINE-04）。
//
// 递归处理参数树，支持 ${var} / ${a.b[0].c} / ${a.b[0].c:default}；
// 未定义策略由 executor.undefined_variable_policy 决定。
func (e *Engine) substituteVariables(params map[string]interface{}, variables *VariableScope) (map[string]interface{}, error) {
	if params == nil {
		return map[string]interface{}{}, nil
	}
	policy := config.UndefinedVariableEmpty
	if e.config != nil && e.config.UndefinedVariablePolicy != "" {
		policy = e.config.UndefinedVariablePolicy
	}
	resolved, err := ResolveValue(params, variables.Flatten(), policy)
	if err != nil {
		return nil, err
	}
	out, ok := resolved.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("参数求值结果不是对象: %T", resolved)
	}
	return out, nil
}

// replaceVariables replace variables in string（供单值模板使用，
// 语义与 substituteVariables 一致）。
func (e *Engine) replaceVariables(s string, variables *VariableScope) (string, error) {
	policy := config.UndefinedVariableEmpty
	if e.config != nil && e.config.UndefinedVariablePolicy != "" {
		policy = e.config.UndefinedVariablePolicy
	}
	return ResolveTemplate(s, variables.Flatten(), policy)
}

// createCancelledResult 取消/中止结果（Phase 153 / RUNTIME-03）。
// 状态恒为 cancelled（绝不以 success 收尾），错误信息保留取消原因。
func (e *Engine) createCancelledResult(result *types.ExecutionResult, cause error, step, total int) *types.ExecutionResult {
	result.Status = types.StatusCancelled
	result.ErrorMessage = "execution cancelled: " + cause.Error()
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Step = step
	result.Total = total

	e.logger.Info("task execution cancelled",
		logger.String("execution_id", result.ExecutionID),
		logger.Int("step", step),
		logger.Int("total", total),
		logger.Err(cause),
	)
	return result
}

// createErrorResult create error result
func (e *Engine) createErrorResult(result *types.ExecutionResult, err error, step, total int) *types.ExecutionResult {
	result.Status = types.StatusFailed
	result.ErrorMessage = err.Error()
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Step = step
	result.Total = total

	e.logger.Error("task execution failed",
		logger.String("execution_id", result.ExecutionID),
		logger.Int("step", step),
		logger.Int("total", total),
		logger.Err(err),
	)

	return result
}

// reportSubTaskProgress 上报子任务进度到 Redis Pub/Sub
func (e *Engine) reportSubTaskProgress(ctx context.Context, parentExecutionID string, subTaskIndex, step, total int, status types.ExecutionStatus, message string) {
	if e.redisClient == nil {
		return
	}

	progressMsg := map[string]interface{}{
		"subTaskIndex": subTaskIndex,
		"status":       status,
		"step":         step,
		"total":        total,
		"message":      message,
		"timestamp":    time.Now(),
	}

	progressChannel := fmt.Sprintf("subtask:progress:%s", parentExecutionID)
	data, err := json.Marshal(progressMsg)
	if err != nil {
		e.logger.Warn("failed to marshal subtask progress", logger.Err(err))
		return
	}

	if err := e.redisClient.Publish(ctx, progressChannel, data).Err(); err != nil {
		e.logger.Warn("failed to publish subtask progress", logger.Err(err))
	}
}

// reportSubTaskResult 上报子任务执行结果到 Redis Pub/Sub
func (e *Engine) reportSubTaskResult(ctx context.Context, task *types.TaskMessage, result *types.ExecutionResult) {
	// 检查是否是子任务
	isSubTask, parentExecutionID, subTaskIndex := e.extractSubTaskMeta(task)

	if !isSubTask || parentExecutionID == "" || e.redisClient == nil {
		return
	}

	subTaskResult := &types.SubTaskResult{
		SubTaskIndex: subTaskIndex,
		Status:       result.Status,
		ErrorMessage: result.ErrorMessage,
		StartedAt:    result.StartedAt,
		CompletedAt:  result.CompletedAt,
	}

	resultChannel := fmt.Sprintf("subtask:result:%s", parentExecutionID)
	data, err := json.Marshal(subTaskResult)
	if err != nil {
		e.logger.Warn("failed to marshal subtask result", logger.Err(err))
		return
	}

	if err := e.redisClient.Publish(ctx, resultChannel, data).Err(); err != nil {
		e.logger.Warn("failed to publish subtask result", logger.Err(err))
	}

	e.logger.Debug("subtask result reported",
		logger.String("parent_execution_id", parentExecutionID),
		logger.Int("subtask_index", subTaskIndex),
		logger.String("status", string(result.Status)))
}

package executor

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/ruoyi-next/rpa-worker/internal/browser"
    "github.com/ruoyi-next/rpa-worker/internal/communication"
    "github.com/ruoyi-next/rpa-worker/internal/config"
    "github.com/ruoyi-next/rpa-worker/internal/logger"
    "github.com/ruoyi-next/rpa-worker/internal/types"
)

// Engine execution engine
type Engine struct {
    config              *config.ExecutorConfig
    workerConfig        *config.WorkerConfig
    browserPool         *browser.Pool
    progressReporter    *communication.ProgressReporter
    logger              logger.Logger
    redisClient         *redis.Client
    currentExecutionID  string
    currentWorkerID     string
    currentStep         int
    currentTotal        int
    loopConcurrency     int // 循环内并发执行数量
}

// NewEngine create execution engine
func NewEngine(
    cfg *config.ExecutorConfig,
    workerCfg *config.WorkerConfig,
    browserPool *browser.Pool,
    progressReporter *communication.ProgressReporter,
    log logger.Logger,
    redisClient *redis.Client,
) *Engine {
    return &Engine{
        config:           cfg,
        workerConfig:     workerCfg,
        browserPool:      browserPool,
        progressReporter: progressReporter,
        logger:           log,
        redisClient:      redisClient,
        loopConcurrency:  workerCfg.LoopConcurrency, // 从配置中获取循环并发数
    }
}

// Execute execute task
func (e *Engine) Execute(ctx context.Context, task *types.TaskMessage, workerID string) (*types.ExecutionResult, error) {
    startTime := time.Now()

    // Set current execution context for pause action
    e.currentExecutionID = task.ExecutionID
    e.currentWorkerID = workerID

    e.logger.Info("start executing task",
        logger.String("execution_id", task.ExecutionID),
        logger.String("task_id", task.TaskID),
        logger.String("worker_id", workerID),
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

    // Acquire browser page
    pooledPage, err := e.browserPool.Acquire(ctx)
    if err != nil {
        return e.createErrorResult(result, err, 0, len(task.Script)), nil
    }
    defer e.browserPool.Release(pooledPage)

    // Create page manager
    pageMgr := browser.NewPageManager(pooledPage.Page(), e.logger)

    // Initialize variable context
    variables := make(map[string]interface{})
    for k, v := range task.Variables {
        variables[k] = v
    }

    // Navigate to target URL
    if task.TargetURL != "" {
        if err := pageMgr.Goto(task.TargetURL); err != nil {
            return e.createErrorResult(result, err, 0, len(task.Script)), nil
        }
    }

    // Execute script actions
    for i, action := range task.Script {
        // Update current step and total for screenshot reporting
        e.currentStep = i + 1
        e.currentTotal = len(task.Script)

        result.Step = i + 1
        result.Status = types.StatusRunning

        // Report progress
        if err := e.progressReporter.ReportProgress(ctx, &types.ProgressReport{
            ExecutionID:     task.ExecutionID,
            ProgressCurrent: i + 1,
            ProgressTotal:   len(task.Script),
            Message:         fmt.Sprintf("executing: %s %s", action.Type, action.Description),
            Status:          types.StatusRunning,
        }); err != nil {
            e.logger.Warn("failed to report progress", logger.Err(err))
        }

        // Execute action
        if err := e.executeAction(ctx, pageMgr, &action, variables); err != nil {
            // Retry logic
            if action.Retry > 0 {
                lastErr := err
                for retry := 0; retry <= action.Retry; retry++ {
                    e.logger.Warn("action failed, retrying",
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
                return e.createErrorResult(result, err, i+1, len(task.Script)), nil
            }
        }
    }

    // Task completed successfully
    result.Status = types.StatusSuccess
    result.CompletedAt = time.Now()
    result.Duration = result.CompletedAt.Sub(startTime)

    e.logger.Info("task executed successfully",
        logger.String("execution_id", task.ExecutionID),
        logger.Duration("duration", result.Duration),
    )

    return result, nil
}

// executeAction execute single action
func (e *Engine) executeAction(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
    e.logger.Debug("executing action",
        logger.String("type", string(action.Type)),
        logger.String("selector", action.Selector),
        logger.String("description", action.Description),
    )

    // Substitute variables
    params := e.substituteVariables(action.Params, variables)

    // Execute by action type
    switch action.Type {
    case types.ActionNavigate:
        url, ok := params["url"].(string)
        if !ok {
            return fmt.Errorf("missing url parameter")
        }
        return pageMgr.Goto(url)

    case types.ActionClick:
        if action.Selector == "" {
            return fmt.Errorf("missing selector")
        }
        return pageMgr.Click(action.Selector)

    case types.ActionFill:
        if action.Selector == "" {
            return fmt.Errorf("missing selector")
        }
        value, ok := params["value"].(string)
        if !ok {
            return fmt.Errorf("missing value parameter")
        }
        return pageMgr.Fill(action.Selector, value)

    case types.ActionSelect:
        if action.Selector == "" {
            return fmt.Errorf("missing selector")
        }
        value, ok := params["value"].(string)
        if !ok {
            return fmt.Errorf("missing value parameter")
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
        if e.progressReporter != nil {
            if err := e.progressReporter.ReportProgress(ctx, &types.ProgressReport{
                ExecutionID:     e.currentExecutionID,
                ProgressCurrent: e.currentStep,
                ProgressTotal:   e.currentTotal,
                Message:         fmt.Sprintf("截图: %s", screenshotName),
                Status:          types.StatusRunning,
                Screenshot:      base64Data,
            }); err != nil {
                e.logger.Warn("failed to report screenshot", logger.Err(err))
            }
        }

        return nil

    case types.ActionWaitFor:
        if action.Selector == "" {
            return fmt.Errorf("missing selector")
        }
        timeout := time.Duration(action.Timeout) * time.Millisecond
        if timeout == 0 {
            timeout = e.workerConfig.ActionTimeout
        }
        return pageMgr.WaitFor(action.Selector, timeout)

    case types.ActionLoop:
        // Loop action - iterate over data array and execute nested actions
        return e.executeLoop(ctx, pageMgr, action, variables)

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
        return e.executeAutoLogin(ctx, pageMgr, action, variables)

    default:
        return fmt.Errorf("unsupported action type: %s", action.Type)
    }
}

// executeLoop execute loop action for batch processing
// 支持并发执行：当 loopConcurrency > 0 时，多个迭代并行执行
func (e *Engine) executeLoop(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
    // Get loop data source
    dataSource, ok := action.Params["dataSource"]
    if !ok {
        return fmt.Errorf("loop action requires dataSource parameter")
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
        return fmt.Errorf("dataSource must be an array of objects")
    }

    if len(dataArray) == 0 {
        e.logger.Info("loop data source is empty, skipping")
        return nil
    }

    // Get nested actions to execute for each iteration
    nestedActionsRaw, ok := action.Params["actions"]
    if !ok {
        return fmt.Errorf("loop action requires actions parameter")
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

    // 检查是否需要并发执行
    // 可以通过 action.Params["concurrency"] 覆盖全局配置
    concurrency := e.loopConcurrency
    if concurrencyOverride, ok := action.Params["concurrency"].(int); ok && concurrencyOverride >= 0 {
        concurrency = concurrencyOverride
    }

    e.logger.Info("starting loop",
        logger.String("item_var", itemName),
        logger.Int("iterations", len(dataArray)),
        logger.Int("concurrency", concurrency),
    )

    // 如果并发数为0或1，或者数据量小，使用串行执行
    if concurrency <= 1 || len(dataArray) <= 1 {
        return e.executeLoopSerial(ctx, pageMgr, dataArray, nestedActions, itemName, variables)
    }

    // 并发执行
    return e.executeLoopParallel(ctx, dataArray, nestedActions, itemName, variables, concurrency)
}

// executeLoopSerial 串行执行循环（原有逻辑）
func (e *Engine) executeLoopSerial(ctx context.Context, pageMgr *browser.PageManager, dataArray []map[string]interface{}, nestedActions []types.Action, itemName string, variables map[string]interface{}) error {
    for i, item := range dataArray {
        e.logger.Debug("loop iteration (serial)",
            logger.Int("index", i),
            logger.Int("total", len(dataArray)),
        )

        // Create new variable context for this iteration
        iterationVars := make(map[string]interface{})
        for k, v := range variables {
            iterationVars[k] = v
        }

        // Add current item to variables (can access with ${item.fieldName})
        for k, v := range item {
            iterationVars[itemName+"."+k] = v
            // Also add direct access with field name
            iterationVars[k] = v
        }

        // Add iteration index
        iterationVars[itemName+".index"] = i
        iterationVars["index"] = i

        // Execute nested actions
        for _, nestedAction := range nestedActions {
            if err := e.executeAction(ctx, pageMgr, &nestedAction, iterationVars); err != nil {
                return fmt.Errorf("loop iteration %d failed: %w", i, err)
            }
        }
    }

    e.logger.Info("loop completed (serial)", logger.Int("iterations", len(dataArray)))
    return nil
}

// executeLoopParallel 并发执行循环
func (e *Engine) executeLoopParallel(ctx context.Context, dataArray []map[string]interface{}, nestedActions []types.Action, itemName string, variables map[string]interface{}, concurrency int) error {
    totalIterations := len(dataArray)

    // 创建错误收集通道
    errChan := make(chan error, totalIterations)
    doneChan := make(chan int, totalIterations)

    // 创建信号量控制并发数
    sem := make(chan struct{}, concurrency)

    // 用于等待所有 goroutine 完成
    var wg sync.WaitGroup

    // 启动多个 worker 并发执行
    for i, item := range dataArray {
        wg.Add(1)

        go func(index int, itemData map[string]interface{}) {
            defer wg.Done()

            // 获取信号量
            sem <- struct{}{}
            defer func() { <-sem }()

            // 为每个迭代获取独立的浏览器页面
            pooledPage, err := e.browserPool.Acquire(ctx)
            if err != nil {
                e.logger.Error("failed to acquire browser page for iteration",
                    logger.Int("index", index),
                    logger.Err(err))
                errChan <- fmt.Errorf("iteration %d: failed to acquire browser: %w", index, err)
                return
            }
            defer e.browserPool.Release(pooledPage)

            // 创建页面管理器
            pageMgr := browser.NewPageManager(pooledPage.Page(), e.logger)

            // 创建迭代变量上下文
            iterationVars := make(map[string]interface{})
            for k, v := range variables {
                iterationVars[k] = v
            }

            // 添加当前项到变量
            for k, v := range itemData {
                iterationVars[itemName+"."+k] = v
                iterationVars[k] = v
            }

            // 添加索引
            iterationVars[itemName+".index"] = index
            iterationVars["index"] = index

            // 如果第一个动作是导航，先导航到目标URL
            if len(nestedActions) > 0 && nestedActions[0].Type == types.ActionNavigate {
                if err := e.executeAction(ctx, pageMgr, &nestedActions[0], iterationVars); err != nil {
                    e.logger.Error("navigation failed in loop iteration",
                        logger.Int("index", index),
                        logger.Err(err))
                    errChan <- fmt.Errorf("iteration %d: navigation failed: %w", index, err)
                    return
                }
                // 跳过第一个导航动作，已执行
                nestedActions = nestedActions[1:]
            }

            // 执行嵌套动作
            for _, nestedAction := range nestedActions {
                if err := e.executeAction(ctx, pageMgr, &nestedAction, iterationVars); err != nil {
                    e.logger.Error("loop iteration failed",
                        logger.Int("index", index),
                        logger.String("action", string(nestedAction.Type)),
                        logger.Err(err))
                    errChan <- fmt.Errorf("iteration %d failed: %w", index, err)
                    return
                }
            }

            // 报告进度
            completed := index + 1
            if e.progressReporter != nil {
                if err := e.progressReporter.ReportProgress(ctx, &types.ProgressReport{
                    ExecutionID:     e.currentExecutionID,
                    ProgressCurrent: completed,
                    ProgressTotal:   totalIterations,
                    Message:         fmt.Sprintf("循环进度: %d/%d", completed, totalIterations),
                    Status:          types.StatusRunning,
                }); err != nil {
                    e.logger.Warn("failed to report loop progress", logger.Err(err))
                }
            }

            e.logger.Debug("loop iteration completed",
                logger.Int("index", index),
                logger.Int("completed", completed),
                logger.Int("total", totalIterations))

            doneChan <- index
        }(i, item)
    }

    // 等待所有 goroutine 完成
    wg.Wait()
    close(errChan)
    close(doneChan)

    // 收集错误
    var errors []error
    for err := range errChan {
        errors = append(errors, err)
    }

    // 统计完成的迭代数
    completedCount := len(doneChan)
    for range doneChan {
        completedCount++
    }

    e.logger.Info("loop completed (parallel)",
        logger.Int("total_iterations", totalIterations),
        logger.Int("completed", completedCount),
        logger.Int("errors", len(errors)),
        logger.Int("concurrency", concurrency))

    // 如果有错误，返回第一个错误
    if len(errors) > 0 {
        return errors[0]
    }

    return nil
}

// executePause execute pause action for human intervention
func (e *Engine) executePause(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
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

	if e.progressReporter != nil {
		if err := e.progressReporter.ReportProgress(ctx, progressReport); err != nil {
			e.logger.Warn("failed to report pause status", logger.Err(err))
		}
	}

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
		// User input received, add to variables
		for key, value := range input {
			variables[key] = value
			e.logger.Info("received user input",
				logger.String("key", key),
				logger.String("value", fmt.Sprintf("%v", value)))
		}
		return nil

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
func (e *Engine) executeExtract(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
    if action.Selector == "" {
        return fmt.Errorf("extract action requires selector")
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
            return fmt.Errorf("attribute extract type requires attribute parameter")
        }
        extractedValue, err = pageMgr.GetAttribute(action.Selector, attrName)
    default:
        return fmt.Errorf("unsupported extract type: %s", extractType)
    }

    if err != nil {
        return fmt.Errorf("extract failed: %w", err)
    }

    // Store extracted value in variables
    variables[targetVar] = extractedValue

    e.logger.Debug("data extracted",
        logger.String("target_var", targetVar),
        logger.String("value", fmt.Sprintf("%v", extractedValue)),
    )

    return nil
}

// executeScroll execute scroll action
func (e *Engine) executeScroll(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
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
            return fmt.Errorf("scroll to element requires selector")
        }
        return pageMgr.ScrollToElement(action.Selector)
    default:
        return fmt.Errorf("unsupported scroll direction: %s", direction)
    }
}

// executeAutoLogin execute auto login action
func (e *Engine) executeAutoLogin(ctx context.Context, pageMgr *browser.PageManager, action *types.Action, variables map[string]interface{}) error {
	// Get login URL from params
	loginURL, ok := action.Params["loginUrl"].(string)
	if !ok {
		return fmt.Errorf("autologin action requires loginUrl parameter")
	}

	// Get session data from variables (passed from TaskMessage)
	sessionDataInterface, hasSession := variables["__sessionData"]
	credInterface, hasCred := variables["__credentials"]

	// If we have session data (token/cookies), apply it directly
	if hasSession && sessionDataInterface != nil {
		e.logger.Info("using existing session for auto login")

		// Navigate to target URL first
		if err := pageMgr.Goto(loginURL); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}

		// Apply session data (cookies, localStorage)
		return e.applySessionData(ctx, pageMgr, sessionDataInterface)
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

// applySessionData applies stored session data (cookies, localStorage) to the page
func (e *Engine) applySessionData(ctx context.Context, pageMgr *browser.PageManager, sessionDataInterface interface{}) error {
	// Convert session data to proper format
	sessionDataBytes, err := json.Marshal(sessionDataInterface)
	if err != nil {
		return err
	}

	var sessionData types.SessionData
	if err := json.Unmarshal(sessionDataBytes, &sessionData); err != nil {
		return err
	}

	// Apply cookies
	if len(sessionData.Cookies) > 0 {
		for _, cookie := range sessionData.Cookies {
			// Set cookie via page manager (needs implementation)
			e.logger.Debug("setting cookie",
				logger.String("name", cookie.Name),
				logger.String("domain", cookie.Domain))
		}
	}

	// Apply localStorage items
	if sessionData.SessionData != nil {
		for key := range sessionData.SessionData {
			// Set localStorage item via page manager (needs implementation)
			e.logger.Debug("setting localStorage",
				logger.String("key", key))
		}
	}

	// Apply access token if available (as Authorization header)
	if sessionData.AccessToken != "" {
		// This would be handled via API request interceptors
		e.logger.Debug("setting access token")
	}

	return nil
}

// mapToAction convert map to Action struct
// 后端已经发送正确格式的数据，这里只需简单映射字段
func (e *Engine) mapToAction(m map[string]interface{}) types.Action {
	action := types.Action{}
	if id, ok := m["id"].(string); ok {
		action.ID = id
	}
	if actionType, ok := m["type"].(string); ok {
		action.Type = types.ActionType(actionType)
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

	return action
}

// substituteVariables substitute variables
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

// replaceVariables replace variables in string
func (e *Engine) replaceVariables(s string, variables map[string]interface{}) string {
    result := s
    for k, v := range variables {
        placeholder := fmt.Sprintf("${%s}", k)
        result = e.replaceAll(result, placeholder, fmt.Sprintf("%v", v))
    }
    return result
}

// replaceAll replace all matches
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

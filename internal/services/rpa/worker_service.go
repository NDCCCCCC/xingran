package rpa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// rpaWorkerProgressPublishTimeout RPA Worker 进度发布超时
const rpaWorkerProgressPublishTimeout = 5 * time.Second

// ==================== 接口定义（遵循接口隔离原则）====================

// WorkerRepository Worker 基础查询接口
// 遵循 Go 最佳实践：接口隔离原则 - 将大接口拆分为小接口
type WorkerRepository interface {
	// GetByID 根据 ID 获取 Worker
	GetByID(ctx context.Context, id string) (*rpamodels.Worker, error)
	// List 查询 Worker 列表
	List(ctx context.Context, params *WorkerListParams) (*PageResult, error)
	// Statistics Worker 统计(全量查询后按实时心跳判定在线状态,不依赖分页列表)
	Statistics(ctx context.Context) (*WorkerStatisticsResult, error)
}

// workerHeartbeatTimeoutSeconds 心跳超时秒数(与前端 getWorkerActualStatus 的 120s 一致)
const workerHeartbeatTimeoutSeconds int64 = 120

// WorkerStatisticsResult Worker 统计结果。
// online/offline/busy 按实时心跳(now - last_heartbeat <= 120s)派生,与前端判定一致。
type WorkerStatisticsResult struct {
	Total         int64 `json:"total"`
	Online        int64 `json:"online"`
	Offline       int64 `json:"offline"`
	Busy          int64 `json:"busy"`
	Error         int64 `json:"error"`
	TotalCapacity int64 `json:"totalCapacity"`
	UsedCapacity  int64 `json:"usedCapacity"`
}

// WorkerRuntime Worker 运行时操作接口
type WorkerRuntime interface {
	// Register 注册新 Worker
	Register(ctx context.Context, req *WorkerRegisterRequest) (*rpamodels.Worker, error)
	// Heartbeat 心跳检测
	Heartbeat(ctx context.Context, req *WorkerHeartbeatRequest) error
	// Offline 下线 Worker
	Offline(ctx context.Context, id string) error
	// GetAvailable 获取可用 Worker 列表
	GetAvailable(ctx context.Context) ([]rpamodels.Worker, error)
	// CheckOfflineWorkers 检查并下线超时的 Worker
	CheckOfflineWorkers(ctx context.Context, timeoutSeconds int64) error
}

// WorkerProgressReporter Worker 进度报告接口
type WorkerProgressReporter interface {
	// Progress 报告执行进度
	Progress(ctx context.Context, req *WorkerProgressRequest) error
}

// WorkerService Worker 完整服务接口
// 组合多个小接口，便于灵活实现和测试
type WorkerService interface {
	WorkerRepository
	WorkerRuntime
	WorkerProgressReporter
}

// workerServiceImpl Worker 服务实现
type workerServiceImpl struct {
	db               *gorm.DB
	executionService ExecutionService
	screenshotsDir   string
}

// NewWorkerService 创建 Worker 服务
func NewWorkerService(db *gorm.DB, executionService ExecutionService, screenshotsDir string) WorkerService {
	// 默认截图目录
	if screenshotsDir == "" {
		screenshotsDir = "./uploads/rpa/screenshots"
	}
	return &workerServiceImpl{
		db:               db,
		executionService: executionService,
		screenshotsDir:   screenshotsDir,
	}
}

// Register 注册 Worker
func (s *workerServiceImpl) Register(ctx context.Context, req *WorkerRegisterRequest) (*rpamodels.Worker, error) {
	// 构建能力配置
	caps := rpamodels.WorkerCapability{
		BrowserTypes: req.Capabilities,
		Headless:     true,
		MaxTimeout:   300,
	}

	capsJSON, err := json.Marshal(caps)
	if err != nil {
		return nil, fmt.Errorf("能力配置序列化失败: %w", err)
	}

	now := time.Now().Unix()

	// 使用原生 SQL ON CONFLICT DO UPDATE 避免 BaseModel 字段问题
	// Worker 是自动注册的，不需要用户审计字段
	query := `
		INSERT INTO sys_rpa_workers (
			worker_name, worker_id, ip_address, port, status,
			capabilities, max_concurrency, current_tasks, last_heartbeat,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (worker_id) DO UPDATE SET
			worker_name = EXCLUDED.worker_name,
			ip_address = EXCLUDED.ip_address,
			port = EXCLUDED.port,
			status = EXCLUDED.status,
			capabilities = EXCLUDED.capabilities,
			max_concurrency = EXCLUDED.max_concurrency,
			last_heartbeat = EXCLUDED.last_heartbeat,
			updated_at = NOW()
		RETURNING *
	`

	worker := &rpamodels.Worker{}
	err = s.db.WithContext(ctx).Raw(query,
		req.Name, req.WorkerID, req.Host, req.Port, rpamodels.WorkerStatusOnline,
		string(capsJSON), req.MaxConcurrency, 0, now,
	).Scan(worker).Error

	if err != nil {
		return nil, fmt.Errorf("注册 Worker 失败: %w", err)
	}

	return worker, nil
}

// Heartbeat 心跳上报
func (s *workerServiceImpl) Heartbeat(ctx context.Context, req *WorkerHeartbeatRequest) error {
	now := time.Now().Unix()
	updates := map[string]interface{}{
		"last_heartbeat": now,
		"current_tasks":  req.CurrentTasks,
		"status":         req.Status,
	}

	return s.db.WithContext(ctx).Model(&rpamodels.Worker{}).Where("worker_id = ?", req.WorkerID).Updates(updates).Error
}

// Progress 进度上报
func (s *workerServiceImpl) Progress(ctx context.Context, req *WorkerProgressRequest) error {
	// 查询执行记录
	var execution rpamodels.Execution
	if err := s.db.WithContext(ctx).Where("id = ?", req.ExecutionID).First(&execution).Error; err != nil {
		return fmt.Errorf("执行记录不存在: %w", err)
	}

	// 更新执行记录进度
	updates := map[string]interface{}{
		"progress_current": req.ProgressCurrent,
		"progress_total":   req.ProgressTotal,
	}

	// 更新状态
	if req.Status != "" {
		updates["status"] = req.Status

		// 根据状态更新开始/结束时间
		switch req.Status {
		case "running":
			if execution.StartTime == nil {
				now := time.Now()
				updates["start_time"] = &now
			}
		case "success", "failed", "cancelled", "timeout":
			now := time.Now()
			updates["end_time"] = &now
			if execution.StartTime != nil {
				duration := now.Sub(*execution.StartTime).Milliseconds()
				updates["duration"] = duration
			}
		}
	}

	// 更新错误消息
	if req.Message != "" && (req.Status == "failed" || req.Status == "error") {
		updates["error_message"] = req.Message
	}

	// 处理截图 - 保存到文件系统并存储路径
	var screenshotURL string
	if req.Screenshot != "" {
		screenshotPath, err := s.saveScreenshot(ctx, req.ExecutionID, req.ProgressCurrent, req.Screenshot)
		if err != nil {
			// 记录错误但不中断进度更新
			// 遵循 Go 最佳实践：使用结构化日志而非 fmt.Printf
			logger.Errorf("保存截图失败: executionID=%s, step=%d, error=%v", req.ExecutionID, req.ProgressCurrent, err)
		} else {
			screenshotURL = "/" + filepath.ToSlash(screenshotPath)
			// 获取现有截图并追加新截图
			existingScreenshots := rpamodels.StringArray{}
			if execution.Screenshots != nil {
				existingScreenshots = execution.Screenshots
			}
			// 追加新截图 URL
			existingScreenshots = append(existingScreenshots, screenshotURL)
			updates["screenshots"] = existingScreenshots
		}
	}

	// 添加日志
	if req.Log != "" {
		logEntry := "\n" + FormatLog(req.Log)
		updates["logs"] = gorm.Expr("COALESCE(logs, '') || ?", logEntry)
	} else if req.Message != "" {
		logEntry := "\n" + FormatLog(req.Message)
		updates["logs"] = gorm.Expr("COALESCE(logs, '') || ?", logEntry)
	}

	if err := s.db.WithContext(ctx).Model(&rpamodels.Execution{}).Where("id = ?", req.ExecutionID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新执行记录失败: %w", err)
	}

	// 发布进度到 WebSocket
	if s.executionService != nil {
		workerID := ""
		if execution.WorkerID != nil {
			workerID = *execution.WorkerID
		}
		workerName := execution.WorkerName
		progressUpdate := &ProgressUpdate{
			ExecutionID:   req.ExecutionID,
			TaskID:        execution.TaskID,
			TaskName:      execution.TaskName,
			Step:          req.ProgressCurrent,
			Total:         req.ProgressTotal,
			Message:       req.Message,
			Status:        req.Status,
			TriggeredBy:   execution.TriggeredBy,
			WorkerID:      workerID,
			WorkerName:    workerName,
			ScreenshotURL: screenshotURL, // 使用文件路径 URL 而不是 base64 数据
		}

		// 异步推送，避免阻塞 Worker
		// 遵循 Go 最佳实践：使用带超时的派生 context，从父 context 派生而非使用 Background()
		go func() {
			// 使用带超时的派生 context，确保 goroutine 不会永远阻塞
			// 从父 context 派生，支持请求取消时一并取消 goroutine
			pubCtx, cancel := context.WithTimeout(ctx, rpaWorkerProgressPublishTimeout)
			defer cancel()
			_ = s.executionService.PublishProgress(pubCtx, progressUpdate)
		}()
	}

	return nil
}

// List 查询 Worker 列表
func (s *workerServiceImpl) List(ctx context.Context, params *WorkerListParams) (*PageResult, error) {
	var workers []rpamodels.Worker
	var total int64

	query := s.db.WithContext(ctx).Model(&rpamodels.Worker{}).Where("deleted_at IS NULL")

	if params.Name != "" {
		query = query.Where("worker_name LIKE ?", "%"+params.Name+"%")
	}
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (params.Current - 1) * params.PageSize
	query = base.ApplySort(query, params.BaseListRequest, workerAllowedSortFields)
	if params.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(params.PageSize).Find(&workers).Error; err != nil {
		return nil, err
	}

	return &PageResult{
		List:     workers,
		Total:    total,
		Current:  params.Current,
		PageSize: params.PageSize,
	}, nil
}

// Statistics 统计 Worker(全量查询后在内存按实时心跳派生 online/offline/busy)。
// worker 数量通常很少(几台~几十台 RPA 节点),全量查询无性能问题;
// online/offline 是基于 last_heartbeat 的实时判定,跨库做 SQL 时间函数不一致,故放 Go 层算。
// 判定顺序与前端 getWorkerActualStatus 一致: busy+在线 → busy; status=error → error; 在线 → online; 否则 offline。
func (s *workerServiceImpl) Statistics(ctx context.Context) (*WorkerStatisticsResult, error) {
	var workers []rpamodels.Worker
	if err := s.db.WithContext(ctx).
		Model(&rpamodels.Worker{}).
		Where("deleted_at IS NULL").
		Find(&workers).Error; err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	result := &WorkerStatisticsResult{Total: int64(len(workers))}
	for i := range workers {
		w := &workers[i]
		// 心跳在线 = 有心跳时间戳且距今 <= 120s
		online := w.LastHeartbeat != nil && (now-*w.LastHeartbeat) <= workerHeartbeatTimeoutSeconds

		switch {
		case w.Status == rpamodels.WorkerStatusBusy && online:
			result.Busy++
		case w.Status == "error":
			result.Error++
		case online:
			result.Online++
		default:
			result.Offline++
		}

		// 容量聚合(max_concurrency 默认 3,与前端 fallback 一致)
		maxConcurrency := w.MaxConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 3
		}
		result.TotalCapacity += int64(maxConcurrency)
		result.UsedCapacity += int64(w.CurrentTasks)
	}

	return result, nil
}

// GetByID 获取 Worker 详情
func (s *workerServiceImpl) GetByID(ctx context.Context, id string) (*rpamodels.Worker, error) {
	var worker rpamodels.Worker
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&worker).Error
	if err != nil {
		return nil, err
	}
	return &worker, nil
}

// GetAvailable 获取可用 Worker 列表
func (s *workerServiceImpl) GetAvailable(ctx context.Context) ([]rpamodels.Worker, error) {
	var workers []rpamodels.Worker

	err := s.db.WithContext(ctx).
		Where("status = ? AND current_tasks < max_concurrency", rpamodels.WorkerStatusOnline).
		Order("current_tasks ASC").
		Find(&workers).Error

	return workers, err
}

// Offline 下线 Worker
func (s *workerServiceImpl) Offline(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Model(&rpamodels.Worker{}).Where("id = ?", id).Update("status", rpamodels.WorkerStatusOffline).Error
}

// CheckOfflineWorkers 检查并下线超时的 Worker
// 将超过一定时间没有发送心跳的 Worker 标记为离线
func (s *workerServiceImpl) CheckOfflineWorkers(ctx context.Context, timeoutSeconds int64) error {
	now := time.Now().Unix()
	timeout := now - timeoutSeconds

	result := s.db.WithContext(ctx).
		Model(&rpamodels.Worker{}).
		Where("status = ? AND last_heartbeat < ?", rpamodels.WorkerStatusOnline, timeout).
		Updates(map[string]interface{}{
			"status": rpamodels.WorkerStatusOffline,
		})

	return result.Error
}

// saveScreenshot 保存 base64 截图到文件系统
// 返回相对路径用于数据库存储和 HTTP 访问
func (s *workerServiceImpl) saveScreenshot(ctx context.Context, executionID string, step int, base64Data string) (string, error) {
	// 解析 base64 数据
	// 支持带或不带 data URI 前缀
	data := base64Data
	if strings.HasPrefix(data, "data:image/") {
		// 移除 data URI 前缀 (例如: data:image/png;base64,)
		parts := strings.SplitN(data, ",", 2)
		if len(parts) == 2 {
			data = parts[1]
		}
	}

	// 解码 base64
	imageBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("解码 base64 截图失败: %w", err)
	}

	// 创建执行记录专属目录
	execDir := filepath.Join(s.screenshotsDir, executionID)
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return "", fmt.Errorf("创建截图目录失败: %w", err)
	}

	// 生成文件名: step_{step}_{timestamp}.png
	timestamp := time.Now().UnixMilli()
	filename := fmt.Sprintf("step_%d_%d.png", step, timestamp)
	filePath := filepath.Join(execDir, filename)

	// 写入文件
	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		return "", fmt.Errorf("保存截图文件失败: %w", err)
	}

	// 返回相对路径用于 HTTP 访问
	// 例如: rpa/screenshots/{executionID}/step_1_1234567890.png
	relativePath := filepath.Join("rpa", "screenshots", executionID, filename)
	return relativePath, nil
}

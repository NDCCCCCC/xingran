package communication

import (
	"context"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ProgressReporter 进度上报器
type ProgressReporter struct {
	apiClient *APIClient
	workerID  string
	logger    logger.Logger
	interval  time.Duration
}

// NewProgressReporter 创建进度上报器
func NewProgressReporter(apiClient *APIClient, workerID string, log logger.Logger) *ProgressReporter {
	return &ProgressReporter{
		apiClient: apiClient,
		workerID:  workerID,
		logger:    log,
		interval:  5 * time.Second,
	}
}

// ReportProgress 上报进度
func (p *ProgressReporter) ReportProgress(ctx context.Context, req *types.ProgressReport) error {
	req.WorkerID = p.workerID
	req.Timestamp = time.Now()
	return p.apiClient.ReportProgress(ctx, req)
}

// ReportCompletion 上报完成
func (p *ProgressReporter) ReportCompletion(ctx context.Context, result *types.ExecutionResult) error {
	req := &types.ProgressReport{
		ExecutionID:     result.ExecutionID,
		ProgressCurrent: result.Step,
		ProgressTotal:   result.Total,
		Message:         "任务执行完成",
		Status:          types.ExecutionStatus(result.Status),
		Timestamp:       time.Now(),
	}

	if result.ErrorMessage != "" {
		req.Message = result.ErrorMessage
	}

	return p.ReportProgress(ctx, req)
}

// ReportError 上报错误
func (p *ProgressReporter) ReportError(ctx context.Context, executionID string, step int, total int, errMsg string, screenshot string) error {
	return p.ReportProgress(ctx, &types.ProgressReport{
		ExecutionID:     executionID,
		ProgressCurrent: step,
		ProgressTotal:   total,
		Message:         errMsg,
		Status:          types.StatusFailed,
		Screenshot:      screenshot,
		Timestamp:       time.Now(),
	})
}

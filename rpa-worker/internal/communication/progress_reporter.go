package communication

import (
	"strings"

	"context"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ProgressReporter progress reporter
type ProgressReporter struct {
	apiClient *APIClient
	workerID  string
	logger    logger.Logger
}

// NewProgressReporter create progress reporter
func NewProgressReporter(apiClient *APIClient, workerID string, log logger.Logger) *ProgressReporter {
	return &ProgressReporter{
		apiClient: apiClient,
		workerID:  workerID,
		logger:    log,
	}
}

// ReportProgress report progress
func (p *ProgressReporter) ReportProgress(ctx context.Context, req *types.ProgressReport) error {
	req.WorkerID = p.workerID
	req.Timestamp = time.Now()
	return p.apiClient.ReportProgress(ctx, req)
}

// ReportCompletion report completion
func (p *ProgressReporter) ReportCompletion(ctx context.Context, result *types.ExecutionResult) error {
	req := &types.ProgressReport{
		ExecutionID:     result.ExecutionID,
		WorkerID:        p.workerID,
		ProgressCurrent: result.Step,
		ProgressTotal:   result.Total,
		Step:            result.Step,
		Total:           result.Total,
		Message:         "task completed",
		Status:          types.ExecutionStatus(result.Status),
		Timestamp:       time.Now(),
	}

	if result.ErrorMessage != "" {
		req.Message = result.ErrorMessage
	}

	// TIDY-03 配套：执行日志（含选择器契约信封行，若 worker 采集到）随收尾
	// 上报体追加进执行记录 logs。后端 Progress 通道按 FormatLog 给首行加
	// "[timestamp] " 前缀，故信封行前垫一行普通日志保证信封行首为 "{"（契约
	// 解析按行 HasPrefix("{") 判定）。
	if len(result.Logs) > 0 {
		lines := append([]string{"execution log report"}, result.Logs...)
		req.Log = strings.Join(lines, "\n")
	}

	return p.ReportProgress(ctx, req)
}

// ReportError report error
func (p *ProgressReporter) ReportError(ctx context.Context, executionID string, step int, total int, errMsg string, screenshot string) error {
	return p.ReportProgress(ctx, &types.ProgressReport{
		ExecutionID:     executionID,
		WorkerID:        p.workerID,
		ProgressCurrent: step,
		ProgressTotal:   total,
		Step:            step,
		Total:           total,
		Message:         errMsg,
		Status:          types.StatusFailed,
		Screenshot:      screenshot,
		Timestamp:       time.Now(),
	})
}

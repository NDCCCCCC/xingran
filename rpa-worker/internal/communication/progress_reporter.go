package communication

import (
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

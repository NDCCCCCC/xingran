package rpa

import (
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Phase 158 / CLEANUP-01：批量执行报告的真实落库模型
//
// 背景：services/rpa/excel_service.go 的 GetBatchExecutionReport 此前直接返回
// 硬编码常量（total=100 / success=95 / failed=5），Create/Update 是空壳存根。
// 本模型给「逐项」粒度一个真实表，使报告可写、可查、可统计：
//
//	sys_rpa_batch_item_reports —— 一行 = 一个批量执行项
//
// 统计口径（GetBatchExecutionReport 聚合）：
//   - totalItems   = 该 execution 的全部项数
//   - successCount = status='success'
//   - failedCount  = status='failed'
//   - skippedCount = status='skipped'
//   - 其余（pending 等）不计入三类，仅体现在 totalItems 与明细里
//
// 无数据时 totalItems=0 且明细为空 —— 不再返回任何常量假值。
//
// 方言：字段 tag 无 PG-only 片段（无函数式默认值 / 无数组类型），
// 双方言（PG / sqlite）通用，无需 sanitize。
// ============================================================================

// BatchItemReportStatus 批量项状态取值。
const (
	BatchItemStatusPending = "pending" // 待执行
	BatchItemStatusSuccess = "success" // 成功
	BatchItemStatusFailed  = "failed"  // 失败
	BatchItemStatusSkipped = "skipped" // 跳过
)

// BatchItemReport RPA 批量执行项报告（真实落库行）。
type BatchItemReport struct {
	models.BaseModel
	// ExecutionID 关联的执行记录 ID（sys_rpa_executions.id）
	ExecutionID string `gorm:"type:uuid;not null;index:idx_rpa_batch_item_exec,priority:1" json:"executionId"`
	// TaskID 关联的 RPA 任务 ID（sys_rpa_tasks.id）
	TaskID string `gorm:"type:uuid;index" json:"taskId"`
	// ItemIndex 项序号（0 基），与 (execution_id, item_index) 组成项级定位
	ItemIndex int `gorm:"not null;index:idx_rpa_batch_item_exec,priority:2" json:"itemIndex"`
	// Status 项状态：pending / success / failed / skipped
	Status string `gorm:"size:20;not null;default:'pending'" json:"status"`
	// InputData 输入数据（JSON 字符串）
	InputData string `gorm:"type:text" json:"inputData"`
	// OutputData 输出数据（JSON 字符串）
	OutputData string `gorm:"type:text" json:"outputData"`
	// ErrorMessage 失败原因
	ErrorMessage string `gorm:"type:text" json:"error"`
	// Screenshot 失败截图路径
	Screenshot string `gorm:"size:500" json:"screenshot"`
	// StartTime 项开始时间
	StartTime time.Time `gorm:"column:start_time" json:"startTime"`
	// EndTime 项结束时间（未完成为 nil）
	EndTime *time.Time `gorm:"column:end_time" json:"endTime"`
	// Duration 项耗时（毫秒）
	Duration int64 `gorm:"column:duration" json:"duration"`
}

// TableName 指定表名
func (BatchItemReport) TableName() string {
	return "sys_rpa_batch_item_reports"
}

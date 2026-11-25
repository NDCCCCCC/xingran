package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ExcelParseResult Excel 解析结果
type ExcelParseResult struct {
	FileName   string                   `json:"fileName"`
	FileSize   int64                    `json:"fileSize"`
	SheetCount int                      `json:"sheetCount"`
	Columns    []string                 `json:"columns"`
	RowCount   int                      `json:"rowCount"`
	Preview    []map[string]interface{} `json:"preview"`
	MaxPreview int                      `json:"maxPreview"`
	ParsedAt   time.Time                `json:"parsedAt"`
}

// BatchExecutionReport 批量执行报告
type BatchExecutionReport struct {
	ExecutionID  string                 `json:"executionId"`
	TaskID       string                 `json:"taskId"`
	TaskName     string                 `json:"taskName"`
	TotalItems   int                    `json:"totalItems"`
	SuccessCount int                    `json:"successCount"`
	FailedCount  int                    `json:"failedCount"`
	SkippedCount int                    `json:"skippedCount"`
	StartTime    time.Time              `json:"startTime"`
	EndTime      time.Time              `json:"endTime"`
	Duration     time.Duration          `json:"duration"`
	Items        []BatchItemReport      `json:"items"`
	Summary      map[string]interface{} `json:"summary"`
}

// BatchItemReport 批量项报告
type BatchItemReport struct {
	Index      int                    `json:"index"`
	Status     string                 `json:"status"` // success, failed, skipped
	InputData  map[string]interface{} `json:"inputData"`
	OutputData map[string]interface{} `json:"outputData"`
	Error      string                 `json:"error,omitempty"`
	StartTime  time.Time              `json:"startTime"`
	EndTime    time.Time              `json:"endTime"`
	Duration   time.Duration          `json:"duration"`
	Screenshot string                 `json:"screenshot,omitempty"`
}

// HumanInterventionRequest 人工干预请求
type HumanInterventionRequest struct {
	ExecutionID string                 `json:"executionId" binding:"required"`
	Action      string                 `json:"action" binding:"required,oneof=resume skip abort"`
	Input       map[string]interface{} `json:"input"`
	Reason      string                 `json:"reason"`
}

// HumanInterventionEvent 人工干预事件
type HumanInterventionEvent struct {
	ID          string     `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExecutionID string     `gorm:"type:uuid;not null;index:idx_rpa_hi_exec,priority:1" json:"executionId"`
	WorkerID    string     `gorm:"size:100;index" json:"workerId"`
	Action      string     `gorm:"size:20;not null" json:"action"` // pause, resume, skip, abort
	Message     string     `gorm:"type:text" json:"message"`
	InputData   string     `gorm:"type:jsonb" json:"inputData"` // 用户输入的数据
	Reason      string     `gorm:"type:text" json:"reason"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
	Status      string     `gorm:"size:20;default:'pending'" json:"status"` // pending, processed, timeout
}

func (HumanInterventionEvent) TableName() string {
	return "sys_rpa_human_interventions"
}

// RPAExcelService RPA Excel 服务
type RPAExcelService struct {
	db *gorm.DB
}

// NewRPAExcelService 创建 RPA Excel 服务
func NewRPAExcelService(db *gorm.DB) *RPAExcelService {
	return &RPAExcelService{db: db}
}

// ParseExcelFile 解析 Excel 文件
func (s *RPAExcelService) ParseExcelFile(file *multipart.FileHeader) (*ExcelParseResult, error) {
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 使用 excelize 读取 Excel
	f, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("解析Excel失败: %w", err)
	}
	defer f.Close()

	// 获取所有工作表
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel文件中没有工作表")
	}

	// 使用第一个工作表
	sheetName := sheets[0]

	// 获取所有行
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取行失败: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("工作表为空")
	}

	// 第一行作为列名
	columns := append([]string{}, rows[0]...)

	// 解析数据行（跳过表头）
	data := make([]map[string]interface{}, 0)
	maxPreview := 10 // 预览最多10行

	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}

		rowData := make(map[string]interface{})
		for j, cell := range row {
			if j < len(columns) {
				rowData[columns[j]] = cell
			}
		}

		data = append(data, rowData)

		// 限制预览数量
		if len(data) >= maxPreview {
			break
		}
	}

	return &ExcelParseResult{
		FileName:   file.Filename,
		FileSize:   file.Size,
		SheetCount: len(sheets),
		Columns:    columns,
		RowCount:   len(rows) - 1, // 减去表头
		Preview:    data,
		MaxPreview: maxPreview,
		ParsedAt:   time.Now(),
	}, nil
}

// ParseExcelForExecution 解析 Excel 并准备执行数据
func (s *RPAExcelService) ParseExcelForExecution(file *multipart.FileHeader) ([]map[string]interface{}, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("解析Excel失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel文件中没有工作表")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取行失败: %w", err)
	}

	if len(rows) <= 1 {
		return nil, fmt.Errorf("工作表没有数据行")
	}

	// 第一行作为列名
	columns := append([]string{}, rows[0]...)

	// 解析所有数据行
	data := make([]map[string]interface{}, 0)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}

		rowData := make(map[string]interface{})
		for j, cell := range row {
			if j < len(columns) {
				rowData[columns[j]] = cell
			}
		}

		data = append(data, rowData)
	}

	return data, nil
}

// CreateBatchExecutionReport 创建批量执行报告
//
// Phase 158 / CLEANUP-01：真实落库 —— 逐项写入 sys_rpa_batch_item_reports。
// 此前是空壳存根（构造内存 report 后直接丢弃），报告永远查不到真值。
func (s *RPAExcelService) CreateBatchExecutionReport(
	ctx context.Context,
	executionID string,
	task *rpamodels.Task,
	dataItems []map[string]interface{},
) error {
	if len(dataItems) == 0 {
		// 无数据项：不写行（读侧会返回空统计而非假数据）
		return nil
	}

	now := time.Now()
	rows := make([]rpamodels.BatchItemReport, 0, len(dataItems))
	for i, item := range dataItems {
		inputJSON := "{}"
		if raw, err := json.Marshal(item); err == nil {
			inputJSON = string(raw)
		}
		rows = append(rows, rpamodels.BatchItemReport{
			ExecutionID: executionID,
			TaskID:      task.ID,
			ItemIndex:   i,
			Status:      rpamodels.BatchItemStatusPending,
			InputData:   inputJSON,
			StartTime:   now,
		})
	}

	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return fmt.Errorf("写入批量执行项报告失败: %w", err)
	}
	return nil
}

// UpdateBatchItemReport 更新批量项报告
//
// Phase 158 / CLEANUP-01：真实落库 —— 按 (execution_id, item_index) 更新单项状态。
// 此前是空壳存根（注释写「实际实现需要读取缓存」）。
func (s *RPAExcelService) UpdateBatchItemReport(
	ctx context.Context,
	executionID string,
	itemIndex int,
	status string,
	outputData map[string]interface{},
	errMsg string,
	screenshot string,
) error {
	var item rpamodels.BatchItemReport
	err := s.db.WithContext(ctx).
		Where("execution_id = ? AND item_index = ?", executionID, itemIndex).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("批量执行项不存在: execution=%s index=%d", executionID, itemIndex)
		}
		return fmt.Errorf("查询批量执行项失败: %w", err)
	}

	updates := map[string]interface{}{
		"status": status,
	}
	if len(outputData) > 0 {
		if raw, mErr := json.Marshal(outputData); mErr == nil {
			updates["output_data"] = string(raw)
		}
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if screenshot != "" {
		updates["screenshot"] = screenshot
	}

	now := time.Now()
	updates["end_time"] = &now
	if !item.StartTime.IsZero() && now.After(item.StartTime) {
		updates["duration"] = now.Sub(item.StartTime).Milliseconds()
	}

	if err := s.db.WithContext(ctx).
		Model(&rpamodels.BatchItemReport{}).
		Where("id = ?", item.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("更新批量执行项失败: %w", err)
	}
	return nil
}

// GetBatchExecutionReport 获取批量执行报告
//
// Phase 158 / CLEANUP-01：真实统计 —— 从 sys_rpa_batch_item_reports 聚合。
// 此前返回硬编码常量（total=100 / success=95 / failed=5），是本 phase 的头号假数据。
// 无数据时返回空统计（totalItems=0 / 明细空），绝不返回任何常量假值。
func (s *RPAExcelService) GetBatchExecutionReport(ctx context.Context, executionID string) (*BatchExecutionReport, error) {
	report := &BatchExecutionReport{
		ExecutionID: executionID,
		Items:       make([]BatchItemReport, 0),
		Summary:     make(map[string]interface{}),
	}

	var rows []rpamodels.BatchItemReport
	if err := s.db.WithContext(ctx).
		Where("execution_id = ?", executionID).
		Order("item_index ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询批量执行项报告失败: %w", err)
	}

	report.TotalItems = len(rows)
	pending := 0
	var earliest, latest time.Time

	for _, row := range rows {
		switch row.Status {
		case rpamodels.BatchItemStatusSuccess:
			report.SuccessCount++
		case rpamodels.BatchItemStatusFailed:
			report.FailedCount++
		case rpamodels.BatchItemStatusSkipped:
			report.SkippedCount++
		default:
			pending++
		}

		item := BatchItemReport{
			Index:     row.ItemIndex,
			Status:    row.Status,
			Error:     row.ErrorMessage,
			Screenshot: row.Screenshot,
			StartTime: row.StartTime,
			Duration:  time.Duration(row.Duration) * time.Millisecond,
		}
		if row.EndTime != nil {
			item.EndTime = *row.EndTime
		}
		if row.InputData != "" {
			var input map[string]interface{}
			if jErr := json.Unmarshal([]byte(row.InputData), &input); jErr == nil {
				item.InputData = input
			}
		}
		if row.OutputData != "" {
			var output map[string]interface{}
			if jErr := json.Unmarshal([]byte(row.OutputData), &output); jErr == nil {
				item.OutputData = output
			}
		}
		report.Items = append(report.Items, item)

		if earliest.IsZero() || row.StartTime.Before(earliest) {
			earliest = row.StartTime
		}
		if row.EndTime != nil && (latest.IsZero() || row.EndTime.After(latest)) {
			latest = *row.EndTime
		}
	}

	// 执行记录的任务名（可空 —— 执行记录可能已被清理）
	var execRow rpamodels.Execution
	if err := s.db.WithContext(ctx).
		Where("id = ?", executionID).
		First(&execRow).Error; err == nil {
		report.TaskID = execRow.TaskID
		report.TaskName = execRow.TaskName
	}

	if !earliest.IsZero() {
		report.StartTime = earliest
	}
	if !latest.IsZero() {
		report.EndTime = latest
		report.Duration = latest.Sub(earliest)
	}

	report.Summary = map[string]interface{}{
		"total":   report.TotalItems,
		"success": report.SuccessCount,
		"failed":  report.FailedCount,
		"skipped": report.SkippedCount,
		"pending": pending,
	}

	return report, nil
}

// CreateHumanInterventionEvent 创建人工干预事件
func (s *RPAExcelService) CreateHumanInterventionEvent(ctx context.Context, req *HumanInterventionRequest, workerID string) (*HumanInterventionEvent, error) {
	event := &HumanInterventionEvent{
		ExecutionID: req.ExecutionID,
		WorkerID:    workerID,
		Action:      req.Action,
		Message:     fmt.Sprintf("用户操作: %s", req.Action),
		Status:      "pending",
	}

	// 序列化输入数据
	if len(req.Input) > 0 {
		dataBytes, err := json.Marshal(req.Input)
		if err == nil {
			event.InputData = string(dataBytes)
		}
	}

	if req.Reason != "" {
		event.Reason = req.Reason
	}

	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

// GetPendingHumanIntervention 获取待处理的人工干预
func (s *RPAExcelService) GetPendingHumanIntervention(ctx context.Context, executionID string) (*HumanInterventionEvent, error) {
	var event HumanInterventionEvent
	err := s.db.WithContext(ctx).
		Where("execution_id = ? AND status = ? AND created_at > ?",
			executionID, "pending", time.Now().Add(-30*time.Minute)).
		Order("created_at DESC").
		First(&event).Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

// ProcessHumanIntervention 处理人工干预事件
func (s *RPAExcelService) ProcessHumanIntervention(ctx context.Context, eventID string, processed bool) error {
	updates := map[string]interface{}{
		"status": "processed",
	}

	if processed {
		now := time.Now()
		updates["processed_at"] = &now
	}

	return s.db.WithContext(ctx).
		Model(&HumanInterventionEvent{}).
		Where("id = ?", eventID).
		Updates(updates).Error
}

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
func (s *RPAExcelService) CreateBatchExecutionReport(
	ctx context.Context,
	executionID string,
	task *rpamodels.Task,
	dataItems []map[string]interface{},
) error {
	// 初始化报告
	report := &BatchExecutionReport{
		ExecutionID: executionID,
		TaskID:      task.ID,
		TaskName:    task.TaskName,
		TotalItems:  len(dataItems),
		StartTime:   time.Now(),
		Items:       make([]BatchItemReport, 0, len(dataItems)),
		Summary:     make(map[string]interface{}),
	}

	// 为每个数据项创建报告项
	for i, item := range dataItems {
		report.Items = append(report.Items, BatchItemReport{
			Index:     i,
			Status:    "pending",
			InputData: item,
			StartTime: time.Now(),
		})
	}

	// 保存报告到数据库或缓存
	// 这里使用缓存存储，key 格式: rpa:batch_report:{execution_id}
	// 实际存储可以在 Execution 表中增加 batch_report 字段

	return nil
}

// UpdateBatchItemReport 更新批量项报告
func (s *RPAExcelService) UpdateBatchItemReport(
	ctx context.Context,
	executionID string,
	itemIndex int,
	status string,
	outputData map[string]interface{},
	errMsg string,
	screenshot string,
) error {
	// 更新缓存中的报告
	// 实际实现需要读取缓存，更新对应项，然后写回

	return nil
}

// GetBatchExecutionReport 获取批量执行报告
func (s *RPAExcelService) GetBatchExecutionReport(ctx context.Context, executionID string) (*BatchExecutionReport, error) {
	// 从缓存或数据库获取报告
	// 这里返回模拟数据

	return &BatchExecutionReport{
		ExecutionID:  executionID,
		TotalItems:   100,
		SuccessCount: 95,
		FailedCount:  5,
		SkippedCount: 0,
		Items:        make([]BatchItemReport, 0),
		Summary:      make(map[string]interface{}),
	}, nil
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

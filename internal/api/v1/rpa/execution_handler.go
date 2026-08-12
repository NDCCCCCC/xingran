package rpa

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ExecutionHandler 执行记录处理器
type ExecutionHandler struct {
	executionService rpa.ExecutionService
	excelService     *rpa.RPAExcelService
	core             *core.Core
}

// NewExecutionHandler 创建执行记录处理器
func NewExecutionHandler(executionService rpa.ExecutionService, excelService *rpa.RPAExcelService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
		excelService:     excelService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *ExecutionHandler) WithCore(core *core.Core) *ExecutionHandler {
	h.core = core
	return h
}

// List 执行记录列表
func (h *ExecutionHandler) List(c *gin.Context) {
	var params rpa.ExecutionListParams
	if !bindAndValidate(c, &params) {
		return
	}

	setPaginationDefaults(&params.Current, &params.PageSize)

	result, err := h.executionService.List(c.Request.Context(), &params)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, result)
}

// Statistics 执行记录统计(读操作,不记操作日志)
func (h *ExecutionHandler) Statistics(c *gin.Context) {
	result, err := h.executionService.Statistics(c.Request.Context())
	if handleError(c, err, http.StatusInternalServerError, "统计失败") {
		return
	}

	success(c, result)
}

// GetByID 获取执行记录详情
func (h *ExecutionHandler) GetByID(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	execution, err := h.executionService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "执行记录不存在")
		return
	}

	success(c, execution)
}

// Cancel 取消执行
func (h *ExecutionHandler) Cancel(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	if handleError(c, h.executionService.Cancel(c.Request.Context(), id), http.StatusInternalServerError, "取消失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA执行", operlog.OperTypeStatus)

	successMsg(c, "执行已取消")
}

// GetLogs 获取执行日志
func (h *ExecutionHandler) GetLogs(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	execution, err := h.executionService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "执行记录不存在")
		return
	}

	success(c, gin.H{"logs": execution.Logs})
}

// GetBatchReport 获取批量执行报告
func (h *ExecutionHandler) GetBatchReport(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	if h.excelService == nil {
		response.Error(c, http.StatusInternalServerError, "批量报告服务未初始化")
		return
	}

	report, err := h.excelService.GetBatchExecutionReport(c.Request.Context(), id)
	if handleError(c, err, http.StatusInternalServerError, "获取批量报告失败") {
		return
	}

	success(c, report)
}

// RequestHumanIntervention 请求人工干预
func (h *ExecutionHandler) RequestHumanIntervention(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	if h.excelService == nil {
		response.Error(c, http.StatusInternalServerError, "人工干预服务未初始化")
		return
	}

	// 获取待处理的人工干预事件
	event, err := h.excelService.GetPendingHumanIntervention(c.Request.Context(), id)
	if err != nil {
		// 没有待处理的干预，返回等待状态
		success(c, gin.H{
			"status":      "waiting",
			"message":     "正在等待用户输入",
			"executionId": id,
		})
		return
	}

	success(c, gin.H{
		"status":      "pending",
		"eventId":     event.ID,
		"action":      event.Action,
		"message":     event.Message,
		"executionId": id,
		"workerId":    event.WorkerID,
		"createdAt":   event.CreatedAt,
	})
}

// SubmitHumanIntervention 提交人工干预响应
func (h *ExecutionHandler) SubmitHumanIntervention(c *gin.Context) {
	var req rpa.HumanInterventionRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if h.excelService == nil {
		response.Error(c, http.StatusInternalServerError, "人工干预服务未初始化")
		return
	}

	workerID := c.Query("workerId")
	if workerID == "" {
		workerID = "unknown"
	}

	// 创建人工干预事件
	event, err := h.excelService.CreateHumanInterventionEvent(c.Request.Context(), &req, workerID)
	if handleError(c, err, http.StatusInternalServerError, "创建人工干预事件失败") {
		return
	}

	// 标记事件为已处理
	if err := h.excelService.ProcessHumanIntervention(c.Request.Context(), event.ID, true); err != nil {
		_ = err // 记录日志但不影响响应
	}

	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA执行", operlog.OperTypeOther)

	success(c, gin.H{
		"eventId": event.ID,
		"status":  "submitted",
		"message": "人工输入已提交",
		"action":  req.Action,
	})
}

// DownloadArtifacts 下载执行产物（截图和日志）为 ZIP 文件
func (h *ExecutionHandler) DownloadArtifacts(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	// 获取执行记录
	execution, err := h.executionService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "执行记录不存在")
		return
	}

	// 创建临时 ZIP 文件
	zipFileName := fmt.Sprintf("execution_%s_%s.zip", id, execution.TaskName)
	zipFileName = strings.ReplaceAll(zipFileName, " ", "_")
	zipFileName = strings.ReplaceAll(zipFileName, "/", "-")

	// 设置响应头为文件下载
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	// 创建 ZIP writer
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// 添加截图到 ZIP
	if len(execution.Screenshots) > 0 {
		uploadsDir := "./uploads" // 默认上传目录

		for _, screenshotPath := range execution.Screenshots {
			// 移除开头的斜杠并构建完整路径
			cleanPath := strings.TrimPrefix(screenshotPath, "/")
			fullPath := filepath.Join(uploadsDir, cleanPath)

			// 打开截图文件
			file, err := os.Open(fullPath)
			if err != nil {
				// 文件不存在，跳过
				continue
			}
			defer file.Close()

			// 在 ZIP 中创建文件
			w, err := zipWriter.Create(filepath.Join("screenshots", filepath.Base(cleanPath)))
			if err != nil {
				continue
			}

			// 复制文件内容
			if _, err := io.Copy(w, file); err != nil {
				continue
			}
		}
	}

	// 添加日志到 ZIP
	if execution.Logs != "" {
		w, err := zipWriter.Create("execution_log.txt")
		if err == nil {
			_, _ = w.Write([]byte(execution.Logs))
		}
	}

	// 添加执行信息
	infoContent := fmt.Sprintf("执行ID: %s\n任务ID: %s\n任务名称: %s\n状态: %s\n开始时间: %s\n结束时间: %s\n耗时: %dms\n步骤: %d/%d\n",
		id, execution.TaskID, execution.TaskName, execution.Status,
		formatTime(execution.StartTime), formatTime(execution.EndTime),
		execution.Duration, execution.Step, execution.TotalSteps)
	if execution.ErrorMessage != "" {
		infoContent += fmt.Sprintf("错误信息: %s\n", execution.ErrorMessage)
	}

	w, err := zipWriter.Create("execution_info.txt")
	if err == nil {
		_, _ = w.Write([]byte(infoContent))
	}
}

// formatTime 格式化时间指针
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

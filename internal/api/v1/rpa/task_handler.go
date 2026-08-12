package rpa

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// rpaBatchReportTimeout RPA 任务批量报告生成超时
const rpaBatchReportTimeout = 10 * time.Second

// TaskHandler 任务处理器
type TaskHandler struct {
	taskService  rpa.TaskService
	excelService *rpa.RPAExcelService
	core         *core.Core
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskService rpa.TaskService, excelService *rpa.RPAExcelService) *TaskHandler {
	return &TaskHandler{
		taskService:  taskService,
		excelService: excelService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *TaskHandler) WithCore(core *core.Core) *TaskHandler {
	h.core = core
	return h
}

// List 任务列表
func (h *TaskHandler) List(c *gin.Context) {
	var params rpa.TaskListParams
	if !bindAndValidate(c, &params) {
		return
	}

	setPaginationDefaults(&params.Current, &params.PageSize)

	result, err := h.taskService.List(c.Request.Context(), &params)
	if handleError(c, err, http.StatusInternalServerError, "查询失败") {
		return
	}

	success(c, result)
}

// Create 创建任务
func (h *TaskHandler) Create(c *gin.Context) {
	var req rpa.CreateTaskRequest
	if !bindAndValidate(c, &req) {
		return
	}

	userID := c.GetString("userId")
	task, err := h.taskService.Create(c.Request.Context(), &req, userID)
	if handleError(c, err, http.StatusInternalServerError, "创建失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeCreate)

	success(c, task)
}

// GetByID 获取任务详情
func (h *TaskHandler) GetByID(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	task, err := h.taskService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "任务不存在")
		return
	}

	success(c, task)
}

// Update 更新任务
func (h *TaskHandler) Update(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req rpa.UpdateTaskRequest
	if !bindAndValidate(c, &req) {
		return
	}

	req.ID = id
	userID := c.GetString("userId")
	if handleError(c, h.taskService.Update(c.Request.Context(), &req, userID), http.StatusInternalServerError, "更新失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeUpdate)

	successMsg(c, "更新成功")
}

// Delete 删除任务
func (h *TaskHandler) Delete(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	if handleError(c, h.taskService.Delete(c.Request.Context(), id), http.StatusInternalServerError, "删除失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeDelete)

	successMsg(c, "删除成功")
}

// Execute 执行任务
func (h *TaskHandler) Execute(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req rpa.ExecuteTaskRequest
	if c.ShouldBindJSON(&req) != nil {
		req = rpa.ExecuteTaskRequest{TaskID: id}
	} else {
		req.TaskID = id
	}

	userID := c.GetString("userId")
	execution, err := h.taskService.Execute(c.Request.Context(), &req, userID)
	if handleError(c, err, http.StatusInternalServerError, "执行失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeOther)

	success(c, execution)
}

// UploadExcel 上传并解析 Excel 文件
func (h *TaskHandler) UploadExcel(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择文件")
		return
	}

	// 验证文件类型
	if file.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" &&
		file.Header.Get("Content-Type") != "application/vnd.ms-excel" {
		response.Error(c, http.StatusBadRequest, "请上传 Excel 文件 (.xlsx 或 .xls)")
		return
	}

	// 解析 Excel
	result, err := h.excelService.ParseExcelFile(file)
	if handleError(c, err, http.StatusInternalServerError, "Excel 解析失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeUpload)

	success(c, result)
}

// ExecuteWithExcel 上传 Excel 并执行批量任务
func (h *TaskHandler) ExecuteWithExcel(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择 Excel 文件")
		return
	}

	// 解析 Excel 获取数据
	dataItems, err := h.excelService.ParseExcelForExecution(file)
	if handleError(c, err, http.StatusInternalServerError, "Excel 解析失败") {
		return
	}

	// 创建批量执行请求
	req := &rpa.ExecuteTaskRequest{
		TaskID: id,
		InputParams: map[string]interface{}{
			"dataSource": dataItems,
			"fileName":   file.Filename,
			"itemCount":  len(dataItems),
		},
	}

	userID := c.GetString("userId")
	execution, err := h.taskService.Execute(c.Request.Context(), req, userID)
	if handleError(c, err, http.StatusInternalServerError, "执行失败") {
		return
	}

	// 初始化批量报告（异步执行，避免阻塞响应）
	if h.excelService != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), rpaBatchReportTimeout)
			defer cancel()
			task, err := h.taskService.GetByID(ctx, id)
			if err == nil && task != nil {
				_ = h.excelService.CreateBatchExecutionReport(ctx, execution.ID, task, dataItems)
			}
		}()
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeOther)

	success(c, gin.H{
		"execution":  execution,
		"totalItems": len(dataItems),
		"message":    "批量任务已开始执行",
	})
}

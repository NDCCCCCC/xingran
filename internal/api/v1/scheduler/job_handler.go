package scheduler

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/common"
	schedulerServices "github.com/xingran-next/xingran-go-backend/internal/services/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// JobHandler 定时任务处理器
type JobHandler struct {
	jobService    schedulerServices.JobService
	jobLogService schedulerServices.JobLogService
	core          *core.Core
}

// NewJobHandler 创建定时任务处理器实例
func NewJobHandler(
	jobService schedulerServices.JobService,
	jobLogService schedulerServices.JobLogService,
) *JobHandler {
	return &JobHandler{
		jobService:    jobService,
		jobLogService: jobLogService,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *JobHandler) WithCore(core *core.Core) *JobHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// Create 创建定时任务
// @Summary 创建定时任务
// @Description 创建新的定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body object{jobName=string,jobGroup=string,invokeTarget=string,cronExpression=string,misfirePolicy=string,concurrent=bool,remark=string} true "任务信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs [post]
func (h *JobHandler) Create(c *gin.Context) {
	var req schedulerServices.JobCreateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	job, err := h.jobService.Create(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "创建定时任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeCreate)
	response.Success(c, job)
}

// List 查询定时任务列表
// @Summary 查询定时任务列表
// @Description 分页查询定时任务列表
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,jobName=string,jobGroup=string,status=int} false "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/list [post]
func (h *JobHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := &schedulerServices.JobListParams{
		ListParams: common.DefaultListParams(),
	}

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["jobName"].(string); ok {
		params.JobName = val
	}
	if val, ok := rawReq["jobGroup"].(string); ok {
		params.JobGroup = val
	}
	if val, ok := rawReq["status"]; ok {
		switch v := val.(type) {
		case float64:
			status := int(v)
			params.Status = &status
		case int:
			params.Status = &v
		}
	}

	result, err := h.jobService.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// GetByID 获取定时任务详情
// @Summary 获取定时任务详情
// @Description 根据任务ID获取定时任务详细信息
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id} [post]
func (h *JobHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	job, err := h.jobService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, job)
}

// Update 更新定时任务
// @Summary 更新定时任务
// @Description 更新定时任务信息
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body object{jobName=string,jobGroup=string,invokeTarget=string,cronExpression=string,misfirePolicy=string,concurrent=bool,remark=string} true "任务信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/update [post]
func (h *JobHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	var req schedulerServices.JobUpdateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}
	req.ID = id

	err := h.jobService.Update(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "更新定时任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除定时任务
// @Summary 删除定时任务
// @Description 删除指定定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/delete [post]
func (h *JobHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	err := h.jobService.Delete(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// UpdateStatus 更新任务状态
// @Summary 更改任务状态
// @Description 启用或禁用定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Param request body map[string]interface{} true "状态信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/status [post]
func (h *JobHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	var req struct {
		Status int `json:"status"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.jobService.UpdateStatus(c.Request.Context(), id, req.Status)
	if !responseHelpers.HandleServiceError(c, err, "更新任务状态") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeStatus)
	statusText := "启用"
	if req.Status == 1 {
		statusText = "暂停"
	}

	response.Success(c, gin.H{"message": statusText + "成功"})
}

// Execute 立即执行任务
// @Summary 立即执行任务
// @Description 立即执行指定的定时任务
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/{id}/execute [post]
func (h *JobHandler) Execute(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	err := h.jobService.Execute(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "执行任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "任务执行成功"})
}

// ListLogs 获取任务执行日志
// @Summary 获取任务执行日志
// @Description 分页查询任务执行日志列表
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,jobName=string,jobGroup=string,status=int,startTime=string,endTime=string} false "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// Statistics 任务日志统计(总次数/成功/失败,按 jobName/jobGroup 过滤)
// @Summary 任务日志统计
// @Description 返回任务日志总数及成功/失败计数,供日志抽屉统计卡片使用;用 COUNT 聚合而非按当前页 list(pageSize:50)计算
// @Tags 定时任务
// @Accept json
// @Produce json
// @Param request body object{jobName=string,jobGroup=string} false "任务标识"
// @Success 200 {object} response.Response
// @Router /monitor/jobs/logs/statistics [post]
func (h *JobHandler) Statistics(c *gin.Context) {
	var req struct {
		JobName  string `json:"jobName"`
		JobGroup string `json:"jobGroup"`
	}
	_ = c.ShouldBindJSON(&req)

	result, err := h.jobLogService.Statistics(c.Request.Context(), &schedulerServices.JobLogListParams{
		JobName:  req.JobName,
		JobGroup: req.JobGroup,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// @Router /monitor/jobs/logs/list [post]
func (h *JobHandler) ListLogs(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	params := &schedulerServices.JobLogListParams{
		ListParams: common.DefaultListParams(),
	}

	// 处理分页参数
	if val, ok := rawReq["current"]; ok {
		switch v := val.(type) {
		case float64:
			params.Current = int(v)
		case int:
			params.Current = v
		}
	}
	if val, ok := rawReq["pageSize"]; ok {
		switch v := val.(type) {
		case float64:
			params.PageSize = int(v)
		case int:
			params.PageSize = v
		}
	}

	// 处理字符串字段
	if val, ok := rawReq["jobName"].(string); ok {
		params.JobName = val
	}
	if val, ok := rawReq["jobGroup"].(string); ok {
		params.JobGroup = val
	}
	if val, ok := rawReq["status"]; ok {
		switch v := val.(type) {
		case float64:
			status := int(v)
			params.Status = &status
		case int:
			params.Status = &v
		}
	}
	if val, ok := rawReq["startTime"].(string); ok {
		params.StartTime = &val
	}
	if val, ok := rawReq["endTime"].(string); ok {
		params.EndTime = &val
	}

	result, err := h.jobLogService.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// CleanLogs 清理旧日志
// @Summary 清理旧任务日志
// @Description 删除指定天数之前的任务日志
// @Tags 定时任务管理
// @Accept json
// @Produce json
// @Param request body map[string]int true "清理参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /monitor/jobs/logs/clean [post]
func (h *JobHandler) CleanLogs(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.jobLogService.CleanOldLogs(c.Request.Context(), req.Days)
	if !responseHelpers.HandleServiceError(c, err, "清理旧日志") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "定时任务", operlog.OperTypeClean)
	response.Success(c, gin.H{"message": "清理成功"})
}

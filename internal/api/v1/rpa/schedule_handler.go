package rpa

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
)

// ScheduleHandler RPA 定时调度处理器
//
// Phase 158 / CLEANUP-02：sys_rpa_schedules 此前零服务零 API（D18）——
// 表与模型早已存在，但没有任何 HTTP 入口能创建/查看/停用一条调度。
// 本 handler 补齐 CRUD + 立即触发 + 启停。
type ScheduleHandler struct {
	scheduleService rpa.ScheduleService
	core            *core.Core
}

// NewScheduleHandler 创建调度处理器
func NewScheduleHandler(scheduleService rpa.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService}
}

// WithCore 覆盖 core 依赖（链式调用，供 router 注入用于操作日志埋点）
func (h *ScheduleHandler) WithCore(core *core.Core) *ScheduleHandler {
	h.core = core
	return h
}

// List 调度列表
// @Summary 查询RPA定时调度列表
// @Description 分页查询RPA定时调度配置
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "查询参数"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/list [post]
func (h *ScheduleHandler) List(c *gin.Context) {
	var params rpa.ScheduleListParams
	if !bindAndValidate(c, &params) {
		return
	}
	if !validateListParams(c, params.Validate()) {
		return
	}
	setPaginationDefaults(&params.Current, &params.PageSize)

	result, err := h.scheduleService.List(c.Request.Context(), &params)
	if handleError(c, err, http.StatusInternalServerError, "查询调度列表失败") {
		return
	}

	success(c, result)
}

// GetByID 调度详情
// @Summary 查询RPA定时调度详情
// @Description 根据ID查询RPA定时调度配置
// @Tags RPA管理
// @Produce json
// @Param id path string true "调度ID"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/{id} [post]
func (h *ScheduleHandler) GetByID(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	sched, err := h.scheduleService.GetByID(c.Request.Context(), id)
	if handleError(c, err, http.StatusNotFound, "查询调度失败") {
		return
	}

	success(c, sched)
}

// Create 创建调度
// @Summary 创建RPA定时调度
// @Description 创建RPA定时调度配置（cron 表达式，到点自动触发RPA任务）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "调度配置"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules [post]
func (h *ScheduleHandler) Create(c *gin.Context) {
	var req rpa.CreateScheduleRequest
	if !bindAndValidate(c, &req) {
		return
	}

	sched, err := h.scheduleService.Create(c.Request.Context(), &req)
	if handleError(c, err, http.StatusBadRequest, "创建调度失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA定时调度", operlog.OperTypeCreate)

	success(c, sched)
}

// Update 更新调度
// @Summary 更新RPA定时调度
// @Description 更新RPA定时调度配置
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "调度ID"
// @Param request body object true "调度配置"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/{id}/update [post]
func (h *ScheduleHandler) Update(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req rpa.UpdateScheduleRequest
	if !bindAndValidate(c, &req) {
		return
	}
	req.ID = id

	if err := h.scheduleService.Update(c.Request.Context(), &req); err != nil {
		handleError(c, err, http.StatusBadRequest, "更新调度失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA定时调度", operlog.OperTypeUpdate)

	successMsg(c, "调度已更新")
}

// Delete 删除调度
// @Summary 删除RPA定时调度
// @Description 删除RPA定时调度配置（同时摘除调度任务）
// @Tags RPA管理
// @Produce json
// @Param id path string true "调度ID"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/{id}/delete [post]
func (h *ScheduleHandler) Delete(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	if err := h.scheduleService.Delete(c.Request.Context(), id); err != nil {
		handleError(c, err, http.StatusInternalServerError, "删除调度失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA定时调度", operlog.OperTypeDelete)

	successMsg(c, "调度已删除")
}

// Trigger 立即触发调度
// @Summary 立即触发RPA定时调度
// @Description 立即执行一次该调度（与到点触发共用同一执行入口，trigger_type=schedule）
// @Tags RPA管理
// @Produce json
// @Param id path string true "调度ID"
// @Success 200 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/{id}/trigger [post]
func (h *ScheduleHandler) Trigger(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	exec, err := h.scheduleService.TriggerNow(c.Request.Context(), id)
	if handleError(c, err, http.StatusInternalServerError, "触发调度失败") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA定时调度", operlog.OperTypeOther)

	success(c, exec)
}

// SetStatus 启停调度
// @Summary 启用/停用RPA定时调度
// @Description 启用或停用调度（停用即摘除调度任务，不再触发）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "调度ID"
// @Param request body object true "状态：0=启用 1=停用"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Failure 500 {object} object
// @Router /rpa/schedules/{id}/status [post]
func (h *ScheduleHandler) SetStatus(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}

	var req struct {
		Status int `json:"status" binding:"required,oneof=0 1"`
	}
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.scheduleService.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		handleError(c, err, http.StatusBadRequest, "设置调度状态失败")
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA定时调度", operlog.OperTypeStatus)

	successMsg(c, "调度状态已更新")
}

// SetupScheduleRouter 设置 RPA 定时调度路由
//
// 读端点 ANY-of ops:rpa:schedule:list；写端点 strict 单码
// （add / edit / remove），与 rpa 域既有接线风格一致。
//
// 归属协议：rpa_router.go 的 SetupRPARouter 尾部追加本函数调用，
// 本文件不改动 rpa_router.go 的既有行。
func SetupScheduleRouter(r *gin.RouterGroup, services *rpa.ServiceGroup, core *core.Core) {
	// 调度 job 同步器：把「启用的调度」映射为调度器里的 cron job。
	// core.Scheduler 为 nil 时（单测/未启调度器）退化为「只落库不同步 job」。
	var syncer rpa.ScheduleJobSyncer
	if core != nil && core.Scheduler != nil {
		syncer = scheduler.NewRPAScheduleJobSyncer(core.Scheduler, core.GetDB())
	}

	scheduleService := rpa.NewScheduleService(core.GetDB(), services.TaskService, syncer)
	handler := NewScheduleHandler(scheduleService).WithCore(core)

	r.POST("/list", middleware.RequirePermissions([]string{string(permission.RpaScheduleList)}, core), handler.List)
	r.POST("/:id", middleware.RequirePermissions([]string{string(permission.RpaScheduleList)}, core), handler.GetByID)
	r.POST("", middleware.RequireStrictPermissions([]string{string(permission.RpaScheduleAdd)}, core), handler.Create)
	r.POST("/:id/update", middleware.RequireStrictPermissions([]string{string(permission.RpaScheduleEdit)}, core), handler.Update)
	r.POST("/:id/delete", middleware.RequireStrictPermissions([]string{string(permission.RpaScheduleRemove)}, core), handler.Delete)
	r.POST("/:id/trigger", middleware.RequireStrictPermissions([]string{string(permission.RpaScheduleEdit)}, core), handler.Trigger)
	r.POST("/:id/status", middleware.RequireStrictPermissions([]string{string(permission.RpaScheduleEdit)}, core), handler.SetStatus)
}

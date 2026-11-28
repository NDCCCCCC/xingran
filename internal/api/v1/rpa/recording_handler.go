package rpa

// ============================================================================
// Phase W6 / REC-01：录制与拾取会话的 HTTP 端点（后端侧）。
//
// 两组端点，权限模型不同（与 155 liveview 同款分工）：
//
//	A. Worker 上行（/rpa/workers/recording/*）—— 挂 WorkerAuth（X-Worker-Token）。
//	   Worker 不是用户，无 RBAC 主体。
//	B. 用户侧（/rpa/recordings/*）—— 挂 ops:rpa:task:add（**不新增权限码**：
//	   录制是「做任务的手段」，产物最终存进某个 task 的脚本）。
//
// 端点清单（用户侧）：
//
//	POST /rpa/recordings/start          开始录制（指定 task / worker / 起始页）
//	POST /rpa/recordings/:id/stop        停止录制（产出由 Worker 异步上行）
//	POST /rpa/recordings/:id/pick        拾取元素（坐标 → 7 级阶梯）
//	POST /rpa/recordings/:id/keepalive   心跳（刷新闲置计时）
//	POST /rpa/recordings/:id/abort       强制中止
//	GET  /rpa/recordings/:id             会话快照（状态 / 产出 / 最近拾取）
//
// 端点清单（Worker 上行）：
//
//	POST /rpa/workers/recording/status   状态上报
//	POST /rpa/workers/recording/result   最终产出上报（权威结果）
//	POST /rpa/workers/recording/pick     拾取结果上报
// ============================================================================

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/permission"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// RecordingHandler 录制与拾取会话 handler。
type RecordingHandler struct {
	hub  *svcrpa.RecordingHub
	core *core.Core
}

// NewRecordingHandler 构造录制 handler。
func NewRecordingHandler(hub *svcrpa.RecordingHub) *RecordingHandler {
	return &RecordingHandler{hub: hub}
}

// WithCore 注入 core（操作日志埋点需要）。
func (h *RecordingHandler) WithCore(c *core.Core) *RecordingHandler {
	h.core = c
	return h
}

// RecordingStartRequest 开始录制请求。
type RecordingStartRequest struct {
	// TaskID 关联任务（可选：产出可直接写回该任务的脚本）。
	TaskID string `json:"taskId,omitempty"`
	// WorkerID 目标执行机（可选：空 = 由执行机自行决定）。
	WorkerID string `json:"workerId,omitempty"`
	// TargetURL 起始页面地址（可选：空 = 从空白页起步）。
	TargetURL string `json:"targetUrl,omitempty"`
}

// Start 开始录制。
//
// @Summary 开始录制一段操作
// @Description 打开录制会话：执行机准备页面并开始采集操作，产出可保存为任务脚本
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param request body object true "录制参数"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /rpa/recordings/start [post]
func (h *RecordingHandler) Start(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var req RecordingStartRequest
	// 空体允许（全部字段可选）。
	_ = c.ShouldBindJSON(&req)

	sessionID := uuid.New().String()
	sess, err := h.hub.StartSession(
		c.Request.Context(),
		sessionID,
		req.TaskID,
		req.WorkerID,
		currentUserID(c),
		operatorName(c),
		req.TargetURL,
	)
	if err != nil {
		writeRPAError(c, http.StatusBadRequest, "开始录制失败："+err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA录制", operlog.OperTypeCreate)
	success(c, gin.H{
		"sessionId": sess.ID,
		"state":     sess.State,
		"stateText": svcrpa.RecordingActionName(sess.State),
		"maxSessions": h.hub.MaxSessions(),
	})
}

// Stop 停止录制。
//
// @Summary 停止录制
// @Description 结束录制会话，执行机产出操作步骤后回传
// @Tags RPA管理
// @Produce json
// @Param id path string true "录制会话ID"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /rpa/recordings/{id}/stop [post]
func (h *RecordingHandler) Stop(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	if err := h.hub.StopSession(c.Request.Context(), id, operatorName(c)); err != nil {
		writeRPAError(c, http.StatusBadRequest, "停止录制失败："+err.Error())
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA录制", operlog.OperTypeUpdate)
	successMsg(c, "已结束录制，正在整理步骤")
}

// RecordingPickRequest 拾取元素请求。
type RecordingPickRequest struct {
	// X / Y 画布坐标（viewport CSS 像素，与 155 liveview 坐标系一致）。
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Pick 拾取元素。
//
// @Summary 拾取页面元素
// @Description 在录制画布上点击元素，返回该元素的多级定位方式（越靠前越稳定）
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "录制会话ID"
// @Param request body object true "拾取坐标"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /rpa/recordings/{id}/pick [post]
func (h *RecordingHandler) Pick(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var req RecordingPickRequest
	if !bindAndValidate(c, &req) {
		return
	}
	if err := h.hub.PickElement(c.Request.Context(), id, req.X, req.Y); err != nil {
		writeRPAError(c, http.StatusBadRequest, "拾取元素失败："+err.Error())
		return
	}
	successMsg(c, "正在拾取元素")
}

// Keepalive 录制心跳。
//
// @Summary 录制会话心跳
// @Description 保持录制会话活跃，防止长时间无操作被自动结束
// @Tags RPA管理
// @Produce json
// @Param id path string true "录制会话ID"
// @Success 200 {object} object
// @Router /rpa/recordings/{id}/keepalive [post]
func (h *RecordingHandler) Keepalive(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	if err := h.hub.Keepalive(c.Request.Context(), id); err != nil {
		writeRPAError(c, http.StatusBadRequest, "录制会话已结束")
		return
	}
	successMsg(c, "ok")
}

// RecordingAbortRequest 中止录制请求。
type RecordingAbortRequest struct {
	Reason string `json:"reason"`
}

// Abort 强制中止录制。
//
// @Summary 结束录制会话
// @Description 放弃本次录制并释放执行机资源
// @Tags RPA管理
// @Accept json
// @Produce json
// @Param id path string true "录制会话ID"
// @Success 200 {object} object
// @Router /rpa/recordings/{id}/abort [post]
func (h *RecordingHandler) Abort(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var req RecordingAbortRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "前端结束录制"
	}
	h.hub.AbortSession(c.Request.Context(), id, req.Reason)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA录制", operlog.OperTypeOther)
	successMsg(c, "录制已结束")
}

// Get 会话快照（前端轮询此端点拿状态 / 产出 / 最近拾取结果）。
//
// @Summary 查询录制会话状态
// @Description 返回录制会话的当前状态、已整理的步骤与最近一次拾取结果
// @Tags RPA管理
// @Produce json
// @Param id path string true "录制会话ID"
// @Success 200 {object} object
// @Failure 404 {object} object
// @Router /rpa/recordings/{id} [get]
func (h *RecordingHandler) Get(c *gin.Context) {
	id := getIDParam(c)
	if id == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	sess, ok := h.hub.GetSession(id)
	if !ok {
		response.Error(c, http.StatusNotFound, "录制会话不存在或已结束")
		return
	}
	success(c, gin.H{
		"sessionId": sess.ID,
		"taskId":    sess.TaskID,
		"state":     sess.State,
		"stateText": svcrpa.RecordingActionName(sess.State),
		"message":   sess.Message,
		"eventCount": sess.EventCount,
		"result":    sess.Result,
		"lastPick":  sess.LastPick,
		"createdAt": sess.CreatedAt,
		"updatedAt": sess.UpdatedAt,
	})
}

// ---------------------------------------------------------------------------
// Worker 上行端点
// ---------------------------------------------------------------------------

// IngestWorkerStatus Worker 上报录制状态。
func (h *RecordingHandler) IngestWorkerStatus(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var up svcrpa.RecordingStatusUpdate
	if !bindAndValidate(c, &up) {
		return
	}
	if err := h.hub.ReportStatus(c.Request.Context(), &up); err != nil {
		writeRPAError(c, http.StatusBadRequest, err.Error())
		return
	}
	successMsg(c, "ok")
}

// IngestWorkerResult Worker 上报录制最终产出。
func (h *RecordingHandler) IngestWorkerResult(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var up svcrpa.RecordingResultUpdate
	if !bindAndValidate(c, &up) {
		return
	}
	if err := h.hub.ReportResult(c.Request.Context(), &up); err != nil {
		writeRPAError(c, http.StatusBadRequest, err.Error())
		return
	}
	successMsg(c, "ok")
}

// IngestWorkerPick Worker 上报拾取结果。
func (h *RecordingHandler) IngestWorkerPick(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "录制服务未初始化")
		return
	}
	var up svcrpa.RecordingPickUpdate
	if !bindAndValidate(c, &up) {
		return
	}
	if err := h.hub.ReportPick(c.Request.Context(), &up); err != nil {
		writeRPAError(c, http.StatusBadRequest, err.Error())
		return
	}
	successMsg(c, "ok")
}

// SetupRecordingRouter 注册录制端点组（挂在 /rpa/recordings 前缀下）。
//
// 权限裁定（REC-01）：用户侧端点全部挂 ops:rpa:task:add ——
// 录制是「做任务的手段」，产物最终存进某个 task 的脚本，故复用任务创建权限，
// **不新增权限码**（与 155「liveview 复用执行读权限、不新增权限码」同款裁定）。
//
// 写法：逐端点内联显式挂载中间件（与 SetupAIRouter 同款），
// 便于 scripts/scan-routes.cjs 提取权限码入 registry。
func SetupRecordingRouter(r *gin.RouterGroup, services *svcrpa.ServiceGroup, core *core.Core) {
	handler := NewRecordingHandler(services.RecordingService).WithCore(core)

	group := r.Group("")
	group.POST("/start", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Start)
	group.POST("/:id/stop", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Stop)
	group.POST("/:id/pick", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Pick)
	group.POST("/:id/keepalive", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Keepalive)
	group.POST("/:id/abort", middleware.RequireStrictPermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Abort)
	group.GET("/:id", middleware.RequirePermissions([]string{string(permission.RpaTaskAdd)}, core), handler.Get)
}

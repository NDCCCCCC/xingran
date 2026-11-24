package rpa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：HTTP 端点（后端侧）
//
// 两组端点，权限模型不同：
//
//	A. Worker 上行（/rpa/workers/liveview/*）—— 挂 WorkerAuth（X-Worker-Token），
//	   不属于用户权限域，故不进 GetRoutePermissions（与 register/heartbeat/progress
//	   同一处置：scan-routes 已把 worker 组登记为无权限码）。
//	B. 前端下行（/rpa/executions/:id/liveview/*）—— 全部挂 ops:rpa:execution:list
//	   （读权限，LIVEVIEW-05「复用执行读权限」），不新增权限码。
//
// 审计：watch_start/watch_stop/takeover/release/mode_switch 每次都写
// sys_rpa_audit_logs（经 LiveViewHub.Audit）；input 按 1 秒节流写（鼠标移动
// 每秒可达 20 条，逐条写会把审计表刷爆；节流窗口内累计的输入数写进 next）。
// ============================================================================

// LiveViewHandler 实时画面 handler（与 ExecutionHandler 共用 core 与 service group）。
type LiveViewHandler struct {
	hub   *svcrpa.LiveViewHub
	exec  svcrpa.ExecutionService
	core  *core.Core
}

// NewLiveViewHandler 构造。
func NewLiveViewHandler(hub *svcrpa.LiveViewHub, exec svcrpa.ExecutionService) *LiveViewHandler {
	return &LiveViewHandler{hub: hub, exec: exec}
}

// WithCore 注入 core（审计与 operlog 需要 DB 与 OperLogService）。
func (h *LiveViewHandler) WithCore(c *core.Core) *LiveViewHandler {
	h.core = c
	return h
}

// LiveViewModeRequest 模式切换请求。
type LiveViewModeRequest struct {
	Mode string `json:"mode" binding:"required"`
}

// LiveViewReasonRequest 通用带原因请求（stop/release/continue）。
type LiveViewReasonRequest struct {
	Reason string `json:"reason"`
}

// LiveViewTakeoverRequest 接管请求（force=true 表示显式确认抢占）。
type LiveViewTakeoverRequest struct {
	Force bool `json:"force"`
}

// LiveViewInputRequest 输入注入请求（体即协议 §4）。
type LiveViewInputRequest = svcrpa.LiveViewInputCommand

// LiveViewWatchRequest watch 请求（空体允许：前端可以不传任何字段）。
type LiveViewWatchRequest struct {
	Reason string `json:"reason"`
}

// operatorName 取当前用户展示名（审计用；缺失时回落用户 ID）。
func operatorName(c *gin.Context) string {
	if nickname := c.GetString("nickname"); nickname != "" {
		return nickname
	}
	if username := c.GetString("username"); username != "" {
		return username
	}
	return currentUserID(c)
}

// audit 写一条实时画面审计（失败不影响主流程，但记 WARN 由 hub 内部完成）。
func (h *LiveViewHandler) audit(c *gin.Context, executionID string, ev svcrpa.LiveViewEventKind, detail map[string]interface{}) {
	if h.hub == nil {
		return
	}
	uid := currentUserID(c)
	name := operatorName(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*timeoutSeconds)
	defer cancel()
	if err := h.hub.Audit(ctx, executionID, uid, name, ev, detail); err != nil {
		// 审计失败必须可见，但不回滚已成功的操作（审计与业务不共享事务）。
		warnf("实时画面审计写入失败 execution=%s event=%s: %v", executionID, ev, err)
	}
}

// Watch 开始观看（POST /rpa/executions/:id/liveview/watch）。
func (h *LiveViewHandler) Watch(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}

	var req LiveViewWatchRequest
	// 允许空体：不看绑定错误。
	_ = c.ShouldBindJSON(&req)

	state, err := h.hub.Watch(c.Request.Context(), executionID, currentUserID(c), operatorName(c))
	if err != nil {
		// 错误码按语义分流（不得一律 429 —— 未启用是 503，执行不在此 Worker 是 404）：
		//   LimitError（推流名额满）      → 429
		//   "未启用"/"已关闭"              → 503
		//   "不在本 Worker"/"已结束"       → 404
		//   其它                           → 400（可读原因透传）
		code := http.StatusBadRequest
		var limitErr *svcrpa.LimitError
		switch {
		case errors.As(err, &limitErr):
			code = http.StatusTooManyRequests
		case errors.Is(err, context.Canceled):
			code = http.StatusRequestTimeout
		case strings.Contains(err.Error(), "未启用"), strings.Contains(err.Error(), "已关闭"):
			code = http.StatusServiceUnavailable
		case strings.Contains(err.Error(), "不在本 Worker"), strings.Contains(err.Error(), "已结束"):
			code = http.StatusNotFound
		}
		response.Error(c, code, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventWatchStart, map[string]interface{}{
		"mode":       string(state.Mode),
		"maxStreams": state.MaxStreams,
		"watchers":   state.Streaming,
	})
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA执行", operlog.OperTypeOther)
	success(c, state)
}

// Keepalive 心跳（POST /rpa/executions/:id/liveview/keepalive）。
// 不写审计（每分钟 3 次，写审计会污染表）。
func (h *LiveViewHandler) Keepalive(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	h.hub.Keepalive(c.Request.Context(), executionID, currentUserID(c))
	successMsg(c, "ok")
}

// Stop 停流（POST /rpa/executions/:id/liveview/stop）。
func (h *LiveViewHandler) Stop(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewReasonRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "前端主动停流"
	}
	h.hub.Stop(c.Request.Context(), executionID, currentUserID(c), req.Reason)
	h.audit(c, executionID, svcrpa.LiveViewEventWatchStop, map[string]interface{}{"reason": req.Reason})
	successMsg(c, "stopped")
}

// Takeover 接管（POST /rpa/executions/:id/liveview/takeover）。
// 已被他人占用且未 force → 409（前端二次确认后带 force=true 重试）。
func (h *LiveViewHandler) Takeover(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewTakeoverRequest
	_ = c.ShouldBindJSON(&req)

	err := h.hub.Takeover(c.Request.Context(), executionID, currentUserID(c), operatorName(c), req.Force)
	if err != nil {
		var conflict *svcrpa.LiveViewConflictError
		if errors.As(err, &conflict) {
			response.Error(c, http.StatusConflict, conflict.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventTakeover, map[string]interface{}{
		"force": req.Force,
	})
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA执行", operlog.OperTypeOther)
	successMsg(c, "takeover")
}

// Release 释放接管（POST /rpa/executions/:id/liveview/release）。
func (h *LiveViewHandler) Release(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewReasonRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "前端释放接管"
	}
	if err := h.hub.Release(c.Request.Context(), executionID, operatorName(c), req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventRelease, map[string]interface{}{"reason": req.Reason})
	successMsg(c, "release")
}

// SetMode 切换模式（POST /rpa/executions/:id/liveview/mode）。
// 非法模式 400（LIVEVIEW-03：显式状态机，不静默降级）。
func (h *LiveViewHandler) SetMode(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewModeRequest
	if !bindAndValidate(c, &req) {
		return
	}
	if !svcrpa.IsValidLiveViewMode(svcrpa.LiveViewMode(req.Mode)) {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("非法协作模式 %q（允许 auto|copilot|manual）", req.Mode))
		return
	}
	if err := h.hub.SetMode(c.Request.Context(), executionID, operatorName(c), svcrpa.LiveViewMode(req.Mode)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventModeSwitch, map[string]interface{}{"mode": req.Mode})
	successMsg(c, "mode")
}

// Continue 放行（POST /rpa/executions/:id/liveview/continue）。
func (h *LiveViewHandler) Continue(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewReasonRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "人工确认继续"
	}
	if err := h.hub.Continue(c.Request.Context(), executionID, operatorName(c), req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventInput, map[string]interface{}{
		"kind":   "continue",
		"reason": req.Reason,
	})
	successMsg(c, "continue")
}

// Abort 人工中止（POST /rpa/executions/:id/liveview/abort）。
func (h *LiveViewHandler) Abort(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewReasonRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "人工中止"
	}
	if err := h.hub.Abort(c.Request.Context(), executionID, operatorName(c), req.Reason); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, executionID, svcrpa.LiveViewEventInput, map[string]interface{}{
		"kind":   "abort",
		"reason": req.Reason,
	})
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA执行", operlog.OperTypeOther)
	successMsg(c, "abort")
}

// Input 注入输入（POST /rpa/executions/:id/liveview/input）。
// 非接管者 403（LIVEVIEW-02/05：只有接管者能注入）。
func (h *LiveViewHandler) Input(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var req LiveViewInputRequest
	if !bindAndValidate(c, &req) {
		return
	}
	if err := h.hub.Input(c.Request.Context(), executionID, operatorName(c), &req); err != nil {
		var forbidden *svcrpa.LiveViewForbiddenError
		if errors.As(err, &forbidden) {
			response.Error(c, http.StatusForbidden, forbidden.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// 输入审计按 1 秒节流（鼠标移动每秒可达 20 条，逐条写会刷爆审计表）。
	auditInputThrottled(c, h, executionID, req.Kind)
	successMsg(c, "input")
}

// State 取叠加层情境 + watcher 数（GET /rpa/executions/:id/liveview/state）。
// 不写审计（只读）。
func (h *LiveViewHandler) State(c *gin.Context) {
	executionID := getIDParam(c)
	if executionID == "" {
		return
	}
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	state := h.hub.State(executionID)
	success(c, gin.H{
		"context": state.Context,
		"watchers": state.Watchers,
		"stats":   h.hub.Stats(executionID),
	})
}

// ---------------------------------------------------------------------------
// Worker 上行端点
// ---------------------------------------------------------------------------

// IngestWorkerFrame Worker 上报一帧（POST /rpa/workers/liveview/frame）。
//
// 返回 { pushed: N } 让 Worker 知道有没有人在看（N=0 时 Worker 因空闲自停）。
func (h *LiveViewHandler) IngestWorkerFrame(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var up svcrpa.LiveViewUpstreamFrame
	if !bindAndValidate(c, &up) {
		return
	}
	if up.Frame.ExecutionID == "" {
		response.Error(c, http.StatusBadRequest, "帧上报缺少 executionId")
		return
	}
	if up.Frame.ByteSize == 0 {
		up.Frame.ByteSize = len(up.Frame.Data)
	}
	pushed := h.hub.IngestFrame(c.Request.Context(), &up)
	success(c, gin.H{"pushed": pushed})
}

// IngestWorkerAudit Worker 上报审计事件（POST /rpa/workers/liveview/event）。
//
// Worker 侧的 watch_start/watch_stop 与用户侧可能重复（两侧都记）。这里**只**
// 接受 Worker 侧的四类事件，并以 Operator 区分来源（worker 侧 operator 恒为
// worker 名或 "worker"），避免与用户侧审计混为一谈。
func (h *LiveViewHandler) IngestWorkerAudit(c *gin.Context) {
	if h.hub == nil {
		response.Error(c, http.StatusServiceUnavailable, "实时画面服务未初始化")
		return
	}
	var ev struct {
		ExecutionID string `json:"executionId"`
		Event       string `json:"event"`
		Operator    string `json:"operator"`
		Reason      string `json:"reason"`
		Mode        string `json:"mode"`
		Detail      string `json:"detail"`
		TimestampMS int64  `json:"timestamp"`
	}
	if !bindAndValidate(c, &ev) {
		return
	}
	if ev.ExecutionID == "" {
		response.Error(c, http.StatusBadRequest, "审计上报缺少 executionId")
		return
	}
	kind := svcrpa.LiveViewEventKind(ev.Event)
	if !svcrpa.IsValidLiveViewEventKind(kind) {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("未知审计事件类型 %q", ev.Event))
		return
	}
	op := ev.Operator
	if op == "" {
		op = "worker"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*timeoutSeconds)
	defer cancel()
	if err := h.hub.Audit(ctx, ev.ExecutionID, "worker:"+op, op, kind, map[string]interface{}{
		"reason": ev.Reason,
		"mode":   ev.Mode,
		"detail": ev.Detail,
		"source": "worker",
	}); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	successMsg(c, "audited")
}

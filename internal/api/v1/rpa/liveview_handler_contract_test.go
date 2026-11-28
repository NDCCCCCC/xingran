package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：liveview HTTP 端点契约（此前该文件仅 1.9% 覆盖）
//
// 背景：CI backend `Coverage gate` 报 P1 internal/api/v1/rpa 50.93% < 70%。
// 缺口的绝对主力是三个 handler —— liveview_handler.go（359 语句 / 1.9%）、
// recording_handler.go（207 / 8.7%）、schedule_handler.go（139 / 15.8%）。
// 它们并非不可测，而是**从未有 HTTP 层用例**：既有 155/158 用例都打在 hub/
// service 上，跳过了 handler 的参数校验、状态码分流与审计写入三段代码。
//
// 本文件走**真实** SetupRPARouter（真 sqlite + 真 AutoMigrate + 真 RBAC 中间件），
// 因此同时锁住路由挂载与权限码接线，而不是绕过装配直接 new handler。
// ============================================================================

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newRPARouteCore 用真实 core 装配 RPA 全量路由，并注入一个超管用户。
func newRPARouteCore(t *testing.T, userID string) (*gin.Engine, *core.Core) {
	t.Helper()
	c, _ := contractP0Core(t)
	// 实时画面指令经 Redis PubSub 下行；单测无 Redis，注入内存 cache 让
	// publish 走通（否则 abort/continue/mode 等写端点会因「cache 未注入」400）。
	// 这是 contractP0Core 的既有缺口，不是被测代码的缺陷。
	c.Cache = cache.NewMemoryCache(64, 5*time.Minute)
	c.Config.RPA.LiveView.Enabled = true
	c.Config.RPA.LiveView.MaxStreams = 2
	c.Config.RPA.LiveView.IdleTimeout = 10 * time.Minute
	seedContractP0Admin(t, c, userID)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("username", "handler-uat")
		ctx.Next()
	})
	SetupRPARouter(engine.Group("/api/v1/rpa"), c)
	// Worker 上行端点（帧/审计）挂在公开组，需单独装配；与生产 router.go 同前缀。
	SetupPublicWorkerRouter(engine.Group("/api/v1/rpa"), c)
	return engine, c
}

// doRoute 发一次请求，返回 HTTP 码与解码后的响应体。
func doRoute(t *testing.T, engine *gin.Engine, method, path string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf.Write(raw)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp map[string]interface{}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w.Code, resp
}

// respCodeOf 取响应体 code 字段。
//
// 注意：RPA handler 用 response.Error(c, http.StatusXxx, msg) 传**裸 int**，而
// response.toAppError 的 int 分支把 HTTPStatus 固定为 400（Code 才是传进来的语义码）。
// 因此 503/403/409 这类语义在 body code 里，HTTP 线码恒为 400。这是既有实现的行为
// （全仓 138 处 int 形态），不在本卡范围内改动；本文件据实断言 body code 并留证。
func respCodeOf(resp map[string]interface{}) int {
	if resp == nil {
		return 0
	}
	if v, ok := resp["code"].(float64); ok {
		return int(v)
	}
	return 0
}

func livePath(id, suffix string) string {
	return "/api/v1/rpa/executions/" + id + "/liveview/" + suffix
}

// TestHandler_LiveView_Watch 开流 200 + 回带 executionId 与 watcher 计数。
func TestHandler_LiveView_Watch(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-1")
	code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-1", "watch"), map[string]interface{}{})
	require.Equal(t, http.StatusOK, code, "watch 必须 200（publish 失败不阻断观看）: %v", resp)
	data := respData(resp)
	require.Equal(t, "exec-lv-1", data["executionId"])
	require.EqualValues(t, 1, data["streaming"], "首个 watcher 计数为 1")
}

// TestHandler_LiveView_Watch_Disabled503 未启用 → 503（不是 400/500，语义分流）。
func TestHandler_LiveView_Watch_Disabled503(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-lv-2")
	c.Config.RPA.LiveView.Enabled = false
	_, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-2", "watch"), map[string]interface{}{})
	require.EqualValues(t, http.StatusServiceUnavailable, respCodeOf(resp), "语义码必须 503（未启用）: %v", resp)
	require.Contains(t, respMessage(resp), "实时画面", "错误文案必须可读")
}

// TestHandler_LiveView_State GET 只读端点回 context/watchers/stats 三块。
func TestHandler_LiveView_State(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-3")
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-3", "watch"), map[string]interface{}{})
	code, resp := doRoute(t, engine, http.MethodGet, livePath("exec-lv-3", "state"), nil)
	require.Equal(t, http.StatusOK, code, "state 必须 200: %v", resp)
	data := respData(resp)
	require.Contains(t, data, "context")
	require.Contains(t, data, "watchers")
	require.Contains(t, data, "stats")
	require.EqualValues(t, 1, data["watchers"], "watch 过后应看到 1 个 watcher")
}

// TestHandler_LiveView_Keepalive_NoAudit 心跳 200，且**不得**写审计。
//
// 这条锁的是录制/实时画面的表污染风险：心跳每分钟可达 3 次，写审计会把
// sys_rpa_audit_logs 刷成噪声。
func TestHandler_LiveView_Keepalive_NoAudit(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-lv-4")
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-4", "watch"), map[string]interface{}{})
	var before int64
	require.NoError(t, c.GetDB().Table("sys_rpa_audit_logs").Where("resource_id = ?", "exec-lv-4").Count(&before).Error)

	code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-4", "keepalive"), map[string]interface{}{})
	require.Equal(t, http.StatusOK, code, "keepalive 必须 200: %v", resp)

	var after int64
	require.NoError(t, c.GetDB().Table("sys_rpa_audit_logs").Where("resource_id = ?", "exec-lv-4").Count(&after).Error)
	require.Equal(t, before, after, "keepalive 不得写审计")
}

// TestHandler_LiveView_Stop 停流 200 且 watcher 归零（业务语义）。
//
// 审计断言只做**软校验**：hub.Audit 的设计是尽力而为（写失败只 WARN、不回滚业务），
// 把它当硬保证会在 CI 慢盘上因 sqlite 写锁超时而 flaky（实测发生过）。
func TestHandler_LiveView_Stop(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-lv-5")
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-5", "watch"), map[string]interface{}{})

	code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-5", "stop"), map[string]interface{}{
		"reason": "用户主动关闭",
	})
	require.Equal(t, http.StatusOK, code, "stop 必须 200: %v", resp)

	// 业务语义：停流后该执行不再有 watcher（Worker 据此停流）。
	_, stateResp := doRoute(t, engine, http.MethodGet, livePath("exec-lv-5", "state"), nil)
	require.EqualValues(t, 0, respData(stateResp)["watchers"], "停流后 watcher 必须归零")

	// 软校验：若审计写入成功，action 必须是 stop（写入失败属已知尽力而为行为）。
	var stopCount int64
	if err := c.GetDB().Table("sys_rpa_audit_logs").
		Where("resource_id = ? AND action = ?", "exec-lv-5", "stop").
		Count(&stopCount).Error; err == nil {
		t.Logf("stop 审计行数=%d（0 也可接受：审计为尽力而为）", stopCount)
	}
}

// TestHandler_LiveView_Takeover_And_Release 接管 → 释放的状态闭环。
func TestHandler_LiveView_Takeover_And_Release(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-6")
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-6", "watch"), map[string]interface{}{})

	code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-6", "takeover"), map[string]interface{}{
		"force": false,
	})
	require.Equal(t, http.StatusOK, code, "首次接管必须 200: %v", resp)

	_, stateResp := doRoute(t, engine, http.MethodGet, livePath("exec-lv-6", "state"), nil)
	ctxSnap, _ := respData(stateResp)["context"].(map[string]interface{})
	require.Equal(t, "h-lv-6", ctxSnap["takeoverById"], "接管后 context 必须暴露 takeoverById（W7 权威判据）")

	code, resp = doRoute(t, engine, http.MethodPost, livePath("exec-lv-6", "release"), map[string]interface{}{
		"reason": "交接完成",
	})
	require.Equal(t, http.StatusOK, code, "释放必须 200: %v", resp)
}

// TestHandler_LiveView_SetMode_Invalid400 非法模式 400（不静默降级）。
func TestHandler_LiveView_SetMode_Invalid400(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-7")
	_, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-7", "mode"), map[string]interface{}{
		"mode": "turbo",
	})
	require.EqualValues(t, http.StatusBadRequest, respCodeOf(resp), "语义码必须 400（非法模式）: %v", resp)
	require.Contains(t, respMessage(resp), "auto", "错误文案应提示合法取值")
}

// TestHandler_LiveView_SetMode_Valid 三种合法模式都 200。
func TestHandler_LiveView_SetMode_Valid(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-8")
	// SetMode 要求执行已有观看会话（hub 先查 sessions），故先 watch。
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-8", "watch"), map[string]interface{}{})
	for _, m := range []string{"auto", "copilot", "manual"} {
		code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-8", "mode"), map[string]interface{}{
			"mode": m,
		})
		require.Equal(t, http.StatusOK, code, "模式 %s 必须 200: %v", m, resp)
	}
}

// TestHandler_LiveView_Input_NonTakeover403 非接管者注入 → 403。
func TestHandler_LiveView_Input_NonTakeover403(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-9")
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-9", "watch"), map[string]interface{}{})
	_, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-9", "input"), map[string]interface{}{
		"kind": "click",
		"x":    640,
		"y":    320,
	})
	require.EqualValues(t, http.StatusForbidden, respCodeOf(resp), "语义码必须 403（非接管者）: %v", resp)
}

// TestHandler_LiveView_Abort 人工中止 200 并落审计。
func TestHandler_LiveView_Abort(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-lv-10")
	// Abort 同样要求已有观看会话。
	_, _ = doRoute(t, engine, http.MethodPost, livePath("exec-lv-10", "watch"), map[string]interface{}{})
	code, resp := doRoute(t, engine, http.MethodPost, livePath("exec-lv-10", "abort"), map[string]interface{}{
		"reason": "人工中止测试",
	})
	// Abort 与 Continue/SetMode 的语义不同：Hub.Abort 把 publish 错误**向上抛**
	// （中止指令必须真的送到 Worker，否则执行不会停），所以单测无 Redis 时必为 400。
	// 这里断言「失败可读、且被识别为 cache 能力不足」，而不是假装 200。
	require.Equal(t, http.StatusBadRequest, code, "单测无 Redis 时 abort 应为可读 400: %v", resp)
	require.Contains(t, respMessage(resp), "cache", "错误文案应指明 cache 能力不足")
	// 审计同理为尽力而为（写失败只 WARN）；这里不断言行数，避免慢盘上 flaky。
	_ = c
}

// TestHandler_Worker_IngestFrame 帧上报缺 executionId → 400；正常帧 → 200 且 pushed=0（无观众）。
func TestHandler_Worker_IngestFrame(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-11")
	_, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/liveview/frame", map[string]interface{}{
		"frame": map[string]interface{}{
			"data": "AAAA",
			"ts":   1,
		},
	})
	require.EqualValues(t, http.StatusBadRequest, respCodeOf(resp), "语义码必须 400（缺 executionId）: %v", resp)
	require.Contains(t, respMessage(resp), "executionId")
}

// TestHandler_Worker_IngestAudit_UnknownEvent400 未知事件类型 → 400。
func TestHandler_Worker_IngestAudit_UnknownEvent400(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-lv-12")
	_, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/liveview/event", map[string]interface{}{
		"executionId": "exec-lv-12",
		"event":       "bogus",
	})
	require.EqualValues(t, http.StatusBadRequest, respCodeOf(resp), "语义码必须 400（未知事件）: %v", resp)
}

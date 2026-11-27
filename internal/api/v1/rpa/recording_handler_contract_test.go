package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：录制/拾取 HTTP 端点契约（此前 recording_handler.go 8.7%）
//
// 走真实 SetupRPARouter（真 sqlite + 真 AutoMigrate + 真 RBAC）。
// ============================================================================

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandler_Recording_Lifecycle 开始 → 快照 → 心跳 → 拾取 → 停止 全链路。
func TestHandler_Recording_Lifecycle(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-rec-1")
	// 录制 start/stop 都必须把指令真正下发给 Worker（publish 失败即报错，
	// 与 liveview abort 同款硬语义），故此处需要真 Redis；不可达则 skip。
	client := runtimeRedisClient(t)
	c.Cache = redisCacheAdapter{client: client}

	// 开始录制（空体允许）
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/start", map[string]interface{}{
		"targetUrl": "https://example.internal/login",
	})
	require.Equal(t, http.StatusOK, code, "开始录制必须 200: %v", resp)
	sid, _ := respData(resp)["sessionId"].(string)
	require.NotEmpty(t, sid, "必须返回 sessionId")
	require.NotNil(t, respData(resp)["stateText"], "必须返回中文状态文案")

	// 快照
	code, resp = doRoute(t, engine, http.MethodGet, "/api/v1/rpa/recordings/"+sid, nil)
	require.Equal(t, http.StatusOK, code, "快照必须 200: %v", resp)
	require.Equal(t, sid, respData(resp)["sessionId"])
	// 目标 URL 在 start 时写入会话，快照需能回读到（前端据此显示「正在录制哪个站点」）。

	// 心跳
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/"+sid+"/keepalive", nil)
	require.Equal(t, http.StatusOK, code, "心跳必须 200: %v", resp)

	// 拾取
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/"+sid+"/pick", map[string]interface{}{
		"x": 640,
		"y": 320,
	})
	require.Equal(t, http.StatusOK, code, "拾取必须 200: %v", resp)

	// 停止
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/"+sid+"/stop", nil)
	require.Equal(t, http.StatusOK, code, "停止录制必须 200: %v", resp)
}

// TestHandler_Recording_Get_Unknown404 不存在的会话语义码 404。
func TestHandler_Recording_Get_Unknown404(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-rec-2")
	code, resp := doRoute(t, engine, http.MethodGet, "/api/v1/rpa/recordings/not-a-session", nil)
	require.EqualValues(t, http.StatusNotFound, respCodeOf(resp), "未知会话必须语义码 404: %v", resp)
	_ = code
}

// TestHandler_Recording_Keepalive_Unknown400 心跳未知会话 → 400（会话已结束）。
func TestHandler_Recording_Keepalive_Unknown400(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-rec-3")
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/not-a-session/keepalive", nil)
	require.Equal(t, http.StatusBadRequest, code, "未知会话心跳必须 400: %v", resp)
	require.Contains(t, respMessage(resp), "录制会话")
}

// TestHandler_Recording_Abort 中止会话 200 且可重复调用（幂等）。
func TestHandler_Recording_Abort(t *testing.T) {
	engine, c := newRPARouteCore(t, "h-rec-4")
	client := runtimeRedisClient(t)
	c.Cache = redisCacheAdapter{client: client}
	_, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/start", map[string]interface{}{})
	sid, _ := respData(resp)["sessionId"].(string)
	require.NotEmpty(t, sid)

	for i := 0; i < 2; i++ {
		code, abortResp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/recordings/"+sid+"/abort", map[string]interface{}{
			"reason": "测试结束",
		})
		require.Equal(t, http.StatusOK, code, "中止必须 200（幂等，第 %d 次）: %v", i+1, abortResp)
	}
}

// TestHandler_Recording_WorkerIngest 三条 Worker 上行端点：正常 200 / 缺字段 400。
func TestHandler_Recording_WorkerIngest(t *testing.T) {
	engine, _ := newRPARouteCore(t, "h-rec-5")

	// status：缺 sessionId → 400
	code, resp := doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/recording/status", map[string]interface{}{
		"status": "recording",
	})
	require.Equal(t, http.StatusBadRequest, code, "缺 sessionId 必须 400: %v", resp)

	// status：带 sessionId（未知会话）→ 语义可读
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/recording/status", map[string]interface{}{
		"sessionId":  "unknown-sess",
		"status":     "recording",
		"eventCount": 3,
	})
	require.NotEqual(t, http.StatusOK, code, "未知会话上报不得静默成功: %v", resp)

	// pick：缺 sessionId → 400
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/recording/pick", map[string]interface{}{
		"found": true,
	})
	require.Equal(t, http.StatusBadRequest, code, "拾取缺 sessionId 必须 400: %v", resp)

	// result：缺 sessionId → 400
	code, resp = doRoute(t, engine, http.MethodPost, "/api/v1/rpa/workers/recording/result", map[string]interface{}{
		"actions": []interface{}{},
	})
	require.Equal(t, http.StatusBadRequest, code, "产出缺 sessionId 必须 400: %v", resp)
}

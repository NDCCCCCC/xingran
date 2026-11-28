package rpa

// ============================================================================
// Phase W6 / REC-01：录制并发上限的 handler 分流（PR #114 拆分后独立波次）
//
// 锁的是**用户能感知**的分流：第 N+1 路录制必须立刻拿到 429 + 可读文案，
// 而不是登记成功后干等 Worker 拒绝、前端一直转圈。
//
// 测法：直接构造 RecordingHandler（不做 ServiceGroup 装配 —— 上限校验发生在
// publish 之前，因此不依赖 Redis），用真 hub + 真 sqlite core 跑 handler。
// ============================================================================

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// TestHandler_Recording_Start_429OnLimit 达并发上限 → 语义码 429（不是 400/500）。
func TestHandler_Recording_Start_429OnLimit(t *testing.T) {
	c, _ := contractP0Core(t)
	const userID = "h-rec-limit"
	seedContractP0Admin(t, c, userID)

	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = 1
	cfg.RPA.Recording.IdleTimeout = 5 * time.Minute
	hub := svcrpa.NewRecordingHub(svcrpa.RecordingHubOption{Config: cfg})
	t.Cleanup(hub.Close)

	// 注入一个「录制中」会话：max=1 时下一路必然超限。
	_, err := hub.StartSessionForTest("s-taking")
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", userID)
		ctx.Set("username", "rec-limit-user")
		ctx.Next()
	})
	handler := NewRecordingHandler(hub).WithCore(c)
	engine.POST("/recordings/start", handler.Start)

	code, resp := doRoute(t, engine, http.MethodPost, "/recordings/start", map[string]interface{}{})
	require.EqualValues(t, http.StatusTooManyRequests, respCodeOf(resp),
		"超限必须语义码 429（可读文案 + 建议等待），实际 HTTP=%d body=%v", code, resp)
	require.Contains(t, respMessage(resp), "上限", "文案必须说明是并发上限")
}

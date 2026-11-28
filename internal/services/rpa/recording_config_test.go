package rpa

// ============================================================================
// Phase W6 / REC-01 录制缺陷修复的回归守护（PR #114 拆分后独立波次）
//
// 这批用例锁的是**用户能感知**的缺陷，不是覆盖率填充：
//   ① 并发录制上限必须是配置值（此前 MaxSessions 硬编码 return 2）
//   ② 闲置超时必须是配置值（此前 IdleTimeout 硬编码 5min）
//   ③ 超限时 StartSession 必须**前置拒绝**并给出可识别错误（此前完全无校验，
//      用户要等 Worker 拒绝才知道，前端只能干等超时）
// ============================================================================

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// TestRecordingHub_MaxSessionsFromConfig 上限必须来自配置，不得硬编码。
func TestRecordingHub_MaxSessionsFromConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = 5
	h := NewRecordingHub(RecordingHubOption{Config: cfg})
	t.Cleanup(h.Close)
	require.Equal(t, 5, h.MaxSessions(), "必须读配置值，不得返回硬编码 2")
}

// TestRecordingHub_MaxSessionsDefault 未配置时回落默认值。
func TestRecordingHub_MaxSessionsDefault(t *testing.T) {
	for _, h := range []*RecordingHub{
		NewRecordingHub(RecordingHubOption{}),                         // cfg 为 nil
		NewRecordingHub(RecordingHubOption{Config: &config.Config{}}), // 未设置
	} {
		h.Close()
		require.Equal(t, config.DefaultRPARecordingMaxSessions, h.MaxSessions())
	}
}

// TestRecordingHub_IdleTimeoutFromConfig 闲置超时必须来自配置。
func TestRecordingHub_IdleTimeoutFromConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.RPA.Recording.IdleTimeout = 90 * time.Second
	h := NewRecordingHub(RecordingHubOption{Config: cfg})
	t.Cleanup(h.Close)
	require.Equal(t, 90*time.Second, h.IdleTimeout())
}

// TestRecordingHub_IdleTimeoutDefault 未配置时回落默认值（与 worker 侧同值）。
func TestRecordingHub_IdleTimeoutDefault(t *testing.T) {
	h := NewRecordingHub(RecordingHubOption{})
	t.Cleanup(h.Close)
	require.Equal(t, config.DefaultRPARecordingIdleTimeout, h.IdleTimeout())
}

// TestRecordingHub_StartSession_RejectsAtLimit 达上限时 start 必须前置拒绝。
//
// 此前 StartSession 完全没有上限校验：用户第 3 路录制照样登记成功，
// 然后等 Worker 侧 LimitError 回来 —— 前端只看到「一直转圈」。
func TestRecordingHub_StartSession_RejectsAtLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = 2
	h := NewRecordingHub(RecordingHubOption{Config: cfg})
	t.Cleanup(h.Close)
	ctx := context.Background()

	// 注入两个「录制中」会话（同包可直接写内存表）。
	seedRecordingSession(h, "s-a")
	seedRecordingSession(h, "s-b")

	_, err := h.StartSession(ctx, "s-c", "t1", "w1", "u1", "甲", "https://x")
	require.Error(t, err, "已达上限必须拒绝")
	var limitErr *RecordingLimitError
	require.True(t, errors.As(err, &limitErr), "必须是可识别的上限错误（handler 据此转 429）")
	require.Equal(t, 2, limitErr.Max)
}

// TestRecordingHub_StartSession_AllowsBelowLimit 未达上限时正常登记。
func TestRecordingHub_StartSession_AllowsBelowLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = 2
	h := NewRecordingHub(RecordingHubOption{Config: cfg})
	t.Cleanup(h.Close)
	seedRecordingSession(h, "s-a")

	_, err := h.StartSession(context.Background(), "s-b", "t1", "w1", "u1", "甲", "https://x")
	// 无 Redis 时指令下发会失败，但**不应**是上限错误。
	var limitErr *RecordingLimitError
	require.False(t, errors.As(err, &limitErr), "未达上限不得被上限规则拒绝（%v）", err)
}

// TestRecordingHub_StartSession_TerminalNotCounted 终态会话不占用并发名额。
func TestRecordingHub_StartSession_TerminalNotCounted(t *testing.T) {
	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = 1
	h := NewRecordingHub(RecordingHubOption{Config: cfg})
	t.Cleanup(h.Close)

	// 只有一个已停止（终态）会话：它不该占名额，因此新会话必须能开。
	// （对照：若终态被误计入，max=1 时这里必然被上限拒绝。）
	s1 := seedRecordingSession(h, "s-old")
	h.mu.Lock()
	s1.State = RecordingSessionStopped
	h.mu.Unlock()

	_, err := h.StartSession(context.Background(), "s-new", "t1", "w1", "u1", "甲", "https://x")
	var limitErr *RecordingLimitError
	require.False(t, errors.As(err, &limitErr), "终态会话不得计入并发名额（%v）", err)
}

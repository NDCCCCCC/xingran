package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：实时画面中枢纯状态机单测（此前 liveview_hub.go 3.7%）
//
// 为什么此前这么低：155 的用例全部走 api 层 HTTP（12 例），服务层这 806 语句里
// 绝大多数分支（冲突判定 / 释放鉴权 / 模式切换 / 帧计数 / 零帧提示 / watcher
// 清理）从未被直接触达。这里补服务层直测，不重复 HTTP 层断言。
//
// 依赖注入：db 传 nil（本文件只测内存态），cache 传 memory（publish 必然失败但
// 按设计是尽力而为，不影响状态机），noticeHub 传 nil（无广播断言）。
// ============================================================================

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newMemHub 构造只依赖内存态的中枢（db/noticeHub 为 nil）。
func newMemHub(t *testing.T, enabled bool) *LiveViewHub {
	t.Helper()
	cfg := &config.Config{}
	cfg.RPA.LiveView.Enabled = enabled
	cfg.RPA.LiveView.MaxStreams = 2
	cfg.RPA.LiveView.IdleTimeout = time.Minute
	h := NewLiveViewHub(nil, cfg, nil, cache.NewMemoryCache(64, time.Minute), nil)
	t.Cleanup(h.Close)
	return h
}

// TestLiveViewHub_DisabledRejectsAll 未启用时所有入口都必须拒绝（不静默降级）。
func TestLiveViewHub_DisabledRejectsAll(t *testing.T) {
	h := newMemHub(t, false)
	ctx := context.Background()

	_, err := h.Watch(ctx, "e1", "u1", "甲")
	require.Error(t, err)
	require.Contains(t, err.Error(), "未启用")

	require.Error(t, h.Takeover(ctx, "e1", "u1", "甲", false))
	require.Error(t, h.Release(ctx, "e1", "u1", "甲", "r"))
	require.Error(t, h.SetMode(ctx, "e1", "甲", LiveViewModeAuto))
	require.Error(t, h.Abort(ctx, "e1", "甲", "r"))
}

// TestLiveViewHub_WatchStateStats Watch → State/Stats 字段一致。
func TestLiveViewHub_WatchStateStats(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()

	state, err := h.Watch(ctx, "e1", "u1", "甲")
	require.NoError(t, err, "开流必须成功（publish 失败不阻断）")
	require.Equal(t, 1, state.Streaming)
	require.Equal(t, LiveViewModeCopilot, state.Mode, "默认协作模式是 copilot")
	require.Equal(t, 2, state.MaxStreams)

	st := h.State("e1")
	require.Equal(t, 1, st.Watchers)
	require.NotNil(t, st.Context)

	stats := h.Stats("e1")
	require.Equal(t, 1, stats.Watchers)
	require.EqualValues(t, 0, stats.Frames)
	require.False(t, stats.EmptyPageHint, "刚开流不应立刻判定为空页面")
}

// TestLiveViewHub_WatchIsIdempotent 重复 watch 不重复计数（同一用户）。
func TestLiveViewHub_WatchIsIdempotent(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	state, err := h.Watch(ctx, "e1", "u1", "甲")
	require.NoError(t, err)
	require.Equal(t, 1, state.Streaming, "同一用户重复 watch 不得重复计数")
}

// TestLiveViewHub_TakeoverConflictAndForce 单接管者 + force 抢占。
func TestLiveViewHub_TakeoverConflictAndForce(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	_, _ = h.Watch(ctx, "e1", "u2", "乙")

	require.NoError(t, h.Takeover(ctx, "e1", "u1", "甲", false), "首次接管成功")

	err := h.Takeover(ctx, "e1", "u2", "乙", false)
	require.Error(t, err, "他人已接管且未 force 必须冲突")
	var conflict *LiveViewConflictError
	require.True(t, errors.As(err, &conflict), "必须是可识别的冲突类型（handler 据此转 409）")
	require.Equal(t, "甲", conflict.By)

	require.NoError(t, h.Takeover(ctx, "e1", "u2", "乙", true), "force 可抢占")
	require.Equal(t, "u2", h.State("e1").Context.TakeoverByID, "抢占后权威判据必须换人")
}

// TestLiveViewHub_TakeoverWithoutSession 无观看会话时接管必须给出可读原因。
func TestLiveViewHub_TakeoverWithoutSession(t *testing.T) {
	h := newMemHub(t, true)
	err := h.Takeover(context.Background(), "no-session", "u1", "甲", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "观看会话", "错误文案必须提示先打开实时画面")
}

// TestLiveViewHub_ReleaseOnlyByController 只有接管者能释放。
func TestLiveViewHub_ReleaseOnlyByController(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	_, _ = h.Watch(ctx, "e1", "u2", "乙")
	require.NoError(t, h.Takeover(ctx, "e1", "u1", "甲", false))

	err := h.Release(ctx, "e1", "u2", "乙", "抢释放")
	require.Error(t, err, "非接管者释放必须被拒")

	require.NoError(t, h.Release(ctx, "e1", "u1", "甲", "交接完成"))
	require.Empty(t, h.State("e1").Context.TakeoverByID, "释放后接管者必须清空")
}

// TestLiveViewHub_InputRequiresController 非接管者注入被拒（Forbidden 类型可识别）。
func TestLiveViewHub_InputRequiresController(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	_, _ = h.Watch(ctx, "e1", "u2", "乙")

	cmd := &LiveViewInputCommand{Kind: "click", X: 10, Y: 10}
	err := h.Input(ctx, "e1", "u1", "甲", cmd)
	var forbidden *LiveViewForbiddenError
	require.True(t, errors.As(err, &forbidden), "未接管者注入必须是 Forbidden（handler 据此转 403）")

	require.NoError(t, h.Takeover(ctx, "e1", "u1", "甲", false))
	// 接管者放行后，publish 仍可能失败（memory cache 不提供 Redis 客户端），
	// 这是设计内的尽力而为语义 —— 断言「不再是 Forbidden」而不是「完全无错」。
	injectErr := h.Input(ctx, "e1", "u1", "甲", cmd)
	var forbidden2 *LiveViewForbiddenError
	require.False(t, errors.As(injectErr, &forbidden2), "接管者不得被判为无权限（%v）", injectErr)
}

// TestLiveViewHub_SetMode 三种模式切换 + 非法模式拒绝。
func TestLiveViewHub_SetMode(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")

	for _, m := range []LiveViewMode{LiveViewModeAuto, LiveViewModeCopilot, LiveViewModeManual} {
		require.NoError(t, h.SetMode(ctx, "e1", "甲", m), "模式 %s 必须可切换", m)
		require.Equal(t, m, h.State("e1").Context.Mode)
	}

	require.Error(t, h.SetMode(ctx, "e1", "甲", LiveViewMode("turbo")), "非法模式必须拒绝")
}

// TestLiveViewHub_IngestFrameCountsAndContext 帧计数 + context 权威值覆盖。
func TestLiveViewHub_IngestFrameCountsAndContext(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")

	up := &LiveViewUpstreamFrame{
		Frame: LiveViewFrame{
			ExecutionID: "e1",
			Seq:         1,
			TimestampMS: time.Now().UnixMilli(),
			Data:        "AAAA",
			ByteSize:    4,
		},
		Context: &LiveViewContextSnapshot{
			ExecutionID:  "e1",
			Mode:         LiveViewModeAuto, // worker 自己的视图（应与后端不一致）
			TakeoverBy:   "worker-side-view",
			TakeoverByID: "not-authoritative",
		},
	}
	require.Equal(t, 1, h.IngestFrame(ctx, up), "有 watcher 时必须推送")

	stats := h.Stats("e1")
	require.EqualValues(t, 1, stats.Frames)
	require.EqualValues(t, up.Frame.TimestampMS, stats.LastFrameMS, "LastFrameMS 必须回带帧自身的时间戳")

	// W7 / UI-FIX-02：帧里的 context 必须是后端权威值，不能透传 worker 视图。
	sc := h.State("e1").Context
	require.NotEqual(t, "not-authoritative", sc.TakeoverByID, "不得透传 worker 的 takeoverById")
	require.Equal(t, LiveViewModeCopilot, sc.Mode, "mode 必须以后端会话表为准")
}

// TestLiveViewHub_IngestFrameNoSession 无会话时帧直接丢弃（返回 0，Worker 据此停流）。
func TestLiveViewHub_IngestFrameNoSession(t *testing.T) {
	h := newMemHub(t, true)
	up := &LiveViewUpstreamFrame{Frame: LiveViewFrame{ExecutionID: "no-such"}}
	require.Equal(t, 0, h.IngestFrame(context.Background(), up))
}

// TestLiveViewHub_EmptyPageHint 开流 5s 零帧 → EmptyPageHint（W7.1 缺陷 C 判据）。
func TestLiveViewHub_EmptyPageHint(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")

	// 刚开流：不判空页
	require.False(t, h.Stats("e1").EmptyPageHint)

	// 把推流起点拨回 6 秒前（直接改内部态，模拟「已等超阈值」）
	h.mu.Lock()
	h.sessions["e1"].streamStartedAt = time.Now().Add(-6 * time.Second)
	h.mu.Unlock()

	stats := h.Stats("e1")
	require.True(t, stats.EmptyPageHint, "超过 5s 零帧必须给出空页提示")
	require.Greater(t, stats.EmptyPageWaitingMS, int64(5000))

	// 收到一帧后提示必须消失（不能一直挂着）
	up := &LiveViewUpstreamFrame{Frame: LiveViewFrame{ExecutionID: "e1", TimestampMS: time.Now().UnixMilli()}}
	_ = h.IngestFrame(ctx, up)
	require.False(t, h.Stats("e1").EmptyPageHint, "收到帧后空页提示必须清除")
}

// TestLiveViewHub_StopRemovesWatcher Stop 后 watcher 归零；不存在会话时幂等。
func TestLiveViewHub_StopRemovesWatcher(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	_, _ = h.Watch(ctx, "e1", "u2", "乙")
	require.Equal(t, 2, h.State("e1").Watchers)

	h.Stop(ctx, "e1", "u1", "关掉")
	require.Equal(t, 1, h.State("e1").Watchers, "停流只移除自己的 watcher")

	h.Stop(ctx, "e1", "u2", "关掉")
	require.Equal(t, 0, h.State("e1").Watchers)

	// 重复停流不得 panic
	h.Stop(ctx, "no-such", "u9", "关掉")
}

// TestLiveViewHub_SessionAndWatcherCounts 计数口径（FLEET 观测用）。
func TestLiveViewHub_SessionAndWatcherCounts(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	require.Equal(t, 0, h.SessionCount())
	require.Equal(t, 0, h.WatcherCount())

	_, _ = h.Watch(ctx, "e1", "u1", "甲")
	_, _ = h.Watch(ctx, "e2", "u2", "乙")
	require.Equal(t, 2, h.SessionCount())
	require.Equal(t, 2, h.WatcherCount())
}

// TestLiveViewHub_KeepaliveUnknownSafe 不存在会话的心跳不得 panic。
func TestLiveViewHub_KeepaliveUnknownSafe(t *testing.T) {
	h := newMemHub(t, true)
	h.Keepalive(context.Background(), "no-such", "u1")
}

// TestLiveViewHub_DropWatcher 主动移除 watcher（断线清理路径）。
func TestLiveViewHub_DropWatcher(t *testing.T) {
	h := newMemHub(t, true)
	ctx := context.Background()
	_, _ = h.Watch(ctx, "e1", "u1", "甲")

	require.True(t, h.DropWatcher("e1", "u1", "断线"), "已存在的 watcher 必须移除成功")
	require.False(t, h.DropWatcher("e1", "u1", "再断一次"), "重复移除必须返回 false")
}

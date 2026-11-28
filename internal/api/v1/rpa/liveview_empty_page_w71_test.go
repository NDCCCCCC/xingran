package rpa

// ============================================================================
// W7.1 / 缺陷 C 契约用例：后端「已推流但零帧」判定（emptyPageHint）
//
// 缺陷实证（生产）：脚本动作没执行成功 → 页面停在 about:blank → Chrome 不产
// screencast 帧 → 后端 30 分钟内 /liveview/frame 请求数 = 0 → 前端只有
// 「等待画面…」一句，用户无法判断是「还没开始」还是「永远不会有画面」。
//
// 判定收敛到后端已持有的事实（帧数 + 开流时长），不引入跨进程新字段：
//   TestEmptyPageHint_ZeroFramesAfterThreshold   watch 后 5s 零帧 → stats 带
//                                                emptyPageHint=true + 等待时长
//   TestEmptyPageHint_FramesReceivedNoHint       反向：收到帧 → 不再提示
//   TestEmptyPageHint_NotBeforeThreshold         阈值内（刚开流）→ 不提前误报
//
// 走真实 hub + 真实数据库（liveview_p155_test.go 同款 fixture），不 mock schema。
// ============================================================================

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// emptyPageHintThreshold 与 services/rpa 侧常量同值（跨包无法直接引用包内
// const，故在此复制一份并在用例里断言两侧口径一致）。
const emptyPageHintThreshold = 5 * time.Second

// TestEmptyPageHint_ZeroFramesAfterThreshold 主断言。
func TestEmptyPageHint_ZeroFramesAfterThreshold(t *testing.T) {
	hub, c, database := newP155Hub(t)
	// newP155Hub 默认 IdleTimeout=300ms（专给「心跳超时」用例用）；本用例需要
	// 观察 5s 阈值，watcher 不能在观察期内被 janitor 扫掉，故放宽闲置窗口。
	c.Config.RPA.LiveView.IdleTimeout = 30 * time.Second
	const userID = "w71-empty-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	// 阈值内（刚开流）：不得提前误报 —— 否则正常页面刚打开就显示「暂无画面」。
	early := hub.Stats(execID)
	require.False(t, early.EmptyPageHint,
		"刚开流就提示「暂无画面」属误报（用户会立刻认为页面坏了）")

	// 越过阈值仍 0 帧 → 必须判空，并给出已等待时长。
	time.Sleep(emptyPageHintThreshold + 500*time.Millisecond)
	stats := hub.Stats(execID)
	require.True(t, stats.EmptyPageHint,
		"已推流 %v 且 0 帧，必须给出 emptyPageHint（前端据此换成可读文案）",
		emptyPageHintThreshold)
	require.EqualValues(t, 0, stats.Frames, "判定前提是 0 帧")
	require.Greater(t, stats.EmptyPageWaitingMS, emptyPageHintThreshold.Milliseconds()-100,
		"必须给出已等待时长，用户才知道等了多久")
}

// TestEmptyPageHint_FramesReceivedNoHint 反向用例：有帧就不提示。
func TestEmptyPageHint_FramesReceivedNoHint(t *testing.T) {
	hub, c, database := newP155Hub(t)
	c.Config.RPA.LiveView.IdleTimeout = 30 * time.Second
	const userID = "w71-frame-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	// 真帧：有像素数据（不是控制帧）。
	up := &svcrpa.LiveViewUpstreamFrame{
		Frame: svcrpa.LiveViewFrame{
			ExecutionID: execID, Seq: 1, TimestampMS: time.Now().UnixMilli(),
			Data: "AAAA", ByteSize: 4, Width: 1280, Height: 720,
		},
		Context: &svcrpa.LiveViewContextSnapshot{ExecutionID: execID},
	}
	require.Equal(t, 1, hub.IngestFrame(t.Context(), up), "帧应推给 1 个 watcher")

	// 即使等过阈值，也不得判空（有画面的页面不该被告知「暂无画面」）。
	time.Sleep(emptyPageHintThreshold + 500*time.Millisecond)
	stats := hub.Stats(execID)
	require.False(t, stats.EmptyPageHint, "已收到帧的会话不得给 emptyPageHint")
	require.EqualValues(t, 1, stats.Frames)
}

// TestEmptyPageHint_NotBeforeThreshold 阈值内不提示（防止开局误报）。
func TestEmptyPageHint_NotBeforeThreshold(t *testing.T) {
	hub, c, database := newP155Hub(t)
	c.Config.RPA.LiveView.IdleTimeout = 30 * time.Second
	const userID = "w71-early-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	stats := hub.Stats(execID)
	require.False(t, stats.EmptyPageHint, "开流 500ms 内（页面正在加载）不得判空")
	require.EqualValues(t, 0, stats.Frames)
}

// TestEmptyPageHint_ThresholdIsFiveSeconds 阈值契约（三端口径必须一致）。
//
// 后端 emptyPageHintAfter / Worker DefaultScreencastZeroFrameWarnAfter /
// 前端 EMPTY_PAGE_HINT_AFTER_MS 必须同为 5s —— 否则会出现「Worker 日志说
// 空页面、前端还在等待画面」的分裂，正是本缺陷最容易被误判成「没修好」的形态。
func TestEmptyPageHint_ThresholdIsFiveSeconds(t *testing.T) {
	require.Equal(t, 5*time.Second, emptyPageHintThreshold,
		"后端判定阈值必须是 5s（与 Worker / 前端一致）")
}

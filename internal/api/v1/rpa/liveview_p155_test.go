package rpa

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：服务层契约用例（真实 sqlite，无 mock）
//
// 遵循 Phase 152/153 确立的样板（contract_p0_smoke_test.go / runtime_p153_test.go）：
// httptest 与真实库在本文件也用 contractP0Core（真实的 db.NewDatabase →
// AutoMigrate → rpa.NewServiceGroup），**不自建表、不 mock service**。
//
// 覆盖：
//   TestLiveView05_Audit_PersistsToRealTable   四类审计事件真写 sys_rpa_audit_logs
//   TestLiveView03_Continue_PausedToRunning    paused → running 走 RUNTIME-05 状态机
//   TestLiveView03_TerminalNotRevived          终态不可被 continue 复活
//   TestLiveView03_CanTransitionShared         与 RUNTIME-05 共用同一套校验
//   TestLiveView05_Watch_NoRedisReadable       无 Redis 时 watch 仍可用（publish 失败不阻断）
//   TestLiveView05_TakeoverConflict            单执行单接管者 + force 抢占
//   TestLiveView05_InputNonTakeoverer          非接管者注入被拒
//   TestLiveView03_InvalidModeRejected         非法模式被拒（不静默降级）
//   TestLiveView01_IngestFrame_NoWatcherZero   无人观看时 pushed=0（Worker 侧据此 idle 停流）
//   TestLiveView05_KeepaliveTimeoutDrops       心跳超时移除 watcher（断开 N 秒停流）
// ============================================================================

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// newP155Hub 用真实 core 装配 LiveViewHub（与生产相同的 ServiceGroup 路径）。
func newP155Hub(t *testing.T) (*rpa.LiveViewHub, *core.Core, *db.Database) {
	t.Helper()
	c, database := contractP0Core(t)
	// 启用实时画面（默认配置下 Enabled 未显式设置时核心构造会保留 false，
	// 因此这里显式打开）。
	c.Config.RPA.LiveView.Enabled = true
	c.Config.RPA.LiveView.MaxStreams = 2
	c.Config.RPA.LiveView.IdleTimeout = 300 * time.Millisecond

	services := rpa.NewServiceGroup(c.GetDB(), c.Config, c.NoticeHub, c.Cache,
		c.SM4Cipher, system.NewCacheProvider(c.DataCacheService))
	t.Cleanup(func() {
		if services.LiveViewService != nil {
			services.LiveViewService.Close()
		}
	})
	return services.LiveViewService, c, database
}

// TestLiveView05_Audit_PersistsToRealTable 审计事件必须真的写进 sys_rpa_audit_logs。
func TestLiveView05_Audit_PersistsToRealTable(t *testing.T) {
	hub, c, database := newP155Hub(t)
	require.NotNil(t, hub, "LiveViewHub 应已构造")
	const userID = "p155-audit-user"
	seedContractP0Admin(t, c, userID)

	execID := seedRuntimeExecution(t, database, "running")

	events := []string{"watch_start", "takeover", "release", "watch_stop"}
	for i, e := range events {
		require.NoError(t,
			hub.Audit(t.Context(), execID, userID, "审计用户", rpa.LiveViewEventKind(e),
				map[string]interface{}{"index": i, "reason": "契约用例"}),
			"写审计 %s 失败", e)
	}

	var count int64
	require.NoError(t, database.DB.Model(&rpamodels.AuditLog{}).
		Where("resource_id = ?", execID).Count(&count).Error)
	require.EqualValues(t, len(events), count, "审计行数应等于事件数")

	// 未知事件类型必须被拒（审计表不被污染）。
	err := hub.Audit(t.Context(), execID, userID, "x", rpa.LiveViewEventKind("bogus"), nil)
	require.Error(t, err, "未知事件类型应被拒")
	require.Contains(t, err.Error(), "未知事件类型")
}

// TestLiveView03_Continue_PausedToRunning paused → running 走 RUNTIME-05 状态机。
func TestLiveView03_Continue_PausedToRunning(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-continue-user"
	seedContractP0Admin(t, c, userID)

	execID := seedRuntimeExecution(t, database, "paused")
	require.NoError(t, hub.Continue(t.Context(), execID, userID, "人工确认继续"))

	var status string
	require.NoError(t, database.DB.Raw(
		`SELECT status FROM sys_rpa_executions WHERE id = ?`, execID).Row().Scan(&status))
	require.Equal(t, "running", status, "paused 应迁移到 running（走 CanTransition）")
}

// TestLiveView03_TerminalNotRevived 终态不可被 continue 复活。
func TestLiveView03_TerminalNotRevived(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-terminal-user"
	seedContractP0Admin(t, c, userID)

	termID := seedRuntimeExecution(t, database, "success")
	// Continue 对非 paused 状态不做迁移（也不应报错：publish 是尽力而为）。
	_ = hub.Continue(t.Context(), termID, userID, "非法续跑")

	var status string
	require.NoError(t, database.DB.Raw(
		`SELECT status FROM sys_rpa_executions WHERE id = ?`, termID).Row().Scan(&status))
	require.Equal(t, "success", status, "终态不可被覆盖（RUNTIME-05 硬约束）")
}

// TestLiveView03_CanTransitionShared 与 RUNTIME-05 共用同一套校验（静态证伪）。
func TestLiveView03_CanTransitionShared(t *testing.T) {
	require.False(t, rpa.CanTransition(rpamodels.RPAExecutionStatusSuccess, rpamodels.RPAExecutionStatusRunning),
		"success → running 应为非法迁移")
	require.False(t, rpa.CanTransition(rpamodels.RPAExecutionStatusCancelled, rpamodels.RPAExecutionStatusRunning),
		"cancelled → running 应为非法迁移")
	require.True(t, rpa.CanTransition(rpamodels.RPAExecutionStatusPaused, rpamodels.RPAExecutionStatusRunning),
		"paused → running 应为合法迁移")
	require.True(t, rpa.CanTransition(rpamodels.RPAExecutionStatusRunning, rpamodels.RPAExecutionStatusPaused),
		"running → paused 应为合法迁移（copilot/manual 暂停点）")
}

// TestLiveView05_Watch_NoRedisReadable 无 Redis 时 watch 不 500（publish 失败不阻断观看）。
func TestLiveView05_Watch_NoRedisReadable(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-watch-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	state, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err, "publish 失败不应阻断观看（HTTP 帧上行与 Redis 下行解耦）")
	require.NotNil(t, state)
	require.Equal(t, rpa.LiveViewModeCopilot, state.Mode, "默认模式必须是 copilot")
	require.Equal(t, 2, state.MaxStreams)
	require.GreaterOrEqual(t, hub.WatcherCount(), 1)

	// 重复 watch 幂等（不重复占名额）。
	_, err = hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.Equal(t, 1, hub.WatcherCount(), "重复 watch 不应重复登记")
}

// TestLiveView05_TakeoverConflict 单执行单接管者；抢占需显式 force。
func TestLiveView05_TakeoverConflict(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-takeover-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "alice", false))

	// bob 无 force → 冲突错误（handler 转 409）。
	err = hub.Takeover(t.Context(), execID, "bob-id", "bob", false)
	require.Error(t, err)
	var conflict *rpa.LiveViewConflictError
	require.True(t, errors.As(err, &conflict), "错误类型应为 *LiveViewConflictError")
	require.Contains(t, conflict.Error(), "已被")

	// bob 带 force → 成功。
	require.NoError(t, hub.Takeover(t.Context(), execID, "bob-id", "bob", true))
}

// TestLiveView05_InputNonTakeoverer 非接管者注入被拒（handler 转 403）。
func TestLiveView05_InputNonTakeoverer(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-input-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	// 无接管者 → 拒绝。
	err = hub.Input(t.Context(), execID, userID, "alice", &rpa.LiveViewInputCommand{Kind: "mouse_move", X: 1, Y: 1})
	require.Error(t, err)
	var forbidden *rpa.LiveViewForbiddenError
	require.True(t, errors.As(err, &forbidden), "无接管者时应返回 *LiveViewForbiddenError")

	// 有接管者但不是本人 → 拒绝。
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "alice", false))
	err = hub.Input(t.Context(), execID, "bob-id", "bob", &rpa.LiveViewInputCommand{Kind: "mouse_move", X: 1, Y: 1})
	require.Error(t, err)
	require.True(t, errors.As(err, &forbidden))

	// 接管者本人（同 userID）→ 通过鉴权（W7 / UI-FIX-02：以 userID 为权威判据）。
	// 注意：这里只断言**不是 403**。publish 需要有 cache 才能下发，本用例的
	// core 未注入 Redis，故最终错误是「cache 未注入」——那属于下发失败而非鉴权失败。
	ownErr := hub.Input(t.Context(), execID, userID, "alice",
		&rpa.LiveViewInputCommand{Kind: "mouse_move", X: 1, Y: 1})
	require.False(t, errors.As(ownErr, &forbidden), "接管者本人不得被判非接管者")
}

// TestLiveView03_InvalidModeRejected 非法模式被拒（不静默降级）。
func TestLiveView03_InvalidModeRejected(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-mode-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)

	err = hub.SetMode(t.Context(), execID, userID, rpa.LiveViewMode("wizard"))
	require.Error(t, err, "非法模式必须被拒")
	require.Contains(t, err.Error(), "非法协作模式")

	// 三个合法模式都通过。
	for _, m := range []rpa.LiveViewMode{rpa.LiveViewModeAuto, rpa.LiveViewModeCopilot, rpa.LiveViewModeManual} {
		require.NoError(t, hub.SetMode(t.Context(), execID, userID, m), "合法模式 %s 应通过", m)
	}
}

// TestLiveView01_IngestFrame_NoWatcherZero 无人观看时 pushed=0（Worker 侧据此停流）。
func TestLiveView01_IngestFrame_NoWatcherZero(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-frame-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	up := &rpa.LiveViewUpstreamFrame{
		Frame: rpa.LiveViewFrame{
			ExecutionID: execID, Seq: 1, TimestampMS: time.Now().UnixMilli(),
			Data: "AAAA", ByteSize: 4, Width: 1280, Height: 720,
			DeviceWidth: 1920, DeviceHeight: 1080,
		},
		Context: &rpa.LiveViewContextSnapshot{ExecutionID: execID, Step: 1, TotalSteps: 3, Mode: rpa.LiveViewModeCopilot},
	}
	// 未 watch → 无人接收 → 0。
	require.Equal(t, 0, hub.IngestFrame(t.Context(), up), "无人观看时应返回 pushed=0")

	// watch 后同一帧 → 至少 1 个接收者（WS hub 无该用户客户端时仍返回列表长度）。
	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	pushed := hub.IngestFrame(t.Context(), up)
	require.GreaterOrEqual(t, pushed, 1, "有观看者后应返回 >=1")

	stats := hub.Stats(execID)
	require.GreaterOrEqual(t, stats.Frames, int64(1))
}

// TestLiveView05_KeepaliveTimeoutDrops 心跳超时 → watcher 被移除（断流判定）。
func TestLiveView05_KeepaliveTimeoutDrops(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "p155-idle-user"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.Equal(t, 1, hub.WatcherCount())

	// IdleTimeout = 300ms：等 600ms 后推一帧，watcher 应因超时被移除。
	time.Sleep(600 * time.Millisecond)
	pushed := hub.IngestFrame(t.Context(), &rpa.LiveViewUpstreamFrame{
		Frame: rpa.LiveViewFrame{ExecutionID: execID, Seq: 2, TimestampMS: time.Now().UnixMilli(), Data: "BBBB"},
	})
	require.Equal(t, 0, pushed, "心跳超时后应无接收者（LIVEVIEW-05 断开 N 秒停流）")
	require.Equal(t, 0, hub.WatcherCount())
}

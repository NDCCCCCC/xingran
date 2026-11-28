package rpa

// ============================================================================
// W7 / UI-FIX-02 契约用例：接管权以 userID 为权威
//
// 背景（生产缺陷实证，2026-11-26 真实浏览器 10.62.10.39）：
//   点「接管」后页面角标显示「已接管 · ninedrunk」，但遮罩仍在、输入发不出去。
//   根因是后端 takeoverBy 存的是 operatorName(c)（SM2 token 无 nickname →
//   恒为 username），而前端用 user.nickname 比对组控制权判定字符串。
//   ninedrunk 恰好 nickname==username 所以「碰巧能用」；把 store nickname 改成
//   「超级管理员」刷新后再接管 → 遮罩不消失（mask=1），缺陷稳定复现。
//
// 修复：后端同时记录 takeoverByID（用户 ID），Takeover/Release/Input 的判据
// 全部以 userID 为准；takeoverBy 只作展示。本文件锁定这套语义。
//
// 覆盖：
//   TestUI02_TakeoverByUserID_NotDisplayName    展示名漂移不影响控制权归属
//   TestUI02_StateExposesTakeoverByID           state/watch 必须把 userID 下发前端
//   TestUI02_ReleaseUsesUserID                  释放以 userID 判定，展示名只用于文案
//   TestUI02_InputUsesUserID                    输入鉴权同口径
//   TestUI02_ForceOverridesOtherUser            抢占语义不变（force 仍可夺权）
// ============================================================================

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

// TestUI02_TakeoverByUserID_NotDisplayName 同一 userID 换展示名仍是同一接管者。
//
// 这正是缺陷复现路径：先以 nickname=ninedrunk 接管，再以同一用户的
// nickname=超级管理员 操作 —— 修复前第二次会被判「非接管者」（因为字符串不等），
// 修复后必须认。
func TestUI02_TakeoverByUserID_NotDisplayName(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "w7-takeover-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "ninedrunk")
	require.NoError(t, err)

	// 第一次接管：展示名 = username（SM2 token 没有 nickname 时的真实形态）
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "ninedrunk", false))

	// 同一用户、展示名变了（用户改了昵称）—— 再抢一次不该冲突。
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "超级管理员", false),
		"同一 userID 换展示名不得被判冲突")

	// 状态里两份都要有：展示名更新了，userID 不变。
	state := hub.State(execID)
	require.NotNil(t, state.Context)
	require.Equal(t, "超级管理员", state.Context.TakeoverBy, "展示名应更新")
	require.Equal(t, userID, state.Context.TakeoverByID, "userID 必须保持")

	// 别人（不同 userID、但展示名与当前接管者相同）→ 必须冲突（不能被展示名骗过）。
	var conflict *rpa.LiveViewConflictError
	err = hub.Takeover(t.Context(), execID, "other-uid", "超级管理员", false)
	require.Error(t, err, "不同 userID 即使展示名相同也必须冲突")
	require.True(t, errors.As(err, &conflict))
}

// TestUI02_StateExposesTakeoverByID state 端点必须把 takeoverById 下发给前端。
//
// 前端 isController 只认这个字段；少了它就会回落到展示名比对（缺陷行为）。
func TestUI02_StateExposesTakeoverByID(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "w7-state-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "观众", false))

	state := hub.State(execID)
	require.NotNil(t, state.Context)
	require.Equal(t, userID, state.Context.TakeoverByID,
		"state 必须下发 takeoverById（前端据此判 isController，缺它会退回字符串比对）")

	// watch 端点同样要带上（进入页面时先拿 watch 的返回值）。
	watch, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.Equal(t, userID, watch.TakeoverByID, "watch 响应也必须带 takeoverById")

	// 帧回来后 context 也要带（其它观看者靠帧里的 context 更新 HUD）。
	up := &rpa.LiveViewUpstreamFrame{
		Frame: rpa.LiveViewFrame{ExecutionID: execID, TimestampMS: 1},
		Context: &rpa.LiveViewContextSnapshot{
			ExecutionID: execID,
			// Worker 侧没有 userID，只有展示名 —— 后端必须用权威值覆盖。
			TakeoverBy: "worker 视角的旧名字",
		},
	}
	require.Equal(t, 1, hub.IngestFrame(t.Context(), up),
		"帧上报应推给 1 个 watcher（同时校验 context 覆盖口径）")

	// 覆盖后仍是后端权威值（不是 Worker 回传的旧名字）。
	after := hub.State(execID)
	require.NotNil(t, after.Context)
	require.Equal(t, "观众", after.Context.TakeoverBy,
		"帧回传的 takeoverBy 必须被后端权威值覆盖")
	require.Equal(t, userID, after.Context.TakeoverByID,
		"帧回传后 takeoverById 仍必须是后端权威值")
}
// TestUI02_ReleaseUsesUserID 释放以 userID 判定。
func TestUI02_ReleaseUsesUserID(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "w7-release-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "ninedrunk", false))

	// 同一 userID、展示名不同 → 释放成功（修复前会失败）。
	require.NoError(t, hub.Release(t.Context(), execID, userID, "超级管理员", "改昵称后释放"),
		"同一 userID 换展示名释放必须成功")

	state := hub.State(execID)
	require.NotNil(t, state.Context)
	require.Empty(t, state.Context.TakeoverBy)
	require.Empty(t, state.Context.TakeoverByID)

	// 无接管者时重复释放幂等。
	require.NoError(t, hub.Release(t.Context(), execID, userID, "超级管理员", "重复释放"))
}

// TestUI02_InputUsesUserID 输入鉴权与接管同口径。
func TestUI02_InputUsesUserID(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const userID = "w7-input-uid"
	seedContractP0Admin(t, c, userID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, userID, "观众")
	require.NoError(t, err)
	require.NoError(t, hub.Takeover(t.Context(), execID, userID, "ninedrunk", false))

	cmd := &rpa.LiveViewInputCommand{Kind: "mouse_move", X: 1, Y: 1}

	// 同一 userID、展示名变了 → 不得 403（只允许因未注入 cache 的下发错误）。
	var forbidden *rpa.LiveViewForbiddenError
	ownErr := hub.Input(t.Context(), execID, userID, "超级管理员", cmd)
	require.False(t, errors.As(ownErr, &forbidden),
		"同一 userID 换展示名输入不得被判非接管者")

	// 不同 userID、展示名相同 → 必须 403。
	otherErr := hub.Input(t.Context(), execID, "other-uid", "ninedrunk", cmd)
	require.Error(t, otherErr)
	require.True(t, errors.As(otherErr, &forbidden),
		"不同 userID 即使展示名相同也必须被拒")
}

// TestUI02_ForceOverridesOtherUser force 抢占语义保持不变。
func TestUI02_ForceOverridesOtherUser(t *testing.T) {
	hub, c, database := newP155Hub(t)
	const ownerID = "w7-force-owner"
	seedContractP0Admin(t, c, ownerID)
	execID := seedRuntimeExecution(t, database, "running")

	_, err := hub.Watch(t.Context(), execID, ownerID, "观众")
	require.NoError(t, err)
	require.NoError(t, hub.Takeover(t.Context(), execID, ownerID, "alice", false))

	// 他人无 force → 409 语义（冲突错误）。
	var conflict *rpa.LiveViewConflictError
	err = hub.Takeover(t.Context(), execID, "w7-force-bob", "bob", false)
	require.Error(t, err)
	require.True(t, errors.As(err, &conflict))

	// 他人带 force → 夺权成功，且 userID 换成 bob。
	require.NoError(t, hub.Takeover(t.Context(), execID, "w7-force-bob", "bob", true))
	state := hub.State(execID)
	require.NotNil(t, state.Context)
	require.Equal(t, "w7-force-bob", state.Context.TakeoverByID)
	require.Equal(t, "bob", state.Context.TakeoverBy)
}

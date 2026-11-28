package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：录制会话中枢单测（此前 recording_service.go 4.6%）
//
// W6 的录制功能只在 api 层有 HTTP 用例（且需真 Redis 才不 skip），服务层这 497
// 语句里「会话状态机 / 上限裁决 / 上报落态 / 拾取结果 / 闲置清理 / 产出落库」
// 从未被直接触达。这里补服务层直测，全部零外部依赖（db 可用真 sqlite）。
// ============================================================================

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newRecHub 构造录制中枢（真 sqlite，无 Redis/无需 noticeHub）。
func newRecHub(t *testing.T, maxSessions int, idle time.Duration) (*RecordingHub, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	// 录制会话落 Redis（persist），不落库；库里只需任务表（产出回填用）。
	require.NoError(t, db.AutoMigrate(&rpamodels.Task{}), "真实 AutoMigrate 必须能建出任务表")

	// taskSvc 注入真实 TaskService：ReportResult 的产出回写走它（REC-01 要求）。
	h := NewRecordingHub(RecordingHubOption{
		Cache:       cache.NewMemoryCache(64, time.Minute),
		DB:          db,
		TaskService: NewTaskService(db, cache.NewMemoryCache(16, time.Minute), nil),
	})
	t.Cleanup(h.Close)
	return h, db
}

// seedRecordingSession 直接注入一个「录集中」会话。
//
// 为什么不用 StartSession：它必须把 start 指令送达 Worker（无 Redis 即落 failed
// 终态），而终态会话会让 ReportStatus/Abort/sweep 全部走短路分支 —— 测不到
// 状态机本身。同包测试直接写内存表，等价于「Worker 已收到指令、正在录制」。
func seedRecordingSession(h *RecordingHub, id string) *RecordingSession {
	now := time.Now()
	s := &RecordingSession{
		ID:           id,
		TaskID:       "t1",
		WorkerID:     "w1",
		OperatorID:   "u1",
		OperatorName: "甲",
		TargetURL:    "https://seed",
		State:        RecordingSessionRecording,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastSeen:     now,
	}
	h.mu.Lock()
	h.sessions[id] = s
	h.mu.Unlock()
	return s
}

// TestRecordingHub_MaxSessionsAndIdle 配置默认值口径。
func TestRecordingHub_MaxSessionsAndIdle(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	require.Greater(t, h.MaxSessions(), 0, "上限必须有正数默认值")
	require.Greater(t, h.IdleTimeout(), time.Duration(0), "闲置超时必须有正数默认值")
}

// TestRecordingHub_StartSession_RequiresID 空会话 ID 必须被拒。
func TestRecordingHub_StartSession_RequiresID(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	_, err := h.StartSession(context.Background(), "", "t1", "w1", "u1", "甲", "https://x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "会话 ID")
}

// TestRecordingHub_StartSession_Duplicate 同一会话 ID 不得重复创建。
func TestRecordingHub_StartSession_Duplicate(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	_, err := h.StartSession(ctx, "s-dup", "t1", "w1", "u1", "甲", "https://x")
	require.Error(t, err, "无 Redis 时 start 会失败（指令必须送达 Worker）")

	// 即便首次失败，会话仍登记在内（失败态），重复创建必须被拒。
	_, err2 := h.StartSession(ctx, "s-dup", "t1", "w1", "u1", "甲", "https://x")
	require.Error(t, err2)
}

// TestRecordingHub_GetAndList 会话快照与列表。
func TestRecordingHub_GetAndList(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	_, _ = h.StartSession(ctx, "s-1", "t1", "w1", "u1", "甲", "https://a")

	sess, ok := h.GetSession("s-1")
	require.True(t, ok, "刚创建的会话必须能查到")
	require.Equal(t, "t1", sess.TaskID)
	require.Equal(t, "甲", sess.OperatorName)
	require.Equal(t, "https://a", sess.TargetURL)

	_, ok = h.GetSession("nope")
	require.False(t, ok)

	list := h.ListSessions()
	require.NotEmpty(t, list, "列表必须含刚创建的会话")
}

// TestRecordingHub_Keepalive_Unknown 未知会话心跳必须报错（handler 转可读 400）。
func TestRecordingHub_Keepalive_Unknown(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	err := h.Keepalive(context.Background(), "nope")
	require.Error(t, err, "未知会话心跳必须报错")
}

// TestRecordingHub_ReportStatus 状态上报落到会话快照。
func TestRecordingHub_ReportStatus(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	seedRecordingSession(h, "s-2")

	err := h.ReportStatus(ctx, &RecordingStatusUpdate{
		SessionID:  "s-2",
		Status:     string(RecordingSessionRecording),
		EventCount: 7,
		Message:    "已采集 7 个动作",
	})
	require.NoError(t, err, "状态上报必须成功")

	sess, ok := h.GetSession("s-2")
	require.True(t, ok)
	require.EqualValues(t, 7, sess.EventCount, "事件数必须落态")
	require.Equal(t, "已采集 7 个动作", sess.Message)
}

// TestRecordingHub_ReportStatus_Unknown 未知会话上报必须报错（不静默丢弃）。
func TestRecordingHub_ReportStatus_Unknown(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	err := h.ReportStatus(context.Background(), &RecordingStatusUpdate{SessionID: "nope"})
	require.Error(t, err)
}

// TestRecordingHub_ReportPick 拾取结果落到 LastPick。
func TestRecordingHub_ReportPick(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	seedRecordingSession(h, "s-3")

	err := h.ReportPick(ctx, &RecordingPickUpdate{
		SessionID:        "s-3",
		Found:            true,
		Tag:              "INPUT",
		Text:             "用户名",
		Recommended:      "#username",
		RecommendedTier:  1,
		RecommendedScore: 95,
	})
	require.NoError(t, err)

	sess, ok := h.GetSession("s-3")
	require.True(t, ok)
	require.NotNil(t, sess.LastPick, "拾取结果必须落态（前端浮层消费）")
	require.Equal(t, "#username", sess.LastPick.Recommended)
}

// TestRecordingHub_ReportResult 录制产出落到 Result + 事件数。
func TestRecordingHub_ReportResult(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	seedRecordingSession(h, "s-4")

	err := h.ReportResult(ctx, &RecordingResultUpdate{
		SessionID:  "s-4",
		EventCount: 12,
		Actions: []map[string]interface{}{
			{"type": "navigate", "value": "https://d"},
			{"type": "click", "selector": "#submit"},
		},
		Steps: []string{"打开页面", "点击提交"},
	})
	require.NoError(t, err)

	sess, ok := h.GetSession("s-4")
	require.True(t, ok)
	require.NotNil(t, sess.Result, "产出必须落态")
	require.EqualValues(t, 12, sess.EventCount)
}

// TestRecordingHub_AbortSession 中止后会话状态为 aborted，重复中止幂等。
func TestRecordingHub_AbortSession(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	ctx := context.Background()
	seedRecordingSession(h, "s-5")

	h.AbortSession(ctx, "s-5", "测试结束")
	sess, ok := h.GetSession("s-5")
	require.True(t, ok)
	require.Equal(t, RecordingSessionAborted, sess.State)
	require.Equal(t, "测试结束", sess.Message)

	// 重复中止：不得 panic
	h.AbortSession(ctx, "s-5", "再来一次")
	h.AbortSession(ctx, "nope", "不存在")
}

// TestRecordingHub_StopSession_Unknown 未知会话停止必须报错。
func TestRecordingHub_StopSession_Unknown(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	err := h.StopSession(context.Background(), "nope", "甲")
	require.Error(t, err)
}

// TestRecordingHub_PickElement_Unknown 未知会话拾取必须报错。
func TestRecordingHub_PickElement_Unknown(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	err := h.PickElement(context.Background(), "nope", 10, 10)
	require.Error(t, err)
}

// TestRecordingHub_Close_IsIdempotent 关闭必须可重复调用。
func TestRecordingHub_Close_IsIdempotent(t *testing.T) {
	h, _ := newRecHub(t, 0, 0)
	h.Close()
	h.Close()
}

// TestRecordingHub_SweepIdleSessions 闲置超时后 sweep 必须清掉会话。
func TestRecordingHub_SweepIdleSessions(t *testing.T) {
	h, _ := newRecHub(t, 0, 50*time.Millisecond)
	seedRecordingSession(h, "s-6")

	// 把 lastSeen 拨回很久之前
	h.mu.Lock()
	if s, ok := h.sessions["s-6"]; ok {
		s.LastSeen = time.Now().Add(-time.Hour)
	}
	h.mu.Unlock()

	h.sweep()
	_, ok := h.GetSession("s-6")
	require.False(t, ok, "闲置超时的会话必须被清理")
}

// TestRecordingHub_SaveActionsToTask 产出落库到任务脚本（真实 sqlite 断言）。
func TestRecordingHub_SaveActionsToTask(t *testing.T) {
	h, db := newRecHub(t, 0, 0)
	ctx := context.Background()

	task := &rpamodels.Task{TaskName: "录制回填任务", Script: json.RawMessage("[]")}
	require.NoError(t, db.Create(task).Error)

	actions := []map[string]interface{}{{"type": "navigate", "value": "https://g"}}
	require.NoError(t, h.saveActionsToTask(ctx, task.ID, actions), "产出必须能落库")

	var script string
	require.NoError(t, db.Table("sys_rpa_tasks").Where("id = ?", task.ID).Pluck("script", &script).Error)
	require.Contains(t, script, "navigate", "脚本必须被真实写回")
}

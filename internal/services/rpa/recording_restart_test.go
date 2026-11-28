package rpa

// ============================================================================
// Phase W6 / REC-01：录制会话的 Redis 回读恢复（PR #114 拆分后续修）
//
// 缺陷：persist() 只 Set 不回读 —— 后端进程重启后 sessions map 为空，
// 而 Worker 侧的录制会话还在跑。前端轮询 GetSession 得到「会话不存在」，
// 用户以为录制丢了；随后 sweep/Close 也永远清不掉那些 Redis 键（2h TTL）。
//
// 修法：get() 内存未命中时从 Redis 回读并装回内存表；真正丢弃会话的路径
// （sweep / Close）同步删 Redis 键，避免「清掉的会话被下一次 get 复活」。
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

// TestRecordingHub_SurvivesRestart 模拟进程重启：新 Hub 只有 Redis，无内存态。
func TestRecordingHub_SurvivesRestart(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute) // 扮演 Redis：跨 Hub 实例共享
	ctx := context.Background()

	// 第一个 Hub：写入一个录制中会话（persist 落 Redis）。
	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	seedRecordingSession(h1, "s-restart")
	s1 := mustSession(t, h1, "s-restart")
	h1.persist(ctx, s1)
	// 注意：不要 h1.Close() —— Close 会中止所有会话并把「aborted」写回 Redis，
	// 那样测到的是「会话已结束」而不是「重启后恢复录制中会话」。
	// 这里直接丢弃 h1 引用，等价于进程重启（内存态消失、Redis 保留）。

	// 第二个 Hub：内存表为空，只能靠 Redis 回读。
	h2 := NewRecordingHub(RecordingHubOption{Cache: shared})
	t.Cleanup(h2.Close)
	require.Empty(t, h2.ListSessions(), "新 Hub 启动时内存表应为空（模拟重启）")

	sess, ok := h2.GetSession("s-restart")
	require.True(t, ok, "重启后必须能从 Redis 回读会话（否则用户以为录制丢了）")
	require.Equal(t, "s-restart", sess.ID)
	require.Equal(t, RecordingSessionRecording, sess.State)

	// 回读后应装回内存表（后续操作不必每次走 Redis）。
	require.Len(t, h2.ListSessions(), 1)
}

// TestRecordingHub_GetMissReturnsFalse 键不存在时返回 false（不 panic）。
func TestRecordingHub_GetMissReturnsFalse(t *testing.T) {
	h := NewRecordingHub(RecordingHubOption{Cache: cache.NewMemoryCache(16, time.Minute)})
	t.Cleanup(h.Close)
	_, ok := h.GetSession("never-existed")
	require.False(t, ok)
}

// TestRecordingHub_SweepClearsRedis 回收会话必须同时清 Redis，否则会被复活。
func TestRecordingHub_SweepClearsRedis(t *testing.T) {
	ctx := context.Background()
	shared := cache.NewMemoryCache(64, time.Minute)
	h := NewRecordingHub(RecordingHubOption{
		Cache:  shared,
		Config: mustCfgRecording(t, 2, 50*time.Millisecond),
	})
	t.Cleanup(h.Close)

	s := seedRecordingSession(h, "s-sweep")
	h.persist(ctx, s)

	// 把 lastSeen 拨回很久之前，触发闲置回收
	h.mu.Lock()
	s.LastSeen = time.Now().Add(-time.Hour)
	h.mu.Unlock()
	h.sweep()

	_, ok := h.GetSession("s-sweep")
	require.False(t, ok, "回收后不得被 Redis 回读复活")
}

// TestRecordingHub_PersistIsIdempotent 同一会话重复 persist 不报错、可回读。
func TestRecordingHub_PersistIsIdempotent(t *testing.T) {
	ctx := context.Background()
	h := NewRecordingHub(RecordingHubOption{Cache: cache.NewMemoryCache(16, time.Minute)})
	t.Cleanup(h.Close)
	s := seedRecordingSession(h, "s-idem")
	h.persist(ctx, s)
	h.persist(ctx, s)
	loaded, err := h.loadPersisted(ctx, "s-idem")
	require.NoError(t, err)
	require.NotNil(t, loaded)
}

// mustSession 取内存里的会话（测试辅助）。
func mustSession(t *testing.T, h *RecordingHub, sessionID string) *RecordingSession {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[sessionID]
	require.True(t, ok, "会话 %s 应存在", sessionID)
	return s
}

// mustCfgRecording 构造带录制参数的配置（测试辅助）。
func mustCfgRecording(t *testing.T, maxSessions int, idle time.Duration) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.RPA.Recording.MaxSessions = maxSessions
	cfg.RPA.Recording.IdleTimeout = idle
	return cfg
}

// TestRecordingHub_RestartRestoresLastSeen 回读后 LastSeen 必须有值（不得为 1970）。
//
// 这是加回读后的**次生缺陷**：LastSeen 曾是私有字段，json.Marshal 不序列化，
// 回读后恒为零值 → sweep 立刻把刚恢复的会话判成闲置清掉。
func TestRecordingHub_RestartRestoresLastSeen(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute)
	ctx := context.Background()

	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	s1 := seedRecordingSession(h1, "s-ls")
	s1.LastSeen = time.Now().Add(-10 * time.Second) // 仍在活跃窗口内
	h1.persist(ctx, s1)

	h2 := NewRecordingHub(RecordingHubOption{Cache: shared})
	t.Cleanup(h2.Close)
	sess, ok := h2.GetSession("s-ls")
	require.True(t, ok)
	require.False(t, sess.LastSeen.IsZero(), "回读后 LastSeen 不得为零值")
	require.Equal(t, RecordingSessionRecording, sess.State, "10s 内的活跃会话不得被判失效")
}

// TestRecordingHub_RestartMarksStaleSessionExpired 僵尸会话回读必须判 expired。
//
// 场景：后端重启期间 Worker 也挂了 → Redis 里仍写着「录制中」，但 2 分钟无任何
// 刷新。若不判失效，前端会一直看到「正在录制」的僵尸会话，且它占着并发名额。
func TestRecordingHub_RestartMarksStaleSessionExpired(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute)
	ctx := context.Background()

	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	s1 := seedRecordingSession(h1, "s-stale")
	s1.LastSeen = time.Now().Add(-30 * time.Minute) // 远超 staleSessionAfter
	h1.persist(ctx, s1)

	h2 := NewRecordingHub(RecordingHubOption{Cache: shared})
	t.Cleanup(h2.Close)
	sess, ok := h2.GetSession("s-stale")
	require.True(t, ok, "僵尸会话也要能查到一次（让前端看到「已失效」而不是「不存在」）")
	require.Equal(t, RecordingSessionExpired, sess.State, "30 分钟无刷新的录制中会话必须判 expired")
	require.Contains(t, sess.Message, "失效")
}

// TestRecordingHub_StaleSessionDoesNotConsumeSlot 僵尸会话不得占并发名额。
func TestRecordingHub_StaleSessionDoesNotConsumeSlot(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute)
	ctx := context.Background()

	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	s1 := seedRecordingSession(h1, "s-dead")
	s1.LastSeen = time.Now().Add(-time.Hour)
	h1.persist(ctx, s1)

	cfg := mustCfgRecording(t, 1, 5*time.Minute)
	h2 := NewRecordingHub(RecordingHubOption{Cache: shared, Config: cfg})
	t.Cleanup(h2.Close)

	// 先让它被回读并判成 expired
	_, _ = h2.GetSession("s-dead")

	// max=1，僵尸不占名额 → 新会话必须能开
	_, err := h2.StartSession(ctx, "s-fresh", "t1", "w1", "u1", "甲", "https://x")
	var limitErr *RecordingLimitError
	require.False(t, errors.As(err, &limitErr), "僵尸会话不得占并发名额（%v）", err)
}

// TestRecordingHub_WorkerReportRevivesExpiredSession Worker 上报能推翻「后端推断的 expired」。
//
// 关键区分：expired 是**后端凭 LastSeen 老化推断**的，不是 Worker 声明的终态。
// 只要 Worker 还在上报，就证明执行机活着，必须恢复而不是永久判死。
// （若把 expired 当成不可逆终态，重启期间 Worker 重启后的会话就再也救不回来。）
func TestRecordingHub_WorkerReportRevivesExpiredSession(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute)
	ctx := context.Background()

	// 造一份「后端看来已失效」的快照（30 分钟无刷新）
	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	s1 := seedRecordingSession(h1, "s-revive")
	s1.LastSeen = time.Now().Add(-30 * time.Minute)
	h1.persist(ctx, s1)

	// 新 Hub（模拟重启）：前端先查一次 → 被判 expired
	h2 := NewRecordingHub(RecordingHubOption{Cache: shared})
	t.Cleanup(h2.Close)
	first, ok := h2.GetSession("s-revive")
	require.True(t, ok)
	require.Equal(t, RecordingSessionExpired, first.State, "先按老化判成 expired")

	// Worker 随后上报「仍在录制」→ 必须恢复
	err := h2.ReportStatus(ctx, &RecordingStatusUpdate{
		SessionID:  "s-revive",
		Status:     string(RecordingSessionRecording),
		EventCount: 9,
	})
	require.NoError(t, err, "Worker 上报必须被接受（不得回「会话不存在」）")

	again, ok := h2.GetSession("s-revive")
	require.True(t, ok)
	require.Equal(t, RecordingSessionRecording, again.State, "Worker 上报必须推翻后端推断的 expired")
	require.EqualValues(t, 9, again.EventCount)
}

// TestRecordingHub_WorkerReportRecoversAfterRestart 重启后 Worker 上报能自愈会话。
func TestRecordingHub_WorkerReportRecoversAfterRestart(t *testing.T) {
	shared := cache.NewMemoryCache(64, time.Minute)
	ctx := context.Background()

	h1 := NewRecordingHub(RecordingHubOption{Cache: shared})
	s1 := seedRecordingSession(h1, "s-report-restart")
	s1.LastSeen = time.Now().Add(-5 * time.Second)
	h1.persist(ctx, s1)

	// 新 Hub 内存表为空，且**没有人**先调过 GetSession
	h2 := NewRecordingHub(RecordingHubOption{Cache: shared})
	t.Cleanup(h2.Close)
	require.Empty(t, h2.ListSessions())

	err := h2.ReportStatus(ctx, &RecordingStatusUpdate{
		SessionID:  "s-report-restart",
		Status:     string(RecordingSessionRecording),
		EventCount: 3,
	})
	require.NoError(t, err, "重启后 Worker 上报必须能自愈会话（否则永远丢）")
	require.Len(t, h2.ListSessions(), 1)
}

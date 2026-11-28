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
	s.lastSeen = time.Now().Add(-time.Hour)
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

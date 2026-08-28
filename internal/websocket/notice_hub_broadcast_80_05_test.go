//go:build !skip_db_tests
// +build !skip_db_tests

package websocket

// =====================================================================
// Phase 80-05 Task 3: notice_hub.go Broadcast 群真 WS 对测试。
// (基线 35.7% → ≥70%;BroadcastToUsers/ToAll/RPAProgress×3/encodeMessage。)
//
// harness 形态(照 notice_hub_readpump_test.go 范式 + 本包零改动):
//   - httptest.NewServer + gorilla Upgrader 服务端升级,server conn 交给
//     hub.RegisterClient;测试侧 DefaultDialer.Dial 拿 client conn 读帧。
//   - 关键点:hub 写的是 SERVER 侧 conn(writePump 持有),帧经 TCP 到达
//     client conn —— 由测试启动的 sink goroutine 读取并收集。
//
// 纪律(R5):
//   - 断言一律 assert.Eventually,零裸 time.Sleep 轮询。
//   - 收口顺序照 readpump:hub.Stop() → client conn Close → server.Close()。
//   - 零 t.Parallel(共享 hub fixture)。
// =====================================================================

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nhb8005Client 一对真 WS 连接:hub 侧(server conn,被 writePump 持有)+
// 读侧(client conn,sink goroutine 从这里收帧)。
type nhb8005Client struct {
	userID  string
	readConn *websocket.Conn // 测试侧读帧
}

// nhb8005Sink 帧收集器(mutex 保护,配 Eventually 断言)。
type nhb8005Sink struct {
	mu     sync.Mutex
	frames []string
}

// Frames 返回已收帧快照。
func (s *nhb8005Sink) Frames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.frames))
	copy(out, s.frames)
	return out
}

// start 启动后台读 goroutine,连接关闭时自然退出。
func (s *nhb8005Sink) start(conn *websocket.Conn) {
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.frames = append(s.frames, string(data))
			s.mu.Unlock()
		}
	}()
}

// wsFixture8005 共享的 WS 测试底座(hub + 真服务端 + 服务端 conn 通道)。
// 提供两种注册:有 sink(下游 drain → 帧收集)/ 无 sink(不下游 drain →
// 用来测试 toDelete 死连接分支)。
type wsFixture8005 struct {
	hub            *NoticeHub
	server         *httptest.Server
	wsURL          string
	serverConns    chan *websocket.Conn
	registeredConn []*websocket.Conn
}

func newWSFixture8005(t *testing.T) *wsFixture8005 {
	t.Helper()
	hub := NewNoticeHub()
	go hub.Run()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConns := make(chan *websocket.Conn, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConns <- c
		<-r.Context().Done()
	}))

	f := &wsFixture8005{
		hub:         hub,
		server:      server,
		wsURL:       "ws" + strings.TrimPrefix(server.URL, "http"),
		serverConns: serverConns,
	}

	// 收口顺序照 readpump 先例(hub.Stop → conn close → server.Close)。
	t.Cleanup(func() {
		hub.Stop()
		for _, c := range f.registeredConn {
			_ = c.Close()
		}
		server.Close()
	})
	return f
}

// registerSink 注册带 sink 的客户端(下游 drain → 可用 sink.Frames 断言)。
func (f *wsFixture8005) registerSink(t *testing.T, userID string) (*websocket.Conn, *nhb8005Sink) {
	t.Helper()
	conn, serverConn := f.dialAndRegister(t, userID)
	_ = serverConn
	sink := &nhb8005Sink{}
	sink.start(conn)
	return conn, sink
}

// dialAndRegister 注册(由 RegisterClient 启动 writePump/readPump),
// 用于有 sink 的常规客户端。
func (f *wsFixture8005) dialAndRegister(t *testing.T, userID string) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(f.wsURL, nil)
	require.NoError(t, err)
	serverConn := <-f.serverConns
	f.registeredConn = append(f.registeredConn, conn)
	f.hub.RegisterClient(userID, serverConn)
	return conn, serverConn
}

// waitRegisteredEventually 等待 hub.clients 长度等于 expected(R5:非 sleep)。
func (f *wsFixture8005) waitRegisteredEventually(t *testing.T, expected int) {
	t.Helper()
	assert.Eventually(t, func() bool {
		f.hub.mu.RLock()
		defer f.hub.mu.RUnlock()
		return len(f.hub.clients) == expected
	}, 2*time.Second, 10*time.Millisecond, "hub.clients 应有 %d 个客户端", expected)
}

// newHubWithClients8005 起 n 个有 sink 客户端(便捷包装)。
func newHubWithClients8005(t *testing.T, n int) (*NoticeHub, []nhb8005Client, []*nhb8005Sink) {
	t.Helper()
	f := newWSFixture8005(t)
	clients := make([]nhb8005Client, 0, n)
	sinks := make([]*nhb8005Sink, 0, n)
	for i := 0; i < n; i++ {
		userID := fmt.Sprintf("nhb8005-user-%d", i)
		conn, sink := f.registerSink(t, userID)
		clients = append(clients, nhb8005Client{userID: userID, readConn: conn})
		sinks = append(sinks, sink)
	}
	f.waitRegisteredEventually(t, n)
	return f.hub, clients, sinks
}

// nhb8005SampleNotice 样例通知消息。
func nhb8005SampleNotice() NoticeMessage {
	return NoticeMessage{
		Type:      "new_notice",
		NoticeID:  "n-8005",
		Title:     "测试通知",
		Content:   "广播内容",
		Priority:  2,
		Timestamp: 1700000000,
	}
}

// TestNhb8005_BroadcastToAll:2 客户端 → BroadcastToAll → 两端各收 1 帧,
// JSON 字段断言(encodeMessage 产物)。
func TestNhb8005_BroadcastToAll(t *testing.T) {
	hub, clients, sinks := newHubWithClients8005(t, 2)
	msg := nhb8005SampleNotice()
	hub.BroadcastToAll(msg)

	for i, sink := range sinks {
		_ = i
		assert.Eventually(t, func() bool { return len(sink.Frames()) == 1 },
			2*time.Second, 10*time.Millisecond, "客户端 %d 应收到 1 帧", i)
	}

	var got NoticeMessage
	require.NoError(t, json.Unmarshal([]byte(sinks[0].Frames()[0]), &got))
	assert.Equal(t, msg.Type, got.Type)
	assert.Equal(t, msg.NoticeID, got.NoticeID)
	assert.Equal(t, msg.Title, got.Title)
	assert.Equal(t, msg.Content, got.Content)
	assert.Equal(t, msg.Priority, got.Priority)
	assert.Equal(t, msg.Timestamp, got.Timestamp)
	_ = clients
}

// TestNhb8005_BroadcastToUsers:目标/非目标各一 → 只有目标收到;
// 空 userIDs 与不存在 userID → 无人收到、不 panic。
func TestNhb8005_BroadcastToUsers(t *testing.T) {
	hub, clients, sinks := newHubWithClients8005(t, 2)

	hub.BroadcastToUsers([]string{clients[0].userID}, nhb8005SampleNotice())

	assert.Eventually(t, func() bool { return len(sinks[0].Frames()) == 1 },
		2*time.Second, 10*time.Millisecond, "目标用户应收到帧")
	assert.Eventually(t, func() bool { return len(sinks[1].Frames()) == 0 },
		200*time.Millisecond, 20*time.Millisecond, "非目标用户不应收到帧")

	// 空 userIDs → 无人收到。
	hub.BroadcastToUsers(nil, nhb8005SampleNotice())
	// 不存在 userID → 直接跳过(ok 分支不命中)。
	hub.BroadcastToUsers([]string{"ghost-user"}, nhb8005SampleNotice())

	assert.Eventually(t, func() bool {
		return len(sinks[0].Frames()) == 1 && len(sinks[1].Frames()) == 0
	}, 300*time.Millisecond, 20*time.Millisecond, "空/不存在 userIDs 不产生新帧")
}

// TestNhb8005_BroadcastRPAProgress_Trio:BroadcastRPAProgress / ToUser /
// ToUsers 三方法各一用例,RPAProgressMessage 字段透传断言。
func TestNhb8005_BroadcastRPAProgress_Trio(t *testing.T) {
	hub, clients, sinks := newHubWithClients8005(t, 3)

	progress := RPAProgressMessage{
		Type:        MessageTypeRPAProgress,
		ExecutionID: "exec-8005",
		TaskID:      "task-8005",
		TaskName:    "巡检任务",
		Step:        2,
		Total:       5,
		Message:     "执行中",
		Status:      "running",
		Timestamp:   1700000001,
	}

	// (1) BroadcastRPAProgress:全员;RPA 消息被包进 NoticeMessage.Content。
	hub.BroadcastRPAProgress(progress)
	for i, sink := range sinks {
		assert.Eventually(t, func() bool { return len(sink.Frames()) == 1 },
			2*time.Second, 10*time.Millisecond, "全员广播:客户端 %d 应收到 1 帧", i)
	}
	var envelope NoticeMessage
	require.NoError(t, json.Unmarshal([]byte(sinks[0].Frames()[0]), &envelope))
	assert.Equal(t, MessageTypeRPAProgress, envelope.Type, "外层 Type 透传 RPA 消息类型")
	assert.NotZero(t, envelope.Timestamp, "外层时间戳由 hub 补齐")
	var inner RPAProgressMessage
	require.NoError(t, json.Unmarshal([]byte(envelope.Content), &inner))
	assert.Equal(t, progress, inner, "Content 应为 RPAProgressMessage 原 JSON")

	// (2) BroadcastRPAProgressToUser:仅目标。
	progress.Step = 3
	hub.BroadcastRPAProgressToUser(clients[1].userID, progress)
	assert.Eventually(t, func() bool { return len(sinks[1].Frames()) == 2 },
		2*time.Second, 10*time.Millisecond, "目标用户收到第 2 帧")
	assert.Eventually(t, func() bool {
		return len(sinks[0].Frames()) == 1 && len(sinks[2].Frames()) == 1
	}, 300*time.Millisecond, 20*time.Millisecond, "非目标不增帧")

	// (3) BroadcastRPAProgressToUsers:子集。
	progress.Step = 4
	hub.BroadcastRPAProgressToUsers([]string{clients[0].userID, clients[2].userID}, progress)
	assert.Eventually(t, func() bool { return len(sinks[0].Frames()) == 2 },
		2*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return len(sinks[2].Frames()) == 2 },
		2*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return len(sinks[1].Frames()) == 2 },
		300*time.Millisecond, 20*time.Millisecond, "非子集成员不增帧")
}

// TestNhb8005_EncodeMessage:合法 msg → JSON 可解析。
// 错误分支(:304 json.Marshal err)对 NoticeMessage 不可达 —— 全字段均为
// string/int/json.Marshal 永不报错 → 按 D-80-04 豁免口径落 SUMMARY
// (3 stmts:fmt.Errorf 包装 + nil 返回,防御式代码,无可达输入)。
func TestNhb8005_EncodeMessage(t *testing.T) {
	hub := NewNoticeHub()

	data, err := hub.encodeMessage(nhb8005SampleNotice())
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var got NoticeMessage
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "测试通知", got.Title)

	// 零值消息同样成功(无不可序列化字段)。
	_, err = hub.encodeMessage(NoticeMessage{})
	assert.NoError(t, err)
}

// TestNhb8005_Broadcast_DisconnectedClient_BroadcastToUsers:BroadcastToUsers
// 中 client.send 满 256 → select 命中 default → toDelete → 注销,在线者
// 不受影响、广播不 panic。
// 关键: 同包白盒构造 victim Client 并直注 hub.clients(不经 RegisterClient
// → 不起 writePump/readPump → 不会主动 drain send),确定性触发 default 分支。
func TestNhb8005_Broadcast_DisconnectedClient_BroadcastToUsers(t *testing.T) {
	f := newWSFixture8005(t)

	_, observerSink := f.registerSink(t, "observer-btu-8005")
	_, controlSink := f.registerSink(t, "control-btu-8005")
	f.waitRegisteredEventually(t, 2)

	hub := f.hub

	// 直注 victim(无 writePump/readPump):send 满 256 后 BroadcastToUsers
	// 内 select 立即命中 default → toDelete → 删除 + 关闭 send。
	victimUserID := "victim-btu-8005"
	victim := &Client{
		userID: victimUserID,
		conn:   nil,
		send:   make(chan []byte, noticeHubBufferSize),
		hub:    hub,
	}
	hub.mu.Lock()
	hub.clients[victimUserID] = victim
	hub.mu.Unlock()

	for i := 0; i < noticeHubBufferSize; i++ {
		victim.send <- []byte("filler")
	}

	hub.BroadcastToUsers([]string{victimUserID, "observer-btu-8005"}, nhb8005SampleNotice())

	assert.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, exists := hub.clients[victimUserID]
		return !exists
	}, 2*time.Second, 10*time.Millisecond, "send 满 256 后 victim 应被清理")

	// 在线者仍可收到新消息。
	hub.BroadcastToUsers([]string{"observer-btu-8005"}, nhb8005SampleNotice())
	assert.Eventually(t, func() bool { return len(observerSink.Frames()) >= 1 },
		2*time.Second, 10*time.Millisecond, "在线观察者应继续收到广播")

	assert.Eventually(t, func() bool { return len(controlSink.Frames()) == 0 },
		300*time.Millisecond, 20*time.Millisecond, "未订阅对照客户端不应收到定向广播")
}

// TestNhb8005_Broadcast_DisconnectedClient_RunLoop:同型清理走 Run 的
// broadcast 分支(BroadcastToAll)—— Run 循环内 toDelete 收集 + 删除。
func TestNhb8005_Broadcast_DisconnectedClient_RunLoop(t *testing.T) {
	f := newWSFixture8005(t)

	_, observerSink := f.registerSink(t, "observer-run-8005")
	f.waitRegisteredEventually(t, 1)

	hub := f.hub

	victimUserID := "victim-run-8005"
	victim := &Client{
		userID: victimUserID,
		conn:   nil,
		send:   make(chan []byte, noticeHubBufferSize),
		hub:    hub,
	}
	hub.mu.Lock()
	hub.clients[victimUserID] = victim
	hub.mu.Unlock()

	for i := 0; i < noticeHubBufferSize; i++ {
		victim.send <- []byte("filler")
	}

	// BroadcastToAll 经 Run 循环 → select 命中 default → victim 移出 map。
	hub.BroadcastToAll(nhb8005SampleNotice())

	assert.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		_, exists := hub.clients[victimUserID]
		return !exists
	}, 2*time.Second, 10*time.Millisecond, "Run 循环应清理 send 满的客户端")

	assert.Eventually(t, func() bool { return len(observerSink.Frames()) >= 1 },
		2*time.Second, 10*time.Millisecond, "在线观察者应持续收到全员广播")
}

// TestNhb8005_BroadcastToAll_NoClients:无客户端时空转不 panic。
func TestNhb8005_BroadcastToAll_NoClients(t *testing.T) {
	hub := NewNoticeHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	assert.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == 0
	}, time.Second, 10*time.Millisecond)

	assert.NotPanics(t, func() {
		hub.BroadcastToAll(nhb8005SampleNotice())
		hub.BroadcastToUsers([]string{"nobody"}, nhb8005SampleNotice())
	})
}

package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01：实时画面专用广播的「可丢帧不丢连接」契约。
//
// 为什么这条契约重要：既有 BroadcastToUsers 在 client.send 满时会**删除该用户
// 连接**（见 notice_hub.go:141-158 与 BroadcastToUsers）。帧流是持续高频，
// 任何一次瞬时拥塞都会把用户踢下线（前端一次 5 秒卡顿就永久失去 WS）。
// 本文件的用例锁定新行为：队列满 → 丢帧 + 计数，连接保留。
// ============================================================================

// TestPublishLiveToUsers_DropsFrameKeepsConnection 队列满时丢帧但不删连接。
func TestPublishLiveToUsers_DropsFrameKeepsConnection(t *testing.T) {
	hub := NewNoticeHub()
	go hub.Run()
	defer hub.Stop()

	// 用真实 *websocket.Conn 需要一个 server；这里直接注入 client（不经过
	// RegisterClient 的写泵，避免写泵把队列消费掉）。
	conn, cleanup := newTestWSConn(t)
	defer cleanup()

	// send 容量 1：塞满后新帧应被丢弃。
	client := &Client{
		userID: "live-user-1",
		conn:   conn,
		send:   make(chan []byte, 1),
		hub:    hub,
	}
	client.send <- []byte("occupied") // 占满队列

	hub.mu.Lock()
	hub.clients["live-user-1"] = client
	hub.mu.Unlock()

	// 第一次推送：队列满 → 丢弃 1 帧。
	dropped := hub.PublishLiveToUsers([]string{"live-user-1"}, "exec-1", MessageTypeRPALiveFrame, `{"frame":{"seq":1}}`)
	if dropped != 1 {
		t.Fatalf("队列满时应丢弃 1 帧，实际 dropped=%d", dropped)
	}

	// **关键断言**：连接必须仍在 hub 里（不得被删除）。
	hub.mu.RLock()
	_, stillThere := hub.clients["live-user-1"]
	hub.mu.RUnlock()
	if !stillThere {
		t.Fatalf("帧丢弃后用户连接被删除（违反「可丢帧不丢连接」契约）")
	}

	// 消费掉占位消息后，再次推送应成功入队（dropped=0）。
	<-client.send
	dropped = hub.PublishLiveToUsers([]string{"live-user-1"}, "exec-1", MessageTypeRPALiveFrame, `{"frame":{"seq":2}}`)
	if dropped != 0 {
		t.Fatalf("队列有空位时应成功入队，实际 dropped=%d", dropped)
	}
	select {
	case msg := <-client.send:
		if len(msg) == 0 {
			t.Fatalf("入队的消息为空")
		}
	case <-time.After(time.Second):
		t.Fatalf("消息未入队")
	}

	// 进程级丢弃计数应 >= 1（观测用）。
	if got := DroppedLiveFrames(); got < 1 {
		t.Fatalf("DroppedLiveFrames()=%d，期望 >=1", got)
	}
}

// TestPublishLiveToUsers_OfflineUserNoDrop 不在线的用户不算丢弃（不是丢帧，是没送）。
func TestPublishLiveToUsers_OfflineUserNoDrop(t *testing.T) {
	hub := NewNoticeHub()
	go hub.Run()
	defer hub.Stop()

	if dropped := hub.PublishLiveToUsers([]string{"nobody"}, "exec-2", MessageTypeRPALiveFrame, `{}`); dropped != 0 {
		t.Fatalf("离线用户不应计为丢弃，实际 dropped=%d", dropped)
	}
}

// TestLiveFrameMessageTypes 两个新消息类型的取值必须稳定（前端按 type 分流）。
func TestLiveFrameMessageTypes(t *testing.T) {
	if MessageTypeRPALiveFrame != "rpa_liveview_frame" {
		t.Fatalf("MessageTypeRPALiveFrame=%q，契约值应为 rpa_liveview_frame", MessageTypeRPALiveFrame)
	}
	if MessageTypeRPALiveState != "rpa_liveview_state" {
		t.Fatalf("MessageTypeRPALiveState=%q，契约值应为 rpa_liveview_state", MessageTypeRPALiveState)
	}
}

// newTestWSConn 起一个本地 WS 服务端并返回已连接的客户端（只用于注入 hub）。
//
// 本用例不需要真的收发帧（只验证「队列满时是否删连接」），故用 gorilla 的
// 官方 Upgrader 起一个最小 echo 服务端；写泵不参与（client 直接注入 hub）。
func newTestWSConn(t *testing.T) (*websocket.Conn, func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// 保持连接存活直到对端关闭。
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				_ = c.Close()
				return
			}
		}
	}))
	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("连接测试 WS 服务端失败: %v", err)
	}
	return conn, func() {
		_ = conn.Close()
		srv.Close()
	}
}

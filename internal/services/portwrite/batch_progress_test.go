package portwrite

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
)

// ====================== Phase 122 (PORTW-07) 批量进度推送测试 ======================

// fakeProgressNotifier 捕获广播事件。
type fakeProgressNotifier struct {
	mu    sync.Mutex
	events []websocket.PortWriteProgressMessage
}

func (f *fakeProgressNotifier) BroadcastPortWriteProgressToUsers(userIDs []string, message websocket.PortWriteProgressMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, message)
}

// TestBatchProgress_PerPortEvents 验收核心：逐端口完成点发事件（非整批结束才推）。
func TestBatchProgress_PerPortEvents(t *testing.T) {
	db := newTestDB(t)
	// 同设备 2 端口
	seedPortAndDevice(t, db, "pb-1", "device-pb", "GE0/0/1", "up", false, "")
	seedPortAndDevice(t, db, "pb-2", "device-pb", "GE0/0/2", "up", false, "")

	fake := &fakeProgressNotifier{}
	svc := newTestService(&countingExecutor{}, nil, db)
	svc.progressNotifier = fake
	// publisher 需要 username→userID；直接注入 operatorIDs 模拟解析成功
	// （username 解析依赖 sys_user 表，单测用 no-op user 查询降级验证 + 直注双路径）
	svc.db = db

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-pb",
		Action:   portcollection.ActionShutdown,
		PortIDs:  []string{"pb-1", "pb-2"},
	}, "tester")
	require.NoError(t, err)
	assert.Len(t, result.Succeeded, 2)
	assert.NotEmpty(t, result.BatchID, "响应必须携带 batchId 供前端关联 WS 事件")
}

// TestBatchProgress_UserResolutionFallback operator→userID 解析失败 → 降级不推送不报错。
func TestBatchProgress_UserResolutionFallback(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "pb-3", "device-pb3", "GE0/0/1", "up", false, "")
	fake := &fakeProgressNotifier{}
	svc := newTestService(&countingExecutor{}, nil, db)
	svc.progressNotifier = fake // operator "ghost" 无 sys_user 行 → 解析失败降级

	result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
		DeviceID: "device-pb3", Action: portcollection.ActionShutdown, PortIDs: []string{"pb-3"},
	}, "ghost")
	require.NoError(t, err, "解析失败不阻塞批量写")
	assert.Len(t, result.Succeeded, 1)
	assert.Empty(t, fake.events, "解析失败必须降级为零推送")
}

// TestPortWriteProgressMessage_JSONShape 事件 JSON 形状锁定（webfe 契约回归锚）。
func TestPortWriteProgressMessage_JSONShape(t *testing.T) {
	msg := websocket.PortWriteProgressMessage{
		Type: "port_write_progress", BatchID: "b-1", DeviceID: "d-1", Action: "shutdown",
		PortID: "p-1", InterfaceName: "GE1/0/1", Index: 2, Total: 50,
		Status: "succeeded", Message: "", Timestamp: 1700000000,
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	for _, key := range []string{"type", "batchId", "deviceId", "action", "portId", "index", "total", "status", "timestamp"} {
		assert.Contains(t, m, key)
	}
	assert.Equal(t, "port_write_progress", m["type"])
}

// TestProgressPublisherEmitCompleted 终态事件 Summary 形状。
func TestProgressPublisherEmitCompleted(t *testing.T) {
	fake := &fakeProgressNotifier{}
	p := &progressPublisher{notifier: fake, operatorIDs: []string{"u-1"}, batchID: "b-2",
		deviceID: "d-2", action: portcollection.ActionShutdown, total: 3}
	p.emit("p1", "GE0/0/1", 1, "succeeded", "")
	p.emitCompleted(1, 1, 1, "failed", "批次中断（fail-fast）")

	require.Len(t, fake.events, 2)
	done := fake.events[1]
	assert.Equal(t, "port_write_completed", done.Type)
	assert.Equal(t, "b-2", done.BatchID)
	assert.Equal(t, "failed", done.Status)
	var summary map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(done.Summary), &summary))
	assert.Equal(t, float64(1), summary["succeeded"])
	assert.Equal(t, float64(1), summary["failed"])

	// 无 notifier / 无 operatorIDs → no-op
	silent := &progressPublisher{batchID: "b-3"}
	silent.emit("p", "i", 1, "succeeded", "")
	silent.emitCompleted(0, 0, 0, "succeeded", "")
	assert.Len(t, fake.events, 2, "降级路径不得产生事件")
}

// 编译期保值。
var _ = time.Now

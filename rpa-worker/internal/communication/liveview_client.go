package communication

import (
	"context"
	"fmt"

	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01 + LIVEVIEW-05：帧与审计的上行出口。
//
// 为什么走 HTTP 而不是 Worker→后端 WS：
//   Worker 现有通信面是纯 HTTP client（APIClient）+ Redis Pub/Sub。新增一条
//   Worker→后端 WS 长连会引入第二套重连/鉴权/背压生命周期，而帧本身的背压
//   已经由 CDP 的逐帧 ACK 实现（见 browser/screencast.go）。故帧走 HTTP POST，
//   后端收到后用**既有** NoticeHub 推前端 —— 协议要求「经既有 WS 通道推前端」
//   成立，且零新增依赖。
// ============================================================================

// PushFrame LIVEVIEW-01：上报一帧画面 + 最新叠加层情境。
//
// 实现 `liveview.FrameSink`。超时由调用方通过 ctx 控制（帧是延迟敏感路径）。
func (c *APIClient) PushFrame(ctx context.Context, frame *types.LiveViewUpstreamFrame) error {
	if frame == nil {
		return fmt.Errorf("帧上报失败: payload 为 nil")
	}
	if frame.Frame.ExecutionID == "" {
		return fmt.Errorf("帧上报失败: executionId 为空")
	}
	return c.post(ctx, "/rpa/workers/liveview/frame", frame, nil)
}

// PushEvent LIVEVIEW-05：上报审计事件（watch_start/watch_stop/takeover/release/
// mode_switch/input）。
//
// 实现 `liveview.AuditSink`。审计失败必须由调用方记日志，不得静默 —— 但不得
// 阻断画面流（可用性优先，审计的持久化由后端保证）。
func (c *APIClient) PushEvent(ctx context.Context, ev *types.LiveViewAuditEvent) error {
	if ev == nil {
		return fmt.Errorf("审计上报失败: payload 为 nil")
	}
	if ev.ExecutionID == "" || !types.IsValidLiveViewEvent(ev.Event) {
		return fmt.Errorf("审计上报失败: executionId 为空或事件类型非法 (%q)", ev.Event)
	}
	return c.post(ctx, "/rpa/workers/liveview/event", ev, nil)
}

// 编译期断言：APIClient 满足 liveview 的两个 sink 契约。
// （此处不 import liveview 包，避免 communication → liveview 的反向依赖；
//  断言放在 liveview 包的测试里做。）

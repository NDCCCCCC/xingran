package worker

// ============================================================================
// Phase W6 / REC-01：录制会话下行指令的处理循环。
//
// 与 155 liveview_listener.go 同族先例：
//   - 一个 goroutine 覆盖所有会话（PSubscribe recording:session:*），
//     不按会话起 goroutine（避免反复 start/stop 导致 goroutine 增长）。
//   - 频道名 recording:session:{sessionId}，从 channel 反解 sessionID。
//
// 指令 → 动作：
//
//	start      开页面 + 注入录制器 + 开始采集
//	stop       拉事件 + 产出 actions + 上行给后端 + 释放页面
//	pick       按坐标拾取元素（7 级阶梯）
//	keepalive  刷新会话闲置计时
//	abort      强制中止（释放页面）
//
// **画面不在此通道**：录制会话的画面帧复用 155 的 liveview 帧通道
//（execution:liveview:{sessionId}），坐标系与可丢广播语义共用一套。
// ============================================================================

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// recordingCommandListener 订阅并处理录制会话指令。
func (w *Worker) recordingCommandListener() {
	defer w.wg.Done()

	pubsub, err := w.redisClient.SubscribeRecording(w.ctx)
	if err != nil {
		w.logger.Error("subscribe recording commands failed", logger.Err(err))
		return
	}
	defer pubsub.Close()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.shutdownSignal:
			return
		case msg := <-pubsub.Channel():
			if msg == nil || msg.Payload == "" {
				continue
			}
			// 频道 recording:session:{sessionId}
			sessionID := strings.TrimPrefix(msg.Channel, "recording:session:")
			if sessionID == "" || sessionID == msg.Channel {
				continue
			}

			var cmd types.RecordingCommand
			if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
				w.logger.Error("failed to parse recording command",
					logger.Err(err),
					logger.String("session_id", sessionID))
				continue
			}
			if cmd.SessionID == "" {
				cmd.SessionID = sessionID
			}
			// 指令指定了别的 Worker → 拒收（多 Worker 集群下的定向投递）。
			if cmd.WorkerID != "" && cmd.WorkerID != w.id {
				continue
			}
			w.handleRecordingCommand(&cmd)
		}
	}
}

// handleRecordingCommand 分派一条录制指令。
func (w *Worker) handleRecordingCommand(cmd *types.RecordingCommand) {
	if w.recordingHub == nil || cmd == nil {
		return
	}
	// 任何指令都刷新闲置计时（orphan page 回收的依据）。
	w.recordingHub.Keepalive(cmd.SessionID)

	switch cmd.Action {
	case types.RecordingActionStart:
		if err := w.recordingHub.Start(w.browserManager, cmd); err != nil {
			w.logger.Warn("recording start failed",
				logger.String("session_id", cmd.SessionID),
				logger.Err(err))
		}
	case types.RecordingActionStop:
		if _, err := w.recordingHub.Stop(w.browserManager, cmd.SessionID); err != nil {
			w.logger.Warn("recording stop failed",
				logger.String("session_id", cmd.SessionID),
				logger.Err(err))
		}
	case types.RecordingActionPick:
		if _, err := w.recordingHub.Pick(cmd.SessionID, cmd.PickX, cmd.PickY); err != nil {
			w.logger.Warn("recording pick failed",
				logger.String("session_id", cmd.SessionID),
				logger.Err(err))
		}
	case types.RecordingActionKeepalive:
		// Keepalive 已在入口完成，无需其它动作。
	case types.RecordingActionAbort:
		w.recordingHub.Abort(w.browserManager, cmd.SessionID, "后端要求中止")
	default:
		w.logger.Debug("unknown recording command ignored",
			logger.String("session_id", cmd.SessionID),
			logger.String("action", string(cmd.Action)))
	}
}

// recordingPollLoop 周期性上报录制状态（让前端显示「正在录制」）。
//
// 频率刻意保守（2s 一次）：状态是低频信息，不需要推流级别的密度。
func (w *Worker) recordingPollLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.shutdownSignal:
			return
		case <-ticker.C:
			if w.recordingHub == nil {
				continue
			}
			w.recordingHub.PollEvents(w.browserManager)
		}
	}
}

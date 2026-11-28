package worker

import (
	"encoding/json"
	"strings"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/liveview"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-02/03/05：实时画面下行指令的处理循环。
//
// 一个 goroutine 覆盖所有执行的指令（PSubscribe execution:liveview:*），
// 与 RUNTIME-03 的 cancelCommandListener 同族先例（不按执行起 goroutine，
// 因此 watch/stop 反复 100 次不会增长 goroutine 数 —— LIVEVIEW-05 泄漏验收）。
//
// 指令 → 动作：
//
//	watch            开流（占名额，超限返回可读错误由后端转 429）
//	stop             停流
//	keepalive        刷新闲置计时（LIVEVIEW-05：断开 N 秒才停流，不是立即）
//	input            注入鼠标/键盘（仅接管者）
//	takeover/release 控制权转移（单 page 单接管者）
//	mode             切模式（auto/copilot/manual，非法值拒绝）
//	continue         放行被闸门阻塞的动作
//	abort            人工中止（走 RUNTIME-03 的 ctx cancel，得到 cancelled 终态）
// ============================================================================

// liveViewCommandListener 订阅并处理实时画面指令。
func (w *Worker) liveViewCommandListener() {
	defer w.wg.Done()

	pubsub, err := w.redisClient.SubscribeLiveView(w.ctx)
	if err != nil {
		w.logger.Error("subscribe liveview commands failed", logger.Err(err))
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
			executionID := strings.TrimPrefix(msg.Channel, "execution:liveview:")
			if executionID == "" || executionID == msg.Channel {
				continue
			}

			var cmd types.LiveViewCommand
			if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
				w.logger.Error("failed to parse liveview command",
					logger.Err(err),
					logger.String("execution_id", executionID))
				continue
			}
			if cmd.ExecutionID == "" {
				cmd.ExecutionID = executionID
			}
			w.handleLiveViewCommand(executionID, &cmd)
		}
	}
}

// handleLiveViewCommand 分派一条指令。
func (w *Worker) handleLiveViewCommand(executionID string, cmd *types.LiveViewCommand) {
	if w.liveView == nil || cmd == nil {
		return
	}
	// 任何下行指令都刷新空闲计时（LIVEVIEW-05：断流判定看的是「有没有人还在看」）。
	w.liveView.Touch(executionID)

	switch cmd.Action {
	case types.LiveCmdWatch:
		if err := w.liveView.Watch(executionID, cmd.Operator); err != nil {
			w.logger.Warn("liveview watch failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdStop:
		w.liveView.Stop(executionID, cmd.Operator, "前端主动停流")
	case types.LiveCmdKeepalive:
		// Touch 已在入口完成，无需其它动作。
	case types.LiveCmdInput:
		if cmd.Input == nil {
			return
		}
		if cmd.Input.Operator == "" {
			cmd.Input.Operator = cmd.Operator
		}
		if err := w.liveView.Input(executionID, cmd.Input); err != nil {
			w.logger.Warn("liveview input rejected/failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdTakeover:
		if err := w.liveView.Takeover(executionID, cmd.Operator, cmd.Force); err != nil {
			w.logger.Warn("liveview takeover failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdRelease:
		if err := w.liveView.Release(executionID, cmd.Operator, cmd.Reason); err != nil {
			w.logger.Warn("liveview release failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdMode:
		if err := w.liveView.SetMode(executionID, cmd.Mode, cmd.Operator); err != nil {
			w.logger.Warn("liveview mode switch failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdContinue:
		if err := w.liveView.Continue(executionID, cmd.Operator, cmd.Reason); err != nil {
			w.logger.Warn("liveview continue failed",
				logger.String("execution_id", executionID),
				logger.Err(err))
		}
	case types.LiveCmdAbort:
		w.abortExecution(executionID, cmd.Operator, "人工中止（LIVEVIEW-03）")
	default:
		w.logger.Debug("unknown liveview command ignored",
			logger.String("execution_id", executionID),
			logger.String("action", cmd.Action))
	}
}

// abortExecution 人工中止：复用 RUNTIME-03 的取消管道（ctx cancel → cancelled 终态），
// 不另造中止路径（否则执行终态判定会分叉成两套）。
func (w *Worker) abortExecution(executionID, operator, reason string) {
	w.activeCancelsMu.Lock()
	cancelFn, ok := w.activeCancels[executionID]
	w.activeCancelsMu.Unlock()
	if !ok {
		w.logger.Debug("abort for unknown/finished execution ignored",
			logger.String("execution_id", executionID))
		return
	}
	w.logger.Warn("liveview abort received, cancelling context",
		logger.String("execution_id", executionID),
		logger.String("operator", operator),
		logger.String("reason", reason))
	cancelFn()
}

// attachLiveView 执行开始时登记可推流（engine 在 Acquire 后调用）。
//
// 薄封装保证 nil 安全：单测构造的 Worker 或未启用 liveview 时无副作用。
func (w *Worker) attachLiveView(executionID string, page browser.Page, meta *types.LiveViewContext) {
	if w == nil || w.liveView == nil {
		return
	}
	w.liveView.Attach(executionID, page, meta)
}

// detachLiveView 执行结束注销（engine 在 Release 前调用）。
func (w *Worker) detachLiveView(executionID string) {
	if w == nil || w.liveView == nil {
		return
	}
	w.liveView.Detach(executionID)
}

// LiveViewHub 暴露会话表（health/metrics 与测试用）。
func (w *Worker) LiveViewHub() *liveview.Hub {
	if w == nil {
		return nil
	}
	return w.liveView
}

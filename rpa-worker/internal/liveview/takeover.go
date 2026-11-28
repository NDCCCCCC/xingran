package liveview

import (
	"fmt"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-02 + LIVEVIEW-03：接管、模式与动作边界闸门。
//
// 三条硬约束的落点：
//
//  1. **单 page 同时只允许一个接管者**（LIVEVIEW-05）。粒度是 page 而非 execution：
//     同一个 Manager 下不同 page 是独立标签页，可以各有接管者；同一 page 上
//     两个 execution 共用（不会发生，Acquire 独占）但抢占仍要显式确认。
//  2. **接管期间 RPA 动作暂停**（LIVEVIEW-02/03）。在执行的动作边界插闸门：
//     BeforeAction 返回 GateWaitConfirm 时阻塞，直到 Continue/Release/Stop。
//  3. **模式切换复用 RUNTIME-05 校验**：本文件**不**判断 running→paused 合法性，
//     只把模式与「是否需要人工确认」映射成闸门决策；真正的状态迁移发生在后端
//     （CanTransition/CheckTransition），Worker 侧只按命令行事。
// ============================================================================

var (
	errNotEnabled   = fmt.Errorf("实时画面未启用（worker.liveview.enabled=false）")
	errNotRunning   = fmt.Errorf("执行不在本 Worker 运行或已结束")
	errNoSession    = fmt.Errorf("该执行未登记实时画面会话")
	errNotTaker     = fmt.Errorf("当前不是接管者，无权注入输入")
	errAlreadyTaken = fmt.Errorf("该画面已被他人接管")
)

// LimitError 并发推流超限（LIVEVIEW-01/05：可读错误，不静默降级）。
type LimitError struct {
	Max int
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("单 Worker 实时画面推流路数已达上限 %d，请稍后重试或等待他人结束观看", e.Max)
}

// Takeover 申请接管（单 page 单接管者）。force=false 时若已被他人接管则报错，
// 需要前端显式二次确认后带 force=true 重试（LIVEVIEW-05「抢占需显式确认」）。
func (h *Hub) Takeover(executionID, operator string, force bool) error {
	if h == nil || executionID == "" {
		return errNoSession
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return errNotRunning
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.takeoverBy != "" && s.takeoverBy != operator && !force {
		return errAlreadyTaken
	}
	if s.takeoverBy == operator {
		// 幂等：重复接管只刷新活跃时间。
		s.lastSeen = time.Now()
		return nil
	}
	s.takeoverBy = operator
	s.takeoverAt = time.Now()
	s.lastSeen = time.Now()
	if s.context != nil {
		s.context.TakeoverBy = operator
		s.context.TakeoverAtMS = s.takeoverAt.UnixMilli()
	}
	// 接管即暂停后续 RPA 动作：让闸门进入等待态。
	s.confirmCh = make(chan struct{})

	if h.audit != nil {
		go func() {
			ev := &types.LiveViewAuditEvent{
				ExecutionID: executionID,
				Event:       types.LiveViewEventTakeover,
				Operator:    operator,
				Detail:      forceNote(force),
				TimestampMS: types.NowMS(),
			}
			_ = h.audit.PushEvent(h.ctx, ev)
		}()
	}
	if h.logger != nil {
		h.logger.Info("liveview takeover",
			logger.String("execution_id", executionID),
			logger.String("operator", operator),
			logger.Any("force", force))
	}
	return nil
}

// forceNote 抢占的审计说明（普通接管为空）。
func forceNote(force bool) string {
	if force {
		return "force=true（抢占他人接管）"
	}
	return ""
}

// Release 释放接管：放行被闸门阻塞的动作，让 RPA 从断点继续（LIVEVIEW-03）。
func (h *Hub) Release(executionID, operator, reason string) error {
	if h == nil || executionID == "" {
		return errNoSession
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return errNotRunning
	}

	s.mu.Lock()
	taker := s.takeoverBy
	if taker == "" {
		s.mu.Unlock()
		return nil // 幂等：本来就没有接管者
	}
	if operator != "" && operator != taker {
		s.mu.Unlock()
		return fmt.Errorf("释放失败：当前接管者是 %s，不是 %s", taker, operator)
	}
	s.takeoverBy = ""
	s.takeoverAt = time.Time{}
	if s.context != nil {
		s.context.TakeoverBy = ""
		s.context.TakeoverAtMS = 0
	}
	ch := s.confirmCh
	s.confirmCh = make(chan struct{})
	s.mu.Unlock()

	safeClose(ch)

	if h.audit != nil {
		ctx, cancel := timeoutCtx(3 * time.Second)
		defer cancel()
		_ = h.audit.PushEvent(ctx, &types.LiveViewAuditEvent{
			ExecutionID: executionID,
			Event:       types.LiveViewEventRelease,
			Operator:    operator,
			Reason:      reason,
			TimestampMS: types.NowMS(),
		})
	}
	if h.logger != nil {
		h.logger.Info("liveview takeover released",
			logger.String("execution_id", executionID),
			logger.String("operator", operator),
			logger.String("reason", reason))
	}
	return nil
}

// TakeoverBy 当前接管者（空串表示自动执行）。叠加层「已接管 / 自动执行」角标用。
func (h *Hub) TakeoverBy(executionID string) string {
	if h == nil || executionID == "" {
		return ""
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.takeoverBy
}

// Mode 当前协作模式（默认 copilot）。
func (h *Hub) Mode(executionID string) types.LiveMode {
	if h == nil || executionID == "" {
		return ""
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// SetMode 切换协作模式。
//
// **不接受未知取值**：非法模式直接报错（LIVEVIEW-03 要求显式状态机，
// 不静默降级）。模式切换本身不改变执行状态 —— 状态迁移由后端按
// RUNTIME-05 裁决后，用 Continue 指令放行/暂停。
func (h *Hub) SetMode(executionID string, mode types.LiveMode, operator string) error {
	if h == nil || executionID == "" {
		return errNoSession
	}
	normalized, ok := types.NormalizeLiveMode(mode)
	if !ok {
		return fmt.Errorf("非法协作模式 %q（允许 auto|copilot|manual）", mode)
	}

	h.mu.Lock()
	s, found := h.sessions[executionID]
	h.mu.Unlock()
	if !found || s == nil {
		return errNotRunning
	}

	s.mu.Lock()
	old := s.mode
	s.mode = normalized
	if s.context != nil {
		s.context.Mode = normalized
	}
	// 从 copilot/manual 切到 auto：立刻放行所有闸门等待者（不再需要人工确认）。
	// 从 auto 切到 copilot/manual：后续动作在边界自然进入等待态，无需提前阻塞。
	var ch chan struct{}
	if old != normalized && normalized == types.LiveModeAuto {
		ch = s.confirmCh
		s.confirmCh = make(chan struct{})
	}
	s.mu.Unlock()

	if ch != nil {
		safeClose(ch)
	}

	if h.audit != nil {
		go func() {
			_ = h.audit.PushEvent(h.ctx, &types.LiveViewAuditEvent{
				ExecutionID: executionID,
				Event:       types.LiveViewEventModeSwitch,
				Operator:    operator,
				Mode:        normalized,
				Detail:      fmt.Sprintf("%s → %s", old, normalized),
				TimestampMS: types.NowMS(),
			})
		}()
	}
	if h.logger != nil {
		h.logger.Info("liveview mode switched",
			logger.String("execution_id", executionID),
			logger.String("from", string(old)),
			logger.String("to", string(normalized)),
			logger.String("operator", operator))
	}
	return nil
}

// Input 注入接管者的输入指令（仅接管者可注入，且必须正在流）。
//
// 返回值语义：
//   - errNotTaker：不是接管者（前端按钮应置灰，这是权限而非故障）；
//   - 其它 error：CDP 注入失败，需要让用户看到（不能静默）。
func (h *Hub) Input(executionID string, cmd *types.LiveInputCommand) error {
	if h == nil || executionID == "" || cmd == nil {
		return errNoSession
	}
	h.mu.Lock()
	s, found := h.sessions[executionID]
	h.mu.Unlock()
	if !found || s == nil || s.page == nil || s.page.Closed() {
		return errNotRunning
	}

	s.mu.Lock()
	taker := s.takeoverBy
	streaming := s.stopStream != nil
	s.mu.Unlock()

	if taker == "" {
		return errNotTaker
	}
	if cmd.Operator != "" && cmd.Operator != taker {
		return errNotTaker
	}
	if !streaming {
		// 未在观看时注入会「看不见却改变页面」，语义上必须拒绝。
		return fmt.Errorf("实时画面未在推流，拒绝注入输入")
	}

	raw := s.page.RawPage()
	if raw == nil {
		return errNotRunning
	}
	if err := browser.DispatchLiveInput(raw, cmd); err != nil {
		return err
	}

	s.mu.Lock()
	s.lastSeen = time.Now()
	s.inputCount++
	s.mu.Unlock()

	h.Touch(executionID)
	return nil
}

// BeforeAction 动作边界闸门（LIVEVIEW-02/03 的核心：接管期间 RPA 动作暂停）。
//
// 调用时机：executor 在每个动作**执行前**调用（与 RUNTIME-03 的 ctx 取消检查点
// 同一位置）。返回 nil 表示放行，返回 error 表示中止（含 ctx 取消）。
//
// 决策规则：
//
//	接管中            → 等 Continue/Release/Stop 才放行（RPA 动作暂停）
//	manual            → 每个动作都等人工继续（完全人工）
//	copilot           → 只在「关键节点」等（pause 动作、带 confirm 标记的动作、
//	                    以及失败重试前的那个动作）
//	auto / 无会话      → 立即放行
//
// 关键节点的定义（copilot）：动作 Type 为 pause，或 Params["confirm"]==true，
// 或 Params["checkpoint"]==true。这是 LIVEVIEW-03「动作级标记 + 失败自动暂停」
// 里的「动作级标记」来源；「失败自动暂停」由 executor 在错误分支调用
// BeforeAction(force=true) 实现。
func (h *Hub) BeforeAction(ctx ctxLike, executionID string, action *types.Action, force bool) error {
	if h == nil || executionID == "" {
		return nil
	}
	h.mu.Lock()
	s, found := h.sessions[executionID]
	h.mu.Unlock()
	if !found || s == nil {
		return nil
	}

	s.mu.Lock()
	mode := s.mode
	taker := s.takeoverBy
	s.lastSeen = time.Now()
	ch := s.confirmCh
	s.mu.Unlock()

	needWait := false
	switch {
	case taker != "":
		needWait = true
	case force:
		// 失败自动暂停：copilot/manual 都要停下来等人看，auto 不打断。
		//
		// W7.1 / 缺陷 B 契约：**force 只应来自「可重试的运行时失败」**
		//（网络超时 / 元素未找到 / 导航失败 / 目标站 5xx 等真正执行失败）。
		//
		// 反例（生产事故）：动作 type="goto" 在 worker 里没有对应 case，
		// executeAction 立即返回 unsupported action type —— 这是脚本配置错误，
		// 重试 100 次也不会成功。当时该动作 retry=2 > 0，于是走到 gate(force=true)
		// → copilot 模式 needWait 恒 true → 永久阻塞 10 分钟后 deadline exceeded，
		// 页面停在 about:blank，screencast 一帧不发（后端 30 分钟零帧请求）。
		//
		// 现在 executor 在调用本闸门之前用 types.IsNonRetryable 拦住了配置错误
		// （fail-fast，不进闸门、不空转重试）。本 case 保留，服务真实失败场景；
		// 如果将来有新的调用点，必须遵守「force 只给可重试失败」这条契约。
		needWait = mode != types.LiveModeAuto
	case mode == types.LiveModeManual:
		needWait = true
	case mode == types.LiveModeCopilot:
		needWait = isCheckpointAction(action)
	}

	if !needWait {
		return nil
	}

	// 需要等待：登记等待者 + 检查/消费「带时效的放行令牌」。
	//
	// 令牌解决的真实竞态：用户点「继续」时引擎可能还在导航/上一个动作里、
	// 尚未进入本闸门。若 Continue 只 close 当前 confirmCh，那个信号会被之后
	// 才拿到的新 ch 错过 —— 用户看到「点了继续却卡住」（端到端用例已复现）。
	// 令牌有 TTL，防止「早点点了一下，很久之后的关键节点被无条件放行」。
	s.mu.Lock()
	s.waiting++
	if s.passTokens > 0 {
		if time.Since(s.passTokenAt) <= passTokenTTL {
			s.passTokens = 0
			s.waiting--
			s.mu.Unlock()
			if h.logger != nil {
				h.logger.Info("liveview gate released by pre-issued token",
					logger.String("execution_id", executionID),
					logger.String("mode", string(mode)))
			}
			return nil
		}
		// 过期令牌作废。
		s.passTokens = 0
	}
	s.mu.Unlock()

	if h.logger != nil {
		h.logger.Info("liveview gate waiting",
			logger.String("execution_id", executionID),
			logger.String("mode", string(mode)),
			logger.String("takeover_by", taker),
			logger.Any("force", force))
	}

	select {
	case <-ch:
		s.mu.Lock()
		if s.waiting > 0 {
			s.waiting--
		}
		s.mu.Unlock()
		return nil
	case <-ctxDone(ctx):
		s.mu.Lock()
		if s.waiting > 0 {
			s.waiting--
		}
		s.mu.Unlock()
		return ctxErr(ctx)
	}
}

// isCheckpointAction copilot 的「关键节点」判定（动作级标记）。
func isCheckpointAction(action *types.Action) bool {
	if action == nil {
		return false
	}
	if action.Type == types.ActionPause {
		return true
	}
	for _, key := range []string{"confirm", "checkpoint", "需要确认"} {
		if v, ok := action.Params[key]; ok {
			if b, isBool := v.(bool); isBool && b {
				return true
			}
			if str, isStr := v.(string); isStr && (str == "true" || str == "1") {
				return true
			}
		}
	}
	return false
}

// Continue 由人工确认放行被闸门阻塞的动作（copilot 确认 / manual 继续）。
//
// reason 记审计（谁放行的、为什么）。
func (h *Hub) Continue(executionID, operator, reason string) error {
	if h == nil || executionID == "" {
		return errNoSession
	}
	h.mu.Lock()
	s, found := h.sessions[executionID]
	h.mu.Unlock()
	if !found || s == nil {
		return errNotRunning
	}

	s.mu.Lock()
	ch := s.confirmCh
	s.confirmCh = make(chan struct{})
	hasWaiter := s.waiting > 0
	if !hasWaiter {
		// 无等待者（引擎还在导航/上一个动作）：预置一枚**带时效**的令牌，
		// 让即将进入闸门的那个动作能立即放行。有等待者时**不发令牌** ——
		// 直接 close ch 唤醒即可，否则令牌会漏给下一个动作（曾导致关键节点
		// 被无确认放行，端到端用例已复现）。
		s.passTokens = 1
		s.passTokenAt = time.Now()
	}
	taker := s.takeoverBy
	s.mu.Unlock()

	if taker != "" && operator != "" && operator != taker {
		return fmt.Errorf("放行失败：当前接管者是 %s", taker)
	}
	safeClose(ch)
	h.Touch(executionID)

	if h.audit != nil {
		ctx, cancel := timeoutCtx(3 * time.Second)
		defer cancel()
		_ = h.audit.PushEvent(ctx, &types.LiveViewAuditEvent{
			ExecutionID: executionID,
			Event:       types.LiveViewEventInput,
			Operator:    operator,
			Detail:      "continue: " + reason,
			TimestampMS: types.NowMS(),
		})
	}
	return nil
}

// Stats 单执行的推流统计快照（nil 安全）。
func (h *Hub) Stats(executionID string) browser.ScreencastStatsSnapshot {
	if h == nil || executionID == "" {
		return browser.ScreencastStatsSnapshot{}
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return browser.ScreencastStatsSnapshot{}
	}
	s.mu.Lock()
	stats := s.stats
	s.mu.Unlock()
	if stats == nil {
		return browser.ScreencastStatsSnapshot{}
	}
	return stats.Snapshot()
}

// Context 当前叠加层上下文快照（前端 canvas 上方 HUD 用）。
func (h *Hub) Context(executionID string) *types.LiveViewContext {
	if h == nil || executionID == "" {
		return nil
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.context
}

// UpdateContext 由 executor 在动作边界刷新叠加层情境（步骤/动作/日志/脱敏变量）。
func (h *Hub) UpdateContext(executionID string, mutate func(*types.LiveViewContext)) {
	if h == nil || executionID == "" || mutate == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.context == nil {
		s.context = types.NewLiveViewContext(executionID)
	}
	if s.context.Mode == "" {
		s.context.Mode = s.mode
	}
	s.context.TakeoverBy = s.takeoverBy
	mutate(s.context)
	if s.context.Mode == "" {
		s.context.Mode = types.LiveModeAuto
	}
}

// timeoutCtx 便捷超时上下文（审计上报用，不阻塞主路径）。
func timeoutCtx(d time.Duration) (ctxLike, func()) {
	return newTimeout(d)
}

package liveview

import (
	"context"
	"sync"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01/02/03/05：单 Worker 的实时画面会话管理。
//
// 职责边界（本包**只**管 Worker 侧）：
//
//	Attach   执行开始时把 page 登记为「可推流」（由 executor 在 Acquire 后调用）
//	Watch    前端点了「看一看」→ 启动 screencast + 帧泵 → 经 FrameSink 上报后端
//	Input    接管者的鼠标/键盘指令 → CDP Input.dispatch*
//	BeforeAction  动作边界闸门（copilot/manual 的暂停确认；接管期间阻塞 RPA）
//	Stop/Detach   停流并释放 screencast，**不关 page、不关 browser**
//
// 硬约束落点：
//   - LIVEVIEW-01：单 Worker 并发推流上限 maxStreams（默认 2），超限返回可读错误。
//   - LIVEVIEW-02：单 page 同时只允许一个接管者；抢占需显式 confirm 参数。
//   - LIVEVIEW-03：模式是控制权档位（auto/copilot/manual），暂停/继续的具体
//     执行状态迁移由**后端**用 RUNTIME-05 的 CanTransition 裁决，Worker 不另造
//     状态枚举（本包不定义任何 ExecutionStatus）。
//   - LIVEVIEW-05：无下行指令超过 IdleTimeout 即停流（默认 30s，不等前端主动 stop
//     才发生 —— 断网/关页签时前端不会再发指令）；stopScreencast 不牵连 page/browser。
// ============================================================================

// FrameSink 帧上行出口（生产实现走后端 HTTP，测试可注入内存 sink 做 fps 实测）。
type FrameSink interface {
	PushFrame(ctx context.Context, frame *types.LiveViewUpstreamFrame) error
}

// AuditSink 审计上行出口（四类事件：watch_start / watch_stop / takeover / release）。
//
// Worker 侧审计是「尽力而为」：失败只记本地日志，不阻断画面流（画面可用性优先）。
type AuditSink interface {
	PushEvent(ctx context.Context, ev *types.LiveViewAuditEvent) error
}

// passTokenTTL 「放行令牌」的有效期。
//
// 令牌是为「用户在引擎尚未进入闸门时点了继续」这一真实时序准备的：引擎从接受
// 命令到进入下一个动作边界，稳态下只需几十~几百毫秒；但机器负载高时（同机
// 并行跑多个真 Chromium 用例、或页面导航慢）可能到数秒甚至十几秒。TTL 取 30s
// 覆盖该窗口；令牌「消费即清零」且上限 1，故不会把后续多个动作无确认放行
// （端到端用例在全量 go test 下曾因 3s TTL 过短而卡住，此处放宽）。
const passTokenTTL = 30 * time.Second

// GateDecision 动作边界闸门的裁决。
type GateDecision int

const (
	// GateProceed 立即放行（auto，或 copilot 未到暂停点）。
	GateProceed GateDecision = iota
	// GateWaitConfirm 需要人工确认后放行（copilot 关键节点 / manual 每步）。
	GateWaitConfirm
)

// Session 一路推流会话（每个 executionID 至多一个）。
type Session struct {
	executionID string
	page        browser.Page
	stats       *browser.ScreencastStats

	mu          sync.Mutex
	stopStream  func()
	framesDone  chan struct{}
	mode        types.LiveMode
	takeoverBy  string
	takeoverAt  time.Time
	lastSeen    time.Time
	lastSignal  time.Time
	context     *types.LiveViewContext
	pushErrs    int
	stopped     bool
	// confirmCh 当前等待者共享的「放行」通道；Continue/Release/Stop 关闭它唤醒等待者。
	confirmCh chan struct{}
	// passTokens 已到达但尚未被闸门消费的「放行令牌」（上限 1）。
	//
	// 解决的真实竞态（Phase 155 端到端用例暴露）：用户点「继续」时引擎可能还
	// 在导航/上一个动作里、尚未进入 BeforeAction。若 Continue 只 close 当前
	// confirmCh，那个信号会被引擎「之后才拿到的新 ch」错过 —— 用户看到点了继续
	// 却永远卡住。令牌语义：无等待者时 Continue 发一枚带时效的令牌，闸门进入
	// 等待前先检查并消费它。
	//
	// **时限**（passTokenTTL）防「早点点了一下，30 秒后的关键节点被无条件放行」：
	// 令牌只在短期内有效，过期即作废。有等待者时**不发令牌**（直接 close ch 唤醒），
	// 否则令牌会漏给下一个动作（同族泄漏，端到端用例曾踩到）。
	passTokens  int
	passTokenAt time.Time
	// waiting 当前阻塞在闸门上的等待者数量（Continue 据此判断「发给令牌」还是「唤醒」）。
	waiting int
	// inputSeq 输入注入计数（审计与「接管者是否真的在操作」判定）。
	inputCount int
}

// Hub 单 Worker 的 liveview 会话表。
type Hub struct {
	cfg     *config.LiveViewConfig
	logger  logger.Logger
	frames  FrameSink
	audit   AuditSink

	mu       sync.Mutex
	sessions map[string]*Session
	closed   bool

	// janitorDone janitor 退出信号（泄漏测试要看到它真的退）。
	janitorDone chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewHub 构造会话表并启动唯一的 janitor（每 Worker 一个，不随 watch 增减）。
func NewHub(cfg *config.LiveViewConfig, sink FrameSink, audit AuditSink, log logger.Logger) *Hub {
	if cfg == nil {
		cfg = &config.LiveViewConfig{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		cfg:         cfg,
		logger:      log,
		frames:      sink,
		audit:       audit,
		sessions:    make(map[string]*Session),
		ctx:         ctx,
		cancel:      cancel,
		janitorDone: make(chan struct{}),
	}
	go h.janitor()
	return h
}

// Enabled 是否允许推流（配置关闭时 Watch 一律拒绝，给可读原因）。
func (h *Hub) Enabled() bool {
	if h == nil || h.cfg == nil {
		return false
	}
	return h.cfg.Enabled
}

// MaxStreams 单 Worker 并发推流上限。
func (h *Hub) MaxStreams() int {
	if h == nil || h.cfg == nil || h.cfg.MaxStreams <= 0 {
		return config.DefaultLiveViewMaxStreams
	}
	return h.cfg.MaxStreams
}

// Attach 登记一个「执行中、可推流」的 page（executor 在 Acquire 成功后调用）。
//
// page 必须是 browser.Page（内部用 RawPage() 取当前活动页签句柄）。
// meta 提供叠加层初始情境（步骤/动作/模式）；nil 时用默认值。
func (h *Hub) Attach(executionID string, page browser.Page, meta *types.LiveViewContext) {
	if h == nil || executionID == "" || page == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	s := &Session{
		executionID: executionID,
		page:        page,
		mode:        types.LiveMode(h.cfg.DefaultMode),
		lastSeen:    time.Now(),
		lastSignal:  time.Now(),
		context:     meta,
		confirmCh:   make(chan struct{}),
		}
	if s.context == nil {
		s.context = types.NewLiveViewContext(executionID)
		s.context.Mode = s.mode
	} else {
		s.context.Mode = s.mode
	}
	// 同 executionID 重复 Attach（重试/子任务复用 ID）时先清掉旧会话的推流。
	if old, ok := h.sessions[executionID]; ok && old.stopStream != nil {
		old.stopStream()
	}
	h.sessions[executionID] = s
	if h.logger != nil {
		h.logger.Debug("liveview attach", logger.String("execution_id", executionID))
	}
}

// Detach 注销执行（executor 在 Release 之前调用）。
//
// 必须停掉该执行的推流，但**不关 page**（page 由 Manager 回收）。
func (h *Hub) Detach(executionID string) {
	if h == nil || executionID == "" {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	if ok {
		delete(h.sessions, executionID)
	}
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	s.stop()
	h.releaseWaiters(s)
	if h.audit != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.audit.PushEvent(ctx, &types.LiveViewAuditEvent{
			ExecutionID: executionID,
			Event:       types.LiveViewEventWatchStop,
			Operator:    "worker",
			Reason:      "execution finished",
			TimestampMS: types.NowMS(),
		})
	}
	if h.logger != nil {
		h.logger.Debug("liveview detach", logger.String("execution_id", executionID))
	}
}

// Watch 开始观看（启动 screencast，占一个推流名额）。
//
// 错误情形（都必须给可读原因，不得静默降级）：
//   - 未启用 → "实时画面未启用"
//   - 执行未在本 Worker 运行 → "执行不在本 Worker 运行（或已结束）"
//   - 已达 maxStreams → "单 Worker 推流路数已达上限 N，请稍后重试"
//   - 已在观看 → 幂等返回 nil（重复 watch 不重复占名额）
func (h *Hub) Watch(executionID, operator string) error {
	if h == nil {
		return errNotEnabled
	}
	if !h.Enabled() {
		return errNotEnabled
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return errNotEnabled
	}
	s, ok := h.sessions[executionID]
	if !ok || s.page == nil || s.page.Closed() {
		h.mu.Unlock()
		return errNotRunning
	}
	if s.stopStream != nil {
		// 重复 watch：只刷新活跃时间。
		s.mu.Lock()
		s.lastSeen = time.Now()
		s.mu.Unlock()
		h.mu.Unlock()
		return nil
	}
	streaming := 0
	for _, other := range h.sessions {
		if other.stopStream != nil {
			streaming++
		}
	}
	if streaming >= h.MaxStreams() {
		h.mu.Unlock()
		return &LimitError{Max: h.MaxStreams()}
	}
	h.mu.Unlock()

	raw := s.page.RawPage()
	if raw == nil {
		return errNotRunning
	}

	opts := browser.ScreencastOptions{
		Quality:         h.cfg.Quality,
		MaxWidth:        h.cfg.MaxWidth,
		MaxHeight:       h.cfg.MaxHeight,
		EveryNthFrame:   h.cfg.EveryNthFrame,
		AckTimeout:      h.cfg.AckTimeout,
		ChannelCapacity: 2,
		Logger:          h.logger,
	}
	framesCh, stats, stopFn, err := browser.StartScreencast(raw, opts)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.stopStream = stopFn
	s.stats = stats
	s.lastSeen = time.Now()
	s.lastSignal = time.Now()
	s.stopped = false
	done := make(chan struct{})
	s.framesDone = done
	s.mu.Unlock()

	go h.pump(s, framesCh, done)

	if h.audit != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.audit.PushEvent(ctx, &types.LiveViewAuditEvent{
			ExecutionID: executionID,
			Event:       types.LiveViewEventWatchStart,
			Operator:    operator,
			TimestampMS: types.NowMS(),
		})
	}
	if h.logger != nil {
		h.logger.Info("liveview watch started",
			logger.String("execution_id", executionID),
			logger.String("operator", operator),
			logger.Int("max_streams", h.MaxStreams()),
			logger.Int("max_width", opts.MaxWidth),
			logger.Int("quality", opts.Quality),
			logger.Int("every_nth_frame", opts.EveryNthFrame))
	}
	return nil
}

// pump 帧泵：读 screencast 帧 → base64 → FrameSink 上报 → ACK（背压闭环）。
//
// **ACK 放在上报之后**：上报慢 = ACK 慢 = Chrome 少发帧，即 LIVEVIEW-01 要求的
// 「ACK 节流」。上报连续失败 3 次停流，避免对端已死却继续烧 CPU 压缩帧。
func (h *Hub) pump(s *Session, frames <-chan *browser.ScreencastFrame, done chan struct{}) {
	defer close(done)
	ctx := h.ctx
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			h.pushFrame(s, f)
			// ACK 必须在处理完（无论成败）之后，这是背压的唯一实现方式。
			f.Ack()
		}
	}
}

// pushFrame 把一帧转成协议体上报后端。
func (h *Hub) pushFrame(s *Session, f *browser.ScreencastFrame) {
	if h.frames == nil || f == nil {
		return
	}
	s.mu.Lock()
	ctx := s.context
	s.mu.Unlock()

	up := &types.LiveViewUpstreamFrame{
		Frame: types.LiveFrame{
			ExecutionID: s.executionID,
			Seq:         f.Seq,
			TimestampMS: f.CapturedAt.UnixMilli(),
			Data:        f.Base64Data(),
			ByteSize:    len(f.Data),
			Width:       f.Width,
			Height:      f.Height,
		},
		Context: ctx,
	}
	if f.Metadata != nil {
		up.Frame.DeviceWidth = f.Metadata.DeviceWidth
		up.Frame.DeviceHeight = f.Metadata.DeviceHeight
		up.Frame.ScrollOffsetX = f.Metadata.ScrollOffsetX
		up.Frame.ScrollOffsetY = f.Metadata.ScrollOffsetY
		up.Frame.PageScaleFactor = f.Metadata.PageScaleFactor
	}

	ctxPush, cancel := context.WithTimeout(h.ctx, 3*time.Second)
	defer cancel()
	if err := h.frames.PushFrame(ctxPush, up); err != nil {
		s.mu.Lock()
		s.pushErrs++
		count := s.pushErrs
		s.mu.Unlock()
		if h.logger != nil {
			h.logger.Warn("liveview 帧上报失败", logger.Err(err), logger.Int("consecutive", count))
		}
		if count >= 3 {
			// 对端不可用：主动停流（LIVEVIEW-05「无执行时停流、不空转烧 CPU」同理）。
			h.Stop(s.executionID, "worker", "帧上报连续失败，自动停流")
		}
		return
	}
	s.mu.Lock()
	s.pushErrs = 0
	s.mu.Unlock()
}

// Stop 停流（前端主动 stop / 空闲超时 / 上报失败 / 执行结束）。
//
// 只 stopScreencast + 退出帧泵，**不关 page、不关 browser**（154 硬语义）。
func (h *Hub) Stop(executionID, operator, reason string) {
	if h == nil || executionID == "" {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	flowing := s.stop()
	if !flowing {
		return
	}
	h.releaseWaiters(s)
	if h.audit != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.audit.PushEvent(ctx, &types.LiveViewAuditEvent{
			ExecutionID: executionID,
			Event:       types.LiveViewEventWatchStop,
			Operator:    operator,
			Reason:      reason,
			TimestampMS: types.NowMS(),
		})
	}
	if h.logger != nil {
		h.logger.Info("liveview stream stopped",
			logger.String("execution_id", executionID),
			logger.String("operator", operator),
			logger.String("reason", reason))
	}
}

// stop 幂等停流，返回「本次调用是否真的停止了正在流的会话」。
func (s *Session) stop() bool {
	s.mu.Lock()
	stopFn := s.stopStream
	done := s.framesDone
	if stopFn == nil {
		s.mu.Unlock()
		return false
	}
	s.stopStream = nil
	s.framesDone = nil
	s.stopped = true
	s.mu.Unlock()

	if stopFn != nil {
		stopFn()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	return true
}

// releaseWaiters 放行所有阻塞在动作边界的等待者（停流/释放接管时不能把人锁死）。
func (h *Hub) releaseWaiters(s *Session) {
	s.mu.Lock()
	ch := s.confirmCh
	s.confirmCh = make(chan struct{})
	s.mu.Unlock()
	if ch != nil {
		// 关闭旧 channel 让所有等待者醒来（继续执行）。
		safeClose(ch)
	}
}

// safeClose 关闭 channel 且不因重复关闭 panic。
func safeClose(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}

// Touch 记录一次下行指令到达（重置空闲计时；LIVEVIEW-05 的停流判定依据）。
func (h *Hub) Touch(executionID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	s.mu.Lock()
	s.lastSeen = time.Now()
	s.lastSignal = time.Now()
	s.mu.Unlock()
}

// janitor 唯一的内务 goroutine：每秒检查空闲超时与接管超时。
func (h *Hub) janitor() {
	defer close(h.janitorDone)
	interval := 500 * time.Millisecond
	if h.cfg.IdleTimeout > 0 && h.cfg.IdleTimeout < 5*time.Second {
		interval = h.cfg.IdleTimeout / 2
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case now := <-ticker.C:
			h.sweep(now)
		}
	}
}

// sweep 一轮内务：空闲超时停流 + 接管超时自动释放。
func (h *Hub) sweep(now time.Time) {
	idle := h.cfg.IdleTimeout
	if idle <= 0 {
		idle = config.DefaultLiveViewIdleTimeout
	}
	takeoverIdle := h.cfg.TakeoverTimeout
	if takeoverIdle <= 0 {
		takeoverIdle = config.DefaultLiveViewTakeoverTimeout
	}

	h.mu.Lock()
	sessions := make([]*Session, 0, len(h.sessions))
	for _, s := range h.sessions {
		sessions = append(sessions, s)
	}
	h.mu.Unlock()

	for _, s := range sessions {
		s.mu.Lock()
		streaming := s.stopStream != nil
		last := s.lastSignal
		taker := s.takeoverBy
		takerAt := s.lastSeen
		s.mu.Unlock()

		// 只看「已建立 WS 通道、但没有任何下行指令」的会话：
		// 未 Watch 的会话（只 Attach 未看）不计空闲，否则执行中的任务会被误停。
		if streaming && !last.IsZero() && now.Sub(last) >= idle {
			h.Stop(s.executionID, "worker", "无下行指令超过空闲时限，自动停流")
			continue
		}
		if taker != "" && !takerAt.IsZero() && now.Sub(takerAt) >= takeoverIdle {
			h.Release(s.executionID, "worker", "接管者长时间无操作，自动释放")
		}
	}
}

// Close 关闭会话表：停所有流并退出 janitor。幂等。
//
// **不关 page、不关 browser** —— 那是 Manager.Close 的职责。
func (h *Hub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	sessions := h.sessions
	h.sessions = make(map[string]*Session)
	h.mu.Unlock()

	h.cancel()
	for _, s := range sessions {
		s.stop()
		h.releaseWaiters(s)
	}
	select {
	case <-h.janitorDone:
	case <-time.After(2 * time.Second):
	}
}

// SessionCount 当前会话数（测试用）。
func (h *Hub) SessionCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.sessions)
}

// StreamingCount 当前正在推流的会话数（测试与容量观测用）。
func (h *Hub) StreamingCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, s := range h.sessions {
		s.mu.Lock()
		if s.stopStream != nil {
			n++
		}
		s.mu.Unlock()
	}
	return n
}

package recording

// ============================================================================
// Phase W6 / REC-01：Worker 侧录制与拾取会话管理。
//
// 归属协议第 2 条：154 冻结的 browser/manager.go 与 page_manager.go **零改动**。
// 本包只**调用**既有契约：
//   - Manager.Acquire(ctx) 取一个 page（不新增 Manager 方法）
//   - Manager.Release(page) 归还（**不关 browser**，154 硬语义）
//   - browser.NewRecorder(page, log) 录制器（recorder.go 零改动）
//   - browser.NewPicker(page, log)  拾取器（picker.go 零改动）
//
// 与 155 liveview 的关系（见 .planning/phases/w6-rec-and-schedules/W6-PLAN.md §2.2）：
//   - **画面帧复用 liveview 通道**：录制会话的画面经 liveview 的帧上行端点
//     与前端 WS 消息类型流转（坐标系、可丢广播、背压全部共用一套）。
//   - **会话生命周期独立**：录制会话有自己的频道与状态机（recording:session:*），
//     因为「录制」不是「一次执行」——没有 execution 行、不进执行状态机、
//     不写执行审计。
//
// 并发约束：单 Worker 同时最多 N 个录制会话（worker.recording.max_sessions），
// 裁决在 Worker 侧（与 155 的 max_streams 同款：后端只展示上限与错误文案）。
// ============================================================================

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// 上行出口（由 communication.APIClient 实现；此处只声明契约，避免包依赖倒置）。
type Upstream interface {
	// ReportStatus 上报会话状态变化（节流调用）。
	ReportStatus(ctx context.Context, st *types.RecordingUpstreamStatus) error
	// ReportResult 上报最终产出（actions + steps）。
	ReportResult(ctx context.Context, res *types.RecordingUpstreamResult) error
	// ReportPick 上报拾取结果（7 级阶梯）。
	ReportPick(ctx context.Context, pick *types.RecordingUpstreamPick) error
}

// 会话状态（Worker 侧内部）。
type sessionState string

const (
	stateStarting  sessionState = "starting"
	stateRecording sessionState = "recording"
	stateStopping  sessionState = "stopping"
	stateStopped   sessionState = "stopped"
	stateAborted   sessionState = "aborted"
)

// recordingSession 单个录制会话。
type recordingSession struct {
	id       string
	operator string
	taskID   string

	page     browser.Page
	recorder *browser.Recorder

	state    sessionState
	lastSeen time.Time
	created  time.Time

	mu       sync.Mutex
	eventCnt int
	// stopOnce 保证 stop 只执行一次（重复 stop 幂等）。
	stopOnce sync.Once
}

// Hub 单 Worker 的录制会话表。
type Hub struct {
	cfg      *config.RecordingConfig
	logger   logger.Logger
	upstream Upstream

	mu       sync.Mutex
	sessions map[string]*recordingSession
	closed   bool

	ctx    context.Context
	cancel context.CancelFunc

	janitorDone chan struct{}
}

// NewHub 构造录制会话表。
func NewHub(cfg *config.RecordingConfig, upstream Upstream, log logger.Logger) *Hub {
	if cfg == nil {
		cfg = &config.RecordingConfig{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		cfg:         cfg,
		logger:      log,
		upstream:    upstream,
		sessions:    make(map[string]*recordingSession),
		ctx:         ctx,
		cancel:      cancel,
		janitorDone: make(chan struct{}),
	}
	go h.janitor()
	return h
}

// MaxSessions 单 Worker 并发录制会话上限。
func (h *Hub) MaxSessions() int {
	if h == nil || h.cfg == nil || h.cfg.MaxSessions <= 0 {
		return config.DefaultRecordingMaxSessions
	}
	return h.cfg.MaxSessions
}

// IdleTimeout 会话闲置超时（无任何指令超过该时限即回收）。
func (h *Hub) IdleTimeout() time.Duration {
	if h == nil || h.cfg == nil || h.cfg.IdleTimeout <= 0 {
		return config.DefaultRecordingIdleTimeout
	}
	return h.cfg.IdleTimeout
}

// SessionCount 当前会话数（观测与测试用）。
func (h *Hub) SessionCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.sessions)
}

// LimitError 并发录制会话超限（可读错误，不静默降级）。
type LimitError struct {
	Max int
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("本机同时进行的录制已达上限 %d 个，请先结束其中一个再试", e.Max)
}

// ErrSessionNotFound 会话不存在（已结束或从未存在）。
var ErrSessionNotFound = fmt.Errorf("录制会话不存在或已结束")

// ErrNoPage 会话存在但页面已失效。
var ErrNoPage = fmt.Errorf("录制页���已关闭，无法继续")

// Start 开始一个录制会话：取页面 → 导航 → 注入录制器 → 开始采集。
//
// targetURL 为空时在当前空白页起步（不导航）。
func (h *Hub) Start(mgr *browser.Manager, cmd *types.RecordingCommand) error {
	if h == nil || mgr == nil {
		return fmt.Errorf("录制管理器未初始化")
	}
	if cmd == nil || cmd.SessionID == "" {
		return fmt.Errorf("录制会话 ID 不能为空")
	}

	h.mu.Lock()
	if _, exists := h.sessions[cmd.SessionID]; exists {
		h.mu.Unlock()
		return fmt.Errorf("录制会话已在进行中")
	}
	active := 0
	for _, s := range h.sessions {
		if s.state != stateStopped && s.state != stateAborted {
			active++
		}
	}
	if active >= h.MaxSessions() {
		h.mu.Unlock()
		return &LimitError{Max: h.MaxSessions()}
	}
	h.mu.Unlock()

	// Acquire：与 155 liveview 同一 Manager，page 名额共享（录制占用一个 page）。
	acquireCtx, cancelAcquire := context.WithTimeout(h.ctx, 30*time.Second)
	page, err := mgr.Acquire(acquireCtx)
	cancelAcquire()
	if err != nil {
		return fmt.Errorf("打开录制页面失败: %w", err)
	}

	// 导航到起始地址（失败不致命：仍可在当前页录制）。
	if cmd.TargetURL != "" {
		if err := page.Goto(cmd.TargetURL); err != nil {
			h.logger.Warn("录制起始页导航失败（继续在当前页录制）",
				logger.String("session_id", cmd.SessionID),
				logger.String("target_url", cmd.TargetURL),
				logger.Err(err))
		}
	}

	rec := browser.NewRecorder(page.RawPage(), h.logger)
	if err := rec.Start(); err != nil {
		// 启动失败必须归还 page，否则名额泄漏。
		mgr.Release(page)
		return fmt.Errorf("启动录制失败: %w", err)
	}

	now := time.Now()
	s := &recordingSession{
		id:       cmd.SessionID,
		operator: cmd.Operator,
		taskID:   cmd.TaskID,
		page:     page,
		recorder: rec,
		state:    stateRecording,
		lastSeen: now,
		created:  now,
	}

	h.mu.Lock()
	h.sessions[cmd.SessionID] = s
	h.mu.Unlock()

	h.logger.Info("录制会话已开始",
		logger.String("session_id", cmd.SessionID),
		logger.String("operator", cmd.Operator),
		logger.Int("active_sessions", active+1),
		logger.Int("max_sessions", h.MaxSessions()))

	h.reportStatus(cmd.SessionID, types.RecordingStatusRecording, 0, "")
	return nil
}

// Stop 停止录制：拉事件 → 产出 actions → 上行 → 释放页面。
//
// 返回产出（同时已上行给后端）。重复 stop 幂等（第二次返回已缓存的产出）。
func (h *Hub) Stop(mgr *browser.Manager, sessionID string) (*types.RecordingUpstreamResult, error) {
	if h == nil {
		return nil, fmt.Errorf("录制管理器未初始化")
	}
	h.mu.Lock()
	s, ok := h.sessions[sessionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return nil, ErrSessionNotFound
	}

	var result *types.RecordingUpstreamResult
	s.stopOnce.Do(func() {
		// 拉全部事件（recorder.Stop 内部会先 Events 再停用监听）。
		events, err := s.recorder.Stop()
		if err != nil {
			h.logger.Warn("拉取录制事件失败（按空产出处理）",
				logger.String("session_id", sessionID), logger.Err(err))
		}
		built := browser.BuildRecording(events)
		result = &types.RecordingUpstreamResult{
			SessionID:  sessionID,
			Actions:    built.Actions,
			Steps:      built.Steps,
			StartURL:   built.StartURL,
			EventCount: built.EventCount,
			Timestamp:  types.RecordingNowMS(),
		}

		s.mu.Lock()
		s.state = stateStopped
		s.eventCnt = built.EventCount
		s.mu.Unlock()

		// 上行产出（权威结果）：后端据此写任务脚本。
		if h.upstream != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.upstream.ReportResult(ctx, result); err != nil {
				h.logger.Error("录制产出上报失败",
					logger.String("session_id", sessionID), logger.Err(err))
			}
		}

		// 释放页面（**不关 browser**，154 硬语义）。
		if mgr != nil && s.page != nil {
			mgr.Release(s.page)
		}
		s.page = nil

		h.logger.Info("录制会话已停止",
			logger.String("session_id", sessionID),
			logger.Int("actions", len(built.Actions)),
			logger.Int("events", built.EventCount))
	})

	if result != nil {
		return result, nil
	}
	// 重复 stop：返回已缓存的结果（幂等）。
	s.mu.Lock()
	cached := &types.RecordingUpstreamResult{
		SessionID:  sessionID,
		EventCount: s.eventCnt,
	}
	s.mu.Unlock()
	return cached, nil
}

// Pick 按坐标拾取元素（7 级阶梯）。
func (h *Hub) Pick(sessionID string, x, y float64) (*types.RecordingUpstreamPick, error) {
	if h == nil {
		return nil, fmt.Errorf("录制管理器未初始化")
	}
	h.mu.Lock()
	s, ok := h.sessions[sessionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return nil, ErrSessionNotFound
	}
	if s.page == nil || s.page.Closed() {
		return nil, ErrNoPage
	}

	picker := browser.NewPicker(s.page.RawPage(), h.logger)
	// PickInFrames 处理 iframe 场景（与 156 的验收一致）。
	elem, err := picker.PickInFrames(x, y)
	if err != nil {
		return nil, fmt.Errorf("拾取元素失败: %w", err)
	}

	pick := &types.RecordingUpstreamPick{
		SessionID:  sessionID,
		Found:      elem.Found,
		Tag:        elem.Tag,
		Type:       elem.Type,
		Text:       elem.Text,
		Recommended: elem.Recommended,
		RecommendedTier:  elem.RecommendedTier,
		RecommendedScore: elem.RecommendedScore,
		Frame:      elem.Frame,
		Timestamp:  types.RecordingNowMS(),
	}
	for _, t := range elem.Tiers {
		pick.Tiers = append(pick.Tiers, types.RecordedTier{
			Tier:     t.Tier,
			Name:     t.Name,
			Selector: t.Selector,
			OK:       t.OK,
			Score:    t.Score,
		})
	}
	if !elem.Found {
		pick.Message = "该位置没有可拾取的元素"
	}

	// 上行拾取结果（前端浮层据此渲染 7 级阶梯）。
	if h.upstream != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.upstream.ReportPick(ctx, pick); err != nil {
			h.logger.Error("拾取结果上报失败",
				logger.String("session_id", sessionID), logger.Err(err))
		}
	}
	return pick, nil
}

// Keepalive 刷新会话闲置计时。
func (h *Hub) Keepalive(sessionID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[sessionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	s.mu.Lock()
	s.lastSeen = time.Now()
	s.mu.Unlock()
}

// Abort 强制中止会话：释放页面（不产出结果）。
func (h *Hub) Abort(mgr *browser.Manager, sessionID, reason string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[sessionID]
	h.mu.Unlock()
	if !ok || s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.state = stateAborted
		s.mu.Unlock()
		// 停掉录制器（移除注入脚本，避免 page 复用后继续录音）。
		if s.recorder != nil {
			if _, err := s.recorder.Stop(); err != nil {
				h.logger.Debug("中止时停止录制器失败（忽略）",
					logger.String("session_id", sessionID), logger.Err(err))
			}
		}
		if mgr != nil && s.page != nil {
			mgr.Release(s.page)
		}
		s.page = nil
		h.logger.Info("录制会话已中止",
			logger.String("session_id", sessionID),
			logger.String("reason", reason))
	})
	h.reportStatus(sessionID, types.RecordingStatusAborted, 0, reason)
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
}

// PollEvents 拉取当前会话已采集的事件数（节流上报，让前端显示「已录 N 个动作」）。
//
// 注意：recorder.Events() 会**清空**缓冲，因此这里不能用它做计数。
// 用 recorder.Recording() 判断是否仍在录制，并按 recorder 内部累积值上报。
func (h *Hub) PollEvents(mgr *browser.Manager) {
	if h == nil {
		return
	}
	h.mu.Lock()
	sessions := make([]*recordingSession, 0, len(h.sessions))
	for _, s := range h.sessions {
		sessions = append(sessions, s)
	}
	h.mu.Unlock()

	for _, s := range sessions {
		s.mu.Lock()
		if s.state != stateRecording {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()
		if s.recorder == nil || !s.recorder.Recording() {
			continue
		}
		// recorder 的事件数在 Stop 时才可知（Events 会清空缓冲），
		// 因此这里只上报「仍在录制」状态，事件数以停止时的结果为准。
		h.reportStatus(s.id, types.RecordingStatusRecording, 0, "")
	}
}

// reportStatus 上报状态（失败只记日志，不阻断录制）。
func (h *Hub) reportStatus(sessionID string, st types.RecordingStatus, eventCount int, message string) {
	if h == nil || h.upstream == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := h.upstream.ReportStatus(ctx, &types.RecordingUpstreamStatus{
		SessionID:  sessionID,
		Status:     st,
		EventCount: eventCount,
		Message:    message,
		Timestamp:  types.RecordingNowMS(),
	})
	if err != nil {
		h.logger.Warn("录制状态上报失败",
			logger.String("session_id", sessionID), logger.Err(err))
	}
}

// janitor 唯一的内务 goroutine：清理闲置超时的会话（孤儿 page 兜底）。
//
// 这是 REC-01 最大的资源风险点：录制会话持有真实 Chromium page，
// 若后端崩溃 / Redis 断连，会话会悬挂并泄漏 page。
func (h *Hub) janitor() {
	defer close(h.janitorDone)
	interval := 5 * time.Second
	idle := h.IdleTimeout()
	if idle > 0 && idle < 10*time.Second {
		interval = idle / 2
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.sweep(time.Now())
		}
	}
}

// sweep 一轮清理：闲置超时的会话强制中止。
func (h *Hub) sweep(now time.Time) {
	h.mu.Lock()
	expired := make([]*recordingSession, 0)
	for _, s := range h.sessions {
		s.mu.Lock()
		stale := now.Sub(s.lastSeen) > h.IdleTimeout()
		s.mu.Unlock()
		if stale {
			expired = append(expired, s)
		}
	}
	h.mu.Unlock()

	for _, s := range expired {
		h.Abort(nil, s.id, "长时间无操作，自动结束")
	}
}

// Close 关闭会话表：中止全部会话并退出 janitor。
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
	ids := make([]string, 0, len(h.sessions))
	for id := range h.sessions {
		ids = append(ids, id)
	}
	h.mu.Unlock()

	for _, id := range ids {
		h.Abort(nil, id, "worker 关闭")
	}
	h.cancel()
	select {
	case <-h.janitorDone:
	case <-time.After(2 * time.Second):
	}
}

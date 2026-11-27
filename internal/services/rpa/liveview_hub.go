package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：实时画面的后端中枢（watcher 注册表 + 三模式
// 状态机 + 审计 + 帧中转）。
//
// 关键设计裁定：
//
//  1. **模式不是执行状态**。auto/copilot/manual 是「控制权档位」，不引入新的
//     RPAExecutionStatus 常量。模式切换需要暂停执行时，走 RUNTIME-05 的
//     running → paused 合法迁移，**复用** CanTransition/CheckTransition
//     （execution_state_machine.go 是唯一真相源，本文件不另造校验）。
//  2. **帧可丢，状态不可丢**。帧走 NoticeHub 的 liveview 专用可丢路径（队列满
//     丢帧不丢连接）；叠加层状态走正常路径。
//  3. **watcher 以「执行读权限」为唯一准入**：复用 ops:rpa:execution:list
//     + 数据范围（handler 层挂中间件，本文件只做注册表与权限复核）。
//  4. 断开/失权即停：watcher 心跳（keepalive）超过 IdleTimeout 移除；帧推送前
//     复核读权限（缓存 5s，避免每帧查库）。
// ============================================================================

// LiveViewMode 协作模式（与 rpa-worker 的 types.LiveMode 同值，跨进程约定）。
type LiveViewMode string

const (
	LiveViewModeAuto    LiveViewMode = "auto"
	LiveViewModeCopilot LiveViewMode = "copilot"
	LiveViewModeManual  LiveViewMode = "manual"
)

// IsValidLiveViewMode 模式取值校验（未知值拒绝，不静默降级）。
func IsValidLiveViewMode(m LiveViewMode) bool {
	switch m {
	case LiveViewModeAuto, LiveViewModeCopilot, LiveViewModeManual:
		return true
	}
	return false
}

// NormalizeLiveViewMode 空值归一为默认 copilot。
func NormalizeLiveViewMode(m LiveViewMode) (LiveViewMode, bool) {
	if m == "" {
		return LiveViewModeCopilot, true
	}
	return m, IsValidLiveViewMode(m)
}

// LiveViewEventKind 审计事件类型（与 worker 侧 types.LiveViewEvent 同值）。
type LiveViewEventKind string

const (
	LiveViewEventWatchStart LiveViewEventKind = "watch_start"
	LiveViewEventWatchStop  LiveViewEventKind = "watch_stop"
	LiveViewEventTakeover   LiveViewEventKind = "takeover"
	LiveViewEventRelease    LiveViewEventKind = "release"
	LiveViewEventModeSwitch LiveViewEventKind = "mode_switch"
	LiveViewEventInput      LiveViewEventKind = "input"
)

// IsValidLiveViewEventKind 白名单（拒绝未知事件，避免审计表被污染）。
func IsValidLiveViewEventKind(e LiveViewEventKind) bool {
	switch e {
	case LiveViewEventWatchStart, LiveViewEventWatchStop,
		LiveViewEventTakeover, LiveViewEventRelease,
		LiveViewEventModeSwitch, LiveViewEventInput:
		return true
	}
	return false
}

// auditActionFor 事件 → 既有 AuditAction 取值（复用 sys_rpa_audit_logs 的 Action 列，
// 不新增枚举：start/stop 是「看」的边界，update 覆盖接管/释放/模式/输入）。
func auditActionFor(e LiveViewEventKind) rpamodels.AuditAction {
	switch e {
	case LiveViewEventWatchStart:
		return rpamodels.AuditActionStart
	case LiveViewEventWatchStop:
		return rpamodels.AuditActionStop
	default:
		return rpamodels.AuditActionUpdate
	}
}

// LiveViewFrame Worker 上行的帧（字段与 rpa-worker types.LiveFrame 一一对应）。
type LiveViewFrame struct {
	ExecutionID     string  `json:"executionId"`
	Seq             int64   `json:"seq"`
	TimestampMS     int64   `json:"timestamp"`
	Data            string  `json:"data"`
	ByteSize        int     `json:"byteSize"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	DeviceWidth     float64 `json:"deviceWidth"`
	DeviceHeight    float64 `json:"deviceHeight"`
	ScrollOffsetX   float64 `json:"scrollOffsetX"`
	ScrollOffsetY   float64 `json:"scrollOffsetY"`
	PageScaleFactor float64 `json:"pageScaleFactor"`
}

// LiveViewContextSnapshot 叠加层情境（Worker 侧已脱敏）。
type LiveViewContextSnapshot struct {
	ExecutionID string       `json:"executionId"`
	TaskName    string       `json:"taskName"`
	WorkerID    string       `json:"workerId"`
	WorkerName  string       `json:"workerName"`
	Step        int          `json:"step"`
	TotalSteps  int          `json:"totalSteps"`
	ActionType  string       `json:"actionType"`
	ActionDesc  string       `json:"actionDesc"`
	Status      string       `json:"status"`
	Mode        LiveViewMode `json:"mode"`
	TakeoverBy  string       `json:"takeoverBy,omitempty"`
	// TakeoverByID 接管者用户 ID（W7 / UI-FIX-02）
	//
	// takeoverBy 是展示名（nickname/username），会随用户改昵称、随 SM2/HS256
	// 分支取值口径不同而漂移 —— 用字符串当控制权判据曾让「已接管但遮罩不消失」
	// 在生产复现。真正的控制权判据是 userID：前端拿自己的 user.id 比对即可，
	// 不依赖任何展示名。takeoverBy 仅用于 HUD 展示。
	//
	// omitempty 是安全的：前端每次收到 context 都**整体替换**（不是合并），
	// 释放后字段缺失即清空，不会残留上一次的接管者。
	TakeoverByID string            `json:"takeoverById,omitempty"`
	TakeoverAtMS int64             `json:"takeoverAt,omitempty"`
	LastLog      string            `json:"lastLog,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	StartedAtMS  int64             `json:"startedAt"`
}

// LiveViewUpstreamFrame Worker 上报体（帧 + 情境）。
type LiveViewUpstreamFrame struct {
	Frame   LiveViewFrame            `json:"frame"`
	Context *LiveViewContextSnapshot `json:"context,omitempty"`
}

// LiveViewInputCommand 输入注入（与 worker 侧同构）。
type LiveViewInputCommand struct {
	ExecutionID string  `json:"executionId"`
	Kind        string  `json:"kind"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Button      string  `json:"button,omitempty"`
	ClickCount  int     `json:"clickCount,omitempty"`
	Modifiers   int     `json:"modifiers,omitempty"`
	DeltaX      float64 `json:"deltaX,omitempty"`
	DeltaY      float64 `json:"deltaY,omitempty"`
	Key         string  `json:"key,omitempty"`
	Code        string  `json:"code,omitempty"`
	Text        string  `json:"text,omitempty"`
	VirtualKey  int     `json:"virtualKey,omitempty"`
	Operator    string  `json:"operator,omitempty"`
	TimestampMS int64   `json:"timestamp,omitempty"`
}

// LiveViewCommand 下行指令（publish 到 execution:liveview:{id}）。
type LiveViewCommand struct {
	Action      string                `json:"action"`
	ExecutionID string                `json:"executionId"`
	Operator    string                `json:"operator,omitempty"`
	Mode        LiveViewMode          `json:"mode,omitempty"`
	Input       *LiveViewInputCommand `json:"input,omitempty"`
	Force       bool                  `json:"force,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	TimestampMS int64                 `json:"timestamp,omitempty"`
}

// 下行指令动作名（与 worker 侧一致）。
const (
	liveCmdWatch     = "watch"
	liveCmdStop      = "stop"
	liveCmdInput     = "input"
	liveCmdTakeover  = "takeover"
	liveCmdRelease   = "release"
	liveCmdMode      = "mode"
	liveCmdKeepalive = "keepalive"
	liveCmdContinue  = "continue"
	liveCmdAbort     = "abort"
)

// liveViewChannelFmt 下行频道格式（与 worker 一致）。
const liveViewChannelFmt = "execution:liveview:%s"

// LiveViewWatchState watch 端点响应。
type LiveViewWatchState struct {
	ExecutionID string       `json:"executionId"`
	Mode        LiveViewMode `json:"mode"`
	MaxStreams  int          `json:"maxStreams"`
	Streaming   int          `json:"streaming"`
	TakeoverBy  string       `json:"takeoverBy,omitempty"`
	// TakeoverByID 接管者用户 ID（前端据此判 isController，见 LiveViewContextSnapshot）
	TakeoverByID string `json:"takeoverById,omitempty"`
}

// LiveViewStateResponse state 端点响应（叠加层 + 统计）。
type LiveViewStateResponse struct {
	Context  *LiveViewContextSnapshot `json:"context"`
	Watchers int                      `json:"watchers"`
}

// LiveViewStats 推流统计快照（后端侧观测：watcher 数、帧数、丢弃数）。
type LiveViewStats struct {
	Watchers    int   `json:"watchers"`
	Frames      int64 `json:"frames"`
	Dropped     int64 `json:"dropped"`
	LastFrameMS int64 `json:"lastFrameMs"`
	// W7.1 / 缺陷 C：已推流但超过阈值仍无帧 → 页面没有可画内容。
	//
	// 前端据此把「等待画面…」换成「当前页面暂无画面（可能停在空白页或正在
	// 加载）」，并给「刷新状态」按钮。判据全部在后端（帧数 + 开流时长），
	// 不依赖 Worker 回传任何新字段 —— 少一个跨进程契约面。
	EmptyPageHint bool `json:"emptyPageHint"`
	// EmptyPageWaitingMS 已等待的毫秒数（给用户看「等了多久」）。
	EmptyPageWaitingMS int64 `json:"emptyPageWaitingMs"`
}

// watcher 一个观看者（用户级，不按连接，因为 WS 是每用户一条）。
type watcher struct {
	userID   string
	lastSeen time.Time
	joinedAt time.Time
	lastPerm time.Time
	permOK   bool
}

// LiveViewHub 后端侧实时画面中枢（单例，挂在 ServiceGroup）。
type LiveViewHub struct {
	db        *gorm.DB
	cfg       *config.Config
	noticeHub *websocket.NoticeHub
	cache     cache.Cache
	log       interface{ Info(string, ...interface{}) }

	mu          sync.Mutex
	sessions    map[string]*liveViewSession
	closed      bool
	janitorDone chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

// liveViewSession 单执行的观看态。
type liveViewSession struct {
	executionID string
	taskName    string
	mode        LiveViewMode
	takeoverBy  string
	// takeoverByID 接管者用户 ID（控制权权威判据，W7 / UI-FIX-02）
	takeoverByID string
	takeoverAt   time.Time
	watchers     map[string]*watcher
	context      *LiveViewContextSnapshot
	lastFrameAt  int64
	frames       int64
	dropped      int64
	// W7.1 / 缺陷 C：本次推流开始时刻（首个 watcher 触发 watch 时记下）。
	// 与 frames 一起构成「已推流但 N 秒零帧」的判据；watcher 全走即会话销毁，
	// 所以不需要重置逻辑。
	streamStartedAt time.Time
	// emptyPageNotified 「暂无画面」提示已推过一次（避免 janitor 每轮重推）。
	emptyPageNotified bool
	// started 用于「无执行时停流」判定：Worker 上报帧即刷新。
	lastWorkerSignal time.Time
}

// NewLiveViewHub 构造中枢并启动唯一 janitor。
func NewLiveViewHub(db *gorm.DB, cfg *config.Config, noticeHub *websocket.NoticeHub, cacheInstance cache.Cache, log interface{ Info(string, ...interface{}) }) *LiveViewHub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &LiveViewHub{
		db:          db,
		cfg:         cfg,
		noticeHub:   noticeHub,
		cache:       cacheInstance,
		log:         log,
		sessions:    make(map[string]*liveViewSession),
		ctx:         ctx,
		cancel:      cancel,
		janitorDone: make(chan struct{}),
	}
	go h.janitor()
	return h
}

// enabled 是否启用（配置 rpa.liveview.enabled，未配置默认启用）。
func (h *LiveViewHub) enabled() bool {
	if h == nil || h.cfg == nil {
		return false
	}
	return h.cfg.RPA.LiveView.Enabled
}

// maxStreams 单 Worker 推流上限（后端只做展示与错误文案，真正限流在 Worker）。
func (h *LiveViewHub) maxStreams() int {
	if h == nil || h.cfg == nil || h.cfg.RPA.LiveView.MaxStreams <= 0 {
		return config.DefaultRPALiveViewMaxStreams
	}
	return h.cfg.RPA.LiveView.MaxStreams
}

// idleTimeout watcher 心跳超时（默认 30s，LIVEVIEW-05「断开 N 秒即停流」）。
func (h *LiveViewHub) idleTimeout() time.Duration {
	if h == nil || h.cfg == nil || h.cfg.RPA.LiveView.IdleTimeout <= 0 {
		return config.DefaultRPALiveViewIdleTimeout
	}
	return h.cfg.RPA.LiveView.IdleTimeout
}

// ensureSession 取或建会话。
func (h *LiveViewHub) ensureSession(executionID string) *liveViewSession {
	s, ok := h.sessions[executionID]
	if ok {
		return s
	}
	s = &liveViewSession{
		executionID: executionID,
		mode:        LiveViewModeCopilot,
		watchers:    make(map[string]*watcher),
		context: &LiveViewContextSnapshot{
			ExecutionID: executionID,
			Mode:        LiveViewModeCopilot,
		},
	}
	h.sessions[executionID] = s
	return s
}

// Watch 登记一个观看者并向 Worker 下发 watch 指令。
//
// 返回可读错误（未启用 / Worker 未运行由 Worker 侧拒绝并记审计）。
func (h *LiveViewHub) Watch(ctx context.Context, executionID, userID, userName string) (*LiveViewWatchState, error) {
	if h == nil || !h.enabled() {
		return nil, fmt.Errorf("实时画面未启用（rpa.liveview.enabled=false）")
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, fmt.Errorf("实时画面中枢已关闭")
	}
	s := h.ensureSession(executionID)
	now := time.Now()
	if w, ok := s.watchers[userID]; ok {
		w.lastSeen = now
		w.lastPerm = now
		w.permOK = true
	} else {
		s.watchers[userID] = &watcher{
			userID:   userID,
			lastSeen: now,
			joinedAt: now,
			lastPerm: now,
			permOK:   true,
		}
	}
	state := &LiveViewWatchState{
		ExecutionID:  executionID,
		Mode:         s.mode,
		MaxStreams:   h.maxStreams(),
		Streaming:    len(s.watchers),
		TakeoverBy:   s.takeoverBy,
		TakeoverByID: s.takeoverByID,
	}
	// W7.1 / 缺陷 C：记录推流起点（首个 watcher 触发的 watch）。重复 watch 不重置
	// —— 「已经等了多久」就该从第一次开流算起，否则每次刷新都归零。
	if s.streamStartedAt.IsZero() {
		s.streamStartedAt = now
	}
	h.mu.Unlock()

	// 通知 Worker 开流（Worker 侧做真正的并发上限裁决）。
	if err := h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdWatch,
		ExecutionID: executionID,
		Operator:    userName,
		TimestampMS: now.UnixMilli(),
	}); err != nil {
		// publish 失败不阻断观看（Worker 可能还没订阅），记审计由 handler 负责。
		h.logf("liveview watch publish failed: %v", err)
	}
	return state, nil
}

// Keepalive 刷新观看者心跳（LIVEVIEW-05：断流判定依据）。
func (h *LiveViewHub) Keepalive(ctx context.Context, executionID, userID string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	var notify bool
	if ok {
		if w, found := s.watchers[userID]; found {
			w.lastSeen = time.Now()
			notify = true
		}
	}
	h.mu.Unlock()
	if notify {
		_ = h.publish(ctx, &LiveViewCommand{
			Action:      liveCmdKeepalive,
			ExecutionID: executionID,
			TimestampMS: time.Now().UnixMilli(),
		})
	}
}

// Stop 停止观看（单用户）。最后一个 watcher 离开时才真正下发 stop 指令。
func (h *LiveViewHub) Stop(ctx context.Context, executionID, userID, reason string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	lastOne := false
	if ok {
		delete(s.watchers, userID)
		lastOne = len(s.watchers) == 0
		if lastOne {
			delete(h.sessions, executionID)
		}
	}
	h.mu.Unlock()
	if !ok {
		return
	}
	if lastOne {
		_ = h.publish(ctx, &LiveViewCommand{
			Action:      liveCmdStop,
			ExecutionID: executionID,
			Operator:    userID,
			Reason:      reason,
			TimestampMS: time.Now().UnixMilli(),
		})
	}
}

// DropWatcher 无条件下线一个 watcher（权限失效 / 心跳超时 / 连接断开）。
//
// 返回「是否因此移除了整个执行会话」（供 handler 决定是否下发 stop）。
func (h *LiveViewHub) DropWatcher(executionID, userID, reason string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	lastOne := false
	if ok {
		if _, found := s.watchers[userID]; found {
			delete(s.watchers, userID)
		}
		lastOne = len(s.watchers) == 0
		if lastOne {
			delete(h.sessions, executionID)
		}
	}
	h.mu.Unlock()
	if lastOne && ok {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.publish(ctx, &LiveViewCommand{
			Action:      liveCmdStop,
			ExecutionID: executionID,
			Operator:    "system",
			Reason:      reason,
			TimestampMS: time.Now().UnixMilli(),
		})
	}
	return lastOne
}

// Takeover 接管（单执行单接管者）。
//
// force=false 且已被他人占用 → 返回可读冲突错误（handler 转 409），
// 前端二次确认后带 force=true 重试（LIVEVIEW-05「抢占需显式确认」）。
//
// W7 / UI-FIX-02：控制权以 **userID** 为权威（takeoverByID），userName 只做展示。
// 历史实现只比 userName 字符串，而 userName 来源（SM2 token 无 nickname）与前端
// 取值口径（user.nickname）不一致时会「接管成功但遮罩不消失」。
func (h *LiveViewHub) Takeover(ctx context.Context, executionID, userID, userName string, force bool) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("该执行当前无观看会话，请先打开实时画面")
	}
	// 冲突判定：优先比 userID（权威）；仅当旧会话没有 userID（历史数据）时回落比展示名。
	if s.takeoverByID != "" && s.takeoverByID != userID && !force {
		cur := s.takeoverBy
		h.mu.Unlock()
		return &LiveViewConflictError{By: cur}
	}
	if s.takeoverByID == "" && s.takeoverBy != "" && s.takeoverBy != userName && !force {
		cur := s.takeoverBy
		h.mu.Unlock()
		return &LiveViewConflictError{By: cur}
	}
	s.takeoverBy = userName
	s.takeoverByID = userID
	s.takeoverAt = time.Now()
	if s.context != nil {
		s.context.TakeoverBy = userName
		s.context.TakeoverByID = userID
		s.context.TakeoverAtMS = s.takeoverAt.UnixMilli()
	}
	h.mu.Unlock()

	// publish 尽力而为：本地控制权已转移成功，Worker 若尚未订阅（或 Redis 不可达）
	// 只记日志、不回滚控制权（回滚会让用户看到「接管成功」却又被撤销）。
	if err := h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdTakeover,
		ExecutionID: executionID,
		Operator:    userName,
		Force:       force,
		TimestampMS: time.Now().UnixMilli(),
	}); err != nil {
		h.logf("liveview takeover publish failed (execution=%s): %v", executionID, err)
	}
	return nil
}

// Release 释放接管。
//
// W7 / UI-FIX-02：签名改为以 userID 权威比对（userName 只用于错误文案）。
// 历史上 handler 传的是 operatorName(c)（展示名），释放失败会误伤其他用户。
func (h *LiveViewHub) Release(ctx context.Context, executionID, userID, userName, reason string) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("该执行当前无观看会话")
	}
	cur := s.takeoverBy
	curID := s.takeoverByID
	if cur == "" && curID == "" {
		h.mu.Unlock()
		return nil // 幂等
	}
	// 权威判据是 userID；userID 为空（历史数据/内部调用）时才回落展示名。
	if userID != "" && curID != "" && userID != curID {
		h.mu.Unlock()
		return fmt.Errorf("释放失败：当前接管者是 %s", cur)
	}
	if curID == "" && userName != "" && userName != cur {
		h.mu.Unlock()
		return fmt.Errorf("释放失败：当前接管者是 %s", cur)
	}
	s.takeoverBy = ""
	s.takeoverByID = ""
	s.takeoverAt = time.Time{}
	if s.context != nil {
		s.context.TakeoverBy = ""
		s.context.TakeoverByID = ""
		s.context.TakeoverAtMS = 0
	}
	h.mu.Unlock()

	// publish 尽力而为（同 Takeover 的裁定）：本地已释放，失败只记日志。
	if err := h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdRelease,
		ExecutionID: executionID,
		Operator:    userName,
		Reason:      reason,
		TimestampMS: time.Now().UnixMilli(),
	}); err != nil {
		h.logf("liveview release publish failed (execution=%s): %v", executionID, err)
	}
	return nil
}

// LimitError 并发推流超限（LIVEVIEW-01/05：可读错误，不静默降级）。
//
// 后端侧只用于「本进程 watcher 数已满」的展示；真正的推流名额裁决在 Worker
// （screencast 绑定在 Worker 的 page 上）。此处保留同型错误便于 handler 统一转 429。
type LimitError struct {
	Max int
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("实时画面观看路数已达上限 %d，请稍后重试或等待他人结束观看", e.Max)
}

// LiveViewConflictError 接管冲突（handler 转 409，message 可直接展示）。
type LiveViewConflictError struct {
	By string
}

func (e *LiveViewConflictError) Error() string {
	return fmt.Sprintf("该画面已被 %s 接管；如需抢占请显式确认后重试", e.By)
}

// Input 注入输入（只有接管者可注入）。
//
// W7 / UI-FIX-02：签名改为 (userID, userName)。判据以 userID 为权威 —— 与
// Takeover 同一口径，避免「我是接管者但输入被 403」的字符串漂移问题。
func (h *LiveViewHub) Input(ctx context.Context, executionID, userID, userName string, cmd *LiveViewInputCommand) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	if cmd == nil {
		return fmt.Errorf("输入指令为空")
	}
	h.mu.Lock()
	s, ok := h.sessions[executionID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("该执行当前无观看会话，请先打开实时画面")
	}
	taker := s.takeoverBy
	takerID := s.takeoverByID
	h.mu.Unlock()
	// 权威：userID 对 userID；历史会话没有 userID 时才回落展示名比对。
	if takerID != "" {
		if userID == "" || takerID != userID {
			return &LiveViewForbiddenError{Reason: "只有当前接管者可以注入输入"}
		}
	} else if taker == "" || taker != userName {
		return &LiveViewForbiddenError{Reason: "只有当前接管者可以注入输入"}
	}

	cmd.ExecutionID = executionID
	cmd.Operator = userName
	if cmd.TimestampMS == 0 {
		cmd.TimestampMS = time.Now().UnixMilli()
	}
	return h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdInput,
		ExecutionID: executionID,
		Operator:    userName,
		Input:       cmd,
		TimestampMS: cmd.TimestampMS,
	})
}

// LiveViewForbiddenError 非接管者注入（handler 转 403）。
type LiveViewForbiddenError struct {
	Reason string
}

func (e *LiveViewForbiddenError) Error() string { return e.Reason }

// SetMode 切换模式。
//
// **本文件不判断 running→paused 合法性**：状态迁移由
// execution_state_machine.go 的 CanTransition/CheckTransition 裁决（RUNTIME-05
// 唯一真相源）。这里只做模式取值校验 + 下发 + 审计。
func (h *LiveViewHub) SetMode(ctx context.Context, executionID, userName string, mode LiveViewMode) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	normalized, ok := NormalizeLiveViewMode(mode)
	if !ok {
		return fmt.Errorf("非法协作模式 %q（允许 auto|copilot|manual）", mode)
	}
	h.mu.Lock()
	s, found := h.sessions[executionID]
	if !found {
		h.mu.Unlock()
		return fmt.Errorf("该执行当前无观看会话")
	}
	old := s.mode
	s.mode = normalized
	if s.context != nil {
		s.context.Mode = normalized
	}
	h.mu.Unlock()

	if old != normalized {
		if err := h.publish(ctx, &LiveViewCommand{
			Action:      liveCmdMode,
			ExecutionID: executionID,
			Operator:    userName,
			Mode:        normalized,
			TimestampMS: time.Now().UnixMilli(),
		}); err != nil {
			h.logf("liveview mode publish failed: %v", err)
		}
	}
	return nil
}

// Continue 放行被闸门阻塞的动作（copilot 确认 / manual 继续 / 释放接管后断点续跑）。
//
// 若执行当前是 paused，先按 RUNTIME-05 状态机做 paused → running 迁移（不合法
// 直接拒绝，绝不静默改状态）；再下发 continue。
func (h *LiveViewHub) Continue(ctx context.Context, executionID, userName, reason string) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	if h.db != nil {
		var exec rpamodels.Execution
		if err := h.db.WithContext(ctx).Where("id = ?", executionID).First(&exec).Error; err == nil {
			from := rpamodels.RPAExecutionStatus(exec.Status)
			if from == rpamodels.RPAExecutionStatusPaused {
				// RUNTIME-05 唯一真相源：非法迁移（终态等）直接拒绝。
				if _, err := CheckTransition(from, rpamodels.RPAExecutionStatusRunning); err != nil {
					return fmt.Errorf("无法继续执行: %w", err)
				}
				if err := h.db.WithContext(ctx).Model(&rpamodels.Execution{}).
					Where("id = ?", executionID).
					Update("status", string(rpamodels.RPAExecutionStatusRunning)).Error; err != nil {
					return fmt.Errorf("恢复执行状态失败: %w", err)
				}
			}
		}
	}
	// publish 尽力而为：DB 侧的 paused→running 已按 RUNTIME-05 生效，
	// Worker 若未订阅则它自己的 idle 闸门仍会放行下一个动作边界。
	if err := h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdContinue,
		ExecutionID: executionID,
		Operator:    userName,
		Reason:      reason,
		TimestampMS: time.Now().UnixMilli(),
	}); err != nil {
		h.logf("liveview continue publish failed (execution=%s): %v", executionID, err)
	}
	return nil
}

// Abort 人工中止（走 RUNTIME-03 的取消管道 → cancelled 终态）。
func (h *LiveViewHub) Abort(ctx context.Context, executionID, userName, reason string) error {
	if h == nil || !h.enabled() {
		return fmt.Errorf("实时画面未启用")
	}
	return h.publish(ctx, &LiveViewCommand{
		Action:      liveCmdAbort,
		ExecutionID: executionID,
		Operator:    userName,
		Reason:      reason,
		TimestampMS: time.Now().UnixMilli(),
	})
}

// IngestFrame Worker 上行帧：转前端可丢广播 + 刷新情境。
//
// 返回实际推送的 watcher 数（0 表示没人看，Worker 侧会因空闲停流）。
func (h *LiveViewHub) IngestFrame(ctx context.Context, up *LiveViewUpstreamFrame) int {
	if h == nil || up == nil || !h.enabled() {
		return 0
	}
	execID := up.Frame.ExecutionID
	h.mu.Lock()
	s, ok := h.sessions[execID]
	if !ok {
		h.mu.Unlock()
		return 0
	}
	// 权限复核（每 5s 一次，避免每帧查库）：失权 watcher 立刻移除。
	h.mu.Unlock()
	recipients := h.revalidateWatchers(ctx, execID)

	h.mu.Lock()
	s, ok = h.sessions[execID]
	if !ok {
		h.mu.Unlock()
		return 0
	}
	s.frames++
	s.lastFrameAt = up.Frame.TimestampMS
	s.lastWorkerSignal = time.Now()
	if up.Context != nil {
		// 模式与接管者以后端会话表为准（Worker 回传的是它自己的视图，
		// 后端是控制权转移的权威：避免两端漂移）。
		ctxSnap := *up.Context
		ctxSnap.Mode = s.mode
		ctxSnap.TakeoverBy = s.takeoverBy
		ctxSnap.TakeoverByID = s.takeoverByID
		s.context = &ctxSnap
	}
	// W7 / UI-FIX-02：帧里带的 context 必须是**后端权威值**，不能直接转发
	// up.Context —— Worker 的 takeoverBy 只是它收到的 operator 展示名，没有
	// userID；历史 bug 就是「帧里的 context」与「state 端点」不同步，前端
	// 遮罩判定反复抖动。这里在锁内拷一份快照，出锁后不再碰 s.context
	// （否则与 Takeover/Release 并发写构成数据竞争）。
	var ctxForFrame *LiveViewContextSnapshot
	if s.context != nil {
		snap := *s.context
		ctxForFrame = &snap
	}
	h.mu.Unlock()

	if len(recipients) == 0 {
		return 0
	}

	// 帧体按协议 §1/§6 走 liveview 专用可丢广播。
	payload, err := json.Marshal(map[string]interface{}{
		"frame":   up.Frame,
		"context": ctxForFrame,
	})
	if err != nil {
		h.logf("liveview frame marshal failed: %v", err)
		return 0
	}
	dropped := h.noticeHub.PublishLiveToUsers(recipients, execID, websocket.MessageTypeRPALiveFrame, string(payload))
	if dropped > 0 {
		h.mu.Lock()
		if s2, ok2 := h.sessions[execID]; ok2 {
			s2.dropped += int64(dropped)
		}
		h.mu.Unlock()
	}
	return len(recipients)
}

// IngestContext Worker 上行情境（无帧，仅叠加层）：走不可丢广播。
//
// Worker 的帧上报已带 context（IngestFrame 内处理）。本方法留给「只更新情境」
// 的场景（如执行结束时的最终态），保证叠加层最终一致。
func (h *LiveViewHub) IngestContext(ctx context.Context, execID string, snap *LiveViewContextSnapshot) {
	if h == nil || !h.enabled() || snap == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.sessions[execID]
	if !ok {
		h.mu.Unlock()
		return
	}
	kn := *snap
	kn.Mode = s.mode
	kn.TakeoverBy = s.takeoverBy
	kn.TakeoverByID = s.takeoverByID
	s.context = &kn
	uids := make([]string, 0, len(s.watchers))
	for uid := range s.watchers {
		uids = append(uids, uid)
	}
	h.mu.Unlock()
	if len(uids) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]interface{}{"context": kn})
	if err != nil {
		return
	}
	h.noticeHub.PublishLiveStateToUsers(uids, execID, websocket.MessageTypeRPALiveState, string(payload))
}

// revalidateWatchers 复核每个 watcher 是否仍持有该执行的读权限。
//
// 「复用执行读权限」的落点：不是再查一次 sys_user_role（代价高、语义窄），
// 而是以「执行所属任务对当前用户是否可见」为准 —— 这里用 execution 的
// task 归属 + 用户能否读该任务（task_service 的可见性由数据范围中间件保证）。
// 由于本 phase 的 handler 已挂 ops:rpa:execution:list，**权限码层面已通过**；
// 数据范围复核通过 reader 回调注入，缺省时视为通过（不阻塞帧流）。
func (h *LiveViewHub) revalidateWatchers(ctx context.Context, executionID string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[executionID]
	if !ok {
		return nil
	}
	now := time.Now()
	valid := make([]string, 0, len(s.watchers))
	for uid, w := range s.watchers {
		// 心跳超时：LIVEVIEW-05「用户断开后 N 秒内停流」。
		if now.Sub(w.lastSeen) > h.idleTimeout() {
			delete(s.watchers, uid)
			continue
		}
		valid = append(valid, uid)
	}
	if len(s.watchers) == 0 {
		// 没人看了：删会话（Worker 侧由 idle 停流）。
		delete(h.sessions, executionID)
	}
	return valid
}

// State 取叠加层情境 + watcher 数。
func (h *LiveViewHub) State(executionID string) *LiveViewStateResponse {
	if h == nil {
		return &LiveViewStateResponse{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[executionID]
	if !ok {
		return &LiveViewStateResponse{Context: &LiveViewContextSnapshot{ExecutionID: executionID, Mode: LiveViewModeCopilot}}
	}
	out := &LiveViewStateResponse{Watchers: len(s.watchers)}
	if s.context != nil {
		snap := *s.context
		out.Context = &snap
	}
	return out
}

// Stats 推流统计（观测用，也可供 FLEET-03 带宽核对）。
func (h *LiveViewHub) Stats(executionID string) LiveViewStats {
	if h == nil {
		return LiveViewStats{}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[executionID]
	if !ok {
		return LiveViewStats{}
	}
	out := LiveViewStats{
		Watchers:    len(s.watchers),
		Frames:      s.frames,
		Dropped:     s.dropped,
		LastFrameMS: s.lastFrameAt,
	}
	// W7.1 / 缺陷 C 判据：已开流超过 EmptyPageHintAfter 且一帧未收到。
	//
	// 为什么判据在后端而不在 Worker：Worker 与后端是两条独立部署，Worker 侧的
	// 「零帧」结论要经过一次 HTTP 上报才能到前端，而帧本身一次也没到过 ——
	// 由「帧数 + 开流时长」这两个后端已持有的事实判定，恢复路径最短。
	if !s.streamStartedAt.IsZero() && s.frames == 0 {
		waited := time.Since(s.streamStartedAt)
		if waited >= emptyPageHintAfter {
			out.EmptyPageHint = true
			out.EmptyPageWaitingMS = waited.Milliseconds()
		}
	}
	return out
}

// emptyPageHintAfter 「已推流但零帧」的判定阈值（W7.1 / 缺陷 C）。
//
// 5s 与 Worker 侧 ZeroFrameWarnAfter 同值：两侧看到同一条事实，日志与 UI
// 不会出现「Worker 说空页面、前端还在等待画面」的分裂。
const emptyPageHintAfter = 5 * time.Second

// janitor 唯一内务 goroutine：清心跳超时的 watcher（不按执行起 goroutine）。
func (h *LiveViewHub) janitor() {
	defer close(h.janitorDone)
	interval := 2 * time.Second
	if h.idleTimeout() > 0 && h.idleTimeout() < 10*time.Second {
		interval = h.idleTimeout() / 2
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.sweep()
		}
	}
}

// sweep 一轮：移除心跳超时的 watcher；空会话即删；必要时推「暂无画面」提示。
func (h *LiveViewHub) sweep() {
	// 锁内只做状态维护与收集，**不**在持锁期间做 IO（publish 会走 cache/redis）。
	h.mu.Lock()
	now := time.Now()
	idle := h.idleTimeout()
	var hints []*emptyPageHint
	for execID, s := range h.sessions {
		for uid, w := range s.watchers {
			if now.Sub(w.lastSeen) > idle {
				delete(s.watchers, uid)
			}
		}
		if len(s.watchers) == 0 {
			delete(h.sessions, execID)
			continue
		}
		// W7.1 / 缺陷 C：已推流但零帧超过阈值 → 给前端推一条带 emptyPageHint
		// 的 rpa_liveview_state（不可丢广播），让「等待画面…」及时换成
		// 「当前页面暂无画面」。
		//
		// 只在「首次越线」时推一次（emptyPageNotified 去重），否则每 2s 一条
		// 会变成噪声；用户点「刷新状态」时前端会重新 GET state，那里不限次数。
		if !s.emptyPageNotified && !s.streamStartedAt.IsZero() && s.frames == 0 &&
			now.Sub(s.streamStartedAt) >= emptyPageHintAfter {
			s.emptyPageNotified = true
			hint := &emptyPageHint{
				execID:    execID,
				waitingMS: now.Sub(s.streamStartedAt).Milliseconds(),
			}
			for uid := range s.watchers {
				hint.uids = append(hint.uids, uid)
			}
			if len(hint.uids) > 0 {
				hints = append(hints, hint)
			}
		}
	}
	h.mu.Unlock()

	for _, hint := range hints {
		payload, err := json.Marshal(map[string]interface{}{
			"context": map[string]interface{}{
				"executionId": hint.execID,
			},
			"emptyPageHint":      true,
			"emptyPageWaitingMs": hint.waitingMS,
		})
		if err != nil {
			h.logf("liveview empty page hint marshal failed (execution=%s): %v", hint.execID, err)
			continue
		}
		h.noticeHub.PublishLiveStateToUsers(hint.uids, hint.execID,
			websocket.MessageTypeRPALiveState, string(payload))
		h.logf("liveview empty page hint pushed (execution=%s, waited=%dms)",
			hint.execID, hint.waitingMS)
	}
}

// emptyPageHint 待推送的「零帧」提示（锁内收集、出锁发送）。
type emptyPageHint struct {
	execID    string
	waitingMS int64
	uids      []string
}

// Close 关闭中枢（停 janitor，清会话表；不下发 stop —— Worker 会因 idle 自行停流）。
func (h *LiveViewHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	sessions := make([]string, 0, len(h.sessions))
	for id := range h.sessions {
		sessions = append(sessions, id)
	}
	h.sessions = make(map[string]*liveViewSession)
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sort.Strings(sessions)
	for _, id := range sessions {
		_ = h.publish(ctx, &LiveViewCommand{
			Action:      liveCmdStop,
			ExecutionID: id,
			Operator:    "system",
			Reason:      "backend shutting down",
			TimestampMS: time.Now().UnixMilli(),
		})
	}
	h.cancel()
	select {
	case <-h.janitorDone:
	case <-time.After(2 * time.Second):
	}
}

// WatcherCount 当前 watcher 总数（测试与观测）。
func (h *LiveViewHub) WatcherCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, s := range h.sessions {
		n += len(s.watchers)
	}
	return n
}

// SessionCount 当前会话数（测试与观测）。
func (h *LiveViewHub) SessionCount() int {
	if h == nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.sessions)
}

// publish 下发一条指令到 Worker（Redis Pub/Sub，尽力而为）。
func (h *LiveViewHub) publish(ctx context.Context, cmd *LiveViewCommand) error {
	if h == nil || cmd == nil {
		return fmt.Errorf("指令为空")
	}
	if h.cache == nil {
		return fmt.Errorf("cache 未注入，无法下发实时画面指令")
	}
	channel := fmt.Sprintf(liveViewChannelFmt, cmd.ExecutionID)
	return publishRedisChannel(ctx, h.cache, channel, cmd)
}

// logf 便捷日志（logger 接口窄化为 printf，避免本文件依赖具体日志实现）。
func (h *LiveViewHub) logf(format string, args ...interface{}) {
	if h == nil || h.log == nil {
		return
	}
	h.log.Info(fmt.Sprintf(format, args...))
}

// Audit 写实时画面审计（LIVEVIEW-05 四类事件必写）。
//
// 落既有 sys_rpa_audit_logs（不新建表）：ResourceType=execution，
// ResourceID=executionId，Action 由 auditActionFor 映射，NewValue 存事件细节。
func (h *LiveViewHub) Audit(ctx context.Context, executionID, operatorID, operatorName string, event LiveViewEventKind, detail map[string]interface{}) error {
	if h == nil || h.db == nil {
		return fmt.Errorf("审计失败：数据库未注入")
	}
	if !IsValidLiveViewEventKind(event) {
		return fmt.Errorf("审计失败：未知事件类型 %q", event)
	}
	if detail == nil {
		detail = map[string]interface{}{}
	}
	detail["event"] = string(event)

	newValue := make(map[string]interface{}, len(detail)+1)
	for k, v := range detail {
		newValue[k] = v
	}
	newValue["timestamp"] = time.Now().UnixMilli()

	rec := &rpamodels.AuditLog{
		ResourceType: rpamodels.ResourceTypeExecution,
		ResourceID:   executionID,
		Action:       auditActionFor(event),
		NewValue:     newValue,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Result:       rpamodels.AuditResultSuccess,
	}
	if err := h.db.WithContext(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("写实时画面审计失败: %w", err)
	}
	return nil
}

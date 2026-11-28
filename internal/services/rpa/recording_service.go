package rpa

// ============================================================================
// Phase W6 / REC-01：录制与拾取会话的后端中枢。
//
// 架构裁定（详见 .planning/phases/w6-rec-and-schedules/W6-PLAN.md §2）：
//
//	下行指令  后端 --Redis PubSub recording:session:{id}--> Worker
//	上行产出  Worker --HTTP POST /rpa/workers/recording/*--> 后端
//	画面帧    Worker --HTTP POST /rpa/workers/liveview/frame--> 后端
//	          --NoticeHub WS rpa_liveview_frame--> 前端（**复用 155 帧通道**）
//
// 为什么帧复用 liveview 而会话独立：
//   - 帧的语义（Worker 浏览器画面 + viewport 坐标系 + 可丢广播 + ACK 背压）
//     与 155 liveview **完全相同**。另立一套必然产生第二套坐标系约定 ——
//     这是最典型的双源漂移温床，且 155 已经把最硬的两个问题解决了
//     （队列满丢帧不 close 连接；帧 ACK 即背压）。
//   - 但录制会话**不是一次执行**：没有 execution 行、不进执行状态机、
//     不写执行审计。故会话生命周期另立频道与状态机。
//
// 状态机（后端侧）：
//
//	created --start--> starting --(Worker 报 recording)--> recording
//	   |                     |                                  |
//	   |                     +-------------- stop --------------+
//	   |                                                    --> stopped（终态）
//	   +-- 闲置超时 / 前端离开 --> expired（终态，清理 Worker 侧页面）
//
// 终态语义：stopped 保留产出（可查），expired 直接丢弃（没有产出可留）。
// ============================================================================

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// ============================================================================
// 后端侧镜像类型（与 rpa-worker internal/types/recording.go 同构）
//
// 后端是独立 module，不能 import rpa-worker，故在此声明镜像。
// 镜像的同源性由 internal/api/v1/rpa/recording_protocol_test.go 锁定
// （直接读 worker 源文件抽取字段名做双向比对，缺一即红）。
// ============================================================================

// RecordingSessionState 录制会话状态（后端侧）。
type RecordingSessionState string

const (
	// RecordingSessionCreated 已创建（已下发 start 指令，等待 Worker 就绪）。
	RecordingSessionCreated RecordingSessionState = "created"
	// RecordingSessionStarting Worker 正在准备（开页面、注入录制器）。
	RecordingSessionStarting RecordingSessionState = "starting"
	// RecordingSessionRecording 正在录制。
	RecordingSessionRecording RecordingSessionState = "recording"
	// RecordingSessionStopped 已停止，产出可用（终态）。
	RecordingSessionStopped RecordingSessionState = "stopped"
	// RecordingSessionFailed 出错（终态）。
	RecordingSessionFailed RecordingSessionState = "failed"
	// RecordingSessionExpired 闲置超时回收（终态，产出丢弃）。
	RecordingSessionExpired RecordingSessionState = "expired"
	// RecordingSessionAborted 被强制中止（终态）。
	RecordingSessionAborted RecordingSessionState = "aborted"
)

// RecordingLimitError 并发录制会话已达上限（handler 据此转 429，与 liveview 的
// LimitError 同款语义：可读文案 + 建议等他人结束）。
type RecordingLimitError struct {
	Max int
}

func (e *RecordingLimitError) Error() string {
	return fmt.Sprintf("并发录制会话已达上限 %d，请稍后重试或等他人结束录制", e.Max)
}

// IsTerminalRecordingState 是否终态（终态不再接受状态变更）。
func IsTerminalRecordingState(s RecordingSessionState) bool {
	switch s {
	case RecordingSessionStopped, RecordingSessionFailed, RecordingSessionExpired,
		RecordingSessionAborted:
		return true
	}
	return false
}

// RecordingActionName 中文名（前端可见文案，零技术术语）。
func RecordingActionName(s RecordingSessionState) string {
	switch s {
	case RecordingSessionCreated:
		return "正在准备"
	case RecordingSessionStarting:
		return "正在准备"
	case RecordingSessionRecording:
		return "正在录制"
	case RecordingSessionStopped:
		return "已完成"
	case RecordingSessionFailed:
		return "出错了"
	case RecordingSessionExpired:
		return "已结束"
	case RecordingSessionAborted:
		return "已中止"
	}
	return "未知状态"
}

// RecordingTier 一级选择器候选（7 级阶梯的其中一级）。
type RecordingTier struct {
	Tier     int    `json:"tier"`
	Name     string `json:"name"`
	Selector string `json:"selector"`
	OK       bool   `json:"ok"`
	Score    int    `json:"score"`
}

// RecordingPickResult 拾取结果（7 级阶梯）。
type RecordingPickResult struct {
	SessionID        string          `json:"sessionId"`
	Found            bool            `json:"found"`
	Tag              string          `json:"tag,omitempty"`
	Type             string          `json:"type,omitempty"`
	Text             string          `json:"text,omitempty"`
	Tiers            []RecordingTier `json:"tiers,omitempty"`
	Recommended      string          `json:"recommended,omitempty"`
	RecommendedTier  int             `json:"recommendedTier,omitempty"`
	RecommendedScore int             `json:"recommendedScore,omitempty"`
	Frame            string          `json:"frame,omitempty"`
	Message          string          `json:"message,omitempty"`
}

// RecordingResultData 录制产出（后端权威持有）。
type RecordingResultData struct {
	// Actions 动作序列（与 worker types.Action 同构：type/selector/params/...）。
	Actions []map[string]interface{} `json:"actions"`
	// Steps 每个动作的中文描述（与 Actions 等长）。
	Steps []string `json:"steps"`
	// StartURL 录制起始地址。
	StartURL string `json:"startUrl,omitempty"`
	// EventCount 原始事件数（诊断用）。
	EventCount int `json:"eventCount"`
}

// RecordingStatusUpdate Worker → 后端的状态上报体。
type RecordingStatusUpdate struct {
	SessionID  string `json:"sessionId"`
	WorkerID   string `json:"workerId,omitempty"`
	Status     string `json:"status"`
	EventCount int    `json:"eventCount"`
	Message    string `json:"message,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
}

// RecordingPickUpdate Worker → 后端的拾取结果上报体。
type RecordingPickUpdate struct {
	SessionID        string          `json:"sessionId"`
	WorkerID         string          `json:"workerId,omitempty"`
	Found            bool            `json:"found"`
	Tag              string          `json:"tag,omitempty"`
	Type             string          `json:"type,omitempty"`
	Text             string          `json:"text,omitempty"`
	Tiers            []RecordingTier `json:"tiers,omitempty"`
	Recommended      string          `json:"recommended,omitempty"`
	RecommendedTier  int             `json:"recommendedTier,omitempty"`
	RecommendedScore int             `json:"recommendedScore,omitempty"`
	Frame            string          `json:"frame,omitempty"`
	Message          string          `json:"message,omitempty"`
	Timestamp        int64           `json:"timestamp,omitempty"`
}

// RecordingResultUpdate Worker → 后端的最终产出上报体。
type RecordingResultUpdate struct {
	SessionID  string                   `json:"sessionId"`
	WorkerID   string                   `json:"workerId,omitempty"`
	Actions    []map[string]interface{} `json:"actions"`
	Steps      []string                 `json:"steps"`
	StartURL   string                   `json:"startUrl,omitempty"`
	EventCount int                      `json:"eventCount"`
	Timestamp  int64                    `json:"timestamp,omitempty"`
}

// ============================================================================
// 会话表
// ============================================================================

// RecordingSession 单个录制会话。
type RecordingSession struct {
	ID       string `json:"id"`
	TaskID   string `json:"taskId,omitempty"`
	WorkerID string `json:"workerId,omitempty"`
	// OperatorID / OperatorName 操作者（审计与归属）。
	OperatorID   string `json:"operatorId,omitempty"`
	OperatorName string `json:"operatorName,omitempty"`
	// TargetURL 起始地址（可空）。
	TargetURL string `json:"targetUrl,omitempty"`

	State RecordingSessionState `json:"state"`
	// Message 状态附带的可读说明（如出错原因）。
	Message string `json:"message,omitempty"`
	// EventCount 已采集事件数。
	EventCount int `json:"eventCount"`

	// Result 录制产出（stop 后可用）。
	Result *RecordingResultData `json:"result,omitempty"`
	// LastPick 最近一次拾取结果（前端浮层消费）。
	LastPick *RecordingPickResult `json:"lastPick,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// LastSeen 最近一次活动（闲置超时判定）。
	//
	// **必须导出并带 json tag**：私有字段不会被 json.Marshal 序列化，回读后
	// 恒为零值（1970），sweep 会立刻把刚恢复的会话判成闲置——这正是 PR #114
	// 加回读后暴露的次生缺陷（回读路径与老化判据必须同时成立才自洽）。
	LastSeen time.Time `json:"lastSeen"`
}

// RecordingHub 后端侧录制会话中枢（单例，挂在 ServiceGroup）。
type RecordingHub struct {
	cache     cache.Cache
	noticeHub *websocket.NoticeHub
	log       interface{ Info(string, ...interface{}) }
	// db 任务脚本落库用（产出直接写回关联任务的 script 列）。
	db *gorm.DB
	// taskSvc 任务服务（写回脚本走既有 Update 路径，保持字段语义一致）。
	taskSvc TaskService
	// cfg RPA 配置（max_sessions / idle_timeout 来源；nil 时回落默认值）。
	cfg *config.Config

	mu       sync.Mutex
	sessions map[string]*RecordingSession
	closed   bool

	ctx         context.Context
	cancel      context.CancelFunc
	janitorDone chan struct{}
}

// Redis 键（后端侧会话态暂存，跨进程重启不丢）。
const (
	// recordingSessionKeyFmt 单会话的 Redis 键。
	recordingSessionKeyFmt = "rpa:recording:session:%s"
	// recordingSessionTTL 会话在 Redis 的存活时长（超过即视为过期）。
	recordingSessionTTL = 2 * time.Hour
)

// RecordingHubOption 装配选项。
type RecordingHubOption struct {
	Cache     cache.Cache
	NoticeHub *websocket.NoticeHub
	Log       interface{ Info(string, ...interface{}) }
	// DB 任务脚本落库用（可选：nil 时不落库）。
	DB *gorm.DB
	// TaskService 任务服务（可选：nil 时不落库）。
	TaskService TaskService
	// Config RPA 配置（可选：nil 时回落默认值）。max_sessions / idle_timeout
	// 必须与 rpa-worker 的 worker.recording 同值，否则会出现「后端说还能开、
	// Worker 说已满」的口径分裂。
	Config *config.Config
}

// NewRecordingHub 构造录制中枢并启动唯一 janitor。
func NewRecordingHub(opt RecordingHubOption) *RecordingHub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &RecordingHub{
		cache:       opt.Cache,
		noticeHub:   opt.NoticeHub,
		log:         opt.Log,
		db:          opt.DB,
		taskSvc:     opt.TaskService,
		cfg:         opt.Config,
		sessions:    make(map[string]*RecordingSession),
		ctx:         ctx,
		cancel:      cancel,
		janitorDone: make(chan struct{}),
	}
	go h.janitor()
	return h
}

// MaxSessions 单 Worker 并发录制会话上限（后端侧只做展示与前置校验，
// 真正的裁决在 Worker 侧 —— 与 155 的 max_streams 分工一致）。
func (h *RecordingHub) MaxSessions() int {
	if h == nil || h.cfg == nil || h.cfg.RPA.Recording.MaxSessions <= 0 {
		return config.DefaultRPARecordingMaxSessions
	}
	return h.cfg.RPA.Recording.MaxSessions
}

// IdleTimeout 后端侧会话闲置超时（默认 5 分钟，与 worker 侧同值）。
func (h *RecordingHub) IdleTimeout() time.Duration {
	if h == nil || h.cfg == nil || h.cfg.RPA.Recording.IdleTimeout <= 0 {
		return config.DefaultRPARecordingIdleTimeout
	}
	return h.cfg.RPA.Recording.IdleTimeout
}

// StartSession 创建并启动一个录制会话：登记会话 → 下发 start 指令给 Worker。
//
// 参数：
//   - taskID：关联任务（可选，用于产出直接写回该任务脚本）
//   - workerID：目标 Worker（空 = 由 Worker 自行拒收非本机指令）
//   - targetURL：起始页面地址（可空）
func (h *RecordingHub) StartSession(ctx context.Context, sessionID, taskID, workerID, operatorID, operatorName, targetURL string) (*RecordingSession, error) {
	if h == nil {
		return nil, fmt.Errorf("录制服务未初始化")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("录制会话 ID 不能为空")
	}

	now := time.Now()
	s := &RecordingSession{
		ID:           sessionID,
		TaskID:       taskID,
		WorkerID:     workerID,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		TargetURL:    targetURL,
		State:        RecordingSessionCreated,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastSeen:     now,
	}

	h.mu.Lock()
	if _, exists := h.sessions[sessionID]; exists {
		h.mu.Unlock()
		return nil, fmt.Errorf("录制会话已在进行中")
	}
	// 并发录制上限前置校验（REC-01 分工：真正的 page 名额裁决在 Worker 侧，
	// 后端在这里拦一份是为了「用户一点就拿到可读 429」，而不是等 Worker 拒绝后
	// 前端干等超时。两端阈值同源 config，见 RPARecordingConfig 注释）。
	active := 0
	for _, existing := range h.sessions {
		if !IsTerminalRecordingState(existing.State) {
			active++
		}
	}
	if active >= h.MaxSessions() {
		h.mu.Unlock()
		return nil, &RecordingLimitError{Max: h.MaxSessions()}
	}
	h.sessions[sessionID] = s
	h.mu.Unlock()

	h.persist(ctx, s)

	// 下发 start 指令（尽力而为：publish 失败不回滚会话登记，
	// 由 janitor 在闲置超时后清理 —— 与 155 的 watch publish 同款裁定）。
	cmd := map[string]interface{}{
		"action":    "start",
		"sessionId": sessionID,
		"workerId":  workerID,
		"operator":  operatorName,
		"targetUrl": targetURL,
		"taskId":    taskID,
		"timestamp": now.UnixMilli(),
	}
	if err := h.publish(ctx, recordingSessionChannel(sessionID), cmd); err != nil {
		h.logf("录制开始指令下发失败 (session=%s): %v", sessionID, err)
		s.State = RecordingSessionFailed
		s.Message = "无法通知执行机开始录制，请稍后重试"
		h.mu.Lock()
		s.UpdatedAt = time.Now()
		h.mu.Unlock()
		h.persist(ctx, s)
		return s, fmt.Errorf("开始录制失败：%w", err)
	}

	s.State = RecordingSessionStarting
	h.mu.Lock()
	s.UpdatedAt = time.Now()
	h.mu.Unlock()
	h.persist(ctx, s)

	h.logf("录制会话已创建 (session=%s, worker=%s)", sessionID, workerID)
	return s, nil
}

// StopSession 停止录制：下发 stop 指令给 Worker。
//
// 产出由 Worker 通过 HTTP 上行（ReportResult），本方法只负责下发指令。
func (h *RecordingHub) StopSession(ctx context.Context, sessionID, operatorName string) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	s, ok := h.get(sessionID)
	if !ok {
		return fmt.Errorf("录制会话不存在或已结束")
	}
	if IsTerminalRecordingState(s.State) {
		// 终态：幂等返回（不报错，让前端能安全重复点「停止」）。
		return nil
	}

	cmd := map[string]interface{}{
		"action":    "stop",
		"sessionId": sessionID,
		"operator":  operatorName,
		"timestamp": time.Now().UnixMilli(),
	}
	if err := h.publish(ctx, recordingSessionChannel(sessionID), cmd); err != nil {
		return fmt.Errorf("停止录制失败：%w", err)
	}
	return nil
}

// PickElement 请求拾取元素：下发 pick 指令给 Worker。
//
// 结果由 Worker 通过 HTTP 上行（ReportPick），前端轮询 GetSession 取 lastPick。
func (h *RecordingHub) PickElement(ctx context.Context, sessionID string, x, y float64) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	s, ok := h.get(sessionID)
	if !ok {
		return fmt.Errorf("录制会话不存在或已结束")
	}
	if IsTerminalRecordingState(s.State) {
		return fmt.Errorf("录制已结束，无法拾取元素")
	}

	cmd := map[string]interface{}{
		"action":    "pick",
		"sessionId": sessionID,
		"pickX":     x,
		"pickY":     y,
		"timestamp": time.Now().UnixMilli(),
	}
	if err := h.publish(ctx, recordingSessionChannel(sessionID), cmd); err != nil {
		return fmt.Errorf("拾取元素失败：%w", err)
	}
	return nil
}

// Keepalive 刷新会话闲置计时（前端心跳）。
func (h *RecordingHub) Keepalive(ctx context.Context, sessionID string) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	h.mu.Lock()
	s, ok := h.getMut(sessionID)
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("录制会话不存在或已结束")
	}
	s.LastSeen = time.Now()
	s.UpdatedAt = s.LastSeen
	h.mu.Unlock()

	cmd := map[string]interface{}{
		"action":    "keepalive",
		"sessionId": sessionID,
		"timestamp": time.Now().UnixMilli(),
	}
	// keepalive 下发失败不阻断（Worker 侧也有自己的兜底清理）。
	if err := h.publish(ctx, recordingSessionChannel(sessionID), cmd); err != nil {
		h.logf("录制心跳下发失败 (session=%s): %v", sessionID, err)
	}
	return nil
}

// AbortSession 强制中止会话（前端离开 / 出错）。
func (h *RecordingHub) AbortSession(ctx context.Context, sessionID, reason string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	s, ok := h.getMut(sessionID)
	if !ok {
		h.mu.Unlock()
		return
	}
	if IsTerminalRecordingState(s.State) {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()
	cmd := map[string]interface{}{
		"action":    "abort",
		"sessionId": sessionID,
		"reason":    reason,
		"timestamp": time.Now().UnixMilli(),
	}
	if err := h.publish(ctx, recordingSessionChannel(sessionID), cmd); err != nil {
		h.logf("录制中止指令下发失败 (session=%s): %v", sessionID, err)
	}
	h.mu.Lock()
	s.State = RecordingSessionAborted
	s.Message = reason
	s.UpdatedAt = time.Now()
	h.mu.Unlock()
	h.persist(ctx, s)
}

// ReportStatus 接收 Worker 的状态上报。
func (h *RecordingHub) ReportStatus(ctx context.Context, up *RecordingStatusUpdate) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	if up == nil || up.SessionID == "" {
		return fmt.Errorf("状态上报缺少 sessionId")
	}
	// Worker 上报是「执行机还在推进这个会话」的最强证据：
	// 后端重启后内存表为空、或被老化判成 expired，都应当据此恢复会话，
	// 而不是回一句「会话不存在」让 Worker 的上报永久丢失。
	if _, ok := h.GetSessionForRecovery(up.SessionID); !ok {
		if _, recovered := h.recoverFromPersist(ctx, up.SessionID); !recovered {
			return fmt.Errorf("录制会话不存在或已结束")
		}
	}

	h.mu.Lock()
	s, ok := h.getMut(up.SessionID)
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("录制会话不存在或已结束")
	}
	// expired 是**后端凭 LastSeen 老化推断**的，不是 Worker 声明的终态。
	// 有活跃上报 = 执行机还在跑 → 推翻推断，把会话救回来。
	// （stopped/failed/aborted 才是真终态，仍按「迟到上报忽略」处理。）
	if s.State == RecordingSessionExpired && isActiveRecordingStatus(up.Status) {
		h.logf("录制会话被 Worker 上报唤醒（expired → %s）(session=%s)", up.Status, up.SessionID)
		s.State = RecordingSessionRecording
		s.Message = ""
	}
	if IsTerminalRecordingState(s.State) && up.Status != string(RecordingSessionStopped) {
		h.mu.Unlock()
		return nil // 真终态不再变更（Worker 迟到的上报）
	}
	switch up.Status {
	case string(RecordingSessionRecording):
		s.State = RecordingSessionRecording
	case string(RecordingSessionStarting):
		s.State = RecordingSessionStarting
	case string(RecordingSessionStopped):
		s.State = RecordingSessionStopped
	case string(RecordingSessionFailed):
		s.State = RecordingSessionFailed
		s.Message = up.Message
	case string(RecordingSessionAborted):
		s.State = RecordingSessionExpired
	}
	if up.EventCount > 0 {
		s.EventCount = up.EventCount
	}
	if up.Message != "" && s.Message == "" {
		s.Message = up.Message
	}
	s.UpdatedAt = time.Now()
	s.LastSeen = s.UpdatedAt
	h.mu.Unlock()
	h.persist(ctx, s)
	return nil
}

// ReportPick 接收 Worker 的拾取结果。
func (h *RecordingHub) ReportPick(ctx context.Context, up *RecordingPickUpdate) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	if up == nil || up.SessionID == "" {
		return fmt.Errorf("拾取上报缺少 sessionId")
	}
	h.mu.Lock()
	s, ok := h.getMut(up.SessionID)
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("录制会话不存在或已结束")
	}
	h.mu.Unlock()

	pick := &RecordingPickResult{
		SessionID:        up.SessionID,
		Found:            up.Found,
		Tag:              up.Tag,
		Type:             up.Type,
		Text:             up.Text,
		Tiers:            up.Tiers,
		Recommended:      up.Recommended,
		RecommendedTier:  up.RecommendedTier,
		RecommendedScore: up.RecommendedScore,
		Frame:            up.Frame,
		Message:          up.Message,
	}
	h.mu.Lock()
	s.LastPick = pick
	s.UpdatedAt = time.Now()
	s.LastSeen = s.UpdatedAt
	h.mu.Unlock()
	h.persist(ctx, s)
	return nil
}

// ReportResult 接收 Worker 的最终产出（**权威结果**）。
func (h *RecordingHub) ReportResult(ctx context.Context, up *RecordingResultUpdate) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	if up == nil || up.SessionID == "" {
		return fmt.Errorf("产出上报缺少 sessionId")
	}
	h.mu.Lock()
	s, ok := h.getMut(up.SessionID)
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("录制会话不存在或已结束")
	}
	s.Result = &RecordingResultData{
		Actions:    up.Actions,
		Steps:      up.Steps,
		StartURL:   up.StartURL,
		EventCount: up.EventCount,
	}
	s.State = RecordingSessionStopped
	s.EventCount = up.EventCount
	s.UpdatedAt = time.Now()
	s.LastSeen = s.UpdatedAt
	taskID := s.TaskID
	h.mu.Unlock()
	h.persist(ctx, s)
	h.logf("录制产出已接收 (session=%s, actions=%d)", up.SessionID, len(up.Actions))

	// 产出直接写回关联任务的脚本（REC-01 验收要求：
	// 「停止 → 产出 actions JSON 存为该 task 的脚本」）。
	// 落库失败**不回滚**产出（产出已在会话里，前端仍能取到并填进向导）。
	if taskID != "" {
		if err := h.saveActionsToTask(ctx, taskID, up.Actions); err != nil {
			h.logf("录制产出写回任务脚本失败 (task=%s): %v", taskID, err)
		}
	}
	return nil
}

// saveActionsToTask 把录制产出写为任务的脚本。
//
// 走 TaskService.Update（既有更新路径），保证字段语义与其它更新入口一致。
func (h *RecordingHub) saveActionsToTask(ctx context.Context, taskID string, actions []map[string]interface{}) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	if taskID == "" {
		return fmt.Errorf("任务 ID 为空")
	}
	if len(actions) == 0 {
		return fmt.Errorf("录制产出为空，未写入任务")
	}
	if h.taskSvc == nil {
		return fmt.Errorf("任务服务未注入")
	}
	// []map[string]interface{} → []interface{}（UpdateTaskRequest.Script 的形状）。
	script := make([]interface{}, 0, len(actions))
	for _, a := range actions {
		script = append(script, a)
	}
	return h.taskSvc.Update(ctx, &UpdateTaskRequest{
		ID:     taskID,
		Script: script,
	}, "")
}

// StartSessionForTest 仅供跨包测试注入一个「录制中」会话（生产不使用）。
//
// 动机：api 包无法直接写 RecordingHub 的私有 map，而上限分流是用户能感知的
// 行为，必须在 HTTP 层断言。用受控入口而非放宽结构体可见性。
func (h *RecordingHub) StartSessionForTest(sessionID string) (*RecordingSession, error) {
	if h == nil {
		return nil, fmt.Errorf("录制服务未初始化")
	}
	now := time.Now()
	s := &RecordingSession{
		ID:        sessionID,
		State:     RecordingSessionRecording,
		CreatedAt: now,
		UpdatedAt: now,
		LastSeen:  now,
	}
	h.mu.Lock()
	h.sessions[sessionID] = s
	h.mu.Unlock()
	return s, nil
}

// GetSession 取会话快照。
func (h *RecordingHub) GetSession(sessionID string) (*RecordingSession, bool) {
	return h.get(sessionID)
}

// dropPersisted 删除 Redis 里的会话快照。
//
// 与 loadPersisted 配对：既然 get() 会从 Redis 回读，那么**真正**要丢弃会话的
// 路径（sweep 回收、Close 关停）就必须同时删键，否则刚清掉的会话会被下一次
// get() 从 Redis 复活（W6 实测：sweep 后 GetSession 仍能命中）。
func (h *RecordingHub) dropPersisted(ctx context.Context, sessionID string) {
	if h == nil || h.cache == nil || sessionID == "" {
		return
	}
	if err := h.cache.Delete(ctx, fmt.Sprintf(recordingSessionKeyFmt, sessionID)); err != nil {
		h.logf("录制会话 Redis 清理失败 (session=%s): %v", sessionID, err)
	}
}

// GetSessionForRecovery 查内存表是否存在该会话（**不触发** Redis 回读与老化判定）。
//
// 用途：Worker 上报路径要区分「内存里已有（继续按上报更新）」与「内存里没有
// （需要从 Redis 快照恢复）」。用 get() 会判僵尸，从而把活跃会话误删。
func (h *RecordingHub) GetSessionForRecovery(sessionID string) (*RecordingSession, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[sessionID]
	if !ok {
		return nil, false
	}
	cp := *s
	return &cp, true
}

// get 取会话（内部加锁）。
func (h *RecordingHub) get(sessionID string) (*RecordingSession, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.Lock()
	if s, ok := h.sessions[sessionID]; ok {
		cp := *s // 返回副本（避免调用方在锁外改字段）
		h.mu.Unlock()
		return &cp, true
	}
	h.mu.Unlock()

	// 内存未命中 → 从 Redis 回读（后端进程重启后内存表为空，但会话仍在
	// Worker 侧跑着；不回读会让前端轮询拿到「会话不存在」，用户以为录制丢了）。
	// 纯前端读取没有任何存活证据，因此按 LastSeen 老化判僵尸。
	s, err := h.loadPersisted(context.Background(), sessionID)
	if err != nil || s == nil {
		return nil, false
	}
	h.markStaleIfIdle(s)
	h.mu.Lock()
	// 双检：回读期间可能有并发写入的更新版本，以内存中的为准。
	if existing, ok := h.sessions[sessionID]; ok {
		cp := *existing
		h.mu.Unlock()
		return &cp, true
	}
	h.sessions[sessionID] = s
	cp := *s
	h.mu.Unlock()

	// 已判定失效的僵尸会话：装回内存表是为了让前端这一次能查到「已失效」的
	// 结论（而不是「不存在」），但它不占并发名额也不该留到最后 —— 立刻删键，
	// 下次 get 就会按「不存在」处理，sweep 也会在下个周期把它清出内存。
	if IsTerminalRecordingState(s.State) {
		h.dropPersisted(context.Background(), sessionID)
	}
	return &cp, true
}

// loadPersisted 从 Redis 回读会话快照（**纯回读**，不做失效判定）。
//
// 为什么不在回读里判僵尸：判定必须分清「谁说的」——
//
//	· 前端 get() 触发回读：没有任何存活证据 → 按 LastSeen 老化判僵尸是对的
//	· Worker 上报触发回读：上报本身就是「它还在录」的最强证据 → 绝不能判僵尸
//
// 把判定留在各自调用方，才能避免「Worker 明明活着，却被后端凭过期时间判死」。
//
// 反序列化失败/键不存在都返回 (nil, nil)，让上层按「会话不存在」处理。
func (h *RecordingHub) loadPersisted(ctx context.Context, sessionID string) (*RecordingSession, error) {
	if h == nil || h.cache == nil || sessionID == "" {
		return nil, nil
	}
	raw, err := h.cache.Get(ctx, fmt.Sprintf(recordingSessionKeyFmt, sessionID))
	if err != nil || raw == "" {
		return nil, nil
	}
	var s RecordingSession
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil, fmt.Errorf("回读录制会话失败: %w", err)
	}
	if s.ID == "" {
		return nil, nil
	}
	// 兼容历史快照（旧代码 LastSeen 是私有字段，压根没进 JSON）：
	// 用 UpdatedAt 兜底，避免被当成「1970 年就没活动过」。
	if s.LastSeen.IsZero() {
		s.LastSeen = s.UpdatedAt
	}
	return &s, nil
}

// markStaleIfIdle 无存活证据时的僵尸判定（仅前端 get 路径使用）。
//
// 判据：Worker 的 recordingPollLoop 每 2s 上报一次状态，前端心跳 1.5s 一次。
// 若 LastSeen 超过 staleSessionAfter 仍标「录制中」，说明执行机不再推进该会话。
func (h *RecordingHub) markStaleIfIdle(s *RecordingSession) {
	if s == nil {
		return
	}
	if IsTerminalRecordingState(s.State) || s.LastSeen.IsZero() {
		return
	}
	if age := time.Since(s.LastSeen); age > staleSessionAfter {
		s.State = RecordingSessionExpired
		s.Message = "会话已失效（执行机长时间无响应）"
		s.UpdatedAt = time.Now()
	}
}

// recoverFromPersist 在 Worker/前端上报触发回读时把会话装回内存。
//
// 与 get() 的关键差别：**不判僵尸**，且允许把后端推断出的 expired 恢复成
// 上报里声明的活跃状态 —— 因为有活跃上报就证明执行机还在跑。
// （expired 是后端凭过期时间推断的，不是 Worker 声明的终态，因此可被证据推翻。）
func (h *RecordingHub) recoverFromPersist(ctx context.Context, sessionID string) (*RecordingSession, bool) {
	s, err := h.loadPersisted(ctx, sessionID)
	if err != nil || s == nil {
		return nil, false
	}
	h.mu.Lock()
	if existing, ok := h.sessions[sessionID]; ok {
		h.mu.Unlock()
		return existing, true
	}
	if IsTerminalRecordingState(s.State) {
		h.logf("录制会话从过期快照恢复（%s → 由上报状态接管）(session=%s)", s.State, sessionID)
		s.State = RecordingSessionRecording
		s.Message = ""
	}
	s.UpdatedAt = time.Now()
	s.LastSeen = s.UpdatedAt
	h.sessions[sessionID] = s
	h.mu.Unlock()
	return s, true
}

// isActiveRecordingStatus 判断上报状态是否代表「执行机仍在推进该会话」。
func isActiveRecordingStatus(status string) bool {
	switch RecordingSessionState(status) {
	case RecordingSessionRecording, RecordingSessionStarting:
		return true
	}
	return false
}

// staleSessionAfter 回读快照的「僵尸会话」阈值。
//
// 取值依据：Worker 的 recordingPollLoop 每 2s 上报一次状态（见
// rpa-worker/internal/worker/recording_listener.go），前端心跳 1.5s 一次。
// 后端只要连续 2 分钟没收到任何刷新，就认为执行机不再推进该会话。
// 取 10 倍冗余（≥2 分钟）是为了容忍网络抖动与后端暂停（GC/GC 暂停/慢盘）。
const staleSessionAfter = 2 * time.Minute

// getMut 取 map 中的**真实指针**（调用方必须已持有 h.mu）。”
//
// 为什么需要它：get() 返回的是副本（对外只读语义正确），但 ReportStatus /
// ReportPick / ReportResult / AbortSession / Keepalive 这 5 个方法拿到副本后
// 在锁内改字段 —— 改的是副本，map 里的会话永远不变，于是：
//
//	· Keepalive 刷不到 lastSeen → 用户正在录，janitor 却判闲置把会话清掉
//	· EventCount / LastPick / Result 全部丢失 → 前端轮询 GetSession 永远看到旧态
//	· AbortSession 置不了终态 → 会话停在 recording，稍后被 sweep 当 expired 清
//
// 这个缺陷在 PR #114 补覆盖率用例时被实测暴露（5 个用例同时红）。
func (h *RecordingHub) getMut(sessionID string) (*RecordingSession, bool) {
	if h == nil {
		return nil, false
	}
	s, ok := h.sessions[sessionID]
	return s, ok
}

// ListSessions 列出会话（按创建时间倒序）。
func (h *RecordingHub) ListSessions() []*RecordingSession {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	out := make([]*RecordingSession, 0, len(h.sessions))
	for _, s := range h.sessions {
		cp := *s
		out = append(out, &cp)
	}
	h.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// janitor 唯一内务 goroutine：回收闲置超时的会话。
func (h *RecordingHub) janitor() {
	defer close(h.janitorDone)
	ticker := time.NewTicker(30 * time.Second)
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

// sweep 一轮回收：闲置超时 → 下发 abort + 标记 expired + 删会话。
func (h *RecordingHub) sweep() {
	h.mu.Lock()
	now := time.Now()
	expired := make([]string, 0)
	for id, s := range h.sessions {
		if IsTerminalRecordingState(s.State) {
			// 终态会话保留一段时间（供前端查产出），超时后清理。
			if now.Sub(s.UpdatedAt) > h.IdleTimeout() {
				expired = append(expired, id)
			}
			continue
		}
		if now.Sub(s.LastSeen) > h.IdleTimeout() {
			expired = append(expired, id)
		}
	}
	h.mu.Unlock()

	for _, id := range expired {
		h.AbortSession(context.Background(), id, "长时间无操作，已自动结束")
		h.mu.Lock()
		delete(h.sessions, id)
		h.mu.Unlock()
		// 同步清 Redis：否则下一次 get() 会把刚回收的会话读回来。
		h.dropPersisted(context.Background(), id)
	}
}

// Close 关闭中枢（中止全部会话，退出 janitor）。
func (h *RecordingHub) Close() {
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
		h.AbortSession(context.Background(), id, "系统关闭")
	}
	h.cancel()
	select {
	case <-h.janitorDone:
	case <-time.After(2 * time.Second):
	}
}

// persist 把会话态写入 Redis（跨进程重启不丢；缓存不可用时静默跳过）。
func (h *RecordingHub) persist(ctx context.Context, s *RecordingSession) {
	if h == nil || h.cache == nil || s == nil {
		return
	}
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	if err := h.cache.Set(ctx, fmt.Sprintf(recordingSessionKeyFmt, s.ID), string(data), recordingSessionTTL); err != nil {
		h.logf("录制会话持久化失败 (session=%s): %v", s.ID, err)
	}
}

// publish 下发一条录制指令到 Redis（尽力而为）。
func (h *RecordingHub) publish(ctx context.Context, channel string, payload interface{}) error {
	if h == nil {
		return fmt.Errorf("录制服务未初始化")
	}
	if h.cache == nil {
		return fmt.Errorf("cache 未注入，无法下发录制指令")
	}
	return publishRedisChannel(ctx, h.cache, channel, payload)
}

// logf 便捷日志。
func (h *RecordingHub) logf(format string, args ...interface{}) {
	if h == nil || h.log == nil {
		return
	}
	h.log.Info(fmt.Sprintf(format, args...))
}

// recordingSessionChannel 录制会话的下行频道名（与 worker 端同名）。
func recordingSessionChannel(sessionID string) string {
	return "recording:session:" + sessionID
}

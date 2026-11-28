package types

import "time"

// ============================================================================
// Phase W6 / REC-01：录制与拾取会话的协议类型。
//
// 归属协议第 4 条：**禁止**把录制类型写进 types.go，本文件是唯一归属地。
// 目的：154（ENGINE-02 动作全集）在 types.go 上持续追加 ActionType 常量，
// 若 W6 也写 types.go 会与 154 同文件冲突。
//
// 协议形态（Worker ↔ 后端 ↔ 前端）：
//
//	下行会话指令 后端 --Redis PubSub recording:session:{sessionId}--> Worker
//	上行产出     Worker --HTTP POST /rpa/workers/recording/*--> 后端
//	画面帧       Worker --HTTP POST /rpa/workers/liveview/frame--> 后端
//	             --NoticeHub WS rpa_liveview_frame--> 前端（**复用 155 帧通道**）
//
// 画面帧**不复用** recording 自己的消息类型：帧的语义（Worker 浏览器画面 +
// viewport 坐标系 + 可丢广播）与 155 liveview 完全相同，另立一套必然产生
// 第二套坐标系约定（详见 .planning/phases/w6-rec-and-schedules/W6-PLAN.md §2.2）。
// 会话生命周期**另立**频道：录制会话不是「一次执行」，没有 execution 行、
// 不进执行状态机、不写执行审计。
// ============================================================================

// RecordingSessionChannel 单录制会话的下行频道名（与后端 publish 侧同名）。
func RecordingSessionChannel(sessionID string) string {
	return "recording:session:" + sessionID
}

// RecordingSessionPattern 全局订阅模式（Worker 用 PSubscribe 一次覆盖所有会话）。
const RecordingSessionPattern = "recording:session:*"

// RecordingAction 下行指令的动作名。
type RecordingAction string

const (
	// RecordingActionStart 开始录制：开页面 + 注入录制器 + 开始采集。
	RecordingActionStart RecordingAction = "start"
	// RecordingActionStop 停止录制：拉事件 + 产出 actions + 释放页面。
	RecordingActionStop RecordingAction = "stop"
	// RecordingActionPick 拾取元素：按坐标取 7 级选择器阶梯。
	RecordingActionPick RecordingAction = "pick"
	// RecordingActionKeepalive 心跳：刷新会话闲置计时。
	RecordingActionKeepalive RecordingAction = "keepalive"
	// RecordingActionAbort 强制中止：释放页面（前端异常退出 / 后端超时清理）。
	RecordingActionAbort RecordingAction = "abort"
)

// IsValidRecordingAction 动作名白名单（未知动作拒绝，避免指令被误执行）。
func IsValidRecordingAction(a RecordingAction) bool {
	switch a {
	case RecordingActionStart, RecordingActionStop, RecordingActionPick,
		RecordingActionKeepalive, RecordingActionAbort:
		return true
	}
	return false
}

// RecordingCommand 后端 → Worker 的录制会话指令（Redis PubSub 载体）。
type RecordingCommand struct {
	// Action 指令动作（见 RecordingAction）。
	Action RecordingAction `json:"action"`
	// SessionID 会话 ID（与频道名后缀一致，Worker 从频道反解时以此为准）。
	SessionID string `json:"sessionId"`
	// WorkerID 目标 Worker（start 时指定；Worker 侧据此拒收不属于自己的指令）。
	WorkerID string `json:"workerId,omitempty"`
	// TaskID 关联任务 ID（可选：录制结果可直接写回该任务的脚本）。
	TaskID string `json:"taskId,omitempty"`
	// Operator 操作者展示名（审计与前端提示用）。
	Operator string `json:"operator,omitempty"`
	// TargetURL 起始页面地址（start 时可空 = 空白页起步）。
	TargetURL string `json:"targetUrl,omitempty"`
	// PickX / PickY 拾取坐标（**viewport CSS 像素**，与 155 liveview 坐标系一致）。
	PickX float64 `json:"pickX,omitempty"`
	PickY float64 `json:"pickY,omitempty"`
	// Timestamp 指令发出时间（毫秒）。
	Timestamp int64 `json:"timestamp,omitempty"`
}

// RecordingStatus 会话状态取值（Worker → 后端上报 + 后端 → 前端展示）。
type RecordingStatus string

const (
	// RecordingStatusStarting 正在准备（开页面、注入录制器）。
	RecordingStatusStarting RecordingStatus = "starting"
	// RecordingStatusRecording 正在录制。
	RecordingStatusRecording RecordingStatus = "recording"
	// RecordingStatusStopped 已停止（已产出结果）。
	RecordingStatusStopped RecordingStatus = "stopped"
	// RecordingStatusFailed 出错中止。
	RecordingStatusFailed RecordingStatus = "failed"
	// RecordingStatusAborted 被强制中止。
	RecordingStatusAborted RecordingStatus = "aborted"
)

// IsValidRecordingStatus 状态白名单（未知状态拒绝，避免前端显示乱码）。
func IsValidRecordingStatus(s RecordingStatus) bool {
	switch s {
	case RecordingStatusStarting, RecordingStatusRecording, RecordingStatusStopped,
		RecordingStatusFailed, RecordingStatusAborted:
		return true
	}
	return false
}

// RecordingUpstreamStatus Worker → 后端的状态上报体。
//
// 节流上报（默认 1s 一次）：让前端显示「已录 N 个动作」而不必等停止。
type RecordingUpstreamStatus struct {
	SessionID string          `json:"sessionId"`
	WorkerID  string          `json:"workerId,omitempty"`
	Status    RecordingStatus `json:"status"`
	// EventCount 已采集到的事件数（前端展示「已录 N 个动作」）。
	EventCount int    `json:"eventCount"`
	Message    string `json:"message,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
}

// RecordingUpstreamResult Worker → 后端的最终产出（**权威结果**）。
//
// Actions 的 schema 与 types.Action 完全一致（recorder.BuildRecording 保证），
// 后端直接落库为任务脚本，无需二次转换。
type RecordingUpstreamResult struct {
	SessionID string `json:"sessionId"`
	WorkerID  string `json:"workerId,omitempty"`
	// Actions 产出的动作序列（与 types.Action 同构）。
	Actions []Action `json:"actions"`
	// Steps 每个动作的中文描述（与 Actions 等长；recorder 生成）。
	Steps []string `json:"steps"`
	// StartURL 录制起始地址。
	StartURL string `json:"startUrl,omitempty"`
	// EventCount 本次录制累计采集到的事件数。
	EventCount int   `json:"eventCount"`
	Timestamp  int64 `json:"timestamp,omitempty"`
}

// RecordedTier 一级选择器候选（7 级阶梯的其中一级）。
//
// 与 browser.RecordedTier 同构（JSON 字段名一致），但独立声明以避免
// worker 的 browser 包与 types 包互相依赖。
type RecordedTier struct {
	// Tier 层级序号（1 = 最稳）。
	Tier int `json:"tier"`
	// Name 层级名称（data-testid / role / text / css / xpath ...）。
	Name string `json:"name"`
	// Selector 该级的选择器表达式。
	Selector string `json:"selector"`
	// OK 该级在当前页面上是否可用（逐级实测，非静态推断）。
	OK bool `json:"ok"`
	// Score 该级评分（越高越稳）。
	Score int `json:"score"`
}

// RecordingUpstreamPick Worker → 后端的拾取结果。
//
// Tiers 是 7 级阶梯（逐级可用性），前端据此渲染浮层。
type RecordingUpstreamPick struct {
	SessionID string `json:"sessionId"`
	WorkerID  string `json:"workerId,omitempty"`
	// Found 是否拾取到元素。
	Found bool `json:"found"`
	// Tag 元素标签名。
	Tag string `json:"tag,omitempty"`
	// Type 元素类型（input 的 type 等）。
	Type string `json:"type,omitempty"`
	// Text 元素可见文本。
	Text string `json:"text,omitempty"`
	// Tiers 7 级选择器阶梯。
	Tiers []RecordedTier `json:"tiers,omitempty"`
	// Recommended 推荐的最高可用层级选择器。
	Recommended string `json:"recommended,omitempty"`
	// RecommendedTier 推荐层级（1 起）。
	RecommendedTier int `json:"recommendedTier,omitempty"`
	// RecommendedScore 推荐层级评分。
	RecommendedScore int `json:"recommendedScore,omitempty"`
	// Frame 元素所在 iframe 的选择器（拾取在 iframe 内时非空）。
	Frame string `json:"frame,omitempty"`
	// Message 拾取失败时的可读原因。
	Message  string `json:"message,omitempty"`
	Timestamp int64 `json:"timestamp,omitempty"`
}

// RecordingNowMS 毫秒时间戳（与 liveview 协议同款，避免秒/纳秒混用）。
func RecordingNowMS() int64 { return time.Now().UnixMilli() }

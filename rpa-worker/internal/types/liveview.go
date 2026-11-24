package types

import "time"

// ============================================================================
// Phase 155 / LIVEVIEW-01~05：实时画面与人工接管的协议类型。
//
// 归属协议第 2 条：**禁止**把 liveview 类型写进 types.go，本文件是唯一归属地。
// 目的：154（ENGINE-02 动作全集）在 types.go 上持续追加 ActionType 常量，
// 若 155 也写 types.go 会与 154 同文件冲突（154-SUMMARY §8 已登记该风险）。
//
// 协议形态（Worker ↔ 后端 ↔ 前端）：
//
//	上行帧   Worker --HTTP POST /rpa/workers/liveview/frame--> 后端 --既有 WS--> 前端 Canvas
//	下行输入 前端 --HTTP POST /rpa/executions/:id/liveview/input--> 后端
//	          后端 --Redis PubSub execution:liveview:{id}--> Worker --> CDP Input.dispatch*
//
// 帧不走 Worker→后端的 WS 长连：Worker 现为纯 HTTP client（communication.APIClient），
// 新建 Worker WS 长连会引入第二条连接生命周期（重连/背压/鉴权两套）。后端收到 HTTP
// 帧后用既有 NoticeHub 推给前端，即「经既有 WS 通道推前端」成立且零新增依赖。
// ============================================================================

// LiveSessionState 单路推流的状态（Worker 与后端两侧共用取值）。
type LiveSessionState string

const (
	// LiveSessionIdle 已 watch 但尚未收到首帧。
	LiveSessionIdle LiveSessionState = "idle"
	// LiveSessionStreaming 正在推帧。
	LiveSessionStreaming LiveSessionState = "streaming"
	// LiveSessionStopped 已停流（前端 stop / 静默超时 / 执行结束）。
	LiveSessionStopped LiveSessionState = "stopped"
)

// LiveMode 协作模式（LIVEVIEW-03）。
//
// 注意：模式是「控制权档位」，**不是执行状态**，故不引入新的 ExecutionStatus
// 常量（RUNTIME-05 的执行状态机只有 pending/running/paused/终态）。模式切换时
// 若需要暂停执行，走 RUNTIME-05 的 running → paused 合法迁移，不另造校验。
type LiveMode string

const (
	// LiveModeAuto 全自动：不暂停，仅可观。
	LiveModeAuto LiveMode = "auto"
	// LiveModeCopilot 关键节点暂停等人工确认（默认，符合一般员工信任曲线）。
	LiveModeCopilot LiveMode = "copilot"
	// LiveModeManual 完全人工：默认暂停，人点「继续」才放行动作。
	LiveModeManual LiveMode = "manual"
)

// IsValidLiveMode 模式取值校验（未知取值必须显式拒绝，不得静默降级为默认）。
func IsValidLiveMode(m LiveMode) bool {
	switch m {
	case LiveModeAuto, LiveModeCopilot, LiveModeManual:
		return true
	}
	return false
}

// NormalizeLiveMode 空值归一为默认 copilot；非法值返回 false 由调用方拒绝。
func NormalizeLiveMode(m LiveMode) (LiveMode, bool) {
	if m == "" {
		return LiveModeCopilot, true
	}
	if !IsValidLiveMode(m) {
		return m, false
	}
	return m, true
}

// LiveViewEvent 审计与时间线事件类型（LIVEVIEW-05 要求四类 + 模式切换 + 输入）。
type LiveViewEvent string

const (
	// LiveViewEventWatchStart 开始看（谁、何时、哪个执行）。
	LiveViewEventWatchStart LiveViewEvent = "watch_start"
	// LiveViewEventWatchStop 结束看。
	LiveViewEventWatchStop LiveViewEvent = "watch_stop"
	// LiveViewEventTakeover 接管（抢占需显式确认，见 LIVEVIEW-05）。
	LiveViewEventTakeover LiveViewEvent = "takeover"
	// LiveViewEventRelease 释放控制权。
	LiveViewEventRelease LiveViewEvent = "release"
	// LiveViewEventModeSwitch 模式切换（LIVEVIEW-03 要求记审计与时间线）。
	LiveViewEventModeSwitch LiveViewEvent = "mode_switch"
	// LiveViewEventInput 输入注入（接管期内的真实操作留痕）。
	LiveViewEventInput LiveViewEvent = "input"
)

// IsValidLiveViewEvent 事件类型白名单（后端拒绝未知事件，避免审计表被污染）。
func IsValidLiveViewEvent(e LiveViewEvent) bool {
	switch e {
	case LiveViewEventWatchStart, LiveViewEventWatchStop,
		LiveViewEventTakeover, LiveViewEventRelease,
		LiveViewEventModeSwitch, LiveViewEventInput:
		return true
	}
	return false
}

// LiveFrame 一帧 screencast 画面（LIVEVIEW-01）。
//
// Data 为 base64 编码的 JPEG。前端 Canvas 以 Data 直接建 Image 对象渲染。
// Width/Height 是 **JPEG 实际像素尺寸**（由 Worker 侧读 JPEG 头得到，非元数据推算）；
// DeviceWidth/DeviceHeight 是 CSS 像素 viewport 尺寸（DIP），前端做
// 「画布像素 → viewport CSS 像素」缩放映射时用这一对（LIVEVIEW-02 坐标系）。
type LiveFrame struct {
	ExecutionID string `json:"executionId"`
	Seq         int64  `json:"seq"`
	TimestampMS int64  `json:"timestamp"`
	// Data base64(JPEG)。空串表示本帧为控制帧（如 stop 通知），无像素。
	Data            string  `json:"data,omitempty"`
	ByteSize        int     `json:"byteSize"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	DeviceWidth     float64 `json:"deviceWidth"`
	DeviceHeight    float64 `json:"deviceHeight"`
	ScrollOffsetX   float64 `json:"scrollOffsetX"`
	ScrollOffsetY   float64 `json:"scrollOffsetY"`
	PageScaleFactor float64 `json:"pageScaleFactor"`
}

// LiveViewContext 叠加层情境数据（LIVEVIEW-04 —— 情境感知，不是装饰）。
//
// Variables 必须是**脱敏后**的字符串快照：Worker 侧写前用 MaskSensitive 处理键名，
// 敏感键（password/token/secret/...）的值恒为 ******。原始值绝不进入本结构。
type LiveViewContext struct {
	ExecutionID  string `json:"executionId"`
	TaskName     string `json:"taskName"`
	WorkerID     string `json:"workerId"`
	WorkerName   string `json:"workerName"`
	Step         int    `json:"step"`
	TotalSteps   int    `json:"totalSteps"`
	ActionType   string `json:"actionType"`
	ActionDesc   string `json:"actionDesc"`
	Status       string `json:"status"`
	Mode         LiveMode `json:"mode"`
	TakeoverBy   string `json:"takeoverBy,omitempty"`
	TakeoverAtMS int64  `json:"takeoverAt,omitempty"`
	LastLog      string `json:"lastLog,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	StartedAtMS  int64  `json:"startedAt"`
}

// LiveViewUpstreamFrame Worker → 后端的帧上报体（一帧 + 最新叠加层上下文）。
type LiveViewUpstreamFrame struct {
	Frame   LiveFrame        `json:"frame"`
	Context *LiveViewContext `json:"context,omitempty"`
}

// LiveViewAuditEvent Worker → 后端的审计事件上报体（LIVEVIEW-05 四类 +
// 模式切换 + 输入留痕）。
//
// 落点：后端写 sys_rpa_audit_logs（既有审计表，ResourceType=execution），
// 不新建表结构 —— 审计是「谁在何时对哪个执行做了什么」，现有列已够（
// Action 承载事件类型、NewValue 存 JSON 细节、OperatorID/Name 是操作者）。
type LiveViewAuditEvent struct {
	ExecutionID string        `json:"executionId"`
	Event       LiveViewEvent `json:"event"`
	Operator    string        `json:"operator"`
	Reason      string        `json:"reason,omitempty"`
	Mode        LiveMode      `json:"mode,omitempty"`
	Detail      string        `json:"detail,omitempty"`
	TimestampMS int64         `json:"timestamp"`
}

// LiveInputKind 前端下行输入指令类型（LIVEVIEW-02）。
type LiveInputKind string

const (
	LiveInputMouseMove  LiveInputKind = "mouse_move"
	LiveInputMouseDown  LiveInputKind = "mouse_down"
	LiveInputMouseUp    LiveInputKind = "mouse_up"
	LiveInputMouseWheel LiveInputKind = "mouse_wheel"
	LiveInputKeyDown    LiveInputKind = "key_down"
	LiveInputKeyUp      LiveInputKind = "key_up"
	LiveInputText       LiveInputKind = "text"
)

// IsValidLiveInputKind 输入类型白名单（未知类型拒绝，避免把任意 CDP 方法透传）。
func IsValidLiveInputKind(k LiveInputKind) bool {
	switch k {
	case LiveInputMouseMove, LiveInputMouseDown, LiveInputMouseUp,
		LiveInputMouseWheel, LiveInputKeyDown, LiveInputKeyUp, LiveInputText:
		return true
	}
	return false
}

// LiveInputCommand 注入同一 page 的一条输入指令。
//
// X/Y 是 **viewport CSS 像素**（前端已完成画布→viewport 缩放换算）；
// Modifiers 位掩码：Alt=1, Ctrl=2, Meta=4, Shift=8（与 CDP 一致）；
// Key/Code/WindowsVirtualKeyCode 由前端 KeyboardEvent 原样带过来，Worker 不做猜测。
type LiveInputCommand struct {
	ExecutionID string        `json:"executionId"`
	Kind        LiveInputKind `json:"kind"`
	X           float64       `json:"x,omitempty"`
	Y           float64       `json:"y,omitempty"`
	Button      string        `json:"button,omitempty"`      // left|right|middle|none
	ClickCount  int           `json:"clickCount,omitempty"`  // 默认 1
	Modifiers   int           `json:"modifiers,omitempty"`
	DeltaX      float64       `json:"deltaX,omitempty"`
	DeltaY      float64       `json:"deltaY,omitempty"`
	Key         string        `json:"key,omitempty"` // DOM KeyboardEvent.key
	Code        string        `json:"code,omitempty"`
	Text        string        `json:"text,omitempty"`
	VirtualKey  int           `json:"virtualKey,omitempty"`
	Operator    string        `json:"operator,omitempty"`   // 注入者（审计）
	TimestampMS int64         `json:"timestamp,omitempty"`
}

// LiveViewCommand 后端 → Worker 的下行指令（经 Redis PubSub execution:liveview:{id}）。
type LiveViewCommand struct {
	Action      string           `json:"action"` // watch|stop|input|takeover|release|mode|keepalive|continue|abort
	ExecutionID string           `json:"executionId"`
	Operator    string           `json:"operator,omitempty"`
	Mode        LiveMode         `json:"mode,omitempty"`
	Input       *LiveInputCommand `json:"input,omitempty"`
	// Force 抢占确认（LIVEVIEW-05：已被他人接管时必须显式确认才允许抢占）。
	Force       bool   `json:"force,omitempty"`
	Reason      string `json:"reason,omitempty"`
	TimestampMS int64  `json:"timestamp,omitempty"`
}

// 下行指令动作名常量。
const (
	LiveCmdWatch     = "watch"
	LiveCmdStop      = "stop"
	LiveCmdInput     = "input"
	LiveCmdTakeover  = "takeover"
	LiveCmdRelease   = "release"
	LiveCmdMode      = "mode"
	LiveCmdKeepalive = "keepalive"
	// LiveCmdContinue LIVEVIEW-03：放行被暂停的 RPA 动作（copilot 确认 / manual
	// 人工继续 / 释放接管后的断点续跑）。
	LiveCmdContinue = "continue"
	// LiveCmdAbort 人工中止（等价 RUNTIME-04 的 abort 哨兵 → cancelled 终态）。
	LiveCmdAbort = "abort"
)

// LiveViewChannel 单执行的下行指令频道名（与后端 publish 侧必须同名）。
func LiveViewChannel(executionID string) string {
	return "execution:liveview:" + executionID
}

// LiveViewPattern 全局订阅模式（Worker 用 PSubscribe 一次覆盖所有执行）。
const LiveViewPattern = "execution:liveview:*"

// NowMS 毫秒时间戳（协议统一用 ms，避免秒/纳秒混用）。
func NowMS() int64 { return time.Now().UnixMilli() }

// NewLiveViewContext 构造空上下文（nil 安全的起点，供实现在缺省时返回）。
func NewLiveViewContext(executionID string) *LiveViewContext {
	return &LiveViewContext{ExecutionID: executionID, Mode: LiveModeCopilot, StartedAtMS: NowMS()}
}

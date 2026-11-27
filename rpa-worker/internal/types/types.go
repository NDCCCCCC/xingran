package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Task variable key constants
const (
	VariableIsSubTask         = "__isSubTask"
	VariableParentExecutionID = "__parentExecutionID"
	VariableSubTaskIndex      = "__subTaskIndex"
	VariableSessionData       = "__sessionData"
	VariableCredentials       = "__credentials"
)

// ErrExecutionAborted 人工干预 abort 的哨兵错误（Phase 153 / RUNTIME-04）：
// executePause 收到 action=abort 时返回，Execute 捕获后把结果置为 cancelled。
var ErrExecutionAborted = NewAbortError()

// AbortError abort 哨兵（独立类型便于 errors.Is 判定）。
type AbortError struct{}

func NewAbortError() *AbortError    { return &AbortError{} }
func (e *AbortError) Error() string { return "execution aborted by human intervention" }
func (e *AbortError) Is(target error) bool {
	_, ok := target.(*AbortError)
	return ok
}

// ============================================================================
// W7.1 / 缺陷 B：可重试性判定 —— 「配置错误」与「运行时失败」必须分开
//
// 生产实证（v1.56 W7 之后仍 0 帧）：
//
//	action type="goto" → executeAction switch 无 case → default 返回
//	"unsupported action type: goto" → 该动作 retry=2 > 0 → 进入重试分支 →
//	先调 liveActionGate(force=true) → copilot 模式下 needWait 恒 true →
//	永久阻塞，10 分钟后 context deadline exceeded，页面停在 about:blank，
//	screencast 一帧不发。后端 30 分钟内 /liveview/frame 请求数 = 0。
//
// 这类错误重试 100 次也不会成功，它不是「动作真的执行失败」（那才是 force
// 闸门的设计意图：让人看一眼），而是**脚本配置写错了**。把它送进人工闸门
// 等于让一次配置错误吃掉一个执行 + 一个值守人力。
//
// 判定口径（本文件是唯一真相）：
//   - 不可重试：unsupported action type / missing selector / missing url
//     parameter / 其它参数校验失败 → 立即 failed，给出可读原因，不 gate、不空转。
//   - 可重试：网络超时 / 元素未找到 / 导航失败 / 目标站 5xx → 保持原逻辑
//     （含 force 闸门：人工看一眼再重试）。
//
// 实现选择「显式类型而非字符串匹配」：错误文案是可读产物，会随文案调整而
// 漂移；用类型断言 errors.As 判定才稳。
// ============================================================================

// ErrNonRetryable 不可重试错误的哨兵（配合 NonRetryableError 使用）。
var ErrNonRetryable = nonRetryableSentinel{}

type nonRetryableSentinel struct{}

func (nonRetryableSentinel) Error() string { return "non-retryable configuration error" }

// NonRetryableError 标记「重试无意义」的错误（配置/参数错误）。
//
// 包装形式：errors.As 可取出本类型；Unwrap 保留底层错误便于日志溯源。
type NonRetryableError struct {
	// Reason 给用户看的可读原因（一般员工能看懂，不含内部术语）。
	Reason string
	Err    error
}

// NewNonRetryableError 构造不可重试错误（无底层错误）。
func NewNonRetryableError(reason string) *NonRetryableError {
	return &NonRetryableError{Reason: reason}
}

// NewNonRetryableErrorf 构造不可重试错误（格式化可读原因）。
func NewNonRetryableErrorf(format string, args ...interface{}) *NonRetryableError {
	return &NonRetryableError{Reason: fmt.Sprintf(format, args...)}
}

// WrapNonRetryable 把既有错误标记为不可重试，reason 为给用户看的原因。
func WrapNonRetryable(reason string, err error) *NonRetryableError {
	return &NonRetryableError{Reason: reason, Err: err}
}

func (e *NonRetryableError) Error() string {
	if e == nil {
		return "non-retryable error"
	}
	if e.Err != nil && e.Reason != "" {
		return e.Reason + ": " + e.Err.Error()
	}
	if e.Reason != "" {
		return e.Reason
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "non-retryable error"
}

// Unwrap 暴露底层错误（errors.Is/As 可穿透）。
func (e *NonRetryableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is 让 errors.Is(err, types.ErrNonRetryable) 成立。
func (e *NonRetryableError) Is(target error) bool {
	_, ok := target.(nonRetryableSentinel)
	return ok
}

// IsNonRetryable 判定错误是否属于「重试无意义」的配置/参数错误。
//
// nil 安全：nil 视为不可重试（没有错误就没有重试的语义），但调用方通常
// 在 err != nil 分支里问，因此不会掩盖真实失败。
func IsNonRetryable(err error) bool {
	if err == nil {
		return false
	}
	var nre *NonRetryableError
	return errors.As(err, &nre)
}

// Task 任务定义
type Task struct {
	ID          string                 `json:"id"`
	TaskName    string                 `json:"taskName"`
	Description string                 `json:"description"`
	TargetURL   string                 `json:"targetUrl"`
	Script      *Script                `json:"script"`
	Timeout     time.Duration          `json:"timeout"`
	MaxRetry    int                    `json:"maxRetry"`
	Priority    int                    `json:"priority"`
	Status      int                    `json:"status"`
	Variables   map[string]interface{} `json:"inputParams"`
}

// Script 脚本定义
type Script struct {
	Actions []Action `json:"actions"`
}

// Action 动作定义
type Action struct {
	ID          string                 `json:"id"`
	Type        ActionType             `json:"type"`
	Description string                 `json:"description"`
	Selector    string                 `json:"selector,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Retry       int                    `json:"retry,omitempty"`
	Value       string                 `json:"value,omitempty"`
	AIAssisted  bool                   `json:"aiAssisted,omitempty"`
}

// ============================================================================
// 动作类型别名归一化（W7.1 / 缺陷 A）：唯一真相在 types 包
//
// 为什么放这里而不是 executor：动作有**两条**进入路径，只要有一处漏做，
// 生产就会复现「动作压根没执行」的假象：
//
//	路径 1（顶层）：TaskMessage.Script 是强类型 []Action，json.Unmarshal 直接
//	               按字段赋值 —— "type":"goto" 原样落进 Action.Type，**不经过**
//	               executor 的 mapToAction。生产库 6 个 goto 任务全走这条。
//	路径 2（嵌套）：loop / condition 的 params.actions 是 []map[string]interface{},
//	               经 executor.mapToAction 转换。
//
// 两条路径都必须调用 NormalizeAction，且都在「Action.Type 赋值处」完成后就不再
// 分散判断 —— 这正是任务书要求的形态。
// ============================================================================

// ActionTypeAliases 历史/前端惯用写法 → 规范 ActionType（key 必须小写）。
//
// 取值依据（先取证再定表，不凭想象扩）：
//
//	· scripts/ops/rpa_action_type_census 普查生产库 19 个任务 / 12 种 type：
//	  非规范写法只有 "goto"（6 步，全部 2026-03-10 创建的测试数据）
//	· 前端 RPA_ACTION_TYPES（xingran-react-frontend/src/types/rpa.ts）只有规范写法
//	· "url" 是历史脚本把 URL 当类型名的惯用写法（后端 extractURLFromScript /
//	  convertToWorkerAction 都把 URL 放 value 字段，说明存量确有该形状）
//	· goTo / GOTO 等变体由「查表前统一小写」覆盖，不为每种大小写各写一行
var ActionTypeAliases = map[string]ActionType{
	"goto": ActionNavigate,
	"url":  ActionNavigate,
}

// NormalizeActionType 归一化单个类型字符串；命中别名时返回别名原文。
func NormalizeActionType(raw string) (ActionType, string) {
	if raw == "" {
		return ActionType(raw), ""
	}
	if alias, ok := ActionTypeAliases[strings.ToLower(raw)]; ok {
		return alias, raw
	}
	return ActionType(raw), ""
}

// NormalizeAction 就地归一化一个动作（**唯一入口**，两条路径都调它）：
//
//  1. 别名 type → 规范 type（返回命中的别名原文，供调用方打 WARN）
//
// 2. navigate 的 URL 键位提升：value → params["url"]（不覆盖已有值）
// 3. fill / select 的填值键位提升：value → params["value"]（不覆盖已有值）
// 4. 历史脚本的**参数键名**归一化（见 paramKeyAliases，同族缺陷）
//
// 第 2~4 步不是"顺手"做的：生产库里的历史脚本把「这一步的主角值」统一放在
// 顶层 value 字段、把参数写成另一套键名（code / extractType / outputVariable /
// dataSource 之外的 itemName / scroll 的 distance ...），而 worker 的各个 case
// 只认一种写法。后端 convertToWorkerAction 只在 type=="navigate"/"fill"/"select"
// 时做提升 —— 而 goto 走不到 navigate 那一句，于是「类型没归一化」之外还叠了
// 「参数键位不匹配」。只修类型，错误会从 "unsupported action type" 变成
// "missing url parameter"/"缺少 value 参数"/"缺少 script 参数"——任务照样跑不通。
//
// 与后端语义完全同构（同样三条规则），因此后端已提升过的消息再经本函数是
// 幂等的（已有键不覆盖）。
func NormalizeAction(a *Action) (aliasRaw string) {
	if a == nil {
		return ""
	}
	normalized, hitAlias := NormalizeActionType(string(a.Type))
	if hitAlias != "" && hitAlias != string(normalized) {
		a.Type = normalized
		aliasRaw = hitAlias
	}
	switch a.Type {
	case ActionNavigate:
		// navigate case 读 params["url"]。
		putParamIfAbsent(a, "url", a.Value)
	case ActionFill, ActionSelect:
		// fill/select case 读 params["value"]。
		putParamIfAbsent(a, "value", a.Value)
	}
	normalizeParamKeys(a)
	return aliasRaw
}

// paramKeyAliases 历史脚本的参数键名 → worker case 读的规范键名（按动作类型分组）。
//
// 取证来源：scripts/ops/rpa_goto_task_dump 导出的生产库 6 个 goto 任务的真实
// 脚本（全部 2026-03-10 创建）。这些键名在前端旧版编辑器与手工脚本里很常见，
// worker 各 case 却只读规范键 —— 与"type 别名"是同一族缺陷（同一份数据，
// 两种写法），所以放在同一个唯一入口里归一化。
var paramKeyAliases = map[ActionType]map[string]string{
	// script / evaluate：历史脚本写 params.code，case 读 params.script。
	ActionScript:   {"code": "script"},
	ActionEvaluate: {"code": "script", "expression": "script"},
	// extract：历史脚本写 extractType/outputVariable，case 读 type/targetVar。
	ActionExtract: {"extractType": "type", "outputVariable": "targetVar"},
	// scroll：历史脚本写 distance，case 读 pixels。
	ActionScroll: {"distance": "pixels"},
	// loop：历史脚本写 itemName，case 读 itemVar。
	ActionLoop: {"itemName": "itemVar"},
	// wait：历史脚本写 delay（毫秒），case 读 duration。
	ActionWait: {"delay": "duration"},
	// waitFor：历史脚本复用 delay 表达超时。
	ActionWaitFor: {"delay": "timeout"},
}

// normalizeParamKeys 按动作类型把历史键名映射成规范键名。
//
// 语义边界：
//   - 只做**改名**，不改值类型；目标键已存在时不覆盖（显式规范写法优先）。
//   - 值类型不同（例如 delay 本身已是 int）时也照搬，由各 case 的类型断言兜底。
//   - 不改原键（保留现场，便于日志与排查看到「脚本原本怎么写的」）。
func normalizeParamKeys(a *Action) {
	if a == nil || a.Params == nil {
		return
	}
	aliases, ok := paramKeyAliases[a.Type]
	if !ok {
		return
	}
	for from, to := range aliases {
		v, exists := a.Params[from]
		if !exists || v == nil {
			continue
		}
		if _, already := a.Params[to]; already {
			continue // 规范键已存在 → 显式写法优先
		}
		a.Params[to] = v
	}
}

// putParamIfAbsent 把非空值写进 params[key]（已有键永远优先，不覆盖）。
//
// 空值不写：留空的 value 是「这一步没有值」的信号，补一个空串进去会让
// 「缺少 value 参数」的可读报错变成一个必然失败的空填写。
func putParamIfAbsent(a *Action, key, value string) {
	if a == nil || value == "" {
		return
	}
	if a.Params == nil {
		a.Params = make(map[string]interface{}, 1)
	}
	if _, exists := a.Params[key]; !exists {
		a.Params[key] = value
	}
}

// ActionType 动作类型
//
// Phase 154 / ENGINE-02：本文件是动作枚举的唯一真相（前端
// ACTION_TYPE_OPTIONS 由 154 扩充、156 只消费）。新增动作必须同时在
// internal/executor/engine.go 的 executeAction switch 里落 case，
// 否则 engine_action_coverage_test.go 的静态断言会红。
type ActionType string

const (
	// —— 基础动作 ——
	ActionNavigate   ActionType = "navigate"
	ActionClick      ActionType = "click"
	ActionFill       ActionType = "fill"
	ActionSelect     ActionType = "select"
	ActionWait       ActionType = "wait"
	ActionScreenshot ActionType = "screenshot"
	ActionExtract    ActionType = "extract"
	ActionScroll     ActionType = "scroll"
	ActionUpload     ActionType = "upload"
	ActionDownload   ActionType = "download"
	ActionEvaluate   ActionType = "evaluate"
	ActionWaitFor    ActionType = "waitFor"
	ActionClose      ActionType = "close"
	ActionLoop       ActionType = "loop"      // Loop over data array
	ActionPause      ActionType = "pause"     // Wait for human input
	ActionCondition  ActionType = "condition" // Conditional branching
	ActionAutoLogin  ActionType = "autologin" // Auto login with stored credentials

	// —— Phase 154 / ENGINE-02 差集 B：常量与 case 双缺，本 phase 一并补齐 ——
	ActionAssert      ActionType = "assert"      // 断言 + 失败即终止
	ActionHover       ActionType = "hover"       // 悬停
	ActionDoubleClick ActionType = "doubleClick" // 双击
	ActionRightClick  ActionType = "rightClick"  // 右键
	ActionDrag        ActionType = "drag"        // 拖拽（HTML5 兼容）
	ActionKeyboard    ActionType = "keyboard"    // 组合键 / 快捷键
	ActionSwitchTab   ActionType = "switchTab"   // 页签切换
	ActionIframe      ActionType = "iframe"      // 进入 / 退出 frame
	ActionScript      ActionType = "script"      // 自定义脚本（等价 evaluate）
)

// TaskMessage 任务消息（从 Redis）
type TaskMessage struct {
	ExecutionID string                 `json:"executionId"`
	TaskID      string                 `json:"taskId"`
	TaskName    string                 `json:"taskName"`
	TargetURL   string                 `json:"targetUrl"`
	Script      []Action               `json:"script"`
	Timeout     time.Duration          `json:"timeout"`
	MaxRetry    int                    `json:"maxRetry"`
	Variables   map[string]interface{} `json:"inputParams"`
	TriggeredBy string                 `json:"triggeredBy"`
	TriggerType string                 `json:"triggerType"`
	CreatedAt   time.Time              `json:"createdAt"`

	// 自动登录相关
	CredentialID string       `json:"credentialId,omitempty"`
	SessionID    string       `json:"sessionId,omitempty"`   // 已存在的会话ID
	SessionData  *SessionData `json:"sessionData,omitempty"` // 会话数据（token/cookie）

	// normalizedAliases W7.1 / 缺陷 A：反序列化时命中的动作类型别名原文
	// （"goto" / "url" / 大小写变体）。不参与 JSON 序列化，仅用于 executor 打
	// WARN（"normalized action type goto -> navigate"），让运维能继续发现历史写法。
	normalizedAliases []string
}

// NormalizedAliases 返回反序列化阶段归一化过的别名原文（只读副本）。
func (t *TaskMessage) NormalizedAliases() []string {
	if t == nil || len(t.normalizedAliases) == 0 {
		return nil
	}
	out := make([]string, len(t.normalizedAliases))
	copy(out, t.normalizedAliases)
	return out
}

// UnmarshalJSON 自定义反序列化（W7.1 / 缺陷 A / 路径 1）。
//
// 为什么必须自定义：Script 是强类型 []Action，标准 json.Unmarshal 直接按字段
// 赋值，"type":"goto" 会被**原样**落进 Action.Type，完全不经过 executor 的
// mapToAction —— 生产库 6 个 goto 任务走的正是这条路径（顶层 script），
// 因此只在 mapToAction 里归一化是不够的（第一版修复即踩到此坑）。
//
// 做法：先用别名类型避免递归，再把每个动作过一遍 NormalizeAction（唯一入口）。
// 归一化只改内存中的 Action，不写回 Redis 消息原文。
//
// 命中的别名原文收进 NormalizedAliases（不参与 JSON 序列化）：types 包没有
// logger，WARN 由 executor 在拿到任务时统一打 —— 这样「日志里有
// normalized action type」这条运维可观测性在两条路径上口径一致。
func (t *TaskMessage) UnmarshalJSON(data []byte) error {
	// 别名类型避免递归调用本方法（标准 Go 手法）。
	type taskMessageAlias TaskMessage
	var alias taskMessageAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	for i := range alias.Script {
		if raw := NormalizeAction(&alias.Script[i]); raw != "" {
			alias.normalizedAliases = append(alias.normalizedAliases, raw)
		}
	}
	*t = TaskMessage(alias)
	return nil
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID  string                 `json:"executionId"`
	TaskID       string                 `json:"taskId"`
	Status       ExecutionStatus        `json:"status"`
	ErrorMessage string                 `json:"error,omitempty"`
	StartedAt    time.Time              `json:"startedAt"`
	CompletedAt  time.Time              `json:"completedAt"`
	Duration     time.Duration          `json:"duration"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Screenshots  []string               `json:"screenshots,omitempty"`
	Logs         []string               `json:"logs,omitempty"`
	Step         int                    `json:"step"`
	Total        int                    `json:"total"`
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusSuccess   ExecutionStatus = "success"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusTimeout   ExecutionStatus = "timeout"
	StatusPaused    ExecutionStatus = "paused"
)

// SessionData 会话数据（用于自动登录）
type SessionData struct {
	AccessToken  string                 `json:"accessToken,omitempty"`
	RefreshToken string                 `json:"refreshToken,omitempty"`
	Cookies      []Cookie               `json:"cookies,omitempty"`
	SessionData  map[string]interface{} `json:"sessionData,omitempty"`
}

// Cookie HTTP Cookie
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain,omitempty"`
	Path     string     `json:"path,omitempty"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure,omitempty"`
	HTTPOnly bool       `json:"httpOnly,omitempty"`
}

// ProgressReport progress report
type ProgressReport struct {
	ExecutionID     string          `json:"executionId"`
	WorkerID        string          `json:"workerId,omitempty"`
	ProgressCurrent int             `json:"progressCurrent"`
	ProgressTotal   int             `json:"progressTotal"`
	Message         string          `json:"message"`
	Status          ExecutionStatus `json:"status"`
	Screenshot      string          `json:"screenshot,omitempty"`
	Log             string          `json:"log,omitempty"`
	Step            int             `json:"step,omitempty"`
	Total           int             `json:"total,omitempty"`
	Timestamp       time.Time       `json:"timestamp,omitempty"`
}

// WorkerInfo Worker 信息
type WorkerInfo struct {
	WorkerID       string   `json:"workerId"`
	WorkerName     string   `json:"workerName"`
	Host           string   `json:"host,omitempty"`
	Port           int      `json:"port,omitempty"`
	Version        string   `json:"version,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	MaxConcurrency int      `json:"maxConcurrency"`
}

// WorkerRegisterRequest Worker 注册请求
type WorkerRegisterRequest struct {
	WorkerID       string   `json:"workerId" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Host           string   `json:"host,omitempty"`
	Port           int      `json:"port"`
	MaxConcurrency int      `json:"maxConcurrency"`
	Version        string   `json:"version,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
}

// WorkerRegisterResponse Worker 注册响应
type WorkerRegisterResponse struct {
	WorkerID string `json:"workerId"`
	Token    string `json:"token,omitempty"`
}

// WorkerHeartbeatRequest 心跳请求
type WorkerHeartbeatRequest struct {
	WorkerID      string `json:"workerId"`
	CurrentTasks  int    `json:"currentTasks"`
	TotalExecuted int    `json:"totalExecuted"`
	TotalFailed   int    `json:"totalFailed"`
	Status        string `json:"status"`
	// RunningExecutionIDs Phase 153 / RUNTIME-01：当前运行中的执行 ID，
	// 后端心跳通道据此为无归属执行补回写。
	RunningExecutionIDs []string `json:"runningExecutionIds,omitempty"`
}

// APIResponse API 响应
type APIResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

// ScaleDirection 扩缩容方向
type ScaleDirection string

const (
	ScaleUp   ScaleDirection = "up"
	ScaleDown ScaleDirection = "down"
)

// ScaleCommand 扩缩容指令
type ScaleCommand struct {
	CommandID   string         `json:"commandId"`
	WorkerID    string         `json:"workerId,omitempty"`
	Direction   ScaleDirection `json:"direction"`
	Concurrency int            `json:"concurrency"`
	Reason      string         `json:"reason"`
	Timestamp   int64          `json:"timestamp"`
}

// ScaleEvent 扩缩容事件
type ScaleEvent struct {
	EventID   string         `json:"eventId"`
	WorkerID  string         `json:"workerId"`
	Direction ScaleDirection `json:"direction"`
	OldValue  int            `json:"oldValue"`
	NewValue  int            `json:"newValue"`
	Reason    string         `json:"reason"`
	Timestamp time.Time      `json:"timestamp"`
}

// ========== 混合模式相关类型 ==========

// SubTaskMessage 子任务消息（继承 TaskMessage）
type SubTaskMessage struct {
	*TaskMessage                             // 嵌入原始任务消息
	ParentExecutionID string                 `json:"parentExecutionId"` // 父任务执行ID
	SubTaskIndex      int                    `json:"subTaskIndex"`      // 子任务索引
	SubTaskTotal      int                    `json:"subTaskTotal"`      // 子任务总数
	LoopItemVar       string                 `json:"loopItemVar"`       // 循环项变量名
	LoopItemData      map[string]interface{} `json:"loopItemData"`      // 循环项数据
}

// SubTaskResult 子任务执行结果（用于进度聚合）
type SubTaskResult struct {
	SubTaskIndex int                    `json:"subTaskIndex"`
	Status       ExecutionStatus        `json:"status"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	StartedAt    time.Time              `json:"startedAt"`
	CompletedAt  time.Time              `json:"completedAt"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// ProgressAggregateMessage 进度聚合消息
type ProgressAggregateMessage struct {
	ParentExecutionID string          `json:"parentExecutionId"`
	CompletedCount    int             `json:"completedCount"`
	TotalCount        int             `json:"totalCount"`
	SuccessCount      int             `json:"successCount"`
	FailureCount      int             `json:"failureCount"`
	Status            ExecutionStatus `json:"status"`
	SubTaskResults    []SubTaskResult `json:"subTaskResults,omitempty"`
	Timestamp         time.Time       `json:"timestamp"`
}

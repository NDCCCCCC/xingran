package rpa

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ============================================================================
// Phase 156 / NOCODE-03：AI 生成动作的枚举校验
//
// 需求原文：后端返回的 actions 必须通过 types.go 动作枚举校验（非法动作服务端
// 拒绝并给出可读原因）。
//
// 归属协议第 1 条：**156 不是动作全集的第三写方**。因此这里的白名单是
// 「rpa-worker/internal/types/types.go 的镜像」，不是新定义的动作集合 ——
// 任何新增动作必须先在 154 的 types.go 落地，再同步到本文件
// （ai_action_validator_test.go 有静态断言锁住这一点）。
//
// 为什么后端需要镜像而不是直接 import：
//   rpa-worker 是独立 Go module，根模块不能 import 它；且后端在「生成」环节
//   需要快速失败（不能等 Worker 执行时才报 unsupported action type）。
//
// 可读原因的设计原则（NOCODE-03「给出可读原因」）：
//   错误信息面向**一般员工**，不是开发者 —— 不出现 "enum"、"unmarshal"、
//   字段名等术语，而是说「第 2 步的动作类型「xxx」系统不认识」。
// ============================================================================

// workerActionTypes rpa-worker 动作枚举的镜像（与 types/types.go 严格同源）。
//
// 分两组标注来源，便于核对：基础动作 + ENGINE-02 扩充动作。
var workerActionTypes = map[string]string{
	// —— 基础动作（types.go 第一批）——
	"navigate":   "打开网页",
	"click":      "点击",
	"fill":       "填入",
	"select":     "选择",
	"wait":       "等待",
	"screenshot": "截图",
	"extract":    "提取内容",
	"scroll":     "滚动",
	"upload":     "上传文件",
	"download":   "下载文件",
	"evaluate":   "执行脚本",
	"waitFor":    "等待元素出现",
	"close":      "关闭页面",
	"loop":       "循环",
	"pause":      "暂停等待",
	"condition":  "条件判断",
	"autologin":  "自动登录",

	// —— ENGINE-02 扩充动作（Phase 154 差集 A/B）——
	"assert":      "检查结果",
	"hover":       "鼠标悬停",
	"doubleClick": "双击",
	"rightClick":  "右键点击",
	"drag":        "拖拽",
	"keyboard":    "按键",
	"switchTab":   "切换页面",
	"iframe":      "切换内嵌页面",
	"script":      "执行脚本",
}

// WorkerActionTypes 返回全部合法动作类型（供测试与前端文档使用）。
func WorkerActionTypes() []string {
	out := make([]string, 0, len(workerActionTypes))
	for k := range workerActionTypes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// actionTypeLabel 返回动作的中文名（错误信息用，避免向用户暴露英文枚举）。
func actionTypeLabel(t string) string {
	if label, ok := workerActionTypes[t]; ok {
		return label
	}
	return t
}

// InvalidActionError 单个非法动作的详情。
type InvalidActionError struct {
	// Index 第几个动作（从 1 开始，面向用户）。
	Index int
	// Type AI 给出的原始动作类型。
	Type string
	// Reason 可读原因（中文，零技术术语）。
	Reason string
}

func (e *InvalidActionError) Error() string {
	return fmt.Sprintf("第 %d 步：%s", e.Index, e.Reason)
}

// ActionValidationError 一次生成结果里全部非法动作的聚合错误。
type ActionValidationError struct {
	Invalid []*InvalidActionError
	// Total 本次返回的动作总数（用于「N 个动作里 M 个不认识」的表述）。
	Total int
}

func (e *ActionValidationError) Error() string {
	if len(e.Invalid) == 0 {
		return "动作校验通过"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "AI 返回的 %d 个步骤中有 %d 个无法执行：", e.Total, len(e.Invalid))
	for i, item := range e.Invalid {
		if i > 0 {
			b.WriteString("；")
		}
		b.WriteString(item.Error())
	}
	b.WriteString("。请补充更具体的任务描述后重新生成，或改用「录制」方式创建任务。")
	return b.String()
}

// ============================================================================
// NOCODE-03：AI 不可用的可辨识信号
// ============================================================================

// ErrAIDisabled AI 未启用的哨兵错误。
//
// 前端需要区分「AI 不可用」（应引导用户改用录制）与「AI 生成失败」（应重试）。
// 若两者都只是 500 + 一段中文，前端只能靠字符串匹配猜，极不可靠 ——
// 因此这里给出可 errors.Is 判定的哨兵，handler 层映射成独立的业务错误码。
var ErrAIDisabled = fmt.Errorf("AI 脚本生成未启用")

// IsAIDisabled 判定错误是否为「AI 未启用」（含包装）。
func IsAIDisabled(err error) bool {
	if err == nil {
		return false
	}
	return err == ErrAIDisabled || strings.Contains(err.Error(), ErrAIDisabled.Error())
}
type rawScriptAction struct {
	Type string `json:"type"`
}

// ValidateAIActions 校验 AI 返回的动作序列。
//
// 返回 (合法动作数, error)。error 为 *ActionValidationError 时携带逐条原因，
// handler 层据此返回 400 + 可读 message（而不是 500 让用户以为系统坏了）。
//
// 设计取舍：**不静默丢弃非法动作**。前端 mapAIActions 已经会过滤，但服务端
// 必须也拒绝 —— 否则「前端过滤」成了唯一防线，换个客户端就能把非法动作写进
// 任务脚本，Worker 执行时才炸。这正是 NOCODE-03 要求服务端校验的原因。
func ValidateAIActions(actions []json.RawMessage) (int, error) {
	if len(actions) == 0 {
		return 0, nil
	}

	invalid := make([]*InvalidActionError, 0)
	valid := 0

	for i, raw := range actions {
		idx := i + 1
		var a rawScriptAction
		if err := json.Unmarshal(raw, &a); err != nil {
			invalid = append(invalid, &InvalidActionError{
				Index:  idx,
				Reason: "这一步的格式系统无法识别，可能生成内容不完整",
			})
			continue
		}
		t := strings.TrimSpace(a.Type)
		if t == "" {
			invalid = append(invalid, &InvalidActionError{
				Index:  idx,
				Reason: "这一步缺少动作类型",
			})
			continue
		}
		if _, ok := workerActionTypes[t]; !ok {
			invalid = append(invalid, &InvalidActionError{
				Index:  idx,
				Type:   t,
				Reason: fmt.Sprintf("动作类型「%s」系统不支持", t),
			})
			continue
		}
		valid++
	}

	if len(invalid) > 0 {
		return valid, &ActionValidationError{Invalid: invalid, Total: len(actions)}
	}
	return valid, nil
}

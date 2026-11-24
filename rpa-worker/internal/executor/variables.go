package executor

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 154 / ENGINE-04：变量与数据流
//
// 三层作用域显式化（查找顺序由窄到宽，写回只写最窄层）：
//
//	Loop  —— 循环项级：每轮迭代新建，供 ${item.xxx} / ${index} 使用
//	Task  —— 任务级：task.Variables（后端下发）+ extract/evaluate 回写
//	Session —— 会话级：__sessionData / __credentials 等跨任务复用数据
//
// 表达式支持：
//   - 扁平 ${var}
//   - 嵌套 ${a.b} / ${a.b[0].c}（数组下标，负下标从尾部取）
//   - 未定义时按 executor.undefined_variable_policy：
//       strict → 动作直接失败（error）
//       empty  → 替换为 ""
//
// 敏感变量脱敏：__credentials / __sessionData 等键在日志与上报体中
// 以 "***" 替代（复用 page_manager 的 maskValue 思路，不做部分暴露）。
// ============================================================================

// SensitiveVariableKeys 敏感变量键（日志/上报前必须脱敏）。
var SensitiveVariableKeys = map[string]bool{
	types.VariableCredentials: true,
	types.VariableSessionData: true,
	"password":                true,
	"token":                   true,
	"accessToken":             true,
	"refreshToken":            true,
}

// IsSensitiveVariable 判断变量名（含嵌套路径的首段）是否敏感。
func IsSensitiveVariable(name string) bool {
	first := name
	if i := strings.IndexAny(first, ".["); i > 0 {
		first = first[:i]
	}
	return SensitiveVariableKeys[first]
}

// MaskSensitive 脱敏任意值：敏感键一律替换为 "***"。
func MaskSensitive(key string, value interface{}) interface{} {
	if IsSensitiveVariable(key) {
		return "***"
	}
	return value
}

// MaskVariables 返回脱敏后的浅拷贝（供日志与上报使用，不改原 map）。
func MaskVariables(vars map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(vars))
	for k, v := range vars {
		out[k] = MaskSensitive(k, v)
	}
	return out
}

// Scope 变量作用域层级（由窄到宽）。
type Scope int

const (
	ScopeLoop Scope = iota
	ScopeTask
	ScopeSession
)

func (s Scope) String() string {
	switch s {
	case ScopeLoop:
		return "loop"
	case ScopeTask:
		return "task"
	case ScopeSession:
		return "session"
	default:
		return "unknown"
	}
}

// VariableScope 三层作用域容器。
type VariableScope struct {
	Loop    map[string]interface{}
	Task    map[string]interface{}
	Session map[string]interface{}
}

// NewVariableScope 从任务变量构造作用域（会话级数据保留原键，任务级与其
// 相同来源，但写回分离——extract 结果不回灌 __sessionData）。
func NewVariableScope(taskVars map[string]interface{}) *VariableScope {
	vs := &VariableScope{
		Loop:    map[string]interface{}{},
		Task:    map[string]interface{}{},
		Session: map[string]interface{}{},
	}
	for k, v := range taskVars {
		if IsSensitiveVariable(k) || k == types.VariableCredentials || k == types.VariableSessionData {
			vs.Session[k] = v
			vs.Task[k] = v
			continue
		}
		vs.Task[k] = v
	}
	return vs
}

// Flatten 合并成单 map（宽层覆盖窄层，实际使用时窄层优先，故这里反转）。
func (vs *VariableScope) Flatten() map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range vs.Session {
		out[k] = v
	}
	for k, v := range vs.Task {
		out[k] = v
	}
	for k, v := range vs.Loop {
		out[k] = v
	}
	return out
}

// Lookup 按 Loop → Task → Session 顺序查找，返回 (值, 所在层, 是否命中)。
func (vs *VariableScope) Lookup(path string) (interface{}, Scope, bool) {
	if v, ok := lookupIn(vs.Loop, path); ok {
		return v, ScopeLoop, true
	}
	if v, ok := lookupIn(vs.Task, path); ok {
		return v, ScopeTask, true
	}
	if v, ok := lookupIn(vs.Session, path); ok {
		return v, ScopeSession, true
	}
	return nil, ScopeTask, false
}

// Set 写入指定层（extract 默认写 Task；循环项由 Loop 层负责）。
func (vs *VariableScope) Set(scope Scope, path string, value interface{}) error {
	var target map[string]interface{}
	switch scope {
	case ScopeLoop:
		target = vs.Loop
	case ScopeSession:
		target = vs.Session
	default:
		target = vs.Task
	}
	return setPath(target, path, value)
}

// BreakLoopScope 每轮迭代结束重置循环项级（避免上一轮 item 泄漏）。
func (vs *VariableScope) BreakLoopScope() { vs.Loop = map[string]interface{}{} }

// varExprRe 匹配 ${...}，非贪婪；支持嵌套路径。
var varExprRe = regexp.MustCompile(`\$\{([^{}]+)\}`)

// pathSeg 路径段：key 或 [index]。
type pathSeg struct {
	key   string
	index int
	isIdx bool
}

// parsePath 解析 "a.b[0].c" → [key a, key b, index 0, key c]。
func parsePath(path string) ([]pathSeg, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil, fmt.Errorf("空变量路径")
	}
	var segs []pathSeg
	for i := 0; i < len(p); {
		switch p[i] {
		case '.':
			i++
			if i >= len(p) {
				return nil, fmt.Errorf("变量路径以 . 结尾: %q", path)
			}
		case '[':
			end := strings.IndexByte(p[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("变量路径括号未闭合: %q", path)
			}
			raw := strings.TrimSpace(p[i+1 : i+end])
			idx, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("变量路径下标非法 %q: %w", raw, err)
			}
			segs = append(segs, pathSeg{index: idx, isIdx: true})
			i += end + 1
			if i < len(p) && p[i] == '.' {
				i++
			}
			continue
		default:
			j := i
			for j < len(p) && p[j] != '.' && p[j] != '[' {
				j++
			}
			key := p[i:j]
			if key == "" {
				return nil, fmt.Errorf("变量路径含空段: %q", path)
			}
			segs = append(segs, pathSeg{key: key})
			i = j
		}
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("变量路径无有效段: %q", path)
	}
	return segs, nil
}

// lookupIn 在 map 中按嵌套路径取值。
func lookupIn(root map[string]interface{}, path string) (interface{}, bool) {
	if root == nil {
		return nil, false
	}
	if v, ok := root[path]; ok {
		return v, true
	}
	segs, err := parsePath(path)
	if err != nil {
		return nil, false
	}
	var cur interface{} = root
	for _, seg := range segs {
		if seg.isIdx {
			arr, ok := toSlice(cur)
			if !ok {
				return nil, false
			}
			idx := seg.index
			if idx < 0 {
				idx += len(arr)
			}
			if idx < 0 || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
			continue
		}
		m, ok := toMap(cur)
		if !ok {
			return nil, false
		}
		next, ok := m[seg.key]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// setPath 按嵌套路径写回（中间节点缺失时自动创建 map / 数组）。
func setPath(root map[string]interface{}, path string, value interface{}) error {
	segs, err := parsePath(path)
	if err != nil {
		return err
	}
	var cur interface{} = root
	for i, seg := range segs {
		last := i == len(segs)-1
		if seg.isIdx {
			arr, ok := cur.([]interface{})
			if !ok {
				return fmt.Errorf("路径 %q 的第 %d 段期望数组，实际 %T", path, i, cur)
			}
			idx := seg.index
			if idx < 0 {
				idx += len(arr)
			}
			if idx < 0 || idx >= len(arr) {
				return fmt.Errorf("路径 %q 的第 %d 段下标越界", path, i)
			}
			if last {
				arr[idx] = value
				return nil
			}
			cur = arr[idx]
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return fmt.Errorf("路径 %q 的第 %d 段期望对象，实际 %T", path, i, cur)
		}
		if last {
			m[seg.key] = value
			return nil
		}
		next, exists := m[seg.key]
		if !exists || next == nil {
			// 依据下一段形态预建容器。
			if segs[i+1].isIdx {
				next = []interface{}{}
			} else {
				next = map[string]interface{}{}
			}
			m[seg.key] = next
		}
		cur = next
	}
	return nil
}

// toMap 归一化 map（兼容 map[string]string 等弱类型来源）。
func toMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[string]string:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out, true
	default:
		return nil, false
	}
}

// toSlice 归一化数组。
func toSlice(v interface{}) ([]interface{}, bool) {
	switch a := v.(type) {
	case []interface{}:
		return a, true
	case []string:
		out := make([]interface{}, len(a))
		for i, s := range a {
			out[i] = s
		}
		return out, true
	default:
		return nil, false
	}
}

// ResolvePath 解析单个路径（不带 ${} 包裹），未找到返回 (nil,false)。
func ResolvePath(vars map[string]interface{}, path string) (interface{}, bool) {
	return lookupIn(vars, path)
}

// ResolveTemplate 替换字符串中的 ${expr}。
//
// policy: config.UndefinedVariableStrict / config.UndefinedVariableEmpty。
// 严格模式返回 error（且错误信息里给出未定义路径，便于定位录制脚本缺参）。
func ResolveTemplate(s string, vars map[string]interface{}, policy string) (string, error) {
	var firstErr error
	out := varExprRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := strings.TrimSpace(match[2 : len(match)-1])

		// 支持 ${a.b[0].c:default} 形式（冒号后的字面量作为兜底值）。
		defVal := ""
		hasDefault := false
		if i := strings.LastIndex(inner, ":"); i > 0 {
			if j := strings.Index(inner, "["); j < 0 || strings.Index(inner[i:], "]") < 0 {
				defVal = inner[i+1:]
				inner = inner[:i]
				hasDefault = true
			}
		}

		v, ok := lookupIn(vars, inner)
		if !ok {
			if hasDefault {
				return defVal
			}
			if policy == config.UndefinedVariableStrict {
				if firstErr == nil {
					firstErr = fmt.Errorf("变量未定义: ${%s}", inner)
				}
				return match
			}
			return ""
		}
		if IsSensitiveVariable(inner) {
			return "***"
		}
		return fmt.Sprintf("%v", v)
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

// ResolveValue 递归替换 string / []interface{} / map[string]interface{} 中的
// ${...}；用于参数字典整棵树的求值。
func ResolveValue(v interface{}, vars map[string]interface{}, policy string) (interface{}, error) {
	switch val := v.(type) {
	case string:
		return ResolveTemplate(val, vars, policy)
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			r, err := ResolveValue(item, vars, policy)
			if err != nil {
				return nil, err
			}
			out[i] = r
		}
		return out, nil
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys) // 确定性求值顺序，便于测试与排错
		for _, k := range keys {
			r, err := ResolveValue(val[k], vars, policy)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	default:
		return v, nil
	}
}

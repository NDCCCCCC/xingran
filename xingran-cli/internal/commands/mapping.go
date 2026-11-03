package commands

import (
	"strings"
)

// pathArgMarkers Swagger 路径参数的两种写法。
const (
	colonPrefix = ":"
	braceOpen   = "{"
)

// Mapping 将 API 路径映射为三级命令（module/resource/action）。
//
// 规则（按 lead 约定）：
//   POST /system/users/list          → system user list
//   POST /system/users               → system user create
//   POST /system/users/:id           → system user get
//   POST /system/users/:id/update    → system user update
//   POST /system/users/:id/delete    → system user delete
//   POST /system/users/:id/<verb>    → system user <verb>
//
// 资源名去复数（尾缀 s）；不规则路径经 OverrideMap 白名单修正。
type Mapping struct {
	Module       string
	ResourcePath []string // 多级名词段（已单数化），如 [ad-domain config]
	Action       string
	PathArgs     []string // 路径参数名（按出现顺序）
}

// OverrideMap 不规则路径的手工白名单（key = METHOD + " " + 规范化 API 路径）。
// 由生成器在调用 MapEndpoint 后追加修正；资源名不变化只需写 action 修正。
var OverrideMap = map[string]Mapping{}

// NormalizePathSegs 拆分并规范化 API 路径段：去 /api/v1 前缀、去首尾斜杠。
func NormalizePathSegs(apiPath string) []string {
	p := strings.TrimPrefix(apiPath, "/api/v1")
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	segs := strings.Split(p, "/")
	// 路径参数两种写法统一保留原样（:id / {id}），构建 Use 时再处理
	return segs
}

// isPathArg 判断段是否为路径参数。
func isPathArg(seg string) bool {
	return strings.HasPrefix(seg, colonPrefix) ||
		(strings.HasPrefix(seg, braceOpen) && strings.HasSuffix(seg, "}"))
}

// pathArgName 提取路径参数名。
func pathArgName(seg string) string {
	if strings.HasPrefix(seg, colonPrefix) {
		return seg[1:]
	}
	return strings.TrimSuffix(strings.TrimPrefix(seg, braceOpen), "}")
}

// irregularPlural 不规则复数映射（-ies 类与手工修正）。
var irregularPlural = map[string]string{
	"categories":  "category",
	"discoveries": "discovery",
	"entries":     "entry",
	"libraries":   "library",
	"queries":     "query",
}

// Singular 资源名去复数：不规则映射优先；尾缀 s 去除；白名单保持不变。
// ad-domain / dict-data / business 等不以 s 结尾或非复数语义的词自动不变。
func Singular(name string) string {
	if v, ok := irregularPlural[name]; ok {
		return v
	}
	const keep = "business,status,success,process,class,address"
	for _, k := range strings.Split(keep, ",") {
		if name == k {
			return name
		}
	}
	if strings.HasSuffix(name, "s") && len(name) > 3 {
		return name[:len(name)-1]
	}
	return name
}

// MapEndpoint 由 API 路径推导多级资源映射。
//
// 算法：从 rest 中剥离路径参数段得到静态段 static：
//   - static 只剩 1 段：resourcePath = [Singular(该段)]；有参数 → get，
//     否则 POST → create / GET → list
//   - static 多段：resourcePath = Singular 化的 static[:-1]，action = static[-1]
//     （末段恒为 action；/dashboards/:id/versions/:versionId/restore →
//     dashboard version restore <id> <versionId>）
//
// 语义约定：后端路由「路径中间段均为资源名词，末段为动作/子资源」。
func MapEndpoint(method, apiPath string) Mapping {
	segs := NormalizePathSegs(apiPath)
	if len(segs) == 0 {
		return Mapping{}
	}
	m := Mapping{Module: segs[0]}

	rest := segs[1:]
	if len(rest) == 0 {
		m.ResourcePath = []string{"root"}
		m.Action = strings.ToLower(method)
		return m
	}

	// 收集路径参数与静态段
	var static []string
	for _, s := range rest {
		if isPathArg(s) {
			m.PathArgs = append(m.PathArgs, pathArgName(s))
		} else {
			static = append(static, s)
		}
	}

	hasArgs := len(m.PathArgs) > 0
	// 参数结尾（rest 末段是路径参数）→ 动作恒为 get，名词段取全部静态段
	if hasArgs && isPathArg(rest[len(rest)-1]) {
		for _, s := range static {
			m.ResourcePath = append(m.ResourcePath, Singular(s))
		}
		m.Action = "get"
	} else {
		switch len(static) {
		case 0:
			// 全参数路径（罕见）
			m.ResourcePath = []string{"item"}
			m.Action = "get"
		case 1:
			m.ResourcePath = []string{Singular(static[0])}
			if strings.EqualFold(method, "GET") {
				m.Action = "list"
			} else {
				m.Action = "create"
			}
		default:
			for _, s := range static[:len(static)-1] {
				m.ResourcePath = append(m.ResourcePath, Singular(s))
			}
			m.Action = static[len(static)-1]
		}
	}

	// 白名单修正（key 不含前缀差异：METHOD + path）
	if ov, ok := OverrideMap[method+" "+apiPath]; ok {
		if ov.Action != "" {
			m.Action = ov.Action
		}
		if len(ov.ResourcePath) > 0 {
			m.ResourcePath = ov.ResourcePath
		}
	}
	return m
}

// RenderPath 用位置参数值还原 API 路径。
// pathVals 与 meta.PathArgs 顺序一致。
func RenderPath(apiPath string, pathArgs, pathVals []string) string {
	out := apiPath
	for i, name := range pathArgs {
		val := "<" + name + ">"
		if i < len(pathVals) {
			val = pathVals[i]
		}
		// 两种占位写法都替换
		out = replaceArg(out, ":"+name, val)
		out = replaceArg(out, "{"+name+"}", val)
	}
	return out
}

func replaceArg(s, old, new string) string {
	// 逐段替换避免前缀误伤（:id vs :idName）
	segs := strings.Split(s, "/")
	for i, seg := range segs {
		if seg == old || (strings.HasPrefix(seg, "{") && strings.TrimSuffix(strings.TrimPrefix(seg, "{"), "}") == old[1:] && old[0] == ':') {
			segs[i] = new
		}
	}
	return strings.Join(segs, "/")
}

// Package commands 提供三级命令树（xingran <module> <resource> <action>）的
// 运行时构建框架：生成器（tools/cmdgen）从 Swagger 元数据产出各模块的
// EndpointMeta 数据表（cmd/<module>_cmds.go），本包负责把数据表装配成
// cobra 命令树并执行。
//
// 设计约定：
//   - 本包不依赖 cmd 包（避免循环）：执行入口 RunFunc 由 cmd 层注入
//   - 字段中文说明词典由 fielddocs.go（crypto-dev 维护）经 init() 向
//     FieldDocLookup / EnumHintLookup 赋值；未赋值时退化为「字段名+类型」
package commands

import "context"

// EndpointMeta 一个端点的完整元数据（由 tools/cmdgen 生成）。
type EndpointMeta struct {
	Module       string   // 模块：system / operations / ...
	ResourcePath []string // 多级资源路径（名词段，已单数化）：[ad-domain config] / [user]
	Action       string   // 动作：list / get / create / update / delete / <verb>
	Method       string   // HTTP 方法（默认 POST）
	APIPath      string   // 完整 API 路径（含 /api/v1 前缀），执行与 help 展示用

	Summary     string // Swagger summary（Short）
	Description string // Swagger description（Long 主体）
	Tags        string // 主 tag（中文模块名，取 tags[0]）

	Flags    []FlagSpec // body/query 参数 → flag
	PathArgs []string   // 路径参数名（按出现顺序 → 位置参数）
	Kind     string     // ""=JSON | upload(multipart 文件上传) | download(二进制流下载)
}

// 端点 Kind 常量。
const (
	KindJSON     = ""
	KindUpload   = "upload"   // POST multipart，flag: --file
	KindDownload = "download" // 响应为文件流，flag: --out
)

// ModuleNames 模块 → 中文固定名（help 展示用；未知模块回退 tags 或模块名）。
// 注意：运维模块的后端路由前缀是 /ops（非 operations）。
var ModuleNames = map[string]string{
	"system":     "系统管理",
	"ops":        "运维管理",
	"operations": "运维管理",
	"network":    "网络管理",
	"monitor":    "监控管理",
	"workorder":  "工单管理",
	"duty":       "值班管理",
	"scheduler":  "调度管理",
	"knowledge":  "知识库管理",
	"vdi":        "VDI 管理",
	"rpa":        "RPA 管理",
	"agent":      "Agent 管理",
	"asset":      "资产管理",
	"auth":       "认证",
}

// ModuleDisplayName 模块展示名。
func ModuleDisplayName(module, fallbackTag string) string {
	if n, ok := ModuleNames[module]; ok {
		return n
	}
	if fallbackTag != "" {
		return fallbackTag
	}
	return module
}

// FlagSpec 一个参数的规格。
type FlagSpec struct {
	Name     string // flag 名（body 字段名，保持后端命名如 pageSize）
	Type     string // string|int|bool|float|json（数组/对象）
	Required bool   // Swagger/definitions required 标注
	Desc     string // 中文说明（FieldDoc 词典命中；否则空）
	Enum     string // 枚举提示（EnumHint 词典命中；否则空）
	In       string // body | query
}

// RunFunc 端点执行入口（cmd 层注入：组装 client / dry-run）。
// pathVals 与 meta.PathArgs 一一对应；flags 为已解析的 flag 值。
// 返回 (result, err) 由 cmd 层统一走 output.Emit 分类退出码。
type RunFunc func(ctx context.Context, m *EndpointMeta, flags map[string]any, pathVals []string) (result any, err error)

// EmitFunc 输出函数（cmd 层注入，包装 output.Emit 并映射退出码）。
// 返回的 error 由 cobra 透传（通常为静默哨兵错误，退出码已定）。
type EmitFunc func(result any, err error) error

// FieldDescription 返回字段的展示说明：
// 优先 FieldDocWithEnum（fielddocs.go 词典，crypto-dev 直接导出，一行渲染
// "说明（枚举: ...）"），否则退化为「字段名」。
func FieldDescription(f FlagSpec) string {
	if d := FieldDocWithEnum(f.Name); d != "" && d != f.Name {
		return d
	}
	if f.Desc != "" {
		return f.Desc
	}
	return f.Name
}

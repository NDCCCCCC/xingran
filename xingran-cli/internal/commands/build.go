package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// rawJSON 把 JSON 文本包装为 RawMessage（非法 JSON 在校验阶段拦截）。
func rawJSON(s string) json.RawMessage { return json.RawMessage(s) }

// BuildModule 把一个模块的 EndpointMeta 数据表装配成多级命令树：
//
//	<module>                          （Short = 中文模块名；Long 列资源树与 api 兜底说明）
//	└── <resource>[/subresource...]   （多级名词资源，Long 列该层动作）
//	    └── <action>                  （Short = Swagger summary；Long = description + API + 参数表）
//
// run 由 cmd 层注入（组装 HTTP client、--dry-run）；
// emit 由 cmd 层注入（统一 output.Emit + 退出码哨兵）。
func BuildModule(metas []EndpointMeta, run RunFunc, emit EmitFunc) *cobra.Command {
	if len(metas) == 0 {
		return &cobra.Command{Use: "empty", Short: "（无端点）"}
	}
	module := metas[0].Module
	moduleName := ModuleDisplayName(module, metas[0].Tags)

	root := &cobra.Command{
		Use:   module,
		Short: ModuleDisplayName(module, "") + "（" + module + " 模块）",
		Long: fmt.Sprintf(`%s —— %s 模块命令集。

资源树（xingran %s <resource...> <action>）：
%s

提示：以上命令由 Swagger 元数据生成，路径即后端 API 的直译。
复杂/未覆盖场景可用万能兜底：xingran api <path> [json-body]（任意端点）。`,
			moduleName, module, module, formatResourceTree(metas)),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	buildTree(root, metas, run, emit)
	return root
}

// treeNode 资源树节点。
type treeNode struct {
	children map[string]*treeNode
	leafs    map[string]EndpointMeta // action → meta
	tag      string
}

func newTreeNode() *treeNode {
	return &treeNode{children: map[string]*treeNode{}, leafs: map[string]EndpointMeta{}}
}

// buildTree 把 metas 按 ResourcePath 插入树并把树展开为 cobra 命令。
func buildTree(root *cobra.Command, metas []EndpointMeta, run RunFunc, emit EmitFunc) {
	top := newTreeNode()
	for _, m := range metas {
		node := top
		for _, seg := range m.ResourcePath {
			child, ok := node.children[seg]
			if !ok {
				child = newTreeNode()
				node.children[seg] = child
			}
			node = child
		}
		if _, dup := node.leafs[m.Action]; dup {
			continue // 同名冲突由生成器消歧后不应出现；防御性跳过
		}
		node.leafs[m.Action] = m
	}
	// 展开（先排序保证 help 稳定）
	for _, name := range sortedKeys(top.children) {
		root.AddCommand(buildResourceCmd(name, top.children[name], metas[0].Module, run, emit))
	}
}

func sortedKeys[mT any](m map[string]mT) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildResourceCmd 递归构建资源层命令（子资源 / 叶子动作）。
func buildResourceCmd(name string, node *treeNode, module string, run RunFunc, emit EmitFunc) *cobra.Command {
	prefix := strings.Join(append(pathOf(node), name), " ")
	tag := firstTag(node)
	cmd := &cobra.Command{
		Use:   name,
		Short: tag + " / " + name,
		Long: fmt.Sprintf(`%s / %s 资源命令（%s 模块）。

可用内容：
%s

提示：每个动作的详细参数见 xingran %s %s <action> --help。`,
			tag, prefix, module, treeNodeSummary(node), module, prefix),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// 子资源与叶子动作可能同名（如 auth 下的 captcha 既是资源又是动作）。
	// 同名时合并为同一个 cobra 命令：既设 RunE（叶子端点），又保留子命令，
	// 否则 AddCommand 两次同名会覆盖，导致其中一个端点不可达。
	for _, sub := range sortedKeys(node.children) {
		childCmd := buildResourceCmd(sub, node.children[sub], module, run, emit)
		if leafMeta, ok := node.leafs[sub]; ok {
			applyLeafBehavior(childCmd, leafMeta, run, emit)
		}
		cmd.AddCommand(childCmd)
	}
	for _, a := range sortedKeys(node.leafs) {
		if _, isChild := node.children[a]; isChild {
			continue // 已在上方合并
		}
		cmd.AddCommand(buildActionCmd(node.leafs[a], run, emit))
	}
	return cmd
}

// applyLeafBehavior 把端点行为应用到已有资源命令（同名资源/动作合并场景）。
// 保留原有子命令，同时让该命令自身可执行（cobra 支持 RunE + AddCommand 共存）。
func applyLeafBehavior(cmd *cobra.Command, m EndpointMeta, run RunFunc, emit EmitFunc) {
	use := m.Action
	for _, p := range m.PathArgs {
		use += " <" + p + ">"
	}
	cmd.Use = use
	cmd.Short = m.Summary
	flags := sortedFlags(m.Flags)
	cmd.Long = buildActionLong(m, flags) + "\n\n本命令既可直接执行（见上），也可携带子命令访问其下级资源。"
	cmd.Example = buildExample(m, flags)
	cmd.Args = cobra.ExactArgs(len(m.PathArgs))
	cmd.RunE = func(c *cobra.Command, args []string) error {
		flagVals := map[string]any{}
		var missing []string
		for _, f := range flags {
			v, changed := flagValue(c, f.Name)
			if f.Required && (!changed || isEmptyValue(v)) {
				missing = append(missing, "--"+f.Name)
				continue
			}
			if changed {
				flagVals[f.Name] = v
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("缺少必填参数: %s", strings.Join(missing, " "))
		}
		result, err := run(c.Context(), &m, flagVals, args)
		return emit(result, err)
	}
	registerFlags(cmd, flags)
}

// pathOf 回溯节点路径（构建 Long 文案用）。
func pathOf(n *treeNode) []string {
	// 树节点没有父指针；Long 里的 prefix 用调用方拼好的 name 即可，
	// 这里返回空以保持签名简单。
	return nil
}

func firstTag(n *treeNode) string {
	for _, m := range n.leafs {
		if m.Tags != "" {
			return m.Tags
		}
	}
	for _, name := range sortedKeys(n.children) {
		if t := firstTag(n.children[name]); t != "" {
			return t
		}
	}
	return ""
}

// treeNodeSummary 资源层 Long 的可用内容清单。
func treeNodeSummary(n *treeNode) string {
	var lines []string
	for _, a := range sortedKeys(n.leafs) {
		m := n.leafs[a]
		use := a
		for _, p := range m.PathArgs {
			use += " <" + p + ">"
		}
		line := "  " + use
		if m.Summary != "" {
			line += "  " + m.Summary
		}
		lines = append(lines, line)
	}
	for _, name := range sortedKeys(n.children) {
		lines = append(lines, "  "+name+"/  （子资源）")
	}
	return strings.Join(lines, "\n")
}

func buildActionCmd(m EndpointMeta, run RunFunc, emit EmitFunc) *cobra.Command {
	use := m.Action
	for _, p := range m.PathArgs {
		use += " <" + p + ">"
	}
	// 参数顺序：必填在前，其余按名称字母序
	flags := sortedFlags(m.Flags)
	cmd := &cobra.Command{
		Use:     use,
		Short:   m.Summary,
		Long:    buildActionLong(m, flags),
		Example: buildExample(m, flags),
		Args:               cobra.ExactArgs(len(m.PathArgs)),
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(c *cobra.Command, args []string) error {
			flagVals := map[string]any{}
			var missing []string
			for _, f := range flags {
				v, changed := flagValue(c, f.Name)
				if f.Required && (!changed || isEmptyValue(v)) {
					missing = append(missing, "--"+f.Name)
					continue
				}
				if changed {
					flagVals[f.Name] = v
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("缺少必填参数: %s", strings.Join(missing, " "))
			}
			result, err := run(c.Context(), &m, flagVals, args)
			return emit(result, err)
		},
	}
	registerFlags(cmd, flags)
	return cmd
}

// formatResourceTree 模块 Long 的资源树清单（多级缩进）。
func formatResourceTree(metas []EndpointMeta) string {
	tree := newTreeNode()
	for _, m := range metas {
		node := tree
		for _, seg := range m.ResourcePath {
			child, ok := node.children[seg]
			if !ok {
				child = newTreeNode()
				node.children[seg] = child
			}
			node = child
		}
		if _, dup := node.leafs[m.Action]; !dup {
			node.leafs[m.Action] = m
		}
	}
	var walk func(n *treeNode, depth int, b *strings.Builder)
	walk = func(n *treeNode, depth int, b *strings.Builder) {
		for _, name := range sortedKeys(n.children) {
			indent := strings.Repeat("  ", depth+1)
			child := n.children[name]
			actions := sortedKeys(child.leafs)
			line := indent + name + "/"
			if len(actions) > 0 {
				line += "  (" + strings.Join(actions, ", ") + ")"
			}
			b.WriteString(line + "\n")
			walk(child, depth+1, b)
		}
	}
	var b strings.Builder
	walk(tree, 0, &b)
	return strings.TrimRight(b.String(), "\n")
}

// sortedFlags 必填在前，其余按名称字母序。
func sortedFlags(flags []FlagSpec) []FlagSpec {
	out := make([]FlagSpec, len(flags))
	copy(out, flags)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Required != out[j].Required {
			return out[i].Required
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// buildActionLong Long 文本：description + 对应 API + 参数表。
func buildActionLong(m EndpointMeta, flags []FlagSpec) string {
	var b strings.Builder
	if m.Description != "" {
		b.WriteString(m.Description)
		b.WriteString("\n\n")
	} else if m.Summary != "" {
		b.WriteString(m.Summary)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "对应 API: %s %s\n", m.Method, m.APIPath)
	if len(m.PathArgs) > 0 {
		fmt.Fprintf(&b, "路径参数: %s\n", strings.Join(m.PathArgs, ", "))
	}
	if len(flags) > 0 {
		b.WriteString("\n参数:\n")
		for _, f := range flags {
			req := ""
			if f.Required {
				req = " 必填"
			}
			fmt.Fprintf(&b, "  --%-22s %-10s%s %s\n", f.Name, flagTypeLabel(f.Type), req, FieldDescription(f))
		}
	}
	b.WriteString("\n万能兜底等价：xingran api " + m.APIPath + " '<json-body>'\n")
	return strings.TrimRight(b.String(), "\n")
}

// buildExample 可复制示例：
//   - list 类：--current 1 --pageSize 10（分页惯例）
//   - get/delete 类：路径占位 + 少量参数
//   - create 类：必填字段 + 合理示例值
func buildExample(m EndpointMeta, flags []FlagSpec) string {
	var b strings.Builder
	b.WriteString("xingran " + m.Module)
	for _, r := range m.ResourcePath {
		b.WriteString(" " + r)
	}
	b.WriteString(" " + m.Action)
	for _, p := range m.PathArgs {
		b.WriteString(" <" + p + ">")
	}

	switch {
	case m.Action == "list":
		if hasFlag(flags, "current") {
			b.WriteString(" --current 1")
		}
		if hasFlag(flags, "pageSize") {
			b.WriteString(" --pageSize 10")
		}
		if !hasFlag(flags, "current") && !hasFlag(flags, "pageSize") {
			appendOptionalExamples(&b, flags, 2)
		}
	case m.Action == "create":
		appendRequiredExamples(&b, flags)
	default: // get/update/delete/动词类：带 1 个可选参数示例
		appendOptionalExamples(&b, flags, 1)
	}
	return b.String()
}

func hasFlag(flags []FlagSpec, name string) bool {
	for _, f := range flags {
		if f.Name == name {
			return true
		}
	}
	return false
}

func appendRequiredExamples(b *strings.Builder, flags []FlagSpec) {
	n := 0
	for _, f := range flags {
		if !f.Required {
			continue
		}
		b.WriteString(" --" + f.Name + " " + exampleValue(f))
		n++
	}
	if n == 0 {
		appendOptionalExamples(b, flags, 2)
	}
}

func appendOptionalExamples(b *strings.Builder, flags []FlagSpec, max int) {
	n := 0
	for _, f := range flags {
		if f.Required {
			continue
		}
		b.WriteString(" --" + f.Name + " " + exampleValue(f))
		n++
		if n >= max {
			break
		}
	}
}

func exampleValue(f FlagSpec) string {
	switch f.Type {
	case "int":
		return "1"
	case "bool":
		return "true"
	case "float":
		return "1.5"
	case "json":
		return `'[{"k":1}]'`
	default:
		return `"xxx"`
	}
}

func flagTypeLabel(t string) string {
	if t == "json" {
		return "json-array"
	}
	return t
}

// registerFlags 按 FlagSpec 注册 flag（保持后端字段名，如 --pageSize）。
func registerFlags(cmd *cobra.Command, flags []FlagSpec) {
	fs := cmd.Flags()
	for _, f := range flags {
		usage := FieldDescription(f)
		if f.Required {
			usage = "必填，" + usage
		}
		switch f.Type {
		case "int":
			fs.Int(f.Name, 0, usage)
		case "bool":
			fs.Bool(f.Name, false, usage)
		case "float":
			fs.Float64(f.Name, 0, usage)
		default: // string / json
			fs.String(f.Name, "", usage)
		}
	}
}

// flagValue 读取已注册 flag 的值与"是否被显式设置"。
func flagValue(c *cobra.Command, name string) (any, bool) {
	fl := c.Flags().Lookup(name)
	if fl == nil {
		return nil, false
	}
	switch fl.Value.Type() {
	case "int":
		v, _ := c.Flags().GetInt(name)
		return v, fl.Changed
	case "bool":
		v, _ := c.Flags().GetBool(name)
		return v, fl.Changed
	case "float64":
		v, _ := c.Flags().GetFloat64(name)
		return v, fl.Changed
	default:
		v, _ := c.Flags().GetString(name)
		return v, fl.Changed
	}
}

func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case string:
		return t == ""
	case nil:
		return true
	default:
		return false
	}
}

// BuildBody 由 flags 组装请求体（json 类型按 RawMessage 原样透传）。
func BuildBody(flags map[string]any, specs []FlagSpec) map[string]any {
	body := map[string]any{}
	for _, f := range specs {
		if v, ok := flags[f.Name]; ok {
			if f.Type == "json" {
				if s, isStr := v.(string); isStr {
					body[f.Name] = rawJSON(s)
					continue
				}
			}
			body[f.Name] = v
		}
	}
	return body
}

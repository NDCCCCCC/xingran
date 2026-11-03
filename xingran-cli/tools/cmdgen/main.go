// cmdgen：从仓库 docs/docs.go（swaggo Swagger spec）生成三级命令树数据表
// （cmd/<module>_cmds.go）。框架在 internal/commands，本工具只产出数据。
//
// 用法：
//
//	go run ./tools/cmdgen -module system          # 生成单个模块（默认 system）
//	go run ./tools/cmdgen -module all             # 全部模块
//	go run ./tools/cmdgen -docs ../docs/docs.go   # 指定 docs 路径
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/commands"
)

// containerParamNames body 参数中的容器名（$ref 顶层包装），不作为 flag。
var containerParamNames = map[string]bool{"request": true, "body": true, "req": true, "data": true}

func main() {
	docs := flag.String("docs", client.DefaultDocsPath(), "docs.go 路径")
	module := flag.String("module", "system", "要生成的模块名（all=全部）")
	outDir := flag.String("out", "cmd", "输出目录")
	flag.Parse()

	eps, basePath, err := client.LoadSchema(*docs)
	if err != nil {
		fatal(err)
	}
	_ = basePath

	// 转换 + 分组
	byModule := map[string][]commands.EndpointMeta{}
	skipped := 0
	for i := range eps {
		e := eps[i]
		meta, ok := toMeta(e)
		if !ok {
			skipped++
			continue
		}
		if *module != "all" && meta.Module != *module {
			continue
		}
		byModule[meta.Module] = append(byModule[meta.Module], meta)
	}

	var mods []string
	for m := range byModule {
		mods = append(mods, m)
	}
	sort.Strings(mods)

	for _, m := range mods {
		metas := disambiguate(byModule[m])
		sort.Slice(metas, func(i, j int) bool {
			if strings.Join(metas[i].ResourcePath, "/") != strings.Join(metas[j].ResourcePath, "/") {
				return strings.Join(metas[i].ResourcePath, "/") < strings.Join(metas[j].ResourcePath, "/")
			}
			return metas[i].Action < metas[j].Action
		})
		// 防回归自检：端点总数 == 唯一叶子命令数（冲突消歧后必须一一对应）
		if err := verifyLeafCount(metas); err != nil {
			fatal(err)
		}
		out := fmt.Sprintf("%s/%s_cmds.go", *outDir, m)
		if err := os.WriteFile(out, []byte(renderModule(m, metas)), 0o644); err != nil {
			fatal(err)
		}
		fmt.Printf("生成 %s：%d 端点（叶子命令校验通过）\n", out, len(metas))
	}
	fmt.Printf("完成（模块 %d 个；无法映射跳过 %d 个端点）\n", len(mods), skipped)
}

// disambiguate 同一资源路径下同名 action 的消歧：action 追加 "-<method>"。
// 典型：GET/POST 共用 /:id/versions（列版本 vs 创建版本快照）。
func disambiguate(metas []commands.EndpointMeta) []commands.EndpointMeta {
	seen := map[string]int{}
	out := make([]commands.EndpointMeta, len(metas))
	for i, m := range metas {
		key := strings.Join(m.ResourcePath, "/") + "|" + m.Action
		seen[key]++
		if seen[key] > 1 {
			m.Action = m.Action + "-" + strings.ToLower(m.Method)
		}
		out[i] = m
	}
	// 二次处理：被消歧的后项若与首项撞名（首项未消歧），首项也加后缀
	counts := map[string]int{}
	for _, m := range out {
		counts[strings.Join(m.ResourcePath, "/")+"|"+m.Action]++
	}
	for i := range out {
		key := strings.Join(out[i].ResourcePath, "/") + "|" + out[i].Action
		if counts[key] > 1 {
			out[i].Action = out[i].Action + "-" + strings.ToLower(out[i].Method)
			counts[key]--
		}
	}
	return out
}

// verifyLeafCount 端点总数必须等于唯一叶子命令数（防端点丢失，lead 要求的自检）。
func verifyLeafCount(metas []commands.EndpointMeta) error {
	leafs := map[string]bool{}
	for _, m := range metas {
		leafs[strings.Join(m.ResourcePath, "/")+"|"+m.Action] = true
	}
	if len(leafs) != len(metas) {
		var dups []string
		counts := map[string]int{}
		for _, m := range metas {
			k := strings.Join(m.ResourcePath, "/") + "|" + m.Action
			counts[k]++
			if counts[k] == 2 {
				dups = append(dups, k)
			}
		}
		return fmt.Errorf("端点数 %d != 唯一叶子命令数 %d，冲突: %v", len(metas), len(leafs), dups)
	}
	return nil
}

func toMeta(e client.EndpointInfo) (commands.EndpointMeta, bool) {
	mp := commands.MapEndpoint(e.Method, e.Path)
	if mp.Module == "" || len(mp.ResourcePath) == 0 || mp.Action == "" {
		return commands.EndpointMeta{}, false
	}
	// API 路径统一带 /api/v1 前缀（client 执行时也会自动补，这里为展示一致性）
	apiPath := e.Path
	if !strings.HasPrefix(apiPath, "/api/v1/") {
		apiPath = "/api/v1" + apiPath
	}

	meta := commands.EndpointMeta{
		Module:       mp.Module,
		ResourcePath: mp.ResourcePath,
		Action:       mp.Action,
		Method:       strings.ToUpper(e.Method),
		APIPath:      apiPath,
		Summary:      e.Summary,
		Description:  e.Description,
		PathArgs:     mp.PathArgs,
	}
	if len(e.Tags) > 0 {
		meta.Tags = e.Tags[0]
	}

	seen := map[string]bool{}
	for _, p := range e.Params {
		switch p.In {
		case "path":
			// 路径参数已在 MapEndpoint 从路径段推导，去重
			found := false
			for _, a := range meta.PathArgs {
				if a == p.Name {
					found = true
				}
			}
			if !found {
				meta.PathArgs = append(meta.PathArgs, p.Name)
			}
		case "query":
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			meta.Flags = append(meta.Flags, commands.FlagSpec{
				Name: p.Name, Type: goFlagType(p.Type), Required: p.Required,
				In: "query",
			})
		default: // body
			if containerParamNames[strings.ToLower(p.Name)] && p.Type == "" {
				continue // $ref 容器包装
			}
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			meta.Flags = append(meta.Flags, commands.FlagSpec{
				Name: p.Name, Type: goFlagType(p.Type), Required: p.Required,
				Desc: p.Description, In: "body",
			})
		}
	}
	return meta, true
}

// goFlagType Swagger type → flag 类型标签。
func goFlagType(t string) string {
	switch t {
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array", "object":
		return "json"
	default:
		return "string"
	}
}

func renderModule(module string, metas []commands.EndpointMeta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by tools/cmdgen from docs/docs.go; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// 重新生成：go run ./tools/cmdgen -module %s\n\n", module)
	fmt.Fprintf(&b, "package cmd\n\nimport (\n\t\"github.com/spf13/cobra\"\n\n")
	fmt.Fprintf(&b, "\t\"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/commands\"\n)\n\n")
	fmt.Fprintf(&b, "// %sEndpoints %s 模块端点元数据（%d 端点）。\n",
		exportName(module), module, len(metas))
	b.WriteString("var " + exportName(module) + "Endpoints = []commands.EndpointMeta{\n")
	for _, m := range metas {
		b.WriteString(renderMeta(m))
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "// new%sModuleCmd 构建 %s 模块三级命令树。\n", exportName(module), module)
	fmt.Fprintf(&b, "func new%sModuleCmd(run commands.RunFunc, emit commands.EmitFunc) *cobra.Command {\n", exportName(module))
	fmt.Fprintf(&b, "\treturn commands.BuildModule(%sEndpoints, run, emit)\n}\n", exportName(module))
	return b.String()
}

func renderMeta(m commands.EndpointMeta) string {
	var b strings.Builder
	b.WriteString("\t{\n")
	fmt.Fprintf(&b, "\t\tModule: %q,\n", m.Module)
	b.WriteString("\t\tResourcePath: []string{")
	for i, r := range m.ResourcePath {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(r))
	}
	b.WriteString("},\n")
	fmt.Fprintf(&b, "\t\tAction: %q,\n", m.Action)
	fmt.Fprintf(&b, "\t\tMethod: %q, APIPath: %q,\n", m.Method, m.APIPath)
	fmt.Fprintf(&b, "\t\tSummary: %q,\n", m.Summary)
	fmt.Fprintf(&b, "\t\tDescription: %q,\n", m.Description)
	fmt.Fprintf(&b, "\t\tTags: %q,\n", m.Tags)
	if len(m.Flags) > 0 {
		b.WriteString("\t\tFlags: []commands.FlagSpec{\n")
		for _, f := range m.Flags {
			fmt.Fprintf(&b, "\t\t\t{Name: %q, Type: %q, Required: %v, Desc: %q, Enum: %q, In: %q},\n",
				f.Name, f.Type, f.Required, f.Desc, f.Enum, f.In)
		}
		b.WriteString("\t\t},\n")
	}
	if len(m.PathArgs) > 0 {
		var quoted []string
		for _, p := range m.PathArgs {
			quoted = append(quoted, strconv.Quote(p))
		}
		fmt.Fprintf(&b, "\t\tPathArgs: []string{%s},\n", strings.Join(quoted, ", "))
	}
	b.WriteString("\t},\n")
	return b.String()
}

// exportName 模块名 → Go 导出标识符（system → System）。
func exportName(module string) string {
	parts := strings.Split(module, "-")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cmdgen:", err)
	os.Exit(1)
}

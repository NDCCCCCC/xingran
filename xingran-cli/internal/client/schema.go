package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// swaggerDoc 是 docs/docs.go 中 Swagger 2.0 spec 的解析目标（只取需要的字段）。
type swaggerDoc struct {
	BasePath    string                          `json:"basePath"`
	Paths       map[string]map[string]swagOp    `json:"paths"`
	Definitions map[string]swagSchema           `json:"definitions"`
}

type swagOp struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Tags        []string    `json:"tags"`
	Parameters  []swagParam `json:"parameters"`
}

type swagParam struct {
	Name     string      `json:"name"`
	In       string      `json:"in"`
	Required bool        `json:"required"`
	Schema   *swagSchema `json:"schema"`
}

type swagSchema struct {
	Ref         string                `json:"$ref"`
	Type        string                `json:"type"`
	Description string                `json:"description"`
	Required    []string              `json:"required"`
	Properties  map[string]swagSchema `json:"properties"`
}

// EndpointInfo 对外输出的端点元数据（供模型自发现）。
type EndpointInfo struct {
	Path        string      `json:"path"`
	Method      string      `json:"method"`
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Params      []ParamInfo `json:"params,omitempty"`
}

// ParamInfo 参数说明（含 body $ref 展开后的字段）。
type ParamInfo struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Type        string `json:"type,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

var (
	// swaggo 生成 `const docTemplate = ` + 反引号（等号两侧可能有空白）。
	templateConstRe = regexp.MustCompile("(?s)const (?:docTemplate|SwaggerTemplate)\\s*=\\s*`(.*?)`")
	placeholderRe   = regexp.MustCompile(`\{\{[^}]*\}\}`)
)

// DefaultDocsPath 返回 docs/docs.go 的默认查找路径（相对当前工作目录向上找）。
func DefaultDocsPath() string {
	candidates := []string{
		"docs/docs.go",
		"../docs/docs.go",
		"../../docs/docs.go",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return candidates[0]
}

// LoadSchema 从 docs/docs.go（swaggo 生成文件）解析端点列表。
func LoadSchema(docsPath string) ([]EndpointInfo, string, error) {
	raw, err := os.ReadFile(docsPath)
	if err != nil {
		return nil, "", fmt.Errorf("读取 Swagger 文档失败: %w", err)
	}
	return ParseSchema(string(raw))
}

// ParseSchema 解析 swaggo 生成的 docs.go 文本。
func ParseSchema(content string) ([]EndpointInfo, string, error) {
	m := templateConstRe.FindStringSubmatch(content)
	if m == nil {
		return nil, "", fmt.Errorf("未在 docs.go 中找到 Swagger docTemplate 常量")
	}
	spec := placeholderRe.ReplaceAllString(m[1], "null")

	var doc swaggerDoc
	if err := json.Unmarshal([]byte(spec), &doc); err != nil {
		return nil, "", fmt.Errorf("解析 Swagger spec 失败: %w", err)
	}

	var out []EndpointInfo
	// 模板占位符在字符串上下文中会变成字面 "null"，视为空 basePath。
	if basePath := doc.BasePath; basePath == "null" || basePath == "{{.BasePath}}" {
		doc.BasePath = ""
	}
	for path, ops := range doc.Paths {
		for method, op := range ops {
			info := EndpointInfo{
				Path:        doc.BasePath + path,
				Method:      strings.ToUpper(method),
				Summary:     op.Summary,
				Description: op.Description,
				Tags:        op.Tags,
			}
			for _, p := range op.Parameters {
				info.Params = append(info.Params, ParamInfo{
					Name: p.Name, In: p.In, Required: p.Required,
				})
				// body 参数 $ref 展开为字段级说明。
				if p.Schema != nil && p.Schema.Ref != "" {
					info.Params = append(info.Params, expandRef(doc.Definitions, p.Schema.Ref)...)
				}
			}
			out = append(out, info)
		}
	}
	sortEndpoints(out)
	return out, doc.BasePath, nil
}

// FilterEndpoints 按路径前缀过滤（prefix 为空返回全部）。
func FilterEndpoints(eps []EndpointInfo, prefix string) []EndpointInfo {
	if prefix == "" {
		return eps
	}
	prefix = strings.Trim(prefix, "/")
	if prefix != "" {
		prefix = "/" + prefix
	}
	var out []EndpointInfo
	for _, e := range eps {
		if strings.HasPrefix(e.Path, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// expandRef 展开定义引用为字段列表。
func expandRef(defs map[string]swagSchema, ref string) []ParamInfo {
	name := strings.TrimPrefix(ref, "#/definitions/")
	def, ok := defs[name]
	if !ok {
		return []ParamInfo{{Name: "(body)", In: "body", Description: "引用未找到: " + ref}}
	}
	requiredSet := map[string]bool{}
	for _, r := range def.Required {
		requiredSet[r] = true
	}
	var out []ParamInfo
	for field, s := range def.Properties {
		out = append(out, ParamInfo{
			Name:        field,
			In:          "body",
			Type:        s.Type,
			Required:    requiredSet[field],
			Description: s.Description,
		})
	}
	return out
}

// sortEndpoints 按路径排序保证输出稳定。
func sortEndpoints(eps []EndpointInfo) {
	for i := 1; i < len(eps); i++ {
		for j := i; j > 0 && eps[j].Path < eps[j-1].Path; j-- {
			eps[j], eps[j-1] = eps[j-1], eps[j]
		}
	}
}

// DocsModuleDir 返回仓库 docs 目录（供 cmd 提示）。
func DocsModuleDir() string { return filepath.Dir(DefaultDocsPath()) }

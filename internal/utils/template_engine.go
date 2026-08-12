package utils

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateEngine 模板引擎
type TemplateEngine struct {
	FuncMap template.FuncMap
}

// NewTemplateEngine 创建模板引擎
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		FuncMap: defaultFuncMap(),
	}
}

// defaultFuncMap 默认模板函数
func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// 字符串操作
		"toUpper":   strings.ToUpper,
		"toLower":   strings.ToLower,
		"title":     cases.Title(language.English).String,
		"trim":      strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join":      strings.Join,
		"repeat":    strings.Repeat,

		// 网络相关
		"ipToHex":         ipToHex,
		"macToHex":        macToHex,
		"formatVLAN":      formatVLAN,
		"formatInterface": formatInterface,
		"formatIP":        formatIP,

		// 条件判断
		"eq": func(a, b interface{}) bool {
			return reflect.DeepEqual(a, b)
		},
		"ne": func(a, b interface{}) bool {
			return !reflect.DeepEqual(a, b)
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"lte": func(a, b int) bool {
			return a <= b
		},
		"and": func(a, b bool) bool {
			return a && b
		},
		"or": func(a, b bool) bool {
			return a || b
		},
		"not": func(a bool) bool {
			return !a
		},

		// 类型转换
		"toString": toString,
		"toInt":    toInt,
		"toBool":   toBool,

		// 列表操作
		"first": func(items []interface{}) interface{} {
			if len(items) == 0 {
				return nil
			}
			return items[0]
		},
		"last": func(items []interface{}) interface{} {
			if len(items) == 0 {
				return nil
			}
			return items[len(items)-1]
		},
		"len": func(items interface{}) int {
			if items == nil {
				return 0
			}
			v := reflect.ValueOf(items)
			switch v.Kind() {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
				return v.Len()
			default:
				return 0
			}
		},
	}
}

// Render 渲染模板
func (e *TemplateEngine) Render(templateContent string, variables map[string]interface{}) (string, error) {
	tmpl, err := template.New("config").Funcs(e.FuncMap).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// RenderWithValidation 渲染模板并验证变量
func (e *TemplateEngine) RenderWithValidation(templateContent string, variables map[string]interface{}, requiredVars []string) (string, error) {
	// 检查必需变量
	for _, varName := range requiredVars {
		if _, exists := variables[varName]; !exists {
			return "", fmt.Errorf("缺少必需变量: %s", varName)
		}
	}

	return e.Render(templateContent, variables)
}

// ExtractVariables 从模板中提取变量
func (e *TemplateEngine) ExtractVariables(templateContent string) []string {
	var variables []string
	varMap := make(map[string]bool)

	// 简单的变量提取（匹配 {{.xxx}} 格式）
	startMarker := "{{."
	endMarker := "}}"

	start := 0
	for {
		startIdx := strings.Index(templateContent[start:], startMarker)
		if startIdx == -1 {
			break
		}
		startIdx += start

		endIdx := strings.Index(templateContent[startIdx:], endMarker)
		if endIdx == -1 {
			break
		}
		endIdx += startIdx

		// 提取变量名
		varContent := templateContent[startIdx+3 : endIdx]
		varContent = strings.TrimSpace(varContent)

		// 处理管道操作
		if pipeIdx := strings.Index(varContent, "|"); pipeIdx != -1 {
			varContent = strings.TrimSpace(varContent[:pipeIdx])
		}

		// 处理函数调用
		if spaceIdx := strings.Index(varContent, " "); spaceIdx != -1 {
			varContent = strings.TrimSpace(varContent[:spaceIdx])
		}

		if varContent != "" && !varMap[varContent] {
			varMap[varContent] = true
			variables = append(variables, varContent)
		}

		start = endIdx + 2
	}

	return variables
}

// ValidateTemplate 验证模板语法
func (e *TemplateEngine) ValidateTemplate(templateContent string) error {
	_, err := template.New("validation").Funcs(e.FuncMap).Parse(templateContent)
	return err
}

// BuildVariablesMap 从变量定义和值构建变量映射
func (e *TemplateEngine) BuildVariablesMap(definitions []TemplateVariable, values map[string]string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, def := range definitions {
		value, exists := values[def.Name]

		// 检查必需变量
		if def.Required && !exists {
			return nil, fmt.Errorf("缺少必需变量: %s", def.Name)
		}

		// 使用默认值或提供的值
		if !exists && def.DefaultValue != "" {
			value = def.DefaultValue
		}

		if value != "" {
			// 根据类型转换值
			convertedValue, err := convertValue(def.Type, value)
			if err != nil {
				return nil, fmt.Errorf("转换变量 %s 失败: %w", def.Name, err)
			}
			result[def.Name] = convertedValue
		}
	}

	return result, nil
}

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name         string
	Description  string
	DefaultValue string
	Required     bool
	Type         string
	Options      []string
}

// convertValue 转换变量值
func convertValue(varType, value string) (interface{}, error) {
	switch varType {
	case "int":
		return toInt(value), nil
	case "bool":
		return toBool(value), nil
	case "string", "":
		return value, nil
	case "ip":
		return formatIP(value), nil
	case "mac":
		return macToHex(value), nil
	case "vlan":
		return formatVLAN(value), nil
	default:
		return value, nil
	}
}

// 类型转换函数
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		// 尝试解析字符串为整数
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(val)
		return lower == "true" || lower == "yes" || lower == "1" || lower == "on"
	case int:
		return val != 0
	default:
		return false
	}
}

// 网络相关函数
func ipToHex(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}

	var result strings.Builder
	for _, part := range parts {
		var b int
		if _, err := fmt.Sscanf(part, "%d", &b); err != nil {
			continue
		}
		result.WriteString(fmt.Sprintf("%02X", b))
	}

	return result.String()
}

func macToHex(mac string) string {
	// 移除所有分隔符
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ReplaceAll(mac, " ", "")

	if len(mac) != 12 {
		return ""
	}

	return strings.ToUpper(mac)
}

func formatVLAN(vlan interface{}) string {
	switch v := vlan.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%d", toInt(v))
	}
}

func formatInterface(iface string) string {
	// 标准化接口名称
	iface = strings.TrimSpace(iface)
	iface = strings.ToUpper(iface)

	// 替换常见简写
	replacements := map[string]string{
		"GI": "GigabitEthernet",
		"GE": "GigabitEthernet",
		"FA": "FastEthernet",
		"TE": "TenGigabitEthernet",
		"FO": "FortyGigE",
	}

	for short, full := range replacements {
		if strings.HasPrefix(iface, short) {
			return full + iface[len(short):]
		}
	}

	return iface
}

func formatIP(ip string) string {
	ip = strings.TrimSpace(ip)
	return ip
}

// ParseCommandVariables 从命令行解析变量
// 格式: key1=value1,key2=value2 或 key1:=value1,key2:=value2
func ParseCommandVariables(input string) map[string]string {
	result := make(map[string]string)

	if input == "" {
		return result
	}

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		var key, value string
		if strings.Contains(pair, ":=") {
			parts := strings.SplitN(pair, ":=", 2)
			key = strings.TrimSpace(parts[0])
			value = strings.TrimSpace(parts[1])
		} else if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			key = strings.TrimSpace(parts[0])
			value = strings.TrimSpace(parts[1])
		}

		if key != "" {
			result[key] = value
		}
	}

	return result
}

// GeneratePreview 生成模板预览（使用示例值）
func (e *TemplateEngine) GeneratePreview(templateContent string, definitions []TemplateVariable) (string, error) {
	// 生成示例变量值
	sampleValues := make(map[string]interface{})
	for _, def := range definitions {
		if def.DefaultValue != "" {
			sampleValues[def.Name] = def.DefaultValue
		} else {
			sampleValues[def.Name] = getSampleValue(def.Type, def.Name)
		}
	}

	return e.Render(templateContent, sampleValues)
}

// getSampleValue 获取示例值
func getSampleValue(varType, varName string) string {
	switch varType {
	case "int":
		return "100"
	case "bool":
		return "true"
	case "ip":
		return "192.168.1.1"
	case "mac":
		return "00:11:22:33:44:55"
	case "vlan":
		return "100"
	case "select":
		return "option1"
	default:
		return fmt.Sprintf("<%s>", varName)
	}
}

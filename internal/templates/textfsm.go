package templates

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// TextFSM 解析器 - 简化且正确的实现
type FSM struct {
	Variables     map[string]*Variable
	States        map[string]*State
	CurrentState  string
	Records       []map[string]string
	CurrentRecord map[string]string
}

// Variable 表示模板中的变量
type Variable struct {
	Name     string
	Required bool
	List     bool
	Regex    string // 正则表达式字符串
}

// State 表示状态机的一个状态
type State struct {
	Name    string
	Rules   []*Rule
	Default string
}

// Rule 表示状态转换规则
type Rule struct {
	RegexPattern string // 原始正则模式（包含 ${VAR} 引用）
	Regex        *regexp.Regexp
	VarNames     []string // 按顺序的变量名
	NextState    string
}

// ParseTemplate 从 embed.FS 或本地文件系统解析 textfsm 模板。
//
// 查找顺序：
//  1. 嵌入的 templatesFS（生产部署不需要本地 templates/ 目录）
//  2. 本地文件系统（dev 模式或自定义模板路径）
//
// 支持的输入格式：
//   - "templates/huawei_xxx.textfsm"             → embedded/templates/huawei_xxx.textfsm
//   - "templates/lldp/lldp_ruijie.textfsm"       → embedded/templates/lldp/lldp_ruijie.textfsm
//   - "lldp/lldp_ruijie.textfsm"                 → embedded/templates/lldp/lldp_ruijie.textfsm
//   - "huawei_xxx.textfsm"                       → embedded/templates/huawei_xxx.textfsm
//   - 绝对路径（开发测试用）：跳过 embed，走本地文件系统
func ParseTemplate(filename string) (*FSM, error) {
	// 绝对路径：直接读本地文件系统（兼容 dev 测试用例）
	if filepath.IsAbs(filename) {
		content, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("读取模板文件失败: %w", err)
		}
		return parseTemplateContent(filename, content)
	}

	// 1) 尝试从 embed.FS 读取（生产路径）
	if embedPath, ok := resolveEmbedPath(filename); ok {
		data, err := fs.ReadFile(embeddedTemplatesRoot(), embedPath)
		if err == nil {
			return parseTemplateContent(filename, data)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("读取嵌入模板失败 %s -> %s: %w", filename, embedPath, err)
		}
		// embed 里没有（理论上不应该），fallback 到本地文件系统
	}

	// 2) Fallback: 本地文件系统（dev 模式无 go:embed 的版本，或自定义模板）
	return parseTemplateFromFilesystem(filename)
}

// resolveEmbedPath 把用户传入的相对路径映射到 embed.FS 内的真实路径。
// 返回值 ok=false 表示输入不解析为模板路径（如 ../escape 或禁止字符）。
func resolveEmbedPath(filename string) (string, bool) {
	clean := path.Clean(filename)
	if clean == "." || clean == "/" || strings.HasPrefix(clean, "..") {
		return "", false
	}

	// 统一路径分隔符，embed.FS 用正斜杠
	clean = filepath.ToSlash(clean)

	// 已有 "templates/" 前缀：去掉前缀直接作为 fs 路径
	if strings.HasPrefix(clean, "templates/") {
		return "embedded/" + clean, true
	}

	// "lldp/xxx" 或 "huawei_xxx.textfsm"：拼到 templates/ 下
	return "embedded/templates/" + clean, true
}

// parseTemplateFromFilesystem 解析本地文件系统模板，保留 dev 模式兼容。
func parseTemplateFromFilesystem(filename string) (*FSM, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("无法获取当前工作目录: %w", err)
	}

	projectRoot := findProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	fullPath := filepath.Join(projectRoot, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件失败 %s (尝试路径: %s): %w", filename, fullPath, err)
	}
	return parseTemplateContent(filename, content)
}

// findProjectRoot 从当前目录向上搜索包含 go.mod 的目录（项目根目录）
func findProjectRoot(dir string) string {
	for {
		// 检查当前目录是否有 go.mod
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// 到达文件系统根目录
		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // 没找到 go.mod
		}

		dir = parent
	}
}

// parseTemplateContent 解析模板内容（内部辅助函数）
func parseTemplateContent(filename string, content []byte) (*FSM, error) {
	fsm, err := ParseTemplateString(string(content))
	if err != nil {
		return nil, err
	}

	// 模板解析调试信息（受日志级别控制：prod=info 静默，dev=debug 可见）
	applogger.Debugf("[TextFSM] 模板解析成功: %s", filename)
	applogger.Debugf("[TextFSM] 变量数量: %d", len(fsm.Variables))
	for name, v := range fsm.Variables {
		applogger.Debugf("[TextFSM]   变量: %s = %s (Required=%v, List=%v)", name, v.Regex, v.Required, v.List)
	}
	applogger.Debugf("[TextFSM] 状态数量: %d", len(fsm.States))
	for name, state := range fsm.States {
		applogger.Debugf("[TextFSM]   状态: %s, 规则数: %d", name, len(state.Rules))
		for i, rule := range state.Rules {
			applogger.Debugf("[TextFSM]     规则%d: %s -> %s", i, rule.RegexPattern, rule.NextState)
			applogger.Debugf("[TextFSM]       编译后正则: %s", rule.Regex.String())
		}
	}

	return fsm, nil
}

// ParseTemplateString 从字符串解析 textfsm 模板
func ParseTemplateString(content string) (*FSM, error) {
	fsm := &FSM{
		Variables:     make(map[string]*Variable),
		States:        make(map[string]*State),
		CurrentState:  "Start",
		Records:       make([]map[string]string, 0),
		CurrentRecord: make(map[string]string),
	}

	// 初始化 Start 状态
	fsm.States["Start"] = &State{Name: "Start", Rules: make([]*Rule, 0)}

	// 变量定义正则（支持 List 关键字）
	valueListRegex := regexp.MustCompile(`^Value\s+List\s+(\w+)\s+\((.+)\)$`)
	valueReqListRegex := regexp.MustCompile(`^Value\s+Required\s+List\s+(\w+)\s+\((.+)\)$`)
	valueRegex := regexp.MustCompile(`^Value\s+Required\s+(\w+)\s+\((.+)\)$`)
	valueSimpleRegex := regexp.MustCompile(`^Value\s+(\w+)\s+\((.+)\)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentState *State = fsm.States["Start"]

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 检测缩进
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// 解析变量定义
		if matches := valueListRegex.FindStringSubmatch(trimmed); matches != nil {
			fsm.Variables[matches[1]] = &Variable{
				Name:  matches[1],
				List:  true,
				Regex: matches[2],
			}
			continue
		}
		if matches := valueReqListRegex.FindStringSubmatch(trimmed); matches != nil {
			fsm.Variables[matches[1]] = &Variable{
				Name:     matches[1],
				Required: true,
				List:     true,
				Regex:    matches[2],
			}
			continue
		}
		if matches := valueRegex.FindStringSubmatch(trimmed); matches != nil {
			fsm.Variables[matches[1]] = &Variable{
				Name:     matches[1],
				Required: true,
				Regex:    matches[2],
			}
			continue
		}
		if matches := valueSimpleRegex.FindStringSubmatch(trimmed); matches != nil {
			fsm.Variables[matches[1]] = &Variable{
				Name:  matches[1],
				Regex: matches[2],
			}
			continue
		}

		// 检查是否是状态定义（首字母大写，单独一行）
		if indent == 0 && isFirstUpper(trimmed) {
			if _, exists := fsm.States[trimmed]; !exists {
				fsm.States[trimmed] = &State{
					Name:  trimmed,
					Rules: make([]*Rule, 0),
				}
			}
			currentState = fsm.States[trimmed]
			continue
		}

		// 解析规则行（使用原始行，保留缩进）
		if strings.HasPrefix(trimmed, "^") {
			rule, err := parseRule(line, fsm)
			if err == nil && currentState != nil {
				currentState.Rules = append(currentState.Rules, rule)
			}
		}
	}

	return fsm, scanner.Err()
}

// parseRule 解析单条规则
func parseRule(line string, fsm *FSM) (*Rule, error) {
	// 保存原始规则用于调试
	originalLine := line

	// 移除行首空白
	line = strings.TrimLeft(line, " \t")

	// 提取状态转移
	var nextState string
	if idx := strings.Index(line, "->"); idx > 0 {
		parts := strings.SplitN(line, "->", 2)
		line = strings.TrimSpace(parts[0])
		nextState = strings.TrimSpace(parts[1])
	} else {
		nextState = "Continue"
	}

	// 移除锚点
	// 检查是否有行首锚点
	hasStartAnchor := strings.HasPrefix(line, "^")
	// 检查是否需要行尾锚点（TextFSM中使用$$表示行尾锚点$）
	hasEndAnchor := strings.HasSuffix(line, "$")

	line = strings.TrimPrefix(line, "^")
	// TextFSM中使用$$表示行尾锚点$，需要移除所有末尾的$
	line = strings.TrimRight(line, "$")

	// 构建正则表达式并记录变量名顺序
	regexPattern, varNames := buildRegexWithVariables(line, fsm)

	// 添加回锚点
	if hasStartAnchor {
		regexPattern = "^" + regexPattern
	}
	// 如果原始行有行尾锚点，添加$到正则表达式末尾
	if hasEndAnchor {
		regexPattern += "$"
	}

	// 调试输出（受日志级别控制）
	applogger.Debugf("[TextFSM] parseRule: 原始规则=%s", originalLine)
	applogger.Debugf("[TextFSM] parseRule: 处理后模式=%s", line)
	applogger.Debugf("[TextFSM] parseRule: 编译后正则=%s", regexPattern)
	applogger.Debugf("[TextFSM] parseRule: 变量列表=%v", varNames)

	// 编译正则
	compiled, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("编译正则失败: %s, 错误: %w", regexPattern, err)
	}

	return &Rule{
		RegexPattern: regexPattern,
		Regex:        compiled,
		VarNames:     varNames,
		NextState:    nextState,
	}, nil
}

// buildRegexWithVariables 构建正则表达式并返回变量名列表
func buildRegexWithVariables(pattern string, fsm *FSM) (string, []string) {
	// 匹配 ${VariableName}
	varRefRegex := regexp.MustCompile(`\$\{(\w+)\}`)

	varNames := []string{}

	// 步骤1：保护可选组 (...)?，用占位符替换
	optionalGroupRegex := regexp.MustCompile(`\(([^)]*\$\{[^}]+\}[^)]*)\)\?`)
	optionalGroups := map[string]string{}
	protectedPattern := pattern
	groupIndex := 0

	for {
		match := optionalGroupRegex.FindStringSubmatchIndex(pattern)
		if match == nil {
			break
		}
		placeholder := fmt.Sprintf("__OPTIONAL_GROUP_%d__", groupIndex)
		groupContent := pattern[match[0]+1 : match[1]-1] // 去掉外层的 ( )
		optionalGroups[placeholder] = groupContent
		protectedPattern = protectedPattern[:match[0]] + placeholder + "?" + protectedPattern[match[1]:]
		pattern = protectedPattern // 更新 pattern 用于下一次匹配
		groupIndex++
	}

	// 步骤2：处理变量替换
	result := ""
	lastEnd := 0

	matches := varRefRegex.FindAllStringSubmatchIndex(protectedPattern, -1)
	for _, match := range matches {
		// 添加前面的文本 - 使用自定义转义，保留反斜杠
		literalText := protectedPattern[lastEnd:match[0]]
		escaped := escapeRegexLiteral(literalText)
		result += escaped

		// 获取变量名
		varName := protectedPattern[match[2]:match[3]]
		varNames = append(varNames, varName)

		// 查找变量定义
		if variable, ok := fsm.Variables[varName]; ok {
			// 使用变量的正则表达式作为捕获组
			result += fmt.Sprintf("(%s)", variable.Regex)
			applogger.Debugf("[TextFSM] buildRegex: 变量 %s = (%s)", varName, variable.Regex)
		} else {
			// 未知变量，使用通用捕获组
			result += "(.+)"
			applogger.Debugf("[TextFSM] buildRegex: 未知变量 %s，使用通用捕获组", varName)
		}

		lastEnd = match[1]
	}

	// 添加剩余文本
	literalText := protectedPattern[lastEnd:]
	escaped := escapeRegexLiteral(literalText)
	result += escaped

	// 步骤3：恢复可选组占位符
	for placeholder, groupContent := range optionalGroups {
		// 查找占位符在结果中的位置（后面跟着 ?）
		placeholderWithQuestion := regexp.QuoteMeta(placeholder + "?")
		replacement := "(\\s+" + groupContent + ")?"
		result = regexp.MustCompile(placeholderWithQuestion).ReplaceAllString(result, replacement)
	}

	return result, varNames
}

// escapeRegexLiteral 转义正则表达式字面量，智能处理正则序列。
// TextFSM 模板中的规则文本本身就是正则表达式片段，不是纯字面量文本。
// 因此：
//   - 裸 . 是正则通配符（匹配任何字符），不应转义
//   - \. 是转义的点（匹配字面量点），应保留原样
//   - \s \d \w 等字符类转义应保留原样
//   - ? + * 作为量词不应转义
//   - ( ) 用于分组和可选组
func escapeRegexLiteral(s string) string {
	result := ""
	i := 0

	for i < len(s) {
		c := s[i]

		// 如果是反斜杠，可能是正则转义序列
		if c == '\\' && i+1 < len(s) {
			nextC := s[i+1]
			// 保留所有有效的正则转义序列：
			// - 非字母字符（元字符转义）：\. \( \| \[ \] \{ \} \^ \$ \* \+ \? \\ 等
			//   在正则中，\ 后跟非字母字符总是"转义该元字符为字面量"，必须保留原样
			// - 字母字符（字符类转义）：\s \S \d \D \w \W \b \B \n \r \t \a \f \v 等
			if (nextC >= 'a' && nextC <= 'z') || (nextC >= 'A' && nextC <= 'Z') {
				// 字母字符：只保留已知的正则字符类转义
				if strings.ContainsAny(string(nextC), "sSdDwWbBnrtafv") {
					result += string([]byte{c, nextC})
					i += 2
					continue
				}
				// 未知字母转义，转义反斜杠
				result += "\\\\"
				i++
				continue
			}
			// 非字母字符（如 . ( ) | [ ] { } ^ $ * + ? \ 等）：保留转义序列原样
			result += string([]byte{c, nextC})
			i += 2
			continue
		}

		// 处理正则元字符
		if c == '(' {
			// 检查是否是可选组 "(...)?"
			// 向前查找匹配的 )
			balance := 1
			j := i + 1
			for j < len(s) && balance > 0 {
				if s[j] == '(' {
					balance++
				} else if s[j] == ')' {
					balance--
				}
				j++
			}
			// 如果找到匹配的 ) 且后面跟着 ?，则是可选组，不转义 (
			if balance == 0 && j < len(s) && s[j] == '?' {
				result += string(c)
			} else {
				result += `\` + string(c)
			}
		} else if c == ')' {
			// 检查后面是否跟着 ?
			if i+1 < len(s) && s[i+1] == '?' {
				result += string(c)
			} else {
				result += `\` + string(c)
			}
		} else if c == '.' {
			// TextFSM 模板中裸 . 是正则通配符（匹配任何字符），不应转义。
			// 字面量点通过 \. 表示（已在上方反斜杠处理中保留）。
			result += string(c)
		} else {
			// 其他元字符：? + * 作为量词不转义，| [ ] { } ^ $ 需要转义
			metaChars := "|[]{}^$"
			if strings.ContainsAny(string(c), metaChars) {
				result += `\` + string(c)
			} else {
				result += string(c)
			}
		}
		i++
	}

	return result
}

// isFirstUpper 检查字符串首字母是否大写
func isFirstUpper(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

// Clone 创建 FSM 的浅拷贝，共享不可变的 Variables 和 States，
// 但使用独立的可变状态（Records、CurrentRecord、CurrentState）。
// 这样缓存的 FSM 模板可以被多个 goroutine 安全地并发使用。
func (f *FSM) Clone() *FSM {
	return &FSM{
		Variables:     f.Variables, // 共享只读
		States:        f.States,    // 共享只读
		CurrentState:  "Start",
		Records:       make([]map[string]string, 0),
		CurrentRecord: make(map[string]string),
	}
}

// ParseText 使用模板解析文本
// 注意：此方法会修改 FSM 的可变状态，通过 Clone 确保并发安全。
func (f *FSM) ParseText(text string) ([]map[string]string, error) {
	// 克隆自身，避免修改缓存中的模板实例
	clone := f.Clone()

	applogger.Debugf("[TextFSM] 开始解析文本，共 %d 行", len(strings.Split(text, "\n")))

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 获取当前状态
		state := clone.States[clone.CurrentState]
		if state == nil {
			clone.CurrentState = "Start"
			state = clone.States["Start"]
		}

		matched := false

		// 尝试匹配规则
		for _, rule := range state.Rules {
			if rule.Regex == nil {
				continue
			}

			if rule.Regex.MatchString(line) {
				matched = true
				// 提取捕获组并赋值给变量
				matches := rule.Regex.FindStringSubmatch(line)
				for i, varName := range rule.VarNames {
					if i+1 < len(matches) {
						clone.CurrentRecord[varName] = matches[i+1]
					}
				}

				// 处理状态转换
				clone.handleStateTransition(rule.NextState)

				break
			}
		}

		// 如果没有匹配任何规则，使用状态默认转换
		if !matched {
			if state.Default != "" {
				clone.handleStateTransition(state.Default)
			}
		}
	}

	// 保存最后一条记录
	if len(clone.CurrentRecord) > 0 {
		clone.copyCurrentRecord()
	}

	applogger.Debugf("[TextFSM] 解析完成，提取 %d 条记录", len(clone.Records))

	return clone.Records, nil
}

// copyCurrentRecord 复制当前记录到记录集
func (f *FSM) copyCurrentRecord() {
	record := make(map[string]string)
	for k, v := range f.CurrentRecord {
		record[k] = v
	}
	f.Records = append(f.Records, record)
}

// handleStateTransition 处理状态转换
func (f *FSM) handleStateTransition(nextState string) {
	switch nextState {
	case "Continue":
		// 继续当前状态
	case "Record":
		// 保存当前记录
		if len(f.CurrentRecord) > 0 {
			f.copyCurrentRecord()
		}
		// 开始新记录
		f.CurrentRecord = make(map[string]string)
	case "Error":
		// 错误状态，返回 Start
		f.CurrentState = "Start"
	default:
		// 跳转到指定状态
		if _, exists := f.States[nextState]; exists {
			f.CurrentState = nextState
		}
	}
}

// GetRecords 获取解析后的所有记录
func (f *FSM) GetRecords() []map[string]string {
	return f.Records
}

// GetFirstRecord 获取第一条记录
func (f *FSM) GetFirstRecord() map[string]string {
	if len(f.Records) > 0 {
		return f.Records[0]
	}
	return nil
}

// Reset 重置状态机
func (f *FSM) Reset() {
	f.Records = make([]map[string]string, 0)
	f.CurrentRecord = make(map[string]string)
	f.CurrentState = "Start"
}

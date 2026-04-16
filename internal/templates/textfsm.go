package templates

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// debug 模式开关（设置为 true 启用 TextFSM 调试日志）
const debugTextFSM = false

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

// ParseTemplate 从文件解析 textfsm 模板
func ParseTemplate(filename string) (*FSM, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件失败: %w", err)
	}

	fsm, err := ParseTemplateString(string(content))
	if err != nil {
		return nil, err
	}

	// 输出调试信息
	if debugTextFSM {
		fmt.Printf("[TextFSM] 模板解析成功: %s\n", filename)
		fmt.Printf("[TextFSM] 变量数量: %d\n", len(fsm.Variables))
		for name, v := range fsm.Variables {
			fmt.Printf("[TextFSM]   变量: %s = %s (Required=%v, List=%v)\n", name, v.Regex, v.Required, v.List)
		}
		fmt.Printf("[TextFSM] 状态数量: %d\n", len(fsm.States))
		for name, state := range fsm.States {
			fmt.Printf("[TextFSM]   状态: %s, 规则数: %d\n", name, len(state.Rules))
			for i, rule := range state.Rules {
				fmt.Printf("[TextFSM]     规则%d: %s -> %s\n", i, rule.RegexPattern, rule.NextState)
				fmt.Printf("[TextFSM]       编译后正则: %s\n", rule.Regex.String())
			}
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
	line = strings.TrimPrefix(line, "^")
	line = strings.TrimSuffix(line, "$")

	// 构建正则表达式并记录变量名顺序
	regexPattern, varNames := buildRegexWithVariables(line, fsm)

	// 调试输出
	if debugTextFSM {
		fmt.Printf("[TextFSM] parseRule: 原始规则=%s\n", originalLine)
		fmt.Printf("[TextFSM] parseRule: 处理后模式=%s\n", line)
		fmt.Printf("[TextFSM] parseRule: 编译后正则=%s\n", regexPattern)
		fmt.Printf("[TextFSM] parseRule: 变量列表=%v\n", varNames)
	}

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
			if debugTextFSM {
				fmt.Printf("[TextFSM] buildRegex: 变量 %s = (%s)\n", varName, variable.Regex)
			}
		} else {
			// 未知变量，使用通用捕获组
			result += "(.+)"
			if debugTextFSM {
				fmt.Printf("[TextFSM] buildRegex: 未知变量 %s，使用通用捕获组\n", varName)
			}
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

// escapeRegexLiteral 转义正则表达式字面量，智能处理正则序列
func escapeRegexLiteral(s string) string {
	result := ""
	i := 0

	for i < len(s) {
		c := s[i]

		// 如果是反斜杠，可能是转义序列
		if c == '\\' && i+1 < len(s) {
			nextC := s[i+1]
			// 常见的正则转义序列，保留
			if strings.ContainsAny(string(nextC), "sSdDwWbBnrt") {
				result += string([]byte{c, nextC})
				i += 2
				continue
			}
			// 其他情况转义反斜杠
			result += "\\\\"
			i++
			continue
		}

		// 处理正则元字符
		// 注意：不转义 ? + * 因为它们是量词
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
		} else {
			// 其他元字符（不包括 ? + *）
			metaChars := ".|[]{}^$"
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

// ParseText 使用模板解析文本
func (f *FSM) ParseText(text string) ([]map[string]string, error) {
	f.Records = make([]map[string]string, 0)
	f.CurrentRecord = make(map[string]string)
	f.CurrentState = "Start"

	if debugTextFSM {
		fmt.Printf("[TextFSM] 开始解析文本，共 %d 行\n", len(strings.Split(text, "\n")))
	}

	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// 获取当前状态
		state := f.States[f.CurrentState]
		if state == nil {
			f.CurrentState = "Start"
			state = f.States["Start"]
		}

		matched := false

		// 尝试匹配规则
		for _, rule := range state.Rules {
			if rule.Regex == nil {
				continue
			}

			if rule.Regex.MatchString(line) {
				matched = true
				if debugTextFSM {
					fmt.Printf("[TextFSM] 行%d匹配: %s\n", lineNum, rule.RegexPattern)

					// 提取捕获组并赋值给变量
					matches := rule.Regex.FindStringSubmatch(line)
					fmt.Printf("[TextFSM]   捕获组: %v\n", matches)
					for i, varName := range rule.VarNames {
						if i+1 < len(matches) {
							f.CurrentRecord[varName] = matches[i+1]
							fmt.Printf("[TextFSM]   %s = %s\n", varName, matches[i+1])
						}
					}
				} else {
					// 提取捕获组并赋值给变量（不输出日志）
					matches := rule.Regex.FindStringSubmatch(line)
					for i, varName := range rule.VarNames {
						if i+1 < len(matches) {
							f.CurrentRecord[varName] = matches[i+1]
						}
					}
				}

				// 处理状态转换
				f.handleStateTransition(rule.NextState)

				break
			}
		}

		// 如果没有匹配任何规则，打印详细的调试信息
		if !matched {
			if debugTextFSM {
				fmt.Printf("[TextFSM] 行%d不匹配任何规则: %s\n", lineNum, line)
				if lineNum < 5 && len(state.Rules) > 0 {
					rule := state.Rules[0]
					fmt.Printf("[TextFSM]   尝试规则1: %s\n", rule.RegexPattern)
					fmt.Printf("[TextFSM]   正则表达式: %s\n", rule.Regex.String())
				}
			}
			if state.Default != "" {
				f.handleStateTransition(state.Default)
			}
		}
	}

	// 保存最后一条记录
	if len(f.CurrentRecord) > 0 {
		f.copyCurrentRecord()
	}

	if debugTextFSM {
		fmt.Printf("[TextFSM] 解析完成，提取 %d 条记录\n", len(f.Records))
	}

	return f.Records, nil
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

package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/client"
)

// 语义化退出码。
const (
	ExitOK        = 0 // 成功
	ExitNetwork   = 1 // 网络错误（连接失败、超时等传输层错误）
	ExitAuth      = 2 // 认证失败（401 / 1007 / 登录失败 / token 无效）
	ExitParam     = 3 // 参数错误（本地参数不合法或服务端 400 / 1001）
	ExitForbidden = 4 // 权限不足（403）
	ExitOther     = 5 // 其他业务/未知错误
)

// 支持的输出格式
const (
	FormatJSON  = "json"
	FormatTable = "table"
	FormatCSV   = "csv"
	FormatYAML  = "yaml"
)

// exitError 由 cmd 层包装"未触达服务端"的本地错误，
// ExitCode 决定进程退出码，Message 进入 stderr 的 JSON。
type exitError struct {
	Code     int
	Message  string
	Err      error
	ExitCode int
}

// NewExitError 构造带退出码的本地错误。
func NewExitError(exitCode int, message string, err error) error {
	return &exitError{Code: -1, Message: message, Err: err, ExitCode: exitCode}
}

func (e *exitError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// ErrorJSON 统一的错误输出结构（stderr）。
type ErrorJSON struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	ExitCode  int    `json:"exit_code"`
	RequestID string `json:"request_id,omitempty"`
}

// Emit 将一次命令执行的结果写入对应流并返回退出码。
// 兼容旧签名：默认按 format="json" (或 tty 下人类可读)。
func Emit(result any, err error, stdout, stderr io.Writer, tty bool) int {
	return EmitWithFormat(result, err, stdout, stderr, tty, "")
}

// EmitWithFormat 将一次命令执行的结果按指定 format 写入对应流并返回退出码。
// format 支持: json, table, csv, yaml。若为空则按 tty 默认 json/human。
func EmitWithFormat(result any, err error, stdout, stderr io.Writer, tty bool, format string) int {
	if err == nil {
		return emitSuccess(result, stdout, tty, format)
	}
	return emitError(err, stderr, tty)
}

func emitSuccess(result any, stdout io.Writer, tty bool, format string) int {
	if result == nil {
		result = struct{}{}
	}

	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case FormatYAML, "yml":
		return writeYAML(stdout, result)
	case FormatCSV:
		return writeCSV(stdout, result)
	case FormatTable:
		return writeTable(stdout, result)
	case FormatJSON:
		return writeJSON(stdout, result, ExitOK)
	}

	// 未显式指定 format 时按原规则：tty 输出人类可读缩进 JSON，非 tty 输出单行 JSON
	if tty {
		writeHuman(stdout, result)
		return ExitOK
	}
	return writeJSON(stdout, result, ExitOK)
}

func emitError(err error, stderr io.Writer, tty bool) int {
	ej := classify(err)
	if tty {
		writeHuman(stderr, ej)
	} else {
		writeJSON(stderr, ej, ej.ExitCode)
	}
	return ej.ExitCode
}

// classify 把任意错误归类为结构化错误与退出码。
func classify(err error) ErrorJSON {
	var local *exitError
	if errors.As(err, &local) {
		return ErrorJSON{Code: local.Code, Message: local.Error(), ExitCode: local.ExitCode}
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return ErrorJSON{
			Code:      apiErr.Resp.Code,
			Message:   apiErr.Resp.Message,
			ExitCode:  ExitFromCode(apiErr.Resp.Code, apiErr.HTTPStatus),
			RequestID: apiErr.Resp.RequestID,
		}
	}

	// auth 模块的登录链路错误：Message 为服务端原文（如锁定提示），原样透传。
	var authErr *auth.APIError
	if errors.As(err, &authErr) {
		return ErrorJSON{Code: authErr.Code, Message: authErr.Message, ExitCode: ExitFromCode(authErr.Code, 0)}
	}

	var transportErr *client.TransportError
	if errors.As(err, &transportErr) {
		return ErrorJSON{Code: -1, Message: transportErr.Error(), ExitCode: ExitNetwork}
	}

	return ErrorJSON{Code: -1, Message: err.Error(), ExitCode: ExitOther}
}

// ExitFromCode 将服务端响应 code / HTTP 状态映射为语义化退出码。
func ExitFromCode(code int, httpStatus int) int {
	switch code {
	case 401, 1007:
		return ExitAuth
	case 400, 1001, 422:
		return ExitParam
	case 403:
		return ExitForbidden
	}
	if code == 0 {
		return ExitOK
	}
	// 响应不是标准包装（code 无语义）时参考 HTTP 状态。
	switch httpStatus {
	case http.StatusUnauthorized:
		return ExitAuth
	case http.StatusForbidden:
		return ExitForbidden
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return ExitParam
	}
	return ExitOther
}

func writeJSON(w io.Writer, v any, exitCode int) int {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(w, "{\"code\":-1,\"message\":\"internal encode error: %s\",\"exit_code\":%d}\n",
			err.Error(), ExitOther)
		return ExitOther
	}
	return exitCode
}

func writeYAML(w io.Writer, v any) int {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(w, "internal yaml encode error: %s\n", err.Error())
		return ExitOther
	}
	return ExitOK
}

// extractRows 递归或从 response/map 中抽取出可用于表格/CSV 显示的对象切片
func extractRows(v any) []map[string]any {
	// 优先利用 JSON 转换为通用 map / slice
	var raw any
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(b, &raw)

	// 如果顶层是 client.Response 形式 { "code": 0, "data": ... }
	if m, ok := raw.(map[string]any); ok {
		if data, exists := m["data"]; exists && data != nil {
			if subRows := extractRows(data); len(subRows) > 0 {
				return subRows
			}
		}
		if list, exists := m["list"]; exists && list != nil {
			if subRows := extractRows(list); len(subRows) > 0 {
				return subRows
			}
		}
		// 单个对象转化为 1 行
		return []map[string]any{m}
	}

	if s, ok := raw.([]any); ok {
		rows := make([]map[string]any, 0, len(s))
		for _, item := range s {
			if im, ok := item.(map[string]any); ok {
				rows = append(rows, im)
			} else {
				rows = append(rows, map[string]any{"value": item})
			}
		}
		return rows
	}

	return nil
}

// collectHeaders 收集所有列名并按常用排序规则排序
func collectHeaders(rows []map[string]any) []string {
	set := make(map[string]bool)
	for _, r := range rows {
		for k := range r {
			set[k] = true
		}
	}
	headers := make([]string, 0, len(set))
	for k := range set {
		headers = append(headers, k)
	}

	// 优先把 id, name, code, status, createdAt 等核心字段排在前面
	priority := map[string]int{
		"id": 1, "code": 2, "name": 3, "deptName": 4, "userName": 5,
		"status": 6, "type": 7, "createdAt": 80, "updatedAt": 90,
	}

	sort.Slice(headers, func(i, j int) bool {
		pi, oki := priority[headers[i]]
		pj, okj := priority[headers[j]]
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return headers[i] < headers[j]
	})

	return headers
}

func writeCSV(w io.Writer, v any) int {
	rows := extractRows(v)
	if len(rows) == 0 {
		return writeJSON(w, v, ExitOK)
	}

	headers := collectHeaders(rows)
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// 写表头
	if err := csvWriter.Write(headers); err != nil {
		fmt.Fprintf(w, "internal csv write error: %s\n", err.Error())
		return ExitOther
	}

	// 写数据行
	for _, row := range rows {
		line := make([]string, len(headers))
		for i, h := range headers {
			val := row[h]
			line[i] = formatCell(val)
		}
		if err := csvWriter.Write(line); err != nil {
			fmt.Fprintf(w, "internal csv write error: %s\n", err.Error())
			return ExitOther
		}
	}
	return ExitOK
}

func writeTable(w io.Writer, v any) int {
	rows := extractRows(v)
	if len(rows) == 0 {
		return writeJSON(w, v, ExitOK)
	}

	headers := collectHeaders(rows)
	// 计算各列最大宽度
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	strRows := make([][]string, len(rows))
	for ri, row := range rows {
		strRows[ri] = make([]string, len(headers))
		for ci, h := range headers {
			s := formatCell(row[h])
			// 避免多行文本破坏终端单行对齐
			s = strings.ReplaceAll(s, "\n", " ")
			s = strings.ReplaceAll(s, "\r", "")
			if len(s) > 40 {
				s = s[:37] + "..."
			}
			strRows[ri][ci] = s
			if len(s) > widths[ci] {
				widths[ci] = len(s)
			}
		}
	}

	// 打印表头
	var buf bytes.Buffer
	for i, h := range headers {
		fmt.Fprintf(&buf, "%-*s  ", widths[i], h)
	}
	fmt.Fprintln(w, strings.TrimRight(buf.String(), " "))

	// 打印分隔线
	buf.Reset()
	for i := range headers {
		fmt.Fprintf(&buf, "%s  ", strings.Repeat("-", widths[i]))
	}
	fmt.Fprintln(w, strings.TrimRight(buf.String(), " "))

	// 打印行
	for _, r := range strRows {
		buf.Reset()
		for i, val := range r {
			fmt.Fprintf(&buf, "%-*s  ", widths[i], val)
		}
		fmt.Fprintln(w, strings.TrimRight(buf.String(), " "))
	}

	return ExitOK
}

func formatCell(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		b, err := json.Marshal(val)
		if err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", val)
	}
}

// writeHuman TTY 人类可读输出。
func writeHuman(w io.Writer, v any) {
	switch t := v.(type) {
	case ErrorJSON:
		fmt.Fprintf(w, "错误 [%d] (exit=%d): %s\n", t.Code, t.ExitCode, t.Message)
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Fprintf(w, "%+v\n", v)
			return
		}
		fmt.Fprintf(w, "%s\n", b)
	}
}

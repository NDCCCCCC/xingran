// Package output 负责命令行输出的统一格式化与语义化退出码。
//
// 规则：
//   - 非 TTY（管道 / agent 调用）：纯 JSON 输出，不染色、不交互；
//     成功结果 JSON → stdout，错误 JSON → stderr。
//   - TTY：人类可读输出。
//   - 退出码：0 成功 / 1 网络错误 / 2 认证失败 / 3 参数错误 / 4 权限不足 / 5 其他。
package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

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
//
// result 非 nil 且 err == nil 时视为成功；否则按 err 分类输出错误并给退出码。
// tty 决定 JSON（非 TTY）或人类可读（TTY）格式。
func Emit(result any, err error, stdout, stderr io.Writer, tty bool) int {
	if err == nil {
		return emitSuccess(result, stdout, tty)
	}
	return emitError(err, stderr, tty)
}

func emitSuccess(result any, stdout io.Writer, tty bool) int {
	if result == nil {
		result = struct{}{}
	}
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
		// 编码失败极罕见（如不可序列化类型），兜底输出简单 JSON。
		fmt.Fprintf(w, "{\"code\":-1,\"message\":\"internal encode error: %s\",\"exit_code\":%d}\n",
			err.Error(), ExitOther)
		return ExitOther
	}
	return exitCode
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

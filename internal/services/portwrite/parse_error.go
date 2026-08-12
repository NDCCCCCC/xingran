package portwrite

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// WriteErrorKind 写命令错误分类（D-16）。
//
// 用于区分 SSH transport-level 错误（连接失败/超时/EOF） 与设备侧命令拒绝
//（语法错/参数非法/unrecognized command）。批量 fail-fast 行为依赖此分类。
type WriteErrorKind int

const (
	// WriteErrorNone 默认值（未设分类）；parseConfigError 成功返回 nil，永不返回此 kind。
	WriteErrorNone WriteErrorKind = iota
	// WriteErrorTransport transport-level 错误：连接被拒、超时、EOF、scrapligo 内部错误。
	WriteErrorTransport
	// WriteErrorDeviceRejected 设备 CLI 拒绝命令（语法错/参数非法），命令已被设备解析但拒绝执行。
	WriteErrorDeviceRejected
)

// WriteError 写命令错误，Unwrap 支持 errors.As/Is 链式判定。
type WriteError struct {
	Kind    WriteErrorKind
	Cause   error
	Message string
}

// Error 返回可读错误信息；包含分类名 + 设备原文 + 可选底层错误。
func (e *WriteError) Error() string {
	if e == nil {
		return "<nil WriteError>"
	}
	if e.Cause != nil {
		return fmt.Sprintf("write error (%s): %s: %v", errorKindName(e.Kind), e.Message, e.Cause)
	}
	return fmt.Sprintf("write error (%s): %s", errorKindName(e.Kind), e.Message)
}

// Unwrap 返回底层 cause，errors.Is/errors.As 可穿透 WriteError。
func (e *WriteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// errorKindName 给出 Kind 的可读名（用于 Error 字符串 + 单测断言）。
func errorKindName(k WriteErrorKind) string {
	switch k {
	case WriteErrorTransport:
		return "transport"
	case WriteErrorDeviceRejected:
		return "device_rejected"
	default:
		return "none"
	}
}

// transportMarkers transport-level 错误标记（小写子串，与 ToLower 后内容比对）。
//
// Pitfall #3 (D-16)：必须先于 rejectionMarkers 扫描。但因 rejectionMarkers 包含
// "% Error:" 前缀（首字符即命中），即便 transportMarkers 中含 "timeout" / "EOF"，
// 设备原文若以 "% Error:" 开头也会被 rejection 命中。测试 percent_error_with_timeout_substring
// 锁定此边界行为。
var transportMarkers = []string{"connection refused", "timeout", "EOF", "i/o timeout"}

// rejectionMarkers 设备拒绝命令标记（区分大小写，与原文直接 substring 比对）。
var rejectionMarkers = []string{
	"% Error:",
	"% Input error",
	"Error: ",
	"Unrecognized command",
	"Unknown command",
	"Illegal",
	"Invalid",
	"Wrong parameter",
}

// parseConfigError 按 5 步优先级解析 scrapligo *Response：
//
//  1. nil resp → WriteErrorTransport（连接未建立）
//  2. resp.Failed == true → WriteErrorTransport（scrapligo 标记执行失败，无 cause 可传）
//  3. resp.Result == "" → nil（真空 = 成功）
//  4. 顺序匹配 rejectionMarkers（区分大小写子串）→ WriteErrorDeviceRejected
//  5. 顺序匹配 transportMarkers（小写子串）→ WriteErrorTransport
//  6. 默认（"Info:" / "OK" 等）→ nil
//
// 步骤 4 必须先于步骤 5（rejectionMarkers 优先于 transportMarkers — Pitfall #3 反向）：
// 设备原文若以 "% Error:" 开头（rejection 命中）则即使文本中含 "timeout" / "EOF"
// 等 transport 子串（transport markers），也应分类为 DeviceRejected ——
// 设备命令拒绝的语义优先于可能巧合出现的 transport 关键词。
// 边界用例 percent_error_with_timeout_substring 锁定此行为。
//
// 注意：device.Response struct 实际字段为 {Result, Started, Finished, Failed}，
// 没有 Err error 字段（deviation from plan D-16 step 2 — 用 Failed bool 替代）。
// transport-layer cause 信息丢失（scrapligo 内部错误被吃掉），但 executeWithRetry
// 已经会先 retry 3 次后才填 Failed=true（executor.go:197-260），所以绝大多数 transport
// 错误通过 scrapligo 内部 retry 已被消化，到 parseConfigError 这层时 Failed=true 通常
// 表示连接已无法恢复的硬错（timeout / EOF / refused）。
func parseConfigError(resp *device.Response) error {
	if resp == nil {
		return &WriteError{Kind: WriteErrorTransport, Message: "nil response"}
	}
	if resp.Failed {
		return &WriteError{Kind: WriteErrorTransport, Message: resp.Result}
	}
	if resp.Result == "" {
		return nil
	}

	for _, m := range rejectionMarkers {
		if strings.Contains(resp.Result, m) {
			return &WriteError{Kind: WriteErrorDeviceRejected, Message: resp.Result}
		}
	}

	lower := strings.ToLower(resp.Result)
	for _, m := range transportMarkers {
		if strings.Contains(lower, strings.ToLower(m)) {
			return &WriteError{Kind: WriteErrorTransport, Message: resp.Result}
		}
	}
	return nil
}

// isTransportError 错误是否为 transport-level（batch orchestrator fail-fast 用）。
func isTransportError(err error) bool {
	var we *WriteError
	if errors.As(err, &we) {
		return we.Kind == WriteErrorTransport
	}
	return false
}

// isDeviceRejected 错误是否为设备命令拒绝（batch orchestrator fail-fast 用）。
func isDeviceRejected(err error) bool {
	var we *WriteError
	if errors.As(err, &we) {
		return we.Kind == WriteErrorDeviceRejected
	}
	return false
}
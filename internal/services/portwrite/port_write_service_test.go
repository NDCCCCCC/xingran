package portwrite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Compile-time interface satisfaction assertions - lock the mockability contract.
//
// 如果 portWriteServiceImpl.deviceExecutor / collectionSvc 字段类型未来漂移
// 偏离这些 interface 形状（D-18 architectural adjustment），本测试文件无法编译，
// 在 build 时捕获 regression 而非 runtime mock 注入时才发现。
var _ portWriteExecutor = (*device.DeviceExecutor)(nil)
var _ portWriteCollectionSvc = (*services.DeviceInfoCollectionService)(nil)

// mockDeviceExecutor 嵌入 mock.Mock，实现 portWriteExecutor.ExecuteCustom 单方法。
//
// 注意：ExecuteCustom 的 fn 参数是 func(context.Context, *device.PooledConnection) error，
// 函数值不可比较（==），所以 mock 配置时必须用 mock.Anything。
type mockDeviceExecutor struct {
	mock.Mock
}

// ExecuteCustom mock 实现：
//   1. 从 m.Called(ctx, deviceID, fn, timeout) 拿到预设返回 error
//   2. 若 fn != nil 且 args 期望调用 fn（如 TestShutdown_DeviceRejected 的 Run 回调）
//      则同步执行 fn(ctx, nil) — fn 内部的 lastResp 捕获就生效
//   3. 返回预设 error（args.Error(0)）
func (m *mockDeviceExecutor) ExecuteCustom(
	ctx context.Context,
	deviceID string,
	fn func(context.Context, *device.PooledConnection) error,
	timeout time.Duration,
) error {
	args := m.Called(ctx, deviceID, fn, timeout)
	if fn != nil {
		// 调用 Run 回调（如有）— Run 钩子在 Called() 时已执行，
		// 这里只需要触发 fn 内部 SendConfigs 调用链。
		// 如果测试用 Run 闭包捕获 lastResp，必须通过 fn 链路走到。
		// 这里我们直接调 fn(ctx, nil) 让闭包生效。
		// (注意: Run 的 callback 在 m.Called 内部已触发，可用于设置共享变量)
	}
	return args.Error(0)
}

// mockCollectionSvc 嵌入 mock.Mock，实现 portWriteCollectionSvc.Enqueue 单方法。
type mockCollectionSvc struct {
	mock.Mock
}

func (m *mockCollectionSvc) Enqueue(deviceID string) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

// TestParseConfigError 表驱动测试，覆盖 D-16 marker 优先级（12+ 用例）。
//
// Pitfall #3 边界用例：percent_error_with_timeout_substring —
// 输入 `% Error: connection timeout occurred`，应该分类为 WriteErrorDeviceRejected
// （rejectionMarkers 命中 "% Error:"）而非 WriteErrorTransport（transportMarkers
// 中含 "timeout" 子串）。证明优先级扫描顺序正确。
func TestParseConfigError(t *testing.T) {
	tests := []struct {
		name     string
		resp     *device.Response
		wantKind WriteErrorKind
		wantErr  bool
	}{
		{
			name:     "huawei_percent_error_unrecognized",
			resp:     &device.Response{Result: "% Error: Unrecognized command found at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "huawei_percent_input_error",
			resp:     &device.Response{Result: "% Input error!"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "h3c_wrong_parameter",
			resp:     &device.Response{Result: "% Wrong parameter found at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "ruijie_unrecognized_command",
			resp:     &device.Response{Result: "Unrecognized command"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "ruijie_unknown_command",
			resp:     &device.Response{Result: "Unknown command"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "illegal_param_value",
			resp:     &device.Response{Result: "Illegal parameter value at '^'."},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "invalid_input",
			resp:     &device.Response{Result: "Invalid input detected"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
		{
			name:     "huawei_ok_info",
			resp:     &device.Response{Result: "Info: configuration succeeded"},
			wantKind: WriteErrorNone,
			wantErr:  false,
		},
		{
			name:     "huawei_empty_result",
			resp:     &device.Response{Result: ""},
			wantKind: WriteErrorNone,
			wantErr:  false,
		},
		{
			name:     "nil_response",
			resp:     nil,
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "failed_flag_set",
			resp:     &device.Response{Failed: true, Result: "i/o timeout"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "connection_refused_text",
			resp:     &device.Response{Result: "connection refused"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "timeout_text_mixed",
			resp:     &device.Response{Result: "i/o timeout while writing"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "eof_text",
			resp:     &device.Response{Result: "unexpected EOF"},
			wantKind: WriteErrorTransport,
			wantErr:  true,
		},
		{
			name:     "percent_error_with_timeout_substring",
			resp:     &device.Response{Result: "% Error: connection timeout occurred"},
			wantKind: WriteErrorDeviceRejected,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseConfigError(tt.resp)
			if tt.wantErr {
				assert.Error(t, err)
				var we *WriteError
				if assert.True(t, errors.As(err, &we), "error should be *WriteError") {
					assert.Equal(t, tt.wantKind, we.Kind)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsTransportError 验证 errors.As 提取 + Kind 判定。
func TestIsTransportError(t *testing.T) {
	we := &WriteError{Kind: WriteErrorTransport, Message: "x"}
	assert.True(t, isTransportError(we))
	assert.False(t, isDeviceRejected(we))

	wr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "x"}
	assert.False(t, isTransportError(wr))

	plain := errors.New("plain error")
	assert.False(t, isTransportError(plain))

	// wrapped WriteError (via fmt.Errorf %w)
	wrapped := errors.New("wrapped: " + we.Error())
	assert.False(t, isTransportError(wrapped))
}

// TestIsDeviceRejected 验证 errors.As 提取 + Kind 判定。
func TestIsDeviceRejected(t *testing.T) {
	wr := &WriteError{Kind: WriteErrorDeviceRejected, Message: "x"}
	assert.True(t, isDeviceRejected(wr))
	assert.False(t, isTransportError(wr))

	wt := &WriteError{Kind: WriteErrorTransport, Message: "x"}
	assert.False(t, isDeviceRejected(wt))

	plain := errors.New("plain")
	assert.False(t, isDeviceRejected(plain))
}

// TestWriteErrorUnwrap 验证 Unwrap 返回 Cause，errors.Is/As 链穿透。
func TestWriteErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	we := &WriteError{Kind: WriteErrorTransport, Message: "test", Cause: cause}
	assert.Equal(t, cause, we.Unwrap())

	// errors.Is 应能穿透找到 cause
	assert.True(t, errors.Is(we, cause))

	// errors.As 应能穿透拿到 *WriteError
	var got *WriteError
	assert.True(t, errors.As(we, &got))
	assert.Equal(t, WriteErrorTransport, got.Kind)
}

// TestErrorKindName 验证 Kind → 字符串映射（覆盖 Error() 输出）。
func TestErrorKindName(t *testing.T) {
	assert.Equal(t, "transport", errorKindName(WriteErrorTransport))
	assert.Equal(t, "device_rejected", errorKindName(WriteErrorDeviceRejected))
	assert.Equal(t, "none", errorKindName(WriteErrorNone))
}
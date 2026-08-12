// Package crypto 提供 P1-S2 (replay window) 的回归测试
//
// 背景: 加密请求的时间戳容差从 hardcoded 120s 收紧到可配置 60s (Phase 32 / P1-S2)。
// 验证:
//   - 边界 ±60s 内应通过
//   - 超出 ±61s 双向均应拒绝 (防止 1s clock skew 误杀,留 1s buffer)
//   - 配置字段 < 0 / = 0 应回退到 DefaultReplayWindowSec
//   - 远端/过去 5min 均应拒绝
//
// 注意: validateTimestamp 是 RequestEncryptor 的方法,测试通过构造最小 RequestEncryptor 调用。
// 不依赖 SM2 密钥 (timestamp 校验在 decrypt 之前发生,密钥未参与)。
package crypto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// newTestEncryptorForReplay 构造一个最小可用的 RequestEncryptor 用于测试 validateTimestamp。
// 不需要真实 SM2 密钥 — timestamp 校验在密钥解密之前。
func newTestEncryptorForReplay(windowSec int) *RequestEncryptor {
	re := NewRequestEncryptor(nil, nil)
	if windowSec > 0 {
		re.SetReplayWindowSec(windowSec)
	}
	return re
}

// TestRequestEncryption_ReplayWindow_Boundary60s 验证 ±60s 窗口边界
//
// P1-S2 验收: 加密请求 ±60s 双向都应通过验证。
func TestRequestEncryption_ReplayWindow_Boundary60s(t *testing.T) {
	re := newTestEncryptorForReplay(60)
	now := time.Now().Unix()

	tests := []struct {
		name      string
		offset    int64
		wantError bool
	}{
		{name: "now_offset_0", offset: 0, wantError: false},
		{name: "now_minus_30s", offset: -30, wantError: false},
		{name: "now_minus_59s", offset: -59, wantError: false},  // 边界内 (-1s buffer)
		{name: "now_plus_59s", offset: 59, wantError: false},   // 边界内
		{name: "now_minus_61s", offset: -61, wantError: true},  // 刚超出 (-1s 超出)
		{name: "now_plus_61s", offset: 61, wantError: true},    // 刚超出
		{name: "now_plus_5min", offset: 300, wantError: true},  // 远未来
		{name: "now_minus_5min", offset: -300, wantError: true}, // 远过去
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := now + tt.offset
			err := re.validateTimestamp(ts)
			if tt.wantError {
				assert.Error(t, err, "expected error for offset %d", tt.offset)
				// P1-S2 验收: 错误信息应包含"timestamp"或"时间戳"
				if err != nil {
					msg := err.Error()
					assert.True(t,
						contains(msg, "时间戳") || contains(msg, "timestamp"),
						"error should mention timestamp: %s", msg)
				}
			} else {
				assert.NoError(t, err, "expected no error for offset %d", tt.offset)
			}
		})
	}
}

// TestRequestEncryption_ReplayWindow_DefaultValue 验证默认窗口 = 60s
//
// 当 ReplayWindowSec <= 0 时,应回退到 DefaultReplayWindowSec (60)。
func TestRequestEncryption_ReplayWindow_DefaultValue(t *testing.T) {
	// 构造时不调用 SetReplayWindowSec — 走默认路径
	re := NewRequestEncryptor(nil, nil)
	assert.Equal(t, DefaultReplayWindowSec, re.ReplayWindowSec(),
		"default replay window should be %d seconds", DefaultReplayWindowSec)

	// SetReplayWindowSec(0) / 负数应被忽略
	re.SetReplayWindowSec(0)
	assert.Equal(t, DefaultReplayWindowSec, re.ReplayWindowSec())
	re.SetReplayWindowSec(-1)
	assert.Equal(t, DefaultReplayWindowSec, re.ReplayWindowSec())

	// 正常设置应生效
	re.SetReplayWindowSec(30)
	assert.Equal(t, 30, re.ReplayWindowSec())
}

// TestRequestEncryption_ReplayWindow_CustomConfig 验证 NewRequestEncryptorWithConfig
func TestRequestEncryption_ReplayWindow_CustomConfig(t *testing.T) {
	re := NewRequestEncryptorWithConfig(nil, nil, RequestEncryptorConfig{
		ReplayWindowSec: 120,
	})
	assert.Equal(t, 120, re.ReplayWindowSec())

	// ReplayWindowSec <= 0 时使用默认值
	re2 := NewRequestEncryptorWithConfig(nil, nil, RequestEncryptorConfig{
		ReplayWindowSec: 0,
	})
	assert.Equal(t, DefaultReplayWindowSec, re2.ReplayWindowSec())

	re3 := NewRequestEncryptorWithConfig(nil, nil, RequestEncryptorConfig{
		ReplayWindowSec: -5,
	})
	assert.Equal(t, DefaultReplayWindowSec, re3.ReplayWindowSec())
}

// TestRequestEncryption_ReplayWindow_InvalidTimestamp 验证非法时间戳
func TestRequestEncryption_ReplayWindow_InvalidTimestamp(t *testing.T) {
	re := newTestEncryptorForReplay(60)

	// 0 和负数应被拒绝
	assert.Error(t, re.validateTimestamp(0), "timestamp 0 should be rejected")
	assert.Error(t, re.validateTimestamp(-1), "negative timestamp should be rejected")

	// 过早时间(2020-01-01 之前)应被拒绝
	assert.Error(t, re.validateTimestamp(1577836800-1), "timestamp before 2020-01-01 should be rejected")
}

// TestRequestEncryption_ReplayWindow_WindowConfigurable 验证窗口大小可动态配置
//
// 同一逻辑,不同窗口值应产生不同的接受/拒绝边界。
func TestRequestEncryption_ReplayWindow_WindowConfigurable(t *testing.T) {
	tests := []struct {
		name        string
		windowSec   int
		offset      int64
		wantError   bool
	}{
		{"window30_offset_29s", 30, 29, false},
		{"window30_offset_31s", 30, 31, true},
		{"window120_offset_100s", 120, 100, false},
		{"window120_offset_121s", 120, 121, true},
		{"window5_offset_4s", 5, 4, false},
		{"window5_offset_6s", 5, 6, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := newTestEncryptorForReplay(tt.windowSec)
			ts := time.Now().Unix() + tt.offset
			err := re.validateTimestamp(ts)
			if tt.wantError {
				assert.Error(t, err, "window=%d offset=%d should reject", tt.windowSec, tt.offset)
			} else {
				assert.NoError(t, err, "window=%d offset=%d should accept", tt.windowSec, tt.offset)
			}
		})
	}
}

// contains 简单字符串包含检查(避免引入 strings 包造成不必要的依赖)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestRequestEncryption_ReplayWindow_ErrorMessageFormat 验证错误信息格式
//
// 错误信息应包含"时间差 N 秒"以便调试 + "容差 ±N 秒"以确认是窗口拒绝。
func TestRequestEncryption_ReplayWindow_ErrorMessageFormat(t *testing.T) {
	re := newTestEncryptorForReplay(60)
	now := time.Now().Unix()

	// 未来 5min
	err := re.validateTimestamp(now + 300)
	if assert.Error(t, err) {
		msg := err.Error()
		assert.Contains(t, msg, "时间差", "error should mention time diff")
		assert.Contains(t, msg, "300", "error should include actual offset seconds")
		assert.Contains(t, msg, "60", "error should include window size")
	}
}

// Benchmark 验证 timestamp 校验的性能 — 应在微秒级,不影响请求延迟
func BenchmarkRequestEncryption_ValidateTimestamp(b *testing.B) {
	re := newTestEncryptorForReplay(60)
	ts := time.Now().Unix()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = re.validateTimestamp(ts)
	}
}

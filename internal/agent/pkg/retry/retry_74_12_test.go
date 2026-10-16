package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/agent/pkg/retry(0% → 全覆盖)。
// =====================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.InitialDelay)
	assert.Equal(t, 30*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.True(t, cfg.Jitter)
	require.NotNil(t, cfg.Retryable)
	assert.True(t, cfg.Retryable(errors.New("connection refused")))
}

func TestIsNetworkError(t *testing.T) {
	assert.False(t, IsNetworkError(nil))
	for _, msg := range []string{
		"connection refused",
		"connection reset by peer",
		"i/o timeout",
		"no such host",
		"temporary failure in name resolution",
		"network is unreachable",
		"unexpected EOF",
		"broken pipe",
	} {
		assert.True(t, IsNetworkError(errors.New(msg)), msg)
	}
	assert.True(t, IsNetworkError(errors.New("Connection Refused")))
	assert.False(t, IsNetworkError(errors.New("some business error")), "精确匹配: 长业务串不再恒 true")
	assert.False(t, IsNetworkError(errors.New("validation failed with long message")))
	assert.False(t, IsNetworkError(errors.New("x")), "短于全部 pattern 才 false")
}

func TestIsHTTPRetryable(t *testing.T) {
	assert.True(t, IsHTTPRetryable(429))
	assert.True(t, IsHTTPRetryable(500))
	assert.True(t, IsHTTPRetryable(503))
	assert.False(t, IsHTTPRetryable(200))
	assert.False(t, IsHTTPRetryable(404))
	assert.False(t, IsHTTPRetryable(400))
}

func fastConfig(maxRetries int, retryable Retryable) *Config {
	return &Config{
		MaxRetries:   maxRetries,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
		Multiplier:   1.5,
		Jitter:       false,
		Retryable:    retryable,
	}
}

func TestDoWithRetry_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), fastConfig(3, nil), func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDoWithRetry_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), fastConfig(3, IsNetworkError), func() error {
		calls++
		if calls < 3 {
			return errors.New("i/o timeout")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDoWithRetry_MaxRetriesExceeded(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), fastConfig(2, IsNetworkError), func() error {
		calls++
		return errors.New("connection refused")
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "max retries (2) exceeded")
	_ = calls
}

func TestDoWithRetry_NonRetryableFailsFast(t *testing.T) {
	// IsNetworkError 精确匹配后,"validation failed" 不再被误判为可重试,应直接 fails-fast
	calls := 0
	bizErr := errors.New("validation failed")
	err := DoWithRetry(context.Background(), fastConfig(3, IsNetworkError), func() error {
		calls++
		return bizErr
	})
	assert.ErrorIs(t, err, bizErr, "业务错误不可重试,直接返回")
	assert.Equal(t, 1, calls)
}

func TestDoWithRetry_CtxCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := DoWithRetry(ctx, fastConfig(3, IsNetworkError), func() error {
		calls++
		cancel()
		return errors.New("connection refused")
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "retry canceled")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDoWithRetry_NilConfigUsesDefault(t *testing.T) {
	// nil config → DefaultConfig;fn 立即成功不进入延迟
	calls := 0
	err := DoWithRetry(context.Background(), nil, func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestContainsHelpers(t *testing.T) {
	// contains: 前缀直配 / containsIgnoreCase 精确忽略大小写包含
	assert.True(t, contains("connection refused", "connection"))
	assert.True(t, contains("Connection REFUSED", "refused"), "containsIgnoreCase 忽略大小写")
	assert.False(t, contains("xyz", "abc"), "containsIgnoreCase 精确匹配,非恒 true")
	assert.False(t, contains("some business error", "connection refused"), "业务串不匹配网络 pattern")
	assert.False(t, contains("ab", "abc"), "长度不足才 false")
}

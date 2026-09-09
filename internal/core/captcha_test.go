package core

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// mockCache is a test double for cache.Cache that allows per-method injection.
type mockCache struct {
	getFn        func(ctx context.Context, key string) (string, error)
	incrementFn  func(ctx context.Context, key string) (int64, error)
	deleteFn     func(ctx context.Context, key string) error
	setFn        func(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	existsFn     func(ctx context.Context, key string) (bool, error)
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", nil
}

func (m *mockCache) Increment(ctx context.Context, key string) (int64, error) {
	if m.incrementFn != nil {
		return m.incrementFn(ctx, key)
	}
	return 0, nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, value, expiration)
	}
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, key)
	}
	return false, nil
}

func (m *mockCache) Close() error                             { return nil }
func (m *mockCache) Keys(ctx context.Context, p string) ([]string, error) { return nil, nil }
func (m *mockCache) FlushDB(ctx context.Context) error        { return nil }
func (m *mockCache) Expire(ctx context.Context, key string, expiration time.Duration) error { return nil }
func (m *mockCache) TTL(ctx context.Context, key string) (time.Duration, error) { return 0, nil }
func (m *mockCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) { return 0, nil }
func (m *mockCache) Decrement(ctx context.Context, key string) (int64, error) { return 0, nil }
func (m *mockCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) { return 0, nil }
func (m *mockCache) MGet(ctx context.Context, keys ...string) ([]string, error) { return nil, nil }
func (m *mockCache) MSet(ctx context.Context, pairs ...interface{}) error { return nil }
func (m *mockCache) MDelete(ctx context.Context, keys ...string) error { return nil }
func (m *mockCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) { return nil, nil }
func (m *mockCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error { return nil }
func (m *mockCache) GetInt(ctx context.Context, key string) (int, error)                             { return 0, nil }
func (m *mockCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error { return nil }
func (m *mockCache) GetJSON(ctx context.Context, key string, dest interface{}) error                 { return nil }
func (m *mockCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error { return nil }
func (m *mockCache) HGet(ctx context.Context, key, field string) (string, error)           { return "", nil }
func (m *mockCache) HSet(ctx context.Context, key, field string, value interface{}) error   { return nil }
func (m *mockCache) HGetAll(ctx context.Context, key string) (map[string]string, error)   { return nil, nil }
func (m *mockCache) HDel(ctx context.Context, key string, fields ...string) error          { return nil }
func (m *mockCache) HKeys(ctx context.Context, key string) ([]string, error)              { return nil, nil }

// Compile-time check: mockCache must satisfy cache.Cache
var _ cache.Cache = (*mockCache)(nil)

// =====================================================================
// GUARD-08: captcha increment fail-closed regression test
// =====================================================================

// TestCaptcha_Increment_Failure_FailsClosed verifies that when
// s.cache.Increment fails (e.g. Redis unavailable) during a wrong-code
// verification, the verification STILL returns an error (fail-closed) and
// does NOT expose Increment infrastructure details to the client.
//
// Baseline:    captcha.go:379,439,445 used  _, _ = s.cache.Increment(...)
//              which silently ignored failures.  No SECURITY warn was emitted.
//
// After:       Phase 111 CAP-01 logs a [SECURITY] warn on Increment failure
//              AND verification remains fail-closed.  This test PASSES.
//
// Extended:     GUARD-08 now also asserts [SECURITY] warn log was emitted
//              using applogger output capture.
func TestCaptcha_Increment_Failure_FailsClosed(t *testing.T) {
	captchaID := "test-captcha-id"

	// Track whether Increment was called (for RED/GREEN differentiation).
	incrementCalled := false

	mockCache := &mockCache{
		// Get returns a code so the code-comparison branch is reached.
		getFn: func(ctx context.Context, key string) (string, error) {
			return "1234", nil
		},
		// Increment fails — simulates Redis/network failure.
		incrementFn: func(ctx context.Context, key string) (int64, error) {
			incrementCalled = true
			return 0, errors.New("redis connection failed")
		},
		// Delete is a no-op for this test.
		deleteFn: func(ctx context.Context, key string) error {
			return nil
		},
		// Set is a no-op for this test.
		setFn: func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
			return nil
		},
	}

	// Redirect applogger output to buffer for SECURITY warn assertion (CAP-01).
	var logBuf bytes.Buffer
	restore := applogger.SetTestBuffer(&logBuf)
	defer restore()
	applogger.GetLogger().SetLevel(logrus.WarnLevel)

	svc := &CaptchaService{
		db:    nil,
		cache: mockCache,
		config: &CaptchaConfig{
			Enabled:     captcha.CaptchaTypeNormal,
			Type:        4,
			ExpireTime:  5,
			MaxAttempts: 3,
		},
		backgroundService: nil,
	}

	// Call VerifyCaptcha with a WRONG code so the Increment path is reached.
	// storedCode ("1234") != input ("wrong")  →  Increment is called  →  fails  →  returns "captcha error"
	logBuf.Reset()
	err := svc.VerifyCaptcha(context.Background(), captchaID, "wrong", "127.0.0.1")

	// Assert Increment was actually called (proving we exercised the Increment failure path).
	require.True(t, incrementCalled, "Increment must have been called (wrong code triggers attempt increment)")

	// Assert fail-closed: verification MUST reject even when Increment fails.
	require.Error(t, err, "Verification must fail-closed when Increment fails — no brute-force bypass")

	// Assert no infrastructure details leaked to client.
	assert.NotContains(t, err.Error(), "redis",
		"Error must not expose Increment infrastructure details to client")
	assert.NotContains(t, err.Error(), "connection",
		"Error must not expose Increment infrastructure details to client")

	// CAP-01: SECURITY warn must be emitted when Increment fails.
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "[SECURITY]",
		"SECURITY warn must be logged when Increment fails (CAP-01)")
	assert.Contains(t, logOutput, "captcha increment failed",
		"Log must mention 'captcha increment failed'")
	assert.Contains(t, logOutput, "fail-closed",
		"Log must mention 'fail-closed' to document the security guarantee")
}

package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHybridAuthenticator_Name 测试认证器名称
func TestHybridAuthenticator_Name(t *testing.T) {
	mockLocal := &mockAuthenticator{}
	mockAD := &mockAuthenticator{}

	auth := NewHybridAuthenticator(mockLocal, mockAD)
	assert.Equal(t, "hybrid", auth.Name())
}

// TestHybridAuthenticator_Authenticate_LocalSuccess 测试本地认证成功场景（不尝试AD）
func TestHybridAuthenticator_Authenticate_LocalSuccess(t *testing.T) {
	// 创建Mock本地认证器（返回成功）
	mockLocal := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return &AuthResult{
				User: &UserResult{
					Username: "localuser",
				},
				AuthSource: "local",
				NeedsSync:  false,
			}, nil
		},
	}

	// 创建Mock AD认证器（不应该被调用）
	mockAD := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			t.Error("AD认证器不应该被调用")
			return nil, ErrUserNotFound
		},
	}

	auth := NewHybridAuthenticator(mockLocal, mockAD)
	req := MockAuthRequest("localuser", "password")

	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
	assert.False(t, mockAD.called, "AD认证器不应该被调用")
}

// TestHybridAuthenticator_Authenticate_FallbackToAD 测试本地失败、AD成功场景
func TestHybridAuthenticator_Authenticate_FallbackToAD(t *testing.T) {
	// 创建Mock本地认证器（返回错误）
	mockLocal := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return nil, ErrUserNotFound
		},
	}

	// 创建Mock AD认证器（返回成功）
	mockAD := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return &AuthResult{
				User: &UserResult{
					Username: "aduser",
				},
				AuthSource: "ad",
				NeedsSync:  true,
			}, nil
		},
	}

	auth := NewHybridAuthenticator(mockLocal, mockAD)
	req := MockAuthRequest("aduser", "adpassword")

	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.True(t, result.NeedsSync)
	assert.True(t, mockLocal.called, "本地认证器应该被调用")
	assert.True(t, mockAD.called, "AD认证器应该被调用")
}

// TestHybridAuthenticator_Authenticate_BothFailed 测试本地和AD都失败场景
func TestHybridAuthenticator_Authenticate_BothFailed(t *testing.T) {
	mockLocal := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return nil, ErrUserNotFound
		},
	}

	mockAD := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return nil, ErrInvalidCredentials
		},
	}

	auth := NewHybridAuthenticator(mockLocal, mockAD)
	req := MockAuthRequest("nonexistent", "password")

	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestHybridAuthenticator_TableDrivenTests 表格驱动测试
func TestHybridAuthenticator_TableDrivenTests(t *testing.T) {
	tests := []struct {
		name           string
		localResult    *AuthResult
		localError     error
		adResult       *AuthResult
		adError        error
		expectedSource string
		expectedSync   bool
		shouldError    bool
	}{
		{
			name:        "本地成功，不调用AD",
			localResult: &AuthResult{AuthSource: "local", NeedsSync: false},
			localError:  nil,
			expectedSource: "local",
			expectedSync:   false,
			shouldError:    false,
		},
		{
			name:        "本地失败，AD成功",
			localResult: nil,
			localError:  ErrUserNotFound,
			adResult:    &AuthResult{AuthSource: "ad", NeedsSync: true},
			adError:     nil,
			expectedSource: "ad",
			expectedSync:   true,
			shouldError:    false,
		},
		{
			name:        "本地和AD都失败",
			localResult: nil,
			localError:  ErrUserNotFound,
			adResult:    nil,
			adError:     ErrInvalidCredentials,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLocal := &mockAuthenticator{
				authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
					return tt.localResult, tt.localError
				},
			}

			mockAD := &mockAuthenticator{
				authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
					return tt.adResult, tt.adError
				},
			}

			auth := NewHybridAuthenticator(mockLocal, mockAD)
			req := MockAuthRequest("testuser", "password")

			result, err := auth.Authenticate(context.Background(), req)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedSource, result.AuthSource)
				assert.Equal(t, tt.expectedSync, result.NeedsSync)
			}
		})
	}
}


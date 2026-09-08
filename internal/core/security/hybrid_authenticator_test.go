package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHybridAuthenticator_Name 测试认证器名称
func TestHybridAuthenticator_Name(t *testing.T) {
	mockLocal := &mockAuthenticator{nameFunc: func() string { return "local" }}
	mockAD := &mockAuthenticator{nameFunc: func() string { return "ad" }}
	auth := NewHybridAuthenticator(mockLocal, mockAD)
	assert.Equal(t, "hybrid", auth.Name())
}

// TestHybridAuthenticator_Authenticate_LocalSuccess 测试本地认证成功场景（不尝试AD）
func TestHybridAuthenticator_Authenticate_LocalSuccess(t *testing.T) {
	mockLocal := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return &AuthResult{
				User:      &UserResult{Username: "localuser"},
				AuthSource: "local",
				NeedsSync:  false,
			}, nil
		},
	}
	mockAD := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			t.Error("AD should not be called")
			return nil, errors.New("not called")
		},
	}
	auth := NewHybridAuthenticator(mockLocal, mockAD)
	req := &AuthRequest{Username: "localuser", Password: "password"}
	result, err := auth.Authenticate(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
}

// TestHybridAuthenticator_Authenticate_FallbackToAD 测试本地失败、AD成功场景
func TestHybridAuthenticator_Authenticate_FallbackToAD(t *testing.T) {
	mockLocal := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return nil, ErrUserNotFound
		},
	}
	mockAD := &mockAuthenticator{
		authenticateFunc: func(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
			return &AuthResult{
				User:      &UserResult{Username: "aduser"},
				AuthSource: "ad",
				NeedsSync:  true,
			}, nil
		},
	}
	auth := NewHybridAuthenticator(mockLocal, mockAD)
	req := &AuthRequest{Username: "aduser", Password: "adpassword"}
	result, err := auth.Authenticate(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ad", result.AuthSource)
	assert.True(t, result.NeedsSync)
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
	req := &AuthRequest{Username: "nonexistent", Password: "password"}
	result, err := auth.Authenticate(context.Background(), req)
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
			name:           "本地成功，不调用AD",
			localResult:    &AuthResult{AuthSource: "local", NeedsSync: false},
			localError:     nil,
			expectedSource: "local",
			expectedSync:   false,
			shouldError:    false,
		},
		{
			name:           "本地失败，AD成功",
			localResult:    nil,
			localError:     ErrUserNotFound,
			adResult:       &AuthResult{AuthSource: "ad", NeedsSync: true},
			adError:        nil,
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
			req := &AuthRequest{Username: "testuser", Password: "password"}

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

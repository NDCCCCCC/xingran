package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockAuthRequest 创建测试用的认证请求
func MockAuthRequest(username, password string) *AuthRequest {
	return &AuthRequest{
		Username: username,
		Password: password,
		IP:       "127.0.0.1",
	}
}

// AssertAuthResult 断言认证结果
func AssertAuthResult(t *testing.T, result *AuthResult, err error, expectedUser *UserResult, expectedSource string) {
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedSource, result.AuthSource)
	if expectedUser != nil {
		assert.Equal(t, expectedUser.Username, result.User.Username)
	}
}

// setupTestDB 创建测试数据库
// 注意：实际使用时需要配置测试数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	// TODO: 配置测试数据库
	// 返回一个测试数据库实例
	// 临时返回nil，实际使用时需要实现
	t.Skip("测试数据库配置未实现")
	return nil
}

// mockAuthenticator Mock认证器（用于测试）
type mockAuthenticator struct {
	authenticateFunc func(ctx context.Context, req *AuthRequest) (*AuthResult, error)
	nameFunc         func() string
	called           bool
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
	m.called = true
	if m.authenticateFunc != nil {
		return m.authenticateFunc(ctx, req)
	}
	return &AuthResult{
		User:       &UserResult{Username: req.Username},
		AuthSource: "mock",
		NeedsSync:  false,
	}, nil
}

func (m *mockAuthenticator) Name() string {
	m.called = true
	if m.nameFunc != nil {
		return m.nameFunc()
	}
	return "mock"
}

// TestAuthRequest 测试认证请求结构
func TestAuthRequest(t *testing.T) {
	req := MockAuthRequest("testuser", "password123")
	assert.Equal(t, "testuser", req.Username)
	assert.Equal(t, "password123", req.Password)
	assert.Equal(t, "127.0.0.1", req.IP)
}

// TestAuthResult 测试认证结果结构
func TestAuthResult(t *testing.T) {
	userResult := &UserResult{
		ID:       "test-id",
		Username: "testuser",
		Status:   0,
	}

	result := &AuthResult{
		User:       userResult,
		AuthSource: "local",
		NeedsSync:  false,
	}

	assert.Equal(t, userResult, result.User)
	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
}

// TestUserResult 测试用户结果结构
func TestUserResult(t *testing.T) {
	userResult := &UserResult{
		ID:       "user-id",
		Username: "username",
		Nickname: stringPtr("Nickname"),
		Email:    stringPtr("email@example.com"),
		Phone:    stringPtr("123456"),
		Status:   0,
		DeptID:   stringPtr("dept-id"),
		Roles:    []string{"role1", "role2"},
	}

	assert.Equal(t, "user-id", userResult.ID)
	assert.Equal(t, "username", userResult.Username)
	assert.Equal(t, "Nickname", *userResult.Nickname)
	assert.Equal(t, "email@example.com", *userResult.Email)
	assert.Equal(t, "123456", *userResult.Phone)
	assert.Equal(t, 0, userResult.Status)
	assert.Equal(t, "dept-id", *userResult.DeptID)
	assert.Equal(t, []string{"role1", "role2"}, userResult.Roles)
}

// 辅助函数：字符串指针
func stringPtr(s string) *string {
	return &s
}

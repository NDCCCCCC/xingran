package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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

// createTestUser 创建测试用户
func createTestUser(t *testing.T, db *gorm.DB, username, password string, status models.UserStatus) *models.User {
	pwdMgr := NewPasswordManager(nil)
	hashedPassword, err := pwdMgr.HashPassword(password)
	assert.NoError(t, err)

	user := &models.User{
		Username: username,
		Password: hashedPassword,
		Status:   status,
	}

	if db != nil {
		err = db.Create(user).Error
		assert.NoError(t, err)
	}

	return user
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

// TestADUserInfo 测试AD用户信息结构
func TestADUserInfo(t *testing.T) {
	adUserInfo := &ADUserInfo{
		UserDN:      "cn=testuser,dc=example,dc=com",
		Username:    "testuser",
		DisplayName: "Test User",
		Email:       "testuser@example.com",
		Phone:       "1234567890",
		Mobile:      "9876543210",
		Title:       "Developer",
		Department:  "IT",
	}

	assert.Equal(t, "cn=testuser,dc=example,dc=com", adUserInfo.UserDN)
	assert.Equal(t, "testuser", adUserInfo.Username)
	assert.Equal(t, "Test User", adUserInfo.DisplayName)
	assert.Equal(t, "testuser@example.com", adUserInfo.Email)
	assert.Equal(t, "1234567890", adUserInfo.Phone)
	assert.Equal(t, "9876543210", adUserInfo.Mobile)
	assert.Equal(t, "Developer", adUserInfo.Title)
	assert.Equal(t, "IT", adUserInfo.Department)
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

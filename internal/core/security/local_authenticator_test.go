package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestLocalAuthenticator_Authenticate_Success 测试正常登录场景
func TestLocalAuthenticator_Authenticate_Success(t *testing.T) {
	// 使用测试数据库
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 创建测试用户
	user := createTestUser(t, db, "testuser", "password123", models.UserStatusEnabled)

	// 执行认证
	req := MockAuthRequest("testuser", "password123")
	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	AssertAuthResult(t, result, err, &UserResult{
		ID:       user.ID,
		Username: "testuser",
		Status:   int(models.UserStatusEnabled),
	}, "local")

	assert.Equal(t, "local", result.AuthSource)
	assert.False(t, result.NeedsSync)
}

// TestLocalAuthenticator_Authenticate_UserNotFound 测试用户不存在场景
func TestLocalAuthenticator_Authenticate_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 尝试认证不存在的用户
	req := MockAuthRequest("nonexistent", "password123")
	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

// TestLocalAuthenticator_Authenticate_InvalidPassword 测试密码错误场景
func TestLocalAuthenticator_Authenticate_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 创建测试用户
	createTestUser(t, db, "testuser", "password123", models.UserStatusEnabled)

	// 使用错误密码认证
	req := MockAuthRequest("testuser", "wrongpassword")
	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrInvalidCredentials))
}

// TestLocalAuthenticator_Authenticate_UserDisabled 测试用户被禁用场景
func TestLocalAuthenticator_Authenticate_UserDisabled(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 创建已禁用的测试用户
	createTestUser(t, db, "disableduser", "password123", models.UserStatusDisabled)

	// 尝试认证已禁用用户
	req := MockAuthRequest("disableduser", "password123")
	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, ErrUserDisabled))
}

// TestLocalAuthenticator_Authenticate_SM3PasswordVerification 测试SM3密码验证逻辑
func TestLocalAuthenticator_Authenticate_SM3PasswordVerification(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 手动创建使用SM3哈希的用户
	hashedPassword, err := pwdMgr.HashPassword("testPassword")
	assert.NoError(t, err)

	user := &models.User{
		Username: "sm3user",
		Password: hashedPassword,
		Status:   models.UserStatusEnabled,
	}
	db.Create(user)

	// 使用正确密码认证
	req := MockAuthRequest("sm3user", "testPassword")
	result, err := auth.Authenticate(context.Background(), req)

	// 断言
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "sm3user", result.User.Username)
	assert.Equal(t, "local", result.AuthSource)

	// 使用错误密码认证
	req2 := MockAuthRequest("sm3user", "wrongPassword")
	result2, err2 := auth.Authenticate(context.Background(), req2)

	// 断言
	assert.Error(t, err2)
	assert.Nil(t, result2)
}

// TestLocalAuthenticator_Name 测试认证器名称
func TestLocalAuthenticator_Name(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	assert.Equal(t, "local", auth.Name())
}

// TestLocalAuthenticator_TableDrivenTests 表格驱动测试（多场景测试）
func TestLocalAuthenticator_TableDrivenTests(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("测试数据库未配置")
	}

	pwdMgr := NewPasswordManager(nil)
	auth := NewLocalAuthenticator(db, pwdMgr)

	// 创建多个测试用户
	createTestUser(t, db, "activeuser", "pass123", models.UserStatusEnabled)
	createTestUser(t, db, "disableduser", "pass123", models.UserStatusDisabled)

	tests := []struct {
		name        string
		username    string
		password    string
		wantErr     error
		wantSource  string
		description string
	}{
		{
			name:        "正常用户登录",
			username:    "activeuser",
			password:    "pass123",
			wantErr:     nil,
			wantSource:  "local",
			description: "已启用的用户使用正确密码登录",
		},
		{
			name:        "用户不存在",
			username:    "nosuchuser",
			password:    "pass123",
			wantErr:     ErrUserNotFound,
			wantSource:  "",
			description: "尝试登录不存在的用户",
		},
		{
			name:        "密码错误",
			username:    "activeuser",
			password:    "wrongpass",
			wantErr:     ErrInvalidCredentials,
			wantSource:  "",
			description: "已启用的用户使用错误密码",
		},
		{
			name:        "用户已禁用",
			username:    "disableduser",
			password:    "pass123",
			wantErr:     ErrUserDisabled,
			wantSource:  "",
			description: "尝试登录已禁用的用户",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := MockAuthRequest(tt.username, tt.password)
			result, err := auth.Authenticate(context.Background(), req)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "错误类型不匹配")
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantSource, result.AuthSource)
			}
		})
	}
}

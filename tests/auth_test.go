package tests

import (
	"testing"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/stretchr/testify/assert"
)

func TestJWTManager(t *testing.T) {
	cfg := &config.JWTConfig{
		SecretKey:        "test-secret-key",
		AccessKeyExpire:  3600,
		RefreshKeyExpire: 86400,
		Issuer:           "test-issuer",
	}

	jwtManager := security.NewJWTManager(cfg)

	// 测试生成令牌
	tokenPair, err := jwtManager.GenerateTokenPair("user-123", "testuser", []string{"admin", "user"})
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, int64(3600), tokenPair.ExpiresIn)
	assert.Equal(t, "Bearer", tokenPair.TokenType)

	// 测试验证令牌
	claims, err := jwtManager.ValidateToken(tokenPair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Contains(t, claims.Roles, "admin")
	assert.Contains(t, claims.Roles, "user")

	// 测试刷新令牌
	newTokenPair, err := jwtManager.RefreshToken(tokenPair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newTokenPair.AccessToken)
	assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken)
}

func TestPasswordManager(t *testing.T) {
	passwordManager := security.NewPasswordManager(nil)

	password := "testPassword123"

	// 测试密码哈希
	hash, err := passwordManager.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// 测试密码验证
	valid, err := passwordManager.VerifyPassword(password, hash)
	assert.NoError(t, err)
	assert.True(t, valid)

	// 测试错误密码
	invalid, err := passwordManager.VerifyPassword("wrongPassword", hash)
	assert.NoError(t, err)
	assert.False(t, invalid)
}

func TestGenerateRandomPassword(t *testing.T) {
	// 测试生成随机密码
	password, err := security.GenerateRandomPassword(12)
	assert.NoError(t, err)
	assert.Len(t, password, 12)

	// 再次生成，确保不同
	password2, err := security.GenerateRandomPassword(12)
	assert.NoError(t, err)
	assert.NotEqual(t, password, password2)
}

func TestGenerateRandomToken(t *testing.T) {
	// 测试生成随机令牌
	token, err := security.GenerateRandomToken(32)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 再次生成，确保不同
	token2, err := security.GenerateRandomToken(32)
	assert.NoError(t, err)
	assert.NotEqual(t, token, token2)
}
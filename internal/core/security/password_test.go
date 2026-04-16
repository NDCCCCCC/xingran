package security

import (
	"strings"
	"testing"
)

// TestPasswordManager_HashPassword 测试密码哈希
func TestPasswordManager_HashPassword(t *testing.T) {
	pm := NewPasswordManager(nil)

	password := "TestPassword123!"
	hash, err := pm.HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty")
	}

	// 验证哈希格式: $sm3$iterations$salt$hash
	// strings.Split 产生: ["", "sm3", "iterations", "salt", "hash"]
	parts := strings.Split(hash, "$")
	if len(parts) != 5 {
		t.Fatalf("Invalid hash format, expected 5 parts, got %d", len(parts))
	}

	if parts[1] != "sm3" {
		t.Errorf("Expected algorithm 'sm3', got '%s'", parts[1])
	}
}

// TestPasswordManager_HashAndVerify 测试密码哈希和验证
func TestPasswordManager_HashAndVerify(t *testing.T) {
	pm := NewPasswordManager(nil)

	password := "TestPassword123!"
	hash, err := pm.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// 验证正确密码
	valid, err := pm.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if !valid {
		t.Error("Password verification should succeed for correct password")
	}

	// 验证错误密码
	valid, err = pm.VerifyPassword("WrongPassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if valid {
		t.Error("Password verification should fail for wrong password")
	}
}

// TestPasswordManager_VerifyInvalidFormat 测试无效哈希格式
func TestPasswordManager_VerifyInvalidFormat(t *testing.T) {
	pm := NewPasswordManager(nil)

	// 测试各种无效格式
	invalidHashes := []string{
		"",
		"invalid",
		"$invalid$format",
		"$sm3$100$salt", // 缺少哈希值
		"$sm3$100",      // 缺少盐和哈希
	}

	for _, hash := range invalidHashes {
		_, err := pm.VerifyPassword("password", hash)
		if err == nil {
			t.Errorf("Expected error for invalid hash '%s'", hash)
		}
	}
}

// TestPasswordManager_DifferentHashesForSamePassword 测试相同密码产生不同哈希
func TestPasswordManager_DifferentHashesForSamePassword(t *testing.T) {
	pm := NewPasswordManager(nil)

	password := "TestPassword123!"
	hash1, err1 := pm.HashPassword(password)
	hash2, err2 := pm.HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("HashPassword failed: %v, %v", err1, err2)
	}

	// 相同密码应该产生不同哈希（因为每次生成不同的盐）
	if hash1 == hash2 {
		t.Error("Same password should produce different hashes")
	}
}

// TestPasswordManager_CustomConfig 测试自定义配置
func TestPasswordManager_CustomConfig(t *testing.T) {
	customConfig := &PasswordConfig{
		Iterations: 500,
		SaltLength: 8,
	}

	pm := NewPasswordManager(customConfig)

	password := "TestPassword123!"
	hash, err := pm.HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// 验证哈希是否包含自定义迭代次数
	parts := strings.Split(hash, "$")
	if parts[2] != "500" {
		t.Errorf("Expected 500 iterations, got '%s'", parts[2])
	}

	// 验证密码仍然有效
	valid, err := pm.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if !valid {
		t.Error("Password verification should succeed")
	}
}

// TestPasswordManager_EmptyPassword 测试空密码
func TestPasswordManager_EmptyPassword(t *testing.T) {
	pm := NewPasswordManager(nil)

	password := ""
	hash, err := pm.HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword failed for empty password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash should not be empty even for empty password")
	}

	// 验证空密码
	valid, err := pm.VerifyPassword("", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if !valid {
		t.Error("Empty password verification should succeed")
	}
}

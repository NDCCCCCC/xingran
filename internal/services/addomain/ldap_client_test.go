package addomain

import (
	"encoding/base64"
	"testing"

	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRDN(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "标准用户DN",
			dn:       "CN=zhangsan,OU=TestDept,DC=company,DC=com",
			expected: "CN=zhangsan",
		},
		{
			name:     "多层OU",
			dn:       "CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com",
			expected: "CN=lisi",
		},
		{
			name:     "单个部分",
			dn:       "CN=zhangsan",
			expected: "CN=zhangsan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRDNFromDN(tt.dn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractOUDNFromUserDN(t *testing.T) {
	tests := []struct {
		name     string
		userDN   string
		expected string
	}{
		{
			name:     "标准用户DN",
			userDN:   "CN=zhangsan,OU=TestDept,DC=company,DC=com",
			expected: "OU=TestDept,DC=company,DC=com",
		},
		{
			name:     "多层OU",
			userDN:   "CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com",
			expected: "OU=SubDept,OU=ParentDept,DC=company,DC=com",
		},
		{
			name:     "空DN",
			userDN:   "",
			expected: "",
		},
		{
			name:     "无OU",
			userDN:   "CN=zhangsan,DC=company,DC=com",
			expected: "DC=company,DC=com",
		},
		{
			name:     "CN在后",
			userDN:   "OU=TestDept,DC=company,DC=com,CN=zhangsan",
			expected: "OU=TestDept,DC=company,DC=com,CN=zhangsan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOUDNFromUserDN(tt.userDN)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRDN_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "空字符串",
			dn:       "",
			expected: "",
		},
		{
			name:     "只有RDN",
			dn:       "CN=zhangsan",
			expected: "CN=zhangsan",
		},
		{
			name:     "特殊字符",
			dn:       "CN=zhang san,OU=Test Dept,DC=company,DC=com",
			expected: "CN=zhang san",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRDNFromDN(tt.dn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractOUDNFromUserDN_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		userDN   string
		expected string
	}{
		{
			name:     "只有CN和DC",
			userDN:   "CN=zhangsan,DC=company,DC=com",
			expected: "DC=company,DC=com",
		},
		{
			name:     "OU在开头",
			userDN:   "OU=TestDept,DC=company,DC=com,CN=zhangsan",
			expected: "OU=TestDept,DC=company,DC=com,CN=zhangsan",
		},
		{
			name:     "多个OU",
			userDN:   "CN=user,OU=Level3,OU=Level2,OU=Level1,DC=company,DC=com",
			expected: "OU=Level3,OU=Level2,OU=Level1,DC=company,DC=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOUDNFromUserDN(tt.userDN)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAccountPoolPasswordRoundTrip 锁定 Phase 36 账号池密码加解密契约（回归守护）。
//
// 背景：ad_account_handler 用 core.SM4Cipher.Encrypt() 加密明文密码存入
// password_ciphertext；ldap_client.tryBindAttempts 必须用 decryptPassword() 解回
// 明文后才能 LDAP bind。修复前 tryBindAttempts 直接拿密文 bind → LDAP error 49，
// 账号池所有账号失败并熔断。
//
// 本测试断言"加密-解密"闭环成立：任何破坏该契约的改动（加解密用不同 cipher、
// 或 tryBindAttempts 漏掉 decryptPassword）都会立即失败。
func TestAccountPoolPasswordRoundTrip(t *testing.T) {
	// 模拟 core.initSM4Cipher：Base64 编码的 16 字节 SM4 key
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	c, err := crypto.NewSM4Cipher(key)
	require.NoError(t, err)

	// 模拟 core.go:134 SetADSM4Cipher：addomain 包内 decryptPassword 读取该全局 cipher
	SetADSM4Cipher(c)
	defer SetADSM4Cipher(nil) // 还原全局状态，避免污染同包其他测试

	// 模拟 ad_account_handler.Create：SM4 加密明文密码
	plaintext := "P@ssw0rd-账号池-123"
	ciphertext, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, ciphertext, "密文应与明文不同")

	// 模拟 ldap_client.tryBindAttempts 修复后的行为：bind 前 decryptPassword
	decrypted := decryptPassword(ciphertext)
	assert.Equal(t, plaintext, decrypted,
		"账号池密文必须能被 decryptPassword 解回明文（LDAP bind 的前提）")
}

// TestDecryptPassword_InvalidCiphertext_ReturnsEmpty 守护 F-03 安全策略：
// 解密失败时返回空字符串而非回退明文，避免攻击者篡改 DB 写入伪密文后被
// ldap_client 当明文 bind（绕过加密保护）。
func TestDecryptPassword_InvalidCiphertext_ReturnsEmpty(t *testing.T) {
	SetADSM4Cipher(nil) // 无 SM4 cipher
	defer SetADSM4Cipher(nil)

	// 非 base64 合法串 → legacy AES 解密失败；无 cipher → SM4 跳过 → 最终返回空
	result := decryptPassword("!!!!not-valid-ciphertext!!!!")
	assert.Equal(t, "", result, "无效密文必须返回空，不得回退明文")
}

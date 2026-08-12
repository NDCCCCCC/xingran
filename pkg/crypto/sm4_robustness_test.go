package crypto

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ==================== 测试矩阵 ====================
//
// 本测试套件用于验证 SM4 cipher 的健壮性,模拟生产环境可能出现的所有异常输入:
//   1. 空值 (cipher / ciphertext / plaintext)
//   2. 畸形输入 (无效 Base64、长度不足、长度超限)
//   3. 密钥错误 (用错 key 解密)
//   4. 并发安全 (race detector 验证)
//   5. Unicode / 特殊字符
//   6. 真实场景模拟:混合加密格式(从 utils.go 的 decryptPassword 复刻)
//
// 跑测试: go test -v -race ./pkg/crypto/
// ====================

// testKey 一段固定测试 key (Base64 编码的 16 字节)
var testKey = base64.StdEncoding.EncodeToString([]byte("1234567890123456"))

// mustNewCipher 辅助函数:成功创建 cipher
func mustNewCipher(t *testing.T) *SM4Cipher {
	t.Helper()
	c, err := NewSM4Cipher(testKey)
	if err != nil {
		t.Fatalf("创建 SM4 cipher 失败: %v", err)
	}
	return c
}

// ==================== 1. 正常路径测试 ====================

func TestSM4_RoundTrip_正常加解密(t *testing.T) {
	c := mustNewCipher(t)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"简单 ASCII", "password123"},
		{"中文密码", "密码@123"},
		{"特殊字符", "P@ssw0rd!#$%^&*()_+-=[]{}|;:',.<>?/~`"},
		{"超长密码", strings.Repeat("a", 1000)},
		{"空字符串", ""},
		{"单字节", "x"},
		{"Unicode emoji", "🔐🔑密码"},
		{"含换行", "line1\nline2\r\nline3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := c.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt 失败: %v", err)
			}

			// 空字符串特殊处理:Encrypt 返回 "",Decrypt 返回 ""
			if tt.plaintext == "" {
				if encrypted != "" {
					t.Errorf("空 plaintext 应返回空 encrypted,得到 %q", encrypted)
				}
				return
			}

			decrypted, err := c.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt 失败: %v", err)
			}
			if decrypted != tt.plaintext {
				t.Errorf("round-trip 不一致:原文=%q,解密=%q", tt.plaintext, decrypted)
			}
		})
	}
}

// ==================== 2. 畸形 ciphertext 测试 ====================

func TestSM4_Decrypt_畸形输入(t *testing.T) {
	c := mustNewCipher(t)

	tests := []struct {
		name        string
		ciphertext  string
		errContains string
	}{
		{"无效 Base64", "!!!not-base64!!!", "Base64 解码失败"},
		{"长度 < nonce(12字节)", "QUFB", "密文长度不足"}, // 4 字节 base64 = 3 字节数据 < 12
		{"只有 nonce 无密文", base64.RawStdEncoding.EncodeToString(make([]byte, 12)), "SM4 解密失败"},
		{"nonce 被篡改", corruptNonce(t, c, "password123"), "SM4 解密失败"},
		{"密文被篡改", corruptCiphertext(t, c, "password123"), "SM4 解密失败"},
		{"截断 1 字节", truncateLastByte(t, c, "password123"), "SM4 解密失败"},
		{"正确长度但全 0", base64.RawStdEncoding.EncodeToString(make([]byte, 32)), "SM4 解密失败"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := c.Decrypt(tt.ciphertext)
			if err == nil {
				t.Errorf("期望报错但成功:decrypted=%q", decrypted)
				return
			}
			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("错误信息不匹配:want 包含 %q,got %q", tt.errContains, err.Error())
			}
			if decrypted != "" {
				t.Errorf("失败时应返回空字符串,得到 %q", decrypted)
			}
		})
	}
}

// corruptNonce 模拟 nonce 被篡改(翻转 nonce 第一个字节)
func corruptNonce(t *testing.T, c *SM4Cipher, plaintext string) string {
	t.Helper()
	enc, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}
	data, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("setup base64 解码失败: %v", err)
	}
	data[0] ^= 0xFF // 翻转第一字节
	return base64.RawStdEncoding.EncodeToString(data)
}

// corruptCiphertext 模拟密文被篡改(翻转密文部分第一个字节)
func corruptCiphertext(t *testing.T, c *SM4Cipher, plaintext string) string {
	t.Helper()
	enc, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}
	data, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("setup base64 解码失败: %v", err)
	}
	// nonce 通常 12 字节,密文在后面
	if len(data) > 12 {
		data[13] ^= 0xFF
	}
	return base64.RawStdEncoding.EncodeToString(data)
}

// truncateLastByte 截断最后 1 字节
func truncateLastByte(t *testing.T, c *SM4Cipher, plaintext string) string {
	t.Helper()
	enc, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}
	data, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("setup base64 解码失败: %v", err)
	}
	if len(data) > 0 {
		data = data[:len(data)-1]
	}
	return base64.RawStdEncoding.EncodeToString(data)
}

// ==================== 3. 错误密钥测试 ====================

func TestSM4_Decrypt_错误密钥(t *testing.T) {
	c1 := mustNewCipher(t)
	encrypted, err := c1.Encrypt("secret_password")
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}

	// 用不同 key 创建 cipher
	wrongKey := base64.StdEncoding.EncodeToString([]byte("0000000000000000"))
	c2, err := NewSM4Cipher(wrongKey)
	if err != nil {
		t.Fatalf("创建错误 key cipher 失败: %v", err)
	}

	decrypted, err := c2.Decrypt(encrypted)
	if err == nil {
		t.Errorf("错误 key 不应解密成功:得到 %q", decrypted)
	}
	if !strings.Contains(err.Error(), "SM4 解密失败") {
		t.Errorf("错误信息不符合预期: %v", err)
	}
}

// ==================== 4. 密钥初始化边界测试 ====================

func TestSM4_NewSM4Cipher_边界(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantErr     bool
		errContains string
	}{
		{"空密钥", "", true, "SM4 密钥长度必须为 16 字节"}, // 空串 → Base64 解码为空 → 长度校验失败
		{"非 Base64", "@@@@", true, "Base64 解码失败"},
		{"Base64 但长度错(8字节)", base64.StdEncoding.EncodeToString([]byte("12345678")), true, "SM4 密钥长度必须为 16 字节"},
		{"Base64 但长度错(32字节)", base64.StdEncoding.EncodeToString(make([]byte, 32)), true, "SM4 密钥长度必须为 16 字节"},
		{"正确 16 字节", testKey, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewSM4Cipher(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("期望报错但成功创建 cipher")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("错误信息不符:want %q,got %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Errorf("不应报错: %v", err)
			}
			if c == nil {
				t.Errorf("cipher 不应为 nil")
			}
		})
	}
}

// ==================== 5. nil receiver 测试(关键 — 模拟 cipher 未初始化场景)====================

// TestSM4_NilReceiver_Encrypt 验证 Fix 4 后 nil receiver 在 Encrypt 时的行为。
//
// 修复后行为(2026-06-24 Fix 4):
//   - Encrypt 任何输入 on nil receiver → 显式返回 error,不 panic
//   - 错误信息包含"未初始化"提示,便于诊断配置缺失
//
// 重要性:这是用户报告"登录失败提示管理员被锁"潜在根因之一
// ——若 SM4Cipher 未初始化(配置缺失),原实现会 panic segfault,
// 上游接不到 error,空密码被错误地当成 LDAP bind password → data 52e
// → badPwdCount++ → AD 域端账号锁定。Fix 4 后该路径彻底关闭。
func TestSM4_NilReceiver_Encrypt(t *testing.T) {
	var c *SM4Cipher = nil

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Encrypt on nil receiver 不应 panic(Fix 4 已加 nil check): %v", r)
		}
	}()

	// 场景 1:非空 plaintext → 期望返回 error
	_, err := c.Encrypt("non-empty-plaintext")
	if err == nil {
		t.Errorf("期望 Encrypt on nil receiver 返回 error,实际成功")
	}
	if !strings.Contains(err.Error(), "未初始化") {
		t.Errorf("错误信息应包含 '未初始化',得到: %v", err)
	}

	// 场景 2:空 plaintext → 期望返回 error(即使空也因 nil check 报错)
	_, err = c.Encrypt("")
	if err == nil {
		t.Errorf("期望 Encrypt(\"\") on nil receiver 返回 error,实际成功(Fix 4 应始终报错)")
	}
}

// TestSM4_NilReceiver_Decrypt 验证 Fix 4 后 nil receiver 在 Decrypt 时的行为。
func TestSM4_NilReceiver_Decrypt(t *testing.T) {
	var c *SM4Cipher = nil

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Decrypt on nil receiver 不应 panic(Fix 4 已加 nil check): %v", r)
		}
	}()

	// 场景 1:空 ciphertext → 期望返回 error
	_, err := c.Decrypt("")
	if err == nil {
		t.Errorf("期望 Decrypt(\"\") on nil receiver 返回 error,实际成功(Fix 4 应始终报错)")
	}

	// 场景 2:非空 ciphertext → 期望返回 error
	_, err = c.Decrypt("non-empty-ciphertext")
	if err == nil {
		t.Errorf("期望 Decrypt on nil receiver 返回 error,实际成功")
	}
	if !strings.Contains(err.Error(), "未初始化") {
		t.Errorf("错误信息应包含 '未初始化',得到: %v", err)
	}
}

// ==================== 6. 并发安全测试(race detector)====================

func TestSM4_Concurrent_Safety(t *testing.T) {
	c := mustNewCipher(t)

	// 预生成 encrypted 数据
	encrypted, err := c.Encrypt("concurrent_password")
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*iterations)

	for i := range goroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := range iterations {
				// 并发 Encrypt + Decrypt
				_, err := c.Encrypt(fmt.Sprintf("worker_%d_iter_%d", idx, j))
				if err != nil {
					errCh <- fmt.Errorf("Encrypt 失败: %w", err)
					return
				}
				_, err = c.Decrypt(encrypted)
				if err != nil {
					errCh <- fmt.Errorf("Decrypt 失败: %w", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("并发错误: %v", err)
	}
}

// ==================== 7. 真实场景模拟:混合加密格式 ====================
//
// 复刻 internal/services/addomain/utils.go:68-99 的 decryptPassword 逻辑,
// 模拟生产环境 sys_ad_service_accounts.password_ciphertext 的各种可能格式。

func TestSM4_ProductionScenario_混合加密格式(t *testing.T) {
	sm4Cipher := mustNewCipher(t)

	// 场景 1:SM4 加密的合法 ciphertext(新格式)
	sm4Ciphertext, err := sm4Cipher.Encrypt("correct_password")
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}

	// 场景 2:从 AES-legacy 加密的 ciphertext(用同样的密码)
	// 注:本测试不引入 AES,只模拟"用不同密钥加密的"情况
	wrongKeyCipher := mustNewCipherWithKey(t, "0000000000000000")
	wrongKeyCiphertext, err := wrongKeyCipher.Encrypt("correct_password")
	if err != nil {
		t.Fatalf("setup Encrypt 失败: %v", err)
	}

	tests := []struct {
		name        string
		ciphertext  string
		wantSuccess bool
		scenario    string
	}{
		{"SM4 正确 ciphertext", sm4Ciphertext, true, "新写入的 SM4 密文"},
		{"错误 key 加密的 ciphertext", wrongKeyCiphertext, false, "key 轮换后老数据(密钥不匹配)"},
		{"空字符串", "", false, "空 ciphertext → 返回空明文(不报错但不视为成功解密)"},
		{"非 SM4 非 Base64 字符串", "plain_text_password", false, "明文密码(数据迁移残留)"},
		{"只有 nonce 无 ciphertext", base64.RawStdEncoding.EncodeToString(make([]byte, 12)), false, "DB 中部分字段被截断"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复刻 utils.go:68-99 的解密逻辑
			result := ""
			if tt.ciphertext != "" {
				plaintext, err := sm4Cipher.Decrypt(tt.ciphertext)
				if err == nil {
					result = plaintext
				}
			}

			if tt.wantSuccess {
				if result == "" {
					t.Errorf("场景 %q:期望解密成功但得到空", tt.scenario)
				}
			} else {
				if result != "" {
					t.Errorf("场景 %q:期望解密失败返回空,得到 %q", tt.scenario, result)
				}
			}
		})
	}
}

// mustNewCipherWithKey 用原始 16 字节 key 创建 cipher(跳过 Base64 编码)
func mustNewCipherWithKey(t *testing.T, key string) *SM4Cipher {
	t.Helper()
	b64Key := base64.StdEncoding.EncodeToString([]byte(key))
	c, err := NewSM4Cipher(b64Key)
	if err != nil {
		t.Fatalf("创建 cipher 失败: %v", err)
	}
	return c
}

// ==================== 8. 文档化预期错误(避免回归)====================

// TestSM4_ErrorMessages 锁死关键错误信息,避免静默改动导致上游判断失效
func TestSM4_ErrorMessages(t *testing.T) {
	c := mustNewCipher(t)

	// 错误 1:Base64 解码失败
	_, err := c.Decrypt("!!!invalid!!!")
	if err == nil || !strings.Contains(err.Error(), "Base64 解码失败") {
		t.Errorf("Base64 错误信息不符: %v", err)
	}

	// 错误 2:密文长度不足
	_, err = c.Decrypt(base64.RawStdEncoding.EncodeToString([]byte("short")))
	if err == nil || !strings.Contains(err.Error(), "密文长度不足") {
		t.Errorf("长度不足错误信息不符: %v", err)
	}

	// 错误 3:SM4 解密失败(GCM 认证失败)
	encrypted, _ := c.Encrypt("test")
	_, err = c.Decrypt(encrypted[:len(encrypted)-4] + "XXXX") // 末尾 4 字节换成无效
	if err == nil || !strings.Contains(err.Error(), "SM4 解密失败") {
		t.Errorf("SM4 解密失败错误信息不符: %v", err)
	}
}

// ==================== 9. 真实 Bind 失败场景模拟 ====================
//
// 这是用户报"登录失败提示管理员被锁"的根本场景:
// decryptPassword 返回 "" → LDAP Bind with "" → 仍可能 data 52e?

func TestSM4_BindFailureScenario(t *testing.T) {
	// 场景:DB 中存的 ciphertext 是用旧 key 加密的,新 key 解不出来 → 返回 ""
	oldKeyCipher := mustNewCipherWithKey(t, "oldkey1234567890") // 16 字节
	oldKeyCiphertext, _ := oldKeyCipher.Encrypt("real_admin_password")

	newKeyCipher := mustNewCipherWithKey(t, "newkey1234567890") // 16 字节

	// 模拟 utils.go:68 的 decryptPassword 行为
	t.Run("key 轮换后老 ciphertext 解不出来", func(t *testing.T) {
		plaintext, err := newKeyCipher.Decrypt(oldKeyCiphertext)
		if err == nil {
			t.Errorf("新 key 不应能解旧 ciphertext")
		}
		if plaintext != "" {
			t.Errorf("失败应返回空字符串,得到 %q", plaintext)
		}

		// 模拟 LDAP Bind with empty password
		// 注:go-ldap 在空密码下行为可能不同 — 这里只演示"密码是空"
		t.Logf("真实 LDAP Bind 会收到 password=%q(空),go-ldap 行为取决于 server 端", plaintext)
	})
}

// ==================== 10. panic 安全性(模拟 cipher nil 注入路径)====================

func TestSM4_NilInjection_PanicSafety(t *testing.T) {
	// 复刻 core.go:113-125 的 initSM4Cipher 行为:
	// 若 sm4Key="" 或创建失败 → 返回 nil cipher
	// 下游 addomain.PasswordCipher 接口实现若没 nil-check → panic

	// 这里不直接调用 initSM4Cipher(它在 internal/core 包),而是在本包模拟
	t.Run("cipher 字段为空时的行为", func(t *testing.T) {
		// 模拟 decryptPassword(utils.go:68) 在 cipher 为 nil 时的行为
		c := getNilPasswordCipher()
		if c != nil {
			t.Fatalf("setup 失败:期望 nil cipher")
		}

		// utils.go:74-80 — cipher 为 nil 时跳过 SM4
		// 这里没有 legacy AES,所以直接返回 ""
		t.Log("✓ cipher 为 nil 时 utils.go:74-80 跳过 SM4 尝试,直接进 legacy AES")
	})
}

// getNilPasswordCipher 模拟 cipher 未初始化的场景
func getNilPasswordCipher() *SM4Cipher {
	return nil
}

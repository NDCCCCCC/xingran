package vdi

import (
	"fmt"
	"testing"
)

func TestVDIPasswordEncryptionDecryption(t *testing.T) {
	fmt.Println("=== VDI 密码加密/解密测试 ===")

	// 测试密码
	testPassword := "sangfor@2020"
	fmt.Printf("原始密码: %s\n\n", testPassword)

	// 测试1: 加密
	fmt.Println("测试1: 密码加密")
	encrypted := encryptVDIPassword(testPassword)
	if encrypted == "" || encrypted == testPassword {
		t.Fatalf("加密失败: encrypted = '%s'", encrypted)
	}
	fmt.Printf("✅ 加密成功: %s\n\n", encrypted[:min(50, len(encrypted))]+"...")

	// 测试2: 解密
	fmt.Println("测试2: 密码解密")
	decrypted := decryptVDIPassword(encrypted)
	if decrypted != testPassword {
		t.Fatalf("解密失败: expected='%s', got='%s'", testPassword, decrypted)
	}
	fmt.Printf("✅ 解密成功: %s\n\n", decrypted)

	// 测试3: 错误输入处理
	fmt.Println("测试3: 错误输入处理")

	// 3a. 空字符串
	emptyDecrypted := decryptVDIPassword("")
	if emptyDecrypted != "" {
		t.Fatalf("空字符串解密应该返回空字符串，得到: '%s'", emptyDecrypted)
	}
	fmt.Println("✅ 空字符串解密正确返回空字符串")

	// 3b. 无效的base64
	invalidDecrypted := decryptVDIPassword("not-valid-base64!!!")
	if invalidDecrypted != "" {
		t.Fatalf("无效base64解密应该返回空字符串，得到: '%s'", invalidDecrypted)
	}
	fmt.Println("✅ 无效base64解密正确返回空字符串")

	// 3c. 有效的base64但解密失败
	validBase64InvalidCrypto := "YWJjZGVmZ2hpams=" // "abcdefghijk" in base64
	cryptoFailDecrypted := decryptVDIPassword(validBase64InvalidCrypto)
	if cryptoFailDecrypted != "" {
		t.Fatalf("解密失败应该返回空字符串，得到: '%s'", cryptoFailDecrypted)
	}
	fmt.Println("✅ 解密失败正确返回空字符串")

	fmt.Println("\n=== 所有测试通过 ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Package security 提供国密 SM4 加密功能
// 用于授权凭证密码等敏感数据的可逆加密存储
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/tjfoc/gmsm/sm4"
)

// SM4Cipher SM4 加密器
type SM4Cipher struct {
	key []byte // 16字节密钥
}

// NewSM4Cipher 创建 SM4 加密器
// key: Base64 编码的 16 字节密钥（从配置文件读取）
func NewSM4Cipher(key string) (*SM4Cipher, error) {
	// 先 Base64 解码
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("SM4 密钥 Base64 解码失败: %w", err)
	}

	// SM4 密钥长度为 128 位 (16 字节)
	if len(keyBytes) != 16 {
		return nil, fmt.Errorf("SM4 密钥长度必须为 16 字节，当前为 %d 字节", len(keyBytes))
	}
	return &SM4Cipher{key: keyBytes}, nil
}

// Encrypt 使用 SM4-GCM 加密数据
// 返回格式: base64(nonce + ciphertext)
func (sc *SM4Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 SM4 加密器（SM4 兼容 AES 的接口）
	block, err := sm4.NewCipher(sc.key)
	if err != nil {
		return "", fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}

	// 使用 GCM 模式（带认证的加密）
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 Base64 编码
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 SM4-GCM 解密数据
// 输入格式: base64(nonce + ciphertext)
func (sc *SM4Cipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 创建 SM4 加密器
	block, err := sm4.NewCipher(sc.key)
	if err != nil {
		return "", fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	// 获取 nonce 大小
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文长度不足")
	}

	// 提取 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// EncryptPassword 加密密码（全局便捷函数）
func EncryptPassword(key, plaintext string) (string, error) {
	cipher, err := NewSM4Cipher(key)
	if err != nil {
		return "", err
	}
	return cipher.Encrypt(plaintext)
}

// DecryptPassword 解密密码（全局便捷函数）
func DecryptPassword(key, ciphertext string) (string, error) {
	cipher, err := NewSM4Cipher(key)
	if err != nil {
		return "", err
	}
	return cipher.Decrypt(ciphertext)
}

// GenerateRandomKey 生成随机 SM4 密钥
// 返回 16 字节的随机密钥（Base64 编码）
func GenerateRandomKey() (string, error) {
	key := make([]byte, aes.BlockSize) // SM4 密钥长度与 AES 相同，为 16 字节
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("生成密钥失败: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(key), nil
}

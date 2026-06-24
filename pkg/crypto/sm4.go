// Package crypto 提供国密 SM4 加密功能
// 用于授权凭证密码等敏感数据的可逆加密存储
package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/tjfoc/gmsm/sm4"
)

// SM4Cipher SM4 加密器
//
// ad-update-attr-no-such-object Fix 4 变更：
//   - 在 NewSM4Cipher 中预创建 block + gcm（一次构造，永久复用），避免
//     每次 Encrypt/Decrypt 都重新调 sm4.NewCipher + cipher.NewGCM 的开销，
//     也避免依赖 tjfoc/gmsm 库未来可能引入的并发限制。
//   - Encrypt/Decrypt 加 sync.Mutex 保护。理论上 GCM 的 Seal/Open 无状态
//     字段写入，但加锁是防御性编程：
//     (1) 杜绝未来 sm4/gmsm 库引入状态的回归风险；
//     (2) 防 io.ReadFull 偶发 bug 导致 gcm.Seal 内部 nonce 生成异常时
//     多个 goroutine 同时操作同一 cipher 时的内存模型竞争。
//   - 接收者 nil 检查：调用方（如 addomain.tryBindAttempts）在 cipher 未
//     初始化场景下不应触发 panic，避免 LDAP bind 走错密码（data 52e →
//     badPwdCount++ → AD 域端账号锁定）。原实现空指针解引用 segfault，
//     上游接不到 error，密码被错误当成空字符串或二进制垃圾 bind。
type SM4Cipher struct {
	key []byte      // 16字节密钥
	gcm cipher.AEAD // 预创建的 GCM 实例（NewGCM 一次构造）
	mu  sync.Mutex  // 保护 Encrypt/Decrypt 调用的互斥锁
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

	// Fix 4: 预创建 block + gcm，避免每次调用都重建 cipher 实例
	block, err := sm4.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	return &SM4Cipher{
		key: keyBytes,
		gcm: gcm,
	}, nil
}

// Encrypt 使用 SM4-GCM 加密数据
// 返回格式: base64(nonce + ciphertext)
func (sc *SM4Cipher) Encrypt(plaintext string) (string, error) {
	// Fix 4: 接收者 nil 检查放在最前（即使 plaintext="" 也要报错）
	// 原因:调用方（如 addomain.tryBindAttempts）在 cipher 未初始化
	// 场景下不应触发 panic，避免 LDAP bind 走错密码（data 52e →
	// badPwdCount++ → AD 域端账号锁定）。原实现空指针解引用 segfault，
	// 上游接不到 error，密码被错误当成空字符串或二进制垃圾 bind。
	if sc == nil {
		return "", fmt.Errorf("SM4Cipher 未初始化（密钥未配置或初始化失败）")
	}
	if plaintext == "" {
		return "", nil
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 生成随机 nonce
	nonce := make([]byte, sc.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// 加密数据（gcm.Seal 把 nonce 拼接到 ciphertext 前）
	ciphertext := sc.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 Base64 编码
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 SM4-GCM 解密数据
// 输入格式: base64(nonce + ciphertext)
func (sc *SM4Cipher) Decrypt(ciphertext string) (string, error) {
	// Fix 4: 接收者 nil 检查放在最前
	if sc == nil {
		return "", fmt.Errorf("SM4Cipher 未初始化（密钥未配置或初始化失败）")
	}
	if ciphertext == "" {
		return "", nil
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Base64 解码
	data, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 获取 nonce 大小
	nonceSize := sc.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文长度不足")
	}

	// 提取 nonce 和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := sc.gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("SM4 解密失败: %w", err)
	}

	return string(plaintext), nil
}

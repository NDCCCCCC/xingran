package addomain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ==================== SM4 Cipher Interface ====================

// PasswordCipher defines the interface for password encryption/decryption.
// This avoids an import cycle with internal/core/security.
type PasswordCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

var (
	globalPasswordCipher PasswordCipher
	passwordCipherMu     sync.RWMutex
)

// SetADSM4Cipher sets the global SM4 cipher for AD password encryption (thread-safe)
func SetADSM4Cipher(c PasswordCipher) {
	passwordCipherMu.Lock()
	defer passwordCipherMu.Unlock()
	globalPasswordCipher = c
}

// getPasswordCipher returns the global SM4 cipher (thread-safe)
func getPasswordCipher() PasswordCipher {
	passwordCipherMu.RLock()
	defer passwordCipherMu.RUnlock()
	return globalPasswordCipher
}

// ==================== Password Encryption/Decryption ====================

// encryptPassword encrypts a password using SM4-GCM.
// Falls back to the legacy AES-GCM key only when SM4 cipher is not initialized.
func encryptPassword(password string) string {
	c := getPasswordCipher()
	if c != nil {
		encrypted, err := c.Encrypt(password)
		if err == nil {
			return encrypted
		}
	}
	return encryptPasswordLegacyAES(password)
}

// DecryptPassword decrypts a password - exported version for backward compatibility
func DecryptPassword(encrypted string) string {
	return decryptPassword(encrypted)
}

// decryptPassword decrypts a password, trying SM4 first, then legacy AES fallback.
// If both fail, returns the original string (for plaintext passwords).
func decryptPassword(encrypted string) string {
	if encrypted == "" {
		return ""
	}

	// Try SM4-GCM decryption first (new format)
	c := getPasswordCipher()
	if c != nil {
		plaintext, err := c.Decrypt(encrypted)
		if err == nil {
			return plaintext
		}
	}

	// Try legacy AES-GCM decryption (old format)
	plaintext, err := decryptPasswordLegacyAES(encrypted)
	if err == nil {
		return plaintext
	}

	// F-03: 两种解密都失败时,不再回退返回原文 —
	// 攻击者可通过 SQL 注入或直接修改 DB 写入伪加密字符串,
	// 原逻辑会"解密失败 → 返回原文"使该字符串被当成 LDAP bind 密码,
	// 等于完全绕过加密保护。改为返回空字符串,让后续 LDAP bind
	// 因密码空而明确失败,且记录 SECURITY 级日志便于排查入侵痕迹。
	applogger.Errorf(
		"[SECURITY] decryptPassword: SM4 与 legacy AES 均解密失败,拒绝回退明文 "+
			"(input_len=%d) — 请检查 DB 是否被篡改或加密密钥是否轮换不一致",
		len(encrypted),
	)
	return ""
}

// ==================== Legacy AES-GCM (for backward compatibility) ====================

// F-02: legacy AES key 不再仅有硬编码值,优先从环境变量 AD_LEGACY_AES_KEY 读取。
//
// 设计:
//   - 加密路径 (encryptPasswordLegacyAES): env 优先,无 env 时回退硬编码并记一次性告警
//   - 解密路径 (decryptPasswordLegacyAES): 先试 env key,再试硬编码 key,
//     保证历史已加密数据(用硬编码 key 加密)在 env 设置后仍能读出
//
// env 值必须恰好 16 字节(AES-128 key)。
// 长期方案: 全量迁移到 SM4 (默认路径) 后可彻底移除 legacy AES。
const legacyAESKeyHardcoded = "xingran-ad-domain-key-16"

var legacyAESWarnOnce sync.Once

func getLegacyAESEncryptKey() []byte {
	if env := os.Getenv("AD_LEGACY_AES_KEY"); env != "" {
		if len(env) >= 16 {
			return []byte(env[:16])
		}
		applogger.Warnf("[SECURITY] AD_LEGACY_AES_KEY 长度 %d < 16,回退硬编码 key", len(env))
	} else {
		legacyAESWarnOnce.Do(func() {
			applogger.Warnf(
				"[SECURITY] AD legacy AES 使用硬编码 key — " +
					"反汇编二进制即可解密所有 AD 密码,建议设置 AD_LEGACY_AES_KEY 环境变量",
			)
		})
	}
	return []byte(legacyAESKeyHardcoded[:16])
}

// getLegacyAESDecryptKeys 返回所有应该尝试的解密 key 列表,先 env 后硬编码,
// 保证 env 配置后仍能解密旧数据。
func getLegacyAESDecryptKeys() [][]byte {
	keys := make([][]byte, 0, 2)
	if env := os.Getenv("AD_LEGACY_AES_KEY"); env != "" && len(env) >= 16 {
		keys = append(keys, []byte(env[:16]))
	}
	keys = append(keys, []byte(legacyAESKeyHardcoded[:16]))
	return keys
}

// encryptPasswordLegacyAES encrypts using the old AES-GCM key.
// Kept for writing passwords when SM4 cipher is not yet available.
func encryptPasswordLegacyAES(password string) string {
	key := getLegacyAESEncryptKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return password
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return password
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return password
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// decryptPasswordLegacyAES decrypts using the old AES-GCM key.
// F-02 fix: 依次尝试 env key 与硬编码 key,保证 key 轮换期间历史数据可读。
func decryptPasswordLegacyAES(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	var lastErr error
	for _, key := range getLegacyAESDecryptKeys() {
		plaintext, err := tryDecryptAESGCM(data, key)
		if err == nil {
			return plaintext, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("legacy AES decrypt failed (all keys tried): %w", lastErr)
}

// tryDecryptAESGCM 尝试用指定 key 解密 GCM 密文
func tryDecryptAESGCM(data, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm failed: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("data too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// ==================== Utility Functions ====================

// extractParentDN extracts the parent DN
func extractParentDN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], ",")
}

// buildOUPath builds the OU path
func buildOUPath(ouDN, baseDN string) string {
	if ouDN == baseDN {
		return "/"
	}
	parts := strings.Split(ouDN, ",")
	var pathParts []string
	for _, part := range parts {
		if strings.Contains(part, "OU=") {
			name := strings.TrimPrefix(part, "OU=")
			pathParts = append([]string{name}, pathParts...)
		}
	}
	return "/" + strings.Join(pathParts, "/")
}

// parseIntOrDefault parses an integer, returns default on failure
func parseIntOrDefault(str string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(str, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// parseFileTime parses AD FileTime format
func parseFileTime(fileTime string) *time.Time {
	var ft int64
	_, err := fmt.Sscanf(fileTime, "%d", &ft)
	if err != nil || ft == 0 {
		return nil
	}

	unix := (ft / 10000000) - 11644473600
	if unix < 0 {
		return nil
	}

	t := time.Unix(unix, 0)
	return &t
}

// ExtractOUDNFromUserDN extracts the OU DN from a user DN
func ExtractOUDNFromUserDN(userDN string) string {
	if userDN == "" {
		return ""
	}

	parts := strings.Split(userDN, ",")
	if len(parts) <= 1 {
		return ""
	}

	for i, part := range parts {
		if strings.HasPrefix(strings.ToUpper(part), "OU=") {
			return strings.Join(parts[i:], ",")
		}
	}

	if len(parts) > 1 {
		return strings.Join(parts[1:], ",")
	}

	return ""
}

// ParseOUDN parses an OU DN string into its component parts.
// Uses strings.Split to properly handle UTF-8 encoded multi-byte characters (like Chinese)
// Example: "OU=基础运维科,OU=科技创新部,OU=分公司本部,DC=example,DC=com"
// Returns: ["OU=基础运维科", "OU=科技创新部", "OU=分公司本部", "DC=example", "DC=com"]
func ParseOUDN(ouDN string) []string {
	if ouDN == "" {
		return nil
	}

	// Use strings.Split to correctly handle UTF-8 encoding
	parts := strings.Split(ouDN, ",")

	// Trim whitespace from each part
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

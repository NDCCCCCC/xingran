package addomain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
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

	// Both failed - assume plaintext (backward compatibility for pre-encryption data)
	return encrypted
}

// ==================== Legacy AES-GCM (for backward compatibility) ====================

// encryptPasswordLegacyAES encrypts using the old AES-GCM key.
// Kept for writing passwords when SM4 cipher is not yet available.
func encryptPasswordLegacyAES(password string) string {
	const encryptionKey = "ruoyi-ad-domain-key-16"
	key := []byte(encryptionKey[:16])

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
func decryptPasswordLegacyAES(encrypted string) (string, error) {
	const encryptionKey = "ruoyi-ad-domain-key-16"
	key := []byte(encryptionKey[:16])

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

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
// Example: "OU=基础运维科,OU=科技创新部,OU=分公司本部,DC=example,DC=com"
// Returns: ["OU=基础运维科", "OU=科技创新部", "OU=分公司本部", "DC=example", "DC=com"]
func ParseOUDN(ouDN string) []string {
	if ouDN == "" {
		return nil
	}

	var parts []string
	current := ""
	inEscape := false

	for i := 0; i < len(ouDN); i++ {
		ch := ouDN[i]

		// Handle escape sequences
		if inEscape {
			current += string(ch)
			inEscape = false
			continue
		}

		// Check for escape character
		if ch == '\\' {
			inEscape = true
			continue
		}

		// Check for separator
		if ch == ',' {
			if current != "" {
				parts = append(parts, strings.TrimSpace(current))
			}
			current = ""
			continue
		}

		current += string(ch)
	}

	// Add the last part
	if current != "" {
		parts = append(parts, strings.TrimSpace(current))
	}

	return parts
}

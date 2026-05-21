package addomain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"
)

// encryptPassword 加密密码（使用AES-GCM）
func encryptPassword(password string) string {
	// AD域加密密钥，实际应从配置文件读取
	const encryptionKey = "ruoyi-ad-domain-key-16"

	// 确保密钥长度为16字节（AES-128）
	key := []byte(encryptionKey[:16])

	block, err := aes.NewCipher(key)
	if err != nil {
		// 加密失败时返回原始密码（向后兼容）
		return password
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return password
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return password
	}

	// 加密并编码为Base64
	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// DecryptPassword 解密密码（使用AES-GCM）- 导出版本
func DecryptPassword(encrypted string) string {
	return decryptPassword(encrypted)
}

// decryptPassword 解密密码（使用AES-GCM）
func decryptPassword(encrypted string) string {
	// AD域加密密钥，实际应从配置文件读取
	const encryptionKey = "ruoyi-ad-domain-key-16"

	// 确保密钥长度为16字节（AES-128）
	key := []byte(encryptionKey[:16])

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		// 解码失败，可能未加密，返回原始值
		return encrypted
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encrypted
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encrypted
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		// 数据长度不足，可能未加密
		return encrypted
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		// 解密失败，返回原始值
		return encrypted
	}

	return string(plaintext)
}

// extractParentDN 提取父DN
func extractParentDN(dn string) string {
	parts := strings.Split(dn, ",")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], ",")
}

// buildOUPath 构建OU路径
func buildOUPath(ouDN, baseDN string) string {
	if ouDN == baseDN {
		return "/"
	}
	// 简化处理：去掉baseDN后反转DN部分
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

// parseIntOrDefault 解析整数，失败返回默认值
func parseIntOrDefault(str string, defaultValue int) int {
	var result int
	_, err := fmt.Sscanf(str, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// parseFileTime 解析AD FileTime格式
func parseFileTime(fileTime string) *time.Time {
	// AD FileTime是100纳秒为单位，从1601-01-01开始的计数
	// 这里简化处理，实际需要完整的转换
	var ft int64
	_, err := fmt.Sscanf(fileTime, "%d", &ft)
	if err != nil || ft == 0 {
		return nil
	}

	// 转换为Unix时间戳（简化版）
	// 实际需要：unix = (ft / 10000000) - 11644473600
	unix := (ft / 10000000) - 11644473600
	if unix < 0 {
		return nil
	}

	t := time.Unix(unix, 0)
	return &t
}

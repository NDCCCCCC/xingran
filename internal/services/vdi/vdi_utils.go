package vdi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

// decryptVDIPassword 解密VDI服务器密码（从models包复制，因为该函数未导出）
func decryptVDIPassword(encrypted string) string {
	const encryptionKey = "xingran-vdi-server-key-16"
	key := []byte(encryptionKey[:16])

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "" // 返回空字符串表示解密失败
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "" // 返回空字符串表示解密失败
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "" // 返回空字符串表示解密失败
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "" // 返回空字符串表示解密失败
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "" // 返回空字符串表示解密失败
	}

	return string(plaintext)
}

// encryptVDIPassword 加密VDI服务器密码（从models包复制，因为该函数未导出）
func encryptVDIPassword(password string) string {
	const encryptionKey = "xingran-vdi-server-key-16"
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

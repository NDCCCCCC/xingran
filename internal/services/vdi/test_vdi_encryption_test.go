package vdi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestVDIPasswordEncryptionDecryption tests password encrypt/decrypt roundtrip
func TestVDIPasswordEncryptionDecryption(t *testing.T) {
	testPassword := "sangfor@2020"

	t.Run("encrypt produces different output", func(t *testing.T) {
		encrypted := encryptVDIPassword(testPassword)
		assert.NotEmpty(t, encrypted, "encrypted password should not be empty")
		assert.NotEqual(t, testPassword, encrypted, "encrypted should differ from original")
	})

	t.Run("decrypt roundtrip", func(t *testing.T) {
		encrypted := encryptVDIPassword(testPassword)
		decrypted := decryptVDIPassword(encrypted)
		assert.Equal(t, testPassword, decrypted, "decrypted password should match original")
	})

	t.Run("empty string decrypt returns empty", func(t *testing.T) {
		result := decryptVDIPassword("")
		assert.Empty(t, result, "empty string decrypt should return empty")
	})

	t.Run("invalid base64 decrypt returns empty", func(t *testing.T) {
		result := decryptVDIPassword("not-valid-base64!!!")
		assert.Empty(t, result, "invalid base64 decrypt should return empty")
	})

	t.Run("valid base64 but invalid ciphertext returns empty", func(t *testing.T) {
		result := decryptVDIPassword("YWJjZGVmZ2hpams=") // "abcdefghijk" in base64
		assert.Empty(t, result, "invalid ciphertext decrypt should return empty")
	})

	t.Run("different passwords produce different ciphertexts", func(t *testing.T) {
		enc1 := encryptVDIPassword("password1")
		enc2 := encryptVDIPassword("password2")
		assert.NotEqual(t, enc1, enc2, "different passwords should produce different ciphertexts")
	})

	t.Run("same password produces different ciphertexts due to random nonce", func(t *testing.T) {
		enc1 := encryptVDIPassword(testPassword)
		enc2 := encryptVDIPassword(testPassword)
		assert.NotEqual(t, enc1, enc2, "same password should produce different ciphertexts (random nonce)")
	})
}

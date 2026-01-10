// Package crypto 提供请求体国密加密功能
// 使用 SM2+SM4 混合加密：SM4-CBC 加密请求体，SM2 加密 SM4 密钥
package crypto

import (
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm4"
)

// EncryptedRequest 加密请求结构
type EncryptedRequest struct {
	Encrypted bool   `json:"encrypted"` // 加密标识
	Data      string `json:"data"`      // SM4-CBC 加密的数据 (Base64)
	SM4Key    string `json:"sm4Key"`    // SM2 加密后的 SM4 密钥 (Base64)
	IV        string `json:"iv"`        // SM4-CBC 的初始化向量 (Base64)
	Timestamp int64  `json:"timestamp"` // 请求时间戳（防重放）
}

// RequestEncryptor 请求加密器
type RequestEncryptor struct {
	sm2PrivateKey *sm2.PrivateKey // SM2 私钥（用于解密 SM4 密钥）
	sm2PublicKey  *sm2.PublicKey  // SM2 公钥（用于加密响应，可选）
}

// NewRequestEncryptor 创建请求加密器
// 需要传入 SM2 密钥对，用于加密/解密 SM4 密钥
func NewRequestEncryptor(privateKey *sm2.PrivateKey, publicKey *sm2.PublicKey) *RequestEncryptor {
	return &RequestEncryptor{
		sm2PrivateKey: privateKey,
		sm2PublicKey:  publicKey,
	}
}

// DecryptRequest 解密请求体
// 步骤：
// 1. 验证时间戳
// 2. 使用 SM2 私钥解密 SM4 密钥（十六进制字符串）
// 3. 将十六进制转换为字节数组
// 4. Base64 解码 IV 和密文
// 5. SM4-CBC 解密
// 6. 去除 PKCS#7 填充
func (re *RequestEncryptor) DecryptRequest(encReq *EncryptedRequest) ([]byte, error) {
	// 1. 验证时间戳（防重放攻击，5分钟内有效）
	if encReq.Timestamp > 0 {
		timeDiff := time.Now().Unix() - encReq.Timestamp
		if timeDiff < -300 || timeDiff > 300 {
			return nil, fmt.Errorf("请求时间戳无效: 时间差 %d 秒", timeDiff)
		}
	}

	// 2. 使用 SM2 私钥解密 SM4 密钥（前端发送十六进制字符串，32字符）
	sm4KeyHex, err := DecryptWithSM2(encReq.SM4Key, re.sm2PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("SM2 解密 SM4 密钥失败: %w", err)
	}

	// 3. 将十六进制字符串转换为字节数组（使用标准库）
	keyBytes, err := hex.DecodeString(sm4KeyHex)
	if err != nil {
		return nil, fmt.Errorf("SM4 密钥十六进制解码失败: %w", err)
	}
	if len(keyBytes) != sm4.BlockSize {
		return nil, fmt.Errorf("SM4 密钥长度无效: 期望 %d, 实际 %d", sm4.BlockSize, len(keyBytes))
	}

	// 4. Base64 解码 IV 和密文
	iv, err := base64.StdEncoding.DecodeString(encReq.IV)
	if err != nil {
		return nil, fmt.Errorf("IV Base64 解码失败: %w", err)
	}
	if len(iv) != sm4.BlockSize {
		return nil, fmt.Errorf("IV 长度无效: 期望 %d, 实际 %d", sm4.BlockSize, len(iv))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encReq.Data)
	if err != nil {
		return nil, fmt.Errorf("密文 Base64 解码失败: %w", err)
	}

	// 5. SM4-CBC 解密
	block, err := sm4.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}

	if len(ciphertext)%sm4.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度不是块大小的倍数")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// 6. 去除 PKCS#7 填充
	plaintext, err := pkcs7Unpad(decrypted)
	if err != nil {
		return nil, fmt.Errorf("去除填充失败: %w", err)
	}

	return plaintext, nil
}

// DecryptRequestWithKeyInfo 解密请求体并返回密钥信息
// 用于响应加密时重用相同的 SM4 密钥和 IV
func (re *RequestEncryptor) DecryptRequestWithKeyInfo(encReq *EncryptedRequest) (plaintext []byte, sm4Key []byte, iv []byte, err error) {
	// 1. 验证时间戳（防重放攻击，5分钟内有效）
	if encReq.Timestamp > 0 {
		timeDiff := time.Now().Unix() - encReq.Timestamp
		if timeDiff < -300 || timeDiff > 300 {
			return nil, nil, nil, fmt.Errorf("请求时间戳无效: 时间差 %d 秒", timeDiff)
		}
	}

	// 2. 使用 SM2 私钥解密 SM4 密钥
	sm4KeyHex, err := DecryptWithSM2(encReq.SM4Key, re.sm2PrivateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("SM2 解密 SM4 密钥失败: %w", err)
	}

	// 3. 将十六进制字符串转换为字节数组
	sm4Key, err = hex.DecodeString(sm4KeyHex)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("SM4 密钥十六进制解码失败: %w", err)
	}
	if len(sm4Key) != sm4.BlockSize {
		return nil, nil, nil, fmt.Errorf("SM4 密钥长度无效: 期望 %d, 实际 %d", sm4.BlockSize, len(sm4Key))
	}

	// 4. Base64 解码 IV 和密文
	iv, err = base64.StdEncoding.DecodeString(encReq.IV)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("IV Base64 解码失败: %w", err)
	}
	if len(iv) != sm4.BlockSize {
		return nil, nil, nil, fmt.Errorf("IV 长度无效: 期望 %d, 实际 %d", sm4.BlockSize, len(iv))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encReq.Data)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("密文 Base64 解码失败: %w", err)
	}

	// 5. SM4-CBC 解密
	block, err := sm4.NewCipher(sm4Key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}

	if len(ciphertext)%sm4.BlockSize != 0 {
		return nil, nil, nil, fmt.Errorf("密文长度不是块大小的倍数")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// 6. 去除 PKCS#7 填充
	plaintext, err = pkcs7Unpad(decrypted)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("去除填充失败: %w", err)
	}

	return plaintext, sm4Key, iv, nil
}

// EncryptResponseWithKey 使用已有的 SM4 密钥和 IV 加密响应体
// 用于重用请求中的 SM4 密钥，这样前端可以用同一个密钥解密响应
func (re *RequestEncryptor) EncryptResponseWithKey(data []byte, sm4Key []byte, iv []byte) (*EncryptedRequest, error) {
	// 1. PKCS#7 填充
	paddedData := pkcs7Pad(data, sm4.BlockSize)

	// 2. SM4-CBC 加密
	block, err := sm4.NewCipher(sm4Key)
	if err != nil {
		return nil, fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(paddedData))
	mode.CryptBlocks(ciphertext, paddedData)

	return &EncryptedRequest{
		Encrypted: true,
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
		SM4Key:    "", // 不需要，前端已有密钥
		IV:        base64.StdEncoding.EncodeToString(iv),
		Timestamp: 0,
	}, nil
}

// pkcs7Pad PKCS#7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs7Unpad 去除 PKCS#7 填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("数据为空")
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return nil, fmt.Errorf("填充长度无效: %d", padding)
	}
	if len(data) < padding {
		return nil, fmt.Errorf("数据长度不足: 数据 %d 字节, 填充 %d 字节", len(data), padding)
	}
	// 验证填充字节
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("填充字节不匹配: 期望 %d, 实际 %d", padding, data[i])
		}
	}
	return data[:len(data)-padding], nil
}

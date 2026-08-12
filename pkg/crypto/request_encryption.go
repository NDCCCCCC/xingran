// Package crypto 提供请求体国密加密功能
// 使用 SM2+SM4 混合加密：SM4-CBC 加密请求体，SM2 加密 SM4 密钥
package crypto

import (
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
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
	Timestamp int64  `json:"timestamp"` // 请求时间戳（防重放，必填）
	Nonce     string `json:"nonce"`     // 随机数（防重放，必填）
}

const (
	// DefaultReplayWindowSec 默认的最大时间差(秒)
	//
	// P1 fix: 从 300s (5 min) 收紧到 120s (2 min),再收紧到 60s (1 min)。
	// 60s 平衡了 NTP 时钟漂移容忍 (±60s 是 NTP-managed 客户端的合理基线)
	// 和网络/处理延迟 (留 60s buffer)。
	// 实际窗口可通过 RequestEncryptor.SetReplayWindowSec() 或 NewRequestEncryptorWithConfig()
	// 覆盖,默认对齐 OWASP V3 会话管理建议。
	DefaultReplayWindowSec = 60
	// minTimestamp 最小有效时间戳（2020-01-01）
	minTimestamp = 1577836800
)

// NonceStorage nonce存储接口（防止重放攻击）
type NonceStorage interface {
	// CheckAndStore 检查nonce是否存在，如果不存在则存储
	// 返回 true 表示nonce是新的（可以接受），false 表示nonce已存在（拒绝请求）
	CheckAndStore(nonce string, timestamp int64) bool
}

// defaultNonceStorage 默认的内存nonce存储实现
type defaultNonceStorage struct {
	nonces map[string]int64 // nonce -> timestamp
	mu     sync.RWMutex     // 保护 nonces map 的并发访问
}

// NewDefaultNonceStorage 创建默认的nonce存储（使用分段锁实现，支持高并发）
func NewDefaultNonceStorage() NonceStorage {
	return NewShardedNonceStorage()
}

// CheckAndStore 检查并存储nonce
func (d *defaultNonceStorage) CheckAndStore(nonce string, timestamp int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.nonces[nonce]; exists {
		return false // nonce已存在，拒绝
	}
	d.nonces[nonce] = timestamp
	return true
}

// RequestEncryptor 请求加密器
type RequestEncryptor struct {
	sm2PrivateKey  *sm2.PrivateKey // SM2 私钥（用于解密 SM4 密钥）
	sm2PublicKey   *sm2.PublicKey  // SM2 公钥（用于加密响应，可选）
	nonceStorage   NonceStorage    // nonce存储（防止重放攻击）
	replayWindowSec int            // 时间戳容差(秒),±N
}

// NewRequestEncryptor 创建请求加密器
// 需要传入 SM2 密钥对，用于加密/解密 SM4 密钥
//  默认使用 DefaultReplayWindowSec (60s) 作为时间戳容差。
// 如需自定义,使用 NewRequestEncryptorWithConfig 或 SetReplayWindowSec。
func NewRequestEncryptor(privateKey *sm2.PrivateKey, publicKey *sm2.PublicKey) *RequestEncryptor {
	re := &RequestEncryptor{
		sm2PrivateKey:  privateKey,
		sm2PublicKey:   publicKey,
		nonceStorage:   NewDefaultNonceStorage(),
		replayWindowSec: DefaultReplayWindowSec,
	}
	return re
}

// RequestEncryptorConfig RequestEncryptor 构造配置
//
// P1 fix (P1-S2): 将 hardcoded 60s/120s 重放窗口从常量改为可配置。
// 不同环境 (dev=120s 方便调试; prod=60s 对齐 OWASP) 可独立调整。
type RequestEncryptorConfig struct {
	ReplayWindowSec int // 时间戳容差(秒),±N。<=0 时使用 DefaultReplayWindowSec (60)
}

// NewRequestEncryptorWithConfig 使用配置创建请求加密器
func NewRequestEncryptorWithConfig(privateKey *sm2.PrivateKey, publicKey *sm2.PublicKey, cfg RequestEncryptorConfig) *RequestEncryptor {
	re := NewRequestEncryptor(privateKey, publicKey)
	if cfg.ReplayWindowSec > 0 {
		re.replayWindowSec = cfg.ReplayWindowSec
	}
	return re
}

// SetReplayWindowSec 设置时间戳容差(秒),±N。值必须 > 0。
func (re *RequestEncryptor) SetReplayWindowSec(sec int) {
	if sec > 0 {
		re.replayWindowSec = sec
	}
}

// ReplayWindowSec 获取当前时间戳容差(秒)
func (re *RequestEncryptor) ReplayWindowSec() int {
	if re.replayWindowSec <= 0 {
		return DefaultReplayWindowSec
	}
	return re.replayWindowSec
}

// SetNonceStorage 设置nonce存储（可选，用于自定义存储实现）
func (re *RequestEncryptor) SetNonceStorage(storage NonceStorage) {
	re.nonceStorage = storage
}

// validateTimestamp 验证时间戳
//
// P1 fix (P1-S2): 使用 RequestEncryptor.replayWindowSec 作为容差
// (默认 60s,可由 config 覆盖)。偏移超出 ±window 即拒绝。
func (re *RequestEncryptor) validateTimestamp(timestamp int64) error {
	window := re.ReplayWindowSec()
	// 强制要求时间戳
	if timestamp <= 0 {
		return errors.New("时间戳不能为空")
	}
	// 检查时间戳是否合理（不能是太早的日期）
	if timestamp < minTimestamp {
		return fmt.Errorf("时间戳无效: 过早的日期")
	}
	// 检查时间差(双向)
	timeDiff := time.Now().Unix() - timestamp
	if timeDiff < -int64(window) || timeDiff > int64(window) {
		return fmt.Errorf("请求时间戳无效: 时间差 %d 秒,容差 ±%d 秒", timeDiff, window)
	}
	return nil
}

// validateNonce 验证nonce
func (re *RequestEncryptor) validateNonce(nonce string, timestamp int64) error {
	if nonce == "" {
		return errors.New("nonce不能为空")
	}
	if !re.nonceStorage.CheckAndStore(nonce, timestamp) {
		return errors.New("请求重复，拒绝处理")
	}
	return nil
}

// DecryptRequest 解密请求体
// 步骤：
// 1. 验证时间戳（强制要求）
// 2. 验证nonce（防重放）
// 3. 使用 SM2 私钥解密 SM4 密钥（十六进制字符串）
// 4. 将十六进制转换为字节数组
// 5. Base64 解码 IV 和密文
// 6. SM4-CBC 解密
// 7. 去除 PKCS#7 填充
func (re *RequestEncryptor) DecryptRequest(encReq *EncryptedRequest) ([]byte, error) {
	// 1. 验证时间戳（防重放攻击，强制要求）
	if err := re.validateTimestamp(encReq.Timestamp); err != nil {
		return nil, err
	}

	// 2. 验证nonce（防重放）
	if err := re.validateNonce(encReq.Nonce, encReq.Timestamp); err != nil {
		return nil, err
	}

	// 3. 使用 SM2 私钥解密 SM4 密钥（前端发送十六进制字符串，32字符）
	sm4KeyHex, err := DecryptWithSM2(encReq.SM4Key, re.sm2PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解密失败")
	}

	// 4. 将十六进制字符串转换为字节数组（使用标准库）
	keyBytes, err := hex.DecodeString(sm4KeyHex)
	if err != nil {
		return nil, fmt.Errorf("密钥解码失败")
	}
	if len(keyBytes) != sm4.BlockSize {
		return nil, fmt.Errorf("密钥长度无效")
	}

	// 5. Base64 解码 IV 和密文
	iv, err := base64.StdEncoding.DecodeString(encReq.IV)
	if err != nil {
		return nil, fmt.Errorf("IV解码失败")
	}
	if len(iv) != sm4.BlockSize {
		return nil, fmt.Errorf("IV长度无效")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encReq.Data)
	if err != nil {
		return nil, fmt.Errorf("数据解码失败")
	}

	// 6. SM4-CBC 解密
	block, err := sm4.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("加密器创建失败")
	}

	if len(ciphertext)%sm4.BlockSize != 0 {
		return nil, fmt.Errorf("密文格式错误")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// 7. 去除 PKCS#7 填充
	plaintext, err := pkcs7Unpad(decrypted)
	if err != nil {
		return nil, fmt.Errorf("解密失败")
	}

	return plaintext, nil
}

// DecryptRequestWithKeyInfo 解密请求体并返回密钥信息
// 用于响应加密时重用相同的 SM4 密钥和 IV
func (re *RequestEncryptor) DecryptRequestWithKeyInfo(encReq *EncryptedRequest) (plaintext []byte, sm4Key []byte, iv []byte, err error) {
	// 1. 验证时间戳（防重放攻击，强制要求）
	if err := re.validateTimestamp(encReq.Timestamp); err != nil {
		return nil, nil, nil, err
	}

	// 2. 验证nonce（防重放）
	if err := re.validateNonce(encReq.Nonce, encReq.Timestamp); err != nil {
		return nil, nil, nil, err
	}

	// 3. 使用 SM2 私钥解密 SM4 密钥
	sm4KeyHex, err := DecryptWithSM2(encReq.SM4Key, re.sm2PrivateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解密失败")
	}

	// 4. 将十六进制字符串转换为字节数组
	sm4Key, err = hex.DecodeString(sm4KeyHex)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("密钥解码失败")
	}
	if len(sm4Key) != sm4.BlockSize {
		return nil, nil, nil, fmt.Errorf("密钥长度无效")
	}

	// 5. Base64 解码 IV 和密文
	iv, err = base64.StdEncoding.DecodeString(encReq.IV)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("IV解码失败")
	}
	if len(iv) != sm4.BlockSize {
		return nil, nil, nil, fmt.Errorf("IV长度无效")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encReq.Data)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("数据解码失败")
	}

	// 6. SM4-CBC 解密
	block, err := sm4.NewCipher(sm4Key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("加密器创建失败")
	}

	if len(ciphertext)%sm4.BlockSize != 0 {
		return nil, nil, nil, fmt.Errorf("密文格式错误")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// 7. 去除 PKCS#7 填充
	plaintext, err = pkcs7Unpad(decrypted)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解密失败")
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

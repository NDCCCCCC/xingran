// Package crypto 提供 XingRan CLI 的请求国密加密客户端实现。
//
// 协议与后端 pkg/crypto/request_encryption.go（服务端 DecryptRequest）严格对齐：
//   - SM4-CBC（PKCS#7 填充）加密请求 body，随机 16 字节 key + 16 字节 IV
//   - SM2（C1C3C2, mode=1）加密 SM4 密钥的十六进制字符串，Base64 输出
//     （与 pkg/crypto.EncryptWithSM2 语义一致：32 字符 hex 会被解码为 16 字节再加密，
//     服务端解密后经 decodeHexEncodedResult 还原为 hex 字符串）
//   - 每次请求携带 timestamp（Unix 秒，后端容差 ±60s）+ 随机 nonce 防重放
//   - 公钥通过 GET /api/v1/system/auth/public-key 获取，data.publicKey 为
//     原始十六进制公钥（04||X||Y，65 字节，由 ExportPublicKeyToHex 导出）
package crypto

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm4"

	pkgcrypto "github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

// PublicKeyPath 后端 SM2 公钥端点（相对 baseURL）
const PublicKeyPath = "/api/v1/system/auth/public-key"

// Encryptor 请求加密器接口（CONTRACT.md 定义，internal/client 通过它接入加密）
type Encryptor interface {
	// FetchPublicKey 从后端拉取 SM2 公钥并缓存
	FetchPublicKey(ctx context.Context, baseURL string) error
	// EncryptRequest 加密请求 body：SM4 加密 body + SM2 加密 SM4 key + ts/nonce，
	// 返回可直接作为请求体发送的 JSON payload（pkg/crypto.EncryptedRequest 结构）
	EncryptRequest(ctx context.Context, body []byte) (payload []byte, err error)
}

// encryptor Encryptor 默认实现
type encryptor struct {
	httpClient   *http.Client
	publicKeyHex string
	publicKey    *sm2.PublicKey
	now          func() time.Time // 可注入的时钟（测试用）
	randReader   io.Reader        // 可注入的随机源（测试用）
}

// NewEncryptor 创建默认加密器
func NewEncryptor() Encryptor {
	return &encryptor{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		now:        time.Now,
		randReader: rand.Reader,
	}
}

// newEncryptorForTest 创建可注入时钟与随机源的加密器（仅测试使用）
func newEncryptorForTest(nowFn func() time.Time, rr io.Reader) *encryptor {
	e := NewEncryptor().(*encryptor)
	e.now = nowFn
	e.randReader = rr
	return e
}

// PublicKeyHex 返回当前缓存的公钥十六进制字符串（未加载时为空）
func PublicKeyHex(e Encryptor) string {
	enc, ok := e.(*encryptor)
	if !ok {
		return ""
	}
	return enc.publicKeyHex
}

// publicKeyResponse GET /system/auth/public-key 的响应结构（后端 response 包装）
type publicKeyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PublicKey string `json:"publicKey"`
	} `json:"data"`
}

// FetchPublicKey 拉取 SM2 公钥
// 后端 handler：internal/api/v1/auth.go getPublicKey，响应 data.publicKey 为 hex 公钥
func (e *encryptor) FetchPublicKey(ctx context.Context, baseURL string) error {
	baseURL = strings.TrimRight(baseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+PublicKeyPath, nil)
	if err != nil {
		return fmt.Errorf("构造公钥请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求公钥失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("读取公钥响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取公钥失败: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var pubResp publicKeyResponse
	if err := json.Unmarshal(respBody, &pubResp); err != nil {
		return fmt.Errorf("解析公钥响应失败: %w", err)
	}
	if pubResp.Code != 0 {
		return fmt.Errorf("获取公钥失败: code=%d, message=%s", pubResp.Code, pubResp.Message)
	}
	if pubResp.Data.PublicKey == "" {
		return errors.New("获取公钥失败: 后端未返回公钥（SM2 可能未启用）")
	}

	pub, err := parsePublicKeyHex(pubResp.Data.PublicKey)
	if err != nil {
		return fmt.Errorf("公钥格式无效: %w", err)
	}
	e.publicKeyHex = pubResp.Data.PublicKey
	e.publicKey = pub
	return nil
}

// parsePublicKeyHex 校验并解析十六进制公钥（04||X||Y，65 字节 = 130 hex 字符）
// 复用 pkg/crypto.ParsePublicKeyFromHex
func parsePublicKeyHex(hexKey string) (*sm2.PublicKey, error) {
	hexKey = strings.TrimSpace(strings.ToLower(hexKey))
	if len(hexKey) != 130 {
		return nil, fmt.Errorf("公钥长度应为 130 个十六进制字符(65 字节), 实际 %d", len(hexKey))
	}
	if !strings.HasPrefix(hexKey, "04") {
		return nil, errors.New("公钥应为未压缩点格式（04 开头）")
	}
	return pkgcrypto.ParsePublicKeyFromHex(hexKey)
}

// EncryptRequest 加密请求 body 并返回 JSON payload
func (e *encryptor) EncryptRequest(ctx context.Context, body []byte) (payload []byte, err error) {
	if e.publicKey == nil {
		return nil, errors.New("SM2 公钥未加载, 请先调用 FetchPublicKey")
	}
	_ = ctx

	// 1. 生成随机 SM4 密钥（16 字节）与 IV（16 字节），密钥以 32 字符 hex 表示，
	//    与前端 generateSM4Key() 产物格式一致
	sm4Key := make([]byte, sm4.BlockSize)
	if _, err := io.ReadFull(e.randReader, sm4Key); err != nil {
		return nil, fmt.Errorf("生成 SM4 密钥失败: %w", err)
	}
	iv := make([]byte, sm4.BlockSize)
	if _, err := io.ReadFull(e.randReader, iv); err != nil {
		return nil, fmt.Errorf("生成 IV 失败: %w", err)
	}

	// 2. PKCS#7 填充 + SM4-CBC 加密 body
	ciphertext, err := sm4CBCEncrypt(body, sm4Key, iv)
	if err != nil {
		return nil, fmt.Errorf("SM4 加密失败: %w", err)
	}

	// 3. SM2 加密 SM4 密钥 hex 字符串（复用 pkg/crypto.EncryptWithSM2：
	//    32 字符 hex 输入会被解码为 16 字节后加密, 服务端解密后还原为 hex 字符串）
	encryptedSM4Key, err := pkgcrypto.EncryptWithSM2(hex.EncodeToString(sm4Key), e.publicKey)
	if err != nil {
		return nil, fmt.Errorf("SM2 加密 SM4 密钥失败: %w", err)
	}

	// 4. 组装 EncryptedRequest（与 pkg/crypto.EncryptedRequest 字段一一对应）
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(e.randReader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	encReq := pkgcrypto.EncryptedRequest{
		Encrypted: true,
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
		SM4Key:    encryptedSM4Key,
		IV:        base64.StdEncoding.EncodeToString(iv),
		Timestamp: e.now().Unix(),
		Nonce:     hex.EncodeToString(nonce),
	}
	return json.Marshal(encReq)
}

// sm4CBCEncrypt PKCS#7 填充后 SM4-CBC 加密（与 pkg/crypto 服务端解密实现对称）
func sm4CBCEncrypt(plaintext, key, iv []byte) ([]byte, error) {
	if len(key) != sm4.BlockSize {
		return nil, fmt.Errorf("SM4 密钥长度应为 %d 字节, 实际 %d", sm4.BlockSize, len(key))
	}
	if len(iv) != sm4.BlockSize {
		return nil, fmt.Errorf("IV 长度应为 %d 字节, 实际 %d", sm4.BlockSize, len(iv))
	}
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 SM4 加密器失败: %w", err)
	}
	padded := pkcs7Pad(plaintext, sm4.BlockSize)
	mode := cipher.NewCBCEncrypter(block, iv)
	out := make([]byte, len(padded))
	mode.CryptBlocks(out, padded)
	return out, nil
}

// pkcs7Pad PKCS#7 填充（与 pkg/crypto/request_encryption.go 实现一致）
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

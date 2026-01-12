package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/x509"
)

// SM2SigningMethod 实现JWT的SM2签名方法
type SM2SigningMethod struct {
	privateKey *sm2.PrivateKey
	publicKey  *sm2.PublicKey
	name       string
}

// NewSM2SigningMethod 创建SM2签名方法实例
func NewSM2SigningMethod(privateKey *sm2.PrivateKey, publicKey *sm2.PublicKey) *SM2SigningMethod {
	return &SM2SigningMethod{
		privateKey: privateKey,
		publicKey:  publicKey,
		name:       "SM2",
	}
}

// Alg 返回算法名称
func (m *SM2SigningMethod) Alg() string {
	return m.name
}

// Verify 验证JWT签名
func (m *SM2SigningMethod) Verify(signingString string, signature []byte, key interface{}) error {
	var publicKey *sm2.PublicKey

	switch k := key.(type) {
	case *sm2.PublicKey:
		publicKey = k
	case []byte:
		// 尝试从字节数组解析公钥
		pubKey, err := x509.ParseSm2PublicKey(k)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		publicKey = pubKey
	case string:
		// 尝试从十六进制字符串解析公钥
		keyBytes, err := hex.DecodeString(k)
		if err != nil {
			return fmt.Errorf("failed to decode public key hex: %w", err)
		}
		pubKey, err := x509.ParseSm2PublicKey(keyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		publicKey = pubKey
	default:
		if m.publicKey != nil {
			publicKey = m.publicKey
		} else {
			return fmt.Errorf("invalid public key type")
		}
	}

	// 验证SM2签名
	if len(signature) == 0 {
		return jwt.ErrSignatureInvalid
	}

	// SM2验签
	isValid := publicKey.Verify([]byte(signingString), signature)
	if !isValid {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

// Sign 对JWT进行SM2签名
func (m *SM2SigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	var privateKey *sm2.PrivateKey

	switch k := key.(type) {
	case *sm2.PrivateKey:
		privateKey = k
	case []byte:
		// 尝试从字节数组解析私钥
		privKey, err := x509.ParseSm2PrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		privateKey = privKey
	case string:
		// 尝试从十六进制字符串解析私钥
		keyBytes, err := hex.DecodeString(k)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key hex: %w", err)
		}
		privKey, err := x509.ParseSm2PrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		privateKey = privKey
	default:
		if m.privateKey != nil {
			privateKey = m.privateKey
		} else {
			return nil, fmt.Errorf("invalid private key type")
		}
	}

	// 使用rand.Reader提供随机数
	signature, err := privateKey.Sign(rand.Reader, []byte(signingString), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with SM2: %w", err)
	}

	return signature, nil
}

// GenerateKeyPair 生成SM2密钥对
func GenerateKeyPair() (*sm2.PrivateKey, *sm2.PublicKey, error) {
	privateKey, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate SM2 key pair: %w", err)
	}

	publicKey := privateKey.Public().(*sm2.PublicKey)
	return privateKey, publicKey, nil
}

// ParsePrivateKeyFromHex 从十六进制字符串解析私钥
// 支持两种格式：
// 1. DER/ASN.1 格式（x509.ParseSm2PrivateKey）
// 2. 原始 D 值格式（32字节，64个十六进制字符）
func ParsePrivateKeyFromHex(hexKey string) (*sm2.PrivateKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	// 先尝试 DER 格式
	privateKey, err := x509.ParseSm2PrivateKey(keyBytes)
	if err == nil {
		return privateKey, nil
	}

	// 如果 DER 解析失败，尝试原始 D 值格式（32字节）
	if len(keyBytes) == 32 {
		// 使用 sm2 生成密钥的方式来创建私钥
		d := new(big.Int).SetBytes(keyBytes)

		// 计算公钥点: Q = d * G
		// SM2 使用的是 sm2p256v1 曲线（类似 P-256）
		curve := sm2.P256Sm2()
		x, y := curve.ScalarBaseMult(d.Bytes())

		privateKey = &sm2.PrivateKey{
			PublicKey: sm2.PublicKey{
				Curve: curve,
				X:     x,
				Y:     y,
			},
			D: d,
		}
		return privateKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key from hex (tried DER and raw D format): %w", err)
}

// ParsePublicKeyFromHex 从十六进制字符串解析公钥
// 支持两种格式：
// 1. DER/ASN.1 格式（x509.ParseSm2PublicKey）
// 2. 原始格式 04 + X + Y（65字节，130个十六进制字符）
func ParsePublicKeyFromHex(hexKey string) (*sm2.PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	// 先尝试 DER 格式
	publicKey, err := x509.ParseSm2PublicKey(keyBytes)
	if err == nil {
		return publicKey, nil
	}

	// 如果 DER 解析失败，尝试原始格式 04 + X + Y（65字节）
	if len(keyBytes) == 65 && keyBytes[0] == 0x04 {
		curve := sm2.P256Sm2()
		x := new(big.Int).SetBytes(keyBytes[1:33])
		y := new(big.Int).SetBytes(keyBytes[33:65])
		publicKey = &sm2.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}
		return publicKey, nil
	}

	return nil, fmt.Errorf("failed to parse public key from hex (tried DER and raw 04+X+Y format): %w", err)
}

// Claims JWT声明结构体
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience []string `json:"aud"`
	jwt.RegisteredClaims
}

// GenerateTokenWithSM2 使用SM2算法生成JWT Token
func GenerateTokenWithSM2(claims *Claims, privateKey *sm2.PrivateKey) (string, error) {
	// 设置Token签发时间
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	// 创建Token对象
	token := jwt.NewWithClaims(&SM2SigningMethod{
		privateKey: privateKey,
		name:       "SM2",
	}, claims)

	// 生成签名字符串
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateTokenWithSM2 使用SM2算法验证JWT Token
func ValidateTokenWithSM2(tokenString string, publicKey *sm2.PublicKey) (*Claims, error) {
	// 创建SM2签名方法实例
	sm2Method := NewSM2SigningMethod(nil, publicKey)

	// 分割token
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token contains an invalid number of segments")
	}

	// 验证header格式
	_, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token header: %w", err)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token signature: %w", err)
	}

	// 构建签名字符串
	signingString := strings.Join(parts[:2], ".")

	// 使用SM2验证签名
	err = sm2Method.Verify(signingString, signature, publicKey)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// 解析claims
	var claims Claims
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// 验证过期时间
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	// 验证生效时间
	if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
		return nil, jwt.ErrTokenNotValidYet
	}

	return &claims, nil
}

// GenerateRefreshTokenWithSM2 生成刷新Token（使用SM2签名）
func GenerateRefreshTokenWithSM2(userID string, username string, privateKey *sm2.PrivateKey) (string, error) {
	// 刷新Token的有效期更长
	expirationTime := time.Now().Add(7 * 24 * time.Hour) // 7天

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    []string{"refresh"}, // 刷新Token的角色标识
		Issuer:   "RuoYi-Next",
		Subject:  userID,
		Audience: []string{"ruoyi-next"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		},
	}

	return GenerateTokenWithSM2(claims, privateKey)
}

// ValidateRefreshTokenWithSM2 验证刷新Token
func ValidateRefreshTokenWithSM2(tokenString string, publicKey *sm2.PublicKey) (*Claims, error) {
	return ValidateTokenWithSM2(tokenString, publicKey)
}

// EncryptWithSM2 使用 SM2 公钥加密数据（用于密码传输）
// plaintext: 可以是明文字符串或十六进制字符串
// 如果 plaintext 是偶数长度的十六进制字符串（如 SM4 密钥），会先解码为二进制再加密
// 返回 Base64 编码的密文
func EncryptWithSM2(plaintext string, publicKey *sm2.PublicKey) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 检查是否为十六进制格式的二进制数据（如 SM4 密钥）
	var dataToEncrypt []byte
	if len(plaintext)%2 == 0 && isHexString(plaintext) {
		// 将十六进制字符串解码为二进制数据
		decoded, err := hex.DecodeString(plaintext)
		if err == nil && len(decoded) <= 128 {
			// 十六进制解码成功，使用二进制数据
			dataToEncrypt = decoded
		} else {
			// 解码失败，使用原始字符串
			dataToEncrypt = []byte(plaintext)
		}
	} else {
		// 不是标准十六进制格式，使用原始字符串
		dataToEncrypt = []byte(plaintext)
	}

	// SM2 加密（使用 C1C3C2 模式，mode = 1）
	ciphertext, err := sm2.Encrypt(publicKey, dataToEncrypt, rand.Reader, 1)
	if err != nil {
		return "", fmt.Errorf("SM2 加密失败: %w", err)
	}

	// 转换为 Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// isHexString 检查字符串是否为有效的十六进制格式
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// decodeHexEncodedResult 处理 SM2 解密后的结果
// 检测解密结果是否为十六进制编码的字符串（如 SM4 密钥）
// 如果是 32 字节且可解码为十六进制，则解码为原始字节数组（16 字节）
// 返回十六进制格式的字符串供后续使用
func decodeHexEncodedResult(plaintext []byte, mode string) (string, error) {
	hexStr := fmt.Sprintf("%x", plaintext)

	// 检测模式：解密结果是否为十六进制字符串的 ASCII 表示
	if len(plaintext) == 32 || len(plaintext) == 64 {
		asciiStr := string(plaintext)

		// 检查是否为有效的十六进制字符串
		if isHexString(asciiStr) {
			// 尝试解码十六进制字符串为原始字节数组
			decoded, err := hex.DecodeString(asciiStr)
			if err == nil && len(decoded) == 16 {
				// 对于 SM4 密钥，期望得到 16 字节
				decodedHex := fmt.Sprintf("%x", decoded)
				return decodedHex, nil
			}
		}
	}

	// 默认返回原始十六进制编码
	return hexStr, nil
}

// DecryptWithSM2 使用 SM2 私钥解密数据
// ciphertext: Base64 编码的密文
func DecryptWithSM2(ciphertext string, privateKey *sm2.PrivateKey) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 兼容 sm-crypto 格式：尝试多种解密方式
	var plaintext []byte

	// 尝试方式 1: 直接使用原始数据 mode 1
	plaintext, err = sm2.Decrypt(privateKey, data, 1)
	if err == nil {
		return decodeHexEncodedResult(plaintext, "mode1 原始格式")
	}

	// 尝试方式 2: 添加 04 前缀后 mode 1
	if len(data) > 0 && data[0] != 0x04 {
		decryptData := make([]byte, len(data)+1)
		decryptData[0] = 0x04
		copy(decryptData[1:], data)
		plaintext, err = sm2.Decrypt(privateKey, decryptData, 1)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode1 添加04前缀")
		}
	}

	// 尝试方式 3: 直接使用原始数据 mode 0
	plaintext, err = sm2.Decrypt(privateKey, data, 0)
	if err == nil {
		return decodeHexEncodedResult(plaintext, "mode0 原始格式")
	}

	// 尝试方式 4: 添加 04 前缀后 mode 0
	if len(data) > 0 && data[0] != 0x04 {
		decryptData := make([]byte, len(data)+1)
		decryptData[0] = 0x04
		copy(decryptData[1:], data)
		plaintext, err = sm2.Decrypt(privateKey, decryptData, 0)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode0 添加04前缀")
		}
	}

	return "", fmt.Errorf("SM2 解密失败: 尝试了所有模式均失败")
}

// ExportPublicKeyToPEM 导出公钥为 PEM 格式
func ExportPublicKeyToPEM(publicKey *sm2.PublicKey) (string, error) {
	// 将公钥转换为 DER 编码（ASN.1 格式）
	derBytes, err := x509.MarshalSm2PublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("公钥 DER 编码失败: %w", err)
	}

	// 构建 PEM 格式
	pemKey := "-----BEGIN PUBLIC KEY-----\n"
	pemKey += base64.StdEncoding.EncodeToString(derBytes)
	pemKey += "\n-----END OF PUBLIC KEY-----\n"

	return pemKey, nil
}

// ExportPublicKeyToHex 导出公钥为原始十六进制格式
// 返回格式: 04 + X坐标(32字节) + Y坐标(32字节) = 65字节（130个十六进制字符）
// 用于前端 sm-crypto 库直接使用
func ExportPublicKeyToHex(publicKey *sm2.PublicKey) string {
	// SM2 公钥的原始字节: 04 + X + Y (非压缩格式)
	// publicKey 是 sm2.PublicKey 类型，包含 X 和 Y 坐标

	// 将公钥转换为原始字节格式
	// 格式: 0x04 (非压缩点) + X(32字节) + Y(32字节)
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// 确保X和Y都是32字节
	paddingX := make([]byte, 32-len(xBytes))
	paddingY := make([]byte, 32-len(yBytes))

	// 拼接: 04 + X + Y
	rawBytes := make([]byte, 0, 65)
	rawBytes = append(rawBytes, 0x04) // 非压缩点标识
	rawBytes = append(rawBytes, paddingX...)
	rawBytes = append(rawBytes, xBytes...)
	rawBytes = append(rawBytes, paddingY...)
	rawBytes = append(rawBytes, yBytes...)

	return hex.EncodeToString(rawBytes)
}

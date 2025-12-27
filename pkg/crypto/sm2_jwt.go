package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
func ParsePrivateKeyFromHex(hexKey string) (*sm2.PrivateKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	privateKey, err := x509.ParseSm2PrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from hex: %w", err)
	}
	return privateKey, nil
}

// ParsePublicKeyFromHex 从十六进制字符串解析公钥
func ParsePublicKeyFromHex(hexKey string) (*sm2.PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	publicKey, err := x509.ParseSm2PublicKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key from hex: %w", err)
	}
	return publicKey, nil
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
	signingString := parts[0] + "." + parts[1]

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
// 返回 Base64 编码的密文
func EncryptWithSM2(plaintext string, publicKey *sm2.PublicKey) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// SM2 加密（使用 C1C3C2 模式，mode = 1）
	ciphertext, err := sm2.Encrypt(publicKey, []byte(plaintext), rand.Reader, 1)
	if err != nil {
		return "", fmt.Errorf("SM2 加密失败: %w", err)
	}

	// 转换为 Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
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

	// SM2 解密（使用 C1C3C2 模式，mode = 1）
	plaintext, err := sm2.Decrypt(privateKey, data, 1)
	if err != nil {
		return "", fmt.Errorf("SM2 解密失败: %w", err)
	}

	return string(plaintext), nil
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

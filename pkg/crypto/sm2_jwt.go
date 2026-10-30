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
		pubKey, err := x509.ParseSm2PublicKey(k)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		publicKey = pubKey
	case string:
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

	if len(signature) == 0 {
		return jwt.ErrSignatureInvalid
	}

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
		privKey, err := x509.ParseSm2PrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		privateKey = privKey
	case string:
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
// 支持 DER/ASN.1 格式或原始 D 值格式（32字节）
func ParsePrivateKeyFromHex(hexKey string) (*sm2.PrivateKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	privateKey, err := x509.ParseSm2PrivateKey(keyBytes)
	if err == nil {
		return privateKey, nil
	}

	if len(keyBytes) == 32 {
		d := new(big.Int).SetBytes(keyBytes)
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
// 支持 DER/ASN.1 格式或原始格式 04 + X + Y（65字节）
func ParsePublicKeyFromHex(hexKey string) (*sm2.PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}

	publicKey, err := x509.ParseSm2PublicKey(keyBytes)
	if err == nil {
		return publicKey, nil
	}

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
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(&SM2SigningMethod{
		privateKey: privateKey,
		name:       "SM2",
	}, claims)

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateTokenWithSM2 使用SM2算法验证JWT Token
func ValidateTokenWithSM2(tokenString string, publicKey *sm2.PublicKey) (*Claims, error) {
	sm2Method := NewSM2SigningMethod(nil, publicKey)

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("token contains an invalid number of segments")
	}

	// P1 fix (algorithm confusion): 必须先解析 header 并严格校验 alg 字段。
	// 之前直接 `_, err := DecodeString(parts[0])` 丢弃 header,攻击者可把
	// alg 改为 "none" 或 "HS256",服务端仍用 SM2 校验,通过算法误用绕过签名。
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to parse token header: %w", err)
	}
	if header.Alg != sm2Method.Alg() {
		return nil, fmt.Errorf("unexpected JWT signing algorithm %q (expected %q)", header.Alg, sm2Method.Alg())
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token signature: %w", err)
	}

	signingString := strings.Join(parts[:2], ".")

	err = sm2Method.Verify(signingString, signature, publicKey)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	var claims Claims
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
		return nil, jwt.ErrTokenNotValidYet
	}

	return &claims, nil
}

// GenerateRefreshTokenWithSM2 生成刷新Token（使用SM2签名）
//
// issuer 与 expiration 必须由调用方（JWTManager）从配置传入，禁止硬编码：
// 曾硬编码 Issuer "Xingran-Next"，与配置 jwt.issuer（如 "XingRan-Next-Dev"）不一致，
// 导致 ValidateToken 签发者校验恒定失败 → 页面刷新时 /system/auth/refresh 100% 401。
func GenerateRefreshTokenWithSM2(userID string, username string, issuer string, expiration time.Duration, privateKey *sm2.PrivateKey) (string, error) {
	expirationTime := time.Now().Add(expiration)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    []string{"refresh"},
		Issuer:   issuer,
		Subject:  userID,
		Audience: []string{"xingran-next"},
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
// plaintext: 明文字符串或十六进制字符串（如 SM4 密钥）
// 返回 Base64 编码的密文
func EncryptWithSM2(plaintext string, publicKey *sm2.PublicKey) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	var dataToEncrypt []byte
	if len(plaintext)%2 == 0 && isHexString(plaintext) {
		decoded, err := hex.DecodeString(plaintext)
		if err == nil && len(decoded) <= 128 {
			dataToEncrypt = decoded
		} else {
			dataToEncrypt = []byte(plaintext)
		}
	} else {
		dataToEncrypt = []byte(plaintext)
	}

	ciphertext, err := sm2.Encrypt(publicKey, dataToEncrypt, rand.Reader, 1)
	if err != nil {
		return "", fmt.Errorf("SM2 encryption failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func decodeHexEncodedResult(plaintext []byte, mode string) (string, error) {
	if isPrintableASCII(plaintext) {
		return string(plaintext), nil
	}

	hexStr := fmt.Sprintf("%x", plaintext)

	if len(plaintext) == 32 || len(plaintext) == 64 {
		asciiStr := string(plaintext)
		if isHexString(asciiStr) {
			decoded, err := hex.DecodeString(asciiStr)
			if err == nil && len(decoded) == 16 {
				decodedHex := fmt.Sprintf("%x", decoded)
				return decodedHex, nil
			}
		}
	}

	return hexStr, nil
}

func isPrintableASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

// DecryptWithSM2 使用 SM2 私钥解密数据
func DecryptWithSM2(ciphertext string, privateKey *sm2.PrivateKey) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	// SM2 mode-1(C1C3C2 未压缩)密文最小 97 字节: 0x04(1) + X||Y(64) + SM3(32) + 密文(≥0)
	if len(data) < 96 {
		return "", fmt.Errorf("sm2 密文长度非法: %d", len(data))
	}

	var plaintext []byte

	if len(data) >= 97 {
		plaintext, err = sm2.Decrypt(privateKey, data, 1)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode1")
		}
	}

	if len(data) > 0 && data[0] != 0x04 {
		decryptData := make([]byte, len(data)+1)
		decryptData[0] = 0x04
		copy(decryptData[1:], data)
		plaintext, err = sm2.Decrypt(privateKey, decryptData, 1)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode1 with 04 prefix")
		}
	}

	if len(data) >= 97 {
		plaintext, err = sm2.Decrypt(privateKey, data, 0)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode0")
		}
	}

	if len(data) > 0 && data[0] != 0x04 {
		decryptData := make([]byte, len(data)+1)
		decryptData[0] = 0x04
		copy(decryptData[1:], data)
		plaintext, err = sm2.Decrypt(privateKey, decryptData, 0)
		if err == nil {
			return decodeHexEncodedResult(plaintext, "mode0 with 04 prefix")
		}
	}

	return "", fmt.Errorf("SM2 decryption failed: all modes failed")
}

func ExportPublicKeyToPEM(publicKey *sm2.PublicKey) (string, error) {
	derBytes, err := x509.MarshalSm2PublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pemKey := "-----BEGIN PUBLIC KEY-----\n"
	pemKey += base64.StdEncoding.EncodeToString(derBytes)
	pemKey += "\n-----END OF PUBLIC KEY-----\n"

	return pemKey, nil
}

// ExportPublicKeyToHex 导出公钥为原始十六进制格式 (04 + X + Y = 65字节)
func ExportPublicKeyToHex(publicKey *sm2.PublicKey) string {
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	paddingX := make([]byte, 32-len(xBytes))
	paddingY := make([]byte, 32-len(yBytes))

	rawBytes := make([]byte, 0, 65)
	rawBytes = append(rawBytes, 0x04)
	rawBytes = append(rawBytes, paddingX...)
	rawBytes = append(rawBytes, xBytes...)
	rawBytes = append(rawBytes, paddingY...)
	rawBytes = append(rawBytes, yBytes...)

	return hex.EncodeToString(rawBytes)
}

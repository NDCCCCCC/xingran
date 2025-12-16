package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// PasswordConfig 密码配置
type PasswordConfig struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// DefaultPasswordConfig 默认密码配置
var DefaultPasswordConfig = &PasswordConfig{
	Time:    3,       // 迭代次数
	Memory:  64 * 1024, // 内存使用量 (64MB)
	Threads: 4,       // 线程数
	KeyLen:  32,      // 密钥长度
}

// PasswordManager 密码管理器
type PasswordManager struct {
	config *PasswordConfig
}

// NewPasswordManager 创建密码管理器
func NewPasswordManager(config *PasswordConfig) *PasswordManager {
	if config == nil {
		config = DefaultPasswordConfig
	}
	return &PasswordManager{config: config}
}

// HashPassword 哈希密码
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	// 生成随机盐
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成盐失败: %w", err)
	}

	// 使用Argon2哈希密码
	hash := argon2.IDKey([]byte(password), salt, pm.config.Time, pm.config.Memory, pm.config.Threads, pm.config.KeyLen)

	// 编码为Base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 组合格式: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	return fmt.Sprintf(format, argon2.Version, pm.config.Memory, pm.config.Time, pm.config.Threads, b64Salt, b64Hash), nil
}

// VerifyPassword 验证密码
func (pm *PasswordManager) VerifyPassword(password, hash string) (bool, error) {
	// 解析哈希格式
	var version int
	var memory, time uint32
	var threads uint8
	var salt, hashBytes []byte

	_, err := fmt.Sscanf(hash, "$argon2id=v=%d&m=%d,t=%d,p=%d$%x$%x",
		&version, &memory, &time, &threads, &salt, &hashBytes)
	if err != nil {
		// 尝试解析Base64格式
		var saltStr, hashStr string
		_, err = fmt.Sscanf(hash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
			&version, &memory, &time, &threads, &saltStr, &hashStr)
		if err != nil {
			return false, fmt.Errorf("解析密码哈希失败: %w", err)
		}

		salt, err = base64.RawStdEncoding.DecodeString(saltStr)
		if err != nil {
			return false, fmt.Errorf("解码盐失败: %w", err)
		}

		hashBytes, err = base64.RawStdEncoding.DecodeString(hashStr)
		if err != nil {
			return false, fmt.Errorf("解码哈希失败: %w", err)
		}
	}

	// 使用相同的参数哈希输入密码
	comparisonHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(hashBytes)))

	// 安全比较哈希
	return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
}

// GenerateRandomPassword 生成随机密码
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	for i := range password {
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			return "", fmt.Errorf("生成随机密码失败: %w", err)
		}
		password[i] = charset[randomByte[0]%byte(len(charset))]
	}

	return string(password), nil
}

// GenerateRandomToken 生成随机令牌
func GenerateRandomToken(length int) (string, error) {
	if length < 32 {
		length = 32
	}

	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("生成随机令牌失败: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}
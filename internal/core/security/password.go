// Package security 提供安全相关功能
// 包括密码哈希、JWT管理、国密算法支持等
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
)

// PasswordConfig 密码配置
// 定义密码哈希算法的参数配置
type PasswordConfig struct {
	Iterations int // PBKDF2迭代次数，影响哈希强度和计算时间
	SaltLength int // 盐值长度，用于防止彩虹表攻击
}

// DefaultPasswordConfig 默认密码配置
// 提供安全且性能平衡的默认配置
//
// P1 fix: Iterations 提升到 600000 (600k):
//   - OWASP 2023 baseline for PBKDF2-SM3; backward-compat with 100k hashes via embedded iteration count
//   - OWASP 2023 推荐 PBKDF2-HMAC-SHA256 ≥ 600,000;SM3 计算成本与 SHA-256 接近
//   - 600k 显著提高离线爆破成本(从分钟级提升到小时级)
//   - 哈希格式 $sm3$iterations$salt$hash 已嵌入 iterations,旧密码可继续验证
//     (用 hash 里嵌入的旧值,仅新 hash 用 600k,无迁移成本)
var DefaultPasswordConfig = &PasswordConfig{
	Iterations: 600000, // SM3-PBKDF2 抗离线爆破基线
	SaltLength: 16,     // 128位盐值，提供足够的随机性
}

// PasswordManager 密码管理器
// 提供密码哈希、验证等安全功能，基于国密SM3算法
type PasswordManager struct {
	config *PasswordConfig // 密码配置参数
}

// NewPasswordManager 创建密码管理器
// 如果未提供配置，则使用默认配置
func NewPasswordManager(config *PasswordConfig) *PasswordManager {
	if config == nil {
		config = DefaultPasswordConfig
	}
	return &PasswordManager{config: config}
}

// pbkdf2SM3 基于SM3的PBKDF2实现（单块契约）
// 使用国密SM3算法作为伪随机函数(PRF)，实现PBKDF2密钥派生
// 参数：
//   - password: 原始密码
//   - salt: 盐值
//   - iterations: 迭代次数
//   - keyLen: 期望的密钥长度
//
// 算法流程：
//  1. U1 = SM3(password || salt || 00000001)
//  2. U2 = SM3(U1)
//  3. ...
//  4. result = U1 XOR U2 XOR ... XOR Uc
//
// 契约（Phase 146）：
//   - 仅支持 1<=keyLen<=32 单块；计数器固定为 block 1 (0x00000001)，未实现多块循环
//     （CRYPTO-04 多块能力扩展当前无需求，缺席不算遗留缺陷）
//   - keyLen<=0 / keyLen>32 / iterations<=0 一律返回 error（禁 panic、禁静默弱密钥）
//   - keyLen=32 时 l=1,i=1，与历史实现逐字节一致（存量 $sm3$ 哈希零迁移）
func (pm *PasswordManager) pbkdf2SM3(password, salt []byte, iterations, keyLen int) ([]byte, error) {
	// 入口守卫：把未定义行为变定义行为，非法入参绝不静默产出「后半段全零」的低熵弱密钥
	if keyLen <= 0 {
		return nil, fmt.Errorf("pbkdf2SM3: 非法 keyLen=%d，须为正数且不超过 SM3 摘要长度 32", keyLen)
	}
	if keyLen > 32 {
		return nil, fmt.Errorf("pbkdf2SM3: keyLen=%d 超过 32，当前实现仅支持单块（INT32_BE(1)），拒绝静默弱密钥", keyLen)
	}
	if iterations <= 0 {
		return nil, fmt.Errorf("pbkdf2SM3: 非法 iterations=%d，须为正数", iterations)
	}

	// 初始化数据块和结果
	block := make([]byte, 32) // SM3输出长度为32字节
	result := make([]byte, keyLen)

	// 构造初始数据块：password || salt || 计数器(00000001)
	// make+append 避免 append(password, salt...) 在 cap 富余时原地改写调用方底层数组（slice 别名）；
	// 输入字节序列仍严格是 password || salt || 0x00000001，与历史实现一致。
	blockData := make([]byte, 0, len(password)+len(salt)+4)
	blockData = append(blockData, password...)
	blockData = append(blockData, salt...)
	blockData = append(blockData, 0x00, 0x00, 0x00, 0x01)

	// 第一次迭代：计算U1
	h := sm3.New()
	h.Write(blockData)
	copy(block, h.Sum(nil))
	copy(result, block) // 初始化结果为U1

	// 后续迭代：计算U2到Uc
	for i := 1; i < iterations; i++ {
		h.Reset()
		h.Write(block)     // 上一次的结果作为输入
		block = h.Sum(nil) // 计算新的Ui

		// XOR累加：result = result XOR Ui
		for j := 0; j < keyLen && j < len(block); j++ {
			result[j] ^= block[j]
		}
	}

	return result, nil
}

// HashPassword 哈希密码
// 使用国密SM3-PBKDF2算法对密码进行安全哈希
// 返回格式：$sm3$iterations$salt$hash
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	// 1. 生成随机盐值
	salt := make([]byte, pm.config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成盐失败: %w", err)
	}

	// 2. 使用SM3进行PBKDF2迭代哈希（keyLen 恒 32，接 KDF error）
	hash, err := pm.pbkdf2SM3([]byte(password), salt, pm.config.Iterations, 32)
	if err != nil {
		return "", fmt.Errorf("哈希密码失败: %w", err)
	}

	// 3. 编码为Base64格式，便于存储
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 4. 组合存储格式：$sm3$iterations$salt$hash
	// 便于解析和验证，同时包含所有必要信息
	format := "$sm3$%d$%s$%s"
	return fmt.Sprintf(format, pm.config.Iterations, b64Salt, b64Hash), nil
}

// VerifyPassword 验证密码
// 验证明文密码是否与存储的哈希值匹配
func (pm *PasswordManager) VerifyPassword(password, hash string) (bool, error) {
	// 1. 解析哈希格式：$sm3$iterations$salt$hash
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[1] != "sm3" {
		return false, fmt.Errorf("解析密码哈希失败: 无效的格式")
	}

	// 2. 提取迭代次数
	iterations, err := strconv.Atoi(parts[2])
	if err != nil {
		return false, fmt.Errorf("解析迭代次数失败: %w", err)
	}

	// 3. 解码Base64编码的盐值和哈希
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("解码盐失败: %w", err)
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("解码哈希失败: %w", err)
	}

	// 4. 显式校验存储哈希长度：$sm3$ 格式恒 32 字节
	// 非 32 字节属数据损坏 / 伪造，与「密码输错」不同安全类别，返回 error（可观测 / 可审计）。
	if len(hashBytes) != 32 {
		return false, fmt.Errorf("密码哈希长度非法: got %d bytes, want 32（$sm3$ 格式恒 32 字节）", len(hashBytes))
	}

	// 5. 使用相同参数重新计算密码哈希（keyLen 写字面量 32，不再用 len(hashBytes) 作 keyLen）
	comparisonHash, err := pm.pbkdf2SM3([]byte(password), salt, iterations, 32)
	if err != nil {
		return false, fmt.Errorf("重算密码哈希失败: %w", err)
	}

	// 6. 使用恒定时间比较，防止时序攻击
	return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
}

// GenerateRandomPassword 生成随机密码
// 生成指定长度的强密码，包含大小写字母、数字和特殊字符
func GenerateRandomPassword(length int) (string, error) {
	// 确保最小长度为8位
	if length < 8 {
		length = 8
	}

	// 定义字符集：大小写字母、数字和特殊字符
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	// P1 fix: 使用 crypto/rand.Int 替代 randomByte[0] % len(charset)。
	// 取模会引入模偏置 (charset=70 时 256 mod 70 = 46,前 46 字符出现概率
	// 比后 24 字符高 ~20%),降低密码熵。
	// rand.Int(reader, max) 使用拒绝采样保证 [0, max) 均匀分布,无偏置。
	for i := range password {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("生成随机密码失败: %w", err)
		}
		password[i] = charset[idx.Int64()]
	}

	return string(password), nil
}

// GenerateRandomToken 生成随机令牌
// 生成指定长度的随机令牌，用于验证码、重置令牌等场景
func GenerateRandomToken(length int) (string, error) {
	// 确保最小长度为32位
	if length < 32 {
		length = 32
	}

	// 生成随机字节序列
	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("生成随机令牌失败: %w", err)
	}

	// 使用URL安全的Base64编码，避免特殊字符问题
	return base64.RawURLEncoding.EncodeToString(token), nil
}

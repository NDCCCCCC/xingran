package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/tjfoc/gmsm/sm2"
)

// JWTManager JWT管理器
type JWTManager struct {
	secretKey        string
	accessKeyExpire  time.Duration
	refreshKeyExpire time.Duration
	issuer           string
	useSM2           bool            // 是否使用SM2算法
	sm2PrivateKey    *sm2.PrivateKey // SM2私钥
	sm2PublicKey     *sm2.PublicKey  // SM2公钥
}

// CustomClaims 自定义JWT声明
type CustomClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(cfg *config.JWTConfig) (*JWTManager, error) {
	// F-04: 强制校验密钥配置,拒绝空值和已知弱默认值,
	// 防止生产环境用公开字符串伪造 token。
	if !cfg.UseSM2 {
		if cfg.SecretKey == "" {
			return nil, fmt.Errorf("JWT secret_key 未配置 — 必须通过 env 或 config 注入 (建议 ≥32 字节高熵随机串)")
		}
		// 拒绝历史硬编码默认值
		if cfg.SecretKey == "xingran-next-secret-key" {
			return nil, fmt.Errorf("JWT secret_key 使用了已知弱默认值 'xingran-next-secret-key' — 任何人都能伪造 token,必须替换")
		}
		if len(cfg.SecretKey) < 16 {
			return nil, fmt.Errorf("JWT secret_key 长度过短 (%d 字节) — 至少 16 字节,推荐 ≥32 字节", len(cfg.SecretKey))
		}
	}

	jwtManager := &JWTManager{
		secretKey:        cfg.SecretKey,
		accessKeyExpire:  time.Duration(cfg.AccessKeyExpire) * time.Second,
		refreshKeyExpire: time.Duration(cfg.RefreshKeyExpire) * time.Second,
		issuer:           cfg.Issuer,
		useSM2:           cfg.UseSM2,
	}

	// 如果配置使用SM2算法，则初始化SM2密钥对
	if cfg.UseSM2 {
		// 如果配置中有密钥，则使用配置的密钥
		if cfg.SM2PrivateKey != "" && cfg.SM2PublicKey != "" {
			privateKey, err := crypto.ParsePrivateKeyFromHex(cfg.SM2PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("解析SM2私钥失败: %w", err)
			}

			publicKey, err := crypto.ParsePublicKeyFromHex(cfg.SM2PublicKey)
			if err != nil {
				return nil, fmt.Errorf("解析SM2公钥失败: %w", err)
			}

			jwtManager.sm2PrivateKey = privateKey
			jwtManager.sm2PublicKey = publicKey
		} else {
			// 否则生成新的密钥对
			privateKey, publicKey, err := crypto.GenerateKeyPair()
			if err != nil {
				return nil, fmt.Errorf("生成SM2密钥对失败: %w", err)
			}

			jwtManager.sm2PrivateKey = privateKey
			jwtManager.sm2PublicKey = publicKey
		}
	}

	return jwtManager, nil
}

// GenerateTokenPair 生成令牌对
func (j *JWTManager) GenerateTokenPair(userID, username, nickname string, roles []string) (*TokenPair, error) {
	now := time.Now()

	// 生成访问令牌
	accessClaims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Nickname: nickname,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        fmt.Sprintf("%d", now.Unix()),
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{"xingran-next"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessKeyExpire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	var accessToken string
	var err error
	if j.useSM2 {
		// 使用SM2算法签名
		claims := &crypto.Claims{
			UserID:   accessClaims.UserID,
			Username: accessClaims.Username,
			Roles:    accessClaims.Roles,
			Issuer:   accessClaims.Issuer,
			Subject:  accessClaims.Subject,
			Audience: accessClaims.Audience,
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        accessClaims.ID,
				ExpiresAt: accessClaims.ExpiresAt,
				NotBefore: accessClaims.NotBefore,
				IssuedAt:  accessClaims.IssuedAt,
			},
		}
		accessToken, err = crypto.GenerateTokenWithSM2(claims, j.sm2PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("生成SM2访问令牌失败: %w", err)
		}
	} else {
		// 使用HS256算法签名
		accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(j.secretKey))
		if err != nil {
			return nil, fmt.Errorf("生成访问令牌失败: %w", err)
		}
	}

	// 生成刷新令牌
	refreshClaims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Roles:    []string{"refresh"}, // 刷新令牌只包含刷新权限
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        fmt.Sprintf("%d", now.Unix()),
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{"xingran-next"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshKeyExpire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	var refreshToken string
	if j.useSM2 {
		// 使用SM2算法签名
		refreshToken, err = crypto.GenerateRefreshTokenWithSM2(refreshClaims.UserID, refreshClaims.Username, j.issuer, j.refreshKeyExpire, j.sm2PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("生成SM2刷新令牌失败: %w", err)
		}
	} else {
		// 使用HS256算法签名
		refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(j.secretKey))
		if err != nil {
			return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
		}
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(j.accessKeyExpire.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken 验证令牌
func (j *JWTManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	if j.useSM2 {
		// 使用SM2算法验证
		claims, err := crypto.ValidateTokenWithSM2(tokenString, j.sm2PublicKey)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, response.ErrTokenExpired
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				return nil, response.ErrTokenNotValidYet
			}
			return nil, response.ErrTokenInvalid
		}

		// 验证签发者
		if claims.Issuer != j.issuer {
			return nil, response.ErrTokenInvalid
		}

		// 转换为CustomClaims格式
		customClaims := &CustomClaims{
			UserID:           claims.UserID,
			Username:         claims.Username,
			Roles:            claims.Roles,
			RegisteredClaims: claims.RegisteredClaims,
		}

		return customClaims, nil
	} else {
		// 使用HS256算法验证
		token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
			}
			return []byte(j.secretKey), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				return nil, response.ErrTokenExpired
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				return nil, response.ErrTokenNotValidYet
			}
			return nil, response.ErrTokenInvalid
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok || !token.Valid {
			return nil, response.ErrTokenInvalid
		}

		// 验证签发者
		if claims.Issuer != j.issuer {
			return nil, response.ErrTokenInvalid
		}

		return claims, nil
	}
}

// RefreshToken 刷新令牌
func (j *JWTManager) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// 验证刷新令牌
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// 检查是否为刷新令牌
	if len(claims.Roles) != 1 || claims.Roles[0] != "refresh" {
		return nil, response.ErrTokenInvalid
	}

	// 从数据库获取用户最新的角色信息
	// 这里返回空角色，调用者需要从数据库加载
	roles := []string{}

	// 生成新的令牌对（使用 claims 中的 nickname，如果没有则用 username）
	nickname := claims.Nickname
	if nickname == "" {
		nickname = claims.Username
	}
	return j.GenerateTokenPair(claims.UserID, claims.Username, nickname, roles)
}

// GetPublicKey 获取 SM2 公钥（用于前端加密密码）
// 返回十六进制格式的公钥（原始字节格式，用于 sm-crypto）
func (j *JWTManager) GetPublicKey() string {
	if !j.useSM2 || j.sm2PublicKey == nil {
		return ""
	}

	// 返回原始十六进制格式，供前端 sm-crypto 直接使用
	return crypto.ExportPublicKeyToHex(j.sm2PublicKey)
}

// DecryptPassword 使用 SM2 私钥解密密码
func (j *JWTManager) DecryptPassword(ciphertext string) (string, error) {
	if !j.useSM2 || j.sm2PrivateKey == nil {
		return "", fmt.Errorf("SM2 未启用")
	}

	// 使用 SM2 解密密码
	plaintext, err := crypto.DecryptWithSM2(ciphertext, j.sm2PrivateKey)
	if err != nil {
		return "", fmt.Errorf("SM2 解密失败: %w", err)
	}

	return plaintext, nil
}

// GetSM2KeyPair 获取 SM2 密钥对（用于请求体加密）
// 返回私钥和公钥，如果 SM2 未启用则返回 nil
func (j *JWTManager) GetSM2KeyPair() (*sm2.PrivateKey, *sm2.PublicKey) {
	if !j.useSM2 {
		return nil, nil
	}
	return j.sm2PrivateKey, j.sm2PublicKey
}

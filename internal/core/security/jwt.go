package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// JWTManager JWT管理器
type JWTManager struct {
	secretKey        string
	accessKeyExpire  time.Duration
	refreshKeyExpire time.Duration
	issuer           string
}

// CustomClaims 自定义JWT声明
type CustomClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
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
func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	return &JWTManager{
		secretKey:        cfg.SecretKey,
		accessKeyExpire:  time.Duration(cfg.AccessKeyExpire) * time.Second,
		refreshKeyExpire: time.Duration(cfg.RefreshKeyExpire) * time.Second,
		issuer:           cfg.Issuer,
	}
}

// GenerateTokenPair 生成令牌对
func (j *JWTManager) GenerateTokenPair(userID, username string, roles []string) (*TokenPair, error) {
	now := time.Now()

	// 生成访问令牌
	accessClaims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        fmt.Sprintf("%d", now.Unix()),
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{"ruoyi-next"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessKeyExpire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
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
			Audience:  []string{"ruoyi-next"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshKeyExpire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
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

	// TODO: 从数据库获取用户最新的角色信息
	// 这里简化处理，实际应该查询数据库
	roles := []string{"user"} // 临时使用

	// 生成新的令牌对
	return j.GenerateTokenPair(claims.UserID, claims.Username, roles)
}
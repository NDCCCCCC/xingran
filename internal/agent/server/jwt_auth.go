package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// JWTAuthenticator JWT 认证管理器
type JWTAuthenticator struct {
	secret         string
	tokenExpiry    time.Duration
	backendURL     string
	currentToken   string
	tokenExpiryAt  time.Time
	agentID        string
	vmID           string
	httpClient     *tlsConfig
}

type tlsConfig struct {
	TLSConfig *tls.Config
}

// jwtClaims JWT 声明
type jwtClaims struct {
	AgentID string `json:"agent_id"`
	VMID    string `json:"vm_id"`
	jwt.RegisteredClaims
}

// NewJWTAuthenticator 创建 JWT 认证管理器
func NewJWTAuthenticator(secret, backendURL, agentID, vmID string) *JWTAuthenticator {
	return &JWTAuthenticator{
		secret:      secret,
		backendURL:  backendURL,
		tokenExpiry: 24 * time.Hour,
		agentID:     agentID,
		vmID:        vmID,
		httpClient: &tlsConfig{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true, // 生产环境应验证证书
			},
		},
	}
}

// Register 注册 Agent 到后端并获取初始令牌
func (a *JWTAuthenticator) Register(ctx context.Context) error {
	// 调用后端注册 API
	// POST /api/v1/agent/register
	// 这里简化处理，实际应发送 HTTP 请求

	// 模拟注册成功，生成初始令牌
	token, err := a.generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	a.currentToken = token
	a.tokenExpiryAt = time.Now().Add(a.tokenExpiry)

	return nil
}

// RefreshToken 刷新令牌（提前 1 小时）
func (a *JWTAuthenticator) RefreshToken(ctx context.Context) error {
	if a.currentToken != "" && time.Now().Add(1*time.Hour).Before(a.tokenExpiryAt) {
		return nil // 令牌仍然有效
	}

	// 调用后端刷新 API
	// POST /api/v1/agent/refresh
	// 这里简化处理，生成新令牌
	token, err := a.generateToken()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	a.currentToken = token
	a.tokenExpiryAt = time.Now().Add(a.tokenExpiry)

	return nil
}

// generateToken 生成 JWT 令牌
func (a *JWTAuthenticator) generateToken() (string, error) {
	claims := jwtClaims{
		AgentID: a.agentID,
		VMID:    a.vmID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secret))
}

// ValidateToken 验证令牌有效性
func (a *JWTAuthenticator) ValidateToken(tokenString string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwtClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetCurrentToken 获取当前有效令牌
func (a *JWTAuthenticator) GetCurrentToken(ctx context.Context) (string, error) {
	if err := a.RefreshToken(ctx); err != nil {
		return "", err
	}
	return a.currentToken, nil
}

// CallBackend 调用后端 API
func (a *JWTAuthenticator) CallBackend(ctx context.Context, method, path string, body interface{}) (*response.Response, error) {
	// 获取有效令牌
	token, err := a.GetCurrentToken(ctx)
	if err != nil {
		return nil, err
	}

	// 构建请求
	// 这里简化处理，实际应使用 HTTP 客户端
	_ = token
	_ = method
	_ = path
	_ = body

	return &response.Response{
		Code: 0,
		Message:  "success",
	}, nil
}

// SendHeartbeat 发送心跳
func (a *JWTAuthenticator) SendHeartbeat(ctx context.Context) error {
	heartbeatData := map[string]interface{}{
		"agent_id": a.agentID,
		"vm_id":    a.vmID,
		"status":   "healthy",
		"timestamp": time.Now().Unix(),
	}

	_, err := a.CallBackend(ctx, "POST", "/api/v1/agent/heartbeat", heartbeatData)
	return err
}

// ReportSystemStatus 报告系统状态
func (a *JWTAuthenticator) ReportSystemStatus(ctx context.Context, status map[string]interface{}) error {
	status["agent_id"] = a.agentID
	status["vm_id"] = a.vmID
	status["timestamp"] = time.Now().Unix()

	_, err := a.CallBackend(ctx, "POST", "/api/v1/agent/status", status)
	return err
}

// ParseTokenClaims 解析令牌声明
func ParseTokenClaims(tokenString string) (map[string]interface{}, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// 转换为 map[string]interface{}
		result := make(map[string]interface{})
		for k, v := range claims {
			result[k] = v
		}
		return result, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// RegisterToBackend 向后端注册 Agent
func (a *JWTAuthenticator) RegisterToBackend(ctx context.Context, registrationData map[string]interface{}) error {
	registrationData["agent_id"] = a.agentID
	registrationData["vm_id"] = a.vmID
	registrationData["platform"] = runtime.GOOS

	resp, err := a.CallBackend(ctx, "POST", "/api/v1/vdi/agent/register", registrationData)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	// 从响应中提取 JWT 令牌
	if resp.Data != nil {
		dataBytes, err := json.Marshal(resp.Data)
		if err != nil {
			return err
		}

		var result struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(dataBytes, &result); err == nil && result.Token != "" {
			a.currentToken = result.Token
			a.tokenExpiryAt = time.Now().Add(a.tokenExpiry)
		}
	}

	return nil
}

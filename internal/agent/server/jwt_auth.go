package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// API 路径常量
const (
	APIPathHeartbeat   = "/api/v1/agent/heartbeat"
	APIPathStatus      = "/api/v1/agent/status"
	APIPathRegister    = "/api/v1/vdi/agent/register"
	APIPathTokenRefresh = "/api/v1/agent/refresh"
)

// 默认令牌过期时间
const defaultTokenExpiry = 24 * time.Hour

// 错误消息
const (
	errInvalidSigningMethod = "意外的签名方法"
	errInvalidToken         = "无效令牌"
	errRegistrationFailed   = "注册失败"
)

// JWTAuthenticator JWT 认证管理器
type JWTAuthenticator struct {
	secret         string
	tokenExpiry    time.Duration
	backendURL     string
	agentID        string
	vmID           string
	httpClient     *http.Client
	mu             sync.RWMutex
	currentToken   string
	tokenExpiryAt  time.Time
}

// jwtClaims JWT 声明
type jwtClaims struct {
	AgentID string `json:"agent_id"`
	VMID    string `json:"vm_id"`
	jwt.RegisteredClaims
}

// NewJWTAuthenticator 创建 JWT 认证管理器
func NewJWTAuthenticator(secret, backendURL, agentID, vmID string, tlsConfig *tls.Config) *JWTAuthenticator {
	// 如果未提供 TLS 配置，使用安全的默认值
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS13,  // 强制 TLS 1.3
			// InsecureSkipVerify 默认为 false，不设置
		}
	}

	return &JWTAuthenticator{
		secret:       secret,
		backendURL:   strings.TrimSuffix(backendURL, "/"),
		tokenExpiry:  defaultTokenExpiry,
		agentID:      agentID,
		vmID:         vmID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				// 启用 HTTP/2 提升性能
				ForceAttemptHTTP2: true,
			},
		},
	}
}

// NewTLSConfigFromConfig 从配置创建 TLS 配置
func NewTLSConfigFromConfig(certFile, keyFile, caFile string, verifyCertificates bool) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// 如果提供了 CA 文件，加载 CA 证书
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		config.RootCAs = caCertPool
	}

	// 如果提供了客户端证书（mTLS），加载证书和私钥
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	// 根据配置决定是否验证服务器证书
	config.InsecureSkipVerify = !verifyCertificates

	if !verifyCertificates {
		Warn("Certificate verification is DISABLED (insecure)")
	}

	return config, nil
}

// AuthError 认证错误（不重试）
type AuthError struct {
	StatusCode int
	Message    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error: %s (status: %d)", e.Message, e.StatusCode)
}

// HTTPError HTTP 错误
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: %s (status: %d)", e.Message, e.StatusCode)
}

// Register 注册 Agent 到后端并获取初始令牌
func (a *JWTAuthenticator) Register(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

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
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检查令牌是否仍然有效（提前 1 小时刷新）
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

// generateToken 生成 JWT 令牌（内部方法，调用者需持有锁）
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
			return nil, fmt.Errorf("%s: %v", errInvalidSigningMethod, token.Header["alg"])
		}
		return []byte(a.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwtClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf(errInvalidToken)
}

// GetCurrentToken 获取当前有效令牌（带读锁保护）
func (a *JWTAuthenticator) GetCurrentToken(ctx context.Context) (string, error) {
	// 先用读锁快速检查是否需要刷新
	a.mu.RLock()
	tokenValid := a.currentToken != "" && time.Now().Add(1*time.Hour).Before(a.tokenExpiryAt)
	currentToken := a.currentToken
	a.mu.RUnlock()

	if tokenValid {
		return currentToken, nil
	}

	// 需要刷新，RefreshToken 内部会获取写锁
	if err := a.RefreshToken(ctx); err != nil {
		return "", err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentToken, nil
}

// CallBackend 调用后端 API（实际 HTTP 实现）
func (a *JWTAuthenticator) CallBackend(ctx context.Context, method, path string, body interface{}) (*response.Response, error) {
	// 获取有效令牌
	token, err := a.GetCurrentToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// 构建请求
	url := a.backendURL + path

	// 序列化请求体
	var bodyReader *bytes.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求（使用配置的 TLS 客户端）
	resp, err := a.httpClient.Do(req)
	if err != nil {
		// 检查是否是 TLS 错误
		if strings.Contains(err.Error(), "x509") || strings.Contains(err.Error(), "certificate") {
			return nil, fmt.Errorf("TLS certificate verification failed: %w", err)
		}
		return nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result response.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// SendHeartbeat 发送心跳
func (a *JWTAuthenticator) SendHeartbeat(ctx context.Context) error {
	heartbeatData := map[string]interface{}{
		"agent_id": a.agentID,
		"vm_id":    a.vmID,
		"status":   "healthy",
		"timestamp": time.Now().Unix(),
	}

	_, err := a.CallBackend(ctx, "POST", APIPathHeartbeat, heartbeatData)
	return err
}

// ReportSystemStatus 报告系统状态
func (a *JWTAuthenticator) ReportSystemStatus(ctx context.Context, status map[string]interface{}) error {
	status["agent_id"] = a.agentID
	status["vm_id"] = a.vmID
	status["timestamp"] = time.Now().Unix()

	_, err := a.CallBackend(ctx, "POST", APIPathStatus, status)
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

	resp, err := a.CallBackend(ctx, "POST", APIPathRegister, registrationData)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("%s: %s", errRegistrationFailed, resp.Message)
	}

	// 从响应中提取 JWT 令牌
	if resp.Data != nil {
		// 直接类型断言，避免冗余的序列化/反序列化
		if dataMap, ok := resp.Data.(map[string]interface{}); ok {
			if token, ok := dataMap["token"].(string); ok && token != "" {
				a.mu.Lock()
				a.currentToken = token
				a.tokenExpiryAt = time.Now().Add(a.tokenExpiry)
				a.mu.Unlock()
			}
		}
	}

	return nil
}

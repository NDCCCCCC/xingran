package vdi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	// vdiAuthHTTPTimeout VDI 认证 HTTP 客户端超时
	vdiAuthHTTPTimeout = 30 * time.Second
	// vdiTokenRefreshSkew VDI token 提前刷新阈值
	vdiTokenRefreshSkew = 5 * time.Minute
	// vdiAuthMaxRetries VDI 认证重试次数
	vdiAuthMaxRetries = 3
)

// VDIAuthManager VDI认证管理器
type VDIAuthManager struct {
	db         *gorm.DB
	httpClient *http.Client
	server     models.VDIServer
	serverID   string // VDI服务器ID
}

// NewVDIAuthManager 创建VDI认证管理器（从数据库模型）
func NewVDIAuthManager(db *gorm.DB, serverID string, server models.VDIServer) *VDIAuthManager {
	return &VDIAuthManager{
		db:         db,
		httpClient: &http.Client{
			Timeout: vdiAuthHTTPTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// F-07 fix: 不再硬编码,改为从 config.VDI.TLSSkipVerify 读取
					// 默认 true 保持向后兼容(VDI 自签证书),生产应在 yaml 中显式设 false
					InsecureSkipVerify: loadTLSSkipVerify(),
				},
			},
		},
		server:   server,
		serverID: serverID,
	}
}

// Authenticate 认证并获取token
func (a *VDIAuthManager) Authenticate(ctx context.Context) (string, error) {
	// 1. 检查缓存的token是否有效
	if a.server.AuthToken != "" && !a.isTokenExpired(a.server.TokenExpiry) {
		applogger.Debugf("[VDI AUTH] Using cached token, length: %d, expiry: %v", len(a.server.AuthToken), a.server.TokenExpiry)
		return a.server.AuthToken, nil
	}

	applogger.Debugf("[VDI AUTH] No valid cached token, requesting new token")
	applogger.Debugf("[VDI AUTH] Username: %s", a.server.Username)

	// 2. 从数据库解密密码
	password := decryptVDIPassword(a.server.PasswordEncrypted)
	if password == "" {
		return "", fmt.Errorf("failed to decrypt VDI server password")
	}

	// 3. 调用VDI API认证（使用数据库中的服务器配置）
	req := map[string]interface{}{
		"auth": map[string]string{
			"name":     a.server.Username,
			"password": password,
		},
	}

	var resp struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Token        struct {
			TenantID  int    `json:"tenant_id"`
			AuthToken string `json:"auth_token"`
		} `json:"token"`
	}

	if err := a.callAPIWithEndpoint(ctx, a.server.Endpoint, "/v1/auth/tokens", req, &resp); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	applogger.Debugf("[VDI AUTH] Auth response - ErrorCode: %d, ErrorMessage: %s", resp.ErrorCode, resp.ErrorMessage)
	if resp.ErrorCode != 0 {
		return "", &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	// 4. 缓存token到数据库
	token := resp.Token.AuthToken
	expiry := time.Now().Add(23 * time.Hour) // 提前1小时过期
	applogger.Debugf("[VDI AUTH] New token received, length: %d, tenant_id: %d", len(token), resp.Token.TenantID)

	if err := a.cacheToken(ctx, token, expiry); err != nil {
		return "", fmt.Errorf("failed to cache token: %w", err)
	}

	// 更新内存中的服务器信息
	a.server.AuthToken = token
	a.server.TokenExpiry = &expiry

	return token, nil
}

// RefreshToken 刷新token
func (a *VDIAuthManager) RefreshToken(ctx context.Context) (string, error) {
	return a.Authenticate(ctx)
}

// IsTokenExpired 检查token是否过期
func (a *VDIAuthManager) IsTokenExpired(ctx context.Context) bool {
	// 重新从数据库读取最新的token信息
	var server models.VDIServer
	err := a.db.WithContext(ctx).Where("id = ?", a.serverID).First(&server).Error
	if err != nil {
		return true
	}
	return a.isTokenExpired(server.TokenExpiry)
}

// isTokenExpired 检查token是否过期
func (a *VDIAuthManager) isTokenExpired(expiry *time.Time) bool {
	if expiry == nil {
		return true
	}
	// 提前 vdiTokenRefreshSkew 判断为过期，避免临界情况
	return time.Now().Add(vdiTokenRefreshSkew).After(*expiry)
}

// cacheToken 缓存token到数据库
func (a *VDIAuthManager) cacheToken(ctx context.Context, token string, expiry time.Time) error {
	return a.db.WithContext(ctx).
		Model(&models.VDIServer{}).
		Where("id = ?", a.serverID).
		Updates(map[string]interface{}{
			"auth_token":   token,
			"token_expiry": &expiry,
		}).Error
}

// callAPI 调用VDI API
func (a *VDIAuthManager) callAPI(ctx context.Context, path string, body, result interface{}) error {
	return a.callAPIWithEndpoint(ctx, a.server.Endpoint, path, body, result)
}

// ClearTokenCache 清除 token 缓存（用于处理 token 失效的情况）
func (a *VDIAuthManager) ClearTokenCache() {
	applogger.Debugf("[VDI AUTH] Clearing token cache")
	a.server.AuthToken = ""
	a.server.TokenExpiry = nil
}

// callAPIWithEndpoint 使用指定端点调用VDI API
func (a *VDIAuthManager) callAPIWithEndpoint(ctx context.Context, endpoint, path string, body, result interface{}) error {
	url := fmt.Sprintf("%s%s", endpoint, path)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// 重试逻辑
	var lastErr error
	for i := 0; i < vdiAuthMaxRetries; i++ {
		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1)) // 指数退避
			continue
		}
		defer resp.Body.Close()

		// 读取原始响应体
		rawBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(rawBody))
		}

		// 解析JSON响应
		if err := json.Unmarshal(rawBody, result); err != nil {
			return fmt.Errorf("failed to decode JSON: %w, raw response: %s", err, string(rawBody))
		}

		return nil
	}

	return fmt.Errorf("HTTP request failed after retries: %w", lastErr)
}

// VDIError VDI API 错误类型
type VDIError struct {
	Code    int
	Message string
}

func (e *VDIError) Error() string {
	return fmt.Sprintf("VDI API error %d: %s", e.Code, e.Message)
}

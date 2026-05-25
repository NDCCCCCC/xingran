package vdi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"gorm.io/gorm"
)

// VDIAuthManager VDI认证管理器
type VDIAuthManager struct {
	db         *gorm.DB
	httpClient *http.Client
	config     config.VDIServerConfig
	serverID   string // VDI服务器ID
}

// NewVDIAuthManager 创建VDI认证管理器
func NewVDIAuthManager(db *gorm.DB, serverID string, cfg config.VDIServerConfig) *VDIAuthManager {
	return &VDIAuthManager{
		db:         db,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		config:     cfg,
		serverID:   serverID,
	}
}

// Authenticate 认证并获取token
func (a *VDIAuthManager) Authenticate(ctx context.Context) (string, error) {
	// 1. 检查缓存的token是否有效
	var server models.VDIServer
	err := a.db.WithContext(ctx).Where("id = ?", a.serverID).First(&server).Error
	if err == nil && server.AuthToken != "" && !a.isTokenExpired(server.TokenExpiry) {
		return server.AuthToken, nil
	}

	// 2. 从数据库解密密码
	password := decryptVDIPassword(server.PasswordEncrypted)
	if password == "" {
		return "", fmt.Errorf("failed to decrypt VDI server password")
	}

	// 3. 调用VDI API认证
	req := map[string]interface{}{
		"auth": map[string]string{
			"name":     a.config.Username,
			"password": password,
		},
	}

	var resp struct {
		ErrorCode    int `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		Data         struct {
			Token struct {
				TenantID  int    `json:"tenant_id"`
				AuthToken string `json:"auth_token"`
			} `json:"token"`
		} `json:"data"`
	}

	if err := a.callAPI(ctx, "/v1/auth/tokens", req, &resp); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	if resp.ErrorCode != 0 {
		return "", &VDIError{Code: resp.ErrorCode, Message: resp.ErrorMessage}
	}

	// 4. 缓存token到数据库
	token := resp.Data.Token.AuthToken
	expiry := time.Now().Add(23 * time.Hour) // 提前1小时过期

	if err := a.cacheToken(ctx, token, expiry); err != nil {
		return "", fmt.Errorf("failed to cache token: %w", err)
	}

	return token, nil
}

// RefreshToken 刷新token
func (a *VDIAuthManager) RefreshToken(ctx context.Context) (string, error) {
	return a.Authenticate(ctx)
}

// IsTokenExpired 检查token是否过期
func (a *VDIAuthManager) IsTokenExpired(ctx context.Context) bool {
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
	// 提前5分钟判断为过期，避免临界情况
	return time.Now().Add(5 * time.Minute).After(*expiry)
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
	url := fmt.Sprintf("%s%s", a.config.Endpoint, path)

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
	for i := 0; i < 3; i++ {
		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1)) // 指数退避
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return json.NewDecoder(resp.Body).Decode(result)
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

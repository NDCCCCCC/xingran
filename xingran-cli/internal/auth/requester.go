package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// 加密模式常量（EncryptionMode）。
const (
	EncryptionModeAuto = "auto" // 默认：探测服务端 GET /api/v1/system/auth/encryption-config，enabled=true 才加密
	EncryptionModeOn   = "on"   // 强制加密（服务端 enabled=true 场景 / 安全不降级要求）
	EncryptionModeOff  = "off"  // 强制明文
)

// HTTPRequester 基于 net/http 的默认 Requester 实现。
//
// 加密策略（EncryptionMode，默认 auto）：
//   - auto: 启动后首次加密请求前探测服务端动态配置（数据库键 sys.request.encryption.enabled
//     经 GET /api/v1/system/auth/encryption-config 暴露）。enabled=false 时明文直发
//     （服务端解密中间件直通，加密 envelope 反而会被 handler 当普通 body 解析失败）；
//     enabled=true 或探测网络失败时按加密发送（安全优先，失败即暴露而非静默降级）。
//   - on/off: 跳过探测。
//
// token 非空时携带 Authorization: <tokenType> <token>（默认 Bearer）。
type HTTPRequester struct {
	BaseURL    string // 如 http://localhost:9000，不包含 /api/v1
	HTTPClient *http.Client
	Encryptor  Encryptor // 可为 nil（无加密能力，等价 off）
	TokenType  string    // 默认 Bearer

	EncryptionMode string // auto(默认)|on|off

	mu             sync.Mutex
	encryptionOnce bool
	encryptionOn   bool // 探测缓存结果
}

// apiV1Prefix 后端业务路由前缀（router 挂载在 /api/v1 下）。
const apiV1Prefix = "/api/v1"

// encryptionConfigPath 服务端加密开关探测端点（公开接口）。
const encryptionConfigPath = "/system/auth/encryption-config"

func (r *HTTPRequester) client() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (r *HTTPRequester) tokenType() string {
	if r.TokenType != "" {
		return r.TokenType
	}
	return "Bearer"
}

// shouldEncrypt 判断本次会话是否加密请求体（auto 模式探测一次并缓存）。
func (r *HTTPRequester) shouldEncrypt(ctx context.Context) (bool, error) {
	if r.Encryptor == nil {
		return false, nil
	}
	switch r.EncryptionMode {
	case EncryptionModeOff:
		return false, nil
	case EncryptionModeOn:
		return true, nil
	}
	// auto
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.encryptionOnce {
		return r.encryptionOn, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.BaseURL+apiV1Prefix+encryptionConfigPath, nil)
	if err != nil {
		return false, err
	}
	resp, err := r.client().Do(req)
	if err != nil {
		// 探测网络失败：安全优先，按加密启用（服务端 require_encryption=false
		// 时明文也合法，但降级决定权交由显式 EncryptionMode=off）
		r.encryptionOnce, r.encryptionOn = true, true
		return true, nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		r.encryptionOnce, r.encryptionOn = true, true
		return true, nil
	}
	var cfg struct {
		Data struct {
			Enabled bool `json:"enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		r.encryptionOnce, r.encryptionOn = true, true
		return true, nil
	}
	r.encryptionOnce, r.encryptionOn = true, cfg.Data.Enabled
	return r.encryptionOn, nil
}

// Post 实现 Requester。
func (r *HTTPRequester) Post(ctx context.Context, path string, body any, token string) (*Response, error) {
	var payload []byte
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		payload = raw
		encrypt, err := r.shouldEncrypt(ctx)
		if err != nil {
			return nil, fmt.Errorf("加密配置探测失败: %w", err)
		}
		if encrypt {
			enc, err := r.Encryptor.EncryptRequest(ctx, payload)
			if err != nil {
				return nil, fmt.Errorf("请求加密失败: %w", err)
			}
			payload = enc
		}
	}

	url := r.BaseURL + apiV1Prefix + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", r.tokenType()+" "+token)
	}

	resp, err := r.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result Response
	if err := json.Unmarshal(data, &result); err != nil {
		// 非 JSON 响应（如网关错误），保留 HTTP 状态码
		return nil, fmt.Errorf("响应解析失败(HTTP %d): %s", resp.StatusCode, truncate(string(data), 200))
	}
	return &result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

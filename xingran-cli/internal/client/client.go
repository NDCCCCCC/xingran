// Package client 是 XingRan CLI 的 HTTP 核心。
//
// 职责：
//   - 按契约组装请求（可选 SM2+SM4 加密，经 Encryptor 接口接入）
//   - 解析响应包装 {code,message,data,timestamp,request_id}
//   - 401 / 1007 时经 Authenticator.Refresh 自动刷新 token 并重试一次
//   - 提供 --dry-run 能力（打印将发送的请求而不发送）
//
// 与 crypto / auth 模块只通过本包声明的 Encryptor / Authenticator 接口耦合，
// 最终在 cmd 层组装（依赖倒置，支持并行开发）。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
)

// Encryptor 契约接口（由 internal/crypto 实现，本包只依赖接口）。
type Encryptor interface {
	FetchPublicKey(ctx context.Context, baseURL string) error
	EncryptRequest(ctx context.Context, body []byte) (payload []byte, err error)
}

// Authenticator 契约接口（由 internal/auth 实现，本包只依赖接口；
// CaptchaInput / TokenPair 类型以 internal/auth 为准）。
type Authenticator interface {
	Login(ctx context.Context, username, password string, captcha auth.CaptchaInput) (*auth.TokenPair, error)
	Refresh(ctx context.Context) (*auth.TokenPair, error)
	GetAccessToken(ctx context.Context) (string, error) // 过期自动 refresh
	Logout(ctx context.Context) error
}

// Client HTTP 核心契约接口。
type Client interface {
	Do(ctx context.Context, method, path string, body any) (*Response, error)
	DryRun(ctx context.Context, method, path string, body any) (*DryRunRequest, error)
	// DoUpload multipart/form-data 文件上传（M8：Excel 导入）。
	DoUpload(ctx context.Context, path, fileField, filePath string, fields map[string]string) (*Response, error)
	// DoDownload 二进制流下载（M8：export/template），返回实际保存路径。
	// method 为空按 GET 处理；destPath 为空时从 Content-Disposition 解析。
	DoDownload(ctx context.Context, path, jsonBody, destPath, method string) (string, error)
}

// Response 后端统一响应包装。
type Response struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	RequestID string          `json:"request_id"`
}

// OK 判断业务是否成功。
func (r *Response) OK() bool { return r.Code == 0 }

// APIError 服务端返回的非 0 code 业务错误。
type APIError struct {
	Resp       Response
	HTTPStatus int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("服务端返回错误 code=%d http=%d: %s", e.Resp.Code, e.HTTPStatus, e.Resp.Message)
}

// TransportError 网络传输层错误（连接失败、超时、非 JSON 响应等）。
type TransportError struct {
	Method string
	URL    string
	Err    error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("网络错误 %s %s: %v", e.Method, e.URL, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// Options Client 构造选项。
type Options struct {
	BaseURL       string        // 例如 http://localhost:9000
	Timeout       time.Duration // 单请求超时，默认 30s
	Encryptor     Encryptor     // 可为 nil（端点未启用加密）
	Authenticator Authenticator // 可为 nil（未登录 / 公开端点）
	HTTPClient    *http.Client  // 测试注入用；nil 则按 Timeout 构造
}

// token 重试相关的响应 code / 语义。
const (
	codeInvalidToken = 1007
	defaultTimeout   = 30 * time.Second
	authHeaderPrefix = "Bearer "
	defaultUserAgent = "xingran-cli/0.1.0"
	maxRetryAuthOnce = 1 // 401/1007 只重试一次
)

type clientImpl struct {
	baseURL       string
	httpClient    *http.Client
	encryptor     Encryptor
	authenticator Authenticator
	pubFetched    bool // FetchPublicKey 只需成功一次（crypto 内部持有缓存）
}

// New 构造 HTTP 核心 Client。
func New(opts Options) Client {
	hc := opts.HTTPClient
	if hc == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		hc = &http.Client{Timeout: timeout}
	}
	return &clientImpl{
		baseURL:       strings.TrimRight(opts.BaseURL, "/"),
		httpClient:    hc,
		encryptor:     opts.Encryptor,
		authenticator: opts.Authenticator,
	}
}

// DryRunRequest --dry-run 模式下描述"将发送的请求"。
type DryRunRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"` // 将要发送的 body 文本
	Encrypted bool              `json:"encrypted"`
}

// DryRun 构造请求但不发送，返回将发送内容的描述。
//
// dry-run 完全离线：不拉公钥、不加密、不发任何网络请求。
// Body 为原始明文 JSON，Encrypted 如实标注 false（真实发送时会加密）。
func (c *clientImpl) DryRun(ctx context.Context, method, path string, body any) (*DryRunRequest, error) {
	payload, hasBody, err := c.serializeBody(body)
	if err != nil {
		return nil, err
	}
	token, _ := c.token(ctx)
	req, err := c.newRequest(ctx, method, path, payload, hasBody, token)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	for k, v := range req.Header {
		headers[k] = strings.Join(v, ",")
	}
	return &DryRunRequest{
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   headers,
		Body:      string(payload),
		Encrypted: false,
	}, nil
}

// Do 执行一次 API 请求：发送 → 解析包装 → 401/1007 刷新重试一次。
func (c *clientImpl) Do(ctx context.Context, method, path string, body any) (*Response, error) {
	resp, err := c.doOnce(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if c.shouldRetryAuth(resp) {
		// 刷新 token 后重试一次。
		if c.authenticator != nil {
			if _, rerr := c.authenticator.Refresh(ctx); rerr != nil {
				return nil, &APIError{Resp: resp.resp, HTTPStatus: resp.httpStatus}
			}
			resp2, err2 := c.doOnce(ctx, method, path, body)
			if err2 != nil {
				return nil, err2
			}
			return c.resolve(resp2)
		}
	}
	return c.resolve(resp)
}

// attempt 一次 HTTP 尝试的中间结果。
type attempt struct {
	resp       Response
	rawBody    []byte
	httpStatus int
	wrapped    bool // 是否为标准 {code,...} 包装
}

func (a *attempt) ok() bool { return a.wrapped && a.resp.Code == 0 }

func (c *clientImpl) doOnce(ctx context.Context, method, path string, body any) (*attempt, error) {
	payload, hasBody, err := c.buildPayload(ctx, body)
	if err != nil {
		return nil, err
	}
	token, _ := c.token(ctx)
	req, err := c.newRequest(ctx, method, path, payload, hasBody, token)
	if err != nil {
		return nil, &TransportError{Method: method, URL: path, Err: err}
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &TransportError{Method: method, URL: req.URL.String(), Err: err}
	}
	defer func() { _ = httpResp.Body.Close() }()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, &TransportError{Method: method, URL: req.URL.String(), Err: err}
	}

	at := &attempt{rawBody: raw, httpStatus: httpResp.StatusCode}
	var wrapped Response
	if err := json.Unmarshal(raw, &wrapped); err == nil && hasCodeField(raw) {
		at.wrapped = true
		at.resp = wrapped
	}
	return at, nil
}

// resolve 把一次尝试转成最终 Response / error。
func (c *clientImpl) resolve(at *attempt) (*Response, error) {
	if at.wrapped {
		if at.resp.Code == 0 {
			return &at.resp, nil
		}
		return nil, &APIError{Resp: at.resp, HTTPStatus: at.httpStatus}
	}
	// 非标准包装：按 HTTP 状态归类。
	msg := strings.TrimSpace(string(at.rawBody))
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	if at.httpStatus >= 200 && at.httpStatus < 300 {
		// 2xx 但不是包装：视为裸数据返回（兼容极少数端点）。
		return &Response{
			Code:    0,
			Message: "success",
			Data:    at.rawBody,
		}, nil
	}
	return nil, &APIError{
		Resp:       Response{Code: at.httpStatus, Message: msg},
		HTTPStatus: at.httpStatus,
	}
}

// shouldRetryAuth 401 HTTP 或 code=1007（token 无效）且尚未重试过。
func (c *clientImpl) shouldRetryAuth(at *attempt) bool {
	if at.wrapped && at.resp.Code == codeInvalidToken {
		return true
	}
	return at.httpStatus == http.StatusUnauthorized
}

// buildPayload 序列化并（可选）加密请求体。
func (c *clientImpl) buildPayload(ctx context.Context, body any) (payload []byte, hasBody bool, err error) {
	payload, hasBody, err = c.serializeBody(body)
	if err != nil || !hasBody {
		return payload, hasBody, err
	}
	if c.encryptor != nil {
		if err := c.ensurePublicKey(ctx); err != nil {
			return nil, false, err
		}
		payload, err = c.encryptor.EncryptRequest(ctx, payload)
		if err != nil {
			return nil, false, fmt.Errorf("加密请求体失败: %w", err)
		}
	}
	return payload, hasBody, nil
}

// serializeBody 仅序列化请求体（不加密），dry-run 使用。
func (c *clientImpl) serializeBody(body any) (payload []byte, hasBody bool, err error) {
	if body == nil {
		return nil, false, nil
	}
	switch b := body.(type) {
	case []byte:
		payload = b
	case string:
		payload = []byte(b)
	default:
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, false, fmt.Errorf("序列化请求体失败: %w", err)
		}
	}
	return payload, len(payload) > 0, nil
}

// ensurePublicKey 首次加密前拉取后端 SM2 公钥（不降级：失败即返回错误）。
func (c *clientImpl) ensurePublicKey(ctx context.Context) error {
	if c.pubFetched {
		return nil
	}
	if err := c.encryptor.FetchPublicKey(ctx, c.baseURL); err != nil {
		return fmt.Errorf("获取加密公钥失败: %w", err)
	}
	c.pubFetched = true
	return nil
}

func (c *clientImpl) token(ctx context.Context) (string, error) {
	if c.authenticator == nil {
		return "", fmt.Errorf("未登录")
	}
	return c.authenticator.GetAccessToken(ctx)
}

// newRequest 组装 http.Request。
func (c *clientImpl) newRequest(ctx context.Context, method, path string, payload []byte, hasBody bool, token string) (*http.Request, error) {
	u, err := url.Parse(c.baseURL + normalizePath(path))
	if err != nil {
		return nil, fmt.Errorf("非法请求路径 %q: %w", path, err)
	}
	var reader io.Reader
	if hasBody {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), u.String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", authHeaderPrefix+token)
	}
	return req, nil
}

// apiV1Prefix 后端业务路由统一挂载在 /api/v1 下（与 auth.HTTPRequester 行为一致）。
const apiV1Prefix = "/api/v1"

// normalizePath 规范化请求路径：保证以 / 开头，并自动补 /api/v1 前缀
// （path 已带 /api/v1 前缀时不重复添加）。
func normalizePath(p string) string {
	if p == "" {
		return apiV1Prefix + "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasPrefix(p, apiV1Prefix+"/") && !strings.HasPrefix(p, apiV1Prefix) {
		p = apiV1Prefix + p
	}
	return p
}

// hasCodeField 判断 JSON 顶层是否有 "code" 字段（宽松判定，容忍大小写差异）。
func hasCodeField(raw []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	_, ok := probe["code"]
	return ok
}

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
)

// pngBase64 将 PNG 字节编码为标准 base64 字符串。
func pngBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// mockRequester 测试用 Requester：按 path 分发 canned 响应，并记录收到的请求体。
type mockRequester struct {
	mu     sync.Mutex
	routes map[string]*Response // path → 固定响应
	bodies map[string][]map[string]any
	tokens map[string]string // path → 最后一次携带的 token
	calls  []string
}

func newMockRequester() *mockRequester {
	return &mockRequester{
		routes: map[string]*Response{},
		bodies: map[string][]map[string]any{},
		tokens: map[string]string{},
	}
}

func (m *mockRequester) Post(_ context.Context, path string, body any, token string) (*Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, path)
	if resp, ok := m.routes[path]; ok {
		if body != nil {
			raw, _ := json.Marshal(body)
			var parsed map[string]any
			_ = json.Unmarshal(raw, &parsed)
			m.bodies[path] = append(m.bodies[path], parsed)
		}
		m.tokens[path] = token
		return resp, nil
	}
	return nil, fmt.Errorf("mockRequester: 未注册的路径 %s", path)
}

func (m *mockRequester) ok(data any) *Response {
	raw, _ := json.Marshal(data)
	return &Response{Code: 0, Message: "success", Data: raw}
}

func (m *mockRequester) err(code int, msg string) *Response {
	return &Response{Code: code, Message: msg}
}

func (m *mockRequester) lastBody(path string) map[string]any {
	if bs := m.bodies[path]; len(bs) > 0 {
		return bs[len(bs)-1]
	}
	return nil
}

// mockEncryptor 测试用 Encryptor。
type mockEncryptor struct {
	fetchCalls int
	encCalls   int
	failFetch  bool
}

func (e *mockEncryptor) FetchPublicKey(_ context.Context, _ string) error {
	e.fetchCalls++
	if e.failFetch {
		return fmt.Errorf("公钥获取失败")
	}
	return nil
}

func (e *mockEncryptor) EncryptRequest(_ context.Context, body []byte) ([]byte, error) {
	e.encCalls++
	return append([]byte("ENC:"), body...), nil
}

// newTestAuthenticator 构造带临时文件存储的 authenticator。
func newTestAuthenticator(req Requester, enc Encryptor) (Authenticator, *FileTokenStore) {
	store := &FileTokenStore{Path: tempStorePath()}
	return NewAuthenticator(req, store, enc, "http://test.local"), store
}

// fetchCaptcha 通过 CaptchaProvider 扩展接口获取验证码（测试辅助）。
func fetchCaptcha(a Authenticator, ctx context.Context) (*CaptchaInfo, error) {
	return a.(CaptchaProvider).FetchCaptcha(ctx)
}

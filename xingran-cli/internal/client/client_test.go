package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
)

// ---------- mocks ----------

type mockEncryptor struct {
	fetchCalled int
	encCalls    int
	encErr      error
}

func (m *mockEncryptor) FetchPublicKey(ctx context.Context, baseURL string) error {
	m.fetchCalled++
	return nil
}

func (m *mockEncryptor) EncryptRequest(ctx context.Context, body []byte) ([]byte, error) {
	m.encCalls++
	if m.encErr != nil {
		return nil, m.encErr
	}
	return []byte("ENC:" + string(body)), nil
}

type mockAuthenticator struct {
	accessToken atomic.Value // string
	refreshN    int32
	loginN      int32
	refreshErr  error
}

func (m *mockAuthenticator) Login(ctx context.Context, username, password string, captcha auth.CaptchaInput) (*auth.TokenPair, error) {
	atomic.AddInt32(&m.loginN, 1)
	m.accessToken.Store("tok-1")
	return &auth.TokenPair{AccessToken: "tok-1", RefreshToken: "rt-1"}, nil
}

func (m *mockAuthenticator) Refresh(ctx context.Context) (*auth.TokenPair, error) {
	atomic.AddInt32(&m.refreshN, 1)
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	m.accessToken.Store("tok-2")
	return &auth.TokenPair{AccessToken: "tok-2", RefreshToken: "rt-2"}, nil
}

func (m *mockAuthenticator) GetAccessToken(ctx context.Context) (string, error) {
	if v, ok := m.accessToken.Load().(string); ok {
		return v, nil
	}
	return "", errors.New("未登录")
}

func (m *mockAuthenticator) Logout(ctx context.Context) error { return nil }

func wrap(code int, msg string, data any) string {
	d, _ := json.Marshal(data)
	b, _ := json.Marshal(map[string]any{
		"code": code, "message": msg,
		"data": json.RawMessage(d), "timestamp": 1700000000, "request_id": "req-1",
	})
	return string(b)
}

// ---------- Do 基本行为 ----------

func TestDoSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/user/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			t.Errorf("Authorization = %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["current"].(float64) != 1 {
			t.Errorf("body = %v", body)
		}
		_, _ = w.Write([]byte(wrap(0, "success", map[string]any{"list": []string{"a"}})))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{}
	auth.accessToken.Store("tok-1")
	c := New(Options{BaseURL: srv.URL, Authenticator: auth})
	resp, err := c.Do(context.Background(), "POST", "api/v1/system/user/list", map[string]any{"current": 1})
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if resp.Code != 0 || resp.RequestID != "req-1" {
		t.Fatalf("resp = %+v", resp)
	}
	if !strings.Contains(string(resp.Data), "a") {
		t.Fatalf("data = %s", resp.Data)
	}
}

func TestDoAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(wrap(400, "参数错误", nil)))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL, Authenticator: &mockAuthenticator{}})
	_, err := c.Do(context.Background(), "POST", "/x", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Resp.Code != 400 {
		t.Fatalf("err = %v", err)
	}
}

func TestDoTransportError(t *testing.T) {
	// 指向必然拒绝连接的地址。
	c := New(Options{BaseURL: "http://127.0.0.1:1", Timeout: 0, Authenticator: &mockAuthenticator{}})
	_, err := c.Do(context.Background(), "POST", "/x", nil)
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want TransportError", err)
	}
}

// ---------- 401/1007 自动 refresh 重试一次 ----------

func TestDo401RefreshRetryOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(wrap(401, "unauthorized", nil)))
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-2" {
			t.Errorf("retry Authorization = %q, want refreshed tok-2", got)
		}
		_, _ = w.Write([]byte(wrap(0, "success", "ok")))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{}
	auth.accessToken.Store("tok-1")
	c := New(Options{BaseURL: srv.URL, Authenticator: auth})
	resp, err := c.Do(context.Background(), "POST", "/x", nil)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("resp = %+v", resp)
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Fatalf("http calls = %d, want 2", n)
	}
	if n := atomic.LoadInt32(&auth.refreshN); n != 1 {
		t.Fatalf("refresh calls = %d, want 1", n)
	}
}

func TestDo1007RefreshRetryOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			_, _ = w.Write([]byte(wrap(1007, "token invalid", nil)))
			return
		}
		_, _ = w.Write([]byte(wrap(0, "success", "ok")))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{}
	auth.accessToken.Store("tok-1")
	c := New(Options{BaseURL: srv.URL, Authenticator: auth})
	if _, err := c.Do(context.Background(), "POST", "/x", nil); err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("http calls = %d, want 2", atomic.LoadInt32(&calls))
	}
}

func TestDo401TwiceNoLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(wrap(401, "unauthorized", nil)))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{}
	auth.accessToken.Store("tok-1")
	c := New(Options{BaseURL: srv.URL, Authenticator: auth})
	_, err := c.Do(context.Background(), "POST", "/x", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Resp.Code != 401 {
		t.Fatalf("err = %v", err)
	}
	if atomic.LoadInt32(&auth.refreshN) != 1 {
		t.Fatalf("refresh calls = %d, want exactly 1 (no loop)", atomic.LoadInt32(&auth.refreshN))
	}
}

func TestDo401WithoutAuthenticator(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(wrap(401, "unauthorized", nil)))
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL})
	_, err := c.Do(context.Background(), "POST", "/x", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v", err)
	}
}

// ---------- 加密接入 ----------

func TestDoWithEncryptor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := readAll(r)
		if !strings.HasPrefix(string(b), "ENC:") {
			t.Errorf("body not encrypted: %s", b)
		}
		_, _ = w.Write([]byte(wrap(0, "success", nil)))
	}))
	defer srv.Close()

	enc := &mockEncryptor{}
	c := New(Options{BaseURL: srv.URL, Encryptor: enc, Authenticator: &mockAuthenticator{}})
	if _, err := c.Do(context.Background(), "POST", "/x", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if enc.encCalls != 1 {
		t.Fatalf("encCalls = %d", enc.encCalls)
	}
}

func readAll(r *http.Request) ([]byte, error) {
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"":                        "/api/v1/",
		"/system/users":           "/api/v1/system/users",
		"system/users":            "/api/v1/system/users",
		"/api/v1/system/users":    "/api/v1/system/users",
		"api/v1/system/users":     "/api/v1/system/users",
		"/system/auth/public-key": "/api/v1/system/auth/public-key",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---------- dry-run ----------

func TestDryRunNoRequest(t *testing.T) {
	enc := &mockEncryptor{}
	c := New(Options{BaseURL: "http://unused.example", Encryptor: enc, Authenticator: &mockAuthenticator{}})
	dr, err := c.DryRun(context.Background(), "post", "api/v1/system/user/list", map[string]any{"a": 1})
	if err != nil {
		t.Fatalf("DryRun err: %v", err)
	}
	// dry-run 完全离线：不拉公钥、不加密，body 为原始明文。
	if dr.Method != "POST" || dr.Encrypted {
		t.Fatalf("dr = %+v", dr)
	}
	if !strings.Contains(dr.Body, `"a":1`) {
		t.Fatalf("body = %q", dr.Body)
	}
	if !strings.Contains(dr.URL, "/api/v1/system/user/list") {
		t.Fatalf("url = %q", dr.URL)
	}
	if enc.encCalls != 0 || enc.fetchCalled != 0 {
		t.Fatalf("dry-run 不应触达加密器: encCalls=%d fetchCalled=%d", enc.encCalls, enc.fetchCalled)
	}
}

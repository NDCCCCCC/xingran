package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestWebServer 构造注入 mockRequester 的一次性登录服务（不经真实 listen）。
func newTestWebServer(req *mockRequester) (*webLoginServer, *httptest.Server) {
	store := &FileTokenStore{Path: tempStorePath()}
	a := NewAuthenticator(req, store, nil, "http://test.local")
	w := newWebLoginServer(a, a.(CaptchaProvider), renderLoginPage("preset-user"))
	ts := httptest.NewServer(w.routes())
	return w, ts
}

func webGet(ts *httptest.Server, path, token string) (*http.Response, []byte) {
	resp, err := http.Get(ts.URL + path + "s=" + token)
	if err != nil {
		panic(err)
	}
	body := make([]byte, 1<<16)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	return resp, body[:n]
}

func webPostJSON(ts *httptest.Server, path, token string, body any) (int, map[string]any) {
	raw, _ := json.Marshal(body)
	resp, err := http.Post(ts.URL+path+"s="+token, "application/json", strings.NewReader(string(raw)))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestWebLoginSuccess(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())

	w, ts := newTestWebServer(req)
	defer ts.Close()

	// 页面可访问（session token 正确）
	resp, body := webGet(ts, "/?", w.token)
	if resp.StatusCode != 200 || !strings.Contains(string(body), "XingRan CLI 登录") {
		t.Fatalf("页面应可访问: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"preset-user"`) {
		t.Fatal("页面应包含预填用户名")
	}

	// disabled 模式（FetchCaptcha 未注册 → mock 报错走不到，直接注册空 data）
	req.routes["/system/auth/captcha"] = req.ok(map[string]any{})
	code, out := webPostJSON(ts, "/login?", w.token, map[string]any{
		"username": "admin", "password": "pass123",
	})
	if code != 200 || out["ok"] != true {
		t.Fatalf("登录应成功: %d %v", code, out)
	}

	select {
	case pair := <-w.done:
		if pair.AccessToken != "access-1" {
			t.Fatalf("done 通道 token 不正确: %+v", pair)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("done 通道未收到 TokenPair")
	}
}

func TestWebLoginSliderFlow(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.ok(CaptchaInfo{
		CaptchaID: "cap-w1", CaptchaType: "slider", Token: "tk-w1",
		SliderImg: "data:image/png;base64,AA", PieceImg: "data:image/png;base64,BB", YPos: 39,
	})
	req.routes["/system/auth/captcha/verify/slider"] = req.ok(map[string]any{"success": true})
	req.routes[loginPath] = req.ok(loginResponseData())

	w, ts := newTestWebServer(req)
	defer ts.Close()

	// 验证码端点
	resp, body := webGet(ts, "/captcha?", w.token)
	if resp.StatusCode != 200 {
		t.Fatalf("captcha 端点失败: %d", resp.StatusCode)
	}
	var payload captchaPayload
	_ = json.Unmarshal(body, &payload)
	if payload.Mode != CaptchaTypeSlider || payload.CaptchaID != "cap-w1" || payload.YPos != 39 {
		t.Fatalf("captcha 响应不正确: %+v", payload)
	}

	// 登录（服务端编排 verify+login）
	code, out := webPostJSON(ts, "/login?", w.token, map[string]any{
		"username": "admin", "password": "pw", "sliderX": 227,
	})
	if code != 200 || out["ok"] != true {
		t.Fatalf("slider 登录应成功: %d %v", code, out)
	}
	verifyBody := req.lastBody("/system/auth/captcha/verify/slider")
	if verifyBody["captchaId"] != "cap-w1" || verifyBody["xPos"] != float64(227) {
		t.Fatalf("verify 请求体不正确: %+v", verifyBody)
	}
}

func TestWebLoginCaptchaRetry(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.ok(map[string]any{})
	// 第一次登录失败（验证码错误），第二次成功
	req.routes[loginPath] = req.err(460, "验证码错误")

	w, ts := newTestWebServer(req)
	defer ts.Close()

	code, out := webPostJSON(ts, "/login?", w.token, map[string]any{
		"username": "admin", "password": "pw",
	})
	if code != 401 || !strings.Contains(out["error"].(string), "验证码错误") {
		t.Fatalf("第一次登录应失败并回显错误: %d %v", code, out)
	}
	select {
	case <-w.done:
		t.Fatal("失败不应触发 done")
	default:
	}

	// 服务端改为成功 → 重试成功
	req.routes[loginPath] = req.ok(loginResponseData())
	code, out = webPostJSON(ts, "/login?", w.token, map[string]any{
		"username": "admin", "password": "pw",
	})
	if code != 200 || out["ok"] != true {
		t.Fatalf("重试应成功: %d %v", code, out)
	}
	select {
	case pair := <-w.done:
		if pair.AccessToken == "" {
			t.Fatal("done 应携带 token")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("重试后 done 未触发")
	}
}

func TestWebBadSessionToken(t *testing.T) {
	req := newMockRequester()
	_, ts := newTestWebServer(req)
	defer ts.Close()

	resp, _ := webGet(ts, "/?", "wrong-token")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("错误 token 访问页面应 403: %d", resp.StatusCode)
	}
	resp, _ = webGet(ts, "/captcha?", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("缺 token 访问 captcha 应 403: %d", resp.StatusCode)
	}
	code, _ := webPostJSON(ts, "/login?", "wrong-token", map[string]any{"username": "a", "password": "b"})
	if code != http.StatusForbidden {
		t.Fatalf("错误 token 登录应 403: %d", code)
	}
}

func TestWebLoginMissingFields(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.ok(map[string]any{})
	w, ts := newTestWebServer(req)
	defer ts.Close()

	code, out := webPostJSON(ts, "/login?", w.token, map[string]any{"username": "admin"})
	if code != 400 || !strings.Contains(out["error"].(string), "密码") {
		t.Fatalf("缺密码应 400: %d %v", code, out)
	}
}

func TestInteractiveLoginTimeout(t *testing.T) {
	// 真实 listen（127.0.0.1 随机端口），极短超时 → 返回超时错误
	opts := WebLoginOptions{
		BaseURL:     "http://127.0.0.1:1", // 不可达（不会真请求）
		Timeout:     150 * time.Millisecond,
		OpenBrowser: false,
		Store:       &FileTokenStore{Path: tempStorePath()},
	}
	_, err := InteractiveLogin(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("应返回超时错误: %v", err)
	}
}

func TestInteractiveLoginRequiresBaseURL(t *testing.T) {
	if _, err := InteractiveLogin(context.Background(), WebLoginOptions{}); err == nil {
		t.Fatal("缺 BaseURL 应报错")
	}
}

func TestInteractiveLoginCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	opts := WebLoginOptions{
		BaseURL: "http://127.0.0.1:1", Timeout: time.Minute, OpenBrowser: false,
		Store: &FileTokenStore{Path: tempStorePath()},
	}
	_, err := InteractiveLogin(ctx, opts)
	if err == nil || !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "取消") {
		t.Fatalf("应返回取消错误: %v", err)
	}
}

func TestRenderLoginPageEscapesUsername(t *testing.T) {
	page := renderLoginPage(`"><script>alert(1)</script>`)
	if strings.Contains(page, "<script>alert(1)</script>") {
		t.Fatal("预填用户名必须 JSON 转义防注入")
	}
}

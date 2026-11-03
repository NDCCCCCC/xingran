package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const loginPath = "/system/auth/login"

func loginResponseData() loginResponse {
	return loginResponse{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresIn:    300,
		TokenType:    "Bearer",
	}
}

// --- disabled 直通 ---

func TestLoginDisabledDirect(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)
	pair, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeNone})
	if err != nil {
		t.Fatalf("disabled 模式登录失败: %v", err)
	}
	if pair.AccessToken != "access-1" {
		t.Fatalf("token 不正确: %+v", pair)
	}

	body := req.lastBody(loginPath)
	if body["username"] != "admin" || body["password"] != "pass123" {
		t.Fatalf("登录请求体不正确: %+v", body)
	}
	if _, has := body["captchaId"]; has {
		t.Fatalf("disabled 模式不应携带 captchaId: %+v", body)
	}
	if _, has := body["captcha"]; has {
		t.Fatalf("disabled 模式不应携带 captcha: %+v", body)
	}
}

// --- normal 模式 ---

func TestLoginNormalRequiresFetchFirst(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)
	_, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeNormal, Code: "1234"})
	if err == nil || !strings.Contains(err.Error(), "FetchCaptcha") {
		t.Fatalf("未先 FetchCaptcha 应报错: %v", err)
	}
}

func TestLoginNormalFullFlow(t *testing.T) {
	req := newMockRequester()
	captchaData := CaptchaInfo{
		CaptchaID:   "cap-1",
		CaptchaType: "normal",
		CaptchaImg:  "data:image/png;base64,iVBORw0KGgo=",
	}
	req.routes["/system/auth/captcha"] = req.ok(captchaData)
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)

	info, err := fetchCaptcha(a, context.Background())
	if err != nil {
		t.Fatalf("FetchCaptcha 失败: %v", err)
	}
	if info.CaptchaID != "cap-1" || info.CaptchaType != "normal" {
		t.Fatalf("验证码信息不正确: %+v", info)
	}

	pair, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeNormal, Code: "Ab12"})
	if err != nil {
		t.Fatalf("normal 模式登录失败: %v", err)
	}
	if pair.AccessToken != "access-1" {
		t.Fatalf("token 不正确: %+v", pair)
	}

	body := req.lastBody(loginPath)
	if body["captchaId"] != "cap-1" {
		t.Fatalf("captchaId 不正确: %+v", body)
	}
	if body["captcha"] != "Ab12" {
		t.Fatalf("captcha 字符不正确: %+v", body)
	}

	// pending 一次性消费
	if _, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeNormal, Code: "Ab12"}); err == nil {
		t.Fatal("第二次登录应要求重新 FetchCaptcha")
	}
}

// --- slider 模式 ---

func TestLoginSliderFullFlow(t *testing.T) {
	req := newMockRequester()
	captchaData := CaptchaInfo{
		CaptchaID:   "cap-s1",
		CaptchaType: "slider",
		SliderImg:   "data:image/png;base64,iVBORw0KGgo=",
		PieceImg:    "data:image/png;base64,iVBORw0KGgo=",
		YPos:        42,
		Token:       "slide-token-9",
	}
	req.routes["/system/auth/captcha"] = req.ok(captchaData)
	req.routes["/system/auth/captcha/verify/slider"] = req.ok(map[string]any{"success": true})
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)

	if _, err := fetchCaptcha(a, context.Background()); err != nil {
		t.Fatalf("FetchCaptcha 失败: %v", err)
	}

	pair, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeSlider, SliderX: 187})
	if err != nil {
		t.Fatalf("slider 模式登录失败: %v", err)
	}
	if pair.AccessToken != "access-1" {
		t.Fatalf("token 不正确: %+v", pair)
	}

	// 核对 verify/slider 请求体
	verifyBody := req.lastBody("/system/auth/captcha/verify/slider")
	if verifyBody["captchaId"] != "cap-s1" || verifyBody["xPos"] != float64(187) || verifyBody["token"] != "slide-token-9" {
		t.Fatalf("verify/slider 请求体不正确: %+v", verifyBody)
	}

	// 登录请求应携带 verified 标记（后端 VerifyCaptcha slider 分支约定）
	body := req.lastBody(loginPath)
	if body["captchaId"] != "cap-s1" || body["captcha"] != "verified" {
		t.Fatalf("登录请求应携带 verified 标记: %+v", body)
	}
}

func TestLoginSliderVerifyFailed(t *testing.T) {
	req := newMockRequester()
	captchaData := CaptchaInfo{CaptchaID: "cap-s2", CaptchaType: "slider", Token: "tk"}
	req.routes["/system/auth/captcha"] = req.ok(captchaData)
	req.routes["/system/auth/captcha/verify/slider"] = req.err(460, "验证失败，位置不正确")

	a, _ := newTestAuthenticator(req, nil)
	_, _ = fetchCaptcha(a, context.Background())

	_, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{Type: CaptchaTypeSlider, SliderX: 10})
	if err == nil {
		t.Fatal("滑动验证失败应返回错误")
	}
	// APIError 应通过错误包装链可提取，message 原样透传
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "验证失败，位置不正确" {
		t.Fatalf("错误应透传服务端 message: %v", err)
	}
	// 失败后不应调用登录接口
	for _, c := range req.calls {
		if c == loginPath {
			t.Fatal("滑动验证失败后不应发起登录请求")
		}
	}
}

func TestLoginSliderMissingFetchOrX(t *testing.T) {
	req := newMockRequester()
	a, _ := newTestAuthenticator(req, nil)

	_, err := a.Login(context.Background(), "admin", "pass", CaptchaInput{Type: CaptchaTypeSlider, SliderX: 10})
	if err == nil || !strings.Contains(err.Error(), "FetchCaptcha") {
		t.Fatalf("未先 FetchCaptcha 应报错: %v", err)
	}

	req.routes["/system/auth/captcha"] = req.ok(CaptchaInfo{CaptchaID: "c", CaptchaType: "slider", Token: "t"})
	_, _ = fetchCaptcha(a, context.Background())
	_, err = a.Login(context.Background(), "admin", "pass", CaptchaInput{Type: CaptchaTypeSlider})
	if err == nil || !strings.Contains(err.Error(), "SliderX") {
		t.Fatalf("缺 SliderX 应报错: %v", err)
	}
}

// --- 跨进程两步登录：显式注入 CaptchaID/Token ---

// slider 显式注入：全新 authenticator 实例（无 pendingCaptcha），模拟进程B 回填
func TestLoginSliderExplicitCaptchaIDToken(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha/verify/slider"] = req.ok(map[string]any{"success": true})
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil) // 不调用 FetchCaptcha，无进程内缓存

	pair, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{
		Type:      CaptchaTypeSlider,
		SliderX:   227,
		CaptchaID: "cap-explicit-1",
		Token:     "explicit-token-7",
	})
	if err != nil {
		t.Fatalf("显式注入 slider 登录失败: %v", err)
	}
	if pair.AccessToken != "access-1" {
		t.Fatalf("token 不正确: %+v", pair)
	}

	// verify/slider 应使用显式注入的 captchaId/token
	verifyBody := req.lastBody("/system/auth/captcha/verify/slider")
	if verifyBody["captchaId"] != "cap-explicit-1" || verifyBody["token"] != "explicit-token-7" || verifyBody["xPos"] != float64(227) {
		t.Fatalf("verify 请求应使用显式注入凭据: %+v", verifyBody)
	}
	// 登录请求应携带同一 captchaId + verified 标记
	body := req.lastBody(loginPath)
	if body["captchaId"] != "cap-explicit-1" || body["captcha"] != "verified" {
		t.Fatalf("登录请求应携带显式 captchaId + verified: %+v", body)
	}
}

// slider 显式注入但缺 Token 且无 pending：应报错并提示显式注入
func TestLoginSliderExplicitMissingToken(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)
	_, err := a.Login(context.Background(), "admin", "pass", CaptchaInput{
		Type: CaptchaTypeSlider, SliderX: 100, CaptchaID: "cap-x",
	})
	if err == nil || !strings.Contains(err.Error(), "CaptchaID+Token") {
		t.Fatalf("显式注入缺 Token 应报错: %v", err)
	}
	for _, c := range req.calls {
		if c == loginPath || c == "/system/auth/captcha/verify/slider" {
			t.Fatal("凭据不完整时不应发起任何请求")
		}
	}
}

// slider 混合场景：pending 存在但显式只给 CaptchaID → CaptchaID 用显式值，Token 回退 pending
func TestLoginSliderMixedExplicitAndPending(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.ok(CaptchaInfo{
		CaptchaID: "cap-pending-1", CaptchaType: "slider", Token: "pending-token",
	})
	req.routes["/system/auth/captcha/verify/slider"] = req.ok(map[string]any{"success": true})
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil)
	_, _ = fetchCaptcha(a, context.Background()) // pending: cap-pending-1/pending-token

	if _, err := a.Login(context.Background(), "admin", "pass", CaptchaInput{
		Type: CaptchaTypeSlider, SliderX: 50, CaptchaID: "cap-explicit-2",
	}); err != nil {
		t.Fatalf("混合模式登录失败: %v", err)
	}
	verifyBody := req.lastBody("/system/auth/captcha/verify/slider")
	if verifyBody["captchaId"] != "cap-explicit-2" || verifyBody["token"] != "pending-token" {
		t.Fatalf("显式 CaptchaID 应优先、Token 回退 pending: %+v", verifyBody)
	}
}

// normal 显式注入 CaptchaID：跨进程两步登录（无 pending）
func TestLoginNormalExplicitCaptchaID(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())

	a, _ := newTestAuthenticator(req, nil) // 无 pending
	pair, err := a.Login(context.Background(), "admin", "pass123", CaptchaInput{
		Type: CaptchaTypeNormal, Code: "Ab12", CaptchaID: "cap-explicit-3",
	})
	if err != nil {
		t.Fatalf("显式注入 normal 登录失败: %v", err)
	}
	if pair.AccessToken != "access-1" {
		t.Fatalf("token 不正确: %+v", pair)
	}
	body := req.lastBody(loginPath)
	if body["captchaId"] != "cap-explicit-3" || body["captcha"] != "Ab12" {
		t.Fatalf("登录请求应携带显式 captchaId: %+v", body)
	}
}

// --- 锁定提示透传 ---

func TestLoginLockMessagePassthrough(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.err(403, "账号已锁定，请 25 分钟后再试")

	a, _ := newTestAuthenticator(req, nil)
	_, err := a.Login(context.Background(), "lockeduser", "pass", CaptchaInput{Type: CaptchaTypeNone})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("期望 APIError, got %T: %v", err, err)
	}
	if apiErr.Code != 403 || apiErr.Message != "账号已锁定，请 25 分钟后再试" {
		t.Fatalf("锁定提示应原样透传: %+v", apiErr)
	}
}

// --- 未知模式 ---

func TestLoginUnknownType(t *testing.T) {
	req := newMockRequester()
	a, _ := newTestAuthenticator(req, nil)
	_, err := a.Login(context.Background(), "admin", "pass", CaptchaInput{Type: "weird"})
	if err == nil || !strings.Contains(err.Error(), "未知验证码模式") {
		t.Fatalf("未知模式应报错: %v", err)
	}
}

// --- 刷新与自动刷新 ---

func TestRefreshFlow(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/refresh"] = req.ok(loginResponse{
		AccessToken: "access-2", RefreshToken: "refresh-2", ExpiresIn: 300, TokenType: "Bearer",
	})

	a, store := newTestAuthenticator(req, nil)
	// 预置旧 token
	if err := store.Save(nil, TokenPair{AccessToken: "old", RefreshToken: "old-rt", ExpiresIn: 3600}); err != nil {
		t.Fatal(err)
	}

	pair, err := a.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	if pair.AccessToken != "access-2" || pair.RefreshToken != "refresh-2" {
		t.Fatalf("刷新结果不正确: %+v", pair)
	}
	body := req.lastBody("/system/auth/refresh")
	if body["refreshToken"] != "old-rt" {
		t.Fatalf("刷新请求体不正确: %+v", body)
	}
}

func TestRefreshWithoutLogin(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/refresh"] = req.ok(loginResponseData())
	a, _ := newTestAuthenticator(req, nil)
	_, err := a.Refresh(context.Background())
	if err == nil || !strings.Contains(err.Error(), "先登录") {
		t.Fatalf("未登录刷新应报错: %v", err)
	}
}

func TestGetAccessTokenAutoRefresh(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/refresh"] = req.ok(loginResponse{
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 600, TokenType: "Bearer",
	})

	a, store := newTestAuthenticator(req, nil)

	// 1. 有效 token：直接返回，不触发刷新
	if err := store.Save(nil, TokenPair{AccessToken: "valid", RefreshToken: "rt", ExpiresIn: 3600}); err != nil {
		t.Fatal(err)
	}
	tok, err := a.GetAccessToken(context.Background())
	if err != nil || tok != "valid" {
		t.Fatalf("有效 token 应直接返回: %q, %v", tok, err)
	}
	if len(req.calls) != 0 {
		t.Fatalf("不应发起刷新: %v", req.calls)
	}

	// 2. 即将过期（<30s）：自动刷新
	expired := TokenPair{AccessToken: "stale", RefreshToken: "old-rt", ExpiresIn: 3600, ExpiresAt: timeNow().Add(5 * time.Second)}
	if err := store.Save(nil, expired); err != nil {
		t.Fatal(err)
	}
	tok, err = a.GetAccessToken(context.Background())
	if err != nil || tok != "new-access" {
		t.Fatalf("过期应自动刷新: %q, %v", tok, err)
	}
}

func TestGetAccessTokenNoLogin(t *testing.T) {
	req := newMockRequester()
	a, _ := newTestAuthenticator(req, nil)
	_, err := a.GetAccessToken(context.Background())
	if err == nil || !strings.Contains(err.Error(), "未登录") {
		t.Fatalf("未登录应报错: %v", err)
	}
}

// --- 登出 ---

func TestLogout(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/logout"] = req.ok(map[string]any{"message": "登出成功"})

	a, store := newTestAuthenticator(req, nil)
	if err := store.Save(nil, TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 300}); err != nil {
		t.Fatal(err)
	}

	if err := a.Logout(context.Background()); err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}
	if req.tokens["/system/auth/logout"] != "at" {
		t.Fatalf("登出应携带 access token: %q", req.tokens["/system/auth/logout"])
	}
	loaded, _ := store.Load(nil)
	if loaded != nil {
		t.Fatal("登出后本地存储应清除")
	}
}

func TestLogoutNotLoggedIn(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/logout"] = req.ok(map[string]any{})
	a, _ := newTestAuthenticator(req, nil)
	if err := a.Logout(context.Background()); err != nil {
		t.Fatalf("未登录登出应静默成功: %v", err)
	}
}

// --- 加密接入 ---

// 加密逻辑归属 HTTPRequester（authenticator 不直接调用 EncryptRequest），
// 故用 httptest 验证 HTTPRequester 的真实加密链路。
func TestHTTPRequesterEncryptsBody(t *testing.T) {
	var gotBody []byte
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{},"request_id":"r1"}`))
	}))
	defer srv.Close()

	enc := &mockEncryptor{}
	req := &HTTPRequester{BaseURL: srv.URL, Encryptor: enc, EncryptionMode: EncryptionModeOn}
	resp, err := req.Post(context.Background(), "/system/auth/login", map[string]any{"username": "admin"}, "tok-1")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("响应不正确: %+v", resp)
	}
	if enc.encCalls == 0 {
		t.Fatal("请求体应经过 EncryptRequest")
	}
	if string(gotBody) != "ENC:"+`{"username":"admin"}` {
		t.Fatalf("加密后的请求体不正确: %q", gotBody)
	}
	if gotAuth != "Bearer tok-1" {
		t.Fatalf("Authorization 头不正确: %q", gotAuth)
	}
}

// auto 模式：探测服务端 encryption-config 决定是否加密。
func TestHTTPRequesterAutoEncryptionProbe(t *testing.T) {
	newSrv := func(enabled bool, gotBody *[]byte) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/v1/system/auth/encryption-config":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"code":0,"message":"success","data":{"enabled":%t}}`, enabled)))
			default:
				b, _ := io.ReadAll(r.Body)
				*gotBody = b
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{}}`))
			}
		}))
	}

	// enabled=false → 明文直发
	var body []byte
	srv := newSrv(false, &body)
	defer srv.Close()
	req := &HTTPRequester{BaseURL: srv.URL, Encryptor: &mockEncryptor{}, EncryptionMode: EncryptionModeAuto}
	if _, err := req.Post(context.Background(), "/system/auth/login", map[string]any{"u": 1}, ""); err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if string(body) != `{"u":1}` {
		t.Fatalf("服务端加密关闭时应明文直发: %q", body)
	}

	// enabled=true → 加密
	srv2 := newSrv(true, &body)
	defer srv2.Close()
	req2 := &HTTPRequester{BaseURL: srv2.URL, Encryptor: &mockEncryptor{}, EncryptionMode: EncryptionModeAuto}
	if _, err := req2.Post(context.Background(), "/system/auth/login", map[string]any{"u": 1}, ""); err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	if !strings.HasPrefix(string(body), "ENC:") {
		t.Fatalf("服务端加密开启时应加密: %q", body)
	}
}

func TestLoginEncryptorFetchFails(t *testing.T) {
	req := newMockRequester()
	req.routes[loginPath] = req.ok(loginResponseData())
	enc := &mockEncryptor{failFetch: true}

	a, _ := newTestAuthenticator(req, enc)
	_, err := a.Login(context.Background(), "admin", "pass", CaptchaInput{Type: CaptchaTypeNone})
	if err == nil || !strings.Contains(err.Error(), "公钥") {
		t.Fatalf("公钥失败应中止登录: %v", err)
	}
	if len(req.calls) != 0 {
		t.Fatal("公钥失败不应发起任何请求")
	}
}

// --- FetchCaptcha 边界 ---

func TestFetchCaptchaDisabled(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.ok(map[string]any{}) // disabled: 空 data

	a, _ := newTestAuthenticator(req, nil)
	info, err := fetchCaptcha(a, context.Background())
	if err != nil {
		t.Fatalf("disabled 模式 FetchCaptcha 不应报错: %v", err)
	}
	if info != nil {
		t.Fatalf("disabled 模式应返回 nil: %+v", info)
	}
}

func TestFetchCaptchaServerError(t *testing.T) {
	req := newMockRequester()
	req.routes["/system/auth/captcha"] = req.err(429, "服务繁忙，请稍后再试")

	a, _ := newTestAuthenticator(req, nil)
	_, err := fetchCaptcha(a, context.Background())
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Message != "服务繁忙，请稍后再试" {
		t.Fatalf("限流提示应透传: %v", err)
	}
}

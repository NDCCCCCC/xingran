//go:build integration

// Package tests XingRan CLI 端到端集成测试（M2/M3）。
//
// 需要真实后端运行（go build 后运行 xingran-backend.exe）。
// 运行方式：
//
//	go test -tags=integration ./tests/
//
// 环境变量：
//
//	XINGRAN_BASE_URL      后端地址，默认 http://localhost:9000
//	XINGRAN_TEST_USER     测试账号（必填才跑登录用例；避免锁号请用专用测试账号）
//	XINGRAN_TEST_PASS     测试密码
//	XINGRAN_CAPTCHA_ANSWER normal 验证码识别字符（多模态 agent / 人工注入后跑完整登录）
//	XINGRAN_SLIDER_X      slider 缺口 x 坐标（±8px 容差）
//
// 安全约束：失败路径用例每环境只执行一次，不触发 loginMaxRetry(5) 锁号。
package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/internal/crypto"
)

const (
	captchaConfigPath = "/system/auth/captcha/config"
	captchaFetchPath  = "/system/auth/captcha"
	loginPath         = "/system/auth/login"
	usersListPath     = "/system/users/list"
)

func baseURL() string {
	if v := os.Getenv("XINGRAN_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:9000"
}

// newTestAuth 构造接真实后端的 Authenticator + CaptchaProvider（crypto 真实实现）。
func newTestAuth(t *testing.T) (auth.Authenticator, auth.CaptchaProvider) {
	t.Helper()
	enc := crypto.NewEncryptor()
	req := &auth.HTTPRequester{BaseURL: baseURL(), Encryptor: enc, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
	store := &auth.FileTokenStore{Path: filepath.Join(t.TempDir(), "tokens.json")}
	a := auth.NewAuthenticator(req, store, enc, baseURL())
	return a, a.(auth.CaptchaProvider)
}

// captchaMode 探测后端验证码模式："disabled" | "normal" | "slider"；后端不可达返回 ""。
func captchaMode(t *testing.T) string {
	t.Helper()
	_, cp := newTestAuth(t)
	_ = cp
	// captcha/config 是公开端点，走无加密直连即可；用 HTTPRequester 不带 encryptor 更简单
	plain := &auth.HTTPRequester{BaseURL: baseURL(), HTTPClient: &http.Client{Timeout: 10 * time.Second}}
	resp, err := plain.Post(context.Background(), captchaConfigPath, map[string]any{}, "")
	if err != nil {
		t.Skipf("后端不可达(%s): %v —— 跳过集成测试", baseURL(), err)
	}
	if resp.Code != 0 {
		t.Fatalf("读取验证码配置失败: %s", resp.Message)
	}
	var cfg struct {
		Enabled string `json:"enabled"`
	}
	if err := json.Unmarshal(resp.Data, &cfg); err != nil {
		t.Fatalf("解析验证码配置失败: %v", err)
	}
	return cfg.Enabled
}

func testCreds() (user, pass string, ok bool) {
	user = os.Getenv("XINGRAN_TEST_USER")
	pass = os.Getenv("XINGRAN_TEST_PASS")
	return user, pass, user != "" && pass != ""
}

// --- 探测与验证码获取（不消耗登录重试） ---

func TestCaptchaConfigProbe(t *testing.T) {
	mode := captchaMode(t)
	t.Logf("后端验证码模式: %q", mode)
	switch mode {
	case "disabled", "normal", "slider":
	default:
		t.Fatalf("未知模式 %q", mode)
	}
}

func TestFetchCaptchaStructure(t *testing.T) {
	mode := captchaMode(t)
	_, cp := newTestAuth(t)

	info, err := cp.FetchCaptcha(context.Background())
	if mode == "disabled" {
		if err != nil || info != nil {
			t.Fatalf("disabled 模式应返回 (nil,nil): %+v, %v", info, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("FetchCaptcha 失败: %v", err)
	}
	if info.CaptchaID == "" {
		t.Fatal("captchaId 为空")
	}
	switch mode {
	case "normal":
		if info.CaptchaType != "normal" || info.CaptchaImg == "" {
			t.Fatalf("normal 响应不完整: %+v", info)
		}
		// 渲染链路可用性：ASCII 与文件双通道
		ascii, err := auth.RenderTerminal(info.CaptchaImg, 80)
		if err != nil || !strings.Contains(ascii, "\n") {
			t.Fatalf("ASCII 渲染失败: %v", err)
		}
		imgPath := filepath.Join(t.TempDir(), "captcha.png")
		if err := auth.SaveImage(info.CaptchaImg, imgPath); err != nil {
			t.Fatalf("保存验证码图片失败: %v", err)
		}
		t.Logf("验证码图片已保存: %s（多模态 agent 可直接读取 dataURL 或该文件识别）", imgPath)
	case "slider":
		if info.CaptchaType != "slider" || info.SliderImg == "" || info.PieceImg == "" || info.Token == "" {
			t.Fatalf("slider 响应不完整: %+v", info)
		}
		// 保存到固定路径（系统临时目录，非 t.TempDir），供两阶段工作流：
		// 阶段1（本用例）拉验证码存图+状态 → 多模态 agent 观察缺口 x → 阶段2 回填验证
		manualDir := manualCaptchaDir()
		_ = os.MkdirAll(manualDir, 0o700)
		bgPath := filepath.Join(manualDir, "slider_bg.png")
		if err := auth.SaveImage(info.SliderImg, bgPath); err != nil {
			t.Fatalf("保存底图失败: %v", err)
		}
		state := map[string]any{
			"captchaId": info.CaptchaID,
			"token":     info.Token,
			"yPos":      info.YPos,
			"bgImage":   bgPath,
			"fetchedAt": time.Now().Format(time.RFC3339),
		}
		stateRaw, _ := json.Marshal(state)
		statePath := filepath.Join(manualDir, "state.json")
		if err := os.WriteFile(statePath, stateRaw, 0o600); err != nil {
			t.Fatalf("保存验证码状态失败: %v", err)
		}
		t.Logf("拼图底图: %s（缺口观察目标）, Y=%d, 状态文件: %s", bgPath, info.YPos, statePath)
	}
}

// manualCaptchaDir 两阶段 slider 验证工作流的共享目录。
func manualCaptchaDir() string {
	return filepath.Join(os.TempDir(), "xingran-captcha-manual")
}

// TestVerifySliderManual 阶段2：读取状态文件 + 注入缺口 x，走真实 verify/slider。
// 运行前先跑 TestFetchCaptchaStructure（阶段1），观察 slider_bg.png 缺口得到 x，
// 然后 XINGRAN_SLIDER_X=<x> go test -tags=integration -run TestVerifySliderManual ./tests/
// 注意：验证码后端 TTL 5 分钟，两阶段需在有效期内完成；本用例不碰登录接口，不会锁号。
func TestVerifySliderManual(t *testing.T) {
	xStr := os.Getenv("XINGRAN_SLIDER_X")
	if xStr == "" {
		t.Skip("未设置 XINGRAN_SLIDER_X（多模态观察 slider_bg.png 缺口后注入）")
	}
	var x int
	if _, err := fmt.Sscanf(xStr, "%d", &x); err != nil || x <= 0 {
		t.Fatalf("XINGRAN_SLIDER_X 非法: %q", xStr)
	}

	stateRaw, err := os.ReadFile(filepath.Join(manualCaptchaDir(), "state.json"))
	if err != nil {
		t.Fatalf("读取阶段1状态文件失败（先跑 TestFetchCaptchaStructure）: %v", err)
	}
	var state struct {
		CaptchaID string `json:"captchaId"`
		Token     string `json:"token"`
	}
	if err := json.Unmarshal(stateRaw, &state); err != nil || state.CaptchaID == "" {
		t.Fatalf("状态文件解析失败: %v", err)
	}

	a, _ := newTestAuth(t)
	if err := a.(auth.CaptchaProvider).VerifySlider(context.Background(), state.CaptchaID, x, state.Token); err != nil {
		var apiErr *auth.APIError
		if errors.As(err, &apiErr) {
			t.Fatalf("滑动验证未通过（服务端原文透传）: %s", apiErr.Message)
		}
		t.Fatalf("滑动验证请求失败: %v", err)
	}
	t.Logf("滑动验证通过: captchaId=%s x=%d", state.CaptchaID, x)
}

// --- 登录端到端（需要 XINGRAN_TEST_USER/PASS） ---

func TestLoginE2E(t *testing.T) {
	mode := captchaMode(t)
	user, pass, ok := testCreds()
	if !ok {
		t.Skip("未设置 XINGRAN_TEST_USER / XINGRAN_TEST_PASS")
	}

	a, cp := newTestAuth(t)
	ctx := context.Background()

	input := auth.CaptchaInput{Type: auth.CaptchaTypeNone}
	switch mode {
	case "disabled":
		// 直通
	case "normal":
		answer := os.Getenv("XINGRAN_CAPTCHA_ANSWER")
		if answer == "" {
			t.Skip("验证码为 normal 模式但未设置 XINGRAN_CAPTCHA_ANSWER（多模态识别结果），跳过完整登录")
		}
		if _, err := cp.FetchCaptcha(ctx); err != nil {
			t.Fatalf("FetchCaptcha 失败: %v", err)
		}
		input = auth.CaptchaInput{Type: auth.CaptchaTypeNormal, Code: answer}
	case "slider":
		x := 0
		if v := os.Getenv("XINGRAN_SLIDER_X"); v != "" {
			fmt.Sscanf(v, "%d", &x)
		}
		if x <= 0 {
			t.Skip("验证码为 slider 模式但未设置 XINGRAN_SLIDER_X（缺口 x 坐标），跳过完整登录")
		}
		if _, err := cp.FetchCaptcha(ctx); err != nil {
			t.Fatalf("FetchCaptcha 失败: %v", err)
		}
		input = auth.CaptchaInput{Type: auth.CaptchaTypeSlider, SliderX: x}
	}

	pair, err := a.Login(ctx, user, pass, input)
	if err != nil {
		t.Fatalf("登录失败: %v", err) // message 已透传服务端原文（含锁定提示）
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("登录响应缺少 token: %+v", pair)
	}
	t.Logf("登录成功, token 长度 access=%d refresh=%d, expiresIn=%ds", len(pair.AccessToken), len(pair.RefreshToken), pair.ExpiresIn)

	// token 已落盘且权限 0600（POSIX）
	pair2, err := a.GetAccessToken(ctx)
	if err != nil || pair2 == "" {
		t.Fatalf("GetAccessToken 失败: %v", err)
	}

	// 登出清理，不留会话
	if err := a.Logout(ctx); err != nil {
		t.Fatalf("登出失败: %v", err)
	}
}

// --- 失败路径（只执行一次，防锁号） ---

func TestLoginWrongPasswordOnce(t *testing.T) {
	mode := captchaMode(t)
	user, _, ok := testCreds()
	if !ok || mode != "disabled" {
		t.Skip("仅在后端 disabled 模式且提供测试账号时执行，避免验证码/锁号干扰")
	}

	a, _ := newTestAuth(t)
	_, err := a.Login(context.Background(), user, "definitely-wrong-password-<RETRY-GUARD>", auth.CaptchaInput{Type: auth.CaptchaTypeNone})
	if err == nil {
		t.Fatal("错误密码应登录失败")
	}
	var apiErr *auth.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("期望 APIError, got %T: %v", err, err)
	}
	t.Logf("服务端错误已透传: code=%d message=%q", apiErr.Code, apiErr.Message)
}

// --- 刷新链路 ---

func TestRefreshFlow(t *testing.T) {
	mode := captchaMode(t)
	user, pass, ok := testCreds()
	if !ok || mode != "disabled" {
		t.Skip("需要 disabled 模式 + 测试账号（自动登录拿真实 refresh token）")
	}

	a, _ := newTestAuth(t)
	ctx := context.Background()
	if _, err := a.Login(ctx, user, pass, auth.CaptchaInput{Type: auth.CaptchaTypeNone}); err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	pair, err := a.Refresh(ctx)
	if err != nil {
		t.Fatalf("刷新失败: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("刷新响应缺少 accessToken")
	}
	t.Log("刷新成功")

	if err := a.Logout(ctx); err != nil {
		t.Logf("登出: %v", err)
	}
}

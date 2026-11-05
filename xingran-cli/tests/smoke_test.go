//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/crypto"
)

// M2 冒烟目标：login → api /system/users/list 全流程。
// 组装方式与 cmd/root.go wiringAuthenticator/wiringEncryptor 一致。
func TestSmokeLoginAndUserList(t *testing.T) {
	mode := captchaMode(t)
	user, pass, ok := testCreds()
	if !ok {
		t.Skip("未设置 XINGRAN_TEST_USER / XINGRAN_TEST_PASS")
	}
	if mode != "disabled" {
		t.Skipf("验证码为 %s 模式，完整冒烟需注入 XINGRAN_CAPTCHA_ANSWER / XINGRAN_SLIDER_X（见 TestLoginE2E）", mode)
	}

	base := baseURL()
	enc := crypto.NewEncryptor()
	httpClient := &http.Client{Timeout: 30 * time.Second}

	// 装配与 cmd/root.go 相同：HTTPRequester(共享 Encryptor) + 文件 TokenStore
	authReq := &auth.HTTPRequester{BaseURL: base, Encryptor: enc, HTTPClient: httpClient}
	store := &auth.FileTokenStore{Path: tempTokenPath(t)}
	authn := auth.NewAuthenticator(authReq, store, enc, base)

	cli := client.New(client.Options{
		BaseURL:       base,
		Timeout:       30 * time.Second,
		Encryptor:     enc,
		Authenticator: authn,
		HTTPClient:    httpClient,
	})
	ctx := context.Background()

	// 1. 登录
	pair, err := authn.Login(ctx, user, pass, auth.CaptchaInput{Type: auth.CaptchaTypeNone})
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	t.Logf("登录成功, expiresIn=%ds", pair.ExpiresIn)

	// 2. 泛化 API 调用：/system/users/list（client 自动带 Bearer token）
	resp, err := cli.Do(ctx, http.MethodPost, usersListPath, map[string]any{
		"current":  1,
		"pageSize": 1,
	})
	if err != nil {
		t.Fatalf("用户列表请求失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("用户列表业务失败: code=%d message=%s", resp.Code, resp.Message)
	}
	var page struct {
		Total int `json:"total"`
		List  []map[string]any
	}
	// list 结构兼容：只校验可解析
	if err := json.Unmarshal(resp.Data, &page); err != nil {
		t.Fatalf("解析用户列表失败: %v, data=%s", err, string(resp.Data))
	}
	t.Logf("冒烟通过: /system/users/list total=%d", page.Total)

	// 3. 登出清理
	if err := authn.Logout(ctx); err != nil {
		t.Fatalf("登出失败: %v", err)
	}

	// 4. 登出后 token 已清除
	if _, err := authn.GetAccessToken(ctx); err == nil {
		t.Fatal("登出后 GetAccessToken 应失败")
	}
}

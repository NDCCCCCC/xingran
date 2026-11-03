package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// 依赖注入用抽象（便于测试）：
// authenticator 由 NewAuthenticator 构造，组合 Requester + TokenStore + 可选 Encryptor。
type authenticator struct {
	req       Requester
	store     TokenStore
	encryptor Encryptor // 可为 nil；非 nil 时 FetchCaptcha 前自动拉公钥
	baseURL   string

	// pendingCaptcha 最近一次 FetchCaptcha 的服务端返回（slider 模式 Login 需要 token）
	pendingCaptcha *CaptchaInfo
}

// NewAuthenticator 构造 Authenticator。
// encryptor 可为 nil（服务端 request_encryption 关闭或 exclude 登录时）；
// 非 nil 时首次需要加密的请求前自动 FetchPublicKey。
func NewAuthenticator(req Requester, store TokenStore, encryptor Encryptor, baseURL string) Authenticator {
	return &authenticator{req: req, store: store, encryptor: encryptor, baseURL: baseURL}
}

// 编译期接口实现断言
var (
	_ Authenticator   = (*authenticator)(nil)
	_ CaptchaProvider = (*authenticator)(nil)
)

// loginResponse 对应后端 v1.LoginResponse。
type loginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

// ensureEncrypted 确保加密公钥可用（幂等，由 Encryptor 实现负责缓存）。
func (a *authenticator) ensureEncrypted(ctx context.Context) error {
	if a.encryptor == nil || a.baseURL == "" {
		return nil
	}
	return a.encryptor.FetchPublicKey(ctx, a.baseURL)
}

// Login 登录。
//
// 三模式流程：
//   - none:  服务端 disabled，直接登录，不带验证码字段
//   - normal: 调用方先 FetchCaptcha → 渲染/保存/输出图片给回填方 → CaptchaInput.Code 填识别字符。
//     captchaId 取内部 pending（FetchCaptcha 缓存）。
//   - slider: 调用方先 FetchCaptcha → 观察缺口 x → CaptchaInput.Code 填 captchaId、
//     SliderX 填缺口 x 坐标。本方法先走 verify/slider，通过后以 "verified" 标记登录。
//
// 登录失败锁定（CheckLoginLock）的服务端提示（如"账号已锁定，请 X 分钟后再试"）
// 以 APIError.Message 原样透传。
func (a *authenticator) Login(ctx context.Context, username, password string, captcha CaptchaInput) (*TokenPair, error) {
	if err := a.ensureEncrypted(ctx); err != nil {
		return nil, err
	}

	body := map[string]any{
		"username":          username,
		"password":          password,
		"encryptedPassword": false, // 请求体整体已被 SM2+SM4 混合加密，密码无需再单独 SM2
	}

	switch captcha.Type {
	case CaptchaTypeNone:
		// 直通
	case CaptchaTypeNormal:
		captchaID := captcha.CaptchaID // 显式注入优先（跨进程两步登录）
		if captchaID == "" {
			if a.pendingCaptcha == nil {
				return nil, fmt.Errorf("normal 验证码模式需要先调用 FetchCaptcha，或显式注入 CaptchaID")
			}
			captchaID = a.pendingCaptcha.CaptchaID
		}
		if captcha.Code == "" {
			return nil, fmt.Errorf("normal 验证码模式需要回填验证码字符")
		}
		body["captchaId"] = captchaID
		body["captcha"] = captcha.Code
	case CaptchaTypeSlider:
		// 凭据来源：显式注入（CaptchaID+Token，跨进程两步登录）优先，
		// 缺省回退进程内 pendingCaptcha 缓存（旧路径兼容）
		captchaID, token := captcha.CaptchaID, captcha.Token
		if captchaID == "" || token == "" {
			if a.pendingCaptcha != nil {
				if captchaID == "" {
					captchaID = a.pendingCaptcha.CaptchaID
				}
				if token == "" {
					token = a.pendingCaptcha.Token
				}
			}
		}
		if captchaID == "" || token == "" {
			return nil, fmt.Errorf("slider 验证码模式需要先调用 FetchCaptcha，或显式注入 CaptchaID+Token")
		}
		if captcha.SliderX <= 0 {
			return nil, fmt.Errorf("slider 验证码模式需要回填缺口 x 坐标（SliderX）")
		}
		// 先验证滑动位置（服务端写入 verified 标记）
		if err := a.VerifySlider(ctx, captchaID, captcha.SliderX, token); err != nil {
			return nil, fmt.Errorf("滑动验证未通过: %w", err)
		}
		body["captchaId"] = captchaID
		body["captcha"] = sliderLoginMarker
	default:
		return nil, fmt.Errorf("未知验证码模式 %q（合法值: normal|slider|none）", captcha.Type)
	}

	resp, err := a.req.Post(ctx, "/system/auth/login", body, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Message}
	}

	var lr loginResponse
	if err := json.Unmarshal(resp.Data, &lr); err != nil {
		return nil, fmt.Errorf("解析登录响应失败: %w", err)
	}
	if lr.AccessToken == "" || lr.RefreshToken == "" {
		return nil, fmt.Errorf("登录响应缺少 token")
	}

	pair := TokenPair{
		AccessToken:  lr.AccessToken,
		RefreshToken: lr.RefreshToken,
		ExpiresIn:    lr.ExpiresIn,
		TokenType:    lr.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(lr.ExpiresIn) * time.Second),
	}
	if err := a.store.Save(ctx, pair); err != nil {
		return nil, fmt.Errorf("保存 token 失败: %w", err)
	}
	a.pendingCaptcha = nil // 一次性
	return &pair, nil
}

// Refresh 用本地 refresh token 换新双 token。
func (a *authenticator) Refresh(ctx context.Context) (*TokenPair, error) {
	old, err := a.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if old == nil || old.RefreshToken == "" {
		return nil, fmt.Errorf("本地无 refresh token，请先登录")
	}
	if err := a.ensureEncrypted(ctx); err != nil {
		return nil, err
	}

	resp, err := a.req.Post(ctx, "/system/auth/refresh", map[string]any{
		"refreshToken": old.RefreshToken,
	}, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Message}
	}

	var lr loginResponse
	if err := json.Unmarshal(resp.Data, &lr); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}
	if lr.AccessToken == "" || lr.RefreshToken == "" {
		return nil, fmt.Errorf("刷新响应缺少 token")
	}

	pair := TokenPair{
		AccessToken:  lr.AccessToken,
		RefreshToken: lr.RefreshToken,
		ExpiresIn:    lr.ExpiresIn,
		TokenType:    lr.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(lr.ExpiresIn) * time.Second),
	}
	if err := a.store.Save(ctx, pair); err != nil {
		return nil, fmt.Errorf("保存 token 失败: %w", err)
	}
	return &pair, nil
}

// GetAccessToken 返回可用 access token；将过期（<30s 余量）自动 refresh。
// refresh 失败时清除本地存储并返回认证错误。
func (a *authenticator) GetAccessToken(ctx context.Context) (string, error) {
	pair, err := a.store.Load(ctx)
	if err != nil {
		return "", err
	}
	if pair == nil || pair.AccessToken == "" {
		return "", fmt.Errorf("未登录，请先执行 login")
	}

	refreshAhead := 30 * time.Second
	if pair.ExpiresAt.IsZero() || time.Until(pair.ExpiresAt) < refreshAhead {
		newPair, err := a.Refresh(ctx)
		if err != nil {
			_ = a.store.Clear(ctx)
			return "", fmt.Errorf("access token 已过期且刷新失败: %w", err)
		}
		return newPair.AccessToken, nil
	}
	return pair.AccessToken, nil
}

// Logout 登出（携带 access token），并清除本地存储。
func (a *authenticator) Logout(ctx context.Context) error {
	pair, err := a.store.Load(ctx)
	if err != nil {
		return err
	}
	if pair == nil {
		return nil // 未登录视为已登出
	}
	if err := a.ensureEncrypted(ctx); err != nil {
		return err
	}
	resp, err := a.req.Post(ctx, "/system/auth/logout", map[string]any{}, pair.AccessToken)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Message: resp.Message}
	}
	return a.store.Clear(ctx)
}

// Package auth XingRan CLI 登录链路：验证码三模式、登录/登出/刷新、JWT 双 token 管理。
//
// 契约见 xingran-cli/CONTRACT.md。本包只依赖标准库；对 crypto 依赖收敛为
// Encryptor 接口（消费端接口，由 internal/crypto 实现），测试可注入 mock。
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CaptchaType 常量：验证码三模式
const (
	CaptchaTypeNone   = "none"   // 后端 disabled：直通，不带验证码字段
	CaptchaTypeNormal = "normal" // 字符验证码：Code 回填
	CaptchaTypeSlider = "slider" // 滑动验证码：SliderX 回填
)

// CaptchaInput 登录验证码输入（契约接口）。
// Type: normal|slider|none。
// normal 模式填 Code；slider 模式填 SliderX；none 模式其余字段忽略。
//
// 验证码凭据来源二选一（slider 为 CaptchaID+Token，normal 为 CaptchaID）：
//   - 显式注入 CaptchaID/Token：跨进程两步登录（进程A FetchCaptcha/落盘展示 →
//     进程B Login），供 CLI login --get-captcha 与后续回填两个独立进程使用
//   - 缺省回退：先调用本进程内 FetchCaptcha，Login 使用其 pending 缓存（旧路径兼容）
type CaptchaInput struct {
	Type      string // normal|slider|none
	Code      string // normal：验证码字符
	SliderX   int    // slider：缺口 x 坐标（后端容差 ±8px；优先取暗区左端，左缘/中心在容差内均可能命中）
	CaptchaID string // 可选：显式注入 FetchCaptcha 得到的 captchaId（跨进程两步登录）
	Token     string // 可选：显式注入 FetchCaptcha 得到的 slider token（跨进程两步登录，仅 slider 需要）
}

// TokenPair JWT 双 token 对（契约接口）。
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"` // access token 有效期（秒）
	TokenType    string    `json:"tokenType"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"` // 本地计算的 access 到期时刻，用于自动刷新
}

// Authenticator 登录链路契约接口。
type Authenticator interface {
	Login(ctx context.Context, username, password string, captcha CaptchaInput) (*TokenPair, error)
	Refresh(ctx context.Context) (*TokenPair, error)
	GetAccessToken(ctx context.Context) (string, error) // access 过期自动 refresh
	Logout(ctx context.Context) error
}

// Encryptor internal/crypto 提供的加密能力（契约接口，消费端声明）。
type Encryptor interface {
	FetchPublicKey(ctx context.Context, baseURL string) error
	EncryptRequest(ctx context.Context, body []byte) (payload []byte, err error)
}

// Response 后端统一响应包装。
type Response struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	RequestID string          `json:"request_id"`
}

// Requester HTTP 抽象：发起 POST 请求（可携带 Bearer token）。
// internal/client 的 Client 可适配为本接口；本包也提供基于 net/http 的默认实现。
type Requester interface {
	Post(ctx context.Context, path string, body any, token string) (*Response, error)
}

// APIError 后端返回 code!=0 时的错误。Message 为服务端原文（如锁定提示），
// 调用方必须原样透传给用户。
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("服务端错误(code=%d): %s", e.Code, e.Message)
}

// IsAuthError 判断是否为认证类错误（401/1007），供上层决定重登。
func IsAuthError(err error) bool {
	if e, ok := err.(*APIError); ok {
		return e.Code == 401 || e.Code == 1007
	}
	return false
}

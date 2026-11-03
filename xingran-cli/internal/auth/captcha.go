package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png" // 注册 PNG 解码器（后端验证码均为 PNG base64）
	"os"
	"strings"
)

// CaptchaInfo GET 验证码返回结构（对应后端 pkg/captcha.CaptchaResponse）。
type CaptchaInfo struct {
	CaptchaID   string `json:"captchaId"`   // 验证码 ID
	CaptchaType string `json:"captchaType"` // normal | slider
	CaptchaImg  string `json:"captchaImg"`  // 字符验证码 base64 图（data:image/png;base64,...）
	SliderImg   string `json:"sliderImg"`   // 滑动底图 base64 图
	PieceImg    string `json:"pieceImg"`    // 拼图块 base64 图
	YPos        int    `json:"yPos"`        // 拼图块 Y 坐标（滑动）
	Token       string `json:"token"`       // 滑动验证 token（滑动）
}

// CaptchaProvider 验证码编排扩展接口（*authenticator 同时实现）。
// cmd 层非交互流程：FetchCaptcha → 展示图片（RenderTerminal/SaveImage/dataURL 直出）
// → 回填 CaptchaInput{Type, Code|SliderX} → Authenticator.Login。
type CaptchaProvider interface {
	FetchCaptcha(ctx context.Context) (*CaptchaInfo, error)
	VerifySlider(ctx context.Context, captchaID string, xPos int, token string) error
}

// sliderLoginMarker 后端约定：滑动验证码 verify 通过后，登录请求的 captcha
// 字段必须传字面量 "verified"（见 internal/core/captcha.go VerifyCaptcha slider 分支）。
const sliderLoginMarker = "verified"

// FetchCaptcha 获取验证码（POST /system/auth/captcha）。
// 后端 disabled 时 data 为空对象，返回 (nil, nil)。
// 获取到的信息缓存于 authenticator 内部（pendingCaptcha），供 slider 模式 Login 使用。
func (a *authenticator) FetchCaptcha(ctx context.Context) (*CaptchaInfo, error) {
	if err := a.ensureEncrypted(ctx); err != nil {
		return nil, err
	}
	resp, err := a.req.Post(ctx, "/system/auth/captcha", map[string]any{}, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Message}
	}
	if len(resp.Data) == 0 || string(resp.Data) == "{}" {
		return nil, nil // 验证码未启用（disabled 直通）
	}
	var info CaptchaInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		return nil, fmt.Errorf("解析验证码响应失败: %w", err)
	}
	a.pendingCaptcha = &info
	return &info, nil
}

// VerifySlider 验证滑动验证码（POST /system/auth/captcha/verify/slider）。
// xPos 由用户/多模态 agent 观察缺口位置给出（±8px 容差）。
// 成功后服务端写入 verified 标记，随后 Login 携带同一 captchaId 即可通过。
func (a *authenticator) VerifySlider(ctx context.Context, captchaID string, xPos int, token string) error {
	if captchaID == "" {
		return errors.New("滑动验证失败：缺少 captchaId，请先调用 FetchCaptcha")
	}
	if err := a.ensureEncrypted(ctx); err != nil {
		return err
	}
	resp, err := a.req.Post(ctx, "/system/auth/captcha/verify/slider", map[string]any{
		"captchaId": captchaID,
		"xPos":      xPos,
		"token":     token,
	}, "")
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Message: resp.Message}
	}
	return nil
}

// decodeDataURL 解析 dataURL 中的图片字节。
func decodeDataURL(dataURL string) ([]byte, error) {
	if dataURL == "" {
		return nil, errors.New("图片数据为空")
	}
	// 剥离 data:image/xxx;base64, 前缀
	if idx := strings.Index(dataURL, ","); idx >= 0 && strings.HasPrefix(dataURL, "data:") {
		dataURL = dataURL[idx+1:]
	}
	return base64.StdEncoding.DecodeString(dataURL)
}

// SaveImage 将验证码 dataURL 图片解码保存到 path（供 agent 读取分析）。
func SaveImage(dataURL, path string) error {
	data, err := decodeDataURL(dataURL)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// asciiRamp 亮度→字符映射（暗→亮）。
const asciiRamp = "@%#*+=-:. "

// RenderTerminal 将验证码 dataURL 渲染为终端 ASCII 字符画（供无 GUI 场景肉眼辨识）。
func RenderTerminal(dataURL string, maxCols int) (string, error) {
	data, err := decodeDataURL(dataURL)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return "", fmt.Errorf("解码验证码图片失败: %w", err)
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	if maxCols <= 0 || w <= maxCols {
		maxCols = w
	}
	scale := (w + maxCols - 1) / maxCols
	if scale < 1 {
		scale = 1
	}

	var sb strings.Builder
	for y := bounds.Min.Y; y < bounds.Max.Y; y += scale {
		for x := bounds.Min.X; x < bounds.Max.X; x += scale {
			r, g, b, _ := img.At(x, y).RGBA()
			// ITU-R BT.601 亮度
			luma := (299*r + 587*g + 114*b) / 1000 / 257 // 0-255
			rampIdx := (int(luma) * (len(asciiRamp) - 1)) / 255
			sb.WriteByte(asciiRamp[rampIdx])
		}
		sb.WriteByte('\n')
	}
	return sb.String(), nil
}

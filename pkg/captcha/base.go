// Package captcha 提供验证码生成功能
// 支持数字字母验证码和滑动拼图验证码
package captcha

// CaptchaType 验证码类型
type CaptchaType string

const (
	CaptchaTypeDisabled CaptchaType = "disabled" // 停用
	CaptchaTypeNormal   CaptchaType = "normal"   // 数字字母验证码
	CaptchaTypeSlider   CaptchaType = "slider"   // 滑动拼图验证码
)

// CaptchaResponse 验证码响应
type CaptchaResponse struct {
	CaptchaID   string      `json:"captchaId"`            // 验证码ID
	CaptchaType CaptchaType `json:"captchaType"`          // 验证码类型
	CaptchaImg  string      `json:"captchaImg,omitempty"` // base64图片 (文字类型)
	SliderImg   string      `json:"sliderImg,omitempty"`  // 滑动底图 (滑动类型)
	PieceImg    string      `json:"pieceImg,omitempty"`   // 拼图块 (滑动类型)
	YPos        int         `json:"yPos,omitempty"`       // 拼图块Y坐标 (滑动类型)
	Token       string      `json:"token,omitempty"`      // 验证token (滑动类型)
}

// TextGenerator 数字字母验证码生成器接口
type TextGenerator interface {
	Generate() (img []byte, code string, err error)
}

// SliderGenerator 滑动拼图验证码生成器接口
type SliderGenerator interface {
	GenerateSlider() (bgImg, pieceImg []byte, yPos int, token string, err error)
}

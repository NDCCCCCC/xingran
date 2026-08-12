// Package captcha 数字字母验证码生成器实现
// 使用 base64Captcha 库生成验证码
package captcha

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mojocn/base64Captcha"
)

// TextCaptcha 数字字母验证码生成器
type TextCaptcha struct {
	driver base64Captcha.Driver
	store  base64Captcha.Store
}

// NewTextCaptcha 创建数字字母验证码生成器
// length: 验证码长度 (4-6)
func NewTextCaptcha(length int) *TextCaptcha {
	// 验证长度范围
	if length < 4 {
		length = 4
	}
	if length > 6 {
		length = 6
	}

	// 创建数字验证码驱动
	// 参数: 高度, 宽度, 验证码长度, 最大扭曲度, 干扰点数量
	driver := base64Captcha.NewDriverDigit(
		40,     // height: 图片高度
		120,    // width: 图片宽度
		length, // length: 验证码长度
		0.7,    // maxSkew: 最大扭曲度
		80,     // dotCount: 干扰点数量
	)

	// 创建内存存储
	store := base64Captcha.NewMemoryStore(100, time.Minute*5)

	return &TextCaptcha{
		driver: driver,
		store:  store,
	}
}

// Generate 生成验证码
// 返回: 图片字节、验证码答案、错误
func (t *TextCaptcha) Generate() ([]byte, string, error) {
	// 创建验证码实例
	captcha := base64Captcha.NewCaptcha(t.driver, t.store)

	// 生成验证码，返回: id, base64图片, 答案, 错误
	id, b64s, answer, err := captcha.Generate()
	if err != nil {
		return nil, "", fmt.Errorf("生成验证码失败: %w", err)
	}

	// b64s 已经是 base64 编码的图片字符串，格式: "data:image/png;base64,..."
	// 需要去掉前缀并解码
	const prefix = "data:image/png;base64,"
	if len(b64s) < len(prefix) {
		return nil, "", fmt.Errorf("生成的验证码格式错误")
	}

	imgData := b64s[len(prefix):]
	imgBytes, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		return nil, "", fmt.Errorf("解码验证码图片失败: %w", err)
	}

	_ = id // id 由库自动生成，我们使用自己的 id
	return imgBytes, answer, nil
}

// GenerateBase64 生成 base64 格式的验证码图片
// 返回: base64编码的图片、验证码答案、错误
func (t *TextCaptcha) GenerateBase64() (string, string, error) {
	imgBytes, code, err := t.Generate()
	if err != nil {
		return "", "", err
	}

	base64Img := base64.StdEncoding.EncodeToString(imgBytes)
	return base64Img, code, nil
}

// GenerateWithID 生成带ID的验证码
// 返回: 图片字节、验证码答案、验证码ID、错误
func (t *TextCaptcha) GenerateWithID() ([]byte, string, string, error) {
	imgBytes, code, err := t.Generate()
	if err != nil {
		return nil, "", "", err
	}

	id := generateCaptchaID()
	return imgBytes, code, id, nil
}

// generateCaptchaID 生成验证码ID
func generateCaptchaID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

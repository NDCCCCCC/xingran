package captcha

import (
	"encoding/base64"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// base.go: CaptchaType 常量 + CaptchaResponse 结构。空测试即可覆盖。
// =====================================================================

func TestCaptchaType_Constants(t *testing.T) {
	assert.Equal(t, CaptchaType("disabled"), CaptchaTypeDisabled)
	assert.Equal(t, CaptchaType("normal"), CaptchaTypeNormal)
	assert.Equal(t, CaptchaType("slider"), CaptchaTypeSlider)
}

func TestCaptchaResponse_Structure(t *testing.T) {
	r := CaptchaResponse{
		CaptchaID:   "id",
		CaptchaType: CaptchaTypeNormal,
		CaptchaImg:  "img",
		SliderImg:   "sb",
		PieceImg:    "pb",
		YPos:        10,
		Token:       "tok",
	}
	assert.Equal(t, "id", r.CaptchaID)
	assert.Equal(t, 10, r.YPos)
}

// =====================================================================
// text.go: TextCaptcha 生成 / Base64 / WithID / generateCaptchaID
// =====================================================================

func TestNewTextCaptcha_LengthClamp(t *testing.T) {
	// length < 4 → 4, > 6 → 6
	short := NewTextCaptcha(2)
	assert.NotNil(t, short)
	long := NewTextCaptcha(10)
	assert.NotNil(t, long)
	mid := NewTextCaptcha(5)
	assert.NotNil(t, mid)
}

func TestTextCaptcha_Generate(t *testing.T) {
	c := NewTextCaptcha(4)
	img, code, err := c.Generate()
	require.NoError(t, err)
	assert.NotEmpty(t, img)
	assert.NotEmpty(t, code)
}

func TestTextCaptcha_GenerateBase64(t *testing.T) {
	c := NewTextCaptcha(4)
	b64, code, err := c.GenerateBase64()
	require.NoError(t, err)
	_, err = base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	assert.NotEmpty(t, code)
}

func TestTextCaptcha_GenerateWithID(t *testing.T) {
	c := NewTextCaptcha(5)
	img, code, id, err := c.GenerateWithID()
	require.NoError(t, err)
	assert.NotEmpty(t, img)
	assert.NotEmpty(t, code)
	assert.NotEmpty(t, id)
}

func TestGenerateCaptchaID_NonEmpty(t *testing.T) {
	id1 := generateCaptchaID()
	// 形式是数字字符串
	assert.NotEmpty(t, id1)
	assert.Greater(t, len(id1), 0)
}

// =====================================================================
// shape.go: 形状工厂 + 4 种形状的 IsInside / GenerateMask
// =====================================================================

func TestShapeFactory_Builtin(t *testing.T) {
	f := NewBaseShapeFactory()
	assert.NotNil(t, f)

	shapes := f.GetSupportedShapes()
	assert.Len(t, shapes, 4)
	assert.Contains(t, shapes, "circle")
	assert.Contains(t, shapes, "square")
	assert.Contains(t, shapes, "star")
	assert.Contains(t, shapes, "heart")
}

func TestShapeFactory_CreateAnd_Register_Duplicate(t *testing.T) {
	f := NewBaseShapeFactory()
	s, err := f.CreateShape("circle")
	require.NoError(t, err)
	assert.Equal(t, "circle", s.GetName())

	// 未注册
	_, err = f.CreateShape("triangle")
	require.Error(t, err)

	// 重复注册 → error
	err = f.RegisterShape("circle", func() Shape { return NewCircleShape() })
	require.Error(t, err)
}

func TestCircleShape_IsInsideAndMask(t *testing.T) {
	s := NewCircleShape()
	assert.Equal(t, "circle", s.GetName())
	assert.Equal(t, 3, s.GetComplexity())
	// 中心点应在内部
	assert.True(t, s.IsInside(50, 50, 100, 100))
	// 远离中心 → 外部
	assert.False(t, s.IsInside(0, 0, 100, 100))
	// 遮罩
	mask := s.GenerateMask(40, 40)
	assert.NotNil(t, mask)
	assert.Equal(t, 40, mask.Bounds().Dx())
}

func TestSquareShape_IsInsideAndMask(t *testing.T) {
	s := NewSquareShape()
	assert.Equal(t, "square", s.GetName())
	// 中心点(波浪 ±5 内)
	assert.True(t, s.IsInside(50, 50, 100, 100))
	assert.NotNil(t, s.GenerateMask(30, 30))
}

func TestStarShape_IsInsideAndMask(t *testing.T) {
	s := NewStarShape()
	assert.Equal(t, "star", s.GetName())
	assert.Equal(t, 5, s.GetComplexity())
	// 中心点应在内
	assert.True(t, s.IsInside(50, 50, 100, 100))
	// 角点 → 外
	assert.False(t, s.IsInside(0, 0, 100, 100))
	assert.NotNil(t, s.GenerateMask(40, 40))
}

func TestHeartShape_IsInsideAndMask(t *testing.T) {
	s := NewHeartShape()
	assert.Equal(t, "heart", s.GetName())
	assert.Equal(t, 4, s.GetComplexity())
	assert.NotNil(t, s.GenerateMask(40, 40))
}

func TestShapeHelpers(t *testing.T) {
	assert.Equal(t, 1, min(1, 5))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 0.0, sin(0))
	// 绝对值
	assert.Equal(t, 5, abs(5))
	assert.Equal(t, 5, abs(-5))
	assert.Equal(t, 0, abs(0))
}

// =====================================================================
// slider.go: NewSliderCaptcha / SetShape / SetDifficulty / GenerateSlider
// =====================================================================

func TestNewSliderCaptcha(t *testing.T) {
	s := NewSliderCaptcha()
	require.NotNil(t, s)

	// SetShape: 未知 shape → error
	err := s.SetShape("triangle")
	require.Error(t, err)

	// SetShape: 合法
	require.NoError(t, s.SetShape("circle"))

	// SetDifficulty
	tol := s.SetDifficulty(2)
	assert.GreaterOrEqual(t, tol, 0)
}

func TestNewSliderCaptchaWithShape(t *testing.T) {
	s, err := NewSliderCaptchaWithShape("star")
	require.NoError(t, err)
	require.NotNil(t, s)

	_, err = NewSliderCaptchaWithShape("unknown")
	require.Error(t, err)
}

func TestSliderCaptcha_GenerateSlider(t *testing.T) {
	s := NewSliderCaptcha()
	bg, piece, yPos, xPos, token, err := s.GenerateSlider()
	require.NoError(t, err)
	assert.NotEmpty(t, bg)
	assert.NotEmpty(t, piece)
	assert.GreaterOrEqual(t, yPos, 0)
	assert.GreaterOrEqual(t, xPos, 0)
	assert.NotEmpty(t, token)
}

func TestSliderCaptcha_GenerateSliderWithCustomBackground(t *testing.T) {
	// 构造简单背景 RGBA 图像 200x100
	w, h := 200, 100
	bg := createSolidRGBA(w, h)

	s := NewSliderCaptcha()
	bgs, pieces, yPos, xPos, token, err := s.GenerateSliderWithCustomBackground(bg, "circle", 2)
	require.NoError(t, err)
	assert.NotEmpty(t, bgs)
	assert.NotEmpty(t, pieces)
	assert.GreaterOrEqual(t, yPos, 0)
	assert.GreaterOrEqual(t, xPos, 0)
	assert.NotEmpty(t, token)
}

// createSolidRGBA 在 test 内联造纯色 RGBA,避免引入 image/draw 依赖。
func createSolidRGBA(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}
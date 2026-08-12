// Package captcha 滑动拼图验证码生成器实现
// 使用标准库 image/draw 生成拼图验证码
package captcha

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SliderCaptcha 滑动拼图验证码生成器
type SliderCaptcha struct {
	Width        int          // 背景图宽度
	Height       int          // 背景图高度
	PieceWidth   int          // 拼图块宽度
	PieceHeight  int          // 拼图块高度
	Shape        Shape        // 拼图形状（新增）
	ShapeFactory ShapeFactory // 形状工厂（新增）
}

// NewSliderCaptcha 创建滑动拼图验证码生成器
func NewSliderCaptcha() *SliderCaptcha {
	factory := NewBaseShapeFactory()
	return &SliderCaptcha{
		Width:        400, // 背景图宽度（增大以提高清晰度）
		Height:       200, // 背景图高度（增大以提高清晰度）
		PieceWidth:   80,  // 拼图块宽度（相应增大）
		PieceHeight:  80,  // 拼图块高度（相应增大）
		Shape:        nil, // 延迟初始化
		ShapeFactory: factory,
	}
}

// NewSliderCaptchaWithShape 创建带指定形状的滑动拼图验证码生成器
func NewSliderCaptchaWithShape(shapeType string) (*SliderCaptcha, error) {
	factory := NewBaseShapeFactory()
	shape, err := factory.CreateShape(shapeType)
	if err != nil {
		return nil, err
	}
	return &SliderCaptcha{
		Width:        400, // 背景图宽度（增大以提高清晰度）
		Height:       200, // 背景图高度（增大以提高清晰度）
		PieceWidth:   80,  // 拼图块宽度（相应增大）
		PieceHeight:  80,  // 拼图块高度（相应增大）
		Shape:        shape,
		ShapeFactory: factory,
	}, nil
}

// SetShape 设置拼图形状
func (s *SliderCaptcha) SetShape(shapeType string) error {
	shape, err := s.ShapeFactory.CreateShape(shapeType)
	if err != nil {
		return err
	}
	s.Shape = shape
	return nil
}

// SetDifficulty 设置难度级别（调整拼图块大小和容差）
func (s *SliderCaptcha) SetDifficulty(level int) (tolerance int) {
	switch level {
	case 1: // 简单
		s.PieceWidth = 90
		s.PieceHeight = 90
		tolerance = 16
	case 2: // 中等
		s.PieceWidth = 80
		s.PieceHeight = 80
		tolerance = 12
	case 3: // 困难
		s.PieceWidth = 70
		s.PieceHeight = 70
		tolerance = 8
	default:
		s.PieceWidth = 80
		s.PieceHeight = 80
		tolerance = 12
	}
	return tolerance
}

// LoadBackgroundFromFile 从文件加载背景图
func (s *SliderCaptcha) LoadBackgroundFromFile(filePath string) (*image.RGBA, error) {
	// 兼容跨平台：将反斜杠统一转为正斜杠，再用系统路径分隔符重建
	normalized := filepath.FromSlash(strings.ReplaceAll(filePath, "\\", "/"))
	file, err := os.Open(normalized)
	if err != nil {
		return nil, fmt.Errorf("打开背景图失败: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("解码背景图失败: %w", err)
	}

	// 转换为RGBA
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)

	return rgba, nil
}

// GenerateSliderWithCustomBackground 使用自定义背景图生成验证码
func (s *SliderCaptcha) GenerateSliderWithCustomBackground(bgImg *image.RGBA, shapeType string, difficulty int) ([]byte, []byte, int, int, string, error) {
	// 设置形状和难度
	if err := s.SetShape(shapeType); err != nil {
		return nil, nil, 0, 0, "", err
	}
	_ = s.SetDifficulty(difficulty) // tolerance返回值暂时不使用

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 调整背景图尺寸（如果需要）
	bgImg = s.resizeImage(bgImg, s.Width, s.Height)

	// 随机生成拼图块位置
	minX := s.PieceWidth + 20
	maxX := s.Width - s.PieceWidth*2
	xPos := r.Intn(maxX-minX+1) + minX

	minY := 20
	maxY := s.Height - s.PieceHeight - 20
	yPos := r.Intn(maxY-minY+1) + minY

	// 先从背景图生成拼图块（提取缺口处的图像内容）
	pieceImg := s.generatePieceFromBackground(bgImg, xPos, yPos)

	// 在背景图上绘制阴影缺口
	s.drawShadowOnBackgroundWithShape(bgImg, xPos, yPos)

	// 生成token（包含难度信息）
	token := s.generateTokenWithDifficulty(xPos, yPos, shapeType, difficulty)

	// 转换为字节
	bgBytes, err := s.imageToBytes(bgImg)
	if err != nil {
		return nil, nil, 0, 0, "", fmt.Errorf("转换背景图失败: %w", err)
	}

	pieceBytes, err := s.imageToBytes(pieceImg)
	if err != nil {
		return nil, nil, 0, 0, "", fmt.Errorf("转换拼图块失败: %w", err)
	}

	return bgBytes, pieceBytes, xPos, yPos, token, nil
}

// resizeImage 调整图片尺寸
func (s *SliderCaptcha) resizeImage(img *image.RGBA, width, height int) *image.RGBA {
	if img.Bounds().Dx() == width && img.Bounds().Dy() == height {
		return img
	}

	// 简单的缩放实现（生产环境可使用更好的缩放算法）
	resized := image.NewRGBA(image.Rect(0, 0, width, height))

	scaleX := float64(img.Bounds().Dx()) / float64(width)
	scaleY := float64(img.Bounds().Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			if srcX < img.Bounds().Dx() && srcY < img.Bounds().Dy() {
				resized.Set(x, y, img.RGBAAt(srcX, srcY))
			}
		}
	}

	return resized
}

// generateTokenWithDifficulty 生成包含难度信息的token
func (s *SliderCaptcha) generateTokenWithDifficulty(xPos, yPos int, shape string, difficulty int) string {
	data := fmt.Sprintf("%d|%d|%s|%d|%d", xPos, yPos, shape, difficulty, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16]
}

// GenerateSlider 生成滑动拼图验证码
// 返回: 背景图、拼图块、X坐标、Y坐标、验证token、错误
func (s *SliderCaptcha) GenerateSlider() ([]byte, []byte, int, int, string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 1. 生成背景图（带噪点的渐变色图片）
	bgImg := s.generateBackground(r)

	// 2. 随机生成拼图块的 X 和 Y 坐标
	// X 坐标范围：从 pieceWidth 到 Width - pieceWidth，确保拼图块可以完全放入
	minX := s.PieceWidth + 20        // 最小 X 位置，给拼图块留出初始移动空间
	maxX := s.Width - s.PieceWidth*2 // 最大 X 位置
	xPos := r.Intn(maxX-minX+1) + minX

	// Y 坐标范围
	minY := 20
	maxY := s.Height - s.PieceHeight - 20
	yPos := r.Intn(maxY-minY+1) + minY

	// 3. 生成拼图块（带缺口的拼图形状）
	pieceImg := s.generatePiece(r)

	// 4. 在背景图上指定位置绘制拼图块位置的阴影（缺口效果）
	s.drawShadowOnBackground(bgImg, xPos, yPos)

	// 5. 生成验证 token（包含X坐标、Y坐标和时间戳）
	token := s.generateTokenWithPosition(xPos, yPos)

	// 6. 转换为 PNG 字节
	bgBytes, err := s.imageToBytes(bgImg)
	if err != nil {
		return nil, nil, 0, 0, "", fmt.Errorf("转换背景图失败: %w", err)
	}

	pieceBytes, err := s.imageToBytes(pieceImg)
	if err != nil {
		return nil, nil, 0, 0, "", fmt.Errorf("转换拼图块失败: %w", err)
	}

	return bgBytes, pieceBytes, xPos, yPos, token, nil
}

// generateBackground 生成背景图
func (s *SliderCaptcha) generateBackground(r *rand.Rand) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, s.Width, s.Height))

	// 生成渐变背景
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			// 根据位置生成渐变色
			r1 := uint8(200 + (x*55)/s.Width)
			g1 := uint8(220 + (y*35)/s.Height)
			b1 := uint8(240 - (x*30)/s.Width)
			img.Set(x, y, color.RGBA{R: r1, G: g1, B: b1, A: 255})
		}
	}

	// 添加随机噪点
	for i := 0; i < 500; i++ {
		x := r.Intn(s.Width)
		y := r.Intn(s.Height)
		noise := color.RGBA{
			R: uint8(r.Intn(256)),
			G: uint8(r.Intn(256)),
			B: uint8(r.Intn(256)),
			A: 100,
		}
		img.Set(x, y, noise)
	}

	// 添加干扰线
	for i := 0; i < 5; i++ {
		x1 := r.Intn(s.Width)
		y1 := r.Intn(s.Height)
		x2 := r.Intn(s.Width)
		y2 := r.Intn(s.Height)
		s.drawLine(img, x1, y1, x2, y2, color.RGBA{R: 150, G: 150, B: 150, A: 150})
	}

	return img
}

// generatePieceWithShape 使用指定形状生成拼图块
func (s *SliderCaptcha) generatePieceWithShape(r *rand.Rand) *image.RGBA {
	// 如果没有设置形状，使用默认圆形
	if s.Shape == nil {
		s.Shape = NewCircleShape()
	}

	// 创建拼图块图片
	pieceImg := image.NewRGBA(image.Rect(0, 0, s.PieceWidth, s.PieceHeight))

	// 根据形状填充
	for y := 0; y < s.PieceHeight; y++ {
		for x := 0; x < s.PieceWidth; x++ {
			if s.Shape.IsInside(x, y, s.PieceWidth, s.PieceHeight) {
				// 在形状内填充渐变色
				r1 := uint8(100 + (x*155)/s.PieceWidth)
				g1 := uint8(150 + (y*105)/s.PieceHeight)
				b1 := uint8(200 - (x*50)/s.PieceWidth)
				pieceImg.Set(x, y, color.RGBA{R: r1, G: g1, B: b1, A: 255})
			} else {
				// 在形状外设置透明
				pieceImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	// 添加边框
	s.addPieceBorder(pieceImg)

	// 添加箭头指示
	s.addPieceText(pieceImg, r)

	return pieceImg
}

// generatePieceFromBackground 从背景图中提取指定位置的图像内容生成拼图块
func (s *SliderCaptcha) generatePieceFromBackground(bgImg *image.RGBA, xPos, yPos int) *image.RGBA {
	// 如果没有设置形状，使用默认圆形
	if s.Shape == nil {
		s.Shape = NewCircleShape()
	}

	// 创建拼图块图片
	pieceImg := image.NewRGBA(image.Rect(0, 0, s.PieceWidth, s.PieceHeight))

	// 从背景图中提取对应位置的像素内容
	for y := 0; y < s.PieceHeight; y++ {
		for x := 0; x < s.PieceWidth; x++ {
			// 计算在背景图上的实际位置
			bgX := xPos + x
			bgY := yPos + y

			// 检查是否在背景图范围内
			if bgX < 0 || bgX >= bgImg.Bounds().Dx() || bgY < 0 || bgY >= bgImg.Bounds().Dy() {
				// 超出范围，设置为透明
				pieceImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
				continue
			}

			// 检查点是否在形状内
			if s.Shape.IsInside(x, y, s.PieceWidth, s.PieceHeight) {
				// 在形状内，复制背景图的像素
				c := bgImg.RGBAAt(bgX, bgY)
				pieceImg.Set(x, y, c)
			} else {
				// 在形状外设置透明
				pieceImg.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 0})
			}
		}
	}

	// 添加边框（使用白色使拼图块更明显）
	s.addPieceBorderWhite(pieceImg)

	// 添加高光效果（使拼图块看起来有立体感）
	s.addPieceHighlight(pieceImg)

	return pieceImg
}

// addPieceBorderWhite 添加白色边框到拼图块
func (s *SliderCaptcha) addPieceBorderWhite(img *image.RGBA) {
	borderColor := color.RGBA{R: 255, G: 255, B: 255, A: 255} // 白色边框
	borderThickness := 2                                      // 边框厚度

	for y := 0; y < s.PieceHeight; y++ {
		for x := 0; x < s.PieceWidth; x++ {
			// 检查点是否在形状内
			if !s.Shape.IsInside(x, y, s.PieceWidth, s.PieceHeight) {
				continue
			}

			// 检查是否是边缘像素（靠近形状边界）
			isEdge := false
			for dy := -borderThickness; dy <= borderThickness; dy++ {
				for dx := -borderThickness; dx <= borderThickness; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= s.PieceWidth || ny < 0 || ny >= s.PieceHeight {
						continue
					}
					if !s.Shape.IsInside(nx, ny, s.PieceWidth, s.PieceHeight) {
						isEdge = true
						break
					}
				}
				if isEdge {
					break
				}
			}

			if isEdge {
				img.Set(x, y, borderColor)
			}
		}
	}
}

// addPieceHighlight 添加高光效果使拼图块更有立体感
func (s *SliderCaptcha) addPieceHighlight(img *image.RGBA) {
	// 在左上角添加高光（假设光源来自左上角）
	for y := 0; y < s.PieceHeight; y++ {
		for x := 0; x < s.PieceWidth; x++ {
			if !s.Shape.IsInside(x, y, s.PieceWidth, s.PieceHeight) {
				continue
			}

			// 计算到中心的距离
			centerX := s.PieceWidth / 2
			centerY := s.PieceHeight / 2
			dist := float64(x-centerX)*float64(x-centerX) + float64(y-centerY)*float64(y-centerY)
			maxDist := float64(centerX*centerX + centerY*centerY)

			// 根据位置和距离调整亮度
			c := img.RGBAAt(x, y)

			// 左上角更亮
			brightnessBoost := int((1 - dist/maxDist) * 50)

			img.Set(x, y, color.RGBA{
				R: uint8(min(255, int(c.R)+brightnessBoost)),
				G: uint8(min(255, int(c.G)+brightnessBoost)),
				B: uint8(min(255, int(c.B)+brightnessBoost)),
				A: c.A,
			})
		}
	}
}

// drawShadowOnBackgroundWithShape 使用指定形状在背景图上绘制阴影
func (s *SliderCaptcha) drawShadowOnBackgroundWithShape(bgImg *image.RGBA, xPos, yPos int) {
	if s.Shape == nil {
		s.Shape = NewCircleShape()
	}

	// 在拼图块位置绘制明显的缺口（阴影效果）
	for y := yPos; y < yPos+s.PieceHeight; y++ {
		for x := xPos; x < xPos+s.PieceWidth; x++ {
			if x >= s.Width || y >= s.Height {
				continue
			}

			// 检查点是否在形状内
			if s.Shape.IsInside(x-xPos, y-yPos, s.PieceWidth, s.PieceHeight) {
				// 叠加深色阴影（变暗到 25%），使缺口更明显
				c := bgImg.RGBAAt(x, y)
				bgImg.Set(x, y, color.RGBA{
					R: uint8(int(c.R) / 4),
					G: uint8(int(c.G) / 4),
					B: uint8(int(c.B) / 4),
					A: 255,
				})
			}
		}
	}

	// 绘制明显的边框，使缺口轮廓更清晰
	s.drawGapBorder(bgImg, xPos, yPos)
}

// drawGapBorder 绘制缺口边框
func (s *SliderCaptcha) drawGapBorder(bgImg *image.RGBA, xPos, yPos int) {
	borderColor := color.RGBA{R: 0, G: 0, B: 0, A: 180} // 半透明黑色边框

	// 遍历拼图块区域的边缘
	for y := yPos; y < yPos+s.PieceHeight; y++ {
		for x := xPos; x < xPos+s.PieceWidth; x++ {
			if x >= s.Width || y >= s.Height {
				continue
			}

			// 检查当前点是否在形状内
			inside := s.Shape.IsInside(x-xPos, y-yPos, s.PieceWidth, s.PieceHeight)

			// 检查相邻点是否在形状外（如果在边缘则绘制边框）
			isEdge := false
			if !inside {
				continue
			}

			// 检查四个方向
			directions := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
			for _, dir := range directions {
				nx, ny := x+dir[0], y+dir[1]
				if nx < xPos || nx >= xPos+s.PieceWidth || ny < yPos || ny >= yPos+s.PieceHeight {
					isEdge = true
					break
				}
				if !s.Shape.IsInside(nx-xPos, ny-yPos, s.PieceWidth, s.PieceHeight) {
					isEdge = true
					break
				}
			}

			if isEdge {
				// 叠加边框颜色（使边缘更深）
				c := bgImg.RGBAAt(x, y)
				bgImg.Set(x, y, color.RGBA{
					R: uint8((int(c.R) + int(borderColor.R)) / 2),
					G: uint8((int(c.G) + int(borderColor.G)) / 2),
					B: uint8((int(c.B) + int(borderColor.B)) / 2),
					A: 255,
				})
			}
		}
	}
}

// generatePiece 生成拼图块（兼容旧代码）
func (s *SliderCaptcha) generatePiece(r *rand.Rand) *image.RGBA {
	return s.generatePieceWithShape(r)
}

// drawShadowOnBackground 在背景图上绘制阴影（缺口）
func (s *SliderCaptcha) drawShadowOnBackground(bgImg *image.RGBA, xPos, yPos int) {
	s.drawShadowOnBackgroundWithShape(bgImg, xPos, yPos)
}

// addPieceBorder 添加拼图块边框
func (s *SliderCaptcha) addPieceBorder(img *image.RGBA) {
	// 绘制边框
	borderColor := color.RGBA{R: 80, G: 80, B: 80, A: 255}

	// 上边
	for x := 0; x < s.PieceWidth; x++ {
		img.Set(x, 0, borderColor)
	}
	// 下边
	for x := 0; x < s.PieceWidth; x++ {
		img.Set(x, s.PieceHeight-1, borderColor)
	}
	// 左边
	for y := 0; y < s.PieceHeight; y++ {
		img.Set(0, y, borderColor)
	}
	// 右边
	for y := 0; y < s.PieceHeight; y++ {
		img.Set(s.PieceWidth-1, y, borderColor)
	}
}

// addPieceText 添加拼图块文字
func (s *SliderCaptcha) addPieceText(img *image.RGBA, r *rand.Rand) {
	// 在拼图块中心添加方向箭头指示
	centerX := s.PieceWidth / 2
	centerY := s.PieceHeight / 2

	// 绘制简单箭头指向右侧
	arrowColor := color.RGBA{R: 255, G: 255, B: 255, A: 200}

	// 箭头主干
	for x := centerX - 10; x < centerX+10; x++ {
		if x >= 0 && x < s.PieceWidth {
			img.Set(x, centerY, arrowColor)
		}
	}
	// 箭头头部
	for i := 0; i < 8; i++ {
		x := centerX + 10 - i
		y1 := centerY - i
		y2 := centerY + i
		if x >= 0 && x < s.PieceWidth && y1 >= 0 && y1 < s.PieceHeight {
			img.Set(x, y1, arrowColor)
		}
		if x >= 0 && x < s.PieceWidth && y2 >= 0 && y2 < s.PieceHeight {
			img.Set(x, y2, arrowColor)
		}
	}
}

// drawLine 绘制线条
func (s *SliderCaptcha) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// generateTokenWithPosition 生成包含X和Y坐标的验证token
func (s *SliderCaptcha) generateTokenWithPosition(xPos, yPos int) string {
	// 生成包含X坐标、Y坐标和时间戳的哈希token
	data := fmt.Sprintf("%d|%d|%d", xPos, yPos, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16]
}

// imageToBytes 将图片转换为PNG字节
func (s *SliderCaptcha) imageToBytes(img *image.RGBA) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateSliderBase64 生成base64格式的滑动拼图验证码
// 返回: 背景图base64、拼图块base64、X坐标、Y坐标、token、错误
func (s *SliderCaptcha) GenerateSliderBase64() (string, string, int, int, string, error) {
	bgBytes, pieceBytes, xPos, yPos, token, err := s.GenerateSlider()
	if err != nil {
		return "", "", 0, 0, "", err
	}

	bgBase64 := base64.StdEncoding.EncodeToString(bgBytes)
	pieceBase64 := base64.StdEncoding.EncodeToString(pieceBytes)

	return bgBase64, pieceBase64, xPos, yPos, token, nil
}

// VerifyToken 验证token是否匹配
// 实际使用时需要根据滑动的X坐标和存储的Y坐标进行验证
func (s *SliderCaptcha) VerifyToken(token string, storedYPos int, xPos int) bool {
	// 在实际应用中，这里应该:
	// 1. 从token或存储中获取正确的Y坐标
	// 2. 验证滑动位置是否在合理范围内
	// 3. 检查token是否过期

	// 简化版本：只需要token非空
	return token != ""
}

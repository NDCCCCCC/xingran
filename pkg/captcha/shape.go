// Package captcha 拼图形状接口和工厂
package captcha

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Shape 拼图形状接口
type Shape interface {
	// IsInside 判断点是否在形状内
	IsInside(x, y, width, height int) bool

	// GetName 获取形状名称
	GetName() string

	// GetComplexity 获取形状复杂度（影响难度）
	GetComplexity() int

	// GenerateMask 生成形状遮罩
	GenerateMask(width, height int) *image.Alpha
}

// ShapeFactory 形状工厂
type ShapeFactory interface {
	CreateShape(shapeType string) (Shape, error)
	GetSupportedShapes() []string
	RegisterShape(shapeType string, creator ShapeCreator) error
}

// ShapeCreator 形状创建器函数类型
type ShapeCreator func() Shape

// BaseShapeFactory 基础形状工厂实现
type BaseShapeFactory struct {
	creators map[string]ShapeCreator
}

// NewBaseShapeFactory 创建形状工厂
func NewBaseShapeFactory() ShapeFactory {
	factory := &BaseShapeFactory{
		creators: make(map[string]ShapeCreator),
	}

	// 注册内置形状（errors impossible for these built-in registrations, explicitly discard）
	_ = factory.RegisterShape("circle", func() Shape { return NewCircleShape() })
	_ = factory.RegisterShape("square", func() Shape { return NewSquareShape() })
	_ = factory.RegisterShape("star", func() Shape { return NewStarShape() })
	_ = factory.RegisterShape("heart", func() Shape { return NewHeartShape() })

	return factory
}

// RegisterShape 注册新形状
func (f *BaseShapeFactory) RegisterShape(shapeType string, creator ShapeCreator) error {
	if _, exists := f.creators[shapeType]; exists {
		return fmt.Errorf("shape type %s already registered", shapeType)
	}
	f.creators[shapeType] = creator
	return nil
}

// CreateShape 创建形状实例
func (f *BaseShapeFactory) CreateShape(shapeType string) (Shape, error) {
	creator, exists := f.creators[shapeType]
	if !exists {
		return nil, fmt.Errorf("unsupported shape type: %s", shapeType)
	}
	return creator(), nil
}

// GetSupportedShapes 获取支持的形状列表
func (f *BaseShapeFactory) GetSupportedShapes() []string {
	shapes := make([]string, 0, len(f.creators))
	for shapeType := range f.creators {
		shapes = append(shapes, shapeType)
	}
	return shapes
}

// ============ 形状实现 ============

// CircleShape 圆形拼图
type CircleShape struct {
	complexity int
}

// NewCircleShape 创建圆形形状
func NewCircleShape() *CircleShape {
	return &CircleShape{
		complexity: 3,
	}
}

// IsInside 判断点是否在圆形内
func (s *CircleShape) IsInside(x, y, width, height int) bool {
	centerX := width / 2
	centerY := height / 2
	// 调整半径为 45%，使圆形占据更多空间
	radius := min(width, height) * 45 / 100

	dx := x - centerX
	dy := y - centerY
	distSq := dx*dx + dy*dy

	return distSq <= radius*radius
}

// GetName 获取形状名称
func (s *CircleShape) GetName() string {
	return "circle"
}

// GetComplexity 获取复杂度
func (s *CircleShape) GetComplexity() int {
	return s.complexity
}

// GenerateMask 生成圆形遮罩
func (s *CircleShape) GenerateMask(width, height int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if s.IsInside(x, y, width, height) {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			} else {
				mask.SetAlpha(x, y, color.Alpha{A: 0})
			}
		}
	}

	return mask
}

// SquareShape 方形拼图（带波浪边缘）
type SquareShape struct {
	complexity int
}

// NewSquareShape 创建方形形状
func NewSquareShape() *SquareShape {
	return &SquareShape{
		complexity: 2,
	}
}

// IsInside 判断点是否在方形内
func (s *SquareShape) IsInside(x, y, width, height int) bool {
	margin := 5
	if x < margin || x > width-margin || y < margin || y > height-margin {
		return false
	}

	// 添加波浪效果
	wave := int(5.0 * sin(float64(x)*0.3))
	return !(x < margin+wave || x > width-margin-wave)
}

// GetName 获取形状名称
func (s *SquareShape) GetName() string {
	return "square"
}

// GetComplexity 获取复杂度
func (s *SquareShape) GetComplexity() int {
	return s.complexity
}

// GenerateMask 生成方形遮罩
func (s *SquareShape) GenerateMask(width, height int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if s.IsInside(x, y, width, height) {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			} else {
				mask.SetAlpha(x, y, color.Alpha{A: 0})
			}
		}
	}

	return mask
}

// StarShape 星形拼图（五角星）
type StarShape struct {
	complexity int
}

// NewStarShape 创建星形形状
func NewStarShape() *StarShape {
	return &StarShape{
		complexity: 5,
	}
}

// IsInside 判断点是否在五角星内
func (s *StarShape) IsInside(x, y, width, height int) bool {
	cx, cy := width/2, height/2
	outerRadius := min(width, height) / 2
	innerRadius := outerRadius / 2

	// 将点转换为极坐标
	dx, dy := x-cx, y-cy
	distance := math.Sqrt(float64(dx*dx + dy*dy))
	angle := math.Atan2(float64(dy), float64(dx))

	// 五角星有5个外顶点和5个内顶点
	// 检查角度对应的半径
	normalizedAngle := angle + math.Pi/2 // 旋转使一个顶点朝上
	if normalizedAngle < 0 {
		normalizedAngle += 2 * math.Pi
	}

	// 计算当前角度对应的星形边界半径
	segmentAngle := 2 * math.Pi / 10 // 36度
	segment := int(normalizedAngle / segmentAngle)
	t := normalizedAngle - float64(segment)*segmentAngle

	var radiusAtAngle float64
	if segment%2 == 0 {
		// 外顶点到内顶点
		radiusAtAngle = float64(outerRadius) - (float64(outerRadius)-float64(innerRadius))*t/segmentAngle
	} else {
		// 内顶点到外顶点
		radiusAtAngle = float64(innerRadius) + (float64(outerRadius)-float64(innerRadius))*t/segmentAngle
	}

	return distance <= radiusAtAngle
}

// GetName 获取形状名称
func (s *StarShape) GetName() string {
	return "star"
}

// GetComplexity 获取复杂度
func (s *StarShape) GetComplexity() int {
	return s.complexity
}

// GenerateMask 生成星形遮罩
func (s *StarShape) GenerateMask(width, height int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if s.IsInside(x, y, width, height) {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			} else {
				mask.SetAlpha(x, y, color.Alpha{A: 0})
			}
		}
	}

	return mask
}

// HeartShape 心形拼图
type HeartShape struct {
	complexity int
}

// NewHeartShape 创建心形形状
func NewHeartShape() *HeartShape {
	return &HeartShape{
		complexity: 4,
	}
}

// IsInside 判断点是否在心形内
func (s *HeartShape) IsInside(x, y, width, height int) bool {
	// 心形参数方程:
	// x = 16sin^3(t)
	// y = 13cos(t) - 5cos(2t) - 2cos(3t) - cos(4t)

	cx, cy := width/2, height/3
	scale := float64(min(width, height)) / 40.0

	dx := float64(x-cx) / scale
	dy := float64(y-cy) / scale

	// 简化的心形判断
	// (x^2 + y^2 - 1)^3 - x^2 * y^3 <= 0
	a := dx*dx + dy*dy - 16
	return a*a*a <= dx*dx*dy*dy*dy
}

// GetName 获取形状名称
func (s *HeartShape) GetName() string {
	return "heart"
}

// GetComplexity 获取复杂度
func (s *HeartShape) GetComplexity() int {
	return s.complexity
}

// GenerateMask 生成心形遮罩
func (s *HeartShape) GenerateMask(width, height int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if s.IsInside(x, y, width, height) {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			} else {
				mask.SetAlpha(x, y, color.Alpha{A: 0})
			}
		}
	}

	return mask
}

// 辅助函数

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sin(x float64) float64 {
	// 简化的正弦函数近似
	x = math.Mod(x, 2*math.Pi)
	result := x
	term := x
	for i := 3; i < 15; i += 2 {
		fact := 1.0
		for j := 2; j <= i; j++ {
			fact *= float64(j)
		}
		if i%4 == 1 {
			result += term / fact
		} else {
			result -= term / fact
		}
		term *= x * x
	}
	return result
}

// abs 绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

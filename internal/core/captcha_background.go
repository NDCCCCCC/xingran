// Package core 验证码背景图服务
// 负责背景图的管理、查询和预生成逻辑
package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
	"gorm.io/gorm"
)

// captchaPregenerateTimeout 图形验证码异步预生成超时
// 在异步补充缓存 goroutine 中使用的 context 超时，避免散落的硬编码字面量
const captchaPregenerateTimeout = 30 * time.Second

// CaptchaBackgroundService 验证码背景图服务
type CaptchaBackgroundService struct {
	db     *db.Database
	cache  cache.Cache
	config *BackgroundConfig
}

// BackgroundConfig 背景图配置
type BackgroundConfig struct {
	Mode           string   // auto, custom, mixed
	DefaultShape   string   // 默认拼图形状
	DefaultDiff    int      // 默认难度
	CachePoolSize  int      // 缓存池大小
	StoragePath    string   // 存储路径
	MaxFileSize    int64    // 最大文件大小
	AllowedFormats []string // 允许的格式
}

// NewCaptchaBackgroundService 创建背景图服务
func NewCaptchaBackgroundService(db *db.Database, cache cache.Cache) *CaptchaBackgroundService {
	return &CaptchaBackgroundService{
		db:    db,
		cache: cache,
		config: &BackgroundConfig{
			Mode:           "mixed",
			DefaultShape:   "circle",
			DefaultDiff:    1,
			CachePoolSize:  50,
			StoragePath:    "./uploads/captcha/backgrounds",
			MaxFileSize:    2 * 1024 * 1024, // 2MB
			AllowedFormats: []string{"jpg", "jpeg", "png"},
		},
	}
}

// GetDB 获取数据库连接
func (s *CaptchaBackgroundService) GetDB() *gorm.DB {
	return s.db.GetDB()
}

// ========== 图片管理 ==========

// Upload 上传背景图
func (s *CaptchaBackgroundService) Upload(ctx context.Context, req *UploadRequest) (*models.CaptchaBackground, error) {
	// 验证文件
	if err := s.validateFile(req.FileName, req.FileSize); err != nil {
		return nil, fmt.Errorf("文件验证失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(s.config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 生成唯一文件名
	ext := filepath.Ext(req.FileName)
	newFileName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), rand.Intn(1000), ext)
	destPath := filepath.Join(s.config.StoragePath, newFileName)

	// 保存文件
	if err := os.WriteFile(destPath, req.FileData, 0644); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 获取图片尺寸
	width, height, err := s.getImageDimensions(destPath)
	if err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("获取图片尺寸失败: %w", err)
	}

	// 计算MD5
	md5Sum := s.calculateMD5(req.FileData)

	// 创建数据库记录（路径使用正斜杠，兼容跨平台）
	bg := &models.CaptchaBackground{
		FileName:        newFileName,
		FilePath:        filepath.ToSlash(destPath),
		FileSize:        req.FileSize,
		FileWidth:       width,
		FileHeight:      height,
		FileMD5:         md5Sum,
		PieceShape:      req.PieceShape,
		DifficultyLevel: models.DifficultyLevel(req.DifficultyLevel),
		AllowedShapes:   req.AllowedShapes,
		Status:          models.CaptchaBgEnabled,
		UseCount:        0,
		SortOrder:       0,
	}

	if err := s.db.GetDB().Create(bg).Error; err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("创建数据库记录失败: %w", err)
	}

	return bg, nil
}

// UploadRequest 上传请求
type UploadRequest struct {
	FileName        string
	FileData        []byte
	FileSize        int64
	PieceShape      models.PieceShape
	DifficultyLevel int
	AllowedShapes   []string
	Remark          string
}

// GetRandomEnabled 获取随机启用的背景图
// 首先尝试精确匹配，如果没找到则尝试通过 allowedShapes 匹配
func (s *CaptchaBackgroundService) GetRandomEnabled(ctx context.Context, shape models.PieceShape, difficulty int) (*models.CaptchaBackground, error) {
	var backgrounds []*models.CaptchaBackground

	// 先尝试从缓存获取
	cacheKey := fmt.Sprintf("captcha:bg:list:%s:%d", shape, difficulty)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		if err := json.Unmarshal([]byte(cached), &backgrounds); err == nil && len(backgrounds) > 0 {
			return backgrounds[rand.Intn(len(backgrounds))], nil
		}
	}

	// 从数据库查询 - 首先尝试精确匹配
	query := s.db.GetDB().
		Where("status = ?", models.CaptchaBgEnabled).
		Where("piece_shape = ?", shape).
		Where("difficulty_level = ?", difficulty).
		Order("RANDOM()").
		Limit(s.config.CachePoolSize)

	if err := query.Find(&backgrounds).Error; err != nil {
		return nil, fmt.Errorf("查询背景图失败: %w", err)
	}

	// 如果精确匹配没有结果，尝试通过 allowedShapes 匹配
	if len(backgrounds) == 0 {
		query = s.db.GetDB().
			Where("status = ?", models.CaptchaBgEnabled).
			Where("difficulty_level = ?", difficulty).
			// allowedShapes 字段包含当前形状，或者为空（表示允许所有形状）
			Where("allowed_shapes @> ? OR allowed_shapes = '[]' OR allowed_shapes IS NULL OR jsonb_array_length(allowed_shapes) IS NULL",
				fmt.Sprintf(`["%s"]`, shape)).
			Order("RANDOM()").
			Limit(s.config.CachePoolSize)

		if err := query.Find(&backgrounds).Error; err != nil {
			return nil, fmt.Errorf("查询背景图失败: %w", err)
		}
	}

	if len(backgrounds) == 0 {
		return nil, fmt.Errorf("没有找到可用的背景图 (shape=%s, difficulty=%d)", shape, difficulty)
	}

	// 缓存查询结果
	if data, err := json.Marshal(backgrounds); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return backgrounds[rand.Intn(len(backgrounds))], nil
}

// IncrementUseCount 增加使用次数
func (s *CaptchaBackgroundService) IncrementUseCount(id string) error {
	return s.db.GetDB().
		Model(&models.CaptchaBackground{}).
		Where("id = ?", id).
		UpdateColumn("use_count", gorm.Expr("use_count + 1")).
		UpdateColumn("last_used_at", time.Now()).
		Error
}

// ========== 预生成缓存池 ==========

// PreGeneratePool 预生成验证码池
func (s *CaptchaBackgroundService) PreGeneratePool(ctx context.Context) error {
	shapes := []models.PieceShape{
		models.PieceShapeCircle,
		models.PieceShapeSquare,
		models.PieceShapeStar,
		models.PieceShapeHeart,
	}
	difficulties := []int{1, 2, 3}

	for _, shape := range shapes {
		for _, diff := range difficulties {
			if err := s.preGenerateForConfig(ctx, string(shape), diff); err != nil {
				// 静默处理预生成失败
				_ = err
			}
		}
	}

	return nil
}

// preGenerateForConfig 为指定配置预生成验证码
func (s *CaptchaBackgroundService) preGenerateForConfig(ctx context.Context, shape string, difficulty int) error {
	// 获取启用的背景图
	var backgrounds []*models.CaptchaBackground
	if err := s.db.GetDB().
		Where("status = ?", models.CaptchaBgEnabled).
		Where("piece_shape = ?", shape).
		Where("difficulty_level = ?", difficulty).
		Find(&backgrounds).Error; err != nil {
		return err
	}

	if len(backgrounds) == 0 {
		return nil
	}

	poolPrefix := fmt.Sprintf("captcha:cache:pool:%s:%d", shape, difficulty)
	poolSize := s.config.CachePoolSize
	counterKey := poolPrefix + ":counter"

	// 获取当前计数
	counterStr, _ := s.cache.Get(ctx, counterKey)
	var counter int64
	if counterStr != "" {
		if parsed, err := strconv.ParseInt(counterStr, 10, 64); err != nil {
			counter = 0
		} else {
			counter = parsed
		}
	}

	// 为每个背景图预生成验证码
	generator := captcha.NewSliderCaptcha()
	idx := 0
	for _, bg := range backgrounds {
		if idx >= int(poolSize) {
			break
		}
		// 加载背景图
		bgImg, err := generator.LoadBackgroundFromFile(bg.FilePath)
		if err != nil {
			continue
		}

		// 生成验证码
		bgBytes, pieceBytes, xPos, yPos, token, err := generator.GenerateSliderWithCustomBackground(bgImg, shape, difficulty)
		if err != nil {
			continue
		}

		// 生成缓存数据
		cacheData := map[string]interface{}{
			"backgroundId": bg.ID,
			"sliderImg":    "data:image/png;base64," + string(bgBytes),
			"pieceImg":     "data:image/png;base64," + string(pieceBytes),
			"xPos":         xPos,
			"yPos":         yPos,
			"token":        token,
			"shape":        shape,
			"difficulty":   difficulty,
			"createdAt":    time.Now().Unix(),
		}

		// 存入缓存（使用索引作为key的一部分）
		data, err := json.Marshal(cacheData)
		if err != nil {
			continue
		}
		itemKey := fmt.Sprintf("%s:%d", poolPrefix, (counter%int64(poolSize))+1)
			_ = s.cache.Set(ctx, itemKey, string(data), 24*time.Hour)

		counter++
		idx++
	}

	// 更新计数器
	_ = s.cache.Set(ctx, counterKey, strconv.FormatInt(counter, 10), 24*time.Hour)

	return nil
}

// GetFromCachePool 从缓存池获取验证码
func (s *CaptchaBackgroundService) GetFromCachePool(ctx context.Context, shape string, difficulty int) (map[string]interface{}, error) {
	poolPrefix := fmt.Sprintf("captcha:cache:pool:%s:%d", shape, difficulty)
	counterKey := poolPrefix + ":counter"
	poolSize := s.config.CachePoolSize

	// 获取当前计数
	counterStr, err := s.cache.Get(ctx, counterKey)
	if err != nil || counterStr == "" {
		return nil, fmt.Errorf("cache pool is empty")
	}

	counter, _ := strconv.ParseInt(counterStr, 10, 64)
	if counter == 0 {
		return nil, fmt.Errorf("cache pool is empty")
	}

	// 获取最新的缓存项
	itemKey := fmt.Sprintf("%s:%d", poolPrefix, (counter-1)%int64(poolSize)+1)
	data, err := s.cache.Get(ctx, itemKey)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	// 更新计数器
	newCounter := counter - 1
	if newCounter > 0 {
		_ = s.cache.Set(ctx, counterKey, strconv.FormatInt(newCounter, 10), 24*time.Hour)
	} else {
		_ = s.cache.Delete(ctx, counterKey)
	}

	// 异步补充缓存
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), captchaPregenerateTimeout)
		defer cancel()
		_ = s.preGenerateForConfig(ctx, shape, difficulty)
	}()

	return result, nil
}

// ========== 辅助方法 ==========

// validateImage 验证图片
func (s *CaptchaBackgroundService) validateFile(fileName string, fileSize int64) error {
	// 检查文件大小
	if fileSize > s.config.MaxFileSize {
		return fmt.Errorf("文件大小超过限制: %d bytes", s.config.MaxFileSize)
	}

	// 检查文件扩展名
	ext := filepath.Ext(fileName)
	if ext == "" {
		return fmt.Errorf("不支持的文件格式: 文件名缺少扩展名")
	}
	ext = ext[1:] // 去掉点
	allowed := false
	for _, format := range s.config.AllowedFormats {
		if ext == format {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}

	return nil
}

// getImageDimensions 获取图片尺寸
func (s *CaptchaBackgroundService) getImageDimensions(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}

// calculateMD5 计算MD5
func (s *CaptchaBackgroundService) calculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}
